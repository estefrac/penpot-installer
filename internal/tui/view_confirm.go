package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfirmModel maneja los diálogos de confirmación
type ConfirmModel struct {
	message    string
	yes        bool
	isDeletion bool // Si es la pantalla de confirmación de borrado de datos
}

func NewConfirmModel(msg string) ConfirmModel {
	return ConfirmModel{
		message: msg,
		yes:     true,
	}
}

func NewUninstallDataModel() ConfirmModel {
	return ConfirmModel{
		isDeletion: true,
	}
}

func (m ConfirmModel) Update(msg tea.Msg) (ConfirmModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.isDeletion {
			return m.handleUninstallDataKey(msg)
		}
		return m.handleConfirmKey(msg)
	}
	return m, nil
}

func (m ConfirmModel) handleConfirmKey(msg tea.KeyMsg) (ConfirmModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "s", "S":
		return m, func() tea.Msg { return msgConfirmResult{confirmed: true} }
	case "n", "N", tea.KeyEsc.String():
		return m, func() tea.Msg { return msgConfirmResult{confirmed: false} }
	case tea.KeyEnter.String():
		return m, func() tea.Msg { return msgConfirmResult{confirmed: m.yes} }
	case tea.KeyLeft.String(), tea.KeyRight.String(), tea.KeyTab.String():
		m.yes = !m.yes
	}
	return m, nil
}

func (m ConfirmModel) handleUninstallDataKey(msg tea.KeyMsg) (ConfirmModel, tea.Cmd) {
	switch msg.String() {
	case "s", "S", "y", "Y":
		return m, func() tea.Msg { return msgUninstallDataResult{deleteData: true} }
	case "n", "N":
		return m, func() tea.Msg { return msgUninstallDataResult{deleteData: false} }
	case tea.KeyEsc.String():
		return m, func() tea.Msg { return msgChangeView{newView: viewMenu} }
	}
	return m, nil
}

func (m ConfirmModel) View(common Common) string {
	if m.isDeletion {
		return m.renderUninstallData(common)
	}
	return m.renderConfirm(common)
}

func (m ConfirmModel) renderConfirm(common Common) string {
	w, h := innerWidth(common.width), innerHeight(common.height)
	boxW := 56
	if w < 62 {
		boxW = w - 6
	}

	msg := lipgloss.NewStyle().
		Foreground(colorText).
		Width(boxW - 4).
		Render(m.message)

	// Botón base
	btnBase := lipgloss.NewStyle().
		Width(10).
		Align(lipgloss.Center).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder())

	var yesLabel, noLabel string
	if m.yes {
		yesLabel = btnBase.
			BorderForeground(colorSuccess).
			Background(colorSuccess).
			Foreground(colorBg).
			Bold(true).
			Render("▸  Sí")
		noLabel = btnBase.
			BorderForeground(colorMuted).
			Foreground(colorMuted).
			Render("   No")
	} else {
		yesLabel = btnBase.
			BorderForeground(colorMuted).
			Foreground(colorMuted).
			Render("   Sí")
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

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, box)
}

func (m ConfirmModel) renderUninstallData(common Common) string {
	w, h := innerWidth(common.width), innerHeight(common.height)
	boxW := 62
	if w < 68 {
		boxW = w - 6
	}

	description := "Los contenedores e imágenes Docker serán eliminados en cualquier caso.\n\nTus proyectos y archivos están guardados en volúmenes de Docker."

	inner := lipgloss.JoinVertical(lipgloss.Left,
		errorStyle.Bold(true).Render("🗑️  ¿Qué hacemos con tus datos?"),
		"",
		lipgloss.NewStyle().Foreground(colorText).Width(boxW-4).Render(description),
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

// Mensajes de resultado de confirmación
type msgConfirmResult struct{ confirmed bool }
type msgUninstallDataResult struct{ deleteData bool }
