# FontGet Style Guide

This document defines the styling system and visual hierarchy for FontGet. FontGet uses a **theme-based color system** that loads colors from YAML theme files, allowing for customization and support for both dark and light modes.

## Theme System Overview

FontGet uses a **semantic color system** where theme files define color keys (like `accent`, `warning`, `error`) that are mapped to UI styles. This approach provides:

- **Consistency**: All styles reference semantic colors, not hardcoded hex values
- **Customization**: Users can create custom themes by editing YAML files
- **Dark/Light Mode**: Themes support both dark and light modes
- **Auto-detection**: Terminal theme can be automatically detected

### Theme File Structure

Themes are stored in `~/.fontget/themes/` and follow this structure:

```yaml
fontget_theme:
  dark_mode:
    accent: "#cba6f7"          # Primary accent color
    accent2: "#94e2d5"          # Secondary accent color
    warning: "#f9e2af"          # Warning messages
    error: "#e78284"            # Error messages
    success: "#a6e3a1"           # Success messages
    page_title: "#cba6f7"       # Page title color
    page_subtitle: "#7f849c"    # Subtitle color
    grey_light: "#cdd6f4"       # Light text color
    grey_mid: "#7f849c"          # Medium grey (borders, placeholders)
    grey_dark: "#313244"         # Dark background color
    progress_bar_gradient:
      color_start: "#cba6f7"    # Gradient start
      color_end: "#eba0ac"       # Gradient end
  light_mode:
    # Same keys with light mode colors
```

### Default Theme

The default theme is **Catppuccin** (Mocha for dark mode, Latte for light mode), which is embedded in the binary. If no custom theme is specified, this theme is used automatically.

### Theme Configuration

Themes are configured in `~/.fontget/config.yaml`:

```yaml
theme:
  name: "catppuccin"  # Theme name (file: catppuccin.yaml or catppuccin-theme.yaml)
  mode: "auto"        # "dark", "light", or "auto" (auto-detects terminal theme)
```

## Semantic Color System

FontGet uses semantic color keys that are mapped to multiple styles. This table shows which styles use each semantic color:

| Semantic Color | Default (Dark) | Used By (Style Names) |
|----------------|----------------|----------------------|
| **accent** | #cba6f7 (Mauve) | `InfoText`, `PageTitle` (FG), `CardTitle` (FG), `CheckboxCursor`, `SpinnerColor`, `ProgressBarGradientStart` |
| **accent2** | #94e2d5 (Teal) | `TableSourceName`, `FormLabel`, `CardLabel`, `CheckboxChecked` |
| **warning** | #f9e2af (Yellow) | `WarningText` |
| **error** | #e78284 (Red) | `ErrorText` |
| **success** | #a6e3a1 (Green) | `SuccessText`, `SpinnerDoneColor` |
| **page_title** | #cba6f7 (Mauve) | `PageTitle` (foreground) |
| **page_subtitle** | #7f849c (Overlay 1) | `PageSubtitle` |
| **grey_light** | #cdd6f4 (Text) | `ButtonNormal`, `ButtonSelected` (FG), `SwitchLeftNormal`, `SwitchRightNormal`, `SwitchLeftSelected` (FG), `SwitchRightSelected` (FG) |
| **grey_mid** | #7f849c (Overlay 1) | `FormPlaceholder`, `CardBorder`, `SwitchSeparator`, `CommandKey` (FG), `CommandLabel`, `FormReadOnly` |
| **grey_dark** | #313244 (Surface 0) | `PageTitle` (BG), `TableRowSelected` (BG), `CardTitle` (BG), `CommandKey` (BG), `ButtonSelected` (BG), `CheckboxItemSelected` (BG), `SwitchLeftSelected` (BG), `SwitchRightSelected` (BG) |

**Note**: Some styles use terminal default colors (no theme color):
- `Text`, `TextBold`, `TableHeader`, `CommandExample`, `CardContent`, `CheckboxUnchecked` - Terminal default
- These styles are not affected by theme changes

## Style Categories

FontGet uses a clear categorization system for different types of UI elements:

### 1. PAGE STRUCTURE STYLES - Layout hierarchy and page elements
- **PageTitle** - Main page titles (uses `accent`/`page_title` with `grey_dark` background)
- **PageSubtitle** - Section subtitles (uses `page_subtitle`/`grey_mid`)

### 2. MESSAGE STYLES - User notifications and responses
- **Text** - Regular text content (Terminal default - no theme color)
- **InfoText** - Informational messages (uses `accent`)
- **SecondaryText** - Secondary informational text (uses `accent2`)
- **WarningText** - Warning messages (uses `warning`)
- **ErrorText** - Error messages (uses `error`)
- **SuccessText** - Success messages (uses `success`)
- **TextBold** - Bold text with terminal default color

### 3. DATA DISPLAY STYLES - Tables, lists, and data presentation
- **TableHeader** - Column headers (Terminal default - no theme color)
- **TableSourceName** - Font names in search/add results (uses `accent2`)
  - Used by `RenderSourceNameWithTag()` for colored source names with tags
  - For plain source names, use `Text.Render(name) + " " + RenderSourceTag(isBuiltIn)`
- **TableRow** - Regular table rows (Terminal default - no theme color)
- **TableRowSelected** - Selected rows (uses `grey_dark` background)

### 4. FORM STYLES - Input interfaces and forms
- **FormLabel** - Field labels (uses `accent2`)
- **FormInput** - Input field content (Terminal default - no theme color)
- **FormPlaceholder** - Placeholder text (uses `grey_mid`)
- **FormReadOnly** - Read-only field content (uses `grey_mid`)

### 5. COMMAND STYLES - Interactive elements and controls
- **CommandKey** - Keyboard shortcuts (uses `grey_mid` with `grey_dark` background)
- **CommandLabel** - Button-like labels (uses `grey_mid`)
- **CommandExample** - Example commands (Terminal default - no theme color)

### 6. CARD STYLES - Card components and layouts
- **CardTitle** - Card titles integrated into borders (uses `accent` with `grey_dark` background)
- **CardLabel** - Labels within cards (uses `accent2`)
- **CardContent** - Regular content within cards (Terminal default - no theme color)
- **CardBorder** - Card border styling (uses `grey_mid`)

### 7. BUTTON COMPONENT STYLES
- **ButtonNormal** - Unselected button text (uses `grey_light`)
- **ButtonSelected** - Selected button (uses `grey_dark` text on `grey_light` background - inverted)

### 8. CHECKBOX COMPONENT STYLES
- **CheckboxUnchecked** - Unchecked checkbox (Terminal default - no theme color)
- **CheckboxChecked** - Checked checkbox (uses `accent2`)
- **CheckboxItemSelected** - Selected checkbox item background (uses `grey_dark`)
- **CheckboxCursor** - Checkbox cursor indicator (uses `accent2`)

### 9. SWITCH COMPONENT STYLES
- **SwitchLeftNormal** - Unselected left option (uses `grey_light`)
- **SwitchLeftSelected** - Selected left option (uses `grey_dark` text on `grey_light` background - inverted)
- **SwitchRightNormal** - Unselected right option (uses `grey_light`)
- **SwitchRightSelected** - Selected right option (uses `grey_dark` text on `grey_light` background - inverted)
- **SwitchSeparator** - Switch separator (uses `grey_mid`)

### 10. SPINNER COMPONENT
- **SpinnerColor** - Spinner color constant (uses `accent`)
- **SpinnerDoneColor** - Spinner done color constant (uses `success`)

## Implementation

### Style Initialization

Styles are initialized from the theme system during application startup:

```go
// In cmd/root.go or similar startup code
ui.InitThemeManager()  // Loads theme from config
ui.InitStyles()        // Applies theme colors to all styles
```

### Style Usage Examples

```go
// Page titles
ui.PageTitle.Render("Font Search Results")

// Status messages
ui.SuccessText.Render("Installed")
ui.WarningText.Render("Skipped")
ui.ErrorText.Render("Failed to install")

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

// Source name rendering
ui.RenderSourceNameWithTag("Google Fonts", true)  // Colored name with tag
ui.RenderSourceTag(true)  // Just the tag: "[Built-in]"
ui.Text.Render("Google Fonts") + " " + ui.RenderSourceTag(true)  // Plain name with styled tag
```

### Status Report Styling

Status reports use a specific pattern where only the status word is colored:

```go
// Instead of coloring the entire message
msg := fmt.Sprintf("  - \"%s\" (%s to %s scope)", 
    fontDisplayName, 
    ui.SuccessText.Render("Installed"), 
    scope)
fmt.Println(ui.Text.Render(msg))
```

This creates:
- **Font name and description** → Normal text color
- **Status word "(Installed)"** → Success color (green)
- **Status word "(Skipped)"** → Warning color (yellow)
- **Status word "(Failed)"** → Error color (red)

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
- **Card titles** → Integrated into top border with accent color and grey_dark background
- **Card labels** → Accent2 color (e.g., "Name:", "ID:", "Category:")
- **Card content** → Terminal default color for values
- **Card borders** → Grey_mid color with rounded corners

## Color Hierarchy

### Primary Visual Elements
1. **Page Titles** - Accent color with grey_dark background
2. **Card Titles** - Accent color with grey_dark background, integrated into borders
3. **Font Names & Labels** - Accent2 color - most prominent content, form labels, and card labels
4. **Warning Messages** - Warning color - warnings and skipped status
5. **Status Words** - Success/Warning/Error colors based on status
6. **Primary Text** - Terminal default for better compatibility

### Status Colors
- **Success** - Success color - Use `SuccessText`
- **Warning** - Warning color - Use `WarningText`
- **Error** - Error color - Use `ErrorText`
- **Info** - Accent color - Use `InfoText`

**Note:** The old `Feedback*` style names (`FeedbackText`, `FeedbackInfo`, `FeedbackWarning`, `FeedbackError`, `FeedbackSuccess`) are deprecated but still available as aliases for backward compatibility. Use the new names (`Text`, `InfoText`, `WarningText`, `ErrorText`, `SuccessText`) in all new code.

### Background Usage
- **Page Titles** - Grey_dark background
- **Command Keys** - Grey_dark background
- **Selected Rows** - Grey_dark background
- **Selected Buttons/Switches** - Grey_light background (inverted)
- **Regular Content** - No background (terminal default)

## Creating Custom Themes

To create a custom theme:

1. **Copy the default theme** from `internal/ui/themes/catppuccin.yaml` as a starting point
2. **Save it** to `~/.fontget/themes/your-theme.yaml` (or `your-theme-theme.yaml`)
3. **Edit the colors** in both `dark_mode` and `light_mode` sections
4. **Update config.yaml** to use your theme:
   ```yaml
   theme:
     name: "your-theme"
     mode: "auto"  # or "dark" or "light"
   ```

### Theme Validation

Themes must include all required color keys for both dark and light modes:
- `accent`, `accent2`, `warning`, `error`, `success`
- `page_title`, `page_subtitle`
- `grey_light`, `grey_mid`, `grey_dark`
- `progress_bar_gradient.color_start`, `progress_bar_gradient.color_end`

If a theme is missing required keys or fails validation, FontGet will fall back to the default Catppuccin theme.

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

## Utility Functions

### Source Name Rendering

FontGet provides utility functions for rendering source names with type tags:

- **`RenderSourceNameWithTag(name, isBuiltIn)`** - Renders source name with tag using colored `TableSourceName` style
  - Used in: search, add, and other commands where colored source names are desired
  - Example: `ui.RenderSourceNameWithTag("Google Fonts", true)` → "Google Fonts [Built-in]" (name colored, tag styled)

- **`RenderSourceTag(isBuiltIn)`** - Renders just the type tag (`[Built-in]` or `[Custom]`)
  - Can be used independently for flexible rendering
  - Example: `ui.RenderSourceTag(true)` → "[Built-in]"

- **Plain source names** - For uncolored source names with styled tags, combine manually:
  - Example: `ui.Text.Render("Google Fonts") + " " + ui.RenderSourceTag(true)`
  - Used in: sources manage command where source names should be plain text

**Note**: The sources manage command uses plain `Text` style for source names to maintain a cleaner, less colorful appearance, while still showing styled tags.

## Usage Guidelines

1. **Consistency** - Use the same style category for similar elements
2. **Hierarchy** - Page titles > Card titles > Font names > Field labels > Regular text
3. **Status Clarity** - Only color status words, not entire messages
4. **Theme Awareness** - Always use theme-aware styles, never hardcode colors
5. **Accessibility** - Ensure sufficient contrast between text and backgrounds in both dark and light modes
6. **Table Width** - Never exceed 120 characters total width for tables
7. **Card Design** - Use integrated titles in borders for better visual hierarchy
8. **Padding Control** - Use vertical and horizontal padding separately for different use cases
9. **Semantic Colors** - Reference semantic color keys in theme files, not specific hex values
10. **Terminal Defaults** - Use terminal default colors for base text to ensure compatibility
11. **Source Name Styling** - Use `RenderSourceNameWithTag()` for colored names, or combine `Text` with `RenderSourceTag()` for plain names

## Migration Notes

If you're updating existing code:

- **Don't hardcode colors** - Use theme-aware styles from `ui` package
- **Don't reference specific hex codes** - Colors come from theme files
- **Use semantic styles** - `InfoText` instead of a specific color
- **Test both modes** - Ensure your code works in both dark and light themes
- **Check contrast** - Verify readability in both theme modes
