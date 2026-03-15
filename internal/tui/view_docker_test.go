package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHandleDockerComposeInstallKey_Yes(t *testing.T) {
	m := DockerModel{os: "linux", view: viewDockerComposeInstall}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	if cmd == nil {
		t.Fatal("esperaba un cmd no nil")
	}

	msg := cmd()
	action, ok := msg.(msgDockerComposeInstallAction)
	if !ok {
		t.Fatalf("esperaba msgDockerComposeInstallAction, got %T", msg)
	}
	if !action.install {
		t.Error("esperaba install == true")
	}
}

func TestHandleDockerComposeInstallKey_No(t *testing.T) {
	m := DockerModel{os: "linux", view: viewDockerComposeInstall}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	if cmd == nil {
		t.Fatal("esperaba un cmd no nil")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("esperaba tea.QuitMsg, got %T", msg)
	}
}

func TestHandleDockerComposeInstallKey_Esc(t *testing.T) {
	m := DockerModel{os: "linux", view: viewDockerComposeInstall}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if cmd == nil {
		t.Fatal("esperaba un cmd no nil")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("esperaba tea.QuitMsg, got %T", msg)
	}
}

func TestHandleDockerComposeWindowsKey_Esc(t *testing.T) {
	m := DockerModel{os: "windows", view: viewDockerComposeWindows}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if cmd == nil {
		t.Fatal("esperaba un cmd no nil")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("esperaba tea.QuitMsg, got %T", msg)
	}
}
