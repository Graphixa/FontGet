# FontGet Theming System

## Overview

FontGet uses a YAML-based theme system that allows you to customize the colors and appearance of the application. Themes are stored in the `~/.fontget/themes/` directory and can be selected via the `config.yaml` file.

## Theme File Structure

Theme files are YAML files that define colors for both dark and light modes. The structure follows a semantic color system where color keys are mapped to multiple UI styles.

### Basic Structure

```yaml
fontget_theme:
  dark_mode:
    accent: "#cba6f7"
    accent2: "#f5c2e7"
    warning: "#f9e2af"
    error: "#e78284"
    success: "#a6e3a1"
    page_title: "#cba6f7"
    page_subtitle: "#7f849c"
    grey_light: "#cdd6f4"
    grey_mid: "#7f849c"
    grey_dark: "#313244"
    progress_bar_gradient:
      color_start: "#cba6f7"
      color_end: "#eba0ac"

  light_mode:
    # ... same structure for light mode
```

## Semantic Color Keys

The theme system uses semantic color keys that are mapped to multiple UI components:

- **`accent`**: Primary accent color (InfoText, PageTitle, CardTitle, CheckboxCursor, SpinnerColor)
- **`accent2`**: Secondary accent color (TableSourceName, FormLabel, CardLabel)
- **`warning`**: Warning messages (WarningText)
- **`error`**: Error messages (ErrorText)
- **`success`**: Success messages (SuccessText)
- **`page_title`**: Page title foreground color
- **`page_subtitle`**: Page subtitle color
- **`grey_light`**: Light text color (Text, unselected items)
- **`grey_mid`**: Medium grey (borders, placeholders, subtitles, separators)
- **`grey_dark`**: Dark background color (PageTitle background, TableRowSelected background)
- **`progress_bar_gradient`**: Progress bar gradient colors (color_start, color_end)

## Creating a New Theme

### Step 1: Create Theme File

1. Create a new YAML file in `~/.fontget/themes/` directory
2. Name it `{theme-name}.yaml` (e.g., `my-theme.yaml`)
3. Use the structure shown above

### Step 2: Define Colors

Choose colors for both `dark_mode` and `light_mode` sections. You can reference existing themes:
- `internal/ui/themes/catppuccin.yaml` - Default Catppuccin theme
- `internal/ui/themes/gruvbox.yaml` - Gruvbox theme

### Step 3: Configure Theme

Edit `~/.fontget/config.yaml` and add/update the Theme section:

```yaml
Theme:
  Name: "my-theme"  # Theme file name without .yaml extension
  Mode: "dark"      # "dark" or "light"
```

### Step 4: Restart FontGet

The theme will be loaded on the next FontGet command execution.

## Theme File Location

- **User themes**: `~/.fontget/themes/{theme-name}.yaml`
- **Default theme**: Embedded in binary (Catppuccin) - always available as fallback

## Available Themes

### Catppuccin (Default)

The default embedded theme using Catppuccin Mocha (dark) and Latte (light) color palettes.

**File**: Embedded in binary (reference: `internal/ui/themes/catppuccin.yaml`)

### Gruvbox

A retro groove color scheme based on Gruvbox.

**File**: `internal/ui/themes/gruvbox.yaml` (copy to `~/.fontget/themes/gruvbox.yaml` to use)

## Theme Configuration

The theme is configured in `~/.fontget/config.yaml`:

```yaml
Theme:
  Name: ""      # Empty string uses embedded default (catppuccin)
  Mode: "auto"  # "auto" (detect from terminal), "dark", or "light"
```

### Configuration Options

- **`Name`**: Theme file name without extension (e.g., "gruvbox" for `gruvbox.yaml`)
  - Empty string (`""`) uses the embedded default theme
  - If theme file is not found, falls back to embedded default
- **`Mode`**: Theme mode - `"auto"`, `"dark"`, or `"light"` (defaults to `"auto"`)
  - **`"auto"`**: Automatically detects terminal theme using `termenv.HasDarkBackground()`
    - Checks `FONTGET_THEME_MODE` environment variable first (allows manual override)
    - Falls back to terminal detection if env var is not set or set to "auto"
  - **`"dark"`**: Force dark mode
  - **`"light"`**: Force light mode

### Automatic Theme Detection

FontGet uses the `termenv` package (already included via lipgloss) to automatically detect your terminal's theme:

- **How it works**: Queries the terminal for its background color to determine if it's dark or light
- **Environment override**: Set `FONTGET_THEME_MODE=dark` or `FONTGET_THEME_MODE=light` to override detection
- **Fallback**: If detection fails, defaults to dark mode

#### Windows Terminal Limitations

**Important**: Automatic theme detection may not work reliably on Windows Terminal. Windows Terminal doesn't properly support the ANSI escape sequences used to query background color, so `termenv.HasDarkBackground()` may return incorrect results.

**Workarounds for Windows Terminal**:

1. **Use environment variable** (recommended):
   ```powershell
   $env:FONTGET_THEME_MODE="light"  # or "dark"
   ```

2. **Set in config.yaml**:
   ```yaml
   Theme:
     Mode: "light"  # or "dark" - explicitly set instead of "auto"
   ```

3. **Per-session override**:
   ```powershell
   $env:FONTGET_THEME_MODE="light"; fontget search roboto
   ```

The environment variable takes precedence over config file settings, making it easy to override when needed.

## Theme Validation

FontGet automatically validates theme files when they are loaded. The validation ensures that all required color keys are present in both `dark_mode` and `light_mode` sections.

### Required Color Keys

All of the following keys must be present and non-empty in both modes:

- `accent`
- `accent2`
- `warning`
- `error`
- `success`
- `page_title`
- `page_subtitle`
- `grey_light`
- `grey_mid`
- `grey_dark`
- `progress_bar_gradient.color_start`
- `progress_bar_gradient.color_end`

### Validation Behavior

- **On theme load**: If a theme file is missing required keys, FontGet will fallback to the embedded default theme (Catppuccin)
- **Error handling**: Validation errors are handled gracefully - the application continues to work with the default theme
- **Both modes validated**: Both `dark_mode` and `light_mode` must have all required keys

## Switching Themes

*(Future feature: Interactive theme switching via command or TUI)*

Currently, themes can be switched by:
1. Editing `~/.fontget/config.yaml`
2. Setting `Theme.Name` to the desired theme name
3. Running any FontGet command (theme loads on startup)

## Examples

### Example: Using Gruvbox Theme

1. Copy `internal/ui/themes/gruvbox.yaml` to `~/.fontget/themes/gruvbox.yaml`
2. Edit `~/.fontget/config.yaml`:
   ```yaml
   Theme:
     Name: "gruvbox"
     Mode: "dark"
   ```
3. Run any FontGet command

### Example: Creating a Custom Theme

1. Create `~/.fontget/themes/my-custom-theme.yaml`:
   ```yaml
   fontget_theme:
     dark_mode:
       accent: "#ff6b6b"
       accent2: "#4ecdc4"
       # ... define all color keys
     light_mode:
       # ... define light mode colors
   ```
2. Edit `~/.fontget/config.yaml`:
   ```yaml
   Theme:
     Name: "my-custom-theme"
     Mode: "dark"
   ```

## Troubleshooting

### Theme Not Loading

- Check that the theme file exists in `~/.fontget/themes/`
- Verify the file name matches `Theme.Name` in config (without `.yaml` extension)
- Check YAML syntax is valid
- FontGet will fallback to embedded default if theme fails to load

### Colors Not Applied

- Ensure all required color keys are defined in both `dark_mode` and `light_mode`
- Check that `Theme.Mode` in config matches the mode you want to use
- Verify YAML indentation is correct (2 spaces)

## Future Enhancements

- Interactive theme switching command
- Theme preview functionality
- Theme validation on load
- Theme marketplace/sharing
- Live theme reloading (without restart)

