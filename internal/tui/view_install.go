package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

// InstallModel maneja el formulario de configuración de instalación
type InstallModel struct {
	inputs []textinput.Model
	focus  int
}

func NewInstallModel(installDir, port string) InstallModel {
	dirInput := textinput.New()
	dirInput.Placeholder = "/home/user/penpot"
	dirInput.CharLimit = 256
	dirInput.SetValue(installDir)
	dirInput.Focus()

	portInput := textinput.New()
	portInput.Placeholder = "9001"
	portInput.CharLimit = 5
	portInput.SetValue(port)

	return InstallModel{
		inputs: []textinput.Model{dirInput, portInput},
		focus:  0,
	}
}

func (m InstallModel) Update(msg tea.Msg) (InstallModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyTab, tea.KeyDown:
			m.inputs[m.focus].Blur()
			m.focus = (m.focus + 1) % len(m.inputs)
			m.inputs[m.focus].Focus()
			return m, nil

		case tea.KeyShiftTab, tea.KeyUp:
			m.inputs[m.focus].Blur()
			m.focus = (m.focus - 1 + len(m.inputs)) % len(m.inputs)
			m.inputs[m.focus].Focus()
			return m, nil

		case tea.KeyEnter:
			if m.focus < len(m.inputs)-1 {
				m.inputs[m.focus].Blur()
				m.focus++
				m.inputs[m.focus].Focus()
				return m, nil
			}
			return m, func() tea.Msg {
				return msgInstallConfirm{
					installDir: m.inputs[0].Value(),
					port:       m.inputs[1].Value(),
				}
			}

		case tea.KeyEsc:
			return m, func() tea.Msg {
				return msgChangeView{newView: viewMenu}
			}
		}
	}

	var cmd tea.Cmd
	m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m InstallModel) View(common Common) string {
	w := innerWidth(common.width)
	banner := RenderBanner(w)
	title := sectionTitle.Render("INSTALACIÓN DE PENPOT")

	labels := []string{"Directorio de instalación:", "Puerto de acceso:"}
	var fields []string

	for i, input := range m.inputs {
		label := lipgloss.NewStyle().Foreground(colorText).Render(labels[i])
		var inputStyled string
		if i == m.focus {
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
		lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(formPanel),
		"",
		lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(help),
	)
}

// msgInstallConfirm solicita confirmar los datos ingresados
type msgInstallConfirm struct {
	installDir string
	port       string
}
