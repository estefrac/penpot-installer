# Proposal: Refactor TUI Model

## Intent
Reducir la complejidad técnica y la deuda acumulada en `internal/tui/model.go`. El archivo actual de 1545 líneas es difícil de mantener, propenso a errores al agregar nuevas funcionalidades y viola el principio de responsabilidad única al manejar el estado de 11 vistas diferentes en un solo objeto "God Object".

## Scope

### In Scope
- Creación de una estructura de sub-modelos para cada vista principal (Splash, Menu, Install, etc.).
- Refactorización de `internal/tui/model.go` para actuar únicamente como un router/orquestador.
- Migración de la lógica de renderizado (`View`) y manejo de eventos (`Update`) a sus respectivos archivos de sub-modelos.
- Preservación total de la funcionalidad y estética actual (refactor puro, zero-regressions).

### Out of Scope
- Agregado de nuevas funcionalidades a la TUI.
- Modificaciones en la lógica de negocio de los paquetes `internal/penpot`, `internal/docker` o `internal/system`.
- Cambios en el diseño visual de la aplicación.

## Approach
Se implementará el patrón de **Modelos Compuestos** (Composed Models) siguiendo la arquitectura Elm:
1. **Delegación de Estado:** Cada sub-modelo tendrá su propia struct con los campos específicos que necesita (ej. `InstallModel` tendrá los `textinput.Model`).
2. **Orquestación:** El modelo principal en `model.go` mantendrá el estado global compartido (Configuración, Versión, Tamaño de Ventana) y delegará los mensajes de `Update` y `View` al sub-modelo activo.
3. **Comunicación por Mensajes:** Se estandarizarán los mensajes para que los sub-modelos puedan disparar acciones que afecten al estado global o cambien la vista activa.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/tui/model.go` | Modified | Se reducirá drásticamente para ser solo el orquestador. |
| `internal/tui/splash.go` | New | Lógica de la pantalla de carga inicial. |
| `internal/tui/menu.go` | New | Lógica del menú principal y el dashboard de estado. |
| `internal/tui/install.go` | New | Lógica del formulario de configuración de instalación. |
| `internal/tui/operation.go` | New | Pantalla de progreso con spinner y logs de streaming. |
| `internal/tui/confirm.go` | New | Diálogos de confirmación de acciones. |
| `internal/tui/result.go` | New | Pantalla de éxito o error final. |
| `internal/tui/docker_views.go` | New | Vistas de instalación de Docker (Linux/Windows). |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Pérdida de estado compartido al cambiar de vista | Medium | Pasar referencias (punteros) a la configuración global a los sub-modelos. |
| Complejidad en el "bubbling" de comandos (`tea.Cmd`) | Low | Usar patrones estándar de Bubble Tea para retornar comandos desde los submódulos. |
| Regresiones en el flujo de navegación | Low | Verificación manual paso a paso de cada transición de pantalla. |

## Rollback Plan
En caso de fallas imprevistas, se puede revertir al estado anterior del archivo `internal/tui/model.go` mediante el uso de control de versiones (`git checkout internal/tui/model.go`) y eliminando los nuevos archivos creados.

## Success Criteria
- [ ] `internal/tui/model.go` reducido a menos de 400 líneas (reducción del >70%).
- [ ] Ningún archivo de sub-modelo supera las 400 líneas de código.
- [ ] La aplicación compila sin errores.
- [ ] Todas las funcionalidades de la TUI (instalación, borrado, arranque, etc.) operan de forma idéntica a la versión actual.
