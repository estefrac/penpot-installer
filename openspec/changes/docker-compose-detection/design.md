# Design: docker-compose-detection

**Change:** `docker-compose-detection`
**Project:** penpot-installer
**Status:** design
**Date:** 2026-03-15

---

## Technical Approach

Cerrar el gap en `checkDockerCmd()` reemplazando el `msgDockerError` genérico por un mensaje tipado `msgDockerComposeNotInstalled{os}`, siguiendo exactamente el mismo patrón que ya existe para `msgDockerNotInstalled` y `msgDockerNotRunning`. Se agregan dos views nuevas (`viewDockerComposeInstall` y `viewDockerComposeWindows`), sus handlers de teclado, sus renders, y en Linux una función `docker.InstallCompose()` con `installDockerComposeCmd()`.

---

## Architecture Decisions

### Decision: Reusar `msgDockerInstallDone` y `msgDockerInstallError`

**Choice:** Los comandos `installDockerComposeCmd()` retorna los mismos mensajes que `installDockerCmd()`.

**Alternatives considered:** Crear `msgDockerComposeInstallDone` y `msgDockerComposeInstallError` específicos.

**Rationale:** La semántica es idéntica — "una operación de instalación de Docker terminó". El handler de `msgDockerInstallDone` en `model.go` vuelve a `viewSplash` y re-ejecuta `checkDockerCmd()`, lo que valida el stack completo incluyendo Compose. No hay ambigüedad porque solo puede haber una instalación en curso a la vez.

---

### Decision: No introducir `msgDockerComposeInstallAction` propio

**Choice:** Usar `msgDockerInstallAction{install: bool}` existente como señal de confirmación del usuario.

**Alternatives considered:** Nuevo tipo `msgDockerComposeInstallAction`.

**Rationale:** El handler de `msgDockerInstallAction` en `model.go` llama a `installDockerCmd()`. Para Compose hay que llamar a `installDockerComposeCmd()`. Pero dado que son dos comandos diferentes, **se necesita distinguirlos**. Solución: se crea `msgDockerComposeInstallAction{install bool}` para que el handler en `model.go` sepa qué instalar. Ver sección de contratos.

> **Corrección frente a la propuesta:** La propuesta decía reutilizar `msgDockerInstallAction`. Esto NO es posible sin modificar el handler existente, lo que rompería el principio de responsabilidad única. Se crea el tipo propio.

---

### Decision: `apt-get` hardcodeado para Linux, sin soporte de otros package managers

**Choice:** `sudo apt-get install -y docker-compose-plugin` como única estrategia primaria.

**Alternatives considered:** Detección de distro + soporte `dnf`/`pacman`/`zypper`.

**Rationale:** Scope mínimo. El 80% de los usuarios de Linux con Docker sin Compose están en Debian/Ubuntu. Distros no-Debian fallan limpiamente con error descriptivo que muestra el fallback manual.

---

### Decision: Inyección de dependencia via variable de función en `docker` package

**Choice:** Exponer `var runCommand = system.RunCommand` en el package `docker` para permitir override en tests.

**Alternatives considered:** Interface `CommandRunner`, build tags, proceso real en tests de integración.

**Rationale:** Es el patrón más simple compatible con la estructura actual. No requiere refactor de `system.go` ni introducir interfaces. El override es local al package y no afecta la API pública.

---

## Data Flow

```
Init()
  └─ checkDockerCmd()
       ├─ !docker.IsInstalled()      → msgDockerNotInstalled{os}        (ya existe)
       ├─ !docker.IsRunning()        → msgDockerNotRunning{os}          (ya existe)
       ├─ !docker.ComposeInstalled() → msgDockerComposeNotInstalled{os} (NUEVO)
       └─ ok                        → msgDockerReady{}                  (ya existe)

msgDockerComposeNotInstalled
  ├─ os == "linux"   → viewDockerComposeInstall
  └─ os != "linux"   → viewDockerComposeWindows

viewDockerComposeInstall (Linux)
  ├─ key s/y → msgDockerComposeInstallAction{install: true}
  │               → viewOperation + installDockerComposeCmd()
  │                    ├─ ok  → msgDockerInstallDone → viewSplash → checkDockerCmd()
  │                    └─ err → msgDockerInstallError → viewResult (error)
  └─ key n/esc → tea.Quit

viewDockerComposeWindows (Windows)
  └─ key esc → tea.Quit
```

---

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/tui/messages.go` | Modify | Agregar `msgDockerComposeNotInstalled` |
| `internal/tui/model.go` | Modify | +2 views en enum, +1 handler de msg, +1 cmd, fix en `checkDockerCmd()`, actualizar `delegateUpdate()` y `View()` |
| `internal/tui/view_docker.go` | Modify | +2 key handlers, +2 renders, +1 tipo de mensaje, actualizar `Update()` y `View()` |
| `internal/docker/docker.go` | Modify | +`InstallCompose()`, +variable `runCommand` para inyección en tests |
| `internal/docker/docker_test.go` | Create | Tests para `ComposeInstalled()` e `InstallCompose()` |
| `internal/tui/view_docker_test.go` | Create | Tests para key handlers de las nuevas views |

---

## Interfaces / Contracts

### `internal/tui/messages.go` — nuevo tipo

```go
// msgDockerComposeNotInstalled se envía cuando docker compose (V2) no está disponible
// pero Docker sí está instalado y corriendo.
type msgDockerComposeNotInstalled struct{ os string }
```

---

### `internal/tui/model.go` — enum de views

Agregar antes de `viewUninstallData`:

```go
viewDockerComposeInstall   // docker compose no disponible (Linux)
viewDockerComposeWindows   // docker compose no disponible (Windows) — caso edge
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

---

### `internal/tui/model.go` — fix en `checkDockerCmd()`

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

El import de `fmt` puede quedar si hay otros usos; si no, removerlo.

---

### `internal/tui/model.go` — handler de mensaje en `Update()`

Agregar después del handler de `msgDockerNotRunning`:

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

---

### `internal/tui/model.go` — handler de `msgDockerComposeInstallAction`

```go
case msgDockerComposeInstallAction:
    if msg.install {
        m.operation = NewOperationModel(m.spinner, "Instalando Docker Compose plugin...")
        m.currentView = viewOperation
        return m, tea.Batch(m.spinner.Tick, installDockerComposeCmd())
    }
    return m, tea.Quit
```

---

### `internal/tui/model.go` — nuevo comando `installDockerComposeCmd()`

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

---

### `internal/tui/model.go` — actualizar `delegateUpdate()` y `View()`

```go
// En delegateUpdate():
case viewDockerInstall, viewDockerWindows, viewDockerNotRunning,
     viewDockerNotRunningWindows, viewDockerComposeInstall, viewDockerComposeWindows:
    m.docker, cmd = m.docker.Update(msg)

// En View():
case viewDockerInstall, viewDockerWindows, viewDockerNotRunning,
     viewDockerNotRunningWindows, viewDockerComposeInstall, viewDockerComposeWindows:
    content = m.docker.View(m.common)
```

---

### `internal/tui/view_docker.go` — nuevo tipo de mensaje

```go
// Agregar junto a los otros tipos al final del archivo:
type msgDockerComposeInstallAction struct{ install bool }
```

---

### `internal/tui/view_docker.go` — actualizar `Update()`

```go
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
        case viewDockerComposeInstall:      // NUEVO
            return m.handleDockerComposeInstallKey(msg)
        case viewDockerComposeWindows:      // NUEVO
            return m.handleDockerComposeWindowsKey(msg)
        }
    }
    return m, nil
}
```

---

### `internal/tui/view_docker.go` — key handlers nuevos

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

func (m DockerModel) handleDockerComposeWindowsKey(msg tea.KeyMsg) (DockerModel, tea.Cmd) {
    switch msg.Type {
    case tea.KeyEsc:
        return m, tea.Quit
    }
    return m, nil
}
```

---

### `internal/tui/view_docker.go` — actualizar `View()`

```go
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
    case viewDockerComposeInstall:          // NUEVO
        return m.renderDockerComposeInstall(common)
    case viewDockerComposeWindows:          // NUEVO
        return m.renderDockerComposeWindows(common)
    }
    return ""
}
```

---

### `internal/tui/view_docker.go` — renders nuevos

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

---

### `internal/docker/docker.go` — inyección de dependencia para tests

```go
// runCommand es la función de ejecución de comandos.
// Se puede overridear en tests para evitar dependencia del entorno.
var runCommand = system.RunCommand

// ComposeInstalled verifica si docker compose (V2) está disponible.
// No detecta docker-compose V1 (standalone) — está EOL desde julio 2023.
func ComposeInstalled() bool {
    _, err := runCommand("docker", "compose", "version")
    return err == nil
}
```

---

### `internal/docker/docker.go` — nueva función `InstallCompose()`

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

**Nota:** `RunCommandInteractive` descarta stdout/stderr (silencioso en TUI). Si en el futuro se necesita streaming de logs, reemplazar con `RunCommandStreaming` y cambiar a `viewOperation` con el modelo de streaming.

---

## Testing Strategy

### Contexto: cero tests en el proyecto

El proyecto no tiene ningún test actualmente. Estos son los primeros. La estrategia es minimalista pero suficiente para dar cobertura a las rutas críticas del cambio.

---

### `internal/docker/docker_test.go` — tests para `ComposeInstalled()` e `InstallCompose()`

**Mecanismo de mocking:** override de `runCommand` (variable de package).

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

**Para `InstallCompose()`:** los tests requieren mockear `system.RunCommandInteractive`, que actualmente no es inyectable. Opciones:

1. Exponer `var runCommandInteractive = system.RunCommandInteractive` en `docker.go` (mismo patrón).
2. Test de integración anotado con `//go:build integration` que requiere sistema real.

Se recomienda la opción 1 para mantener tests unitarios deterministas. Si se prefiere diferir, documentar que `InstallCompose()` queda sin test unitario en este cambio.

---

### `internal/tui/view_docker_test.go` — tests para key handlers

**Mecanismo:** crear `DockerModel` directamente y llamar `Update()` con `tea.KeyMsg` sintético.

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

---

## Migration / Rollout

No hay migración de datos ni cambios de API pública. La app siempre compiló sin tests; agregar tests no afecta el binario.

**Orden de implementación recomendado:**

1. `messages.go` — agregar `msgDockerComposeNotInstalled` (sin dependencias)
2. `model.go` — agregar views al enum (sin dependencias)
3. `docker.go` — agregar `runCommand` var + `InstallCompose()` + fix `ComposeInstalled()` si hace falta
4. `model.go` — fix `checkDockerCmd()`, agregar handler y cmd, actualizar switches
5. `view_docker.go` — agregar tipo, handlers, renders, actualizar `Update()` y `View()`
6. Tests — `docker_test.go` y `view_docker_test.go`

Cada paso compila independientemente (el Go compiler atrapa referencias rotas).

---

## Risks

| Riesgo | Impacto | Mitigación |
|--------|---------|------------|
| `msgDockerInstallDone` mal ruteado tras compose install | Alto | El handler vuelve a `checkDockerCmd()` que valida todo el stack; si compose sigue sin estar, vuelve a `viewDockerComposeInstall` |
| `apt-get` no disponible en distro no-Debian | Bajo | `InstallCompose()` falla con error descriptivo; el render muestra el link de fallback manual |
| `runCommand` override en docker package no thread-safe | Bajo | Solo se usa en tests single-threaded; no afecta producción |
| Regresión en views Docker existentes al modificar el `case` compuesto | Medio | Tests de los handlers existentes + review manual del switch |
