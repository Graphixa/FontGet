package components

import (
	"strings"
	"unicode/utf8"

	"fontget/internal/ui"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/cellbuf"
)

// CardInnerContentWidth is the display width available for card body text inside
// horizontal padding (matches lipgloss wrap: width − left − right padding).
func CardInnerContentWidth(cardWidth int, horizontalPadding int) int {
	if cardWidth <= 0 {
		cardWidth = 80
	}
	if horizontalPadding < 0 {
		horizontalPadding = 0
	}
	w := cardWidth - 2*horizontalPadding
	if w < 1 {
		return 1
	}
	return w
}

func padLineToInnerWidth(line string, inner int) string {
	if inner < 1 {
		return line
	}
	w := ansi.StringWidth(line)
	if w >= inner {
		return line
	}
	return line + strings.Repeat(" ", inner-w)
}

// buildCardSectionContent lays out sections with uniform inner width so lipgloss
// does not insert uneven trailing spaces between rows.
func buildCardSectionContent(sections []CardSection, inner int) string {
	if len(sections) == 0 {
		return ""
	}
	var blocks []string
	for _, sec := range sections {
		switch {
		case sec.Label == "" && sec.Value == "":
			blocks = append(blocks, padLineToInnerWidth("", inner))
		case sec.IsURL && sec.Value != "":
			blocks = append(blocks, strings.Join(wrapURLSection(sec.Label, sec.Value, inner), "\n"))
		default:
			blocks = append(blocks, strings.Join(formatPlainSection(sec, inner), "\n"))
		}
	}
	return strings.Join(blocks, "\n")
}

func formatPlainSection(sec CardSection, inner int) []string {
	if sec.Label == "" {
		line := ui.Text.Render(sec.Value)
		return []string{padLineToInnerWidth(line, inner)}
	}
	prefix := ui.CardLabel.Render(sec.Label + ": ")
	wPref := ansi.StringWidth(prefix)
	rw := inner - wPref
	if rw < 1 {
		rw = 1
	}
	valStyled := ui.Text.Render(sec.Value)
	wrapped := cellbuf.Wrap(valStyled, rw, "")
	parts := strings.Split(wrapped, "\n")
	out := make([]string, 0, len(parts))
	indent := strings.Repeat(" ", wPref)
	for i, p := range parts {
		var line string
		if i == 0 {
			line = prefix + p
		} else {
			line = indent + p
		}
		out = append(out, padLineToInnerWidth(line, inner))
	}
	return out
}

func wrapURLSection(label, href string, inner int) []string {
	prefix := ui.CardLabel.Render(label + ": ")
	wPref := ansi.StringWidth(prefix)
	if inner <= wPref+1 {
		line := prefix + ui.FormatTerminalURL(href)
		return []string{padLineToInnerWidth(line, inner)}
	}
	firstCap := inner - wPref
	contCap := inner - wPref
	chunks := chunkURLForWidth(href, firstCap, contCap)
	if len(chunks) == 0 {
		return []string{padLineToInnerWidth(prefix, inner)}
	}
	lines := make([]string, 0, len(chunks))
	for i, chunk := range chunks {
		var line string
		if i == 0 {
			line = prefix + ui.FormatTerminalURLChunk(href, chunk)
		} else {
			line = strings.Repeat(" ", wPref) + ui.FormatTerminalURLChunk(href, chunk)
		}
		lines = append(lines, padLineToInnerWidth(line, inner))
	}
	return lines
}

// chunkURLForWidth splits urlStr into lines that fit the first and continuation
// widths, preferring breaks at '/'.
func chunkURLForWidth(urlStr string, firstCap, contCap int) []string {
	if urlStr == "" {
		return nil
	}
	if firstCap < 1 {
		firstCap = 1
	}
	if contCap < 1 {
		contCap = 1
	}
	var out []string
	rem := urlStr
	capW := firstCap
	for rem != "" {
		if ansi.StringWidth(rem) <= capW {
			out = append(out, rem)
			break
		}
		cut := breakURLLine(rem, capW)
		if cut < 1 && rem != "" {
			_, sz := utf8.DecodeRuneInString(rem)
			cut = sz
		}
		out = append(out, rem[:cut])
		rem = rem[cut:]
		capW = contCap
	}
	return out
}

// breakURLLine returns a byte index such that ansi.StringWidth(s[:idx]) <= maxW,
// preferring the last '/' before the limit.
func breakURLLine(s string, maxW int) int {
	if s == "" || maxW < 1 {
		return 0
	}
	if ansi.StringWidth(s) <= maxW {
		return len(s)
	}
	var acc int
	lastSlashBytes := -1
	for i := 0; i < len(s); {
		r, sz := utf8.DecodeRuneInString(s[i:])
		if sz == 0 {
			break
		}
		rs := string(r)
		w := ansi.StringWidth(rs)
		if acc+w > maxW {
			if lastSlashBytes >= 0 {
				return lastSlashBytes + 1
			}
			if i == 0 {
				return sz
			}
			return i
		}
		if r == '/' {
			lastSlashBytes = i
		}
		acc += w
		i += sz
	}
	return len(s)
}
