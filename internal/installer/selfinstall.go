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

// InstallResult comunica el resultado de la auto-instalación al TUI.
// El installer es silencioso — no imprime nada a stdout/stderr.
type InstallResult struct {
	Action  string // "none", "installed", "updated", "failed"
	Message string // Mensaje legible para mostrar en el TUI
}

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
// Si ya existe pero se está ejecutando desde otra ubicación (ej: /tmp tras un curl),
// se sobreescribe el binario instalado para actualizarlo.
// Retorna un InstallResult con el estado de la operación para que el TUI lo muestre.
// En caso de error no fatal (permisos, etc.) retorna Action="failed" con un mensaje útil.
func EnsureInstalled() InstallResult {
	// Si ya estamos corriendo desde la ubicación instalada, no hacer nada
	if isInstalledLocation() {
		return InstallResult{Action: "none"}
	}

	target, err := installedPath()
	if err != nil {
		return InstallResult{Action: "none"}
	}

	// Obtener la ruta del ejecutable actual
	execPath, err := os.Executable()
	if err != nil {
		return InstallResult{Action: "none"}
	}

	// Determinar si es instalación nueva o actualización
	_, statErr := os.Stat(target)
	isUpdate := statErr == nil // el archivo ya existe → es una actualización

	// Instalar o actualizar según el OS
	switch runtime.GOOS {
	case "linux", "darwin":
		return installUnix(execPath, target, isUpdate)
	case "windows":
		return installWindows(execPath, target, isUpdate)
	}

	return InstallResult{Action: "none"}
}

// installUnix copia el binario a /usr/local/bin usando sudo si es necesario.
// Si isUpdate es true, sobreescribe el binario existente.
func installUnix(src, dst string, isUpdate bool) InstallResult {
	action := "installed"
	if isUpdate {
		action = "updated"
	}

	// Intentar copiar directo primero (si ya somos root)
	if os.Geteuid() == 0 {
		if err := copyFile(src, dst); err == nil {
			return InstallResult{
				Action:  action,
				Message: fmt.Sprintf("%s en %s", binaryName, filepath.Dir(dst)),
			}
		}
	}

	// Usar sudo para copiar — EnsureInstalled() se ejecuta ANTES del TUI,
	// así que stdin/stdout/stderr están disponibles para el prompt de contraseña.
	if isUpdate {
		fmt.Printf("\n  Actualizando %s en %s...\n", binaryName, filepath.Dir(dst))
	} else {
		fmt.Printf("\n  Instalando %s en %s...\n", binaryName, filepath.Dir(dst))
	}
	cmd := exec.Command("sudo", "cp", src, dst)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return InstallResult{
			Action:  "failed",
			Message: fmt.Sprintf("sudo cp %s %s", src, dst),
		}
	}

	// Asegurar permisos de ejecución
	chmodCmd := exec.Command("sudo", "chmod", "+x", dst)
	chmodCmd.Stdin = os.Stdin
	chmodCmd.Stdout = os.Stdout
	chmodCmd.Stderr = os.Stderr
	_ = chmodCmd.Run()

	return InstallResult{
		Action:  action,
		Message: fmt.Sprintf("%s en %s", binaryName, filepath.Dir(dst)),
	}
}

// installWindows copia el binario a %LOCALAPPDATA%\penpot-manager y lo agrega al PATH del usuario.
// Si isUpdate es true, sobreescribe el binario existente.
func installWindows(src, dst string, isUpdate bool) InstallResult {
	dir := filepath.Dir(dst)

	action := "installed"
	if isUpdate {
		action = "updated"
	}

	// Crear directorio destino
	if err := os.MkdirAll(dir, 0755); err != nil {
		return InstallResult{
			Action:  "failed",
			Message: fmt.Sprintf("No se pudo crear %s: %v", dir, err),
		}
	}

	// Copiar binario
	if err := copyFile(src, dst); err != nil {
		return InstallResult{
			Action:  "failed",
			Message: fmt.Sprintf("No se pudo copiar a %s: %v", dir, err),
		}
	}

	// Agregar al PATH del usuario si no está
	addToWindowsPath(dir)

	return InstallResult{
		Action:  action,
		Message: fmt.Sprintf("%s en %s", binaryName, dir),
	}
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
