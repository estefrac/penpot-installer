# Tasks: Refactor TUI Model

## Phase 1: Infrastructure & Common Types
- [ ] 1.1 Definir `Common` struct en `internal/tui/model.go` para agrupar estado compartido (Config, WindowSize, Version).
- [ ] 1.2 Asegurar que los mensajes en `internal/tui/messages.go` cubran todas las necesidades de comunicación entre sub-modelos.

## Phase 2: Creation of Sub-Models (Standalone)
- [ ] 2.1 Crear `internal/tui/view_splash.go` con `SplashModel`, migrando lógica de `renderSplash` y `handleSplashKey`.
- [ ] 2.2 Crear `internal/tui/view_menu.go` con `MenuModel`, migrando lógica de `renderMain`, `renderMenuPanel`, `renderInfoPanel` y `handleMenuKey`.
- [ ] 2.3 Crear `internal/tui/view_install.go` con `InstallModel`, migrando lógica de `renderInstall` y `handleInstallKey`.
- [ ] 2.4 Crear `internal/tui/view_operation.go` con `OperationModel`, migrando lógica de `renderOperation`, `processDockerLine` y el manejo del spinner.
- [ ] 2.5 Crear `internal/tui/view_confirm.go` con `ConfirmModel`, migrando lógica de `renderConfirm`, `renderUninstallData` y sus respectivos handlers.
- [ ] 2.6 Crear `internal/tui/view_result.go` con `ResultModel`, migrando lógica de `renderResult` y `handleResultKey`.
- [ ] 2.7 Crear `internal/tui/view_docker.go` con `DockerModel`, migrando las 3 vistas de error/instalación de Docker y sus handlers.
- [ ] 2.8 Crear `internal/tui/view_status.go` con `StatusModel`, migrando lógica de `renderStatus` y `handleStatusKey`.

## Phase 3: Main Model Orchestration (The Big Cut)
- [ ] 3.1 Limpiar `Model` struct en `internal/tui/model.go`, dejando solo el estado global y las instancias de los sub-modelos.
- [ ] 3.2 Refactorizar `Update` en `internal/tui/model.go` para delegar mensajes al sub-modelo activo según `currentView`.
- [ ] 3.3 Refactorizar `View` en `internal/tui/model.go` para delegar el renderizado al sub-modelo activo.
- [ ] 3.4 Eliminar todas las funciones de ayuda (`render*`, `handle*`) obsoletas del `model.go` principal.

## Phase 4: Verification & Cleanup
- [ ] 4.1 Verificar que el proyecto compile (`go build`).
- [ ] 4.2 Realizar prueba manual del flujo completo: Splash -> Menu -> Status -> Menu.
- [ ] 4.3 Realizar prueba manual de operación: Menu -> Install -> Confirm -> Operation -> Result.
- [ ] 4.4 Eliminar código muerto y comentarios sobrantes.
