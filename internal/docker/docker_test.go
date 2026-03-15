package docker

import (
	"fmt"
	"testing"
)

func TestComposeInstalled_WhenAvailable(t *testing.T) {
	orig := runCommand
	defer func() { runCommand = orig }()

	runCommand = func(name string, args ...string) (string, error) {
		return "Docker Compose version v2.20.0", nil
	}

	if !ComposeInstalled() {
		t.Error("esperaba ComposeInstalled() == true cuando el comando tiene éxito")
	}
}

func TestComposeInstalled_WhenNotAvailable(t *testing.T) {
	orig := runCommand
	defer func() { runCommand = orig }()

	runCommand = func(name string, args ...string) (string, error) {
		return "", fmt.Errorf("unknown command: compose")
	}

	if ComposeInstalled() {
		t.Error("esperaba ComposeInstalled() == false cuando el comando falla")
	}
}
