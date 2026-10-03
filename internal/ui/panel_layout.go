package ui

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
	marginWidth := config.MarginWidth
	if marginWidth == 0 {
		marginWidth = 2
	}

	separatorWidth := config.SeparatorWidth
	if separatorWidth == 0 {
		separatorWidth = 1
	}

	availableWidth := config.TerminalWidth - marginWidth
	availableHeight := config.TerminalHeight - config.HeaderHeight - config.FooterHeight

	if availableWidth < 40 {
		availableWidth = 40
	}
	if availableHeight < 10 {
		availableHeight = 10
	}

	panelAreaWidth := availableWidth - separatorWidth
	if panelAreaWidth < 0 {
		panelAreaWidth = 0
	}

	leftWidth := int(float64(panelAreaWidth) * 0.3)
	rightWidth := panelAreaWidth - leftWidth

	if leftWidth < 20 {
		leftWidth = 20
	}
	if rightWidth < 20 {
		rightWidth = 20
	}

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

func trimPanelContent(content string) string {
	content = strings.TrimRight(content, "\n")
	lines := strings.Split(content, "\n")
	trimmed := make([]string, len(lines))
	for i, line := range lines {
		trimmed[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(trimmed, "\n")
}

// RenderCombinedPanels renders two panels side-by-side with a shared border.
func RenderCombinedPanels(title string, leftWidth, rightWidth, height int, leftContent, rightContent string, _ lipgloss.Style, separatorColor, borderColor lipgloss.Color, titleStyle lipgloss.Style) string {
	if leftWidth < 4 {
		leftWidth = 4
	}
	if rightWidth < 4 {
		rightWidth = 4
	}
	if height < 3 {
		height = 3
	}

	leftContentWidth := leftWidth - 1
	if leftContentWidth < 0 {
		leftContentWidth = 0
	}
	rightContentWidth := rightWidth - 1
	if rightContentWidth < 0 {
		rightContentWidth = 0
	}
	contentHeight := height - 2
	if contentHeight < 1 {
		contentHeight = 1
	}

	leftConstrained := lipgloss.NewStyle().
		Height(contentHeight).
		Render(trimPanelContent(leftContent))

	rightConstrained := lipgloss.NewStyle().
		Height(contentHeight).
		Render(trimPanelContent(rightContent))

	borderCharStyle := lipgloss.NewStyle().Foreground(borderColor)
	separatorStyle := lipgloss.NewStyle().Foreground(separatorColor)

	topLeftChar := borderCharStyle.Render("╭")
	topRightChar := borderCharStyle.Render("╮")
	bottomLeftChar := borderCharStyle.Render("╰")
	bottomRightChar := borderCharStyle.Render("╯")
	topTeeChar := borderCharStyle.Render("┬")
	bottomTeeChar := borderCharStyle.Render("┴")
	leftBorderChar := borderCharStyle.Render("│")
	rightBorderChar := borderCharStyle.Render("│")
	separatorChar := separatorStyle.Render("│")
	horizontalChar := borderCharStyle.Render("─")

	titleRendered := titleStyle.Render(title)
	titleWidth := lipgloss.Width(titleRendered)
	totalBorderWidth := leftWidth + 1 + rightWidth

	leftInner := leftWidth - 1
	if leftInner < 0 {
		leftInner = 0
	}
	rightInner := rightWidth - 1
	if rightInner < 0 {
		rightInner = 0
	}

	titleSectionWidth := 1 + 1 + titleWidth + 1 + 1
	remainingLeft := leftInner - titleSectionWidth
	if remainingLeft < 0 {
		remainingLeft = 0
	}

	topBorderLeft := topLeftChar + horizontalChar + " " + titleRendered + " " + horizontalChar + strings.Repeat(horizontalChar, remainingLeft)
	if lipgloss.Width(topBorderLeft) != leftWidth {
		actualRemaining := leftWidth - lipgloss.Width(topLeftChar+horizontalChar+" "+titleRendered+" "+horizontalChar)
		if actualRemaining < 0 {
			actualRemaining = 0
		}
		topBorderLeft = topLeftChar + horizontalChar + " " + titleRendered + " " + horizontalChar + strings.Repeat(horizontalChar, actualRemaining)
	}

	topBorderRight := strings.Repeat(horizontalChar, rightInner) + topRightChar
	topBorder := topBorderLeft + topTeeChar + topBorderRight

	if actualWidth := lipgloss.Width(topBorder); actualWidth != totalBorderWidth {
		newRightInner := rightInner + (totalBorderWidth - actualWidth)
		if newRightInner < 0 {
			newRightInner = 0
		}
		topBorderRight = strings.Repeat(horizontalChar, newRightInner) + topRightChar
		topBorder = topBorderLeft + topTeeChar + topBorderRight
	}

	bottomBorder := bottomLeftChar + strings.Repeat(horizontalChar, leftWidth-1) + bottomTeeChar + strings.Repeat(horizontalChar, rightWidth-1) + bottomRightChar

	leftLines := strings.Split(strings.TrimRight(leftConstrained, "\n"), "\n")
	rightLines := strings.Split(strings.TrimRight(rightConstrained, "\n"), "\n")

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

	var middleLines []string
	for i := 0; i < maxLines; i++ {
		leftLine := leftLines[i]
		rightLine := rightLines[i]
		leftLineWidth := lipgloss.Width(leftLine)
		rightLineWidth := lipgloss.Width(rightLine)

		var leftPadded string
		if leftLineWidth < leftContentWidth {
			leftPadded = leftLine + strings.Repeat(" ", leftContentWidth-leftLineWidth)
		} else if leftLineWidth > leftContentWidth {
			leftPadded = lipgloss.NewStyle().Width(leftContentWidth).MaxWidth(leftContentWidth).Render(leftLine)
		} else {
			leftPadded = leftLine
		}

		var rightPadded string
		if rightLineWidth < rightContentWidth {
			rightPadded = rightLine + strings.Repeat(" ", rightContentWidth-rightLineWidth)
		} else if rightLineWidth > rightContentWidth {
			rightPadded = lipgloss.NewStyle().Width(rightContentWidth).MaxWidth(rightContentWidth).Render(rightLine)
		} else {
			rightPadded = rightLine
		}

		middleLines = append(middleLines, leftBorderChar+leftPadded+separatorChar+rightPadded+rightBorderChar)
	}

	var result strings.Builder
	result.WriteString(topBorder)
	result.WriteString("\n")
	result.WriteString(strings.Join(middleLines, "\n"))
	result.WriteString("\n")
	result.WriteString(bottomBorder)
	return result.String()
}
