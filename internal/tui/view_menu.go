package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MenuModel maneja el menú principal y el dashboard de estado
type MenuModel struct {
	cursor int
}

// MenuItem representa una opción del menú
type MenuItem struct {
	Label    string
	Action   menuAction
	Disabled bool
}

func (m MenuModel) Update(msg tea.Msg, items []MenuItem) (MenuModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp, tea.KeyShiftTab:
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(items) - 1
			}
		case tea.KeyDown, tea.KeyTab:
			if m.cursor < len(items)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}
		case tea.KeyEnter:
			if len(items) > 0 && m.cursor < len(items) {
				return m, func() tea.Msg {
					return msgMenuAction{action: items[m.cursor].Action}
				}
			}
		}

		switch msg.String() {
		case "q", "Q":
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m MenuModel) View(common Common, items []MenuItem) string {
	w := innerWidth(common.width)
	banner := RenderBanner(w)

	// Panel izquierdo: menú
	menuPanel := m.renderMenuPanel(items)

	// Panel derecho: info de estado
	infoPanel := m.renderInfoPanel(common)

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

	// Versión actual (pie de página)
	versionLine := ""
	if common.version != "" && common.version != "dev" {
		versionLine = lipgloss.NewStyle().Foreground(colorMuted).Render(common.version)
		versionLine = lipgloss.NewStyle().Width(w).Align(lipgloss.Right).Render(versionLine)
	}

	content := []string{banner, "", panels, ""}
	content = append(content,
		lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(help),
		versionLine,
	)

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

func (m MenuModel) renderMenuPanel(items []MenuItem) string {
	title := sectionTitle.Render("MENÚ PRINCIPAL")

	var rows []string
	for i, item := range items {
		var row string
		if i == m.cursor {
			row = menuItemSelected.Render("▸ " + item.Label)
		} else if item.Disabled {
			row = menuItemDisabled.Render("  " + item.Label)
		} else {
			row = menuItemNormal.Render("  " + item.Label)
		}
		rows = append(rows, row)
	}

	return lipgloss.JoinVertical(lipgloss.Left, title, strings.Join(rows, "\n"))
}

func (m MenuModel) renderInfoPanel(common Common) string {
	title := sectionTitle.Render("ESTADO DE PENPOT")

	// Badge de estado
	var statusBadge string
	if common.isRunning {
		statusBadge = badgeRunning.Render(" ● CORRIENDO ")
	} else if common.isInstalled {
		statusBadge = badgeStopped.Render(" ● DETENIDO ")
	} else {
		statusBadge = badgeNotInstalled.Render(" ○ NO INSTALADO ")
	}

	lines := []string{
		title,
		statusBadge,
		"",
	}

	if common.isInstalled {
		lines = append(lines,
			lipgloss.NewStyle().Foreground(colorMuted).Render("Directorio:"),
			highlightStyle.Render("  "+common.cfg.InstallDir),
			"",
			lipgloss.NewStyle().Foreground(colorMuted).Render("URL:"),
			highlightStyle.Render(fmt.Sprintf("  http://localhost:%s", common.cfg.Port)),
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

// msgMenuAction informa al modelo principal que se seleccionó una opción
type msgMenuAction struct {
	action menuAction
}
