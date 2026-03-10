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

// RenderBanner retorna el banner completo con gradiente
func RenderBanner(width int) string {
	lines := strings.Split(strings.TrimPrefix(penpotASCII, "\n"), "\n")

	// Gradiente: de purple a teal
	colors := []lipgloss.Color{
		colorPrimary,
		lipgloss.Color("#8B4CC8"),
		lipgloss.Color("#A561D7"),
		lipgloss.Color("#4AA8E8"),
		lipgloss.Color("#31EFB8"),
		colorSecondary,
	}

	var rendered []string
	for i, line := range lines {
		idx := i % len(colors)
		style := lipgloss.NewStyle().Foreground(colors[idx]).Bold(true)
		rendered = append(rendered, style.Render(line))
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
