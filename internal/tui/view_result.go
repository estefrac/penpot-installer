package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ResultModel maneja la pantalla de resultado final (éxito o error)
type ResultModel struct {
	message string
	isError bool
}

func (m ResultModel) Update(msg tea.Msg) (ResultModel, tea.Cmd) {
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

func (m ResultModel) View(common Common) string {
	w, h := innerWidth(common.width), innerHeight(common.height)
	boxW := 64
	if w < 70 {
		boxW = w - 6
	}

	var borderColor lipgloss.Color
	var title string

	if m.isError {
		borderColor = colorError
		title = errorStyle.Bold(true).Render("✗  Error")
	} else {
		borderColor = colorSuccess
		title = successStyle.Bold(true).Render("✓  Listo")
	}

	msg := lipgloss.NewStyle().
		Foreground(colorText).
		Width(boxW - 4).
		Render(m.message)

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

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, box)
}
