# FontGet Style Guide

This document defines the styling system and visual hierarchy for FontGet. FontGet uses a **theme-based color system** that loads colors from YAML theme files, allowing for customization and support for both dark and light modes.

## Theme System Overview

FontGet uses a **semantic color system** where theme files define color keys (`primary`, `secondary`, `components`, etc.) that are mapped to UI styles. This approach provides:

- **Consistency**: All styles reference theme colors, not hardcoded hex values
- **Customization**: Users can create custom themes by editing YAML files
- **Dark/Light Mode**: Themes support both dark and light modes (via theme `style` or separate theme files)
- **Auto-detection**: Terminal theme can be automatically detected

### Theme File Structure

Themes are stored in `~/.fontget/themes/` and follow this structure. Each theme file has a single `colors` block; the top-level `style` field is optional (e.g. `dark` or `light`).

```yaml
theme_name: "My Theme"
style: dark   # optional: "dark" or "light"

colors:
  # Required base colors
  primary: "#cba6f7"       # Main accent – InfoText, PageTitle, CardTitle, Cursor, Spinner, QueryText
  secondary: "#94e2d5"    # Secondary accent – TableSourceName, FormLabel, CardLabel
  components: "#cdd6f4"   # Interactive elements – Button, Switch, FormInput
  placeholders: "#7f849c" # Muted elements – borders, placeholders, CommandKey text, CheckboxUnchecked
  base: "#313244"         # Backgrounds – PageTitle, CardTitle, CommandKey, selected states

  # Required status colors
  warning: "#f9e2af"
  error: "#e78284"
  success: "#a6e3a1"

  # Optional component overrides (defaults to base colors above)
  overrides:
    page_title:
      text: "#cba6f7"
      background: "#313244"
    button:
      foreground: "#cdd6f4"
      background: "#313244"
    switch:
      foreground: "#cdd6f4"
      background: "#313244"
    checkbox:
      unchecked: "#7f849c"
      checked: "#cba6f7"
    card:
      title_text: "#cba6f7"
      title_background: "#313244"
      label: "#94e2d5"
      border: "#7f849c"
    command_keys:
      text: "#7f849c"
      background: "#313244"
    table:
      header: "#94e2d5"
      row: "#cba6f7"
      selected: "#cdd6f4"
    spinner:
      normal: "#cba6f7"
      done: "#a6e3a1"
    progress_bar:
      start: "#cba6f7"
      finish: "#eba0ac"
```

### Default Theme

The default theme is **Catppuccin**, which is embedded in the binary. If no custom theme is specified, this theme is used automatically.

### Theme Configuration

Themes are configured in `~/.fontget/config.yaml`:

```yaml
theme:
  name: "catppuccin"  # Theme name (file: catppuccin.yaml or catppuccin-theme.yaml)
  mode: "auto"        # "dark", "light", or "auto" (auto-detects terminal theme)
```

## Theme Color Keys and Style Mapping

Theme YAML files use these **exact key names** under `colors`. This table shows which UI styles use each theme key:

| Theme Key | Default (Catppuccin) | Used By (Style Names) |
|-----------|----------------------|------------------------|
| **primary** | #cba6f7 | `InfoText`, `QueryText`, `PageTitle` (text), `CardTitle` (text), `Cursor`, `CheckboxChecked`, `SpinnerColor`, progress bar start (default) |
| **secondary** | #94e2d5 | `TableSourceName`, `FormLabel`, `CardLabel` |
| **components** | #cdd6f4 | `ButtonNormal` (fg), `FormInput`, `SwitchNormal` (fg), selected-state backgrounds (inverted) for `ButtonSelected`, `SwitchSelected`, `TableRowSelected` |
| **placeholders** | #7f849c | `FormPlaceholder`, `FormReadOnly`, `CardBorder`, `CommandKey` (text), `CheckboxUnchecked`, `SwitchSeparator`, `RenderSourceTag` (built-in tag color) |
| **base** | #313244 | `PageTitle` (bg), `CardTitle` (bg), `CommandKey` (bg), `ButtonSelected` (fg), `CheckboxItemSelected` (bg), `SwitchSelected` (fg), `TableRowSelected` (inverted fg/bg) |
| **warning** | #f9e2af | `WarningText` |
| **error** | #e78284 | `ErrorText` |
| **success** | #a6e3a1 | `SuccessText`, `SpinnerDoneColor` |

**Note**: Some styles use terminal default colors (no theme color):
- `Text`, `TextBold`, `TableHeader`, `CommandExample`, `CardContent` – terminal default
- These are not affected by theme changes. Command labels use `TextBold` (no separate `CommandLabel` style).

## Style Categories

FontGet uses a clear categorization system for different types of UI elements. Theme key names (e.g. `primary`, `secondary`) match the YAML keys in theme files.

### 1. PAGE STRUCTURE STYLES - Layout hierarchy and page elements
- **PageTitle** - Main page titles (theme: `primary` text, `base` background; overridable via `overrides.page_title`)

### 2. MESSAGE STYLES - User notifications and responses
- **Text** - Regular text content (terminal default - no theme color)
- **InfoText** - Informational messages (theme: `primary`)
- **SecondaryText** - Secondary informational text (theme: `secondary`)
- **QueryText** - User input values e.g. search queries (theme: `primary`)
- **WarningText** - Warning messages (theme: `warning`)
- **ErrorText** - Error messages (theme: `error`)
- **SuccessText** - Success messages (theme: `success`)
- **TextBold** - Bold text with terminal default color (use for command labels; no separate CommandLabel style)

### 3. DATA DISPLAY STYLES - Tables, lists, and data presentation
- **TableHeader** - Column headers (terminal default - no theme color)
- **TableSourceName** - Font/source names in tables (theme: `secondary`). Used by `RenderSourceNameWithTag()` for colored source names with tags. For plain source names use `Text.Render(name) + " " + RenderSourceTag(isBuiltIn)`.
- **TableRowSelected** - Selected table rows (theme: inverted `components`/`base` for contrast)

### 4. FORM STYLES - Input interfaces and forms
- **FormLabel** - Field labels (theme: `secondary`)
- **FormInput** - Input field content (theme: `components`)
- **FormPlaceholder** - Placeholder text (theme: `placeholders`)
- **FormReadOnly** - Read-only field content (theme: `placeholders`)

### 5. COMMAND STYLES - Interactive elements and controls
- **CommandKey** - Keyboard shortcuts (theme: `placeholders` text, `base` background; overridable via `overrides.command_keys`)
- **CommandExample** - Example commands (terminal default - no theme color). For button-like labels use **TextBold**.

### 6. CARD STYLES - Card components and layouts
- **CardTitle** - Card titles integrated into borders (theme: `primary` text, `base` background; overridable via `overrides.card`)
- **CardLabel** - Labels within cards (theme: `secondary`)
- **CardContent** - Use **Text** for regular content (terminal default)
- **CardBorder** - Card border (theme: `placeholders`)

### 7. BUTTON COMPONENT STYLES
- **ButtonNormal** - Unselected button text (theme: `components`; overridable via `overrides.button`)
- **ButtonSelected** - Selected button (inverted: `base` text on `components` background)

### 8. CHECKBOX COMPONENT STYLES
- **CheckboxUnchecked** - Unchecked checkbox (theme: `placeholders`; overridable via `overrides.checkbox`)
- **CheckboxChecked** - Checked checkbox (theme: `primary`; overridable via `overrides.checkbox`)
- **CheckboxItemSelected** - Selected checkbox row background (theme: `base`)
- **Cursor** - Cursor indicator for lists/checkboxes (theme: `primary`)

### 9. SWITCH COMPONENT STYLES
- **SwitchNormal** - Unselected switch option (theme: `components`; overridable via `overrides.switch`)
- **SwitchSelected** - Selected switch option (inverted: `base` text on `components` background)
- **SwitchSeparator** - Separator between options (theme: `placeholders`)

### 10. SPINNER COMPONENT
- **SpinnerColor** - Spinner animation color, hex string (theme: `primary`; overridable via `overrides.spinner`)
- **SpinnerDoneColor** - Done checkmark color, hex string (theme: `success`; overridable via `overrides.spinner`)

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
- **Card titles** → Integrated into top border with `primary` color and `base` background
- **Card labels** → `secondary` color (e.g. "Name:", "ID:", "Category:")
- **Card content** → Terminal default color for values
- **Card borders** → `placeholders` color with rounded corners

## Color Hierarchy

### Primary Visual Elements
1. **Page Titles** - `primary` text on `base` background
2. **Card Titles** - `primary` text on `base` background, integrated into borders
3. **Font Names & Labels** - `secondary` for prominent content, form labels, and card labels
4. **Warning Messages** - `warning` for warnings and skipped status
5. **Status Words** - `success` / `warning` / `error` via `SuccessText`, `WarningText`, `ErrorText`
6. **Primary Text** - Terminal default for compatibility

### Status Colors
- **Success** - Use `SuccessText` (theme: `success`)
- **Warning** - Use `WarningText` (theme: `warning`)
- **Error** - Use `ErrorText` (theme: `error`)
- **Info** - Use `InfoText` (theme: `primary`)

**Note:** The old `Feedback*` style names (`FeedbackText`, `FeedbackInfo`, etc.) are deprecated. Use `Text`, `InfoText`, `WarningText`, `ErrorText`, `SuccessText` in new code.

### Background Usage
- **Page titles, card titles, command keys** - `base` background
- **Selected rows, checkbox items** - `base` background
- **Selected buttons/switches** - `components` background (inverted)
- **Regular content** - No background (terminal default)

## Creating Custom Themes

To create a custom theme:

1. **Copy the default theme** from `internal/ui/themes/catppuccin.yaml` or `internal/templates/dark-theme-template.yaml` as a starting point.
2. **Save it** to `~/.fontget/themes/your-theme.yaml` (or `your-theme-theme.yaml`).
3. **Edit the colors** under the single `colors:` block using the key names below. Optionally add `overrides:` for component-specific colors.
4. **Update config.yaml** to use your theme:
   ```yaml
   theme:
     name: "your-theme"
     mode: "auto"  # or "dark" or "light"
   ```

### Theme Validation

Themes must include all **required** color keys under `colors:`:

- **Base colors:** `primary`, `secondary`, `components`, `placeholders`, `base`
- **Status colors:** `warning`, `error`, `success`

**Optional** `overrides` (each defaults to the base color above if omitted):

- `page_title` (text, background)
- `button` (foreground, background)
- `switch` (foreground, background)
- `checkbox` (unchecked, checked)
- `card` (title_text, title_background, label, border)
- `command_keys` (text, background)
- `table` (header, row, selected)
- `spinner` (normal, done)
- `progress_bar` (start, finish)

If a theme is missing required keys or fails validation, FontGet falls back to the default Catppuccin theme.

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
Use the shared table functions and column constants in `internal/ui/tables.go` for consistent formatting:
```go
// For font search/add/remove tables
fmt.Printf("%s\n", ui.TableHeader.Render(ui.GetSearchTableHeader()))
fmt.Printf("%s\n", ui.GetTableSeparator())

// For custom tables, use the column constants (e.g. ui.TableColName, ui.TableColID, ui.TableColSource)
fmt.Printf("%-*s %-*s %-*s\n", 
    ui.TableColName, "Name",
    ui.TableColID, "ID", 
    ui.TableColSource, "Source")
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
9. **Theme keys** - Use theme color keys (`primary`, `secondary`, etc.) in theme YAML; avoid hardcoding hex in code
10. **Terminal Defaults** - Use terminal default colors for base text to ensure compatibility
11. **Source Name Styling** - Use `RenderSourceNameWithTag()` for colored names, or combine `Text` with `RenderSourceTag()` for plain names

## Migration Notes

If you're updating existing code:

- **Don't hardcode colors** - Use theme-aware styles from the `ui` package
- **Don't reference specific hex codes** - Colors come from theme files (`primary`, `secondary`, etc.)
- **Use named styles** - e.g. `InfoText`, `SuccessText` instead of raw colors
- **Test both modes** - Ensure your code works in both dark and light themes
- **Check contrast** - Verify readability in both theme modes
