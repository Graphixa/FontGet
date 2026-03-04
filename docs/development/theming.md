# FontGet Theming System

## Overview

FontGet uses a YAML-based theme system that allows you to customize the colors and appearance of the application. Themes are stored in the `~/.fontget/themes/` directory and can be selected via the `config.yaml` file.

## Theme File Structure

Theme files are YAML files that define a single set of semantic colors. Each theme has a name, an optional `style` (e.g. `dark` or `light`), and a `colors` block that maps semantic keys to UI styles.

### Basic Structure

```yaml
theme_name: "My Theme"
style: dark   # optional: "dark" or "light"

colors:
  # Required base colors
  primary: "#cba6f7"       # Main accent – InfoText, PageTitle, CardTitle, Cursor, Spinner, QueryText
  secondary: "#94e2d5"     # Secondary accent – TableSourceName, FormLabel, CardLabel
  components: "#cdd6f4"    # Interactive elements – Button, Switch, FormInput
  placeholders: "#7f849c"  # Muted elements – borders, placeholders, CheckboxUnchecked
  base: "#313244"          # Backgrounds – selected states (CommandKey uses fixed ANSI 256 grey)

  # Required status colors
  warning: "#f9e2af"
  error: "#e78284"
  success: "#a6e3a1"

  # Optional component overrides (defaults to base colors above)
  overrides:
    page_title:
      text: "#cba6f7"
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
      label: "#94e2d5"
      border: "#7f849c"
    command_keys:
      text: "#7f849c"
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

## Semantic Color Keys

The theme system uses semantic color keys that are mapped to multiple UI components. See `docs/development/style-guide.md` for a full table of which UI styles use each key (`primary`, `secondary`, `components`, `placeholders`, `base`, `warning`, `error`, `success`, and optional `overrides`).

## Creating a New Theme

### Step 1: Create Theme File

1. Create a new YAML file in `~/.fontget/themes/` directory
2. Name it `{theme-name}.yaml` (e.g., `my-theme.yaml`)
3. Use the structure shown above

### Step 2: Define Colors

Choose colors for the `colors` section. You can reference existing themes:
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
  Name: ""                 # Empty string uses embedded default (catppuccin)
  Use256ColorSpace: false  # Downsample theme colors to ANSI 256 for terminals without true color
```

### Configuration Options

- **`Name`**: Theme file name without extension (e.g., `"gruvbox"` for `gruvbox.yaml`)
  - Empty string (`""`) uses the embedded default theme
  - If the theme file is not found, FontGet falls back to the embedded default
- **`Use256ColorSpace`**: When `true`, theme hex colors are downsampled to the nearest ANSI 256-color index before being applied. This is useful for terminals (e.g. Apple Terminal) that do not handle 24-bit true color well.

## Theme Validation

FontGet automatically validates theme files when they are loaded. The validation ensures that all required color keys are present in the theme file.

### Required Color Keys

All of the following keys must be present and non-empty under `colors`:

- `primary`
- `secondary`
- `components`
- `placeholders`
- `base`
- `warning`
- `error`
- `success`

### Validation Behavior

- **On theme load**: If a theme file is missing required keys, FontGet will fallback to the embedded default theme (Catppuccin)
- **Error handling**: Validation errors are handled gracefully - the application continues to work with the default theme

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
     Use256ColorSpace: false
   ```
3. Run any FontGet command

### Example: Creating a Custom Theme

1. Create `~/.fontget/themes/my-custom-theme.yaml` with the structure shown above.
2. Edit `~/.fontget/config.yaml`:
   ```yaml
   Theme:
     Name: "my-custom-theme"
     Use256ColorSpace: false
   ```

## Troubleshooting

### Theme Not Loading

- Check that the theme file exists in `~/.fontget/themes/`
- Verify the file name matches `Theme.Name` in config (without `.yaml` extension)
- Check YAML syntax is valid
- FontGet will fallback to embedded default if theme fails to load

### Colors Not Applied

- Ensure all required color keys are defined under `colors`
- Verify YAML indentation is correct (2 spaces)

## Future Enhancements

- Interactive theme switching command
- Theme preview functionality
- Theme validation on load
- Theme marketplace/sharing
- Live theme reloading (without restart)

