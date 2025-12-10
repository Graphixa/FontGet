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
	card := NewCard("Card Title", previewStyles.CardLabel.Render("Label:")+" "+previewStyles.CardContent.Render("Card content"))
	card.Width = width - 2 // allow a little breathing room
	card.VerticalPadding = 0
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
