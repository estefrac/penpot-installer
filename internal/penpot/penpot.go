package penpot

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/estefrac/penpot-installer/internal/system"
)

const (
	// ComposeURL es la URL del docker-compose oficial de Penpot
	ComposeURL = "https://raw.githubusercontent.com/penpot/penpot/main/docker/images/docker-compose.yaml"
	// DefaultPort es el puerto por defecto de Penpot
	DefaultPort = "9001"
	// ContainerPrefix es el prefijo de los contenedores de Penpot
	ContainerPrefix = "penpot"
)

// Config contiene la configuración de la instalación
type Config struct {
	InstallDir string
	Port       string
}

// DefaultConfig retorna la configuración por defecto
func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	return Config{
		InstallDir: filepath.Join(home, "penpot"),
		Port:       DefaultPort,
	}
}

// IsInstalled verifica si Penpot está instalado
func IsInstalled(cfg Config) bool {
	composeFile := filepath.Join(cfg.InstallDir, "docker-compose.yaml")
	_, err := os.Stat(composeFile)
	return err == nil
}

// IsRunning verifica si los contenedores de Penpot están corriendo
func IsRunning() bool {
	out, err := system.RunCommand("docker", "ps", "--filter", "name=penpot", "--format", "{{.Names}}")
	if err != nil {
		return false
	}
	return strings.Contains(out, "penpot")
}

// Status retorna el estado detallado de los contenedores de Penpot
func Status() ([]ContainerStatus, error) {
	out, err := system.RunCommand(
		"docker", "ps", "-a",
		"--filter", "name=penpot",
		"--format", "{{.Names}}\t{{.Status}}\t{{.Ports}}",
	)
	if err != nil {
		return nil, err
	}

	var statuses []ContainerStatus
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		status := ContainerStatus{Name: parts[0]}
		if len(parts) > 1 {
			status.Status = parts[1]
		}
		if len(parts) > 2 {
			status.Ports = parts[2]
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

// ContainerStatus representa el estado de un contenedor
type ContainerStatus struct {
	Name   string
	Status string
	Ports  string
}

// Install descarga los archivos de Penpot y levanta los contenedores
func Install(cfg Config) error {
	// Crear directorio de instalación
	if err := os.MkdirAll(cfg.InstallDir, 0755); err != nil {
		return fmt.Errorf("error creando directorio: %w", err)
	}

	// Descargar docker-compose.yaml
	fmt.Println("  → Descargando docker-compose.yaml...")
	composeFile := filepath.Join(cfg.InstallDir, "docker-compose.yaml")
	if _, err := system.RunCommand("curl", "-fsSL", "-o", composeFile, ComposeURL); err != nil {
		return fmt.Errorf("error descargando docker-compose.yaml: %w", err)
	}

	// Aplicar puerto personalizado si es diferente al default
	if cfg.Port != DefaultPort {
		if err := updatePort(composeFile, cfg.Port); err != nil {
			return fmt.Errorf("error actualizando puerto: %w", err)
		}
	}

	// Levantar contenedores
	fmt.Println("  → Descargando imágenes e iniciando contenedores (esto puede tardar varios minutos)...")
	if err := system.RunCommandInteractive("docker", "compose", "-f", composeFile, "up", "-d"); err != nil {
		return fmt.Errorf("error iniciando contenedores: %w", err)
	}

	return nil
}

// Start inicia los contenedores de Penpot
func Start(cfg Config) error {
	composeFile := filepath.Join(cfg.InstallDir, "docker-compose.yaml")
	return system.RunCommandInteractive("docker", "compose", "-f", composeFile, "start")
}

// Stop detiene los contenedores de Penpot
func Stop(cfg Config) error {
	composeFile := filepath.Join(cfg.InstallDir, "docker-compose.yaml")
	return system.RunCommandInteractive("docker", "compose", "-f", composeFile, "stop")
}

// Update actualiza Penpot bajando las últimas imágenes y el compose oficial
func Update(cfg Config) error {
	composeFile := filepath.Join(cfg.InstallDir, "docker-compose.yaml")

	fmt.Println("  → Deteniendo contenedores...")
	_ = system.RunCommandInteractive("docker", "compose", "-f", composeFile, "down")

	fmt.Println("  → Actualizando docker-compose.yaml...")
	if _, err := system.RunCommand("curl", "-fsSL", "-o", composeFile, ComposeURL); err != nil {
		return fmt.Errorf("error actualizando docker-compose.yaml: %w", err)
	}

	fmt.Println("  → Descargando últimas imágenes...")
	if err := system.RunCommandInteractive("docker", "compose", "-f", composeFile, "pull"); err != nil {
		return fmt.Errorf("error actualizando imágenes: %w", err)
	}

	fmt.Println("  → Iniciando contenedores actualizados...")
	return system.RunCommandInteractive("docker", "compose", "-f", composeFile, "up", "-d")
}

// Uninstall detiene y elimina todos los contenedores, volúmenes e imágenes de Penpot
func Uninstall(cfg Config, removeData bool) error {
	composeFile := filepath.Join(cfg.InstallDir, "docker-compose.yaml")

	// Bajar contenedores, volúmenes e imágenes
	args := []string{"compose", "-f", composeFile, "down", "--remove-orphans"}
	if removeData {
		args = append(args, "--volumes", "--rmi", "all")
	}

	if err := system.RunCommandInteractive("docker", args...); err != nil {
		return fmt.Errorf("error eliminando contenedores: %w", err)
	}

	// Eliminar directorio de instalación
	if err := os.RemoveAll(cfg.InstallDir); err != nil {
		return fmt.Errorf("error eliminando directorio: %w", err)
	}

	return nil
}

// updatePort reemplaza el puerto en el docker-compose.yaml
func updatePort(composeFile, newPort string) error {
	content, err := os.ReadFile(composeFile)
	if err != nil {
		return err
	}

	updated := strings.ReplaceAll(string(content), "9001:", newPort+":")
	return os.WriteFile(composeFile, []byte(updated), 0644)
}

// runStreaming ejecuta un comando y llama emit() con cada línea de output.
func runStreaming(emit func(string), name string, args ...string) error {
	cmd := exec.Command(name, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	combined := io.MultiReader(stdout, stderr)
	scanner := bufio.NewScanner(combined)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			emit(line)
		}
	}

	return cmd.Wait()
}

// InstallStreaming instala Penpot emitiendo líneas de progreso
func InstallStreaming(cfg Config, emit func(string)) error {
	if err := os.MkdirAll(cfg.InstallDir, 0755); err != nil {
		return fmt.Errorf("error creando directorio: %w", err)
	}

	emit("Descargando docker-compose.yaml...")
	composeFile := filepath.Join(cfg.InstallDir, "docker-compose.yaml")
	if _, err := system.RunCommand("curl", "-fsSL", "-o", composeFile, ComposeURL); err != nil {
		return fmt.Errorf("error descargando docker-compose.yaml: %w", err)
	}
	emit("docker-compose.yaml descargado")

	if cfg.Port != DefaultPort {
		if err := updatePort(composeFile, cfg.Port); err != nil {
			return fmt.Errorf("error actualizando puerto: %w", err)
		}
	}

	emit("Descargando imágenes e iniciando contenedores...")
	return runStreaming(emit, "docker", "compose", "-f", composeFile, "up", "-d")
}

// StartStreaming inicia Penpot emitiendo líneas de progreso
func StartStreaming(cfg Config, emit func(string)) error {
	composeFile := filepath.Join(cfg.InstallDir, "docker-compose.yaml")
	emit("Iniciando contenedores...")
	return runStreaming(emit, "docker", "compose", "-f", composeFile, "start")
}

// StopStreaming detiene Penpot emitiendo líneas de progreso
func StopStreaming(cfg Config, emit func(string)) error {
	composeFile := filepath.Join(cfg.InstallDir, "docker-compose.yaml")
	emit("Deteniendo contenedores...")
	return runStreaming(emit, "docker", "compose", "-f", composeFile, "stop")
}

// UpdateStreaming actualiza Penpot emitiendo líneas de progreso
func UpdateStreaming(cfg Config, emit func(string)) error {
	composeFile := filepath.Join(cfg.InstallDir, "docker-compose.yaml")

	emit("Deteniendo contenedores...")
	if err := runStreaming(emit, "docker", "compose", "-f", composeFile, "down"); err != nil {
		return err
	}

	emit("Actualizando docker-compose.yaml...")
	if _, err := system.RunCommand("curl", "-fsSL", "-o", composeFile, ComposeURL); err != nil {
		return fmt.Errorf("error actualizando docker-compose.yaml: %w", err)
	}
	emit("docker-compose.yaml actualizado")

	emit("Descargando últimas imágenes...")
	if err := runStreaming(emit, "docker", "compose", "-f", composeFile, "pull"); err != nil {
		return err
	}

	emit("Iniciando contenedores actualizados...")
	return runStreaming(emit, "docker", "compose", "-f", composeFile, "up", "-d")
}

// UninstallStreaming desinstala Penpot eliminando contenedores, volúmenes e imágenes
func UninstallStreaming(cfg Config, emit func(string)) error {
	composeFile := filepath.Join(cfg.InstallDir, "docker-compose.yaml")

	emit("Eliminando contenedores, volúmenes e imágenes...")
	if err := runStreaming(emit, "docker", "compose", "-f", composeFile, "down",
		"--volumes", "--rmi", "all", "--remove-orphans"); err != nil {
		return fmt.Errorf("error eliminando contenedores: %w", err)
	}

	emit("Eliminando directorio de instalación...")
	if err := os.RemoveAll(cfg.InstallDir); err != nil {
		return fmt.Errorf("error eliminando directorio: %w", err)
	}
	emit("Directorio eliminado")

	return nil
}

// UninstallKeepDataStreaming desinstala Penpot conservando los volúmenes con los datos
func UninstallKeepDataStreaming(cfg Config, emit func(string)) error {
	composeFile := filepath.Join(cfg.InstallDir, "docker-compose.yaml")

	emit("Eliminando contenedores e imágenes (conservando datos)...")
	if err := runStreaming(emit, "docker", "compose", "-f", composeFile, "down",
		"--rmi", "all", "--remove-orphans"); err != nil {
		return fmt.Errorf("error eliminando contenedores: %w", err)
	}

	emit("Eliminando directorio de instalación...")
	if err := os.RemoveAll(cfg.InstallDir); err != nil {
		return fmt.Errorf("error eliminando directorio: %w", err)
	}
	emit("Directorio eliminado")

	return nil
}
