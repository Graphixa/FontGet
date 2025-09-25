# FontGet Style Guide

This document defines the color scheme and visual hierarchy for FontGet, ensuring a consistent and cohesive user interface across all commands. The colors are based on the Catppuccin Mocha palette.

## Color Palette

| Name       | Hex Code | Usage in FontGet                    |
|------------|----------|-------------------------------------|
| Rosewater  | #f5e0dc  | -                                   |
| Flamingo   | #f2cdcd  | -                                   |
| Pink       | #f5c2e7  | -                                   |
| Mauve      | #cba6f7  | Page titles, report titles, info messages |
| Red        | #f38ba8  | Error messages, critical warnings   |
| Maroon     | #eba0ac  | -                                   |
| Peach      | #fab387  | Warning messages, skipped status    |
| Yellow     | #f9e2af  | Source names, field labels, examples |
| Green      | #a6e3a1  | Success messages, installed status  |
| Teal       | #94e2d5  | -                                   |
| Sky        | #89dceb  | -                                   |
| Sapphire   | #74c7ec  | -                                   |
| Blue       | #89b4fa  | Information, links                  |
| Lavender   | #b4befe  | -                                   |
| Text       | #cdd6f4  | Primary text, regular content       |
| Subtext 1  | #bac2de  | Command keys, command labels        |
| Subtext 0  | #9399b2  | Built-in tags, tertiary text        |
| Overlay 2  | #9399b2  | -                                   |
| Overlay 1  | #7f849c  | Form placeholders                   |
| Overlay 0  | #6c7086  | Page subtitles, form read-only      |
| Surface 2  | #585b70  | -                                   |
| Surface 1  | #45475a  | -                                   |
| Surface 0  | #313244  | Title backgrounds, key backgrounds  |
| Base       | #1e1e2e  | Main background                     |
| Mantle     | #181825  | -                                   |
| Crust      | #11111b  | -                                   |

## Style Categories

FontGet uses a clear categorization system for different types of UI elements:

### 1. PAGE STRUCTURE STYLES - Layout hierarchy and page elements
- **PageTitle** - Main page titles (Mauve with Surface 0 background)
- **PageSubtitle** - Section subtitles (Overlay 0 - grayish)
- **ReportTitle** - Status report titles (Mauve, no background)
- **ContentText** - Regular text content (Adaptive colors for terminal compatibility)
- **ContentHighlight** - Highlighted content (Yellow)

### 2. USER FEEDBACK STYLES - Interactive responses and notifications
- **FeedbackInfo** - Informational messages (Mauve)
- **FeedbackText** - Supporting text (Terminal default)
- **FeedbackWarning** - Warning messages (Peach)
- **FeedbackError** - Error messages (Red)
- **FeedbackSuccess** - Success messages (Green)

### 3. DATA DISPLAY STYLES - Tables, lists, and data presentation
- **TableHeader** - Column headers (Adaptive text)
- **TableSourceName** - Font names in search/add results (Yellow)
- **TableRow** - Regular table rows (Adaptive text)
- **TableSelectedRow** - Selected rows (Surface 0 background)

### 4. FORM STYLES - Input interfaces and forms
- **FormLabel** - Field labels (Yellow)
- **FormInput** - Input field content (Adaptive text)
- **FormPlaceholder** - Placeholder text (Overlay 1)
- **FormReadOnly** - Read-only field content (Overlay 0)

### 5. COMMAND STYLES - Interactive elements and controls
- **CommandKey** - Keyboard shortcuts (Subtext 1 with Surface 0 background)
- **CommandLabel** - Button-like labels (Subtext 1)
- **CommandExample** - Example commands (Yellow)

## Implementation

### Style Usage Examples

```go
// Page titles
ui.PageTitle.Render("Font Search Results")

// Status messages
ui.FeedbackSuccess.Render("Installed")
ui.FeedbackWarning.Render("Skipped")
ui.FeedbackError.Render("Failed to install")

// Table content
ui.TableSourceName.Render("Roboto")
ui.TableHeader.Render("Name")

// Form elements
ui.FormLabel.Render("Name:")
ui.FormInput.Render("user input")
ui.FormPlaceholder.Render("Enter font name...")

// Command elements
ui.CommandKey.Render("Enter")
ui.CommandLabel.Render("Submit")
ui.CommandExample.Render("fontget add google.roboto")
```

### Status Report Styling

Status reports use a specific pattern where only the status word is colored:

```go
// Instead of coloring the entire message
msg := fmt.Sprintf("  - \"%s\" (%s to %s scope)", 
    fontDisplayName, 
    ui.FeedbackSuccess.Render("Installed"), 
    scope)
fmt.Println(ui.ContentText.Render(msg))
```

This creates:
- **Font name and description** → Normal text color
- **Status word "(Installed)"** → Green
- **Status word "(Skipped)"** → Peach
- **Status word "(Failed)"** → Red

## Color Hierarchy

### Primary Visual Elements
1. **Page Titles** - Mauve (#cba6f7) with background
2. **Source Names** - Yellow (#f9e2af) - most prominent content
3. **Field Labels** - Yellow (#f9e2af) - form labels
4. **Status Words** - Green/Peach/Red based on status
5. **Primary Text** - Adaptive text color for terminal compatibility

### Status Colors
- **Success** - Green (#a6e3a1) with bold
- **Warning** - Peach (#fab387) with bold
- **Error** - Red (#f38ba8) with bold
- **Info** - Mauve (#cba6f7) with bold

### Background Usage
- **Page Titles** - Surface 0 (#313244) background
- **Command Keys** - Surface 0 (#313244) background
- **Selected Rows** - Surface 0 (#313244) background
- **Regular Content** - No background (terminal default)

## Adaptive Colors

Several styles use `lipgloss.AdaptiveColor` for better terminal compatibility:

- **ContentText** - Adapts to light/dark terminals
- **TableHeader** - Adapts to light/dark terminals
- **TableRow** - Adapts to light/dark terminals
- **FormInput** - Adapts to light/dark terminals

## Usage Guidelines

1. **Consistency** - Use the same style category for similar elements
2. **Hierarchy** - Page titles > Source names > Field labels > Regular text
3. **Status Clarity** - Only color status words, not entire messages
4. **Terminal Compatibility** - Use adaptive colors where appropriate
5. **Accessibility** - Ensure sufficient contrast between text and backgrounds

## Migration from Old Styles

The old style names have been updated to the new categorization:

| Old Name | New Name | Category |
|----------|----------|----------|
| `Title` | `PageTitle` | Page Structure |
| `Header` | `TableHeader` | Data Display |
| `SourceName` | `TableSourceName` | Data Display |
| `MessageText` | `FeedbackText` | User Feedback |
| `MessageWarning` | `FeedbackWarning` | User Feedback |
| `MessageError` | `FeedbackError` | User Feedback |
| `MessageSuccess` | `FeedbackSuccess` | User Feedback |
| `ContentSubtitle` | `ReportTitle` | Page Structure |