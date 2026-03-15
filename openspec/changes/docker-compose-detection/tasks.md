# Tasks: docker-compose-detection

**Change:** `docker-compose-detection`
**Project:** penpot-installer
**Status:** tasks
**Date:** 2026-03-15

---

## Overview

6 archivos afectados (4 modificaciones, 2 creaciones). 14 tareas ordenadas por dependencia ascendente: primero los tipos y contratos, luego la lógica, luego UI, finalmente tests.

---

## Phase 1 — Tipos y contratos (sin dependencias externas)

### T-01: Agregar `msgDockerComposeNotInstalled` en `messages.go`

**Archivo:** `internal/tui/messages.go`
**Acción:** MODIFY

Agregar el nuevo tipo de mensaje justo después del bloque de mensajes Docker existentes:

```go
// msgDockerComposeNotInstalled se envía cuando docker compose (V2) no está disponible
// pero Docker sí está instalado y corriendo.
type msgDockerComposeNotInstalled struct{ os string }
```

**Criterio de done:** compila sin errores.

---

### T-02: Agregar `viewDockerComposeInstall` y `viewDockerComposeWindows` al enum en `model.go`

**Archivo:** `internal/tui/model.go`
**Acción:** MODIFY

Insertar las dos constantes en el bloque `const (...)` de `view`, antes de `viewUninstallData`:

```go
viewDockerComposeInstall    // docker compose no disponible (Linux)
viewDockerComposeWindows    // docker compose no disponible (Windows) — caso edge
```

Resultado final del bloque:
```go
const (
    viewSplash                  view = iota
    viewMenu
    viewInstall
    viewStatus
    viewConfirm
    viewOperation
    viewResult
    viewDockerInstall
    viewDockerWindows
    viewDockerNotRunning
    viewDockerNotRunningWindows
    viewDockerComposeInstall    // NUEVO
    viewDockerComposeWindows    // NUEVO
    viewUninstallData
)
```

**Criterio de done:** compila sin errores.

---

### T-03: Agregar `msgDockerComposeInstallAction` en `view_docker.go`

**Archivo:** `internal/tui/view_docker.go`
**Acción:** MODIFY

Agregar el tipo junto a los otros tipos de mensaje al final del archivo (o junto a los tipos existentes del package):

```go
// msgDockerComposeInstallAction se emite cuando el usuario confirma o rechaza
// la instalación del plugin docker-compose.
type msgDockerComposeInstallAction struct{ install bool }
```

**Criterio de done:** compila sin errores.

---

## Phase 2 — Capa de dominio (docker package)

### T-04: Exponer `var runCommand` para testability en `docker.go`

**Archivo:** `internal/docker/docker.go`
**Acción:** MODIFY

Agregar la variable al top del package (después de los imports):

```go
// runCommand es la función de ejecución de comandos.
// Se puede overridear en tests para evitar dependencia del entorno.
var runCommand = system.RunCommand
```

**Criterio de done:** compila sin errores.

---

### T-05: Refactorizar `ComposeInstalled()` para usar `runCommand`

**Archivo:** `internal/docker/docker.go`
**Acción:** MODIFY

Reemplazar la llamada directa a `system.RunCommand`:

```go
// Antes:
func ComposeInstalled() bool {
    _, err := system.RunCommand("docker", "compose", "version")
    return err == nil
}

// Después:
// ComposeInstalled verifica si docker compose (V2) está disponible.
// No detecta docker-compose V1 (standalone) — está EOL desde julio 2023.
func ComposeInstalled() bool {
    _, err := runCommand("docker", "compose", "version")
    return err == nil
}
```

**Criterio de done:** compila sin errores. Comportamiento idéntico en runtime.

---

### T-06: Agregar `InstallCompose()` en `docker.go`

**Archivo:** `internal/docker/docker.go`
**Acción:** MODIFY

Agregar la función después de `ComposeInstalled()`:

```go
// InstallCompose instala el plugin docker-compose-plugin via apt-get.
// Solo soportado en Linux (Debian/Ubuntu). Otros package managers deben
// instalar manualmente: https://docs.docker.com/compose/install/linux/
func InstallCompose() error {
    if runtime.GOOS != "linux" {
        return fmt.Errorf("instalación automática de Docker Compose solo soportada en Linux")
    }

    steps := []struct {
        name string
        cmd  string
        args []string
    }{
        {
            "Actualizando índice de paquetes",
            "apt-get", []string{"update", "-y"},
        },
        {
            "Instalando docker-compose-plugin",
            "apt-get", []string{"install", "-y", "docker-compose-plugin"},
        },
    }

    for _, step := range steps {
        if err := system.RunCommandInteractive(step.cmd, step.args...); err != nil {
            return fmt.Errorf("error en '%s': %w", step.name, err)
        }
    }

    return nil
}
```

**Criterio de done:** compila sin errores.

---

## Phase 3 — Model (orquestación TUI)

### T-07: Fix `checkDockerCmd()` — reemplazar `msgDockerError` por `msgDockerComposeNotInstalled`

**Archivo:** `internal/tui/model.go`
**Acción:** MODIFY

Localizar el bloque en `checkDockerCmd()` y reemplazar:

```go
// Antes:
if !docker.ComposeInstalled() {
    return msgDockerError{fmt.Errorf("docker compose no está disponible")}
}

// Después:
if !docker.ComposeInstalled() {
    return msgDockerComposeNotInstalled{os: runtime.GOOS}
}
```

Verificar si `fmt.Errorf` sigue en uso en `checkDockerCmd()`; si era el único uso, remover el import `fmt` de esta función (aunque probablemente `fmt` se usa en otros lugares del archivo).

**Criterio de done:** compila. El case `msgDockerError` para compose ya no es necesario en `Update()`.

---

### T-08: Agregar handler de `msgDockerComposeNotInstalled` en `Update()`

**Archivo:** `internal/tui/model.go`
**Acción:** MODIFY

Agregar el case después del handler de `msgDockerNotRunning`:

```go
case msgDockerComposeNotInstalled:
    m.dockerChecking = false
    m.dockerOS = msg.os
    if msg.os == "linux" {
        m.docker = DockerModel{os: msg.os, view: viewDockerComposeInstall}
    } else {
        m.docker = DockerModel{os: msg.os, view: viewDockerComposeWindows}
    }
    m.currentView = m.docker.view
    return m, tea.ClearScreen
```

**Criterio de done:** compila. El routing Linux/no-Linux es correcto.

---

### T-09: Agregar handler de `msgDockerComposeInstallAction` en `Update()`

**Archivo:** `internal/tui/model.go`
**Acción:** MODIFY

Agregar el case después del handler de `msgDockerComposeNotInstalled`:

```go
case msgDockerComposeInstallAction:
    if msg.install {
        m.operation = NewOperationModel(m.spinner, "Instalando Docker Compose plugin...")
        m.currentView = viewOperation
        return m, tea.Batch(m.spinner.Tick, installDockerComposeCmd())
    }
    return m, tea.Quit
```

**Criterio de done:** compila. Referencia a `installDockerComposeCmd()` se resuelve en T-10.

---

### T-10: Agregar función `installDockerComposeCmd()` en `model.go`

**Archivo:** `internal/tui/model.go`
**Acción:** MODIFY

Agregar junto a `installDockerCmd()`:

```go
func installDockerComposeCmd() tea.Cmd {
    return func() tea.Msg {
        if err := docker.InstallCompose(); err != nil {
            return msgDockerInstallError{err}
        }
        return msgDockerInstallDone{}
    }
}
```

**Rationale de reusar `msgDockerInstallDone`/`msgDockerInstallError`:** el handler de `msgDockerInstallDone` vuelve a `viewSplash` y re-ejecuta `checkDockerCmd()`, que valida el stack completo. No hay ambigüedad porque solo puede haber una instalación en curso.

**Criterio de done:** compila. El proyecto entero compila sin errores al finalizar esta tarea.

---

### T-11: Extender `delegateUpdate()` y `View()` con los nuevos views

**Archivo:** `internal/tui/model.go`
**Acción:** MODIFY

En `delegateUpdate()`, extender el `case` compuesto existente:

```go
// Antes:
case viewDockerInstall, viewDockerWindows, viewDockerNotRunning, viewDockerNotRunningWindows:
    m.docker, cmd = m.docker.Update(msg)

// Después:
case viewDockerInstall, viewDockerWindows, viewDockerNotRunning,
     viewDockerNotRunningWindows, viewDockerComposeInstall, viewDockerComposeWindows:
    m.docker, cmd = m.docker.Update(msg)
```

En `View()`, mismo patrón:

```go
// Antes:
case viewDockerInstall, viewDockerWindows, viewDockerNotRunning, viewDockerNotRunningWindows:
    content = m.docker.View(m.common)

// Después:
case viewDockerInstall, viewDockerWindows, viewDockerNotRunning,
     viewDockerNotRunningWindows, viewDockerComposeInstall, viewDockerComposeWindows:
    content = m.docker.View(m.common)
```

**Criterio de done:** compila. Ningún view nuevo queda sin delegar.

---

## Phase 4 — View Docker (UI handlers y renders)

### T-12: Agregar key handlers y actualizar `Update()` y `View()` en `view_docker.go`

**Archivo:** `internal/tui/view_docker.go`
**Acción:** MODIFY

**4a — Agregar `handleDockerComposeInstallKey()`:**

```go
func (m DockerModel) handleDockerComposeInstallKey(msg tea.KeyMsg) (DockerModel, tea.Cmd) {
    switch msg.String() {
    case "s", "S", "y", "Y":
        return m, func() tea.Msg { return msgDockerComposeInstallAction{install: true} }
    case "n", "N", tea.KeyEsc.String():
        return m, tea.Quit
    }
    return m, nil
}
```

**4b — Agregar `handleDockerComposeWindowsKey()`:**

```go
func (m DockerModel) handleDockerComposeWindowsKey(msg tea.KeyMsg) (DockerModel, tea.Cmd) {
    switch msg.Type {
    case tea.KeyEsc:
        return m, tea.Quit
    }
    return m, nil
}
```

**4c — Extender `Update()` con los nuevos cases:**

```go
// En el switch interno de tea.KeyMsg:
case viewDockerComposeInstall:
    return m.handleDockerComposeInstallKey(msg)
case viewDockerComposeWindows:
    return m.handleDockerComposeWindowsKey(msg)
```

**4d — Extender `View()` con los nuevos cases:**

```go
case viewDockerComposeInstall:
    return m.renderDockerComposeInstall(common)
case viewDockerComposeWindows:
    return m.renderDockerComposeWindows(common)
```

**Criterio de done:** compila. Los 4 sub-pasos pueden hacerse en un solo commit o separados.

---

### T-13: Agregar renders `renderDockerComposeInstall()` y `renderDockerComposeWindows()`

**Archivo:** `internal/tui/view_docker.go`
**Acción:** MODIFY

Agregar los dos renders después de los renders existentes:

```go
func (m DockerModel) renderDockerComposeInstall(common Common) string {
    boxW := 66
    description := "Docker está instalado y corriendo, pero el plugin\n" +
        "docker compose (V2) no está disponible.\n\n" +
        "Se instalará usando apt:\n" +
        "  sudo apt-get install -y docker-compose-plugin\n\n" +
        "Si tu sistema usa otro package manager, instalá el\n" +
        "plugin manualmente:\n" +
        "  https://docs.docker.com/compose/install/linux/"

    inner := lipgloss.JoinVertical(lipgloss.Left,
        warningStyle.Bold(true).Render("⚠  Docker Compose no está disponible"),
        "",
        lipgloss.NewStyle().Foreground(colorText).Width(boxW-4).Render(description),
        "",
        lipgloss.NewStyle().Foreground(colorMuted).Render(strings.Repeat("─", boxW-6)),
        "",
        highlightStyle.Render("¿Instalar docker-compose-plugin ahora?"),
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

func (m DockerModel) renderDockerComposeWindows(common Common) string {
    boxW := 66
    description := "Docker Desktop está instalado y corriendo, pero el\n" +
        "plugin docker compose (V2) no está disponible.\n\n" +
        "Esto es inusual — Docker Desktop siempre incluye\n" +
        "Compose V2 por defecto.\n\n" +
        "Pasos sugeridos:\n" +
        "  1. Abrí Docker Desktop → Settings → General\n" +
        "  2. Verificá que 'Use Docker Compose V2' esté activado\n" +
        "  3. Si no funciona, reinstalá Docker Desktop\n\n" +
        "Instalador oficial: https://www.docker.com/products/docker-desktop/"

    inner := lipgloss.JoinVertical(lipgloss.Left,
        warningStyle.Bold(true).Render("⚠  Docker Compose no está disponible"),
        "",
        lipgloss.NewStyle().Foreground(colorText).Width(boxW-4).Render(description),
        "",
        lipgloss.NewStyle().Foreground(colorMuted).Render(strings.Repeat("─", boxW-6)),
        "",
        lipgloss.NewStyle().Foreground(colorMuted).Render("esc  salir"),
    )
    box := lipgloss.NewStyle().
        Width(boxW).Padding(1, 2).
        Border(lipgloss.RoundedBorder()).
        BorderForeground(colorWarning).
        Render(inner)
    return lipgloss.Place(innerWidth(common.width), innerHeight(common.height), lipgloss.Center, lipgloss.Center, box)
}
```

**Criterio de done:** compila. Los estilos `warningStyle`, `colorText`, `colorMuted`, `colorWarning`, `highlightStyle` ya existen en el package.

---

## Phase 5 — Tests (primeros del proyecto)

### T-14: Crear `internal/docker/docker_test.go`

**Archivo:** `internal/docker/docker_test.go`
**Acción:** CREATE

```go
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
```

**Nota sobre `InstallCompose()`:** usa `system.RunCommandInteractive` que no es inyectable actualmente. En este cambio se deja sin test unitario. Opciones para el siguiente cambio:
- Exponer `var runCommandInteractive = system.RunCommandInteractive` (mismo patrón que `runCommand`)
- Build tag `//go:build integration` para test con sistema real

**Criterio de done:** `go test ./internal/docker/...` pasa.

---

### T-15: Crear `internal/tui/view_docker_test.go`

**Archivo:** `internal/tui/view_docker_test.go`
**Acción:** CREATE

```go
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
    // tea.Quit retorna tea.QuitMsg cuando se ejecuta
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
```

**Criterio de done:** `go test ./internal/tui/...` pasa.

---

## Checklist de implementación

```
Phase 1 — Tipos y contratos
[ ] T-01  messages.go      agregar msgDockerComposeNotInstalled
[ ] T-02  model.go         agregar viewDockerComposeInstall y viewDockerComposeWindows al enum
[ ] T-03  view_docker.go   agregar msgDockerComposeInstallAction

Phase 2 — Dominio docker
[ ] T-04  docker.go        exponer var runCommand
[ ] T-05  docker.go        refactorizar ComposeInstalled() para usar runCommand
[ ] T-06  docker.go        agregar InstallCompose()

Phase 3 — Orquestación TUI (model)
[ ] T-07  model.go         fix checkDockerCmd()
[ ] T-08  model.go         handler msgDockerComposeNotInstalled en Update()
[ ] T-09  model.go         handler msgDockerComposeInstallAction en Update()
[ ] T-10  model.go         agregar installDockerComposeCmd()
[ ] T-11  model.go         extender delegateUpdate() y View()

Phase 4 — UI (view_docker)
[ ] T-12  view_docker.go   key handlers + Update() + View()
[ ] T-13  view_docker.go   renders renderDockerComposeInstall y renderDockerComposeWindows

Phase 5 — Tests
[ ] T-14  docker_test.go   CREATE (2 tests para ComposeInstalled)
[ ] T-15  view_docker_test.go CREATE (4 tests para key handlers)
```

---

## Notas de implementación

- **Orden crítico:** T-01 y T-02 deben ir antes que T-08 (el handler referencia los nuevos tipos y views). T-10 debe existir antes de que compile T-09 (referencia a `installDockerComposeCmd`).
- **Cada phase compila de forma incremental.** Go atrapa referencias rotas al compilar; cada tarea deja el proyecto en estado compilable.
- **T-14 y T-15 son los primeros tests del proyecto.** Verificar que `go test` funciona antes de agregar más cobertura.
- **`InstallCompose()` sin test unitario** es deuda técnica conocida y documentada. El test de integración se puede agregar en el siguiente cambio cuando se exponga `runCommandInteractive`.
