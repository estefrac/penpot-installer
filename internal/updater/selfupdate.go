package updater

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

const downloadBaseURL = "https://github.com/estefrac/penpot-installer/releases/download/%s/penpot-manager-%s"

// SelfUpdate descarga el binario de la versión indicada y reemplaza el ejecutable actual.
func SelfUpdate(version string, emit func(string)) error {
	// Determinar la URL del binario según el OS/arch actual
	binaryName, err := binaryForPlatform()
	if err != nil {
		return err
	}

	url := fmt.Sprintf(downloadBaseURL, version, binaryName)
	emit(fmt.Sprintf("Descargando %s...", url))

	// Descargar el binario nuevo a un archivo temporal
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("error descargando binario: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error descargando binario: HTTP %d", resp.StatusCode)
	}

	// Escribir en archivo temporal al lado del ejecutable actual
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("no se pudo obtener la ruta del ejecutable: %w", err)
	}

	tmpPath := execPath + ".new"
	tmp, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("no se pudo crear archivo temporal: %w", err)
	}

	emit("Escribiendo nueva versión...")
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("error escribiendo binario: %w", err)
	}
	tmp.Close()

	// Reemplazar el ejecutable actual con el nuevo (atómico en Unix)
	emit("Reemplazando ejecutable...")
	if err := os.Rename(tmpPath, execPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("no se pudo reemplazar el ejecutable: %w", err)
	}

	emit("¡Actualización completada!")
	return nil
}

// binaryForPlatform retorna el nombre del binario para el OS/arch actual
func binaryForPlatform() (string, error) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Normalizar arch
	arch := goarch
	if goarch == "amd64" {
		arch = "amd64"
	} else if goarch == "arm64" {
		arch = "arm64"
	} else {
		return "", fmt.Errorf("arquitectura no soportada: %s", goarch)
	}

	switch goos {
	case "linux":
		return fmt.Sprintf("linux-%s", arch), nil
	case "windows":
		return fmt.Sprintf("windows-%s.exe", arch), nil
	default:
		return "", fmt.Errorf("sistema operativo no soportado para auto-update: %s\n\nDescargá el binario manualmente desde:\nhttps://github.com/estefrac/penpot-installer/releases/latest", strings.Title(goos))
	}
}
