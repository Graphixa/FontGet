# Component Architecture

This document describes the architecture and design patterns of the FontGet components library.

## Component Hierarchy

```
Base Components (bubbletea primitives)
    ↓
Simple Components (Button, Checkbox, Switch, TextInput)
    ↓
Composite Components (ButtonGroup, CheckboxList)
    ↓
Form Components (UnifiedFormModel, FormModel, FormNavigation)
    ↓
Command Models (backup, sources_manage, etc.)
```

## Component Categories

### 1. Input Components (User Input)

These components handle direct user input:

- **TextInput**: Use `textinput.Model` directly from `github.com/charmbracelet/bubbles/textinput`
  - No wrapper component needed
  - Apply styles directly: `input.TextStyle = ui.FormInput`
  - Handle background styling at render time if needed

- **CheckboxList**: List of checkboxes with navigation
  - `HasFocus`, `SetFocus()`, `HandleKey()`, `Render()`
  - Well-designed, keep as-is

- **Switch**: Toggle switch component
  - `HasFocus`, `SetFocus()`, `HandleKey()`, `Render()`
  - Standardized interface

### 2. Action Components (User Actions)

These components handle user actions:

- **Button** / **ButtonGroup**: Button navigation and selection
  - `HasFocus`, `SetFocus()`, `HandleKey()`, `Render()`
  - Well-designed, keep as-is

- **ConfirmModel**: Confirmation dialog
  - Uses ButtonGroup internally
  - Full `tea.Model` implementation

### 3. Form Components (Composite)

These components combine multiple input/action components:

- **UnifiedFormModel**: Comprehensive form component
  - Supports mixed component types (text inputs, checkboxes, buttons)
  - Unified navigation and validation
  - Use for new forms

- **FormModel**: Simple text-only form
  - Deprecated in favor of UnifiedFormModel
  - Keep for backward compatibility

- **FormNavigation**: Navigation helper for list + buttons
  - Handles Tab navigation between list and buttons
  - Can be enhanced to work with UnifiedFormModel

### 4. Display Components (Information)

These components display information:

- **CardModel**: Card display component
- **PreviewModel**: Preview display component
- **ProgressBarModel**: Progress indicator
  - Well-designed, keep as-is

### 5. Layout Components (Structure)

These components provide layout structure:

- **OverlayModel**: Overlay/modal layout
- **BlankBackgroundModel**: Blank background for modals

## Standard Component Interface

All interactive components should implement:

```go
type Component interface {
    HasFocus bool
    SetFocus(bool)
    HandleKey(string) (handled bool, ...)
    Render() string
}
```

### Focus Management

- `HasFocus`: Boolean indicating if component currently has focus
- `SetFocus(bool)`: Set focus state
- Components should handle focus in `HandleKey()` when appropriate keys are pressed

### Key Handling

- `HandleKey(string)`: Process keyboard input
- Returns whether the key was handled
- May return additional data (e.g., button actions)

### Rendering

- `Render()`: Return string representation of component
- Should respect `HasFocus` state
- Use UI styles from `internal/ui` package

## Design Principles

### 1. Composition over Inheritance

Prefer composition of simple components over complex inheritance hierarchies.

### 2. Single Responsibility

Each component should have a single, well-defined purpose.

### 3. Consistent Interfaces

All similar components should follow the same interface patterns.

### 4. Direct Use of Primitives

Use bubbletea primitives directly (e.g., `textinput.Model`) rather than wrapping unnecessarily.

### 5. Focus Management

Centralize focus management using integer-based indices rather than string-based states.

## Navigation Patterns

### Tab Navigation

Use modulo arithmetic for wrapping navigation:

```go
// Forward
focusedIdx = (focusedIdx + 1) % len(components)

// Backward
focusedIdx = (focusedIdx - 1 + len(components)) % len(components)
```

### Focus Updates

Centralize focus updates in a single method:

```go
func (m *Model) updateFocus() {
    // Blur all components
    for i := range m.components {
        m.blurComponent(i)
    }
    // Focus current component
    m.focusComponent(m.focusedIdx)
}
```

## Best Practices

1. **Use raw `textinput.Model`**: Don't wrap unnecessarily
2. **Integer-based focus**: Use `focusedComponent int` instead of `FocusState string`
3. **Centralized focus management**: Single `updateFocus()` method
4. **Simple navigation**: 2-line modulo arithmetic for Tab navigation
5. **Consistent styling**: Use `internal/ui` styles
6. **Type-safe**: Prefer enums/constants over strings for types

## Migration Guide

### From TextInput Wrapper to Raw textinput.Model

**Before:**
```go
pathInput := components.NewTextInput(components.TextInputOptions{
    Placeholder:    defaultPath,
    FixedWidth:     60,
    WithBackground: true,
})
```

**After:**
```go
pathInput := textinput.New()
pathInput.Placeholder = defaultPath
pathInput.Width = 60
pathInput.TextStyle = ui.FormInput
pathInput.PlaceholderStyle = ui.FormPlaceholder
// Handle background at render time if needed
```

### From String-Based Focus to Integer-Based

**Before:**
```go
FocusState string // "path", "checkboxes", "buttons"
if m.FocusState == "path" { ... }
```

**After:**
```go
focusedComponent int // 0=path, 1=checkboxes, 2=buttons
if m.focusedComponent == 0 { ... }
```

### From Manual Navigation to Modulo Arithmetic

**Before:**
```go
if key == "tab" {
    switch m.FocusState {
    case "path":
        m.FocusState = "checkboxes"
    case "checkboxes":
        m.FocusState = "buttons"
    case "buttons":
        m.FocusState = "path"
    }
}
```

**After:**
```go
if key == "tab" {
    m.focusedComponent = (m.focusedComponent + 1) % 3
    m.updateFocus()
}
```

## Component Lifecycle

1. **Initialization**: Create component with `New*()` function
2. **Focus**: Set initial focus with `SetFocus(true)` or `Focus()` for text inputs
3. **Update**: Handle messages in `Update()` method
4. **Render**: Display component in `View()` method
5. **Cleanup**: Blur components when done

## Testing

All components should have comprehensive tests covering:

- Rendering with different states
- Key handling
- Focus management
- Edge cases (empty lists, out of bounds, etc.)

See existing test files for examples:
- `button_test.go`
- `checkbox_test.go`
- `switch_test.go`
- `unified_form_test.go`

