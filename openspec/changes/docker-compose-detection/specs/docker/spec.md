# Spec: Docker Compose V2 Detection

**Change:** `docker-compose-detection`
**Project:** penpot-installer
**Status:** draft
**Date:** 2026-03-15

---

## Contexto

El instalador TUI de Penpot (Go + BubbleTea) verifica la disponibilidad de Docker en `checkDockerCmd()` con tres condiciones secuenciales. La tercera condición — `docker.ComposeInstalled()` — actualmente emite `msgDockerError` cuando el plugin `docker compose` (V2) no está disponible, mostrando un error genérico sin contexto ni solución accionable.

Esta spec cubre los requisitos para cerrar ese gap aplicando el mismo patrón UX que ya existe para los casos `IsInstalled()` e `IsRunning()`.

---

## Scope

- **Incluye:** Linux (Debian/Ubuntu con `apt`), Windows (caso informativo)
- **Excluye:** macOS (fuera de scope del instalador), docker-compose V1 (EOL julio 2023)

---

## Requisitos

### REQ-01: Detección diferenciada de Compose V2

El flujo `checkDockerCmd()` DEBE retornar un mensaje específico cuando `docker.ComposeInstalled()` retorna `false`, en lugar del genérico `msgDockerError`.

**Código afectado:** `internal/tui/model.go:421-423`

```
Antes: msgDockerError{fmt.Errorf("docker compose no está disponible")}
Después: msgDockerComposeNotInstalled{os: runtime.GOOS}
```

### REQ-02: Mensaje tipado para Compose no disponible

El sistema DEBE definir el tipo `msgDockerComposeNotInstalled` con un campo `os string`, siguiendo la convención de los mensajes existentes `msgDockerNotInstalled` y `msgDockerNotRunning`.

**Archivo:** `internal/tui/messages.go`

### REQ-03: Views dedicadas por OS

El sistema DEBE registrar dos nuevas constantes de view en el bloque `view` de `model.go`:

- `viewDockerComposeInstall` — para Linux cuando Compose V2 no está disponible
- `viewDockerComposeWindows` — para Windows cuando Compose V2 no está disponible (caso edge)

### REQ-04: Routing por OS en el handler del mensaje

El handler de `msgDockerComposeNotInstalled` en `Update()` DEBE:

- Si `msg.os == "linux"`: asignar `viewDockerComposeInstall`
- En cualquier otro caso: asignar `viewDockerComposeWindows`
- Limpiar el flag `dockerChecking` y disparar `tea.ClearScreen`

### REQ-05: Instalación automática del plugin en Linux

La view `viewDockerComposeInstall` DEBE ofrecer al usuario la opción de instalar automáticamente el plugin vía:

```sh
sudo apt-get install -y docker-compose-plugin
```

La instalación DEBE ejecutarse de forma no interactiva y sin bloquear el TUI (como operación asíncrona).

### REQ-06: Re-verificación tras instalación exitosa

Después de una instalación exitosa del plugin (Linux), el flujo DEBE volver a ejecutar `checkDockerCmd()` para re-validar todo el stack desde cero (Docker instalado → corriendo → Compose disponible).

**Nota:** Esto implica reutilizar `msgDockerInstallDone{}` como señal de éxito; el handler existente de ese mensaje ya dispara `checkDockerCmd()`.

### REQ-07: Fallback manual en Linux

La view `viewDockerComposeInstall` DEBE mostrar, junto a la opción de instalación automática, un fallback manual con referencia a la documentación oficial:

```
https://docs.docker.com/compose/install/linux/
```

### REQ-08: View informativa para Windows

La view `viewDockerComposeWindows` DEBE:

- Explicar que Docker Desktop siempre incluye Compose V2 por diseño
- Sugerir reinstalar Docker Desktop como solución
- NO ofrecer instalación automática (no aplica en Windows)
- Ofrecer solo salida limpia con `esc`

### REQ-09: No soportar docker-compose V1

El sistema NO DEBE detectar ni sugerir el uso del binario `docker-compose` (V1). Este binario está EOL desde julio 2023. Si un usuario lo tiene instalado pero no tiene `docker compose` (V2 plugin), DEBE recibir la misma pantalla `viewDockerComposeInstall` y ser dirigido a instalar el plugin V2.

### REQ-10: Inyección de dependencia en `ComposeInstalled()`

La función `docker.ComposeInstalled()` DEBE usar una variable de función inyectable para `system.RunCommand`, de modo que los tests puedan hacer mock sin depender del entorno:

```go
// internal/docker/docker.go
var runCommand = system.RunCommand

func ComposeInstalled() bool {
    _, err := runCommand("docker", "compose", "version")
    return err == nil
}
```

---

## Escenarios

### Linux — Compose V2 no disponible

#### SCEN-01: Detección emite mensaje correcto

```
Dado que Docker está instalado y corriendo
Y `docker compose version` retorna error (exit code != 0)
Cuando se ejecuta `checkDockerCmd()`
Entonces el modelo recibe `msgDockerComposeNotInstalled{os: "linux"}`
Y NO recibe `msgDockerError`
```

#### SCEN-02: Vista correcta se muestra en Linux

```
Dado que el modelo recibió `msgDockerComposeNotInstalled{os: "linux"}`
Cuando el modelo procesa el mensaje
Entonces `m.currentView` es `viewDockerComposeInstall`
Y la pantalla muestra el título "Docker Compose no está disponible"
Y muestra el comando: `sudo apt install docker-compose-plugin`
Y muestra el link de fallback: `https://docs.docker.com/compose/install/linux/`
Y muestra las opciones de teclado: `s/y  instalar   ·   n/esc  salir`
```

#### SCEN-03: Tecla `s` inicia instalación del plugin

```
Dado que la view activa es `viewDockerComposeInstall`
Cuando el usuario presiona `s` o `y` (mayúsculas o minúsculas)
Entonces el modelo emite `msgDockerInstallAction{install: true}`
Y se dispara `installDockerComposeCmd()`
Y NO se cierra la aplicación
```

#### SCEN-04: Instalación exitosa vuelve a verificar todo el stack

```
Dado que `installDockerComposeCmd()` completó sin error
Cuando el modelo recibe `msgDockerInstallDone{}`
Entonces el modelo dispara `checkDockerCmd()`
Y si Compose ahora está disponible, el flujo continúa normalmente hacia `msgDockerReady{}`
```

#### SCEN-05: Tecla `n` o `esc` cierra la aplicación limpiamente

```
Dado que la view activa es `viewDockerComposeInstall`
Cuando el usuario presiona `n`, `N`, o `esc`
Entonces se emite `tea.Quit`
Y la aplicación termina sin error
```

#### SCEN-06: Instalación falla — se muestra error

```
Dado que `installDockerComposeCmd()` falló (apt-get retornó error)
Cuando el modelo recibe `msgDockerInstallError{err}`
Entonces la pantalla muestra el error ocurrido
Y el usuario puede salir o reintentar según el comportamiento existente de `msgDockerInstallError`
```

---

### Windows — Compose V2 no disponible (caso edge)

#### SCEN-07: Vista correcta se muestra en Windows

```
Dado que el modelo recibió `msgDockerComposeNotInstalled{os: "windows"}`
Cuando el modelo procesa el mensaje
Entonces `m.currentView` es `viewDockerComposeWindows`
Y la pantalla muestra el título "Docker Compose no está disponible"
Y explica que Docker Desktop siempre incluye Compose V2
Y sugiere reinstalar Docker Desktop
Y NO muestra opción de instalación automática
Y muestra solo: `esc  salir`
```

#### SCEN-08: Tecla `esc` cierra la aplicación en Windows

```
Dado que la view activa es `viewDockerComposeWindows`
Cuando el usuario presiona `esc`
Entonces se emite `tea.Quit`
Y la aplicación termina sin error
```

---

### Regresión — Flujo existente no se rompe

#### SCEN-09: Docker no instalado sigue funcionando igual

```
Dado que `docker.IsInstalled()` retorna false
Cuando se ejecuta `checkDockerCmd()`
Entonces el modelo recibe `msgDockerNotInstalled{os}` (sin cambios)
Y NO recibe `msgDockerComposeNotInstalled`
```

#### SCEN-10: Docker no corriendo sigue funcionando igual

```
Dado que `docker.IsInstalled()` retorna true
Y `docker.IsRunning()` retorna false
Cuando se ejecuta `checkDockerCmd()`
Entonces el modelo recibe `msgDockerNotRunning{os}` (sin cambios)
Y NOT recibe `msgDockerComposeNotInstalled`
```

#### SCEN-11: Docker listo con Compose disponible sigue funcionando igual

```
Dado que Docker está instalado, corriendo y `docker compose version` retorna éxito
Cuando se ejecuta `checkDockerCmd()`
Entonces el modelo recibe `msgDockerReady{}` (sin cambios)
```

---

### Tests unitarios

#### SCEN-12: `ComposeInstalled()` retorna true cuando el plugin está disponible

```
Dado que `runCommand("docker", "compose", "version")` es mockeado para retornar éxito
Cuando se llama `docker.ComposeInstalled()`
Entonces retorna `true`
```

#### SCEN-13: `ComposeInstalled()` retorna false cuando el plugin no está disponible

```
Dado que `runCommand("docker", "compose", "version")` es mockeado para retornar error
Cuando se llama `docker.ComposeInstalled()`
Entonces retorna `false`
```

#### SCEN-14: Handler de teclado — `s` emite acción de instalación

```
Dado un `DockerModel` con `view: viewDockerComposeInstall`
Cuando se procesa `tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}`
Entonces el Cmd retornado emite `msgDockerInstallAction{install: true}`
```

#### SCEN-15: Handler de teclado — `n` emite Quit

```
Dado un `DockerModel` con `view: viewDockerComposeInstall`
Cuando se procesa `tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}`
Entonces el Cmd retornado emite `tea.Quit`
```

#### SCEN-16: Handler de teclado Windows — `esc` emite Quit

```
Dado un `DockerModel` con `view: viewDockerComposeWindows`
Cuando se procesa `tea.KeyMsg{Type: tea.KeyEsc}`
Entonces el Cmd retornado emite `tea.Quit`
```

---

## Archivos afectados

| Archivo | Cambio |
|---------|--------|
| `internal/tui/messages.go` | +1 tipo: `msgDockerComposeNotInstalled{os string}` |
| `internal/tui/model.go` | +2 constantes de view, reemplazo en `checkDockerCmd()`, +1 handler en `Update()`, +1 función `installDockerComposeCmd()`, actualización de switches en `delegateUpdate()` y `View()` |
| `internal/tui/view_docker.go` | +2 key handlers, +2 renders, extensión de switches en `Update()` y `View()` |
| `internal/docker/docker.go` | Refactor: `var runCommand = system.RunCommand` para permitir mocking |
| `internal/docker/docker_test.go` | NUEVO — 3 tests de `ComposeInstalled()` con mocks |
| `internal/tui/view_docker_test.go` | NUEVO — 4 tests de key handlers |

---

## Restricciones

| Restricción | Justificación |
|-------------|---------------|
| Solo `apt-get` para instalación automática | Scope mínimo; soporte para dnf/pacman es trabajo futuro |
| No soportar docker-compose V1 | EOL julio 2023; soportarlo agrega deuda sin valor |
| macOS excluido | Fuera de scope del instalador |
| No instalar automáticamente en Windows | Docker Desktop incluye Compose V2; caso imposible en la práctica |
| Reutilizar `msgDockerInstallDone` y `msgDockerInstallError` | Evita duplicación; la semántica es la misma |

---

## Gaps identificados

- **Distros non-Debian:** El comando `apt-get install docker-compose-plugin` falla silenciosamente en distros sin apt (Fedora, Arch). El fallback manual mitiga esto, pero la pantalla no advierte explícitamente que el botón de instalación solo funciona en Debian/Ubuntu. Trabajo futuro: detectar el package manager disponible.
- **Falta de feedback durante instalación:** Una vez que el usuario presiona `s`, no hay spinner ni output de progreso específico para la instalación del plugin. Depende del comportamiento general de `msgDockerInstallAction` y `msgDockerInstallDone` — verificar que esos estados ya muestran feedback adecuado.
- **`msgDockerInstallAction` compartido:** Al reutilizar `msgDockerInstallAction{install: true}` tanto para Docker como para el plugin, el handler en `Update()` necesita distinguir el contexto (¿está instalando Docker o el plugin?). Confirmar que el handler existente invoca el cmd correcto según la view activa, o agregar diferenciación.
