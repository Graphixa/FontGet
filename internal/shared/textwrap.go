package shared

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/charmbracelet/x/term"
)

// WrapOptions provides configuration for text wrapping
type WrapOptions struct {
	Width            int    // Target width (0 = auto-detect terminal width)
	Indent           string // Prefix for each line (e.g., "  " for indentation)
	PreserveMarkdown bool   // Keep markdown formatting (default: false, strips it)
	MaxWidth         int    // Maximum width even if terminal is wider (0 = no max)
	MinWidth         int    // Minimum width (default: 40)
}

// WrapText wraps text to the specified width, preserving words.
// Returns a slice of strings, one per line.
// If width is 0, automatically detects terminal width.
func WrapText(text string, width int) []string {
	opts := WrapOptions{
		Width: width,
	}
	return WrapTextWithOptions(text, opts)
}

// WrapTextWithOptions wraps text with additional configuration options.
func WrapTextWithOptions(text string, opts WrapOptions) []string {
	// Determine target width
	width := opts.Width
	if width == 0 {
		width = GetTerminalWidth()
	}

	// Apply min/max constraints
	if opts.MinWidth > 0 && width < opts.MinWidth {
		width = opts.MinWidth
	}
	if opts.MaxWidth > 0 && width > opts.MaxWidth {
		width = opts.MaxWidth
	}

	// Account for indent in width calculation
	indentLen := len(opts.Indent)
	effectiveWidth := width
	if indentLen > 0 {
		effectiveWidth = width - indentLen
		if effectiveWidth < 10 {
			effectiveWidth = 10 // Minimum readable width after indent
		}
	}

	// Strip markdown if requested
	if !opts.PreserveMarkdown {
		text = stripMarkdown(text)
	}

	// Handle empty text
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{""}
	}

	// Split into paragraphs (preserve existing line breaks)
	paragraphs := strings.Split(text, "\n\n")
	var allLines []string

	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			allLines = append(allLines, "")
			continue
		}

		// Wrap this paragraph
		lines := wrapParagraph(paragraph, effectiveWidth, opts.Indent)
		allLines = append(allLines, lines...)
	}

	return allLines
}

// WrapTextFromFile wraps text from a file, handling markdown if needed.
// Returns a slice of strings, one per line.
func WrapTextFromFile(filePath string, width int) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("unable to open file: %w", err)
	}
	defer file.Close()

	var content strings.Builder
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		content.WriteString(scanner.Text())
		content.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("unable to read file: %w", err)
	}

	return WrapText(content.String(), width), nil
}

// GetTerminalWidth returns the current terminal width, or a default if unavailable.
func GetTerminalWidth() int {
	// Try to get actual terminal size using term.GetSize
	// os.Stdout.Fd() returns uintptr, which is what term.GetSize expects
	width, _, err := term.GetSize(os.Stdout.Fd())
	if err == nil && width > 0 {
		return width
	}

	// Try to get from COLUMNS environment variable (fallback)
	if cols := os.Getenv("COLUMNS"); cols != "" {
		var width int
		if _, err := fmt.Sscanf(cols, "%d", &width); err == nil && width > 0 {
			return width
		}
	}

	// Default to 80 characters (standard terminal width)
	return 80
}

// wrapParagraph wraps a single paragraph to the specified width
func wrapParagraph(text string, width int, indent string) []string {
	if width <= 0 {
		width = 80 // Default width
	}

	// Normalize whitespace - replace multiple spaces with single space
	text = strings.Join(strings.Fields(text), " ")

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{indent}
	}

	var lines []string
	currentLine := ""

	for _, word := range words {
		testLine := currentLine
		if testLine != "" {
			testLine += " " + word
		} else {
			testLine = word
		}

		// Calculate display width (accounting for potential multi-byte characters)
		testWidth := calculateDisplayWidth(testLine)

		if testWidth <= width {
			currentLine = testLine
		} else {
			// Current line is too long, start a new line
			if currentLine != "" {
				lines = append(lines, indent+currentLine)
				currentLine = word
			} else {
				// Word itself is longer than width - add it anyway and break if needed
				if calculateDisplayWidth(word) > width {
					// Break long word (preserve as much as possible)
					lines = append(lines, indent+word)
					currentLine = ""
				} else {
					currentLine = word
				}
			}
		}
	}

	// Add the last line
	if currentLine != "" {
		lines = append(lines, indent+currentLine)
	}

	return lines
}

// calculateDisplayWidth calculates the display width of a string
// This accounts for multi-byte characters and ANSI escape codes
func calculateDisplayWidth(s string) int {
	// Simple implementation: count runes (works for most cases)
	// For more accurate width calculation with emoji/wide chars, we'd need
	// a library like github.com/mattn/go-runewidth, but we'll keep it simple
	// for now to avoid dependencies
	width := 0
	for _, r := range s {
		if unicode.IsPrint(r) {
			// Most characters are 1 width, but some (like emoji) are 2
			// For now, we'll use a simple heuristic
			if r > 0x1F000 && r < 0x1FAFF {
				// Emoji range - typically 2 width
				width += 2
			} else {
				width++
			}
		}
	}
	return width
}

// stripMarkdown removes markdown formatting from text while preserving content
func stripMarkdown(text string) string {
	// Remove markdown headers (# ## ###)
	text = strings.ReplaceAll(text, "# ", "")
	text = strings.ReplaceAll(text, "## ", "")
	text = strings.ReplaceAll(text, "### ", "")
	text = strings.ReplaceAll(text, "#### ", "")
	text = strings.ReplaceAll(text, "##### ", "")
	text = strings.ReplaceAll(text, "###### ", "")

	// Remove bold/italic markers
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "*", "")
	text = strings.ReplaceAll(text, "__", "")
	text = strings.ReplaceAll(text, "_", "")

	// Remove code blocks
	text = strings.ReplaceAll(text, "`", "")

	// Remove links but keep text [text](url) -> text
	// Simple regex-like replacement
	for {
		start := strings.Index(text, "[")
		end := strings.Index(text, "]")
		if start == -1 || end == -1 || end <= start {
			break
		}
		linkText := text[start+1 : end]
		// Find the URL part and remove it
		urlStart := strings.Index(text[end:], "(")
		urlEnd := strings.Index(text[end:], ")")
		if urlStart != -1 && urlEnd != -1 {
			text = text[:start] + linkText + text[end+urlStart+urlEnd+2:]
		} else {
			text = text[:start] + linkText + text[end+1:]
		}
	}

	// Remove horizontal rules
	text = strings.ReplaceAll(text, "---", "")
	text = strings.ReplaceAll(text, "***", "")

	// Clean up extra whitespace
	text = strings.Join(strings.Fields(text), " ")

	return text
}
