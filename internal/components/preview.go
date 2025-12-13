package components

import (
	"fmt"
	"strings"
	"time"

	"fontget/internal/ui"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PreviewModel represents the theme preview component
type PreviewModel struct {
	theme             *ui.Theme
	spinner           spinner.Model
	progressPercent   float64
	progressDirection int // 1 for increasing, -1 for decreasing
}

// NewPreviewModel creates a new preview model
func NewPreviewModel() *PreviewModel {
	// Create spinner
	spin := spinner.New()
	spin.Spinner = spinner.Dot

	return &PreviewModel{
		spinner:           spin,
		progressPercent:   0.0,
		progressDirection: 1, // Start increasing
	}
}

// Init initializes the preview model with tick commands
func (m *PreviewModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		previewTickCmd(),
	)
}

// Update handles messages for animation
func (m *PreviewModel) Update(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return cmd, true // Request redraw

	case previewTickMsg:
		// Update progress bar (cycle 0-100% over ~4 seconds)
		// Each tick is ~100ms, so 40 ticks = 4 seconds
		m.progressPercent += 0.025 * float64(m.progressDirection) // 2.5% per tick

		if m.progressPercent >= 1.0 {
			m.progressPercent = 1.0
			m.progressDirection = -1 // Start decreasing
		} else if m.progressPercent <= 0.0 {
			m.progressPercent = 0.0
			m.progressDirection = 1 // Start increasing
		}

		return previewTickCmd(), true // Request redraw and schedule next tick
	}

	return nil, false
}

// previewTickMsg is sent periodically to update animations
type previewTickMsg time.Time

// previewTickCmd returns a command that sends a tick message after ~100ms
func previewTickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return previewTickMsg(t)
	})
}

// LoadTheme loads a theme for preview
func (m *PreviewModel) LoadTheme(themeName string) error {
	// Try to load theme
	tm := ui.GetThemeManager()
	theme, err := tm.LoadTheme(themeName)
	if err != nil {
		// Try embedded theme
		theme, err = ui.LoadEmbeddedTheme(themeName)
		if err != nil {
			return fmt.Errorf("failed to load theme: %w", err)
		}
	}

	m.theme = theme
	return nil
}

// View renders the preview
func (m *PreviewModel) View(width int) string {
	if m.theme == nil {
		return ui.Text.Render("Loading preview...")
	}

	// Create temporary styles using theme colors
	colors := &m.theme.Colors
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

	// Checkboxes side-by-side to save space
	checkboxChecked := previewStyles.CheckboxChecked.Render("[x]")
	checkboxUnchecked := previewStyles.CheckboxUnchecked.Render("[ ]")
	lines = append(lines, fmt.Sprintf("%s Checked  %s Unchecked", checkboxChecked, checkboxUnchecked))
	lines = append(lines, "") // Spacing

	// Switch - render manually with preview styles (not global ui styles)
	leftPadded := "  On  "
	rightPadded := "  Off  "
	var leftStyled, rightStyled string
	// Left is selected (Value = true)
	leftStyled = previewStyles.ButtonSelected.Render(leftPadded)
	rightStyled = previewStyles.ButtonNormal.Render(rightPadded)
	separator := previewStyles.Text.Render("|")
	switchRendered := fmt.Sprintf("[%s%s%s]", leftStyled, separator, rightStyled)
	lines = append(lines, switchRendered)
	lines = append(lines, "") // Spacing

	// Status messages - all on one line to save space
	successMsg := previewStyles.SuccessText.Render("✓ Success")
	warningMsg := previewStyles.WarningText.Render("! Warning")
	errorMsg := previewStyles.ErrorText.Render("✗ Error")
	statusLine := fmt.Sprintf("%s  %s  %s", successMsg, warningMsg, errorMsg)
	lines = append(lines, statusLine)
	lines = append(lines, "") // Spacing

	// Spinner and Progress bar side-by-side to save space
	spinnerChar := strings.TrimSpace(m.spinner.View())
	// Use preview theme's accent color (not global ui.SpinnerColor which doesn't update)
	// For system theme (empty color), use NoColor to respect terminal defaults
	var spinnerColor lipgloss.TerminalColor
	if colors.Accent == "" {
		spinnerColor = lipgloss.NoColor{}
	} else {
		spinnerColor = lipgloss.Color(colors.Accent)
	}
	spinnerStyle := lipgloss.NewStyle().Foreground(spinnerColor)
	// Only color the spinner character, not the "Loading" text
	spinnerRendered := spinnerStyle.Render(spinnerChar) + " Loading"
	progressPercent := int(m.progressPercent * 100)
	progressBar := m.renderStaticProgressBar(progressPercent, colors)
	lines = append(lines, fmt.Sprintf("%s  %s", spinnerRendered, progressBar))
	lines = append(lines, "") // Spacing

	// Card example using card renderer for consistent title-in-border
	// Make it look like fontget info with multiple sections
	// Labels are colored, but content values use terminal default (no color)
	cardContent := strings.Builder{}
	cardContent.WriteString(previewStyles.CardLabel.Render("Name:") + " " + "Example Font")
	cardContent.WriteString("\n")
	cardContent.WriteString(previewStyles.CardLabel.Render("ID:") + " " + "example.font")
	cardContent.WriteString("\n")
	cardContent.WriteString("\n") // Empty line for spacing
	cardContent.WriteString(previewStyles.CardLabel.Render("Category:") + " " + "Sans Serif")
	cardContent.WriteString("\n")
	cardContent.WriteString(previewStyles.CardLabel.Render("Tags:") + " " + "modern, clean")

	// Render card with preview theme's CardTitle style (not global ui.CardTitle)
	cardWidth := width - 2
	if cardWidth < 20 {
		cardWidth = 20 // Minimum width for card to render properly
	}
	cardRendered := renderCardWithPreviewStyle("Card Title", cardContent.String(), cardWidth, previewStyles, colors)
	lines = append(lines, cardRendered)

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

// createPreviewStyles creates temporary styles for preview using theme colors
// For system theme (empty colors), uses getColorOrNoColor() to respect terminal defaults
func createPreviewStyles(colors *ui.ModeColors) previewStyles {
	return previewStyles{
		PageTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(getColorOrNoColor(colors.PageTitle)).
			Background(getColorOrNoColor(colors.GreyDark)).
			Padding(0, 1),

		PageSubtitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(getColorOrNoColor(colors.PageSubtitle)),

		Text: lipgloss.NewStyle().
			Foreground(getColorOrNoColor(colors.GreyLight)),

		InfoText: lipgloss.NewStyle().
			Foreground(getColorOrNoColor(colors.Accent)),

		CardTitle: lipgloss.NewStyle().
			Foreground(getColorOrNoColor(colors.Accent)).
			Background(getColorOrNoColor(colors.GreyDark)).
			Bold(true).
			Padding(0, 1),

		CardLabel: lipgloss.NewStyle().
			Foreground(getColorOrNoColor(colors.Accent2)).
			Bold(true),

		CardContent: lipgloss.NewStyle().
			Foreground(lipgloss.NoColor{}), // Card content values use terminal default (no color)

		CardBorder: lipgloss.NewStyle().
			BorderForeground(getColorOrNoColor(colors.GreyMid)).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderTop(true).
			BorderBottom(true).
			BorderLeft(true).
			BorderRight(true).
			Padding(1),

		ButtonNormal: lipgloss.NewStyle().
			Foreground(getColorOrNoColor(colors.GreyLight)).
			Bold(true),

		ButtonSelected: func() lipgloss.Style {
			style := lipgloss.NewStyle().
				Foreground(getColorOrNoColor(colors.GreyDark)).
				Background(getColorOrNoColor(colors.GreyLight)).
				Bold(true)
			// For system theme, ensure background is visible (use white) and text is readable (use black)
			if colors.GreyLight == "" {
				style = style.Background(lipgloss.Color("#ffffff"))
			}
			// For system theme, ensure text is dark and readable on white background
			if colors.GreyDark == "" {
				style = style.Foreground(lipgloss.Color("#000000"))
			}
			return style
		}(),

		CheckboxChecked: lipgloss.NewStyle().
			Foreground(getColorOrNoColor(colors.Accent2)).
			Bold(true),

		CheckboxUnchecked: lipgloss.NewStyle().
			Foreground(getColorOrNoColor(colors.GreyMid)),

		SuccessText: lipgloss.NewStyle().
			Foreground(getColorOrNoColor(colors.Success)),

		WarningText: lipgloss.NewStyle().
			Foreground(getColorOrNoColor(colors.Warning)),

		ErrorText: lipgloss.NewStyle().
			Foreground(getColorOrNoColor(colors.Error)),
	}
}

// renderStaticProgressBar creates a static progress bar for preview
// Uses a simplified version without duplicating helper functions from progress_bar.go
// For system theme (empty colors), uses terminal defaults
func (m *PreviewModel) renderStaticProgressBar(percent int, colors *ui.ModeColors) string {
	barWidth := 20 // Compact width for preview
	filled := int(float64(barWidth) * float64(percent) / 100.0)
	empty := barWidth - filled

	// Get gradient colors
	startColor := colors.ProgressBarGradient.ColorStart
	endColor := colors.ProgressBarGradient.ColorEnd

	// Build the progress bar manually with gradient colors
	var barBuilder strings.Builder

	// Filled portion with gradient - simplified interpolation
	for i := 0; i < filled; i++ {
		// Calculate color interpolation for gradient effect
		var ratio float64
		if filled > 1 {
			ratio = float64(i) / float64(filled-1)
		} else {
			ratio = 0.0
		}
		if ratio > 1.0 {
			ratio = 1.0
		}
		if ratio < 0.0 {
			ratio = 0.0
		}

		// Interpolate color between start and end
		// For system theme (empty colors), use terminal default
		var style lipgloss.Style
		if startColor == "" || endColor == "" {
			style = lipgloss.NewStyle().Foreground(lipgloss.NoColor{})
		} else {
			gradientColor := interpolateHexColorSimple(startColor, endColor, ratio)
			style = lipgloss.NewStyle().Foreground(lipgloss.Color(gradientColor))
		}
		barBuilder.WriteString(style.Render("█"))
	}

	// Empty portion - for system theme, use terminal default
	var emptyStyle lipgloss.Style
	if colors.GreyMid == "" {
		emptyStyle = lipgloss.NewStyle().Foreground(lipgloss.NoColor{})
	} else {
		emptyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colors.GreyMid))
	}
	for i := 0; i < empty; i++ {
		barBuilder.WriteString(emptyStyle.Render("░"))
	}

	barVisual := barBuilder.String()
	percentText := fmt.Sprintf("%d%%", percent)

	return fmt.Sprintf("[%s] %s", barVisual, percentText)
}

// interpolateHexColorSimple interpolates between two hex colors (simplified version)
func interpolateHexColorSimple(start, end string, ratio float64) string {
	// Parse hex colors to RGB
	startRGB := hexToRGBSimple(start)
	endRGB := hexToRGBSimple(end)

	// Interpolate each component
	r := int(float64(startRGB[0])*(1-ratio) + float64(endRGB[0])*ratio)
	g := int(float64(startRGB[1])*(1-ratio) + float64(endRGB[1])*ratio)
	b := int(float64(startRGB[2])*(1-ratio) + float64(endRGB[2])*ratio)

	// Convert back to hex
	return rgbToHexSimple(r, g, b)
}

// hexToRGBSimple converts a hex color string to RGB values
func hexToRGBSimple(hex string) [3]int {
	// Remove # if present
	hex = strings.TrimPrefix(hex, "#")

	// Parse hex string
	var r, g, b int
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)

	return [3]int{r, g, b}
}

// rgbToHexSimple converts RGB values to a hex color string
func rgbToHexSimple(r, g, b int) string {
	// Clamp values to valid range
	if r < 0 {
		r = 0
	}
	if r > 255 {
		r = 255
	}
	if g < 0 {
		g = 0
	}
	if g > 255 {
		g = 255
	}
	if b < 0 {
		b = 0
	}
	if b > 255 {
		b = 255
	}

	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// renderCardWithPreviewStyle renders a card using preview theme styles instead of global ui styles
// This ensures the card title uses the preview theme's colors
func renderCardWithPreviewStyle(title, content string, width int, styles previewStyles, colors *ui.ModeColors) string {
	// Create styled title using preview theme
	styledTitle := styles.CardTitle.Render(title)

	// Create content with proper padding using preview border style
	contentStyle := styles.CardBorder.Padding(1, 1)
	contentRendered := contentStyle.Width(width).Render(content)

	// Split content into lines
	lines := strings.Split(contentRendered, "\n")
	if len(lines) < 2 {
		return contentRendered
	}

	// Calculate title length (without ANSI codes)
	plainTitleLength := len(title) + 2 // +2 for CardTitle padding

	// Calculate remaining width for right side
	rightWidth := width - 1 - 1 - plainTitleLength - 1 - 1
	if rightWidth < 0 {
		rightWidth = 0
	}

	// Get border color from preview theme
	var borderColor lipgloss.TerminalColor
	if colors.GreyMid != "" {
		borderColor = lipgloss.Color(colors.GreyMid)
	} else {
		borderColor = lipgloss.NoColor{}
	}

	// Create top border with integrated title
	topLeft := "╭"
	topRight := strings.Repeat("─", rightWidth) + "╮"
	styledTopLeft := lipgloss.NewStyle().Foreground(borderColor).Render(topLeft)
	styledTopRight := lipgloss.NewStyle().Foreground(borderColor).Render(topRight)
	dashStyle := lipgloss.NewStyle().Foreground(borderColor)
	styledDashes := dashStyle.Render("─")
	styledTitleSection := styledDashes + " " + styledTitle + " " + styledDashes
	titleLine := styledTopLeft + styledTitleSection + styledTopRight

	// Build result
	var result strings.Builder
	result.WriteString(titleLine)
	result.WriteString("\n")
	for i := 1; i < len(lines); i++ {
		result.WriteString(lines[i])
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}
