package components

import (
	"fmt"
	"strings"

	"fontget/internal/ui"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// DefaultStatusPopupMaxOuter is the maximum total width for RenderStatusPopup.
const DefaultStatusPopupMaxOuter = 72

// RenderStatusPopup draws a centered modal-style overlay using the shared dialog shell (DialogModal
// padding, integrated CardTitle-style top border, one-cell outer margin) with a font line and
// gradient progress bar. Use anywhere a compact status overlay should match other modal UIs.
func RenderStatusPopup(phase, fontName, sourceName string, progressPercent float64, maxOuter int) string {
	mid := fmt.Sprintf("'%s' from '%s'", fontName, sourceName)
	midStyled := ui.Text.Render(mid)
	return renderStatusPopupShell(phase, midStyled, progressPercent, maxOuter)
}

// RenderStatusPopupPlain is like RenderStatusPopup but with a single custom middle line (already styled if needed).
// If progressPercent is negative, the progress bar row is omitted.
func RenderStatusPopupPlain(phase, middleLine string, progressPercent float64, maxOuter int) string {
	return renderStatusPopupShell(phase, middleLine, progressPercent, maxOuter)
}

func renderStatusPopupShell(phase, midStyled string, progressPercent float64, maxOuter int) string {
	if maxOuter <= 0 {
		maxOuter = DefaultStatusPopupMaxOuter
	}
	outer := maxOuter
	inner := CardInnerContentWidth(outer, 2)

	line1 := ansi.Truncate(midStyled, inner, dialogTruncateTail)
	line1 = padLineToInnerWidth(line1, inner)

	barW := inner - 14
	if barW < 6 {
		barW = 6
	}
	if barW > 28 {
		barW = 28
	}
	var innerContent string
	if progressPercent >= 0 {
		barLine := InlineProgressBarView(progressPercent, barW)
		barLine = ansi.Truncate(barLine, inner, dialogTruncateTail)
		barLine = padLineToInnerWidth(barLine, inner)
		blank := padLineToInnerWidth("", inner)
		innerContent = strings.Join([]string{line1, blank, barLine}, "\n")
	} else {
		innerContent = line1
	}

	rendered := ui.DialogModal.Width(outer).Render(innerContent)
	lines := strings.Split(rendered, "\n")
	if len(lines) == 0 {
		return lipgloss.NewStyle().Margin(1).Render(rendered)
	}
	tw := lipgloss.Width(lines[len(lines)-1])
	if tw <= 0 {
		tw = outer
	}
	lines[0] = IntegratedRoundedTopBorderLine(tw, phase)
	out := strings.Join(lines, "\n")
	return lipgloss.NewStyle().Margin(1).Render(out)
}
