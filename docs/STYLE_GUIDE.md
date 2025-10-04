# FontGet Style Guide

This document defines the color scheme and visual hierarchy for FontGet, ensuring a consistent and cohesive user interface across all commands. The colors are based on the Catppuccin Mocha palette.

## Color Palette

| Name       | Hex Code | Usage in FontGet                    |
|------------|----------|-------------------------------------|
| Rosewater  | #f5e0dc  | -                                   |
| Flamingo   | #f2cdcd  | -                                   |
| Pink       | #f5c2e7  | Font names, field labels, examples |
| Mauve      | #cba6f7  | Page titles, report titles, info messages |
| Red        | #e78284  | Error messages, critical warnings   |
| Maroon     | #eba0ac  | -                                   |
| Peach      | #fab387  | -                                   |
| Yellow     | #f9e2af  | Warning messages, skipped status    |
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
- **FeedbackWarning** - Warning messages (Yellow)
- **FeedbackError** - Error messages (Red)
- **FeedbackSuccess** - Success messages (Green)

### 3. DATA DISPLAY STYLES - Tables, lists, and data presentation
- **TableHeader** - Column headers (Terminal default)
- **TableSourceName** - Font names in search/add results (Pink)
- **TableRow** - Regular table rows (Terminal default)
- **TableSelectedRow** - Selected rows (Surface 0 background)

### 4. FORM STYLES - Input interfaces and forms
- **FormLabel** - Field labels (Pink)
- **FormInput** - Input field content (Adaptive text)
- **FormPlaceholder** - Placeholder text (Overlay 1)
- **FormReadOnly** - Read-only field content (Overlay 0)

### 5. COMMAND STYLES - Interactive elements and controls
- **CommandKey** - Keyboard shortcuts (Subtext 1 with Surface 0 background)
- **CommandLabel** - Button-like labels (Subtext 1)
- **CommandExample** - Example commands (Pink)

### 6. CARD STYLES - Card components and layouts
- **CardTitle** - Card titles integrated into borders (Mauve with Surface 0 background)
- **CardLabel** - Labels within cards (Pink - matches FormLabel)
- **CardContent** - Regular content within cards (Terminal default - matches FeedbackText)
- **CardBorder** - Card border styling (Overlay 1 color)
- **CardContainer** - Container for cards with proper spacing

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

// Card elements
ui.CardTitle.Render("Font Details")
ui.CardLabel.Render("Name:")
ui.CardContent.Render("Roboto Mono")
ui.CardBorder.Render("Card content here")
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
- **Status word "(Skipped)"** → Yellow
- **Status word "(Failed)"** → Red

### Card Component Styling

Card components use a hierarchical approach with integrated titles:

```go
// Card with integrated title in border
card := components.NewCardWithSections("Font Details", []components.CardSection{
    {Label: "Name", Value: "Roboto Mono"},
    {Label: "ID", Value: "google.roboto-mono"},
    {Label: "Category", Value: "Monospace"},
})

// Custom card with specific padding
customCard := components.Card{
    Title:             "Test Font",
    Content:           "Custom content here",
    Width:             80,
    VerticalPadding:   1,  // Top/bottom padding
    HorizontalPadding: 2,  // Left/right padding
}
```

This creates:
- **Card titles** → Integrated into top border with Mauve color and Surface 0 background
- **Card labels** → Pink color (e.g., "Name:", "ID:", "Category:")
- **Card content** → Terminal default color for values
- **Card borders** → Overlay 1 color with rounded corners

## Color Hierarchy

### Primary Visual Elements
1. **Page Titles** - Mauve (#cba6f7) with background
2. **Card Titles** - Mauve (#cba6f7) with Surface 0 background, integrated into borders
3. **Font Names & Labels** - Pink (#f5c2e7) - most prominent content, form labels, and card labels
4. **Warning Messages** - Yellow (#f9e2af) - warnings and skipped status
5. **Status Words** - Green/Yellow/Red based on status
6. **Primary Text** - Terminal default for better compatibility

### Status Colors
- **Success** - Green (#a6e3a1) with bold
- **Warning** - Yellow (#f9e2af) with bold
- **Error** - Red (#f38ba8) with bold
- **Info** - Mauve (#cba6f7) with bold

### Background Usage
- **Page Titles** - Surface 0 (#313244) background
- **Command Keys** - Surface 0 (#313244) background
- **Selected Rows** - Surface 0 (#313244) background
- **Regular Content** - No background (terminal default)

## Adaptive Colors

~~Several styles use `lipgloss.AdaptiveColor` for better terminal compatibility:~~
Not properly supported so don't use

- **ContentText** - Adapts to light/dark terminals
- **FormInput** - Adapts to light/dark terminals

Note: TableHeader and TableRow now use terminal default colors for better compatibility across different terminal environments.

## Table Standards

### Maximum Table Width
All tables in FontGet are designed to efficiently use standard terminal space:
- **Maximum total width**: 120 characters (uses full 120-character terminals)
- **Column spacing**: 1 space between columns
- **Separator line**: Matches table width exactly
- **Space utilization**: Maximum readability with full terminal width

### Table Column Standards
Different commands use different column structures based on their purpose:

#### Font Search/Add/Remove Tables (5 columns, 120 chars total)
- **Name**: 36 characters (font display name - wider for longer names)
- **ID**: 34 characters (font ID like "nerd.font-name" - much wider)
- **License**: 12 characters (license type - slightly wider)
- **Categories**: 16 characters (font categories - wider for multiple)
- **Source**: 18 characters (source name - wider)

#### Font List Tables (5 columns, 120 chars total)
- **Name**: 54 characters (font family name - much wider)
- **Style**: 22 characters (font style/variant - wider)
- **Type**: 10 characters (file type)
- **Installed**: 20 characters (installation date)
- **Scope**: 10 characters (user/machine)

#### Sources Management Tables (2 columns, 120 chars total)
- **Status**: 10 characters (checkbox/status)
- **Name**: 109 characters (source name with tags - much wider)

### Implementation
Use the shared table functions in `cmd/shared.go` for consistent formatting:
```go
// For font search/add/remove tables
fmt.Printf("%s\n", ui.TableHeader.Render(GetTableHeader()))
fmt.Printf("%s\n", GetTableSeparator())

// For custom tables, use the column constants
fmt.Printf("%-*s %-*s %-*s\n", 
    TableColName, "Name",
    TableColID, "ID", 
    TableColSource, "Source")
```

## Usage Guidelines

1. **Consistency** - Use the same style category for similar elements
2. **Hierarchy** - Page titles > Card titles > Font names > Field labels > Regular text
3. **Status Clarity** - Only color status words, not entire messages
4. **Terminal Compatibility** - Use adaptive colors where appropriate
5. **Accessibility** - Ensure sufficient contrast between text and backgrounds
6. **Table Width** - Never exceed 120 characters total width for tables
7. **Card Design** - Use integrated titles in borders for better visual hierarchy
8. **Padding Control** - Use vertical and horizontal padding separately for different use cases

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