package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// StatusModel maneja la vista de estado de los contenedores
type StatusModel struct {
	containers []containerInfo
}

func (m StatusModel) Update(msg tea.Msg) (StatusModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter, tea.KeyEsc:
			return m, func() tea.Msg {
				return msgChangeView{newView: viewMenu}
			}
		}
	}
	return m, nil
}

func (m StatusModel) View(common Common) string {
	w := innerWidth(common.width)
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

		if common.isRunning {
			urlMsg := fmt.Sprintf("Penpot disponible en: http://localhost:%s", common.cfg.Port)
			rows = append(rows, "", highlightStyle.Render(urlMsg))
		}
	}

	content := contentPanelStyle.Width(w - 8).Render(strings.Join(rows, "\n"))
	help := helpStyle.Render("enter / esc  volver al menú")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		banner,
		"",
		lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(content),
		"",
		lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(help),
	)
}
