package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/estefrac/penpot-installer/internal/system"
)

// DockerModel maneja las vistas de estado y error de Docker
type DockerModel struct {
	os   string
	view view
}

func (m DockerModel) Update(msg tea.Msg) (DockerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.view {
		case viewDockerInstall:
			return m.handleDockerInstallKey(msg)
		case viewDockerWindows:
			return m.handleDockerWindowsKey(msg)
		case viewDockerNotRunning:
			return m.handleDockerNotRunningKey(msg)
		case viewDockerNotRunningWindows:
			return m.handleDockerNotRunningWindowsKey(msg)
		}
	}
	return m, nil
}

func (m DockerModel) handleDockerInstallKey(msg tea.KeyMsg) (DockerModel, tea.Cmd) {
	switch msg.String() {
	case "s", "S", "y", "Y":
		return m, func() tea.Msg { return msgDockerInstallAction{install: true} }
	case "n", "N", tea.KeyEsc.String():
		return m, tea.Quit
	}
	return m, nil
}

func (m DockerModel) handleDockerWindowsKey(msg tea.KeyMsg) (DockerModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		_ = system.OpenBrowser("https://desktop.docker.com/win/main/amd64/Docker%20Desktop%20Installer.exe")
		return m, tea.Quit
	case tea.KeyEsc:
		return m, tea.Quit
	}
	return m, nil
}

func (m DockerModel) handleDockerNotRunningKey(msg tea.KeyMsg) (DockerModel, tea.Cmd) {
	switch msg.String() {
	case "s", "S", "y", "Y":
		return m, func() tea.Msg { return msgDockerStartAction{start: true, os: "linux"} }
	case "n", "N", tea.KeyEsc.String():
		return m, tea.Quit
	}
	return m, nil
}

func (m DockerModel) handleDockerNotRunningWindowsKey(msg tea.KeyMsg) (DockerModel, tea.Cmd) {
	switch msg.String() {
	case "s", "S", "y", "Y":
		return m, func() tea.Msg { return msgDockerStartAction{start: true, os: "windows"} }
	case "n", "N", tea.KeyEsc.String():
		return m, tea.Quit
	}
	return m, nil
}

func (m DockerModel) View(common Common) string {
	switch m.view {
	case viewDockerInstall:
		return m.renderDockerInstall(common)
	case viewDockerWindows:
		return m.renderDockerWindows(common)
	case viewDockerNotRunning:
		return m.renderDockerNotRunning(common)
	case viewDockerNotRunningWindows:
		return m.renderDockerNotRunningWindows(common)
	}
	return ""
}

func (m DockerModel) renderDockerInstall(common Common) string {
	boxW := 60
	description := "Docker es necesario para correr Penpot.\n\nSe instalará usando el script oficial de Docker:\n  curl -fsSL https://get.docker.com | sh\n\nNecesitás conexión a internet y permisos de administrador (sudo)."

	inner := lipgloss.JoinVertical(lipgloss.Left,
		warningStyle.Bold(true).Render("⚠  Docker no está instalado"),
		"",
		lipgloss.NewStyle().Foreground(colorText).Width(boxW-4).Render(description),
		"",
		lipgloss.NewStyle().Foreground(colorMuted).Render(strings.Repeat("─", boxW-6)),
		"",
		highlightStyle.Render("¿Instalar Docker ahora?"),
		"",
		lipgloss.NewStyle().Foreground(colorMuted).Render("s / y  instalar   ·   n / esc  salir"),
	)
	box := lipgloss.NewStyle().
		Width(boxW).Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorWarning).
		Render(inner)
	return lipgloss.Place(innerWidth(common.width), innerHeight(common.height), lipgloss.Center, lipgloss.Center, box)
}

func (m DockerModel) renderDockerWindows(common Common) string {
	boxW := 66
	description := "Docker Desktop es necesario para correr Penpot en Windows.\n\n" +
		"Al presionar Enter voy a abrir el instalador oficial en tu navegador.\n\n" +
		"Pasos a seguir:\n" +
		"  1. Descargá e instalá Docker Desktop\n" +
		"  2. Reiniciá la computadora si te lo pide\n" +
		"  3. Abrí Docker Desktop y esperá que arranque\n" +
		"  4. Volvé a ejecutar este instalador"

	inner := lipgloss.JoinVertical(lipgloss.Left,
		warningStyle.Bold(true).Render("⚠  Docker Desktop no está instalado"),
		"",
		lipgloss.NewStyle().Foreground(colorText).Width(boxW-4).Render(description),
		"",
		lipgloss.NewStyle().Foreground(colorMuted).Render(strings.Repeat("─", boxW-6)),
		"",
		lipgloss.NewStyle().Foreground(colorMuted).Render("enter  abrir instalador en navegador   ·   esc  salir"),
	)
	box := lipgloss.NewStyle().
		Width(boxW).Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorWarning).
		Render(inner)
	return lipgloss.Place(innerWidth(common.width), innerHeight(common.height), lipgloss.Center, lipgloss.Center, box)
}

func (m DockerModel) renderDockerNotRunning(common Common) string {
	boxW := 60
	description := "Docker está instalado pero el daemon no está activo.\n\n¿Querés que lo inicie ahora?\n  (equivale a: sudo systemctl start docker)"

	inner := lipgloss.JoinVertical(lipgloss.Left,
		warningStyle.Bold(true).Render("⚠  Docker no está corriendo"),
		"",
		lipgloss.NewStyle().Foreground(colorText).Width(boxW-4).Render(description),
		"",
		lipgloss.NewStyle().Foreground(colorMuted).Render(strings.Repeat("─", boxW-6)),
		"",
		lipgloss.NewStyle().Foreground(colorMuted).Render("s / y  iniciar Docker   ·   n / esc  salir"),
	)
	box := lipgloss.NewStyle().
		Width(boxW).Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorWarning).
		Render(inner)
	return lipgloss.Place(innerWidth(common.width), innerHeight(common.height), lipgloss.Center, lipgloss.Center, box)
}

func (m DockerModel) renderDockerNotRunningWindows(common Common) string {
	boxW := 66
	description := "Docker Desktop está instalado pero no está corriendo.\n\n" +
		"Necesitás que Docker Desktop esté abierto y activo\n" +
		"para que Penpot pueda funcionar.\n\n" +
		"Presioná 's' para que intente abrirlo automáticamente,\n" +
		"o abrilo vos desde el menú de inicio y después presioná 's'.\n\n" +
		"El instalador va a esperar hasta que Docker esté listo."

	inner := lipgloss.JoinVertical(lipgloss.Left,
		warningStyle.Bold(true).Render("⚠  Docker Desktop no está corriendo"),
		"",
		lipgloss.NewStyle().Foreground(colorText).Width(boxW-4).Render(description),
		"",
		lipgloss.NewStyle().Foreground(colorMuted).Render(strings.Repeat("─", boxW-6)),
		"",
		lipgloss.NewStyle().Foreground(colorMuted).Render("s / y  abrir Docker Desktop   ·   n / esc  salir"),
	)
	box := lipgloss.NewStyle().
		Width(boxW).Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorWarning).
		Render(inner)
	return lipgloss.Place(innerWidth(common.width), innerHeight(common.height), lipgloss.Center, lipgloss.Center, box)
}

// Mensajes específicos de Docker
type msgDockerInstallAction struct{ install bool }
type msgDockerStartAction struct {
	start bool
	os    string
}
