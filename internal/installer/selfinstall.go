package installer

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const binaryName = "penpot-manager"

// installDir retorna el directorio destino según el OS:
//   - Linux/macOS: /usr/local/bin
//   - Windows:     %LOCALAPPDATA%\penpot-manager
func installDir() (string, error) {
	switch runtime.GOOS {
	case "linux", "darwin":
		return "/usr/local/bin", nil
	case "windows":
		local := os.Getenv("LOCALAPPDATA")
		if local == "" {
			return "", fmt.Errorf("no se encontró %%LOCALAPPDATA%%")
		}
		return filepath.Join(local, binaryName), nil
	default:
		return "", fmt.Errorf("OS no soportado: %s", runtime.GOOS)
	}
}

// installedPath retorna la ruta completa donde debería estar el binario instalado.
func installedPath() (string, error) {
	dir, err := installDir()
	if err != nil {
		return "", err
	}

	name := binaryName
	if runtime.GOOS == "windows" {
		name += ".exe"
	}

	return filepath.Join(dir, name), nil
}

// isInstalledLocation chequea si el ejecutable actual ya está corriendo
// desde la ubicación de instalación.
func isInstalledLocation() bool {
	execPath, err := os.Executable()
	if err != nil {
		return false
	}
	// Resolver symlinks para comparar paths reales
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return false
	}

	target, err := installedPath()
	if err != nil {
		return false
	}
	target, _ = filepath.EvalSymlinks(target)

	return strings.EqualFold(execPath, target)
}

// EnsureInstalled verifica si el binario está instalado en el PATH del sistema.
// Si no lo está, se copia a la ubicación correcta.
// Retorna true si se instaló (primera vez), false si ya estaba.
// En caso de error no fatal (permisos, etc.) simplemente continúa sin instalar.
func EnsureInstalled() bool {
	// Si ya estamos corriendo desde la ubicación instalada, no hacer nada
	if isInstalledLocation() {
		return false
	}

	// Si ya existe el binario en el PATH (instalado antes), no hacer nada
	target, err := installedPath()
	if err != nil {
		return false
	}
	if _, err := os.Stat(target); err == nil {
		return false
	}

	// Obtener la ruta del ejecutable actual
	execPath, err := os.Executable()
	if err != nil {
		return false
	}

	// Instalar según el OS
	switch runtime.GOOS {
	case "linux", "darwin":
		return installUnix(execPath, target)
	case "windows":
		return installWindows(execPath, target)
	}

	return false
}

// installUnix copia el binario a /usr/local/bin usando sudo si es necesario.
func installUnix(src, dst string) bool {
	fmt.Printf("\n  Instalando %s en %s...\n", binaryName, filepath.Dir(dst))

	// Intentar copiar directo primero (si ya somos root)
	if os.Geteuid() == 0 {
		if err := copyFile(src, dst); err == nil {
			fmt.Printf("  %s instalado. Ahora podés ejecutarlo con: %s\n\n", binaryName, binaryName)
			return true
		}
	}

	// Usar sudo para copiar
	cmd := exec.Command("sudo", "cp", src, dst)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Si falla (usuario cancela sudo, etc.), seguir sin instalar
		fmt.Printf("  No se pudo instalar en %s (se necesitan permisos de administrador).\n", filepath.Dir(dst))
		fmt.Printf("  Podés ejecutar manualmente: sudo cp %s %s\n\n", src, dst)
		return false
	}

	// Asegurar permisos de ejecución
	_ = exec.Command("sudo", "chmod", "+x", dst).Run()

	fmt.Printf("  %s instalado. Ahora podés ejecutarlo con: %s\n\n", binaryName, binaryName)
	return true
}

// installWindows copia el binario a %LOCALAPPDATA%\penpot-manager y lo agrega al PATH del usuario.
func installWindows(src, dst string) bool {
	dir := filepath.Dir(dst)

	// Crear directorio destino
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false
	}

	// Copiar binario
	if err := copyFile(src, dst); err != nil {
		fmt.Printf("  No se pudo instalar en %s: %v\n", dir, err)
		return false
	}

	// Agregar al PATH del usuario si no está
	addToWindowsPath(dir)

	fmt.Printf("\n  %s instalado en %s\n", binaryName, dir)
	fmt.Printf("  Reiniciá la terminal y ejecutalo con: %s\n\n", binaryName)
	return true
}

// addToWindowsPath agrega un directorio al PATH del usuario via registro de Windows.
func addToWindowsPath(dir string) {
	// Leer PATH actual del usuario
	out, err := exec.Command("reg", "query", "HKCU\\Environment", "/v", "Path").CombinedOutput()
	if err != nil {
		return
	}

	currentPath := ""
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "REG_") {
			parts := strings.SplitN(line, "REG_EXPAND_SZ", 2)
			if len(parts) < 2 {
				parts = strings.SplitN(line, "REG_SZ", 2)
			}
			if len(parts) >= 2 {
				currentPath = strings.TrimSpace(parts[1])
			}
		}
	}

	// Si ya está en el PATH, no hacer nada
	for _, p := range strings.Split(currentPath, ";") {
		if strings.EqualFold(strings.TrimSpace(p), dir) {
			return
		}
	}

	// Agregar al PATH
	newPath := currentPath + ";" + dir
	_ = exec.Command("reg", "add", "HKCU\\Environment", "/v", "Path", "/t", "REG_EXPAND_SZ", "/d", newPath, "/f").Run()

	// Notificar al sistema del cambio (broadcast WM_SETTINGCHANGE)
	// Esto no funciona desde Go fácilmente, pero al reiniciar la terminal se toma
}

// copyFile copia un archivo de src a dst preservando permisos de ejecución.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
