package components

import (
	"strings"

	"fontget/internal/ui"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const (
	DefaultDialogMaxWidth = 72
	DefaultDialogMinWidth = 40
)

const dialogTruncateTail = "..."

// DialogOpts constrains the rendered modal width (lipgloss outer width).
type DialogOpts struct {
	MaxWidth int
	MinWidth int
	Width    int // If > 0, use this width (still capped by MaxWidth).
	// ErrorTitle draws the integrated border title with ErrorText instead of CardTitle (e.g. "Error").
	ErrorTitle bool
}

// RenderDialog wraps title, body, and optional buttons in ui.DialogModal (theme border, no shell background).
// Title is drawn on the integrated top border (same as Card). Body and button lines are truncated to the
// inner text width so the rounded border never breaks on overflow. A one-cell margin is applied outside the
// border so the modal does not sit flush against the background.
func RenderDialog(title string, body string, buttons *ButtonGroup, opts DialogOpts) string {
	maxW := opts.MaxWidth
	if maxW <= 0 {
		maxW = DefaultDialogMaxWidth
	}
	minW := opts.MinWidth
	if minW <= 0 {
		minW = DefaultDialogMinWidth
	}

	outer := maxW
	if opts.Width > 0 && opts.Width <= maxW {
		outer = opts.Width
	}
	if outer < minW {
		outer = minW
	}
	if outer > maxW {
		outer = maxW
	}

	inner := CardInnerContentWidth(outer, 2)

	title = strings.TrimSpace(title)
	body = strings.TrimRight(body, "\n")

	var bodyLines []string
	if strings.TrimSpace(body) != "" {
		for _, line := range strings.Split(body, "\n") {
			if strings.TrimSpace(line) == "" {
				bodyLines = append(bodyLines, padLineToInnerWidth("", inner))
				continue
			}
			tr := ansi.Truncate(line, inner, dialogTruncateTail)
			bodyLines = append(bodyLines, padLineToInnerWidth(tr, inner))
		}
	}

	var blocks []string
	if len(bodyLines) > 0 {
		blocks = append(blocks, strings.Join(bodyLines, "\n"))
	}
	if buttons != nil {
		btn := ansi.Truncate(buttons.Render(), inner, dialogTruncateTail)
		blocks = append(blocks, padLineToInnerWidth(btn, inner))
	}
	innerContent := strings.Join(blocks, "\n\n")

	rendered := ui.DialogModal.Width(outer).Render(innerContent)
	out := rendered
	if title != "" {
		lines := strings.Split(rendered, "\n")
		if len(lines) > 0 {
			tw := lipgloss.Width(lines[len(lines)-1])
			if tw <= 0 {
				tw = outer
			}
			if opts.ErrorTitle {
				lines[0] = IntegratedRoundedTopBorderLineError(tw, title)
			} else {
				lines[0] = IntegratedRoundedTopBorderLine(tw, title)
			}
			out = strings.Join(lines, "\n")
		}
	}
	// One cell of breathing room outside the rounded border (card size unchanged).
	return lipgloss.NewStyle().Margin(1).Render(out)
}
