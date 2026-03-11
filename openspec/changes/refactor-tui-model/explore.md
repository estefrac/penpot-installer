## Exploration: Refactor TUI Model

### Current State
- `internal/tui/model.go` is a "God Object" of 1545 lines.
- It manages state, update logic, and view logic for 11 different screen views.
- The `Update` and `View` functions are large switch statements that delegate to helper methods.
- The `Model` struct contains fields for all screens (inputs, spinners, logs, menu items, etc.), leading to high coupling and poor maintainability.

### Affected Areas
- `internal/tui/model.go` — To be split into a main router and multiple sub-models.
- `internal/tui/splash.go` (new) — Initial loading/verifying screen.
- `internal/tui/menu.go` (new) — Main menu and status dashboard.
- `internal/tui/install.go` (new) — Configuration form.
- `internal/tui/operation.go` (new) — Progress/streaming screen.
- `internal/tui/confirm.go` (new) — Confirmation dialogs.
- `internal/tui/result.go` (new) — Success/Error result screen.
- `internal/tui/docker.go` (new) — Docker installation views for different OS.

### Approaches
1. **Composed Model (The Elm Way)** — Each view gets its own `Model`, `Update(msg)`, and `View()` functions. The main `Model` holds an `activeView` interface or a specific sub-model instance and delegates calls.
   - Pros: High decoupling, smaller files, easier testing, easier to add new screens.
   - Cons: Slightly more boilerplate for message passing.
   - Effort: Medium

2. **Partial Refactor (Helper Files)** — Keep one `Model` but move `handleKey` and `render` functions to separate files within the same `tui` package.
   - Pros: Very low effort, no changes to state management.
   - Cons: Still has a giant `Model` struct, high coupling remains, "God Object" lives on.
   - Effort: Low

### Recommendation
**Option 1: Composed Model.** It's the standard pattern for Bubble Tea applications that grow beyond a simple view. It solves the root cause (coupling) and makes the codebase professional.

### Risks
- **Message Dispatching:** Ensuring `tea.Cmd` and messages from sub-models are correctly bubbled up to the main loop.
- **Shared State:** Handling `penpot.Config` and `WindowSize` (width/height) across all sub-models needs a clean strategy (likely passed during `Update` or `New`).

### Ready for Proposal
Yes.
