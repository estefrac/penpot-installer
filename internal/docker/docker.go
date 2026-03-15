package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/estefrac/penpot-installer/internal/system"
)

// runCommand es la función de ejecución de comandos.
// Se puede overridear en tests para evitar dependencia del entorno.
var runCommand = system.RunCommand

// IsInstalled verifica si Docker está instalado
func IsInstalled() bool {
	return system.CommandExists("docker")
}

// IsRunning verifica si el daemon de Docker está corriendo
func IsRunning() bool {
	_, err := system.RunCommand("docker", "info")
	return err == nil
}

// Version retorna la versión de Docker instalada
func Version() string {
	out, err := system.RunCommand("docker", "version", "--format", "{{.Server.Version}}")
	if err != nil {
		return "desconocida"
	}
	return strings.TrimSpace(out)
}

// Install instala Docker según el OS detectado
func Install() error {
	switch runtime.GOOS {
	case "linux":
		return installLinux()
	case "windows":
		return installWindows()
	default:
		return fmt.Errorf("instalación automática de Docker no soportada en este OS")
	}
}

func installLinux() error {
	// Usamos el script oficial de Docker
	steps := []struct {
		name string
		cmd  string
		args []string
	}{
		{"Descargando script de instalación", "sh", []string{"-c", "curl -fsSL https://get.docker.com -o /tmp/get-docker.sh"}},
		{"Instalando Docker", "sh", []string{"/tmp/get-docker.sh"}},
		{"Iniciando servicio Docker", "systemctl", []string{"start", "docker"}},
		{"Habilitando Docker al inicio", "systemctl", []string{"enable", "docker"}},
	}

	for _, step := range steps {
		fmt.Printf("  → %s...\n", step.name)
		if err := system.RunCommandInteractive(step.cmd, step.args...); err != nil {
			return fmt.Errorf("error en '%s': %w", step.name, err)
		}
	}

	// Agregar usuario al grupo docker para no necesitar sudo
	currentUser := getCurrentUser()
	if currentUser != "root" && currentUser != "" {
		_ = system.RunCommandInteractive("usermod", "-aG", "docker", currentUser)
	}

	return nil
}

func installWindows() error {
	// En Windows redirigimos al instalador oficial
	url := "https://desktop.docker.com/win/main/amd64/Docker%20Desktop%20Installer.exe"
	fmt.Println()
	fmt.Println("  En Windows, Docker Desktop se instala manualmente.")
	fmt.Printf("  Descargando desde: %s\n", url)
	fmt.Println()

	if err := system.OpenBrowser(url); err != nil {
		fmt.Printf("  Abrí este link manualmente: %s\n", url)
	}

	fmt.Println("  Una vez instalado Docker Desktop, volvé a ejecutar este instalador.")
	return fmt.Errorf("reiniciá el instalador luego de instalar Docker Desktop")
}

func getCurrentUser() string {
	out, err := exec.Command("whoami").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// ComposeInstalled verifica si docker compose (V2) está disponible.
// No detecta docker-compose V1 (standalone) — está EOL desde julio 2023.
func ComposeInstalled() bool {
	_, err := runCommand("docker", "compose", "version")
	return err == nil
}

// InstallCompose instala el plugin docker-compose-plugin via apt-get.
// Solo soportado en Linux (Debian/Ubuntu). Otros package managers deben
// instalar manualmente: https://docs.docker.com/compose/install/linux/
func InstallCompose() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("instalación automática de Docker Compose solo soportada en Linux")
	}

	steps := []struct {
		name string
		cmd  string
		args []string
	}{
		{
			"Actualizando índice de paquetes",
			"apt-get", []string{"update", "-y"},
		},
		{
			"Instalando docker-compose-plugin",
			"apt-get", []string{"install", "-y", "docker-compose-plugin"},
		},
	}

	for _, step := range steps {
		if err := system.RunCommandInteractive(step.cmd, step.args...); err != nil {
			return fmt.Errorf("error en '%s': %w", step.name, err)
		}
	}

	return nil
}

// StartDesktop intenta iniciar Docker Desktop en Windows buscando el ejecutable
// en las rutas típicas de instalación. Retorna error si no lo encuentra o no puede lanzarlo.
func StartDesktop() error {
	candidates := []string{
		filepath.Join(os.Getenv("ProgramFiles"), "Docker", "Docker", "Docker Desktop.exe"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Docker", "Docker Desktop.exe"),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return exec.Command(path).Start()
		}
	}

	return fmt.Errorf("no se encontró Docker Desktop")
}
