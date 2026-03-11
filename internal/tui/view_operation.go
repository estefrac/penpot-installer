package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// OperationModel maneja la pantalla de progreso con spinner y logs
type OperationModel struct {
	spinner         spinner.Model
	message         string
	logs            []string
	servicesPulling map[string]bool
	servicesPulled  []string
	servicesStarted []string
}

func NewOperationModel(sp spinner.Model, msg string) OperationModel {
	return OperationModel{
		spinner:         sp,
		message:         msg,
		servicesPulling: make(map[string]bool),
	}
}

func (m OperationModel) Update(msg tea.Msg) (OperationModel, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case msgLogLine:
		if !msg.closed {
			m.processDockerLine(msg.line)
		}
	}

	return m, nil
}

func (m OperationModel) View(common Common) string {
	w, h := innerWidth(common.width), innerHeight(common.height)
	boxW := 72
	if w < 78 {
		boxW = w - 6
	}
	logW := boxW - 6

	// Header con spinner
	header := lipgloss.JoinHorizontal(lipgloss.Left,
		m.spinner.View(),
		" ",
		lipgloss.NewStyle().Foreground(colorText).Bold(true).Render(m.message),
	)

	// Barra de progreso de servicios si hay info
	var progressSection string
	total := len(m.servicesPulled) + len(m.servicesPulling) + len(m.servicesStarted)
	if total > 0 {
		pulled := len(m.servicesPulled)
		started := len(m.servicesStarted)

		// Barra visual: bloques llenos vs vacíos
		barWidth := logW - 10
		if barWidth < 10 {
			barWidth = 10
		}
		filled := 0
		if total > 0 {
			filled = (pulled + started) * barWidth / (total + 1)
		}
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
		barStyled := lipgloss.NewStyle().Foreground(colorSecondary).Render(bar)

		counter := lipgloss.NewStyle().Foreground(colorMuted).
			Render(fmt.Sprintf("%d/%d servicios", pulled+started, total))

		progressSection = lipgloss.JoinVertical(lipgloss.Left,
			barStyled,
			counter,
			"",
		)
	}

	// Últimas N líneas de log
	maxLogs := h/2 - 8
	if maxLogs < 3 {
		maxLogs = 3
	}

	var logLines []string
	if len(m.logs) == 0 {
		logLines = append(logLines,
			lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render("Conectando con Docker..."),
		)
	} else {
		recent := m.logs
		if len(recent) > maxLogs {
			recent = recent[len(recent)-maxLogs:]
		}
		for _, l := range recent {
			if len(l) > logW {
				l = l[:logW-3] + "..."
			}
			ll := strings.ToLower(l)
			var styled string
			switch {
			case strings.HasPrefix(l, "✓") || strings.HasPrefix(l, "▶"):
				styled = successStyle.Render(l)
			case strings.HasPrefix(l, "⬇"):
				styled = lipgloss.NewStyle().Foreground(colorSecondary).Render(l)
			case strings.Contains(ll, "error") || strings.Contains(ll, "failed"):
				styled = errorStyle.Render(l)
			default:
				styled = lipgloss.NewStyle().Foreground(colorMuted).Render(l)
			}
			logLines = append(logLines, styled)
		}
	}

	logArea := lipgloss.NewStyle().
		Width(logW).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorMuted).
		Padding(0, 1).
		Render(strings.Join(logLines, "\n"))

	note := lipgloss.NewStyle().
		Foreground(colorMuted).Italic(true).
		Render("Esto puede tardar varios minutos en la primera instalación...")

	inner := lipgloss.JoinVertical(lipgloss.Left,
		sectionTitle.Render("EN PROGRESO"),
		"",
		header,
		"",
		progressSection,
		logArea,
		"",
		note,
	)

	box := lipgloss.NewStyle().
		Width(boxW).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorSecondary).
		Render(inner)

	return lipgloss.Place(
		w, h,
		lipgloss.Center, lipgloss.Center,
		box,
	)
}

func (m *OperationModel) processDockerLine(line string) {
	l := strings.TrimSpace(line)
	ll := strings.ToLower(l)

	if m.servicesPulling == nil {
		m.servicesPulling = make(map[string]bool)
	}

	noisePatterns := []string{
		"pulling fs layer", "verifying checksum", "waiting",
		"already exists", "digest:", "sha256:", "status: image",
		"using default tag",
	}
	for _, noise := range noisePatterns {
		if strings.Contains(ll, noise) {
			return
		}
	}

	if strings.HasSuffix(ll, " pulling") {
		svc := strings.TrimSuffix(l, " Pulling")
		svc = strings.TrimSuffix(svc, " pulling")
		m.servicesPulling[svc] = true
		m.logs = append(m.logs, fmt.Sprintf("⬇  Descargando %s...", svc))
		return
	}

	if strings.HasSuffix(ll, " pulled") {
		svc := strings.TrimSuffix(l, " Pulled")
		svc = strings.TrimSuffix(svc, " pulled")
		delete(m.servicesPulling, svc)
		m.servicesPulled = append(m.servicesPulled, svc)
		m.logs = append(m.logs, fmt.Sprintf("✓  %s descargado", svc))
		return
	}

	if strings.HasSuffix(ll, " started") {
		svc := strings.TrimSuffix(l, " Started")
		svc = strings.TrimSuffix(svc, " started")
		m.servicesStarted = append(m.servicesStarted, svc)
		m.logs = append(m.logs, fmt.Sprintf("▶  %s iniciado", svc))
		return
	}

	if strings.Contains(ll, "download complete") || strings.Contains(ll, "pull complete") {
		return
	}

	if l != "" {
		m.logs = append(m.logs, l)
	}
}
