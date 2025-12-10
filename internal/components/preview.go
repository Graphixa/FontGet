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
	mode              string
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
		mode:              "dark",
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

	// Checkboxes side-by-side to save space
	checkboxChecked := previewStyles.CheckboxChecked.Render("[x]")
	checkboxUnchecked := previewStyles.CheckboxUnchecked.Render("[ ]")
	lines = append(lines, fmt.Sprintf("%s Checked  %s Unchecked", checkboxChecked, checkboxUnchecked))
	lines = append(lines, "") // Spacing

	// Switch
	switchComp := NewSwitchWithLabels("On", "Off", true)
	switchComp.Value = true
	switchRendered := switchComp.Render()
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
	spinnerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Accent))
	spinnerRendered := spinnerStyle.Render(spinnerChar + " Loading")
	progressPercent := int(m.progressPercent * 100)
	progressBar := m.renderStaticProgressBar(progressPercent, colors)
	lines = append(lines, fmt.Sprintf("%s  %s", spinnerRendered, progressBar))
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

// renderStaticProgressBar creates a static progress bar for preview
// Uses a simplified version without duplicating helper functions from progress_bar.go
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
		gradientColor := interpolateHexColorSimple(startColor, endColor, ratio)
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(gradientColor))
		barBuilder.WriteString(style.Render("█"))
	}

	// Empty portion
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colors.GreyMid))
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
