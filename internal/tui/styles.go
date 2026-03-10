package tui

import "github.com/charmbracelet/lipgloss"

// Paleta de colores de Penpot
var (
	colorPrimary   = lipgloss.Color("#7238B9") // Penpot purple
	colorSecondary = lipgloss.Color("#31EFB8") // Penpot teal/mint
	colorAccent    = lipgloss.Color("#E65F47") // Penpot orange-red
	colorBg        = lipgloss.Color("#1E1E2E") // Dark background
	colorSurface   = lipgloss.Color("#2A2A3E") // Surface
	colorMuted     = lipgloss.Color("#6C7086") // Muted text
	colorText      = lipgloss.Color("#CDD6F4") // Main text
	colorSuccess   = lipgloss.Color("#A6E3A1") // Green
	colorWarning   = lipgloss.Color("#F9E2AF") // Yellow
	colorError     = lipgloss.Color("#F38BA8") // Red
)

// Estilos del layout principal
var (
	// Panel izquierdo - menú
	menuPanelStyle = lipgloss.NewStyle().
			Width(32).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary)

	// Panel derecho - contenido/info
	contentPanelStyle = lipgloss.NewStyle().
				Padding(1, 2).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorMuted)

	// Ítem del menú seleccionado
	menuItemSelected = lipgloss.NewStyle().
				Foreground(colorBg).
				Background(colorPrimary).
				Padding(0, 1).
				Bold(true)

	// Ítem del menú normal
	menuItemNormal = lipgloss.NewStyle().
			Foreground(colorText).
			Padding(0, 1)

	// Ítem del menú deshabilitado
	menuItemDisabled = lipgloss.NewStyle().
				Foreground(colorMuted).
				Padding(0, 1)

	// Título de sección
	sectionTitle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true).
			MarginBottom(1)

	// Badge de estado "running"
	badgeRunning = lipgloss.NewStyle().
			Foreground(colorBg).
			Background(colorSuccess).
			Padding(0, 1).
			Bold(true)

	// Badge de estado "stopped"
	badgeStopped = lipgloss.NewStyle().
			Foreground(colorBg).
			Background(colorError).
			Padding(0, 1).
			Bold(true)

	// Badge de estado "not installed"
	badgeNotInstalled = lipgloss.NewStyle().
				Foreground(colorBg).
				Background(colorMuted).
				Padding(0, 1)

	// Texto de ayuda / footer
	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			MarginTop(1)

	// Texto destacado
	highlightStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	// Error inline
	errorStyle = lipgloss.NewStyle().
			Foreground(colorError)

	// Warning inline
	warningStyle = lipgloss.NewStyle().
			Foreground(colorWarning)

	// Success inline
	successStyle = lipgloss.NewStyle().
			Foreground(colorSuccess)

	// Input activo
	inputActiveStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorSecondary).
				Padding(0, 1)

	// Input inactivo
	inputInactiveStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorMuted).
				Padding(0, 1)
)
