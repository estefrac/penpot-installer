package tui

import (
	"fmt"
	"runtime"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/estefrac/penpot-installer/internal/docker"
	"github.com/estefrac/penpot-installer/internal/installer"
	"github.com/estefrac/penpot-installer/internal/penpot"
	"github.com/estefrac/penpot-installer/internal/system"
)

// pendingOpKind identifica qué operación está pendiente de confirmación
type pendingOpKind int

const (
	opNone pendingOpKind = iota
	opStop
	opUpdate
	opUninstall
	opUninstallKeepData
	opUninstallWithData
	opInstall
)

// menuAction identifica la acción de cada ítem del menú
type menuAction int

const (
	actionNone menuAction = iota
	actionInstall
	actionStart
	actionStop
	actionUpdate
	actionStatus
	actionOpenBrowser
	actionUninstall
	actionQuit
)

// view representa cada pantalla del TUI
type view int

const (
	viewSplash           view = iota // Pantalla inicial cargando
	viewMenu                         // Menú principal
	viewInstall                      // Formulario de instalación
	viewStatus                       // Estado de contenedores
	viewConfirm                      // Confirmación de acción destructiva
	viewOperation                    // Operación en progreso (spinner)
	viewResult                       // Resultado de operación
	viewDockerInstall                // Confirmar instalación de Docker (Linux)
	viewDockerWindows                // Instrucciones Docker en Windows
	viewDockerNotRunning             // Docker instalado pero no corriendo
	viewUninstallData                // Segunda confirmación: ¿borrar datos?
)

// Common contiene el estado compartido por todas las vistas/sub-modelos
type Common struct {
	cfg           penpot.Config
	width         int
	height        int
	version       string
	installResult installer.InstallResult
	isInstalled   bool
	isRunning     bool
}

// Model es el modelo principal de Bubble Tea (Orquestador)
type Model struct {
	// Estado compartido
	common Common

	// Estado de la app
	currentView    view
	dockerReady    bool
	dockerChecking bool
	dockerOS       string // "linux" | "windows"
	splashReady    bool

	// Sub-modelos (Composición)
	splash    SplashModel
	menu      MenuModel
	install   InstallModel
	operation OperationModel
	confirm   ConfirmModel
	result    ResultModel
	docker    DockerModel
	status    StatusModel

	// Orquestación
	spinner       spinner.Model
	operationDone bool
	pendingOp     pendingOpKind
	logCh         <-chan string
}

// New crea un nuevo modelo con valores por defecto
func New(version string, installResult installer.InstallResult) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colorSecondary)

	m := Model{
		currentView: viewSplash,
		common: Common{
			cfg:           penpot.DefaultConfig(),
			version:       version,
			installResult: installResult,
		},
		spinner:        sp,
		dockerChecking: true,
	}

	// Inicializar sub-modelos
	m.install = NewInstallModel(m.common.cfg.InstallDir, m.common.cfg.Port)

	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		checkDockerCmd(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.common.width = msg.Width
		m.common.height = msg.Height
		return m, nil

	case msgChangeView:
		m.currentView = msg.newView
		return m, tea.ClearScreen

	case msgMenuAction:
		return m.handleMenuAction(msg.action)

	case msgInstallConfirm:
		m.common.cfg.InstallDir = msg.installDir
		m.common.cfg.Port = msg.port
		m.confirm = NewConfirmModel(fmt.Sprintf(
			"Instalar Penpot con:\n  Directorio: %s\n  Puerto: %s\n\n¿Confirmar?",
			m.common.cfg.InstallDir,
			m.common.cfg.Port,
		))
		m.pendingOp = opInstall
		m.currentView = viewConfirm
		return m, nil

	case msgConfirmResult:
		if msg.confirmed {
			return m.executePendingOp()
		}
		m.currentView = viewMenu
		return m, tea.ClearScreen

	case msgUninstallDataResult:
		if msg.deleteData {
			m.pendingOp = opUninstallWithData
		} else {
			m.pendingOp = opUninstallKeepData
		}
		return m.executePendingOp()

	case msgDockerInstallAction:
		if msg.install {
			m.operation = NewOperationModel(m.spinner, "Instalando Docker...")
			m.currentView = viewOperation
			return m, tea.Batch(m.spinner.Tick, installDockerCmd())
		}
		return m, tea.Quit

	case msgDockerStartAction:
		if msg.start {
			m.operation = NewOperationModel(m.spinner, "Iniciando Docker...")
			m.currentView = viewOperation
			return m, tea.Batch(m.spinner.Tick, startDockerCmd())
		}
		return m, tea.Quit

	case msgDockerReady:
		m.dockerReady = true
		m.dockerChecking = false
		m.common.isInstalled = penpot.IsInstalled(m.common.cfg)
		m.common.isRunning = penpot.IsRunning()
		m.splashReady = true
		m.splash, cmd = m.splash.Update(msg)
		return m, cmd

	case msgDockerNotInstalled:
		m.dockerChecking = false
		m.dockerOS = msg.os
		m.docker = DockerModel{os: msg.os, view: viewDockerInstall}
		if msg.os != "linux" {
			m.docker.view = viewDockerWindows
		}
		m.currentView = m.docker.view
		return m, tea.ClearScreen

	case msgDockerNotRunning:
		m.dockerChecking = false
		m.docker = DockerModel{view: viewDockerNotRunning}
		m.currentView = viewDockerNotRunning
		return m, tea.ClearScreen

	case msgLogLine:
		if msg.closed {
			m.logCh = nil
			return m, nil
		}
		m.operation, cmd = m.operation.Update(msg)
		return m, tea.Batch(cmd, waitForLogCmd(m.logCh))

	case msgDockerInstallDone:
		m.currentView = viewSplash
		return m, tea.Batch(m.spinner.Tick, checkDockerCmd())

	case msgDockerInstallError:
		m.result = ResultModel{message: fmt.Sprintf("Error instalando Docker: %v", msg.err), isError: true}
		m.currentView = viewResult
		return m, tea.ClearScreen

	case msgDockerError:
		m.result = ResultModel{message: fmt.Sprintf("Error con Docker: %v", msg.err), isError: true}
		m.currentView = viewResult
		return m, tea.ClearScreen

	case msgOperationDone:
		m.operationDone = true
		m.common.isInstalled = penpot.IsInstalled(m.common.cfg)
		m.common.isRunning = penpot.IsRunning()
		m.result = ResultModel{message: msg.message, isError: false}
		m.currentView = viewResult
		return m, tea.ClearScreen

	case msgOperationError:
		m.operationDone = true
		m.result = ResultModel{message: msg.err.Error(), isError: true}
		m.currentView = viewResult
		return m, tea.ClearScreen

	case msgStatusLoaded:
		m.common.isInstalled = msg.isInstalled
		m.common.isRunning = msg.isRunning
		m.status = StatusModel{containers: msg.containers}
		m.currentView = viewStatus
		return m, tea.ClearScreen
	}

	// Delegar al sub-modelo activo
	return m.delegateUpdate(msg)
}

func (m Model) delegateUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.currentView {
	case viewSplash:
		m.splash, cmd = m.splash.Update(msg)
	case viewMenu:
		m.menu, cmd = m.menu.Update(msg, m.buildMenuItems())
	case viewInstall:
		m.install, cmd = m.install.Update(msg)
	case viewStatus:
		m.status, cmd = m.status.Update(msg)
	case viewConfirm, viewUninstallData:
		m.confirm, cmd = m.confirm.Update(msg)
	case viewOperation:
		m.operation, cmd = m.operation.Update(msg)
	case viewResult:
		m.result, cmd = m.result.Update(msg)
	case viewDockerInstall, viewDockerWindows, viewDockerNotRunning:
		m.docker, cmd = m.docker.Update(msg)
	}
	return m, cmd
}

func (m Model) View() string {
	if m.common.width == 0 {
		return "Cargando..."
	}

	var content string
	switch m.currentView {
	case viewSplash:
		content = m.splash.View(m.common, m.spinner.View())
	case viewMenu:
		content = m.menu.View(m.common, m.buildMenuItems())
	case viewInstall:
		content = m.install.View(m.common)
	case viewStatus:
		content = m.status.View(m.common)
	case viewConfirm, viewUninstallData:
		content = m.confirm.View(m.common)
	case viewOperation:
		content = m.operation.View(m.common)
	case viewResult:
		content = m.result.View(m.common)
	case viewDockerInstall, viewDockerWindows, viewDockerNotRunning:
		content = m.docker.View(m.common)
	}

	return lipgloss.NewStyle().Padding(globalPadding, globalPadding).Render(content)
}

func (m Model) handleMenuAction(action menuAction) (tea.Model, tea.Cmd) {
	switch action {
	case actionInstall:
		m.currentView = viewInstall
		m.install = NewInstallModel(m.common.cfg.InstallDir, m.common.cfg.Port)
		return m, tea.ClearScreen
	case actionStart:
		m.operation = NewOperationModel(m.spinner, "Iniciando Penpot...")
		return m.startPenpotCmd()
	case actionStop:
		m.confirm = NewConfirmModel("¿Detener Penpot?")
		m.pendingOp = opStop
		m.currentView = viewConfirm
		return m, nil
	case actionUpdate:
		m.confirm = NewConfirmModel("Penpot se detendrá brevemente para actualizar.\n¿Continuar con la actualización?")
		m.pendingOp = opUpdate
		m.currentView = viewConfirm
		return m, nil
	case actionStatus:
		return m, loadStatusCmd(m.common.cfg)
	case actionOpenBrowser:
		_ = system.OpenBrowser(fmt.Sprintf("http://localhost:%s", m.common.cfg.Port))
		m.result = ResultModel{message: fmt.Sprintf("Abriendo http://localhost:%s...", m.common.cfg.Port)}
		m.currentView = viewResult
		return m, nil
	case actionUninstall:
		m.confirm = NewConfirmModel("¿Desinstalar Penpot?\n\nSe eliminarán los contenedores e imágenes Docker.")
		m.pendingOp = opUninstall
		m.currentView = viewConfirm
		return m, nil
	case actionQuit:
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) buildMenuItems() []MenuItem {
	items := []MenuItem{}
	if !m.common.isInstalled {
		items = append(items, MenuItem{Label: "🚀 Instalar Penpot", Action: actionInstall})
	} else {
		if m.common.isRunning {
			items = append(items,
				MenuItem{Label: "⏹️  Detener Penpot", Action: actionStop},
				MenuItem{Label: "🌐 Abrir en navegador", Action: actionOpenBrowser},
				MenuItem{Label: "📊 Ver estado", Action: actionStatus},
				MenuItem{Label: "🔄 Actualizar Penpot", Action: actionUpdate},
			)
		} else {
			items = append(items,
				MenuItem{Label: "▶️  Iniciar Penpot", Action: actionStart},
				MenuItem{Label: "📊 Ver estado", Action: actionStatus},
				MenuItem{Label: "🔄 Actualizar Penpot", Action: actionUpdate},
			)
		}
		items = append(items, MenuItem{Label: "🗑️  Desinstalar Penpot", Action: actionUninstall})
	}
	items = append(items, MenuItem{Label: "❌ Salir", Action: actionQuit})
	return items
}

const globalPadding = 2

func innerWidth(width int) int {
	w := width - globalPadding*2
	if w < 20 {
		return 20
	}
	return w
}

func innerHeight(height int) int {
	h := height - globalPadding*2
	if h < 10 {
		return 10
	}
	return h
}

func checkDockerCmd() tea.Cmd {
	return func() tea.Msg {
		if !docker.IsInstalled() {
			return msgDockerNotInstalled{os: runtime.GOOS}
		}
		if !docker.IsRunning() {
			return msgDockerNotRunning{}
		}
		if !docker.ComposeInstalled() {
			return msgDockerError{fmt.Errorf("docker compose no está disponible")}
		}
		return msgDockerReady{}
	}
}

func installDockerCmd() tea.Cmd {
	return func() tea.Msg {
		if err := docker.Install(); err != nil {
			return msgDockerInstallError{err}
		}
		return msgDockerInstallDone{}
	}
}

func startDockerCmd() tea.Cmd {
	return func() tea.Msg {
		if _, err := system.RunCommand("sudo", "systemctl", "start", "docker"); err != nil {
			return msgDockerInstallError{err}
		}
		return msgDockerInstallDone{}
	}
}

func (m Model) launchStreaming(op func(emit func(string)) error, doneMsg string) (Model, tea.Cmd) {
	resultCmd, ch := startStreamingCmd(op, doneMsg)
	m.logCh = ch
	m.operationDone = false
	m.currentView = viewOperation
	return m, tea.Batch(m.spinner.Tick, resultCmd, waitForLogCmd(ch))
}

func (m Model) installPenpotCmd() (Model, tea.Cmd) {
	return m.launchStreaming(
		func(emit func(string)) error { return penpot.InstallStreaming(m.common.cfg, emit) },
		"¡Penpot instalado correctamente! 🎨",
	)
}

func (m Model) startPenpotCmd() (Model, tea.Cmd) {
	return m.launchStreaming(
		func(emit func(string)) error { return penpot.StartStreaming(m.common.cfg, emit) },
		"Penpot iniciado ▶️",
	)
}

func (m Model) stopPenpotCmd() (Model, tea.Cmd) {
	return m.launchStreaming(
		func(emit func(string)) error { return penpot.StopStreaming(m.common.cfg, emit) },
		"Penpot detenido ⏹️",
	)
}

func (m Model) updatePenpotCmd() (Model, tea.Cmd) {
	return m.launchStreaming(
		func(emit func(string)) error { return penpot.UpdateStreaming(m.common.cfg, emit) },
		"¡Penpot actualizado correctamente! 🎉",
	)
}

func (m Model) uninstallPenpotCmd() (Model, tea.Cmd) {
	return m.launchStreaming(
		func(emit func(string)) error { return penpot.UninstallStreaming(m.common.cfg, emit) },
		"Penpot desinstalado correctamente.",
	)
}

func (m Model) uninstallKeepDataCmd() (Model, tea.Cmd) {
	return m.launchStreaming(
		func(emit func(string)) error { return penpot.UninstallKeepDataStreaming(m.common.cfg, emit) },
		"Penpot desinstalado correctamente (datos conservados).",
	)
}

func waitForLogCmd(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return msgLogLine{closed: true}
		}
		return msgLogLine{line: line}
	}
}

func startStreamingCmd(op func(emit func(string)) error, doneMsg string) (tea.Cmd, <-chan string) {
	lines := make(chan string, 256)
	resultCmd := func() tea.Msg {
		err := op(func(line string) {
			lines <- line
		})
		close(lines)
		if err != nil {
			return msgOperationError{err}
		}
		return msgOperationDone{doneMsg}
	}
	return resultCmd, lines
}

func loadStatusCmd(cfg penpot.Config) tea.Cmd {
	return func() tea.Msg {
		isInstalled := penpot.IsInstalled(cfg)
		isRunning := penpot.IsRunning()
		statuses, _ := penpot.Status()
		var containers []containerInfo
		for _, s := range statuses {
			containers = append(containers, containerInfo{
				name:   s.Name,
				status: s.Status,
				ports:  s.Ports,
			})
		}
		return msgStatusLoaded{
			isInstalled: isInstalled,
			isRunning:   isRunning,
			containers:  containers,
		}
	}
}

func (m Model) executePendingOp() (tea.Model, tea.Cmd) {
	switch m.pendingOp {
	case opInstall:
		m.operation = NewOperationModel(m.spinner, "Instalando Penpot...")
		return m.installPenpotCmd()
	case opStop:
		m.operation = NewOperationModel(m.spinner, "Deteniendo Penpot...")
		return m.stopPenpotCmd()
	case opUpdate:
		m.operation = NewOperationModel(m.spinner, "Actualizando Penpot...")
		return m.updatePenpotCmd()
	case opUninstall:
		m.currentView = viewUninstallData
		m.confirm = NewUninstallDataModel()
		return m, tea.ClearScreen
	case opUninstallKeepData:
		m.operation = NewOperationModel(m.spinner, "Desinstalando Penpot...")
		return m.uninstallKeepDataCmd()
	case opUninstallWithData:
		m.operation = NewOperationModel(m.spinner, "Desinstalando Penpot (incluyendo datos)...")
		return m.uninstallPenpotCmd()
	}
	m.currentView = viewMenu
	return m, tea.ClearScreen
}
