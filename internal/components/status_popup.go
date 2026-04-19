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

// RenderStatusPopup draws a small centered-style box, e.g. Installing / Uninstalling.
// phase is shown in the top rule (e.g. "Installing"); the middle line is
// 'FontName' from 'SourceName' using ui.Text.
func RenderStatusPopup(phase, fontName, sourceName string, maxOuter int) string {
	if maxOuter <= 0 {
		maxOuter = DefaultStatusPopupMaxOuter
	}
	mid := fmt.Sprintf("'%s' from '%s'", fontName, sourceName)
	midStyled := ui.Text.Render(mid)
	innerNeed := lipgloss.Width(midStyled)
	titleMin := dialogMinOuterForTitle(phase)
	outer := max(titleMin, innerNeed+2, 32)
	if outer > maxOuter {
		outer = maxOuter
	}
	if outer < titleMin && titleMin <= maxOuter {
		outer = titleMin
	}

	var b strings.Builder
	b.WriteString(dialogBorderTop(outer, phase))
	b.WriteByte('\n')
	inner := outer - 2
	line := midStyled
	if lipgloss.Width(line) > inner {
		line = ansi.Truncate(line, inner, "")
	}
	for lipgloss.Width(line) < inner {
		line += " "
	}
	b.WriteString("│" + line + "│")
	b.WriteByte('\n')
	b.WriteString(dialogBorderBottom(outer))
	return b.String()
}

// RenderStatusPopupPlain is like RenderStatusPopup but with a single custom middle line (already styled if needed).
func RenderStatusPopupPlain(phase, middleLine string, maxOuter int) string {
	if maxOuter <= 0 {
		maxOuter = DefaultStatusPopupMaxOuter
	}
	innerNeed := lipgloss.Width(middleLine)
	titleMin := dialogMinOuterForTitle(phase)
	outer := max(titleMin, innerNeed+2, 32)
	if outer > maxOuter {
		outer = maxOuter
	}
	if outer < titleMin && titleMin <= maxOuter {
		outer = titleMin
	}
	var b strings.Builder
	b.WriteString(dialogBorderTop(outer, phase))
	b.WriteByte('\n')
	b.WriteString(dialogMiddleLine(outer, middleLine))
	b.WriteByte('\n')
	b.WriteString(dialogBorderBottom(outer))
	return b.String()
}
