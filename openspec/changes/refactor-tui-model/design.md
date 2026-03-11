# Design: Refactor TUI Model

## Technical Approach
Se migrará de un modelo monolítico a un sistema de **Modelos Compuestos**. El modelo principal (`tui.Model`) actuará como un orquestador y contenedor de estado global, delegando la lógica de presentación y eventos a sub-modelos especializados.

## Architecture Decisions

### Decision: Pattern de Modelos Compuestos
**Choice**: Composición de structs con delegación explícita.
**Alternatives considered**: Interfaces dinámicas (`activeView tea.Model`).
**Rationale**: Bubble Tea funciona mejor con tipos concretos para evitar casts innecesarios. Mantener los sub-modelos como campos en el modelo principal facilita el acceso al estado y la persistencia de datos entre transiciones.

### Decision: Estado Compartido (Common State)
**Choice**: Pasar una referencia a los datos globales en cada `Update` de los sub-modelos.
**Alternatives considered**: Punteros compartidos en cada sub-modelo.
**Rationale**: Evita problemas de sincronización. El modelo principal es el único "dueño" de la verdad (`Config`, `WindowSize`), y los sub-modelos reciben lo que necesitan para renderizar.

## Data Flow
El flujo de mensajes (`tea.Msg`) seguirá el patrón estándar de Bubble Tea, pero con una capa de ruteo:

1.  `main.go` envía `msg` al `Model.Update`.
2.  `Model.Update` identifica la `currentView`.
3.  `Model.Update` llama al `Update` del sub-modelo correspondiente (ej. `m.menu.Update(msg)`).
4.  El sub-modelo retorna un nuevo estado de sí mismo y un `tea.Cmd`.
5.  `Model.Update` retorna los comandos al loop principal.

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/tui/model.go` | Modify | Reducción a orquestador, eliminación de renders y handlers locales. |
| `internal/tui/view_splash.go` | Create | Sub-modelo para la pantalla de carga inicial. |
| `internal/tui/view_menu.go` | Create | Sub-modelo para el menú y dashboard de estado. |
| `internal/tui/view_install.go` | Create | Sub-modelo para el formulario de configuración. |
| `internal/tui/view_operation.go` | Create | Sub-modelo para el progreso de Docker (spinner + logs). |
| `internal/tui/view_confirm.go` | Create | Sub-modelo para diálogos de confirmación. |
| `internal/tui/view_result.go` | Create | Sub-modelo para resultados finales. |
| `internal/tui/view_docker.go` | Create | Sub-modelo para las pantallas de error/instalación de Docker. |

## Interfaces / Contracts

Cada sub-modelo seguirá este patrón (ejemplo para el Menú):

```go
type MenuModel struct {
    cursor int
    // ... otros campos locales
}

func (m MenuModel) Update(msg tea.Msg, cfg penpot.Config) (MenuModel, tea.Cmd) {
    // Lógica local
}

func (m MenuModel) View(cfg penpot.Config, width int) string {
    // Renderizado local usando estilos globales
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | Transiciones de estado | Verificar que al presionar 'Enter' en el menú, el modelo principal cambie la `currentView`. |
| Manual | Flujo completo | Instalar, Detener y Desinstalar Penpot verificando que la UI se comporte igual. |

## Migration / Rollout
No requiere migración de datos. Es un refactor de código interno. Se realizará en ramas o pasos incrementales para asegurar que la aplicación siempre compile.
