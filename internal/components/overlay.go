package components

import (
	"strings"

	"fontget/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Position represents a positioning option for overlays
type Position int

const (
	Top Position = iota
	Right
	Bottom
	Left
	Center
)

// ModalResult represents the result from a modal dialog
type ModalResult struct {
	Confirmed bool
	Cancelled bool
	Data      interface{}
}

// OverlayModel represents an overlay/modal that composites a foreground model onto a background
type OverlayModel struct {
	Foreground  tea.Model
	Background  tea.Model
	XPosition   Position
	YPosition   Position
	XOffset     int
	YOffset     int
	Width       int
	Height      int
	ShowBorder  bool // Whether to wrap foreground in a card-like border
	BorderWidth int  // Width of the border (0 = auto)
}

// OverlayOptions contains options for creating an overlay
type OverlayOptions struct {
	ShowBorder  bool // Whether to wrap foreground in a card-like border
	BorderWidth int  // Width of the border (0 = auto, will calculate from content)
}

// NewOverlay creates a new overlay model
func NewOverlay(foreground, background tea.Model, xPos, yPos Position, xOffset, yOffset int) *OverlayModel {
	return &OverlayModel{
		Foreground:  foreground,
		Background:  background,
		XPosition:   xPos,
		YPosition:   yPos,
		XOffset:     xOffset,
		YOffset:     yOffset,
		ShowBorder:  false,
		BorderWidth: 0,
	}
}

// NewOverlayWithOptions creates a new overlay model with options
func NewOverlayWithOptions(foreground, background tea.Model, xPos, yPos Position, xOffset, yOffset int, options OverlayOptions) *OverlayModel {
	return &OverlayModel{
		Foreground:  foreground,
		Background:  background,
		XPosition:   xPos,
		YPosition:   yPos,
		XOffset:     xOffset,
		YOffset:     yOffset,
		ShowBorder:  options.ShowBorder,
		BorderWidth: options.BorderWidth,
	}
}

// Init initializes the overlay
func (m *OverlayModel) Init() tea.Cmd {
	// Initialize both models
	var cmds []tea.Cmd
	if m.Foreground != nil {
		if cmd := m.Foreground.Init(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if m.Background != nil {
		if cmd := m.Background.Init(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if len(cmds) > 0 {
		return tea.Batch(cmds...)
	}
	return nil
}

// Update handles messages - passes updates to both foreground and background
func (m *OverlayModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Update foreground
	if m.Foreground != nil {
		var cmd tea.Cmd
		m.Foreground, cmd = m.Foreground.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		// Check if foreground model wants to quit (for standalone use)
		// This allows models like ConfirmModel to signal quit when used standalone
		if confirmModel, ok := m.Foreground.(*ConfirmModel); ok {
			if confirmModel.Quit {
				return m, tea.Quit
			}
		}
	}

	// Update background
	if m.Background != nil {
		var cmd tea.Cmd
		m.Background, cmd = m.Background.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Handle window size messages
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
	}

	if len(cmds) > 0 {
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

// View renders the overlay by compositing foreground onto background
func (m *OverlayModel) View() string {
	// Render background
	backgroundView := ""
	if m.Background != nil {
		backgroundView = m.Background.View()
	}

	// Render foreground
	foregroundView := ""
	if m.Foreground != nil {
		foregroundView = m.Foreground.View()
	}

	// Wrap foreground in border if requested
	if m.ShowBorder && foregroundView != "" {
		foregroundView = wrapInBorder(foregroundView, m.BorderWidth)
	}

	// Composite foreground onto background
	// Note: We don't pass width/height - let lipgloss measure dimensions directly (like reference implementation)
	return Composite(foregroundView, backgroundView, m.XPosition, m.YPosition, m.XOffset, m.YOffset)
}

// wrapInBorder wraps content in a card-like border using ui.CardBorder
func wrapInBorder(content string, width int) string {
	// Calculate width if not specified
	if width == 0 {
		// Find the maximum line width
		lines := strings.Split(content, "\n")
		maxWidth := 0
		for _, line := range lines {
			lineWidth := lipgloss.Width(line)
			if lineWidth > maxWidth {
				maxWidth = lineWidth
			}
		}
		// Add padding for border (2 chars on each side = 4 total)
		width = maxWidth + 4
		if width < 20 {
			width = 20 // Minimum width
		}
	}

	// Use ui.CardBorder to wrap the content
	// CardBorder already has padding, so we need to account for that
	borderedContent := ui.CardBorder.Width(width).Padding(1, 2).Render(content)
	return borderedContent
}

// Composite composites a foreground string onto a background string at the specified position.
// This implementation matches bubbletea-overlay: https://github.com/rmhubbert/bubbletea-overlay
func Composite(foreground, background string, xPos, yPos Position, xOffset, yOffset int) string {
	if foreground == "" {
		return background
	}
	if background == "" {
		return foreground
	}

	// Normalize line endings and split into lines
	fgLines := normalizeLines(foreground)
	bgLines := normalizeLines(background)

	// Get dimensions using lipgloss
	fgWidth, fgHeight := lipgloss.Size(foreground)
	bgWidth, bgHeight := lipgloss.Size(background)

	// If foreground completely covers background, return foreground
	if fgWidth >= bgWidth && fgHeight >= bgHeight {
		return foreground
	}

	// Calculate offsets
	x, y := calculateOffsets(foreground, background, xPos, yPos, xOffset, yOffset)

	// Clamp to ensure foreground stays within background bounds
	x = clamp(x, 0, bgWidth-fgWidth)
	y = clamp(y, 0, bgHeight-fgHeight)

	var sb strings.Builder

	for i, bgLine := range bgLines {
		if i > 0 {
			sb.WriteByte('\n')
		}

		// If this line is outside the foreground area, just write background
		if i < y || i >= y+fgHeight {
			sb.WriteString(bgLine)
			continue
		}

		// This line is within the foreground area
		pos := 0

		// Write left part of background (before foreground)
		if x > 0 {
			left := ansi.Truncate(bgLine, x, "")
			pos = ansi.StringWidth(left)
			sb.WriteString(left)
			if pos < x {
				sb.WriteString(whitespace(x - pos))
				pos = x
			}
		}

		// Write foreground line
		fgLine := fgLines[i-y]
		sb.WriteString(fgLine)
		pos += ansi.StringWidth(fgLine)

		// Write right part of background (after foreground)
		right := ansi.TruncateLeft(bgLine, pos, "")
		bgLineWidth := ansi.StringWidth(bgLine)
		rightWidth := ansi.StringWidth(right)
		if rightWidth <= bgLineWidth-pos {
			sb.WriteString(whitespace(bgLineWidth - rightWidth - pos))
		}
		sb.WriteString(right)
	}

	return sb.String()
}

// calculateOffsets calculates the actual vertical and horizontal offsets used to position the foreground
// relative to the background, matching bubbletea-overlay's implementation exactly.
func calculateOffsets(fg, bg string, xPos, yPos Position, xOff, yOff int) (int, int) {
	var x, y int

	// Get dimensions using lipgloss (same as reference implementation)
	fgWidth := lipgloss.Width(fg)
	fgHeight := lipgloss.Height(fg)
	bgWidth := lipgloss.Width(bg)
	bgHeight := lipgloss.Height(bg)

	// Handle X axis positioning
	switch xPos {
	case Left:
		x = 0
	case Right:
		x = bgWidth - fgWidth
	case Center:
		// Calculate center position: (bgWidth - fgWidth) / 2
		// This centers the foreground within the background
		x = (bgWidth - fgWidth) / 2
	default:
		x = 0
	}

	// Handle Y axis positioning
	switch yPos {
	case Top:
		y = 0
	case Bottom:
		y = bgHeight - fgHeight
	case Center:
		// Calculate center position: (bgHeight - fgHeight) / 2
		// This centers the foreground within the background
		y = (bgHeight - fgHeight) / 2
	default:
		y = 0
	}

	return x + xOff, y + yOff
}

// clamp clamps a value between lower and upper bounds.
func clamp(v, lower, upper int) int {
	if lower > upper {
		lower, upper = upper, lower
	}
	if v < lower {
		return lower
	}
	if v > upper {
		return upper
	}
	return v
}

// normalizeLines normalizes line endings and splits into lines.
func normalizeLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.Split(s, "\n")
}

// whitespace returns a string of whitespace characters of the requested length.
func whitespace(length int) string {
	return strings.Repeat(" ", length)
}

// BlankBackgroundModel is a simple blank background for modals
type BlankBackgroundModel struct {
	Width  int
	Height int
}

func (m *BlankBackgroundModel) Init() tea.Cmd {
	return nil
}

func (m *BlankBackgroundModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil
	}
	return m, nil
}

func (m *BlankBackgroundModel) View() string {
	// Return a blank screen with appropriate dimensions
	if m.Height == 0 {
		m.Height = 24
	}
	if m.Width == 0 {
		m.Width = 80
	}
	// Return lines with proper width to fill the screen
	// Each line should be the full width so lipgloss.Size() can measure it correctly
	// Don't add trailing newline to avoid affecting height calculation
	blankLine := strings.Repeat(" ", m.Width)
	lines := make([]string, m.Height)
	for i := range lines {
		lines[i] = blankLine
	}
	return strings.Join(lines, "\n")
}
