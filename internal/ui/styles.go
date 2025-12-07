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

	// PageSubtitle - Section subtitles and secondary headers
	// Usage: Section subtitles, secondary headers
	// Example: ui.PageSubtitle.Render("Refreshing font data cache...")
	// Colors: Set by InitStyles() from theme
	PageSubtitle = lipgloss.NewStyle().
			Bold(true)
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
	// Colors: Hardcoded adaptive (light/dark mode) - not theme-aware
	FormInput = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#4c4f69", Dark: "#cdd6f4"})

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

	// CardContent - Regular content within cards
	// Usage: Regular content within card components
	// Example: ui.CardContent.Render("Card content here")
	CardContent = lipgloss.NewStyle().
			Foreground(lipgloss.NoColor{}) // Terminal default - matches Text

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

	// CommandLabel - Button-like labels (Move, Submit, Cancel)
	// Usage: Command labels, button-like text
	// Example: ui.CommandLabel.Render("Submit")
	// Colors: Set by InitStyles() from theme
	CommandLabel = lipgloss.NewStyle().
			Bold(true)

	// CommandExample - Example commands
	// Usage: Example command text
	// Example: ui.CommandExample.Render("fontget add google.roboto")
	CommandExample = lipgloss.NewStyle().
			Foreground(lipgloss.NoColor{}) // No color - uses terminal default
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
	// Colors: Hardcoded (no color) - not theme-aware
	CheckboxUnchecked = lipgloss.NewStyle().
				Foreground(lipgloss.NoColor{})

	// CheckboxItemSelected - Selected checkbox item row style
	// Usage: Highlighting selected checkbox items
	// Example: ui.CheckboxItemSelected.Render("  [x] Item name")
	// Colors: Set by InitStyles() from theme
	CheckboxItemSelected = lipgloss.NewStyle()

	// CheckboxItemNormal - Normal checkbox item row style
	// Usage: Normal checkbox item rows
	// Example: ui.CheckboxItemNormal.Render("  [ ] Item name")
	// Colors: Hardcoded (no color) - not theme-aware
	CheckboxItemNormal = lipgloss.NewStyle().
				Foreground(lipgloss.NoColor{})

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
	// SwitchContainer - Container style for the switch (brackets and separator)
	// Usage: Container for switch component
	// Note: This may be used for spacing/alignment
	SwitchContainer = lipgloss.NewStyle().
			Foreground(lipgloss.NoColor{}) // No color

	// SwitchLeftSelected - Left option when selected (background + contrasting foreground)
	// Usage: Left option (e.g., "Enable") when selected
	// Example: ui.SwitchLeftSelected.Render("  Enable  ")
	// Colors: Set by InitStyles() from theme
	SwitchLeftSelected = lipgloss.NewStyle().
				Bold(true)

	// SwitchLeftNormal - Left option when not selected
	// Usage: Left option (e.g., "Enable") when not selected
	// Example: ui.SwitchLeftNormal.Render("  Enable  ")
	// Colors: Set by InitStyles() from theme
	SwitchLeftNormal = lipgloss.NewStyle().
				Bold(true)

	// SwitchRightSelected - Right option when selected (background + contrasting foreground)
	// Usage: Right option (e.g., "Disable") when selected
	// Example: ui.SwitchRightSelected.Render("  Disable  ")
	// Colors: Set by InitStyles() from theme
	SwitchRightSelected = lipgloss.NewStyle().
				Bold(true)

	// SwitchRightNormal - Right option when not selected
	// Usage: Right option (e.g., "Disable") when not selected
	// Example: ui.SwitchRightNormal.Render("  Disable  ")
	// Colors: Set by InitStyles() from theme
	SwitchRightNormal = lipgloss.NewStyle().
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
func GetProgressBarGradient() (string, string) {
	colors := GetCurrentColors()
	if colors != nil {
		return colors.ProgressBarGradient.ColorStart, colors.ProgressBarGradient.ColorEnd
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

	// PinColorMap - Maps hex color strings to pin package color names
	// Used for converting hex colors to pin.Color constants
	// The pin package uses its own color constants and doesn't accept hex strings directly
	PinColorMap = map[string]string{
		"#a6e3a1": "green",   // Green - matches SuccessText, SpinnerDoneColor
		"#cba6f7": "magenta", // Mauve - matches PageTitle, InfoText, SpinnerColor
		"#b4befe": "blue",    // Blue - available for future use
		"#a6adc8": "cyan",    // Cyan - available for future use
	}
)

// ============================================================================
// UTILITY FUNCTIONS - Common rendering patterns
// ============================================================================

// RenderSourceTag renders just the source type tag ([Built-in] or [Custom])
// Usage: Display source type tags separately from source names
// Example: ui.RenderSourceTag(true) // Returns "[Built-in]"
func RenderSourceTag(isBuiltIn bool) string {
	if isBuiltIn {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#9399b2")).Render("[Built-in]") // Subtext 0
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4")).Render("[Custom]") // Text
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
	return CommandKey.Render(key) + " " + CommandLabel.Render(description)
}

// RenderError renders an error message
// Usage: Display error messages with consistent formatting
// Example: ui.RenderError("Operation failed")
func RenderError(message string) string {
	return ErrorText.Render("Error: " + message)
}

// RenderSuccess renders a success message
// Usage: Display success messages with consistent formatting
// Example: ui.RenderSuccess("Operation completed")
func RenderSuccess(message string) string {
	return SuccessText.Render(message)
}

// RenderWarning renders a warning message
// Usage: Display warning messages with consistent formatting
// Example: ui.RenderWarning("This action cannot be undone")
func RenderWarning(message string) string {
	return WarningText.Render(message)
}

// RenderInfo renders an info message
// Usage: Display informational messages with consistent formatting
// Example: ui.RenderInfo("Multiple fonts found")
func RenderInfo(message string) string {
	return InfoText.Render(message)
}

// ============================================================================
// THEME-AWARE STYLE INITIALIZATION
// ============================================================================

// InitStyles initializes all styles based on the current theme
// This should be called after InitThemeManager() during application startup
func InitStyles() error {
	colors := GetCurrentColors()
	if colors == nil {
		// If no theme is loaded, use defaults (styles are already initialized with defaults)
		return nil
	}

	// Update styles that use theme colors
	// Note: Hardcoded styles (Text, TextBold, TableHeader, FormInput, CommandExample, CardContent, CheckboxUnchecked) are not updated

	// PAGE STRUCTURE
	PageTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(colors.PageTitle)).
		Background(lipgloss.Color(colors.GreyDark)).
		Padding(0, 1)

	PageSubtitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(colors.PageSubtitle))

	// MESSAGE STYLES
	InfoText = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.Accent))

	SecondaryText = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.Accent2))

	WarningText = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.Warning))

	ErrorText = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.Error))

	SuccessText = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.Success))

	// TABLE COMPONENT
	TableSourceName = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.Accent2))

	TableRowSelected = lipgloss.NewStyle().
		Background(lipgloss.Color(colors.GreyDark))

	// FORM COMPONENT
	FormLabel = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.Accent2)).
		Bold(true)

	FormPlaceholder = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.GreyMid))

	FormReadOnly = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.GreyMid))

	// COMMAND COMPONENT
	CommandKey = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.GreyMid)).
		Background(lipgloss.Color(colors.GreyDark)).
		Bold(true).
		Padding(0, 1)

	CommandLabel = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.GreyMid)).
		Bold(true)

	// CARD COMPONENT
	CardTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.Accent)).
		Background(lipgloss.Color(colors.GreyDark)).
		Bold(true).
		Padding(0, 1)

	CardLabel = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.Accent2))

	CardBorder = lipgloss.NewStyle().
		BorderForeground(lipgloss.Color(colors.GreyMid)).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderTop(true).
		BorderBottom(true).
		BorderLeft(true).
		BorderRight(true).
		Padding(1)

	// BUTTON COMPONENT
	ButtonNormal = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.GreyLight)).
		Bold(true)

	ButtonSelected = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.GreyDark)).  // Inverted: dark text
		Background(lipgloss.Color(colors.GreyLight)). // Inverted: light background
		Bold(true)

	// CHECKBOX COMPONENT
	CheckboxChecked = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.Accent2)). // Use accent color instead of success
		Bold(true)

	CheckboxItemSelected = lipgloss.NewStyle().
		Background(lipgloss.Color(colors.GreyDark))

	CheckboxCursor = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.Accent2)).
		Bold(true)

	// SWITCH COMPONENT
	SwitchLeftNormal = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.GreyLight)).
		Bold(true)

	SwitchLeftSelected = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.GreyDark)).  // Inverted: dark text
		Background(lipgloss.Color(colors.GreyLight)). // Inverted: light background
		Bold(true)

	SwitchRightNormal = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.GreyLight)).
		Bold(true)

	SwitchRightSelected = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.GreyDark)).  // Inverted: dark text
		Background(lipgloss.Color(colors.GreyLight)). // Inverted: light background
		Bold(true)

	SwitchSeparator = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.GreyMid))

	// SPINNER COMPONENT
	SpinnerColor = colors.Accent
	SpinnerDoneColor = colors.Success

	return nil
}
