package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/pterm/pterm"
	"github.com/estefrac/penpot-installer/internal/docker"
	"github.com/estefrac/penpot-installer/internal/penpot"
	"github.com/estefrac/penpot-installer/internal/system"
	"github.com/estefrac/penpot-installer/internal/ui"
)

func main() {
	ui.Banner()

	detectedOS := system.Detect()
	pterm.Info.Printf("Sistema operativo detectado: %s\n", pterm.Bold.Sprint(string(detectedOS)))

	if detectedOS == system.Unknown {
		pterm.Error.Println("Sistema operativo no soportado.")
		os.Exit(1)
	}

	ui.Separator()

	// Verificar y preparar Docker
	if err := ensureDocker(); err != nil {
		pterm.Error.Println(err.Error())
		os.Exit(1)
	}

	// Cargar configuración existente o usar defaults
	cfg := penpot.DefaultConfig()

	// Menú principal
	for {
		if err := mainMenu(cfg); err != nil {
			if err.Error() == "exit" {
				pterm.Println()
				pterm.Success.Println("¡Hasta luego! 👋")
				pterm.Println()
				os.Exit(0)
			}
			pterm.Error.Println(err.Error())
		}
	}
}

// ensureDocker verifica que Docker esté instalado y corriendo
func ensureDocker() error {
	if !docker.IsInstalled() {
		pterm.Warning.Println("Docker no está instalado en este sistema.")
		pterm.Println()

		var install bool
		prompt := &survey.Confirm{
			Message: "¿Querés instalar Docker ahora?",
			Default: true,
		}
		if err := survey.AskOne(prompt, &install); err != nil {
			return fmt.Errorf("operación cancelada")
		}

		if !install {
			return fmt.Errorf("Docker es necesario para continuar. Instalalo y volvé a ejecutar el instalador")
		}

		spinner, _ := ui.Spinner("Instalando Docker...")
		if err := docker.Install(); err != nil {
			spinner.Fail("Error instalando Docker")
			return err
		}
		spinner.Success("Docker instalado correctamente")
	} else {
		pterm.Success.Printf("Docker %s detectado\n", docker.Version())
	}

	if !docker.IsRunning() {
		pterm.Warning.Println("El daemon de Docker no está corriendo.")
		pterm.Println()

		var startDocker bool
		prompt := &survey.Confirm{
			Message: "¿Querés iniciar Docker ahora?",
			Default: true,
		}
		if err := survey.AskOne(prompt, &startDocker); err != nil || !startDocker {
			return fmt.Errorf("Docker debe estar corriendo para continuar")
		}

		spinner, _ := ui.Spinner("Iniciando Docker...")
		_, err := system.RunCommand("systemctl", "start", "docker")
		if err != nil {
			spinner.Fail("No se pudo iniciar Docker automáticamente")
			pterm.Info.Println("Intentá: sudo systemctl start docker")
			return fmt.Errorf("iniciá Docker manualmente y volvé a ejecutar")
		}
		spinner.Success("Docker iniciado")
	}

	if !docker.ComposeInstalled() {
		return fmt.Errorf("docker compose no está disponible. Actualizá Docker a una versión reciente")
	}

	ui.Separator()
	return nil
}

// mainMenu muestra el menú principal y ejecuta la opción seleccionada
func mainMenu(cfg penpot.Config) error {
	isInstalled := penpot.IsInstalled(cfg)
	isRunning := penpot.IsRunning()

	// Construir opciones según el estado actual
	options := buildMenuOptions(isInstalled, isRunning)

	var selected string
	prompt := &survey.Select{
		Message: "¿Qué querés hacer?",
		Options: options,
	}
	if err := survey.AskOne(prompt, &selected); err != nil {
		return fmt.Errorf("exit")
	}

	pterm.Println()

	switch selected {
	case "🚀 Instalar Penpot":
		return handleInstall(cfg)
	case "▶️  Iniciar Penpot":
		return handleStart(cfg)
	case "⏹️  Detener Penpot":
		return handleStop(cfg)
	case "🔄 Actualizar Penpot":
		return handleUpdate(cfg)
	case "📊 Ver estado":
		return handleStatus(cfg)
	case "🌐 Abrir en navegador":
		return handleOpenBrowser(cfg)
	case "🗑️  Desinstalar Penpot":
		return handleUninstall(cfg)
	case "❌ Salir":
		return fmt.Errorf("exit")
	}

	return nil
}

func buildMenuOptions(isInstalled, isRunning bool) []string {
	if !isInstalled {
		return []string{
			"🚀 Instalar Penpot",
			"❌ Salir",
		}
	}

	options := []string{}

	if isRunning {
		options = append(options,
			"⏹️  Detener Penpot",
			"🌐 Abrir en navegador",
			"📊 Ver estado",
			"🔄 Actualizar Penpot",
		)
	} else {
		options = append(options,
			"▶️  Iniciar Penpot",
			"📊 Ver estado",
			"🔄 Actualizar Penpot",
		)
	}

	options = append(options,
		"🗑️  Desinstalar Penpot",
		"❌ Salir",
	)

	return options
}

// handleInstall gestiona la instalación de Penpot
func handleInstall(cfg penpot.Config) error {
	pterm.DefaultSection.Println("Instalación de Penpot")

	if penpot.IsInstalled(cfg) {
		pterm.Warning.Println("Penpot ya está instalado.")
		return nil
	}

	// Preguntar directorio de instalación
	var installDir string
	prompt := &survey.Input{
		Message: "Directorio de instalación:",
		Default: cfg.InstallDir,
	}
	if err := survey.AskOne(prompt, &installDir); err != nil {
		return nil
	}
	cfg.InstallDir = installDir

	// Preguntar puerto
	var port string
	portPrompt := &survey.Input{
		Message: "Puerto para acceder a Penpot:",
		Default: penpot.DefaultPort,
		Help:    "El puerto en el que estará disponible Penpot en tu navegador",
	}
	if err := survey.AskOne(portPrompt, &port, survey.WithValidator(validatePort)); err != nil {
		return nil
	}
	cfg.Port = port

	// Confirmar
	pterm.Println()
	pterm.Info.Printfln("Directorio: %s", pterm.Bold.Sprint(cfg.InstallDir))
	pterm.Info.Printfln("Puerto:     %s", pterm.Bold.Sprint(cfg.Port))
	pterm.Println()

	var confirm bool
	confirmPrompt := &survey.Confirm{
		Message: "¿Confirmar instalación?",
		Default: true,
	}
	if err := survey.AskOne(confirmPrompt, &confirm); err != nil || !confirm {
		pterm.Info.Println("Instalación cancelada.")
		return nil
	}

	pterm.Println()

	spinner, _ := ui.Spinner("Instalando Penpot...")
	spinner.Stop()

	if err := penpot.Install(cfg); err != nil {
		return fmt.Errorf("error durante la instalación: %w", err)
	}

	pterm.Println()
	pterm.Success.Println("¡Penpot instalado correctamente! 🎨")
	pterm.Println()
	pterm.Info.Printfln("Accedé a Penpot en: %s", pterm.Bold.Sprintf("http://localhost:%s", cfg.Port))
	pterm.Info.Println("La primera vez puede tardar unos segundos en estar listo.")
	pterm.Println()

	var openBrowser bool
	openPrompt := &survey.Confirm{
		Message: "¿Abrir Penpot en el navegador ahora?",
		Default: true,
	}
	if err := survey.AskOne(openPrompt, &openBrowser); err == nil && openBrowser {
		_ = system.OpenBrowser(fmt.Sprintf("http://localhost:%s", cfg.Port))
	}

	return nil
}

// handleStart inicia Penpot
func handleStart(cfg penpot.Config) error {
	spinner, _ := ui.Spinner("Iniciando Penpot...")
	if err := penpot.Start(cfg); err != nil {
		spinner.Fail("Error al iniciar Penpot")
		return err
	}
	spinner.Success("Penpot iniciado correctamente")
	pterm.Info.Printfln("Accedé en: http://localhost:%s", cfg.Port)
	return nil
}

// handleStop detiene Penpot
func handleStop(cfg penpot.Config) error {
	var confirm bool
	prompt := &survey.Confirm{
		Message: "¿Detener Penpot?",
		Default: true,
	}
	if err := survey.AskOne(prompt, &confirm); err != nil || !confirm {
		return nil
	}

	spinner, _ := ui.Spinner("Deteniendo Penpot...")
	if err := penpot.Stop(cfg); err != nil {
		spinner.Fail("Error al detener Penpot")
		return err
	}
	spinner.Success("Penpot detenido")
	return nil
}

// handleUpdate actualiza Penpot
func handleUpdate(cfg penpot.Config) error {
	pterm.Warning.Println("Esto detendrá Penpot temporalmente para actualizar las imágenes.")
	pterm.Println()

	var confirm bool
	prompt := &survey.Confirm{
		Message: "¿Continuar con la actualización?",
		Default: true,
	}
	if err := survey.AskOne(prompt, &confirm); err != nil || !confirm {
		return nil
	}

	pterm.Println()
	if err := penpot.Update(cfg); err != nil {
		return fmt.Errorf("error actualizando Penpot: %w", err)
	}

	pterm.Success.Println("¡Penpot actualizado correctamente! 🎉")
	return nil
}

// handleStatus muestra el estado de Penpot
func handleStatus(cfg penpot.Config) error {
	pterm.DefaultSection.Println("Estado de Penpot")

	statuses, err := penpot.Status()
	if err != nil || len(statuses) == 0 {
		pterm.Warning.Println("No se encontraron contenedores de Penpot.")
		return nil
	}

	tableData := pterm.TableData{
		{"Contenedor", "Estado", "Puertos"},
	}

	for _, s := range statuses {
		status := s.Status
		if strings.Contains(strings.ToLower(status), "up") {
			status = pterm.Green(status)
		} else {
			status = pterm.Red(status)
		}
		tableData = append(tableData, []string{s.Name, status, s.Ports})
	}

	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
	pterm.Println()

	if penpot.IsRunning() {
		pterm.Info.Printfln("Penpot está disponible en: %s", pterm.Bold.Sprintf("http://localhost:%s", cfg.Port))
	}

	return nil
}

// handleOpenBrowser abre Penpot en el navegador
func handleOpenBrowser(cfg penpot.Config) error {
	url := fmt.Sprintf("http://localhost:%s", cfg.Port)
	pterm.Info.Printfln("Abriendo %s...", url)
	return system.OpenBrowser(url)
}

// handleUninstall desinstala Penpot
func handleUninstall(cfg penpot.Config) error {
	pterm.DefaultSection.Println("Desinstalación de Penpot")
	pterm.Warning.Println("Esta acción eliminará los contenedores de Penpot.")
	pterm.Println()

	var removeData bool
	dataPrompt := &survey.Confirm{
		Message: "¿También eliminar todos los datos y volúmenes? (proyectos, archivos, usuarios)",
		Default: false,
	}
	if err := survey.AskOne(dataPrompt, &removeData); err != nil {
		return nil
	}

	if removeData {
		pterm.Error.Println("¡ADVERTENCIA! Esto eliminará TODOS tus proyectos y datos de Penpot de forma IRREVERSIBLE.")
		pterm.Println()
	}

	var confirm bool
	confirmPrompt := &survey.Confirm{
		Message: "¿Estás seguro que querés desinstalar Penpot?",
		Default: false,
	}
	if err := survey.AskOne(confirmPrompt, &confirm); err != nil || !confirm {
		pterm.Info.Println("Desinstalación cancelada.")
		return nil
	}

	pterm.Println()
	spinner, _ := ui.Spinner("Desinstalando Penpot...")
	spinner.Stop()

	if err := penpot.Uninstall(cfg, removeData); err != nil {
		return fmt.Errorf("error desinstalando Penpot: %w", err)
	}

	pterm.Success.Println("Penpot desinstalado correctamente.")
	return nil
}

// validatePort valida que el puerto sea un número válido
func validatePort(val interface{}) error {
	str, ok := val.(string)
	if !ok {
		return fmt.Errorf("valor inválido")
	}
	port, err := strconv.Atoi(str)
	if err != nil || port < 1024 || port > 65535 {
		return fmt.Errorf("ingresá un puerto válido entre 1024 y 65535")
	}
	return nil
}
