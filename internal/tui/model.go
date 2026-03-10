package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/estefrac/penpot-installer/internal/docker"
	"github.com/estefrac/penpot-installer/internal/penpot"
	"github.com/estefrac/penpot-installer/internal/system"
)

// view representa cada pantalla del TUI
type view int

const (
	viewSplash    view = iota // Pantalla inicial cargando
	viewMenu                  // Menú principal
	viewInstall               // Formulario de instalación
	viewStatus                // Estado de contenedores
	viewConfirm               // Confirmación de acción destructiva
	viewOperation             // Operación en progreso (spinner)
	viewResult                // Resultado de operación
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
	confirmMsg    string
	confirmAction func() tea.Cmd
	confirmYes    bool

	// Layout
	width  int
	height int

	// Logs de operación (para mostrar progreso)
	logs []string
}

// New crea un nuevo modelo con valores por defecto
func New() Model {
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

	case msgDockerReady:
		m.dockerReady = true
		m.dockerChecking = false
		m.isInstalled = penpot.IsInstalled(m.cfg)
		m.isRunning = penpot.IsRunning()
		m.buildMenuItems()
		m.currentView = viewMenu
		return m, nil

	case msgDockerError:
		m.dockerReady = false
		m.dockerChecking = false
		m.resultMsg = fmt.Sprintf("Error con Docker: %s", msg.err.Error())
		m.resultIsError = true
		m.currentView = viewResult
		return m, nil

	case msgOperationDone:
		m.operationDone = true
		m.resultMsg = msg.message
		m.resultIsError = false
		m.isInstalled = penpot.IsInstalled(m.cfg)
		m.isRunning = penpot.IsRunning()
		m.buildMenuItems()
		m.currentView = viewResult
		return m, nil

	case msgOperationError:
		m.operationDone = true
		m.resultMsg = msg.err.Error()
		m.resultIsError = true
		m.currentView = viewResult
		return m, nil

	case msgStatusLoaded:
		m.isInstalled = msg.isInstalled
		m.isRunning = msg.isRunning
		m.containers = msg.containers
		m.currentView = viewStatus
		return m, nil
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
		return m.confirmInstall()

	case tea.KeyEsc:
		m.currentView = viewMenu
		return m, nil
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
		m.currentView = viewOperation
		m.operationDone = false
		return m, tea.Batch(m.spinner.Tick, m.confirmAction())
	case "n", "N", tea.KeyEsc.String():
		m.currentView = viewMenu
	case tea.KeyEnter.String():
		if m.confirmYes {
			m.currentView = viewOperation
			m.operationDone = false
			return m, tea.Batch(m.spinner.Tick, m.confirmAction())
		}
		m.currentView = viewMenu
	case tea.KeyLeft.String(), tea.KeyRight.String(), tea.KeyTab.String():
		m.confirmYes = !m.confirmYes
	}
	return m, nil
}

func (m Model) handleResultKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter, tea.KeyEsc:
		m.currentView = viewMenu
		m.logs = nil
	}
	return m, nil
}

func (m Model) handleStatusKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter, tea.KeyEsc:
		m.currentView = viewMenu
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
		return m, nil

	case "▶️  Iniciar Penpot":
		m.operationMsg = "Iniciando Penpot..."
		m.currentView = viewOperation
		m.operationDone = false
		return m, tea.Batch(m.spinner.Tick, startPenpotCmd(m.cfg))

	case "⏹️  Detener Penpot":
		m.confirmMsg = "¿Detener Penpot?"
		m.confirmYes = false
		m.confirmAction = func() tea.Cmd { return stopPenpotCmd(m.cfg) }
		m.currentView = viewConfirm
		return m, nil

	case "🔄 Actualizar Penpot":
		m.confirmMsg = "Penpot se detendrá brevemente para actualizar.\n¿Continuar con la actualización?"
		m.confirmYes = true
		m.confirmAction = func() tea.Cmd {
			m.operationMsg = "Actualizando Penpot..."
			return updatePenpotCmd(m.cfg)
		}
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
		m.confirmMsg = "⚠️  Esta acción eliminará Penpot y TODOS sus datos.\n¿Estás seguro?"
		m.confirmYes = false
		m.confirmAction = func() tea.Cmd {
			m.operationMsg = "Desinstalando Penpot..."
			return uninstallPenpotCmd(m.cfg)
		}
		m.currentView = viewConfirm
		return m, nil

	case "❌ Salir":
		return m, tea.Quit
	}

	return m, nil
}

// confirmInstall prepara la confirmación antes de instalar
func (m Model) confirmInstall() (tea.Model, tea.Cmd) {
	m.cfg.InstallDir = m.inputs[0].Value()
	m.cfg.Port = m.inputs[1].Value()

	m.confirmMsg = fmt.Sprintf(
		"Instalar Penpot con:\n  Directorio: %s\n  Puerto: %s\n\n¿Confirmar?",
		m.cfg.InstallDir,
		m.cfg.Port,
	)
	m.confirmYes = true
	m.confirmAction = func() tea.Cmd {
		m.operationMsg = "Instalando Penpot..."
		return installPenpotCmd(m.cfg)
	}
	m.currentView = viewConfirm
	return m, nil
}

// buildMenuItems construye las opciones del menú según el estado actual
func (m *Model) buildMenuItems() {
	items := []menuItem{}

	if !m.isInstalled {
		items = append(items,
			menuItem{label: "🚀 Instalar Penpot", icon: "🚀"},
			menuItem{label: "❌ Salir", icon: "❌"},
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
			menuItem{label: "❌ Salir"},
		)
	}

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

// View renderiza el TUI completo
func (m Model) View() string {
	if m.width == 0 {
		return "Cargando..."
	}

	switch m.currentView {
	case viewSplash:
		return m.renderSplash()
	case viewMenu:
		return m.renderMain()
	case viewInstall:
		return m.renderInstall()
	case viewStatus:
		return m.renderStatus()
	case viewConfirm:
		return m.renderConfirm()
	case viewOperation:
		return m.renderOperation()
	case viewResult:
		return m.renderResult()
	}

	return ""
}

// renderSplash muestra la pantalla de carga inicial
func (m Model) renderSplash() string {
	banner := RenderBanner(m.width)
	msg := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render(fmt.Sprintf("%s Verificando Docker...", m.spinner.View()))

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center, banner, "", msg),
	)
}

// renderMain muestra el layout principal: banner + menú + panel info
func (m Model) renderMain() string {
	banner := RenderBanner(m.width)

	// Panel izquierdo: menú
	menuPanel := m.renderMenuPanel()

	// Panel derecho: info de estado
	infoPanel := m.renderInfoPanel()

	// Layout horizontal de paneles
	panelWidth := m.width - 8
	menuW := 36
	infoW := panelWidth - menuW - 4
	if infoW < 20 {
		infoW = 20
	}

	menuStyled := menuPanelStyle.Width(menuW).Render(menuPanel)
	infoStyled := contentPanelStyle.Width(infoW).Render(infoPanel)

	panels := lipgloss.JoinHorizontal(lipgloss.Top, menuStyled, "  ", infoStyled)

	help := helpStyle.Render("↑↓ navegar  ·  enter seleccionar  ·  q salir")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		banner,
		"",
		panels,
		"",
		lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(help),
	)
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
	banner := RenderBanner(m.width)
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
		lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(formPanel),
		"",
		lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(help),
	)
}

// renderStatus renderiza la vista de estado de contenedores
func (m Model) renderStatus() string {
	banner := RenderBanner(m.width)
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

	content := contentPanelStyle.Width(m.width - 8).Render(strings.Join(rows, "\n"))
	help := helpStyle.Render("enter / esc  volver al menú")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		banner,
		"",
		lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(content),
		"",
		lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(help),
	)
}

// renderConfirm renderiza la pantalla de confirmación
func (m Model) renderConfirm() string {
	banner := RenderBanner(m.width)

	msg := lipgloss.NewStyle().
		Foreground(colorText).
		Width(50).
		Render(m.confirmMsg)

	var yesStyle, noStyle lipgloss.Style
	if m.confirmYes {
		yesStyle = lipgloss.NewStyle().Background(colorSuccess).Foreground(colorBg).Padding(0, 2).Bold(true)
		noStyle = lipgloss.NewStyle().Foreground(colorMuted).Padding(0, 2).Border(lipgloss.RoundedBorder()).BorderForeground(colorMuted)
	} else {
		yesStyle = lipgloss.NewStyle().Foreground(colorMuted).Padding(0, 2).Border(lipgloss.RoundedBorder()).BorderForeground(colorMuted)
		noStyle = lipgloss.NewStyle().Background(colorError).Foreground(colorBg).Padding(0, 2).Bold(true)
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Center,
		yesStyle.Render("  Sí  "),
		"   ",
		noStyle.Render("  No  "),
	)

	box := contentPanelStyle.Width(56).Render(
		lipgloss.JoinVertical(lipgloss.Center,
			sectionTitle.Render("CONFIRMACIÓN"),
			"",
			msg,
			"",
			buttons,
		),
	)

	help := helpStyle.Render("s/y confirmar  ·  n cancelar  ·  ←→ / tab alternar  ·  enter aceptar")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		banner,
		"",
		lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(box),
		"",
		lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(help),
	)
}

// renderOperation muestra el spinner durante una operación (pantalla completa)
func (m Model) renderOperation() string {
	spinnerView := lipgloss.JoinHorizontal(lipgloss.Left,
		m.spinner.View(),
		" ",
		lipgloss.NewStyle().Foreground(colorText).Bold(true).Render(m.operationMsg),
	)

	note := lipgloss.NewStyle().
		Foreground(colorMuted).
		Italic(true).
		Render("Esto puede tardar varios minutos...")

	inner := lipgloss.JoinVertical(lipgloss.Left,
		sectionTitle.Render("EN PROGRESO"),
		"",
		spinnerView,
		"",
		note,
	)

	boxW := 64
	if m.width < 70 {
		boxW = m.width - 6
	}

	box := lipgloss.NewStyle().
		Width(boxW).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorSecondary).
		Render(inner)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		box,
	)
}

// renderResult muestra el resultado de una operación (pantalla completa)
func (m Model) renderResult() string {
	boxW := 64
	if m.width < 70 {
		boxW = m.width - 6
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
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		box,
	)
}

// checkDockerCmd verifica si Docker está disponible
func checkDockerCmd() tea.Cmd {
	return func() tea.Msg {
		if !docker.IsInstalled() {
			return msgDockerError{fmt.Errorf("Docker no está instalado. Instalalo y volvé a ejecutar")}
		}
		if !docker.IsRunning() {
			return msgDockerError{fmt.Errorf("Docker no está corriendo. Inicialo con: sudo systemctl start docker")}
		}
		if !docker.ComposeInstalled() {
			return msgDockerError{fmt.Errorf("docker compose no está disponible. Actualizá Docker")}
		}
		return msgDockerReady{}
	}
}

// installPenpotCmd ejecuta la instalación de Penpot
func installPenpotCmd(cfg penpot.Config) tea.Cmd {
	return func() tea.Msg {
		if err := penpot.Install(cfg); err != nil {
			return msgOperationError{err}
		}
		return msgOperationDone{fmt.Sprintf(
			"¡Penpot instalado correctamente! 🎨\n\nAccedé en: http://localhost:%s\n\nLa primera vez puede tardar unos segundos en estar listo.",
			cfg.Port,
		)}
	}
}

// startPenpotCmd inicia Penpot
func startPenpotCmd(cfg penpot.Config) tea.Cmd {
	return func() tea.Msg {
		if err := penpot.Start(cfg); err != nil {
			return msgOperationError{err}
		}
		return msgOperationDone{fmt.Sprintf("Penpot iniciado ▶️\n\nAccedé en: http://localhost:%s", cfg.Port)}
	}
}

// stopPenpotCmd detiene Penpot
func stopPenpotCmd(cfg penpot.Config) tea.Cmd {
	return func() tea.Msg {
		if err := penpot.Stop(cfg); err != nil {
			return msgOperationError{err}
		}
		return msgOperationDone{"Penpot detenido ⏹️"}
	}
}

// updatePenpotCmd actualiza Penpot
func updatePenpotCmd(cfg penpot.Config) tea.Cmd {
	return func() tea.Msg {
		if err := penpot.Update(cfg); err != nil {
			return msgOperationError{err}
		}
		return msgOperationDone{"¡Penpot actualizado correctamente! 🎉"}
	}
}

// uninstallPenpotCmd desinstala Penpot
func uninstallPenpotCmd(cfg penpot.Config) tea.Cmd {
	return func() tea.Msg {
		if err := penpot.Uninstall(cfg, true); err != nil {
			return msgOperationError{err}
		}
		return msgOperationDone{"Penpot desinstalado correctamente."}
	}
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
