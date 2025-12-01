# Color Palette Audit

**Date:** Generated during styles cleanup
**Purpose:** Extract all color hex codes and map them to Catppuccin Mocha palette

## Color Usage in styles.go

### Primary Colors

| Color Name | Hex Code | Usage in styles.go | Catppuccin Name |
|------------|----------|-------------------|-----------------|
| Mauve | `#cba6f7` | PageTitle, OperationTitle, FeedbackInfo, CardTitle, SpinnerColor | Mauve |
| Pink | `#f5c2e7` | TableSourceName, FormLabel, CardLabel | Pink |
| Yellow | `#f9e2af` | FeedbackWarning, ContentHighlight | Yellow |
| Green | `#a6e3a1` | FeedbackSuccess, SpinnerDoneColor | Green |
| Red | `#e78284` | FeedbackError | Red (from Frappe) |

### Surface Colors

| Color Name | Hex Code | Usage in styles.go | Catppuccin Name |
|------------|----------|-------------------|-----------------|
| Surface 0 | `#313244` | PageTitle background, CommandKey background, CardTitle background, TableSelectedRow background | Surface 0 |

### Text Colors

| Color Name | Hex Code | Usage in styles.go | Catppuccin Name |
|------------|----------|-------------------|-----------------|
| Text (Light) | `#4c4f69` | FormInput (light mode) | Text (Frappe Light) |
| Text (Dark) | `#cdd6f4` | FormInput (dark mode) | Text |
| Subtext 1 | `#bac2de` | CommandKey, CommandLabel | Subtext 1 |
| Subtext 0 | `#9399b2` | ProgressBarHeader, Built-in tags | Subtext 0 |
| Overlay 0 | `#6c7086` | PageSubtitle, FormReadOnly | Overlay 0 |
| Overlay 1 | `#7f849c` | FormPlaceholder, CardBorder | Overlay 1 |
| Overlay 2 | `#9399b2` | ProgressBarHeader | Overlay 2 |

### Additional Colors

| Color Name | Hex Code | Usage in styles.go | Catppuccin Name |
|------------|----------|-------------------|-----------------|
| Peach | `#eba0ac` | GetProgressBarGradient (end color) | Maroon |
| Blue | `#b4befe` | PinColorMap (future use) | Lavender |
| Cyan | `#a6adc8` | PinColorMap (future use) | Subtext 1 variant |

### Adaptive Colors

| Color Name | Usage | Notes |
|------------|-------|-------|
| `lipgloss.NoColor{}` | ContentText, FeedbackText, TableHeader, TableRow, CommandExample, CardContent, ReportTitle | Uses terminal default |
| `lipgloss.AdaptiveColor{Light: "#4c4f69", Dark: "#cdd6f4"}` | FormInput | Adapts to terminal theme |

## Color Constants Summary

**Total unique colors:** 15 hex colors + 2 adaptive/no-color patterns

**Most used colors:**
1. `#cba6f7` (Mauve) - 5 usages
2. `#313244` (Surface 0) - 4 usages
3. `#f5c2e7` (Pink) - 3 usages
4. `#bac2de` (Subtext 1) - 2 usages
5. `#9399b2` (Subtext 0/Overlay 2) - 2 usages

## Theme File Structure

For the theme file format, we'll need to map each style to its color properties:

```json
{
  "name": "Catppuccin Mocha",
  "version": "1.0.0",
  "colors": {
    "PageTitle": {
      "foreground": "#cba6f7",
      "background": "#313244",
      "bold": true
    },
    "OperationTitle": {
      "foreground": "#cba6f7",
      "bold": true
    },
    "PageSubtitle": {
      "foreground": "#6c7086",
      "bold": true
    },
    "ReportTitle": {
      "bold": true
    },
    "ContentText": {},
    "ContentLabel": {
      "bold": true
    },
    "ContentHighlight": {
      "foreground": "#f9e2af",
      "bold": true
    },
    "FeedbackInfo": {
      "foreground": "#cba6f7"
    },
    "FeedbackText": {},
    "FeedbackWarning": {
      "foreground": "#f9e2af"
    },
    "FeedbackError": {
      "foreground": "#e78284"
    },
    "FeedbackSuccess": {
      "foreground": "#a6e3a1"
    },
    "TableHeader": {
      "bold": true
    },
    "TableSourceName": {
      "foreground": "#f5c2e7"
    },
    "TableRow": {},
    "TableRowSelected": {
      "background": "#313244"
    },
    "FormLabel": {
      "foreground": "#f5c2e7",
      "bold": true
    },
    "FormInput": {
      "foreground_light": "#4c4f69",
      "foreground_dark": "#cdd6f4"
    },
    "FormPlaceholder": {
      "foreground": "#7f849c"
    },
    "FormReadOnly": {
      "foreground": "#6c7086"
    },
    "CommandKey": {
      "foreground": "#bac2de",
      "background": "#313244",
      "bold": true
    },
    "CommandLabel": {
      "foreground": "#bac2de",
      "bold": true
    },
    "CommandExample": {},
    "CardTitle": {
      "foreground": "#cba6f7",
      "background": "#313244",
      "bold": true
    },
    "CardLabel": {
      "foreground": "#f5c2e7"
    },
    "CardContent": {},
    "CardBorder": {
      "border_foreground": "#7f849c"
    },
    "CardContainer": {},
    "ProgressBarHeader": {
      "foreground": "#9399b2",
      "bold": true
    },
    "ProgressBarText": {
      "foreground": "#a6adc8"
    },
    "ProgressBarContainer": {},
    "SpinnerColor": "#cba6f7",
    "SpinnerDoneColor": "#a6e3a1",
    "ProgressBarGradientStart": "#cba6f7",
    "ProgressBarGradientEnd": "#eba0ac"
  }
}
```

## Notes

- Colors that use `lipgloss.NoColor{}` should have empty color definitions in theme files
- Adaptive colors need both light and dark variants
- Border colors are separate from foreground/background
- Some styles are just color constants (SpinnerColor, SpinnerDoneColor) rather than full styles

