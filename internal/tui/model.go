package tui

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/estefrac/penpot-installer/internal/docker"
	"github.com/estefrac/penpot-installer/internal/penpot"
	"github.com/estefrac/penpot-installer/internal/system"
	"github.com/estefrac/penpot-installer/internal/updater"
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
	opSelfUpdate
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

// menuItem representa una opción del menú
type menuItem struct {
	label    string
	icon     string
	disabled bool
}

// Model es el modelo principal de Bubble Tea
type Model struct {
	// Estado de la app
	currentView    view
	cfg            penpot.Config
	isInstalled    bool
	isRunning      bool
	containers     []containerInfo
	dockerReady    bool
	dockerChecking bool
	dockerOS       string // "linux" | "windows" — para la vista de instalación de Docker

	// Versión y update
	version         string
	updateAvailable string // vacío si no hay update, o "vX.Y.Z" si hay

	// Menú
	menuItems  []menuItem
	menuCursor int

	// Formulario de instalación
	inputs     []textinput.Model
	inputFocus int

	// Operación en curso
	spinner       spinner.Model
	operationMsg  string
	operationDone bool
	resultMsg     string
	resultIsError bool

	// Confirmación
	confirmMsg string
	pendingOp  pendingOpKind
	confirmYes bool

	// Layout
	width  int
	height int

	// Logs de operación (mensajes filtrados para el usuario)
	logs []string

	// Canal de streaming en tiempo real
	logCh <-chan string

	// Progreso de servicios Docker
	servicesPulling map[string]bool // servicio → descargando
	servicesPulled  []string        // servicios completados en orden
	servicesStarted []string        // contenedores arrancados
}

// New crea un nuevo modelo con valores por defecto
func New(version string) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colorSecondary)

	// Inputs para el formulario de instalación
	dirInput := textinput.New()
	dirInput.Placeholder = "/home/user/penpot"
	dirInput.CharLimit = 256

	portInput := textinput.New()
	portInput.Placeholder = "9001"
	portInput.CharLimit = 5

	m := Model{
		currentView:    viewSplash,
		cfg:            penpot.DefaultConfig(),
		spinner:        sp,
		inputs:         []textinput.Model{dirInput, portInput},
		dockerChecking: true,
		version:        version,
	}

	m.inputs[0].SetValue(m.cfg.InstallDir)
	m.inputs[1].SetValue(m.cfg.Port)

	return m
}

// Init es el punto de entrada de Bubble Tea — retorna el primer comando
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		checkDockerCmd(),
		checkUpdateCmd(m.version),
	)
}

// Update procesa todos los mensajes (eventos de teclado, resultados async, etc.)
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case msgUpdateAvailable:
		m.updateAvailable = msg.latestVersion
		return m, nil

	case msgUpdateCheckDone:
		return m, nil

	case msgDockerReady:
		m.dockerReady = true
		m.dockerChecking = false
		m.isInstalled = penpot.IsInstalled(m.cfg)
		m.isRunning = penpot.IsRunning()
		m.buildMenuItems()
		m.currentView = viewMenu
		return m, tea.ClearScreen

	case msgDockerNotInstalled:
		m.dockerChecking = false
		m.dockerOS = msg.os
		if msg.os == "linux" {
			m.currentView = viewDockerInstall
		} else {
			m.currentView = viewDockerWindows
		}
		return m, tea.ClearScreen

	case msgDockerNotRunning:
		m.dockerChecking = false
		m.currentView = viewDockerNotRunning
		return m, tea.ClearScreen

	case msgLogLine:
		m.logs = append(m.logs, msg.line)
		return m, nil

	case msgPollLog:
		if m.logCh == nil {
			return m, nil
		}
		select {
		case line, ok := <-m.logCh:
			if !ok {
				// Canal cerrado — esperar resultado final
				m.logCh = nil
				return m, nil
			}
			m.processDockerLine(line)
			// Pedir la siguiente línea inmediatamente
			return m, pollLogCmd()
		default:
			// No hay línea todavía — volver a intentar en breve
			return m, pollLogCmd()
		}

	case msgDockerInstallDone:
		// Docker recién instalado — volver a verificar
		m.operationMsg = ""
		m.currentView = viewSplash
		return m, tea.Batch(m.spinner.Tick, checkDockerCmd())

	case msgDockerInstallError:
		m.resultMsg = fmt.Sprintf("Error instalando Docker: %s\n\nIntentá instalarlo manualmente:\n  curl -fsSL https://get.docker.com | sh", msg.err.Error())
		m.resultIsError = true
		m.currentView = viewResult
		return m, tea.ClearScreen

	case msgDockerError:
		m.dockerReady = false
		m.dockerChecking = false
		m.resultMsg = fmt.Sprintf("Error con Docker: %s", msg.err.Error())
		m.resultIsError = true
		m.currentView = viewResult
		return m, tea.ClearScreen

	case msgOperationDone:
		m.operationDone = true
		m.resultMsg = msg.message
		m.resultIsError = false
		m.isInstalled = penpot.IsInstalled(m.cfg)
		m.isRunning = penpot.IsRunning()
		m.buildMenuItems()
		m.currentView = viewResult
		return m, tea.ClearScreen

	case msgOperationError:
		m.operationDone = true
		m.resultMsg = msg.err.Error()
		m.resultIsError = true
		m.currentView = viewResult
		return m, tea.ClearScreen

	case msgStatusLoaded:
		m.isInstalled = msg.isInstalled
		m.isRunning = msg.isRunning
		m.containers = msg.containers
		m.currentView = viewStatus
		return m, tea.ClearScreen
	}

	// Delegar updates a componentes activos
	return m.updateActiveComponents(msg)
}

// updateActiveComponents actualiza los componentes que tienen foco
func (m Model) updateActiveComponents(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if m.currentView == viewInstall {
		for i := range m.inputs {
			var cmd tea.Cmd
			m.inputs[i], cmd = m.inputs[i].Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// handleKey procesa los eventos de teclado según la vista actual
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Ctrl+C y q siempre salen (excepto en inputs activos)
	if msg.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}

	switch m.currentView {
	case viewMenu:
		return m.handleMenuKey(msg)
	case viewInstall:
		return m.handleInstallKey(msg)
	case viewConfirm:
		return m.handleConfirmKey(msg)
	case viewResult:
		return m.handleResultKey(msg)
	case viewStatus:
		return m.handleStatusKey(msg)
	case viewDockerInstall:
		return m.handleDockerInstallKey(msg)
	case viewDockerWindows:
		return m.handleDockerWindowsKey(msg)
	case viewDockerNotRunning:
		return m.handleDockerNotRunningKey(msg)
	case viewUninstallData:
		return m.handleUninstallDataKey(msg)
	}

	return m, nil
}

func (m Model) handleMenuKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	activeItems := m.activeMenuItems()

	switch msg.Type {
	case tea.KeyUp, tea.KeyShiftTab:
		if m.menuCursor > 0 {
			m.menuCursor--
		} else {
			m.menuCursor = len(activeItems) - 1
		}
	case tea.KeyDown, tea.KeyTab:
		if m.menuCursor < len(activeItems)-1 {
			m.menuCursor++
		} else {
			m.menuCursor = 0
		}
	case tea.KeyEnter:
		if len(activeItems) > 0 && m.menuCursor < len(activeItems) {
			return m.executeMenuItem(activeItems[m.menuCursor].label)
		}
	}

	switch msg.String() {
	case "q", "Q":
		return m, tea.Quit
	}

	return m, nil
}

func (m Model) handleInstallKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyTab, tea.KeyDown:
		m.inputs[m.inputFocus].Blur()
		m.inputFocus = (m.inputFocus + 1) % len(m.inputs)
		m.inputs[m.inputFocus].Focus()
		return m, nil

	case tea.KeyShiftTab, tea.KeyUp:
		m.inputs[m.inputFocus].Blur()
		m.inputFocus = (m.inputFocus - 1 + len(m.inputs)) % len(m.inputs)
		m.inputs[m.inputFocus].Focus()
		return m, nil

	case tea.KeyEnter:
		// Si no es el último campo, avanzar al siguiente
		if m.inputFocus < len(m.inputs)-1 {
			m.inputs[m.inputFocus].Blur()
			m.inputFocus++
			m.inputs[m.inputFocus].Focus()
			return m, nil
		}
		// Último campo: confirmar instalación
		return m.startInstallConfirm()

	case tea.KeyEsc:
		m.currentView = viewMenu
		return m, tea.ClearScreen
	}

	// Pasar el evento al input activo
	var cmd tea.Cmd
	m.inputs[m.inputFocus], cmd = m.inputs[m.inputFocus].Update(msg)
	return m, cmd
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "s", "S":
		m.confirmYes = true
		return m.executePendingOp()
	case "n", "N", tea.KeyEsc.String():
		m.currentView = viewMenu
		return m, tea.ClearScreen
	case tea.KeyEnter.String():
		if m.confirmYes {
			return m.executePendingOp()
		}
		m.currentView = viewMenu
		return m, tea.ClearScreen
	case tea.KeyLeft.String(), tea.KeyRight.String(), tea.KeyTab.String():
		m.confirmYes = !m.confirmYes
	}
	return m, nil
}

func (m Model) handleDockerInstallKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "s", "S", "y", "Y":
		m.operationMsg = "Instalando Docker..."
		m.currentView = viewOperation
		return m, tea.Batch(m.spinner.Tick, installDockerCmd())
	case "n", "N", tea.KeyEsc.String():
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) handleDockerWindowsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		// Abrir browser y salir — el usuario tiene que reinstalar Docker Desktop y volver
		_ = system.OpenBrowser("https://desktop.docker.com/win/main/amd64/Docker%20Desktop%20Installer.exe")
		return m, tea.Quit
	case tea.KeyEsc:
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) handleDockerNotRunningKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "s", "S", "y", "Y":
		m.operationMsg = "Iniciando Docker..."
		m.currentView = viewOperation
		return m, tea.Batch(m.spinner.Tick, startDockerCmd())
	case "n", "N", tea.KeyEsc.String():
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) handleUninstallDataKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "s", "S", "y", "Y":
		// Borrar datos también
		m.pendingOp = opUninstallWithData
		return m.executePendingOp()
	case "n", "N":
		// Conservar datos
		m.pendingOp = opUninstallKeepData
		return m.executePendingOp()
	case tea.KeyEsc.String():
		m.currentView = viewMenu
		return m, tea.ClearScreen
	}
	return m, nil
}

func (m Model) handleResultKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter, tea.KeyEsc:
		m.currentView = viewMenu
		m.logs = nil
		return m, tea.ClearScreen
	}
	return m, nil
}

func (m Model) handleStatusKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter, tea.KeyEsc:
		m.currentView = viewMenu
		return m, tea.ClearScreen
	}
	switch msg.String() {
	case "q", "Q":
		return m, tea.Quit
	}
	return m, nil
}

// executeMenuItem ejecuta la acción según el ítem del menú seleccionado
func (m Model) executeMenuItem(label string) (tea.Model, tea.Cmd) {
	switch label {
	case "🚀 Instalar Penpot":
		m.currentView = viewInstall
		m.inputFocus = 0
		m.inputs[0].Focus()
		m.inputs[1].Blur()
		return m, tea.ClearScreen

	case "▶️  Iniciar Penpot":
		m.operationMsg = "Iniciando Penpot..."
		return m.startPenpotCmd()

	case "⏹️  Detener Penpot":
		m.confirmMsg = "¿Detener Penpot?"
		m.confirmYes = false
		m.pendingOp = opStop
		m.currentView = viewConfirm
		return m, nil

	case "🔄 Actualizar Penpot":
		m.confirmMsg = "Penpot se detendrá brevemente para actualizar.\n¿Continuar con la actualización?"
		m.confirmYes = true
		m.pendingOp = opUpdate
		m.currentView = viewConfirm
		return m, nil

	case "📊 Ver estado":
		return m, loadStatusCmd(m.cfg)

	case "🌐 Abrir en navegador":
		_ = system.OpenBrowser(fmt.Sprintf("http://localhost:%s", m.cfg.Port))
		m.resultMsg = fmt.Sprintf("Abriendo http://localhost:%s en el navegador...", m.cfg.Port)
		m.resultIsError = false
		m.currentView = viewResult
		return m, nil

	case "🗑️  Desinstalar Penpot":
		m.confirmMsg = "¿Desinstalar Penpot?\n\nSe eliminarán los contenedores e imágenes Docker.\nEn el siguiente paso podrás elegir qué hacer con tus datos."
		m.confirmYes = false
		m.pendingOp = opUninstall
		m.currentView = viewConfirm
		return m, nil

	case "❌ Salir":
		return m, tea.Quit
	}

	// Captura dinámica: "⬆️  Actualizar penpot-manager (vX.Y.Z)"
	if strings.HasPrefix(label, "⬆️") {
		m.operationMsg = fmt.Sprintf("Actualizando penpot-manager a %s...", m.updateAvailable)
		m.pendingOp = opSelfUpdate
		return m.executePendingOp()
	}

	return m, nil
}

// startInstallConfirm prepara la confirmación antes de instalar
func (m Model) startInstallConfirm() (tea.Model, tea.Cmd) {
	m.cfg.InstallDir = m.inputs[0].Value()
	m.cfg.Port = m.inputs[1].Value()

	m.confirmMsg = fmt.Sprintf(
		"Instalar Penpot con:\n  Directorio: %s\n  Puerto: %s\n\n¿Confirmar?",
		m.cfg.InstallDir,
		m.cfg.Port,
	)
	m.confirmYes = true
	m.pendingOp = opInstall
	m.currentView = viewConfirm
	return m, nil
}

// executePendingOp lanza la operación pendiente con streaming real
func (m Model) executePendingOp() (tea.Model, tea.Cmd) {
	switch m.pendingOp {
	case opInstall:
		m.operationMsg = "Instalando Penpot..."
		return m.installPenpotCmd()
	case opStop:
		m.operationMsg = "Deteniendo Penpot..."
		return m.stopPenpotCmd()
	case opUpdate:
		m.operationMsg = "Actualizando Penpot..."
		return m.updatePenpotCmd()
	case opUninstall:
		// Primera confirmación OK → pasar a la segunda pantalla (datos)
		m.currentView = viewUninstallData
		return m, tea.ClearScreen
	case opUninstallKeepData:
		m.operationMsg = "Desinstalando Penpot..."
		return m.uninstallKeepDataCmd()
	case opUninstallWithData:
		m.operationMsg = "Desinstalando Penpot (incluyendo datos)..."
		return m.uninstallPenpotCmd()
	case opSelfUpdate:
		return m.selfUpdateCmd()
	}
	m.currentView = viewMenu
	return m, tea.ClearScreen
}

// buildMenuItems construye las opciones del menú según el estado actual
func (m *Model) buildMenuItems() {
	items := []menuItem{}

	if !m.isInstalled {
		items = append(items,
			menuItem{label: "🚀 Instalar Penpot"},
		)
	} else {
		if m.isRunning {
			items = append(items,
				menuItem{label: "⏹️  Detener Penpot"},
				menuItem{label: "🌐 Abrir en navegador"},
				menuItem{label: "📊 Ver estado"},
				menuItem{label: "🔄 Actualizar Penpot"},
			)
		} else {
			items = append(items,
				menuItem{label: "▶️  Iniciar Penpot"},
				menuItem{label: "📊 Ver estado"},
				menuItem{label: "🔄 Actualizar Penpot"},
			)
		}
		items = append(items,
			menuItem{label: "🗑️  Desinstalar Penpot"},
		)
	}

	if m.updateAvailable != "" {
		items = append(items,
			menuItem{label: fmt.Sprintf("⬆️  Actualizar penpot-manager (%s)", m.updateAvailable)},
		)
	}

	items = append(items, menuItem{label: "❌ Salir"})

	m.menuItems = items
	if m.menuCursor >= len(items) {
		m.menuCursor = 0
	}
}

// activeMenuItems retorna solo los ítems habilitados
func (m Model) activeMenuItems() []menuItem {
	var active []menuItem
	for _, item := range m.menuItems {
		if !item.disabled {
			active = append(active, item)
		}
	}
	return active
}

// globalPadding es el margen interno de toda la TUI respecto a los bordes del terminal
const globalPadding = 2

// innerWidth retorna el ancho disponible descontando el padding global
func (m Model) innerWidth() int {
	w := m.width - globalPadding*2
	if w < 20 {
		return 20
	}
	return w
}

// innerHeight retorna la altura disponible descontando el padding global
func (m Model) innerHeight() int {
	h := m.height - globalPadding*2
	if h < 10 {
		return 10
	}
	return h
}

// View renderiza el TUI completo
func (m Model) View() string {
	if m.width == 0 {
		return "Cargando..."
	}

	var content string

	switch m.currentView {
	case viewSplash:
		content = m.renderSplash()
	case viewMenu:
		content = m.renderMain()
	case viewInstall:
		content = m.renderInstall()
	case viewStatus:
		content = m.renderStatus()
	case viewConfirm:
		content = m.renderConfirm()
	case viewOperation:
		content = m.renderOperation()
	case viewResult:
		content = m.renderResult()
	case viewDockerInstall:
		content = m.renderDockerInstall()
	case viewDockerWindows:
		content = m.renderDockerWindows()
	case viewDockerNotRunning:
		content = m.renderDockerNotRunning()
	case viewUninstallData:
		content = m.renderUninstallData()
	}

	return lipgloss.NewStyle().
		Padding(globalPadding, globalPadding).
		Render(content)
}

// renderSplash muestra la pantalla de carga inicial
func (m Model) renderSplash() string {
	w, h := m.innerWidth(), m.innerHeight()
	banner := RenderBanner(w)
	msg := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render(fmt.Sprintf("%s Verificando Docker...", m.spinner.View()))

	return lipgloss.Place(
		w, h,
		lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center, banner, "", msg),
	)
}

// renderMain muestra el layout principal: banner + menú + panel info
func (m Model) renderMain() string {
	w := m.innerWidth()
	banner := RenderBanner(w)

	// Panel izquierdo: menú
	menuPanel := m.renderMenuPanel()

	// Panel derecho: info de estado
	infoPanel := m.renderInfoPanel()

	// Layout horizontal de paneles
	panelWidth := w - 8
	menuW := 36
	infoW := panelWidth - menuW - 4
	if infoW < 20 {
		infoW = 20
	}

	menuStyled := menuPanelStyle.Width(menuW).Render(menuPanel)
	infoStyled := contentPanelStyle.Width(infoW).Render(infoPanel)

	panels := lipgloss.JoinHorizontal(lipgloss.Top, menuStyled, "  ", infoStyled)

	help := helpStyle.Render("↑↓ navegar  ·  enter seleccionar  ·  q salir")

	// Banner de update disponible
	var updateBanner string
	if m.updateAvailable != "" {
		updateBanner = lipgloss.NewStyle().
			Foreground(colorBg).
			Background(colorWarning).
			Bold(true).
			Padding(0, 2).
			Render(fmt.Sprintf("  Nueva versión disponible: %s  →  descargá el binario actualizado", m.updateAvailable))
		updateBanner = lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(updateBanner)
	}

	// Versión actual (pie de página)
	versionLine := ""
	if m.version != "" && m.version != "dev" {
		versionLine = lipgloss.NewStyle().Foreground(colorMuted).Render(m.version)
		versionLine = lipgloss.NewStyle().Width(w).Align(lipgloss.Right).Render(versionLine)
	}

	content := []string{banner, "", panels, ""}
	if updateBanner != "" {
		content = append(content, updateBanner, "")
	}
	content = append(content,
		lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(help),
		versionLine,
	)

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

// renderMenuPanel renderiza el panel lateral del menú
func (m Model) renderMenuPanel() string {
	title := sectionTitle.Render("MENÚ PRINCIPAL")

	items := m.activeMenuItems()
	var rows []string
	for i, item := range items {
		var row string
		if i == m.menuCursor {
			row = menuItemSelected.Render("▸ " + item.label)
		} else if item.disabled {
			row = menuItemDisabled.Render("  " + item.label)
		} else {
			row = menuItemNormal.Render("  " + item.label)
		}
		rows = append(rows, row)
	}

	return lipgloss.JoinVertical(lipgloss.Left, title, strings.Join(rows, "\n"))
}

// renderInfoPanel renderiza el panel de información de estado
func (m Model) renderInfoPanel() string {
	title := sectionTitle.Render("ESTADO DE PENPOT")

	// Badge de estado
	var statusBadge string
	if m.isRunning {
		statusBadge = badgeRunning.Render(" ● CORRIENDO ")
	} else if m.isInstalled {
		statusBadge = badgeStopped.Render(" ● DETENIDO ")
	} else {
		statusBadge = badgeNotInstalled.Render(" ○ NO INSTALADO ")
	}

	lines := []string{
		title,
		statusBadge,
		"",
	}

	if m.isInstalled {
		lines = append(lines,
			lipgloss.NewStyle().Foreground(colorMuted).Render("Directorio:"),
			highlightStyle.Render("  "+m.cfg.InstallDir),
			"",
			lipgloss.NewStyle().Foreground(colorMuted).Render("URL:"),
			highlightStyle.Render(fmt.Sprintf("  http://localhost:%s", m.cfg.Port)),
		)
	} else {
		lines = append(lines,
			lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render(
				"Penpot no está instalado.\nSeleccioná «Instalar Penpot»\npara comenzar.",
			),
		)
	}

	lines = append(lines,
		"",
		lipgloss.NewStyle().Foreground(colorMuted).Render("────────────────────"),
		"",
		lipgloss.NewStyle().Foreground(colorMuted).Render("Penpot es la alternativa"),
		lipgloss.NewStyle().Foreground(colorMuted).Render("open source a Figma."),
		lipgloss.NewStyle().Foreground(colorMuted).Render("Diseño + prototipado."),
	)

	return strings.Join(lines, "\n")
}

// renderInstall renderiza el formulario de instalación
func (m Model) renderInstall() string {
	w := m.innerWidth()
	banner := RenderBanner(w)
	title := sectionTitle.Render("INSTALACIÓN DE PENPOT")

	labels := []string{"Directorio de instalación:", "Puerto de acceso:"}
	var fields []string

	for i, input := range m.inputs {
		label := lipgloss.NewStyle().Foreground(colorText).Render(labels[i])
		var inputStyled string
		if i == m.inputFocus {
			inputStyled = inputActiveStyle.Render(input.View())
		} else {
			inputStyled = inputInactiveStyle.Render(input.View())
		}
		fields = append(fields, label, inputStyled, "")
	}

	formContent := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		strings.Join(fields, "\n"),
	)

	formPanel := contentPanelStyle.Width(60).Render(formContent)

	help := helpStyle.Render("tab/↓ siguiente campo  ·  enter confirmar  ·  esc volver")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		banner,
		"",
		lipgloss.NewStyle().Width(m.innerWidth()).Align(lipgloss.Center).Render(formPanel),
		"",
		lipgloss.NewStyle().Width(m.innerWidth()).Align(lipgloss.Center).Render(help),
	)
}

// renderStatus renderiza la vista de estado de contenedores
func (m Model) renderStatus() string {
	w := m.innerWidth()
	banner := RenderBanner(w)
	title := sectionTitle.Render("ESTADO DE CONTENEDORES")

	var rows []string
	rows = append(rows, title, "")

	if len(m.containers) == 0 {
		rows = append(rows, warningStyle.Render("No se encontraron contenedores de Penpot."))
	} else {
		// Header de tabla
		header := lipgloss.JoinHorizontal(lipgloss.Left,
			lipgloss.NewStyle().Width(30).Bold(true).Foreground(colorSecondary).Render("CONTENEDOR"),
			lipgloss.NewStyle().Width(20).Bold(true).Foreground(colorSecondary).Render("ESTADO"),
			lipgloss.NewStyle().Foreground(colorSecondary).Bold(true).Render("PUERTOS"),
		)
		rows = append(rows, header)
		rows = append(rows, lipgloss.NewStyle().Foreground(colorMuted).Render(strings.Repeat("─", 70)))

		for _, c := range m.containers {
			statusStyled := c.status
			if strings.Contains(strings.ToLower(c.status), "up") {
				statusStyled = successStyle.Render(c.status)
			} else {
				statusStyled = errorStyle.Render(c.status)
			}

			row := lipgloss.JoinHorizontal(lipgloss.Left,
				lipgloss.NewStyle().Width(30).Foreground(colorText).Render(c.name),
				lipgloss.NewStyle().Width(20).Render(statusStyled),
				lipgloss.NewStyle().Foreground(colorMuted).Render(c.ports),
			)
			rows = append(rows, row)
		}

		if m.isRunning {
			rows = append(rows, "", highlightStyle.Render(
				fmt.Sprintf("Penpot disponible en: http://localhost:%s", m.cfg.Port),
			))
		}
	}

	content := contentPanelStyle.Width(m.innerWidth() - 8).Render(strings.Join(rows, "\n"))
	help := helpStyle.Render("enter / esc  volver al menú")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		banner,
		"",
		lipgloss.NewStyle().Width(m.innerWidth()).Align(lipgloss.Center).Render(content),
		"",
		lipgloss.NewStyle().Width(m.innerWidth()).Align(lipgloss.Center).Render(help),
	)
}

// renderConfirm renderiza la pantalla de confirmación (pantalla completa)
func (m Model) renderConfirm() string {
	w, h := m.innerWidth(), m.innerHeight()
	boxW := 56
	if w < 62 {
		boxW = w - 6
	}

	msg := lipgloss.NewStyle().
		Foreground(colorText).
		Width(boxW - 4).
		Render(m.confirmMsg)

	// Botón base — mismo tamaño y borde para ambos
	btnBase := lipgloss.NewStyle().
		Width(10).
		Align(lipgloss.Center).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder())

	var yesLabel, noLabel string
	if m.confirmYes {
		// Sí: seleccionado — fondo verde, cursor visible
		yesLabel = btnBase.
			BorderForeground(colorSuccess).
			Background(colorSuccess).
			Foreground(colorBg).
			Bold(true).
			Render("▸  Sí")
		// No: inactivo — borde gris, sin cursor
		noLabel = btnBase.
			BorderForeground(colorMuted).
			Foreground(colorMuted).
			Render("   No")
	} else {
		// Sí: inactivo
		yesLabel = btnBase.
			BorderForeground(colorMuted).
			Foreground(colorMuted).
			Render("   Sí")
		// No: seleccionado — fondo rojo, cursor visible
		noLabel = btnBase.
			BorderForeground(colorError).
			Background(colorError).
			Foreground(colorBg).
			Bold(true).
			Render("▸  No")
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Center, yesLabel, "    ", noLabel)

	help := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render("←→ / tab  alternar   ·   enter  confirmar")

	inner := lipgloss.JoinVertical(lipgloss.Center,
		sectionTitle.Render("CONFIRMACIÓN"),
		"",
		msg,
		"",
		buttons,
		"",
		help,
	)

	box := lipgloss.NewStyle().
		Width(boxW).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Render(inner)

	return lipgloss.Place(
		w, h,
		lipgloss.Center, lipgloss.Center,
		box,
	)
}

// renderOperation muestra el spinner + progreso legible (pantalla completa)
func (m Model) renderOperation() string {
	w, h := m.innerWidth(), m.innerHeight()
	boxW := 72
	if w < 78 {
		boxW = w - 6
	}
	logW := boxW - 6

	// Header con spinner
	header := lipgloss.JoinHorizontal(lipgloss.Left,
		m.spinner.View(),
		" ",
		lipgloss.NewStyle().Foreground(colorText).Bold(true).Render(m.operationMsg),
	)

	// Barra de progreso de servicios si hay info
	var progressSection string
	total := len(m.servicesPulled) + len(m.servicesPulling) + len(m.servicesStarted)
	if total > 0 {
		pulled := len(m.servicesPulled)
		started := len(m.servicesStarted)

		// Barra visual: bloques llenos vs vacíos
		barWidth := logW - 10
		if barWidth < 10 {
			barWidth = 10
		}
		filled := 0
		if total > 0 {
			filled = (pulled + started) * barWidth / (total + 1)
		}
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
		barStyled := lipgloss.NewStyle().Foreground(colorSecondary).Render(bar)

		counter := lipgloss.NewStyle().Foreground(colorMuted).
			Render(fmt.Sprintf("%d/%d servicios", pulled+started, total))

		progressSection = lipgloss.JoinVertical(lipgloss.Left,
			barStyled,
			counter,
			"",
		)
	}

	// Últimas N líneas de log
	maxLogs := h/2 - 8
	if maxLogs < 3 {
		maxLogs = 3
	}

	var logLines []string
	if len(m.logs) == 0 {
		logLines = append(logLines,
			lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render("Conectando con Docker..."),
		)
	} else {
		recent := m.logs
		if len(recent) > maxLogs {
			recent = recent[len(recent)-maxLogs:]
		}
		for _, l := range recent {
			if len(l) > logW {
				l = l[:logW-3] + "..."
			}
			ll := strings.ToLower(l)
			var styled string
			switch {
			case strings.HasPrefix(l, "✓") || strings.HasPrefix(l, "▶"):
				styled = successStyle.Render(l)
			case strings.HasPrefix(l, "⬇"):
				styled = lipgloss.NewStyle().Foreground(colorSecondary).Render(l)
			case strings.Contains(ll, "error") || strings.Contains(ll, "failed"):
				styled = errorStyle.Render(l)
			default:
				styled = lipgloss.NewStyle().Foreground(colorMuted).Render(l)
			}
			logLines = append(logLines, styled)
		}
	}

	logArea := lipgloss.NewStyle().
		Width(logW).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorMuted).
		Padding(0, 1).
		Render(strings.Join(logLines, "\n"))

	note := lipgloss.NewStyle().
		Foreground(colorMuted).Italic(true).
		Render("Esto puede tardar varios minutos en la primera instalación...")

	inner := lipgloss.JoinVertical(lipgloss.Left,
		sectionTitle.Render("EN PROGRESO"),
		"",
		header,
		"",
		progressSection,
		logArea,
		"",
		note,
	)

	box := lipgloss.NewStyle().
		Width(boxW).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorSecondary).
		Render(inner)

	return lipgloss.Place(
		w, h,
		lipgloss.Center, lipgloss.Center,
		box,
	)
}

// renderResult muestra el resultado de una operación (pantalla completa)
func (m Model) renderResult() string {
	w, h := m.innerWidth(), m.innerHeight()
	boxW := 64
	if w < 70 {
		boxW = w - 6
	}

	var borderColor lipgloss.Color
	var title, icon string

	if m.resultIsError {
		borderColor = colorError
		title = errorStyle.Bold(true).Render("✗  Error")
		icon = ""
	} else {
		borderColor = colorSuccess
		title = successStyle.Bold(true).Render("✓  Listo")
		icon = ""
	}
	_ = icon

	msg := lipgloss.NewStyle().
		Foreground(colorText).
		Width(boxW - 4).
		Render(m.resultMsg)

	help := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render("↵ enter  /  esc  →  volver al menú")

	inner := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		msg,
		"",
		lipgloss.NewStyle().Foreground(colorMuted).Render(strings.Repeat("─", boxW-6)),
		"",
		help,
	)

	box := lipgloss.NewStyle().
		Width(boxW).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Render(inner)

	return lipgloss.Place(
		w, h,
		lipgloss.Center, lipgloss.Center,
		box,
	)
}

// renderDockerInstall — Linux: preguntar si instalar Docker
func (m Model) renderDockerInstall() string {
	boxW := 60
	inner := lipgloss.JoinVertical(lipgloss.Left,
		warningStyle.Bold(true).Render("⚠  Docker no está instalado"),
		"",
		lipgloss.NewStyle().Foreground(colorText).Width(boxW-4).Render(
			"Docker es necesario para correr Penpot.\n\n"+
				"Se instalará usando el script oficial de Docker:\n"+
				"  curl -fsSL https://get.docker.com | sh\n\n"+
				"Necesitás conexión a internet y permisos de administrador (sudo).",
		),
		"",
		lipgloss.NewStyle().Foreground(colorMuted).Render(strings.Repeat("─", boxW-6)),
		"",
		highlightStyle.Render("¿Instalar Docker ahora?"),
		"",
		lipgloss.NewStyle().Foreground(colorMuted).Render("s / y  instalar   ·   n / esc  salir"),
	)
	box := lipgloss.NewStyle().
		Width(boxW).Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorWarning).
		Render(inner)
	return lipgloss.Place(m.innerWidth(), m.innerHeight(), lipgloss.Center, lipgloss.Center, box)
}

// renderDockerWindows — Windows: instrucciones de instalación manual
func (m Model) renderDockerWindows() string {
	boxW := 64
	inner := lipgloss.JoinVertical(lipgloss.Left,
		warningStyle.Bold(true).Render("⚠  Docker no está instalado"),
		"",
		lipgloss.NewStyle().Foreground(colorText).Width(boxW-4).Render(
			"En Windows, Docker Desktop se instala de forma gráfica.\n\n"+
				"Al presionar Enter voy a abrir el navegador con el instalador oficial.\n\n"+
				"Una vez instalado y con Docker Desktop corriendo,\nvolvé a ejecutar el instalador.",
		),
		"",
		lipgloss.NewStyle().Foreground(colorMuted).Render(strings.Repeat("─", boxW-6)),
		"",
		lipgloss.NewStyle().Foreground(colorMuted).Render("enter  abrir instalador y salir   ·   esc  salir"),
	)
	box := lipgloss.NewStyle().
		Width(boxW).Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorWarning).
		Render(inner)
	return lipgloss.Place(m.innerWidth(), m.innerHeight(), lipgloss.Center, lipgloss.Center, box)
}

// renderDockerNotRunning — Docker instalado pero el daemon no corre
func (m Model) renderDockerNotRunning() string {
	boxW := 60
	inner := lipgloss.JoinVertical(lipgloss.Left,
		warningStyle.Bold(true).Render("⚠  Docker no está corriendo"),
		"",
		lipgloss.NewStyle().Foreground(colorText).Width(boxW-4).Render(
			"Docker está instalado pero el daemon no está activo.\n\n"+
				"¿Querés que lo inicie ahora?\n"+
				"  (equivale a: sudo systemctl start docker)",
		),
		"",
		lipgloss.NewStyle().Foreground(colorMuted).Render(strings.Repeat("─", boxW-6)),
		"",
		lipgloss.NewStyle().Foreground(colorMuted).Render("s / y  iniciar Docker   ·   n / esc  salir"),
	)
	box := lipgloss.NewStyle().
		Width(boxW).Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorWarning).
		Render(inner)
	return lipgloss.Place(m.innerWidth(), m.innerHeight(), lipgloss.Center, lipgloss.Center, box)
}

// renderUninstallData — segunda confirmación: ¿borrar datos o conservarlos?
func (m Model) renderUninstallData() string {
	w, h := m.innerWidth(), m.innerHeight()
	boxW := 62
	if w < 68 {
		boxW = w - 6
	}

	inner := lipgloss.JoinVertical(lipgloss.Left,
		errorStyle.Bold(true).Render("🗑️  ¿Qué hacemos con tus datos?"),
		"",
		lipgloss.NewStyle().Foreground(colorText).Width(boxW-4).Render(
			"Los contenedores e imágenes Docker serán eliminados en cualquier caso.\n\n"+
				"Tus proyectos y archivos están guardados en volúmenes de Docker.",
		),
		"",
		lipgloss.NewStyle().Foreground(colorMuted).Render(strings.Repeat("─", boxW-6)),
		"",
		lipgloss.NewStyle().Foreground(colorSuccess).Bold(true).Render("n  Conservar datos"),
		lipgloss.NewStyle().Foreground(colorMuted).Render("   Los volúmenes quedan en Docker."),
		lipgloss.NewStyle().Foreground(colorMuted).Render("   Reinstalando Penpot los recuperás."),
		"",
		lipgloss.NewStyle().Foreground(colorError).Bold(true).Render("s  Borrar todo"),
		lipgloss.NewStyle().Foreground(colorMuted).Render("   ⚠️  Irreversible. Proyectos y archivos"),
		lipgloss.NewStyle().Foreground(colorMuted).Render("   eliminados para siempre."),
		"",
		lipgloss.NewStyle().Foreground(colorMuted).Render(strings.Repeat("─", boxW-6)),
		"",
		lipgloss.NewStyle().Foreground(colorMuted).Render("esc  cancelar y volver al menú"),
	)

	box := lipgloss.NewStyle().
		Width(boxW).Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorError).
		Render(inner)

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, box)
}

// checkUpdateCmd consulta GitHub en background para ver si hay una versión nueva
func checkUpdateCmd(currentVersion string) tea.Cmd {
	return func() tea.Msg {
		result := updater.Check(currentVersion)
		if result.HasUpdate {
			return msgUpdateAvailable{latestVersion: result.LatestVersion}
		}
		return msgUpdateCheckDone{}
	}
}

// checkDockerCmd verifica si Docker está disponible
func checkDockerCmd() tea.Cmd {
	return func() tea.Msg {
		if !docker.IsInstalled() {
			return msgDockerNotInstalled{os: runtime.GOOS}
		}
		if !docker.IsRunning() {
			return msgDockerNotRunning{}
		}
		if !docker.ComposeInstalled() {
			return msgDockerError{fmt.Errorf("docker compose no está disponible. Actualizá Docker a una versión reciente")}
		}
		return msgDockerReady{}
	}
}

// installDockerCmd instala Docker en Linux usando el script oficial
func installDockerCmd() tea.Cmd {
	return func() tea.Msg {
		if err := docker.Install(); err != nil {
			return msgDockerInstallError{err}
		}
		return msgDockerInstallDone{}
	}
}

// selfUpdateCmd descarga el binario nuevo y reemplaza el ejecutable actual
func (m Model) selfUpdateCmd() (Model, tea.Cmd) {
	m.logs = nil
	m.servicesPulling = make(map[string]bool)
	m.servicesPulled = nil
	m.servicesStarted = nil
	m.operationDone = false
	m.currentView = viewOperation

	cmd := func() tea.Msg {
		if err := updater.SelfUpdate(m.updateAvailable, func(line string) {}); err != nil {
			return msgOperationError{err}
		}
		return msgOperationDone{"¡penpot-manager actualizado correctamente!\n\nReiniciá el programa para usar la nueva versión."}
	}

	return m, tea.Batch(m.spinner.Tick, cmd, pollLogCmd())
}

// startDockerCmd inicia el daemon de Docker
func startDockerCmd() tea.Cmd {
	return func() tea.Msg {
		if _, err := system.RunCommand("sudo", "systemctl", "start", "docker"); err != nil {
			return msgDockerInstallError{fmt.Errorf("no se pudo iniciar Docker: %w\n\nIntentá manualmente: sudo systemctl start docker", err)}
		}
		return msgDockerInstallDone{}
	}
}

// processDockerLine parsea una línea de output de Docker y la convierte
// en un mensaje legible para el usuario, ignorando el ruido.
func (m *Model) processDockerLine(line string) {
	l := strings.TrimSpace(line)
	ll := strings.ToLower(l)

	// Inicializar mapa si hace falta
	if m.servicesPulling == nil {
		m.servicesPulling = make(map[string]bool)
	}

	// Líneas de ruido — ignorar completamente
	noisePatterns := []string{
		"pulling fs layer", "verifying checksum", "waiting",
		"already exists", "digest:", "sha256:", "status: image",
		"using default tag",
	}
	for _, noise := range noisePatterns {
		if strings.Contains(ll, noise) {
			return
		}
	}

	// Servicio empezando a descargar: "penpot-backend Pulling"
	if strings.HasSuffix(ll, " pulling") {
		svc := strings.TrimSuffix(l, " Pulling")
		svc = strings.TrimSuffix(svc, " pulling")
		m.servicesPulling[svc] = true
		m.logs = append(m.logs, fmt.Sprintf("⬇  Descargando %s...", svc))
		return
	}

	// Servicio descargado: "penpot-backend Pulled"
	if strings.HasSuffix(ll, " pulled") {
		svc := strings.TrimSuffix(l, " Pulled")
		svc = strings.TrimSuffix(svc, " pulled")
		delete(m.servicesPulling, svc)
		m.servicesPulled = append(m.servicesPulled, svc)
		m.logs = append(m.logs, fmt.Sprintf("✓  %s descargado", svc))
		return
	}

	// Contenedor arrancado: "penpot-backend Started"
	if strings.HasSuffix(ll, " started") {
		svc := strings.TrimSuffix(l, " Started")
		svc = strings.TrimSuffix(svc, " started")
		m.servicesStarted = append(m.servicesStarted, svc)
		m.logs = append(m.logs, fmt.Sprintf("▶  %s iniciado", svc))
		return
	}

	// "Download complete" por capa — ignorar, ya lo cubrimos con Pulled
	if strings.Contains(ll, "download complete") || strings.Contains(ll, "pull complete") {
		return
	}

	// Cualquier otra línea con contenido útil — mostrar tal cual
	if l != "" {
		m.logs = append(m.logs, l)
	}
}

// launchStreaming arranca una operación con streaming real y retorna
// el modelo actualizado (con logCh asignado) y los Cmds necesarios.
func (m Model) launchStreaming(op func(emit func(string)) error, doneMsg string) (Model, tea.Cmd) {
	resultCmd, ch := startStreamingCmd(op, doneMsg)
	m.logCh = ch
	m.logs = nil
	m.servicesPulling = make(map[string]bool)
	m.servicesPulled = nil
	m.servicesStarted = nil
	m.operationDone = false
	m.currentView = viewOperation
	return m, tea.Batch(m.spinner.Tick, resultCmd, pollLogCmd())
}

// installPenpotCmd ejecuta la instalación de Penpot con streaming
func (m Model) installPenpotCmd() (Model, tea.Cmd) {
	return m.launchStreaming(
		func(emit func(string)) error { return penpot.InstallStreaming(m.cfg, emit) },
		fmt.Sprintf("¡Penpot instalado correctamente! 🎨\n\nAccedé en: http://localhost:%s\n\nLa primera vez puede tardar unos segundos en estar listo.", m.cfg.Port),
	)
}

// startPenpotCmd inicia Penpot con streaming
func (m Model) startPenpotCmd() (Model, tea.Cmd) {
	return m.launchStreaming(
		func(emit func(string)) error { return penpot.StartStreaming(m.cfg, emit) },
		fmt.Sprintf("Penpot iniciado ▶️\n\nAccedé en: http://localhost:%s", m.cfg.Port),
	)
}

// stopPenpotCmd detiene Penpot con streaming
func (m Model) stopPenpotCmd() (Model, tea.Cmd) {
	return m.launchStreaming(
		func(emit func(string)) error { return penpot.StopStreaming(m.cfg, emit) },
		"Penpot detenido ⏹️",
	)
}

// updatePenpotCmd actualiza Penpot con streaming
func (m Model) updatePenpotCmd() (Model, tea.Cmd) {
	return m.launchStreaming(
		func(emit func(string)) error { return penpot.UpdateStreaming(m.cfg, emit) },
		"¡Penpot actualizado correctamente! 🎉",
	)
}

// uninstallPenpotCmd desinstala Penpot borrando también volúmenes e imágenes
func (m Model) uninstallPenpotCmd() (Model, tea.Cmd) {
	return m.launchStreaming(
		func(emit func(string)) error { return penpot.UninstallStreaming(m.cfg, emit) },
		"Penpot desinstalado correctamente.\n\nTodos los datos fueron eliminados.",
	)
}

// uninstallKeepDataCmd desinstala Penpot conservando los volúmenes con los datos
func (m Model) uninstallKeepDataCmd() (Model, tea.Cmd) {
	return m.launchStreaming(
		func(emit func(string)) error { return penpot.UninstallKeepDataStreaming(m.cfg, emit) },
		"Penpot desinstalado correctamente.\n\nTus datos (proyectos y archivos) fueron conservados en los volúmenes de Docker.\nPodés reinstalar Penpot y recuperarlos.",
	)
}

// pollLogCmd retorna un Cmd que emite msgPollLog para seguir leyendo el canal
func pollLogCmd() tea.Cmd {
	return func() tea.Msg { return msgPollLog{} }
}

// startStreamingCmd arranca la operación en background y retorna dos Cmds:
// uno para abrir el canal de logs (asignado al modelo vía msgStreamStarted)
// y uno para escuchar el resultado final.
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

// loadStatusCmd carga el estado de los contenedores
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
