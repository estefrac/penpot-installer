package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ASCII art del logo de Penpot (simplificado)
const penpotASCII = `
 ██████╗ ███████╗███╗   ██╗██████╗  ██████╗ ████████╗
 ██╔══██╗██╔════╝████╗  ██║██╔══██╗██╔═══██╗╚══██╔══╝
 ██████╔╝█████╗  ██╔██╗ ██║██████╔╝██║   ██║   ██║   
 ██╔═══╝ ██╔══╝  ██║╚██╗██║██╔═══╝ ██║   ██║   ██║   
 ██║     ███████╗██║ ╚████║██║     ╚██████╔╝   ██║   
 ╚═╝     ╚══════╝╚═╝  ╚═══╝╚═╝      ╚═════╝    ╚═╝   
`

// RenderBanner retorna el banner completo en verde Penpot
func RenderBanner(width int) string {
	lines := strings.Split(strings.TrimPrefix(penpotASCII, "\n"), "\n")

	logoStyle := lipgloss.NewStyle().Foreground(colorSuccess).Bold(true)

	var rendered []string
	for _, line := range lines {
		rendered = append(rendered, logoStyle.Render(line))
	}

	banner := strings.Join(rendered, "\n")

	subtitle := lipgloss.NewStyle().
		Foreground(colorMuted).
		Italic(true).
		Render("  Instalador interactivo · Docker · Open Source Design Tool")

	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Render(banner + "\n" + subtitle)
}
