package tui

import (
	"fmt"

	"github.com/estefrac/penpot-installer/internal/installer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SplashModel maneja la pantalla inicial de carga y verificación
type SplashModel struct {
	ready bool
}

func (m SplashModel) Update(msg tea.Msg) (SplashModel, tea.Cmd) {
	switch msg := msg.(type) {
	case msgDockerReady:
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "Q":
			return m, tea.Quit
		}

		if !m.ready {
			return m, nil
		}

		// Cualquier tecla pasa al menú cuando el splash está listo
		switch msg.Type {
		case tea.KeyEnter, tea.KeySpace:
			return m, func() tea.Msg {
				return msgChangeView{newView: viewMenu}
			}
		}
	}

	return m, nil
}

func (m SplashModel) View(common Common, spinnerView string) string {
	w, h := innerWidth(common.width), innerHeight(common.height)
	banner := RenderBanner(w)

	var msg string
	if m.ready {
		// Versión
		ver := lipgloss.NewStyle().
			Foreground(colorMuted).
			Render(fmt.Sprintf("v%s", common.version))

		// Estado de auto-instalación (si hubo alguna acción)
		installStatus := renderInstallStatus(common.installResult)

		// Prompt para continuar
		prompt := lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true).
			Render("Presioná Enter para continuar")

		parts := []string{ver}
		if installStatus != "" {
			parts = append(parts, "", installStatus)
		}
		parts = append(parts, "", prompt)
		msg = lipgloss.JoinVertical(lipgloss.Center, parts...)
	} else {
		msg = lipgloss.NewStyle().
			Foreground(colorMuted).
			Render(fmt.Sprintf("%s Verificando Docker...", spinnerView))
	}

	return lipgloss.Place(
		w, h,
		lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center, banner, "", msg),
	)
}

// Funciones de ayuda movidas o adaptadas

func renderInstallStatus(res installer.InstallResult) string {
	switch res.Action {
	case "installed":
		return successStyle.Render("Instalado en " + res.Message)
	case "updated":
		return successStyle.Render("Actualizado en " + res.Message)
	case "failed":
		return warningStyle.Render(
			"No se pudo instalar automaticamente. Ejecuta:\n  " + res.Message,
		)
	default:
		return ""
	}
}
