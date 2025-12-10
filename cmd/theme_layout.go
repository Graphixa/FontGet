package cmd

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// LayoutConfig holds configuration for layout calculations
type LayoutConfig struct {
	TerminalWidth  int
	TerminalHeight int
	HeaderHeight   int
	FooterHeight   int
	MarginWidth    int
	SeparatorWidth int
}

// PanelLayout holds calculated panel dimensions
type PanelLayout struct {
	LeftWidth       int
	RightWidth      int
	PanelHeight     int
	AvailableWidth  int
	AvailableHeight int
}

// CalculatePanelLayout calculates panel dimensions based on terminal size and layout config
func CalculatePanelLayout(config LayoutConfig) PanelLayout {
	// Calculate available space
	marginWidth := config.MarginWidth
	if marginWidth == 0 {
		marginWidth = 2 // Default: 1 char on each side
	}

	separatorWidth := config.SeparatorWidth
	if separatorWidth == 0 {
		separatorWidth = 1 // Default separator width
	}

	availableWidth := config.TerminalWidth - marginWidth
	availableHeight := config.TerminalHeight - config.HeaderHeight - config.FooterHeight

	// Ensure minimum dimensions
	if availableWidth < 40 {
		availableWidth = 40
	}
	if availableHeight < 10 {
		availableHeight = 10
	}

	// Calculate panel widths (30/70 split accounting for separator)
	// Left column: 30%, Right column: 70%
	panelAreaWidth := availableWidth - separatorWidth
	if panelAreaWidth < 0 {
		panelAreaWidth = 0
	}

	// 30% for left, 70% for right
	leftWidth := int(float64(panelAreaWidth) * 0.3)
	rightWidth := panelAreaWidth - leftWidth

	// Ensure minimum panel widths
	if leftWidth < 20 {
		leftWidth = 20
	}
	if rightWidth < 20 {
		rightWidth = 20
	}

	// Safety check: ensure total doesn't exceed terminal width
	maxTotalWidth := config.TerminalWidth
	if leftWidth+rightWidth+separatorWidth > maxTotalWidth {
		panelAreaWidth = maxTotalWidth - separatorWidth
		if panelAreaWidth < 0 {
			panelAreaWidth = 0
		}
		leftWidth = panelAreaWidth / 2
		rightWidth = panelAreaWidth - leftWidth
	}

	return PanelLayout{
		LeftWidth:       leftWidth,
		RightWidth:      rightWidth,
		PanelHeight:     availableHeight,
		AvailableWidth:  availableWidth,
		AvailableHeight: availableHeight,
	}
}

// trimContent removes trailing newlines and whitespace from content
func trimContent(content string) string {
	// Remove all trailing newlines
	content = strings.TrimRight(content, "\n")
	// Remove any trailing whitespace from each line
	lines := strings.Split(content, "\n")
	trimmed := make([]string, len(lines))
	for i, line := range lines {
		trimmed[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(trimmed, "\n")
}

// renderBorderedPanel renders a panel with border and proper constraints
// borderStyle should already have border and padding configured
func renderBorderedPanel(width, height int, content string, borderStyle lipgloss.Style) string {
	// Ensure minimums so borders are visible
	if width < 6 {
		width = 6
	}
	if height < 4 {
		height = 4
	}

	// CardBorder has Padding(1) and border, so:
	// - Border takes 2 chars (left + right border characters)
	// - Padding takes 2 chars (1 on each side)
	// Total: 4 chars consumed from width/height
	contentWidth := width - 4
	if contentWidth < 0 {
		contentWidth = 0
	}
	contentHeight := height - 4
	if contentHeight < 0 {
		contentHeight = 0
	}

	// Trim content to avoid spacing issues
	trimmed := trimContent(content)

	// Constrain content dimensions
	constrained := lipgloss.NewStyle().
		Width(contentWidth).
		Height(contentHeight).
		Render(trimmed)

	// Apply border with explicit dimensions
	bordered := borderStyle.
		Width(width).
		Height(height).
		Render(constrained)

	// Trim any trailing newlines from the border
	return strings.TrimRight(bordered, "\n")
}

// renderPanelWithoutRightBorder renders a panel without the right border (for left panel)
func renderPanelWithoutRightBorder(width, height int, content string, borderStyle lipgloss.Style) string {
	// Ensure minimums
	if width < 6 {
		width = 6
	}
	if height < 4 {
		height = 4
	}

	contentWidth := width - 4
	if contentWidth < 0 {
		contentWidth = 0
	}
	contentHeight := height - 4
	if contentHeight < 0 {
		contentHeight = 0
	}

	trimmed := trimContent(content)
	constrained := lipgloss.NewStyle().
		Width(contentWidth).
		Height(contentHeight).
		Render(trimmed)

	// Create border style without right border
	partialBorder := borderStyle.
		BorderRight(false).
		Width(width).
		Height(height).
		Render(constrained)

	return strings.TrimRight(partialBorder, "\n")
}

// renderPanelWithoutLeftBorder renders a panel without the left border (for right panel)
func renderPanelWithoutLeftBorder(width, height int, content string, borderStyle lipgloss.Style) string {
	// Ensure minimums
	if width < 6 {
		width = 6
	}
	if height < 4 {
		height = 4
	}

	contentWidth := width - 4
	if contentWidth < 0 {
		contentWidth = 0
	}
	contentHeight := height - 4
	if contentHeight < 0 {
		contentHeight = 0
	}

	trimmed := trimContent(content)
	constrained := lipgloss.NewStyle().
		Width(contentWidth).
		Height(contentHeight).
		Render(trimmed)

	// Create border style without left border
	partialBorder := borderStyle.
		BorderLeft(false).
		Width(width).
		Height(height).
		Render(constrained)

	return strings.TrimRight(partialBorder, "\n")
}

// renderCombinedPanels renders two panels side-by-side with a shared border
func renderCombinedPanels(title string, leftWidth, rightWidth, height int, leftContent, rightContent string, borderStyle lipgloss.Style, separatorColor, borderColor lipgloss.Color, titleStyle lipgloss.Style) string {
	// Guard minimums to avoid negative repeat counts
	if leftWidth < 4 {
		leftWidth = 4
	}
	if rightWidth < 4 {
		rightWidth = 4
	}
	if height < 3 {
		height = 3
	}

	// Calculate content dimensions so that middle lines align perfectly:
	// Content has padding (1 on each side = 2 total) added in renderLeftPanelContent/renderRightPanelContent
	// So: 1 (left border) + leftContentWidth (with padding) + 1 (separator) + rightContentWidth (with padding) + 1 (right border)
	// should equal leftWidth + 1 + rightWidth
	// Content with padding = panel width - border (2) = panel width - 2
	// But we account for the border in the line construction, so content width = panel width - 1
	leftContentWidth := leftWidth - 1 // accounts for left border, content has its own padding
	if leftContentWidth < 0 {
		leftContentWidth = 0
	}
	rightContentWidth := rightWidth - 1 // accounts for right border, content has its own padding
	if rightContentWidth < 0 {
		rightContentWidth = 0
	}
	// Content height: borders (top/bottom) consume 2
	contentHeight := height - 2
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Trim content - don't constrain width here, we'll do it per-line to ensure exact alignment
	leftTrimmed := trimContent(leftContent)
	rightTrimmed := trimContent(rightContent)

	// Constrain height only - width will be handled per-line
	leftConstrained := lipgloss.NewStyle().
		Height(contentHeight).
		Render(leftTrimmed)

	rightConstrained := lipgloss.NewStyle().
		Height(contentHeight).
		Render(rightTrimmed)

	// Border color is passed as parameter

	// Build border frame manually with title integrated into top border
	// Top border: ╭ Title ───┬────╮
	// Middle:    │     │     │
	// Bottom:    ╰─────┴─────╯

	topLeft := "╭"
	topRight := "╮"
	bottomLeft := "╰"
	bottomRight := "╯"
	topTee := "┬"
	bottomTee := "┴"
	vertical := "│"
	horizontal := "─"

	// Create styled border characters
	borderCharStyle := lipgloss.NewStyle().Foreground(borderColor)
	separatorStyle := lipgloss.NewStyle().Foreground(separatorColor)

	topLeftChar := borderCharStyle.Render(topLeft)
	topRightChar := borderCharStyle.Render(topRight)
	bottomLeftChar := borderCharStyle.Render(bottomLeft)
	bottomRightChar := borderCharStyle.Render(bottomRight)
	topTeeChar := borderCharStyle.Render(topTee)
	bottomTeeChar := borderCharStyle.Render(bottomTee)
	leftBorderChar := borderCharStyle.Render(vertical)
	rightBorderChar := borderCharStyle.Render(vertical)
	separatorChar := separatorStyle.Render(vertical)
	horizontalChar := borderCharStyle.Render(horizontal)

	// Styled title (with natural padding from PageTitle style: 1 space on each side)
	titleRendered := titleStyle.Render(title)
	titleWidth := lipgloss.Width(titleRendered) // Visual width (strips ANSI codes)

	// Total border width must be: leftWidth + 1 (separator) + rightWidth
	totalBorderWidth := leftWidth + 1 + rightWidth

	// Inner widths for top/bottom segments (width minus corners)
	leftInner := leftWidth - 1 // exclude left corner
	if leftInner < 0 {
		leftInner = 0
	}
	rightInner := rightWidth - 1 // exclude right corner
	if rightInner < 0 {
		rightInner = 0
	}

	// Title placement: match card component pattern: dash + space + title + space + dash
	// Pattern: ╭─ Title ─────┬────╮
	// The title already has padding (1 space on each side) from PageTitle style, but we add spaces around it like the card does
	// Critical: The T-junction must be at exactly leftWidth characters from the start
	// Left segment structure: corner (1) + dash (1) + space (1) + title + space (1) + dash (1) + remaining dashes
	// Total left segment width must be exactly leftWidth
	// So: remaining dashes = leftWidth - 1 (corner) - 1 (dash) - 1 (space) - titleWidth - 1 (space) - 1 (dash)
	titleSectionWidth := 1 + 1 + titleWidth + 1 + 1 // dash + space + title + space + dash (excluding corner)
	remainingLeft := leftInner - titleSectionWidth
	if remainingLeft < 0 {
		remainingLeft = 0
	}

	// Build left segment: corner + title section + remaining dashes
	// This should total exactly leftWidth (corner + leftInner)
	topBorderLeft := topLeftChar + horizontalChar + " " + titleRendered + " " + horizontalChar + strings.Repeat(horizontalChar, remainingLeft)

	// Verify left segment width is exactly leftWidth (corner + leftInner)
	leftSegmentWidth := lipgloss.Width(topBorderLeft)
	if leftSegmentWidth != leftWidth {
		// Adjust remaining dashes to make it exactly leftWidth
		actualRemaining := leftWidth - lipgloss.Width(topLeftChar+horizontalChar+" "+titleRendered+" "+horizontalChar)
		if actualRemaining < 0 {
			actualRemaining = 0
		}
		topBorderLeft = topLeftChar + horizontalChar + " " + titleRendered + " " + horizontalChar + strings.Repeat(horizontalChar, actualRemaining)
	}

	// Right segment should be exactly rightWidth - 1 (rightInner) + corner
	topBorderRight := strings.Repeat(horizontalChar, rightInner) + topRightChar

	// Combine: left segment + tee + right segment
	topBorder := topBorderLeft + topTeeChar + topBorderRight

	// Verify total width matches (should be leftWidth + 1 + rightWidth)
	actualWidth := lipgloss.Width(topBorder)
	if actualWidth != totalBorderWidth {
		// Adjust right segment to fix total width
		adjust := totalBorderWidth - actualWidth
		newRightInner := rightInner + adjust
		if newRightInner < 0 {
			newRightInner = 0
		}
		topBorderRight = strings.Repeat(horizontalChar, newRightInner) + topRightChar
		topBorder = topBorderLeft + topTeeChar + topBorderRight
	}

	// Build bottom border (same inner widths as top, but no title)
	// Left inner = leftWidth - 1, right inner = rightWidth - 1
	bottomBorder := bottomLeftChar + strings.Repeat(horizontalChar, leftWidth-1) + bottomTeeChar + strings.Repeat(horizontalChar, rightWidth-1) + bottomRightChar

	// Split content into lines
	leftLines := strings.Split(strings.TrimRight(leftConstrained, "\n"), "\n")
	rightLines := strings.Split(strings.TrimRight(rightConstrained, "\n"), "\n")

	// Ensure both have the same number of lines
	maxLines := contentHeight
	if len(leftLines) < maxLines {
		leftLines = append(leftLines, make([]string, maxLines-len(leftLines))...)
	}
	if len(rightLines) < maxLines {
		rightLines = append(rightLines, make([]string, maxLines-len(rightLines))...)
	}
	if len(leftLines) > maxLines {
		leftLines = leftLines[:maxLines]
	}
	if len(rightLines) > maxLines {
		rightLines = rightLines[:maxLines]
	}

	// Build middle lines - content already has padding from renderLeftPanelContent/renderRightPanelContent
	var middleLines []string
	for i := 0; i < maxLines; i++ {
		leftLine := leftLines[i]
		rightLine := rightLines[i]

		// Get the actual visual width (strips ANSI codes)
		leftLineWidth := lipgloss.Width(leftLine)
		rightLineWidth := lipgloss.Width(rightLine)

		// Pad to exact width needed (leftContentWidth and rightContentWidth)
		// Content already has padding, so we just need to ensure width matches
		var leftPadded string
		if leftLineWidth < leftContentWidth {
			// Pad with spaces on the right
			leftPadded = leftLine + strings.Repeat(" ", leftContentWidth-leftLineWidth)
		} else if leftLineWidth > leftContentWidth {
			// Truncate if too wide (shouldn't happen, but safety check)
			leftPadded = lipgloss.NewStyle().Width(leftContentWidth).MaxWidth(leftContentWidth).Render(leftLine)
		} else {
			leftPadded = leftLine
		}

		var rightPadded string
		if rightLineWidth < rightContentWidth {
			// Pad with spaces on the right
			rightPadded = rightLine + strings.Repeat(" ", rightContentWidth-rightLineWidth)
		} else if rightLineWidth > rightContentWidth {
			// Truncate if too wide (shouldn't happen, but safety check)
			rightPadded = lipgloss.NewStyle().Width(rightContentWidth).MaxWidth(rightContentWidth).Render(rightLine)
		} else {
			rightPadded = rightLine
		}

		// Build line: │ left content │ right content │
		// Content already includes its own padding, and we've ensured exact width
		middleLine := leftBorderChar + leftPadded + separatorChar + rightPadded + rightBorderChar
		middleLines = append(middleLines, middleLine)
	}

	// Combine all lines
	var result strings.Builder
	result.WriteString(topBorder)
	result.WriteString("\n")
	result.WriteString(strings.Join(middleLines, "\n"))
	result.WriteString("\n")
	result.WriteString(bottomBorder)

	return result.String()
}

// countLines counts the number of lines in a string (excluding trailing newline)
func countLines(s string) int {
	if s == "" {
		return 0
	}
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	return len(lines)
}
