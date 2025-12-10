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

	// Info Text (no emoji), then regular text beneath
	lines = append(lines, previewStyles.InfoText.Render("Info message"))
	lines = append(lines, previewStyles.Text.Render("Regular text content"))
	lines = append(lines, "") // Spacing

	// Buttons - use proper Button and Selected styles
	button1Rendered := previewStyles.ButtonNormal.Render("[  Button  ]")
	button2Rendered := previewStyles.ButtonSelected.Render("[  Selected  ]")
	lines = append(lines, fmt.Sprintf("%s %s", button1Rendered, button2Rendered))
	lines = append(lines, "") // Spacing

	// Checkboxes
	checkboxChecked := previewStyles.CheckboxChecked.Render("[x]")
	checkboxUnchecked := previewStyles.CheckboxUnchecked.Render("[ ]")
	lines = append(lines, fmt.Sprintf("%s Checked item", checkboxChecked))
	lines = append(lines, fmt.Sprintf("%s Unchecked item", checkboxUnchecked))
	lines = append(lines, "") // Spacing

	// Switch
	switchComp := NewSwitchWithLabels("On", "Off", true)
	switchComp.Value = true
	switchRendered := switchComp.Render()
	lines = append(lines, switchRendered)
	lines = append(lines, "") // Spacing

	// Status messages
	lines = append(lines, previewStyles.SuccessText.Render("✓ Success message"))
	lines = append(lines, previewStyles.WarningText.Render("! Warning message"))
	lines = append(lines, previewStyles.ErrorText.Render("✗ Error message"))
	lines = append(lines, "") // Spacing

	// Card example using card renderer for consistent title-in-border
	// Make it look like fontget info with multiple sections
	cardContent := strings.Builder{}
	cardContent.WriteString(previewStyles.CardLabel.Render("Name:") + " " + previewStyles.CardContent.Render("Example Font"))
	cardContent.WriteString("\n")
	cardContent.WriteString(previewStyles.CardLabel.Render("ID:") + " " + previewStyles.CardContent.Render("example.font"))
	cardContent.WriteString("\n")
	cardContent.WriteString("\n") // Empty line for spacing
	cardContent.WriteString(previewStyles.CardLabel.Render("Category:") + " " + previewStyles.CardContent.Render("Sans Serif"))
	cardContent.WriteString("\n")
	cardContent.WriteString(previewStyles.CardLabel.Render("Tags:") + " " + previewStyles.CardContent.Render("modern, clean"))

	card := NewCard("Card Title", cardContent.String())
	// Card width should fit within the available content width
	// The 'width' parameter is the content width available (already accounting for panel padding)
	// The card's Width is used for the content area, and the card adds its own border (2 chars: left + right)
	// So card.Width should be: available width - card border (2) = width - 2
	// But we also need to account for the card's horizontal padding (1 on each side = 2 total)
	// So the actual content width inside the card will be: card.Width - 2 (padding)
	// To fit properly: card.Width = width - 2 (border) - some margin for safety
	// Actually, looking at the card renderer, it uses c.Width for the content width constraint
	// and the border is built separately, so the total card width will be c.Width + 2 (borders)
	// To fit in 'width', we need: c.Width + 2 <= width, so c.Width <= width - 2
	card.Width = width - 2
	if card.Width < 20 {
		card.Width = 20 // Minimum width for card to render properly
	}
	card.VerticalPadding = 1
	card.HorizontalPadding = 1
	lines = append(lines, card.Render())

	content := strings.Join(lines, "\n")

	// Don't add another card border - the preview content should be inside the panel card
	return content
}

// previewStyles holds temporary styles for preview
type previewStyles struct {
	PageTitle         lipgloss.Style
	PageSubtitle      lipgloss.Style
	Text              lipgloss.Style
	InfoText          lipgloss.Style
	CardTitle         lipgloss.Style
	CardLabel         lipgloss.Style
	CardContent       lipgloss.Style
	CardBorder        lipgloss.Style
	ButtonNormal      lipgloss.Style
	ButtonSelected    lipgloss.Style
	CheckboxChecked   lipgloss.Style
	CheckboxUnchecked lipgloss.Style
	SuccessText       lipgloss.Style
	WarningText       lipgloss.Style
	ErrorText         lipgloss.Style
}

// createPreviewStyles creates temporary styles for preview using theme colors
func createPreviewStyles(colors *ui.ModeColors) previewStyles {
	return previewStyles{
		PageTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colors.PageTitle)).
			Background(lipgloss.Color(colors.GreyDark)).
			Padding(0, 1),

		PageSubtitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colors.PageSubtitle)),

		Text: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colors.GreyLight)),

		InfoText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colors.Accent)),

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

		CheckboxChecked: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colors.Accent2)).
			Bold(true),

		CheckboxUnchecked: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colors.GreyMid)),

		SuccessText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colors.Success)),

		WarningText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colors.Warning)),

		ErrorText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colors.Error)),
	}
}
