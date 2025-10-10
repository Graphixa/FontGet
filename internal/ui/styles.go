package ui

import "github.com/charmbracelet/lipgloss"

// FontGet Styles - Centralized styling based on Catppuccin Mocha palette
// This package provides consistent styling across all FontGet commands
// Reference: https://catppuccin.com/palette/

// ============================================================================
// PAGE STRUCTURE STYLES - Layout hierarchy and page elements
// ============================================================================
var (
	// PageTitle - Main page titles and headers
	PageTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#cba6f7")). // Mauve
			Background(lipgloss.Color("#313244")). // Surface 0
			Padding(0, 1)

	// PageSubtitle - Section subtitles and secondary headers
	PageSubtitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#6c7086")) // Overlay 0 - Grayish for subtitles

	// ReportTitle - Status report and data report titles
	ReportTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.NoColor{}) // No color

	// ContentText - Regular text content (uses terminal default for compatibility)
	ContentText = lipgloss.NewStyle().
			Foreground(lipgloss.NoColor{}) // No color - uses terminal default

	// ContentLabel - Bold label with no color (inline labels like paths)
	ContentLabel = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.NoColor{})

	// ContentHighlight - Highlighted content (like font names)
	ContentHighlight = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#f9e2af")). // Yellow
				Bold(true)
)

// ============================================================================
// USER FEEDBACK STYLES - Interactive responses and notifications
// ============================================================================
var (
	// FeedbackInfo - Informational messages (like "Multiple fonts found")
	FeedbackInfo = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#cba6f7")) // Mauve

	// FeedbackText - Supporting informational text
	FeedbackText = lipgloss.NewStyle().
			Foreground(lipgloss.NoColor{}) // No color

	// FeedbackWarning - Warning messages
	FeedbackWarning = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f9e2af")) // Yellow

	// FeedbackError - Error messages
	FeedbackError = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#e78284")) // Red from frappe

	// FeedbackSuccess - Success messages
	FeedbackSuccess = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a6e3a1")) // Green

)

// ============================================================================
// DATA DISPLAY STYLES - Tables, lists, and data presentation
// ============================================================================
var (
	// TableHeader - Column headers in tables
	TableHeader = lipgloss.NewStyle().
			Foreground(lipgloss.NoColor{}). // No color - uses terminal default
			Bold(true)

	// TableSourceName - Font names in search/add results
	TableSourceName = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f5c2e7")) // Pink

	// TableRow - Regular table rows (uses terminal default)
	TableRow = lipgloss.NewStyle().
			Foreground(lipgloss.NoColor{}) // No color - uses terminal default

	// TableSelectedRow - Selected table rows
	TableSelectedRow = lipgloss.NewStyle().
				Background(lipgloss.Color("#313244")) // Surface 0
)

// ============================================================================
// FORM STYLES - Input interfaces and forms
// ============================================================================
var (
	// FormLabel - Field labels (Name:, URL:, etc.)
	FormLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f5c2e7")). // Pink
			Bold(true)

	// FormInput - Input field content
	FormInput = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#4c4f69", Dark: "#cdd6f4"}) // Text

	// FormPlaceholder - Placeholder text
	FormPlaceholder = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7f849c")) // Overlay 1

	// FormReadOnly - Read-only field content
	FormReadOnly = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6c7086")) // Overlay 0
)

// ============================================================================
// COMMAND STYLES - Interactive elements and controls
// ============================================================================
var (
	// CommandKey - Keyboard shortcuts (Enter, Esc)
	CommandKey = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#bac2de")). // Subtext 1
			Background(lipgloss.Color("#313244")). // Surface 0
			Padding(0, 1)

	// CommandLabel - Button-like labels (Move, Submit, Cancel)
	CommandLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#bac2de")). // Subtext 1
			Bold(true)

	// CommandExample - Example commands
	CommandExample = lipgloss.NewStyle().
			Foreground(lipgloss.NoColor{}) // No color - uses terminal default

)

// ============================================================================
// CARD STYLES - Card components and layouts
// ============================================================================
var (
	// CardTitle - Card titles integrated into borders
	CardTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#cba6f7")). // Mauve - matches FeedbackInfo
			Background(lipgloss.Color("#313244")). // Surface 0
			Padding(0, 1)

	// CardLabel - Labels within cards (License:, URL:, etc.)
	CardLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f5c2e7")). // Pink - matches FormLabel
			Bold(true)

	// CardContent - Regular content within cards
	CardContent = lipgloss.NewStyle().
			Foreground(lipgloss.NoColor{}) // Terminal default - matches FeedbackText

	// CardBorder - Card border styling
	CardBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7f849c")) // Overlay 1

	// CardContainer - Container for cards with proper spacing
	CardContainer = lipgloss.NewStyle().
			Padding(0, 0).
			Margin(0, 0)
)

// ============================================================================
// PROGRESS BAR STYLES - Animated progress indicators
// ============================================================================
var (
	// ProgressBarHeader - Progress bar header text
	ProgressBarHeader = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#9399b2")) // Overlay 2

	// ProgressBarText - Progress counter text (1/5, 2/5, etc.)
	ProgressBarText = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a6adc8")) // Subtext 1

	// ProgressBarContainer - Container for the progress bar
	ProgressBarContainer = lipgloss.NewStyle().
				Padding(0, 2) // Add some padding around the progress bar
)

// GetProgressBarGradient returns the gradient colors for the progress bar
func GetProgressBarGradient() (string, string) {
	return "#cba6f7", "#eba0ac" // Mauve to Peach
}

// ============================================================================
// UTILITY FUNCTIONS - Common rendering patterns
// ============================================================================

// RenderSourceNameWithTag renders a source name with its type tag
func RenderSourceNameWithTag(name string, isBuiltIn bool) string {
	baseName := TableSourceName.Render(name)
	var tag string
	if isBuiltIn {
		tag = lipgloss.NewStyle().Foreground(lipgloss.Color("#9399b2")).Render("[Built-in]") // Subtext 0
	} else {
		tag = lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4")).Render("[Custom]") // Text
	}
	return baseName + " " + tag
}

// RenderKeyWithDescription renders a keyboard shortcut with description
func RenderKeyWithDescription(key, description string) string {
	return CommandKey.Render(key) + " " + CommandLabel.Render(description)
}

// RenderError renders an error message
func RenderError(message string) string {
	return FeedbackError.Render("Error: " + message)
}

// RenderSuccess renders a success message
func RenderSuccess(message string) string {
	return FeedbackSuccess.Render(message)
}

// RenderWarning renders a warning message
func RenderWarning(message string) string {
	return FeedbackWarning.Render(message)
}

// RenderInfo renders an info message
func RenderInfo(message string) string {
	return FeedbackInfo.Render(message)
}
