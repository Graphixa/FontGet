# FontGet Style Guide

This document defines the color scheme and visual hierarchy for FontGet, ensuring a consistent and cohesive user interface across all commands. The colors are based on the Catppuccin Mocha palette.

## Color Palette

| Name       | Hex Code | Usage in FontGet                    |
|------------|----------|-------------------------------------|
| Rosewater  | #f5e0dc  | -                                   |
| Flamingo   | #f2cdcd  | -                                   |
| Pink       | #f5c2e7  | -                                   |
| Mauve      | #cba6f7  | -                                   |
| Red        | #f38ba8  | Error messages, critical warnings   |
| Maroon     | #eba0ac  | -                                   |
| Peach      | #fab387  | -                                   |
| Yellow     | #f9e2af  | Source names, field labels, warnings |
| Green      | #a6e3a1  | Success messages, enabled states    |
| Teal       | #94e2d5  | -                                   |
| Sky        | #89dceb  | -                                   |
| Sapphire   | #74c7ec  | -                                   |
| Blue       | #89b4fa  | Information, links                  |
| Lavender   | #b4befe  | -                                   |
| Text       | #cdd6f4  | Primary text, help text, normal text |
| Subtext 1  | #bac2de  | Command keys, secondary text        |
| Subtext 0  | #8c8c8c  | Built-in tags, tertiary text        |
| Overlay 2  | #9399b2  | -                                   |
| Overlay 1  | #7f849c  | -                                   |
| Overlay 0  | #6c7086  | -                                   |
| Surface 2  | #585b70  | -                                   |
| Surface 1  | #45475a  | -                                   |
| Surface 0  | #313244  | Title backgrounds, key backgrounds  |
| Base       | #1e1e2e  | Main background                     |
| Mantle     | #181825  | -                                   |
| Crust      | #11111b  | -                                   |

## FontGet Color Hierarchy

### Primary Colors
- **Source Names**: Yellow (#f9e2af) - Bold, prominent display
- **Field Labels**: Yellow (#f9e2af) - Bold, form field names
- **Form Input**: White (#FFFFFF) - Editable input content
- **Form Read-Only**: Overlay 0 (#6c7086) - Non-editable input content
- **Form Placeholder**: Overlay 1 (#7f849c) - Input placeholder text
- **Primary Text**: Text (#cdd6f4) - Regular content
- **Help Text**: Overlay 0 (#6c7086) - Secondary information, subtitles
- **Command Labels**: Subtext 1 (#bac2de) - Bold, button-like labels (Move, Submit, Cancel)
- **Command Keys**: Subtext 1 (#bac2de) - Bold, key bindings
- **Titles**: Mauve (#cba6f7) - Bold, with Surface 0 background
- **Headers**: Blue (#89b4fa) - Bold, no background (for table headers and labels)
- **Status Report Titles**: Mauve (#cba6f7) - Bold, no background

### Status Colors
- **Success**: Green (#a6e3a1) - Bold, success messages
- **Error**: Red (#f38ba8) - Bold, error messages
- **Information**: Blue (#89b4fa) - Links, info messages

### Tag Colors
- **Built-in Tags**: Custom Gray (#8c8c8c) - [Built-in] labels
- **Custom Tags**: Text (#cdd6f4) - [Custom] labels

### Background Colors
- **Main Background**: Base (#1e1e2e)
- **Surface Elements**: Surface 0 (#313244) - Titles, key backgrounds
- **Selected Row**: #3C3C3C - List item selection

## Style Definitions

### Lipgloss Styles
```go
// Primary text and elements
titleStyle = lipgloss.NewStyle().
    Bold(true).
    Foreground(lipgloss.Color("#cba6f7")).
    Background(lipgloss.Color("#313244")).
    Padding(0, 1)

headerStyle = lipgloss.NewStyle().
    Bold(true).
    Foreground(lipgloss.Color("#b4befe")) // Lavender - no background

statusReportTitleStyle = lipgloss.NewStyle().
    Bold(true).
    Foreground(lipgloss.Color("#cba6f7")) // Mauve (same as Title but no background)

sourceNameStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("#f9e2af"))

fieldLabelStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("#f9e2af")).
    Bold(true)

formInputStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("#cdd6f4")) // Text - same as primary text

formReadOnlyStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("#6c7086")) // Overlay 0 - darker gray for disabled state

formPlaceholderStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("#7f849c")) // Overlay 1 - mid-gray for placeholders

builtInTagStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("#9399b2")) // Overlay 2 - gray for tags used on lists

// Text hierarchy
helpTextStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("#6c7086")) // Overlay 0 - darker gray for secondary info

commandLabelStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("#bac2de")). // Subtext 1 - same as command keys
    Bold(true)

keyStyle = lipgloss.NewStyle().
    Bold(true).
    Foreground(lipgloss.Color("#bac2de")).
    Background(lipgloss.Color("#313244")).
    Padding(0, 1)

// Status colors
errorStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("#f38ba8")).
    Bold(true)

successStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("#a6e3a1")).
    Bold(true)

// Tag colors
builtInTagStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("#a6adc8"))

customTagStyle = lipgloss.NewStyle().
    Foreground(lipgloss.Color("#cdd6f4"))
```

### Fatih Color Usage
For commands that use `github.com/fatih/color`, use these equivalents:
- **Red**: `color.New(color.FgRed).SprintFunc()` → #f38ba8
- **Green**: `color.New(color.FgGreen).SprintFunc()` → #a6e3a1
- **Yellow**: `color.New(color.FgYellow).SprintFunc()` → #f9e2af
- **Cyan**: `color.New(color.FgCyan).SprintFunc()` → #89dceb

## Usage Guidelines

### Text Hierarchy
1. **Titles**: Mauve (#cba6f7) with Surface 0 background
2. **Source Names**: Yellow (#f9e2af) - most prominent content
3. **Field Labels**: Yellow (#f9e2af) - form labels
4. **Primary Text**: Text (#cdd6f4) - regular content
5. **Help Text**: Text (#cdd6f4) - descriptive text
6. **Command Keys**: Subtext 1 (#bac2de) - key bindings
7. **Tags**: Custom Gray (#8c8c8c) for built-in, Text (#cdd6f4) for custom

### Status Messages
- **Success**: Green (#a6e3a1) with bold
- **Error**: Red (#f38ba8) with bold
- **Warning**: Yellow (#f9e2af) with bold
- **Info**: Blue (#89b4fa)

### Background Usage
- **Main Background**: Base (#1e1e2e)
- **Surface Elements**: Surface 0 (#313244) for titles and key backgrounds
- **Selection**: #3C3C3C for selected list items

## Implementation Notes

- Use Lipgloss for TUI components and Bubble Tea interfaces
- Use Fatih Color for CLI output and terminal text
- Ensure sufficient contrast between text and backgrounds
- Maintain consistency across all FontGet commands
- Test color combinations for accessibility

## Examples

### Source List Item
```
> [x] FontSquirrel [Built-in]
```
- `>`: Default terminal color
- `[x]`: Default terminal color
- `FontSquirrel`: Yellow (#f9e2af)
- `[Built-in]`: Custom Gray (#8c8c8c)

### Form Field
```
> Name: FontSquirrel
```
- `>`: Default terminal color
- `Name:`: Yellow (#f9e2af), bold
- `FontSquirrel`: Input field (default)

### Error Message
```
Error: Source with this name already exists
```
- `Error:`: Red (#f38ba8), bold
- Rest: Default terminal color
