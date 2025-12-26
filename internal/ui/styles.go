package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// FontGet Styles - Centralized styling with theme support
// This package provides consistent styling across all FontGet commands
// Styles are theme-aware and initialized from the theme system on startup
//
// Styles are organized by component to ensure each component has dedicated styles
// and prevent style reuse across different purposes.

// ============================================================================
// PAGE STRUCTURE - Layout hierarchy and page elements
// ============================================================================
var (
	// PageTitle - Main page titles and headers
	// Usage: Page titles, dialog titles, component titles
	// Example: ui.PageTitle.Render("Font Search Results")
	// Colors: Set by InitStyles() from theme
	PageTitle = lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1)
)

// ============================================================================
// MESSAGE STYLES - User notifications and responses
// ============================================================================
var (
	// Text - Regular text content
	// Usage: Regular text, descriptions, supporting text
	// Example: ui.Text.Render("This is regular text")
	Text = lipgloss.NewStyle().
		Foreground(lipgloss.NoColor{}) // No color

	// InfoText - Informational messages (like "Multiple fonts found")
	// Usage: Informational messages, tips, hints
	// Example: ui.InfoText.Render("Multiple fonts found")
	// Colors: Set by InitStyles() from theme
	InfoText = lipgloss.NewStyle()

	// SecondaryText - Secondary informational text using accent2 color
	// Usage: Secondary informational messages, headings, labels that need accent2 color
	// Example: ui.SecondaryText.Render("Secondary information")
	// Colors: Set by InitStyles() from theme
	SecondaryText = lipgloss.NewStyle()

	// WarningText - Warning messages
	// Usage: Warning messages, cautions
	// Example: ui.WarningText.Render("Warning: This action cannot be undone")
	// Colors: Set by InitStyles() from theme
	WarningText = lipgloss.NewStyle()

	// ErrorText - Error messages
	// Usage: Error messages, failures
	// Example: ui.ErrorText.Render("Error: Operation failed")
	// Colors: Set by InitStyles() from theme
	ErrorText = lipgloss.NewStyle()

	// SuccessText - Success messages
	// Usage: Success messages, confirmations
	// Example: ui.SuccessText.Render("Success: Operation completed")
	// Colors: Set by InitStyles() from theme
	SuccessText = lipgloss.NewStyle()

	// QueryText - User input values (search queries, filter terms, user-provided values)
	// Usage: Search queries, filter terms, user-provided input values
	// Example: ui.QueryText.Render("roboto")
	// Colors: Set by InitStyles() from theme (uses Primary color)
	QueryText = lipgloss.NewStyle()

	// TextBold - Bold text with terminal default color
	// Usage: Bold labels, report titles, emphasized text without color
	// Example: ui.TextBold.Render("Status Report")
	TextBold = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.NoColor{}) // No color - uses terminal default

)

// ============================================================================
// TABLE COMPONENT - Table-specific styles
// ============================================================================
var (
	// TableHeader - Column headers in tables
	// Usage: Table column headers
	// Example: ui.TableHeader.Render("Name")
	TableHeader = lipgloss.NewStyle().
			Foreground(lipgloss.NoColor{}). // No color - uses terminal default
			Bold(true)

	// TableSourceName - Font names and source names in tables
	// Usage: Font names, source names, identifiers in tables
	// Example: ui.TableSourceName.Render("Roboto")
	// Colors: Set by InitStyles() from theme
	TableSourceName = lipgloss.NewStyle()

	// TableRowSelected - Selected table rows
	// Usage: Highlighting selected rows in tables
	// Example: ui.TableRowSelected.Render(rowText)
	// Colors: Set by InitStyles() from theme
	TableRowSelected = lipgloss.NewStyle()
)

// ============================================================================
// FORM COMPONENT - Form input styles
// ============================================================================
var (
	// FormLabel - Field labels (Name:, URL:, etc.)
	// Usage: Form field labels
	// Example: ui.FormLabel.Render("Name:")
	// Colors: Set by InitStyles() from theme
	FormLabel = lipgloss.NewStyle().
			Bold(true)

	// FormInput - Input field content
	// Usage: Text input field content
	// Example: ui.FormInput.Render("user input")
	// Colors: Set by InitStyles() from theme (uses components color)
	FormInput = lipgloss.NewStyle()

	// FormPlaceholder - Placeholder text
	// Usage: Placeholder text in input fields
	// Example: ui.FormPlaceholder.Render("Enter font name...")
	// Colors: Set by InitStyles() from theme
	FormPlaceholder = lipgloss.NewStyle()

	// FormReadOnly - Read-only field content
	// Usage: Read-only field content
	// Example: ui.FormReadOnly.Render("Read-only value")
	// Colors: Set by InitStyles() from theme
	FormReadOnly = lipgloss.NewStyle()
)

// ============================================================================
// CARD COMPONENT - Card component styles
// ============================================================================
var (
	// CardTitle - Card titles integrated into borders
	// Usage: Card titles
	// Example: ui.CardTitle.Render("Font Details")
	// Colors: Set by InitStyles() from theme
	CardTitle = lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1)

	// CardLabel - Labels within cards (License:, URL:, etc.)
	// Usage: Labels within card components
	// Example: ui.CardLabel.Render("License:")
	// Colors: Set by InitStyles() from theme
	CardLabel = lipgloss.NewStyle()
	// Note: Card content uses Text instead of a separate style

	// CardBorder - Card border styling
	// Usage: Card border styling
	// Example: ui.CardBorder.Render("Card content here")
	// Colors: Set by InitStyles() from theme
	CardBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderTop(true).
			BorderBottom(true).
			BorderLeft(true).
			BorderRight(true).
			Padding(1)
)

// ============================================================================
// COMMAND STYLES - Interactive elements and controls
// ============================================================================
var (
	// CommandKey - Keyboard shortcuts (Enter, Esc)
	// Usage: Keyboard shortcut indicators
	// Example: ui.CommandKey.Render("Enter")
	// Colors: Set by InitStyles() from theme
	CommandKey = lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1)
	// Note: Command labels use TextBold instead of a separate style
	// Note: Command examples use Text instead of a separate style
)

// ============================================================================
// BUTTON COMPONENT - Button styles (for future TUI components)
// ============================================================================
var (
	// ButtonNormal - Unselected button text
	// Usage: Unselected button text
	// Example: ui.ButtonNormal.Render("  OK  ")
	// Colors: Set by InitStyles() from theme
	ButtonNormal = lipgloss.NewStyle().
			Bold(true)

	// ButtonSelected - Selected button (background + contrasting foreground)
	// Usage: Selected button text
	// Example: ui.ButtonSelected.Render("  OK  ")
	// Colors: Set by InitStyles() from theme
	ButtonSelected = lipgloss.NewStyle().
			Bold(true)

	// ButtonGroup - Container style for button groups (spacing)
	// Usage: Spacing between buttons in a group
	// Note: This is a spacing style, not a visual style
	ButtonGroup = lipgloss.NewStyle().
			Margin(0, 1) // Space between buttons
)

// ============================================================================
// CHECKBOX COMPONENT - Checkbox styles (for future TUI components)
// ============================================================================
var (
	// CheckboxChecked - Checked checkbox style ([x])
	// Usage: Checked checkbox indicator
	// Example: ui.CheckboxChecked.Render("[x]")
	// Colors: Set by InitStyles() from theme
	CheckboxChecked = lipgloss.NewStyle().
			Bold(true)

	// CheckboxUnchecked - Unchecked checkbox style ([ ])
	// Usage: Unchecked checkbox indicator
	// Example: ui.CheckboxUnchecked.Render("[ ]")
	// Colors: Set by InitStyles() from theme (empty = no color/terminal default)
	CheckboxUnchecked = lipgloss.NewStyle()

	// CheckboxItemSelected - Selected checkbox item row style
	// Usage: Highlighting selected checkbox items
	// Example: ui.CheckboxItemSelected.Render("  [x] Item name")
	// Colors: Set by InitStyles() from theme
	CheckboxItemSelected = lipgloss.NewStyle()
	// Note: Normal checkbox items use Text instead of a separate style

	// CheckboxCursor - Cursor indicator style (> )
	// Usage: Cursor indicator for selected checkbox item
	// Example: ui.CheckboxCursor.Render("> ")
	// Colors: Set by InitStyles() from theme
	CheckboxCursor = lipgloss.NewStyle().
			Bold(true)
)

// ============================================================================
// SWITCH COMPONENT - Switch/toggle styles (for future TUI components)
// ============================================================================
var (
	// SwitchNormal - Unselected switch option
	// Usage: Switch option when not selected
	// Example: ui.SwitchNormal.Render("  Enable  ")
	// Colors: Set by InitStyles() from theme
	SwitchNormal = lipgloss.NewStyle().
			Bold(true)

	// SwitchSelected - Selected switch option (background + contrasting foreground)
	// Usage: Switch option when selected
	// Example: ui.SwitchSelected.Render("  Enable  ")
	// Colors: Set by InitStyles() from theme
	SwitchSelected = lipgloss.NewStyle().
			Bold(true)

	// SwitchSeparator - Separator style (|)
	// Usage: Separator between switch options
	// Example: ui.SwitchSeparator.Render("|")
	// Colors: Set by InitStyles() from theme
	SwitchSeparator = lipgloss.NewStyle()
)

// ============================================================================
// PROGRESS BAR COMPONENT - Progress indicator styles
// ============================================================================

// GetProgressBarGradient returns the gradient colors for the progress bar
// Uses theme colors if available, otherwise returns defaults
// Default order: primary → secondary (flipped from old secondary → primary)
func GetProgressBarGradient() (string, string) {
	colors := GetCurrentColors()
	if colors != nil {
		// Check for override first
		if colors.Overrides.ProgressBar.Start != "" && colors.Overrides.ProgressBar.Finish != "" {
			return colors.Overrides.ProgressBar.Start, colors.Overrides.ProgressBar.Finish
		}
		// Default: primary → secondary
		return colors.Primary, colors.Secondary
	}
	return "#cba6f7", "#eba0ac" // Default: Mauve to Peach
}

// ============================================================================
// SPINNER COMPONENT - Loading indicator styles
// ============================================================================
var (
	// SpinnerColor - Color for the spinner animation
	// Usage: Spinner animation color
	// Example: Used in spinner component initialization
	// Colors: Set by InitStyles() from theme
	SpinnerColor = "#cba6f7" // Default, will be set by InitStyles()

	// SpinnerDoneColor - Color for the done checkmark symbol
	// Usage: Done checkmark color
	// Example: Used in spinner component initialization
	// Colors: Set by InitStyles() from theme
	SpinnerDoneColor = "#a6e3a1" // Default, will be set by InitStyles()

	// Note: PinColorMap has been removed in favor of dynamic RGB-based color matching
	// The hexToPinColor function now automatically finds the closest ANSI color
	// for any hex color using Euclidean distance calculation
)

// ============================================================================
// UTILITY FUNCTIONS - Common rendering patterns
// ============================================================================

// RenderSourceTag renders just the source type tag ([Built-in] or [Custom])
// Usage: Display source type tags separately from source names
// Example: ui.RenderSourceTag(true) // Returns "[Built-in]"
// For system theme, uses terminal default colors (no color)
func RenderSourceTag(isBuiltIn bool) string {
	colors := GetCurrentColors()
	var tagStyle lipgloss.Style
	if colors != nil && colors.Placeholders != "" {
		// Use theme color (placeholders for both, or components for custom)
		if isBuiltIn {
			tagStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Placeholders))
		} else {
			tagStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Components))
		}
	} else {
		// System theme - use terminal default (no color)
		tagStyle = lipgloss.NewStyle().Foreground(lipgloss.NoColor{})
	}

	if isBuiltIn {
		return tagStyle.Render("[Built-in]")
	}
	return tagStyle.Render("[Custom]")
}

// RenderSourceNameWithTag renders a source name with its type tag using TableSourceName color
// Usage: Display source names with [Built-in] or [Custom] tags (colored name)
// Example: ui.RenderSourceNameWithTag("Google Fonts", true)
// Note: Uses TableSourceName style for the name
func RenderSourceNameWithTag(name string, isBuiltIn bool) string {
	baseName := TableSourceName.Render(name)
	tag := RenderSourceTag(isBuiltIn)
	return baseName + " " + tag
}

// RenderKeyWithDescription renders a keyboard shortcut with description
// Usage: Display keyboard shortcuts with descriptions
// Example: ui.RenderKeyWithDescription("Y", "Yes")
func RenderKeyWithDescription(key, description string) string {
	return CommandKey.Render(key) + " " + TextBold.Render(description)
}

// RenderError renders an error message with consistent prefix
// Usage: Display error messages with consistent formatting
// Example: ui.RenderError("Operation failed") // Returns "Error: Operation failed"
func RenderError(message string) string {
	return ErrorText.Render("Error: " + message)
}

// RenderSuccess renders a success message with consistent prefix
// Usage: Display success messages with consistent formatting
// Example: ui.RenderSuccess("Operation completed") // Returns "Success: Operation completed"
func RenderSuccess(message string) string {
	return SuccessText.Render("Success: " + message)
}

// RenderWarning renders a warning message
// Usage: Display warning messages with consistent formatting
// Example: ui.RenderWarning("This action cannot be undone")
func RenderWarning(message string) string {
	return WarningText.Render(message)
}

// RenderInfo renders an info message with consistent prefix
// Usage: Display informational messages with consistent formatting
// Example: ui.RenderInfo("Multiple fonts found") // Returns "Info: Multiple fonts found"
func RenderInfo(message string) string {
	return InfoText.Render("Info: " + message)
}

// ============================================================================
// THEME-AWARE STYLE INITIALIZATION
// ============================================================================

// getColorOrNoColor returns a TerminalColor that lipgloss can use
// If color string is empty, returns lipgloss.NoColor{} to use terminal defaults
// Otherwise returns lipgloss.Color(color)
// This is used for the "system" theme which has empty color strings
func getColorOrNoColor(color string) lipgloss.TerminalColor {
	if color == "" {
		return lipgloss.NoColor{}
	}
	return lipgloss.Color(color)
}

// resolveColor returns the override color if set, otherwise returns the default color
func resolveColor(override string, defaultColor string) string {
	if override != "" {
		return override
	}
	return defaultColor
}

// applySystemThemeFallback applies system theme fallback colors (white bg, black text)
// when colors are empty (system theme)
func applySystemThemeFallback(style lipgloss.Style, fg, bg string) lipgloss.Style {
	if bg == "" {
		style = style.Background(lipgloss.Color("#ffffff"))
	}
	if fg == "" {
		style = style.Foreground(lipgloss.Color("#000000"))
	}
	return style
}

// InitStyles initializes all styles based on the current theme
// This should be called after InitThemeManager() during application startup
// For "system" theme (empty colors), uses lipgloss.NoColor{} to respect terminal defaults
func InitStyles() error {
	colors := GetCurrentColors()
	if colors == nil {
		// If no theme is loaded, use defaults (styles are already initialized with defaults)
		return nil
	}

	// Resolve override colors with defaults
	pageTitleText := resolveColor(colors.Overrides.PageTitle.Text, colors.Primary)
	pageTitleBg := resolveColor(colors.Overrides.PageTitle.Background, colors.Base)

	buttonFg := resolveColor(colors.Overrides.Button.Foreground, colors.Components)
	buttonBg := resolveColor(colors.Overrides.Button.Background, colors.Base)

	switchFg := resolveColor(colors.Overrides.Switch.Foreground, colors.Components)
	switchBg := resolveColor(colors.Overrides.Switch.Background, colors.Base)

	checkboxChecked := resolveColor(colors.Overrides.Checkbox.Checked, colors.Primary)
	checkboxUnchecked := resolveColor(colors.Overrides.Checkbox.Unchecked, colors.Placeholders)

	cardTitleText := resolveColor(colors.Overrides.Card.TitleText, colors.Primary)
	cardTitleBg := resolveColor(colors.Overrides.Card.TitleBackground, colors.Base)
	cardLabel := resolveColor(colors.Overrides.Card.Label, colors.Secondary)
	cardBorder := resolveColor(colors.Overrides.Card.Border, colors.Placeholders)

	commandKeyText := resolveColor(colors.Overrides.CommandKeys.Text, colors.Placeholders)
	commandKeyBg := resolveColor(colors.Overrides.CommandKeys.Background, colors.Base)

	spinnerNormal := resolveColor(colors.Overrides.Spinner.Normal, colors.Primary)
	spinnerDone := resolveColor(colors.Overrides.Spinner.Done, colors.Success)

	// PAGE STRUCTURE
	PageTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(getColorOrNoColor(pageTitleText)).
		Background(getColorOrNoColor(pageTitleBg)).
		Padding(0, 1)

	// MESSAGE STYLES
	InfoText = lipgloss.NewStyle().
		Foreground(getColorOrNoColor(colors.Primary))

	SecondaryText = lipgloss.NewStyle().
		Foreground(getColorOrNoColor(colors.Secondary))

	// QueryText - User input values (search queries, filter terms, user-provided values)
	// Uses Primary color to distinguish user input from data/content
	QueryText = lipgloss.NewStyle().
		Foreground(getColorOrNoColor(colors.Primary))

	WarningText = lipgloss.NewStyle().
		Foreground(getColorOrNoColor(colors.Warning))

	ErrorText = lipgloss.NewStyle().
		Foreground(getColorOrNoColor(colors.Error))

	SuccessText = lipgloss.NewStyle().
		Foreground(getColorOrNoColor(colors.Success))

	// TABLE COMPONENT
	TableSourceName = lipgloss.NewStyle().
		Foreground(getColorOrNoColor(colors.Secondary))

	TableRowSelected = lipgloss.NewStyle().
		Background(getColorOrNoColor(colors.Base))

	// FORM COMPONENT
	FormLabel = lipgloss.NewStyle().
		Foreground(getColorOrNoColor(colors.Secondary)).
		Bold(true)

	FormInput = lipgloss.NewStyle().
		Foreground(getColorOrNoColor(colors.Components))

	FormPlaceholder = lipgloss.NewStyle().
		Foreground(getColorOrNoColor(colors.Placeholders))

	FormReadOnly = lipgloss.NewStyle().
		Foreground(getColorOrNoColor(colors.Placeholders))

	// COMMAND COMPONENT
	CommandKey = lipgloss.NewStyle().
		Foreground(getColorOrNoColor(commandKeyText)).
		Background(getColorOrNoColor(commandKeyBg)).
		Bold(true).
		Padding(0, 1)
	CommandKey = applySystemThemeFallback(CommandKey, commandKeyText, commandKeyBg)

	// CARD COMPONENT
	CardTitle = lipgloss.NewStyle().
		Foreground(getColorOrNoColor(cardTitleText)).
		Background(getColorOrNoColor(cardTitleBg)).
		Bold(true).
		Padding(0, 1)

	CardLabel = lipgloss.NewStyle().
		Foreground(getColorOrNoColor(cardLabel))

	CardBorder = lipgloss.NewStyle().
		BorderForeground(getColorOrNoColor(cardBorder)).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderTop(true).
		BorderBottom(true).
		BorderLeft(true).
		BorderRight(true).
		Padding(1)

	// BUTTON COMPONENT
	ButtonNormal = lipgloss.NewStyle().
		Foreground(getColorOrNoColor(buttonFg)).
		Bold(true)

	ButtonSelected = lipgloss.NewStyle().
		Foreground(getColorOrNoColor(buttonBg)). // Inverted: base text
		Background(getColorOrNoColor(buttonFg)). // Inverted: components background
		Bold(true)
	ButtonSelected = applySystemThemeFallback(ButtonSelected, buttonBg, buttonFg)

	// CHECKBOX COMPONENT
	CheckboxChecked = lipgloss.NewStyle().
		Foreground(getColorOrNoColor(checkboxChecked)).
		Bold(true)

	CheckboxUnchecked = lipgloss.NewStyle().
		Foreground(getColorOrNoColor(checkboxUnchecked))

	CheckboxItemSelected = lipgloss.NewStyle().
		Background(getColorOrNoColor(colors.Base))

	CheckboxCursor = lipgloss.NewStyle().
		Foreground(getColorOrNoColor(colors.Primary)).
		Bold(true)

	// SWITCH COMPONENT (unified - no left/right distinction)
	SwitchNormal = lipgloss.NewStyle().
		Foreground(getColorOrNoColor(switchFg)).
		Bold(true)

	SwitchSelected = lipgloss.NewStyle().
		Foreground(getColorOrNoColor(switchBg)). // Inverted: base text
		Background(getColorOrNoColor(switchFg)). // Inverted: components background
		Bold(true)
	SwitchSelected = applySystemThemeFallback(SwitchSelected, switchBg, switchFg)

	SwitchSeparator = lipgloss.NewStyle().
		Foreground(getColorOrNoColor(switchFg))

	// SPINNER COMPONENT
	SpinnerColor = spinnerNormal
	SpinnerDoneColor = spinnerDone

	return nil
}
