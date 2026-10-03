# Component Architecture

This document describes the architecture and design patterns of the FontGet components library (`internal/components`).

## Component Hierarchy

```
Base Components (bubbletea / bubbles primitives)
    ↓
Simple Components (Button, Checkbox, Switch, textinput.Model)
    ↓
Composite Components (ButtonGroup, CheckboxList)
    ↓
Display / Layout (Card, Dialog, Overlay, Preview, StatusPopup, ProgressBar, Tables)
    ↓
Command Models (browse, sources_manage, theme, onboarding, etc.)
```

There is **no** form package (`UnifiedFormModel` / `FormModel` / `FormNavigation` were removed). Command models compose the widgets below directly.

## Component Categories

### 1. Input Components (User Input)

- **TextInput**: Use `textinput.Model` directly from `github.com/charmbracelet/bubbles/textinput`
  - No wrapper component
  - Apply styles directly: `input.TextStyle = ui.FormInput`
  - Handle background styling at render time if needed

- **CheckboxList** (`checkbox.go`): List of checkboxes with navigation
  - `HasFocus`, `SetFocus()`, `HandleKey()`, `Render()`

- **Switch** (`switch.go`): Toggle switch
  - `HasFocus`, `SetFocus()`, `HandleKey()`, `Render()`

### 2. Action Components (User Actions)

- **Button** / **ButtonGroup** (`button.go`): Button navigation and selection
  - `HasFocus`, `SetFocus()`, `HandleKey()`, `Render()`

- **ConfirmModel** (`confirm.go`): Confirmation dialog
  - Uses ButtonGroup internally
  - Full `tea.Model` implementation

### 3. Display Components (Information)

- **CardModel** / **card_sections.go**: Bordered cards and multi-section bodies
- **PreviewModel** (`preview.go`): Theme preview TUI
- **ProgressBarModel** (`progress_bar.go`): Multi-item progress display
- **StatusPopup** (`status_popup.go`): Centered install/status popup renderer
- **Dialog** (`dialog.go`): Modal dialog renderer

### 4. Table Components

- **table_static.go**: Static CLI table renderer
- **table_interactive.go**: Bubble Tea interactive table model
- **table_custom.go**: Viewport-controlled custom table
- **table_utils.go**: Shared table config, modes, and column helpers

Used by list/search output and interactive browse/theme flows.

### 5. Layout Components (Structure)

- **OverlayModel** (`overlay.go`): Overlay/modal compositing
- Blank/background helpers as needed by command models

## Standard Component Interface

Interactive widgets should follow:

```go
type Component interface {
    HasFocus bool
    SetFocus(bool)
    HandleKey(string) (handled bool, ...)
    Render() string
}
```

### Focus Management

- `HasFocus`: Whether the component currently has focus
- `SetFocus(bool)`: Set focus state
- Handle focus-related keys inside `HandleKey()` when appropriate

### Key Handling

- `HandleKey(string)`: Process keyboard input
- Returns whether the key was handled
- May return additional data (e.g., button actions)

### Rendering

- `Render()`: String representation of the component
- Should respect `HasFocus`
- Use styles from `internal/ui`

## Design Principles

1. **Composition over inheritance** — compose simple widgets in command models
2. **Single responsibility** — one clear job per component
3. **Consistent interfaces** — same focus/key/render pattern across interactive widgets
4. **Direct use of primitives** — prefer raw `textinput.Model` over wrappers
5. **Integer focus indices** — prefer `focusedComponent int` over string focus states

## Navigation Patterns

### Tab Navigation

```go
// Forward
focusedIdx = (focusedIdx + 1) % len(components)

// Backward
focusedIdx = (focusedIdx - 1 + len(components)) % len(components)
```

### Focus Updates

```go
func (m *Model) updateFocus() {
    for i := range m.components {
        m.blurComponent(i)
    }
    m.focusComponent(m.focusedIdx)
}
```

## Best Practices

1. Use raw `textinput.Model` — don't wrap unnecessarily
2. Integer-based focus — `focusedComponent int` instead of `FocusState string`
3. Centralized focus — single `updateFocus()` method
4. Simple Tab navigation — modulo arithmetic
5. Consistent styling — `internal/ui` styles
6. Type-safe enums/constants over stringly-typed modes

## Migration Notes

### From TextInput Wrapper to Raw textinput.Model

```go
pathInput := textinput.New()
pathInput.Placeholder = defaultPath
pathInput.Width = 60
pathInput.TextStyle = ui.FormInput
pathInput.PlaceholderStyle = ui.FormPlaceholder
// Handle background at render time if needed
```

### From String-Based Focus to Integer-Based

```go
focusedComponent int // 0=path, 1=checkboxes, 2=buttons
if m.focusedComponent == 0 { ... }
```

### From Manual Navigation to Modulo Arithmetic

```go
if key == "tab" {
    m.focusedComponent = (m.focusedComponent + 1) % 3
    m.updateFocus()
}
```

## Component Lifecycle

1. **Initialization**: `New*()` constructor
2. **Focus**: `SetFocus(true)` or `Focus()` for text inputs
3. **Update**: Handle messages in `Update()`
4. **Render**: Display in `View()` / `Render()`
5. **Cleanup**: Blur when leaving the screen

## Testing

Cover rendering states, key handling, focus, and edge cases (empty lists, out of bounds).

Existing tests include:

- `button_test.go`
- `checkbox_test.go`
- `switch_test.go`
- `table_test.go`
- `card_sections_test.go`
