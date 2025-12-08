package components

import (
	"fmt"
	"strings"

	"fontget/internal/ui"

	"github.com/charmbracelet/lipgloss"
)

// PreviewModel represents the theme preview component
type PreviewModel struct {
	theme *ui.Theme
	mode  string
}

// NewPreviewModel creates a new preview model
func NewPreviewModel() *PreviewModel {
	return &PreviewModel{
		mode: "dark",
	}
}

// LoadTheme loads a theme for preview
func (m *PreviewModel) LoadTheme(themeName string, mode string) error {
	// Try to load theme
	tm := ui.GetThemeManager()
	theme, err := tm.LoadTheme(themeName, mode)
	if err != nil {
		// Try embedded theme
		theme, err = ui.LoadEmbeddedTheme(themeName, mode)
		if err != nil {
			return fmt.Errorf("failed to load theme: %w", err)
		}
	}

	m.theme = theme
	m.mode = mode
	return nil
}

// View renders the preview
func (m *PreviewModel) View(width int) string {
	if m.theme == nil {
		return ui.Text.Render("Loading preview...")
	}

	// Create temporary styles using theme colors
	colors := &m.theme.FontGetTheme
	previewStyles := createPreviewStyles(colors)

	var lines []string

	// Page Title
	lines = append(lines, previewStyles.PageTitle.Render("Page Title"))
	lines = append(lines, "") // Spacing

	// Card example (separated at bottom)
	cardTitle := previewStyles.CardTitle.Render("Card Title")
	cardLabel := previewStyles.CardLabel.Render("Label:")
	cardContent := previewStyles.CardContent.Render("Card content goes here")
	card := fmt.Sprintf("%s\n%s %s\n%s", cardTitle, cardLabel, cardContent, cardContent)
	lines = append(lines, "") // Spacing before card
	lines = append(lines, previewStyles.CardBorder.Render(card))

	// Buttons - use proper Button and Selected styles
	// Render with proper button format: "[  Text  ]"
	button1Rendered := previewStyles.ButtonNormal.Render("[  Button  ]")
	button2Rendered := previewStyles.ButtonSelected.Render("[  Selected  ]")
	lines = append(lines, fmt.Sprintf("%s %s", button1Rendered, button2Rendered))
	lines = append(lines, "") // Spacing

	// Status messages with spacing between each
	lines = append(lines, previewStyles.SuccessText.Render("✓ Success message"))
	lines = append(lines, "") // Spacing
	lines = append(lines, previewStyles.WarningText.Render("! Warning message"))
	lines = append(lines, "") // Spacing
	lines = append(lines, previewStyles.ErrorText.Render("✗ Error message"))

	content := strings.Join(lines, "\n")

	// Don't add another card border - the preview content should be inside the panel card
	return content
}

// previewStyles holds temporary styles for preview
type previewStyles struct {
	PageTitle      lipgloss.Style
	CardTitle      lipgloss.Style
	CardLabel      lipgloss.Style
	CardContent    lipgloss.Style
	CardBorder     lipgloss.Style
	ButtonNormal   lipgloss.Style
	ButtonSelected lipgloss.Style
	SuccessText    lipgloss.Style
	WarningText    lipgloss.Style
	ErrorText      lipgloss.Style
}

// createPreviewStyles creates temporary styles for preview using theme colors
func createPreviewStyles(colors *ui.ModeColors) previewStyles {
	return previewStyles{
		PageTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colors.PageTitle)).
			Background(lipgloss.Color(colors.GreyDark)).
			Padding(0, 1),

		CardTitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colors.Accent)).
			Background(lipgloss.Color(colors.GreyDark)).
			Bold(true).
			Padding(0, 1),

		CardLabel: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colors.Accent2)).
			Bold(true),

		CardContent: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colors.GreyLight)),

		CardBorder: lipgloss.NewStyle().
			BorderForeground(lipgloss.Color(colors.GreyMid)).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderTop(true).
			BorderBottom(true).
			BorderLeft(true).
			BorderRight(true).
			Padding(1),

		ButtonNormal: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colors.GreyLight)).
			Bold(true),

		ButtonSelected: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colors.GreyDark)).
			Background(lipgloss.Color(colors.GreyLight)).
			Bold(true),

		SuccessText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colors.Success)),

		WarningText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colors.Warning)),

		ErrorText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colors.Error)),
	}
}
