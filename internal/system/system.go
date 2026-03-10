package system

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
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

// RunCommandInteractive ejecuta un comando descartando el output.
// Cuando el TUI está activo (AltScreen), escribir directo a os.Stdout
// rompe el renderizado — los comandos largos corren en background silencioso.
func RunCommandInteractive(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	return cmd.Run()
}

// RunCommandStreaming ejecuta un comando y envía cada línea de output
// al canal lines. Cierra el canal cuando el comando termina.
// El error final se envía por el canal err (con buffer 1).
func RunCommandStreaming(lines chan<- string, name string, args ...string) <-chan error {
	errCh := make(chan error, 1)

	go func() {
		defer close(lines)
		cmd := exec.Command(name, args...)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			errCh <- err
			return
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			errCh <- err
			return
		}

		if err := cmd.Start(); err != nil {
			errCh <- err
			return
		}

		// Leer stdout y stderr en paralelo, enviar líneas al canal
		combined := io.MultiReader(stdout, stderr)
		scanner := bufio.NewScanner(combined)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				lines <- line
			}
		}

		errCh <- cmd.Wait()
	}()

	return errCh
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
