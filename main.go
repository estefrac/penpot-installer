package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/estefrac/penpot-installer/internal/installer"
	"github.com/estefrac/penpot-installer/internal/tui"
)

// version es inyectada en el build con -ldflags "-X main.version=vX.Y.Z"
var version = "dev"

func main() {
	// Auto-instalarse en el PATH la primera vez que se ejecuta
	installer.EnsureInstalled()

	p := tea.NewProgram(
		tui.New(version),
		tea.WithAltScreen(),       // Usa el buffer alternativo (no ensucia el historial)
		tea.WithMouseCellMotion(), // Soporte opcional de mouse
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error iniciando el TUI: %v\n", err)
		os.Exit(1)
	}
}
