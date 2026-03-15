# Proposal: docker-compose-detection

**Change:** `docker-compose-detection`
**Project:** penpot-installer
**Status:** proposed
**Date:** 2026-03-15

---

## Problema

El instalador tiene un gap crítico de UX en el flujo de detección de Docker. La función `checkDockerCmd()` en `internal/tui/model.go` evalúa tres condiciones en secuencia:

1. `docker.IsInstalled()` → muestra view dedicada con opción de instalar
2. `docker.IsRunning()` → muestra view dedicada con opción de iniciar
3. `docker.ComposeInstalled()` → **actualmente cae en `msgDockerError` con error genérico**

El tercer caso es el gap: cuando Docker está instalado y corriendo pero el plugin `docker compose` (V2) no está disponible, el sistema muestra una pantalla de error sin contexto ni solución accionable. Esto es especialmente común en sistemas Linux donde Docker fue instalado sin el paquete `docker-compose-plugin`, o via paquetes del distro que no incluyen el plugin V2.

### Código actual problemático

```go
// internal/tui/model.go:421-423
if !docker.ComposeInstalled() {
    return msgDockerError{fmt.Errorf("docker compose no está disponible")}
}
```

`msgDockerError` lleva al usuario a `viewResult` con un mensaje de error crudo — sin instrucciones de resolución, sin comando sugerido, sin camino de salida que no sea relanzar el programa manualmente.

---

## Objetivo

Cerrar el gap de UX aplicando el mismo patrón que ya existe para los casos `IsInstalled()` e `IsRunning()`:

- Detectar que `docker compose` (V2) no está disponible como condición específica
- Mostrar una view dedicada y contextual con instrucciones claras de resolución
- En Linux: ofrecer instalación automática del plugin via `apt` con fallback a instrucciones manuales
- En Windows: caso prácticamente imposible en la práctica (Docker Desktop siempre incluye Compose V2), mostrar info con sugerencia de reinstalar Docker Desktop
- NO soportar docker-compose V1 (standalone): está EOL desde julio 2023, es deuda técnica activa

---

## Approach elegido: Approach A — Mínimo (seguir el patrón establecido)

### Justificación

El codebase ya tiene un patrón claro y probado para este tipo de situaciones. Replicarlo minimiza riesgo, mantiene consistencia arquitectural, y no introduce complejidad innecesaria. El Approach B (instalar automáticamente sin confirmación) rompería el principio de consent-first que tiene el TUI; el Approach C (soportar docker-compose V1) agrega deuda técnica sin valor real dado el EOL.

### Patrón existente (referencia)

```
msgDockerNotInstalled{os} → viewDockerInstall (Linux) / viewDockerWindows (Windows)
msgDockerNotRunning{os}   → viewDockerNotRunning (Linux) / viewDockerNotRunningWindows (Windows)
```

### Patrón nuevo (a implementar)

```
msgDockerComposeNotInstalled{os} → viewDockerComposeInstall (Linux) / viewDockerComposeWindows (Windows)
```

---

## Flujo propuesto

```
checkDockerCmd()
├── !docker.IsInstalled()      → msgDockerNotInstalled{os}           (ya existe)
├── !docker.IsRunning()        → msgDockerNotRunning{os}             (ya existe)
├── !docker.ComposeInstalled() → msgDockerComposeNotInstalled{os}    (NUEVO)
└──                            → msgDockerReady{}                    (ya existe)
```

---

## Cambios por archivo

### `internal/tui/messages.go`

Agregar el nuevo mensaje:

```go
// msgDockerComposeNotInstalled se envía cuando docker compose (V2) no está disponible
type msgDockerComposeNotInstalled struct{ os string }
```

### `internal/tui/model.go`

**1. Agregar dos nuevas constantes de view** (en el bloque `const` de tipo `view`):

```go
viewDockerComposeInstall   // docker compose no disponible (Linux)
viewDockerComposeWindows   // docker compose no disponible (Windows) — caso edge
```

**2. Reemplazar el `msgDockerError` en `checkDockerCmd()`:**

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

**3. Agregar handler del nuevo mensaje** en el switch de `Update()`:

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

**4. Agregar las nuevas views a los switches de `delegateUpdate()` y `View()`:**

```go
// En delegateUpdate y View, agregar al caso existente de views Docker:
case viewDockerInstall, viewDockerWindows, viewDockerNotRunning,
     viewDockerNotRunningWindows, viewDockerComposeInstall, viewDockerComposeWindows:
```

**5. Agregar `installDockerComposeCmd()` para Linux:**

```go
func installDockerComposeCmd() tea.Cmd {
    return func() tea.Msg {
        if _, err := system.RunCommand("sudo", "apt-get", "install", "-y", "docker-compose-plugin"); err != nil {
            return msgDockerInstallError{err}
        }
        return msgDockerInstallDone{}
    }
}
```

### `internal/tui/view_docker.go`

**1. Agregar handler de teclado para `viewDockerComposeInstall`** en el switch de `Update()`:

```go
case viewDockerComposeInstall:
    return m.handleDockerComposeInstallKey(msg)
case viewDockerComposeWindows:
    return m.handleDockerComposeWindowsKey(msg)
```

**2. Implementar los dos handlers:**

`handleDockerComposeInstallKey`: `s/y` → `msgDockerInstallAction{install: true}` (reutiliza la acción existente, que ya dispara `installDockerComposeCmd`), `n/esc` → `tea.Quit`.

`handleDockerComposeWindowsKey`: solo `esc` → `tea.Quit` (no hay acción útil en Windows para este caso).

**3. Agregar los dos renders:**

`renderDockerComposeInstall` (Linux):
- Título: `"⚠  Docker Compose no está disponible"`
- Descripción del problema + comando de solución: `sudo apt install docker-compose-plugin`
- Fallback manual: link a docs.docker.com/compose/install/linux/
- Keys: `s / y  instalar plugin   ·   n / esc  salir`

`renderDockerComposeWindows` (Windows):
- Título: `"⚠  Docker Compose no está disponible"`
- Explicar que Docker Desktop siempre incluye Compose V2
- Sugerir reinstalar Docker Desktop
- Keys: `esc  salir`

**4. Agregar los casos al switch de `View()`:**

```go
case viewDockerComposeInstall:
    return m.renderDockerComposeInstall(common)
case viewDockerComposeWindows:
    return m.renderDockerComposeWindows(common)
```

### `internal/docker/docker.go`

Sin cambios necesarios. `ComposeInstalled()` ya existe y funciona correctamente:

```go
func ComposeInstalled() bool {
    _, err := system.RunCommand("docker", "compose", "version")
    return err == nil
}
```

Explícitamente **no** se agrega detección de docker-compose V1 (standalone). Motivo: EOL desde julio 2023, no soportarlo es la decisión correcta técnicamente.

---

## Tests (primera cobertura del proyecto)

El proyecto actualmente no tiene ningún test. Este cambio introduce los primeros tests unitarios como base para el proyecto.

### `internal/docker/docker_test.go`

- `TestComposeInstalled_WhenAvailable`: mock de `system.RunCommand` retornando éxito
- `TestComposeInstalled_WhenNotAvailable`: mock retornando error
- `TestComposeInstalled_WhenDockerNotRunning`: mock retornando error de conexión

### `internal/tui/view_docker_test.go`

- `TestHandleDockerComposeInstallKey_Yes`: verificar que `s` emite `msgDockerInstallAction{install: true}`
- `TestHandleDockerComposeInstallKey_No`: verificar que `n` emite `tea.Quit`
- `TestHandleDockerComposeInstallKey_Esc`: verificar que `esc` emite `tea.Quit`
- `TestHandleDockerComposeWindowsKey_Esc`: verificar que `esc` emite `tea.Quit`

### Nota sobre mocking en Go

Para testear `ComposeInstalled()` sin depender del entorno, se necesita inyección de dependencia en `system.RunCommand`. Si el sistema actual no la soporta, se puede usar una función de variable en el package como punto de inyección:

```go
// internal/docker/docker.go
var runCommand = system.RunCommand

func ComposeInstalled() bool {
    _, err := runCommand("docker", "compose", "version")
    return err == nil
}
```

Esto permite overridearlo en tests sin romper la interfaz pública.

---

## Restricciones y decisiones

| Decisión | Razón |
|----------|-------|
| No soportar docker-compose V1 | EOL julio 2023. Soportarlo es acumular deuda técnica sin beneficio real |
| Solo Linux recibe instalación automática | En macOS/Windows el plugin viene incluido con Docker Desktop por diseño |
| Reutilizar `msgDockerInstallAction` y `msgDockerInstallDone` | Evita duplicación. La semántica es la misma: "instalar un componente de Docker" |
| No agregar `installDockerComposeWindows()` | No existe caso real donde Docker Desktop esté funcionando sin Compose V2 |
| Usar `apt-get` hardcodeado | Scope mínimo. Soporte para otros package managers (dnf, pacman) es trabajo futuro |

---

## Archivos afectados

```
internal/tui/messages.go        — +1 tipo de mensaje
internal/tui/model.go           — +2 views, +1 handler, +1 cmd, cambio en checkDockerCmd
internal/tui/view_docker.go     — +2 renders, +2 handlers de teclado, cambio en Update/View
internal/docker/docker_test.go  — NUEVO (primeros tests del proyecto)
internal/tui/view_docker_test.go — NUEVO
```

---

## Criterios de aceptación

- [ ] `checkDockerCmd()` retorna `msgDockerComposeNotInstalled{os}` cuando `ComposeInstalled()` es false
- [ ] En Linux, se muestra `viewDockerComposeInstall` con opción de instalar el plugin
- [ ] Presionar `s/y` en Linux inicia la instalación via `apt install docker-compose-plugin`
- [ ] Tras instalación exitosa, el flujo vuelve a `viewSplash` y re-ejecuta `checkDockerCmd()`
- [ ] En Windows, se muestra `viewDockerComposeWindows` con info contextual y solo opción de salir
- [ ] Presionar `n/esc` en cualquiera de las dos views sale limpiamente
- [ ] Los tests de `ComposeInstalled()` pasan sin Docker instalado en el entorno de CI
- [ ] Los tests de los handlers de teclado pasan de forma determinista

---

## Riesgos

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|-------------|---------|------------|
| Distros no-Debian sin `apt-get` | Media | Bajo | El comando falla limpiamente con error claro; el usuario ve el link al fallback manual |
| Regresión en views Docker existentes | Baja | Alto | Tests de integración del switch de views |
| `msgDockerInstallDone` re-usado causa re-instalación incorrecta | Baja | Medio | `checkDockerCmd()` valida todo el stack desde cero, incluyendo Compose |
