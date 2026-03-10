package system

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// OS representa el sistema operativo detectado
type OS string

const (
	Linux   OS = "linux"
	Windows OS = "windows"
	MacOS   OS = "darwin"
	Unknown OS = "unknown"
)

// Detect devuelve el sistema operativo actual
func Detect() OS {
	switch runtime.GOOS {
	case "linux":
		return Linux
	case "windows":
		return Windows
	case "darwin":
		return MacOS
	default:
		return Unknown
	}
}

// IsRoot verifica si el proceso corre como root/administrador
func IsRoot() bool {
	if runtime.GOOS == "windows" {
		// En Windows, intentamos abrir un archivo del sistema
		_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
		return err == nil
	}
	return os.Geteuid() == 0
}

// CommandExists verifica si un comando existe en el PATH
func CommandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// RunCommand ejecuta un comando y retorna el output
func RunCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// RunCommandInteractive ejecuta un comando mostrando el output en tiempo real
func RunCommandInteractive(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// OpenBrowser abre una URL en el navegador predeterminado
func OpenBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		return fmt.Errorf("OS no soportado para abrir el navegador")
	}

	return exec.Command(cmd, args...).Start()
}
