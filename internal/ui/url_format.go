package ui

import (
	"net/url"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// FormatTerminalURL renders href as an OSC 8 terminal hyperlink with URLLink styling.
// Use this (not raw lipgloss) for values that should open in the browser on ctrl+click
// in supporting terminals. http and https only; other schemes render as plain Text.
func FormatTerminalURL(href string) string {
	if strings.TrimSpace(href) == "" {
		return ""
	}
	u, err := url.Parse(strings.TrimSpace(href))
	if err != nil || u.Scheme == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return Text.Render(href)
	}
	clean := u.String()
	return ansi.SetHyperlink(clean) + URLLink.Render(clean) + ansi.ResetHyperlink()
}

// FormatTerminalURLChunk wraps a substring of the same href for multi-line URL display.
// Each chunk is independently hyperlinked to href (supported in common terminals).
func FormatTerminalURLChunk(href, visibleChunk string) string {
	if visibleChunk == "" {
		return ""
	}
	u, err := url.Parse(strings.TrimSpace(href))
	if err != nil || u.Scheme == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return Text.Render(visibleChunk)
	}
	clean := u.String()
	return ansi.SetHyperlink(clean) + URLLink.Render(visibleChunk) + ansi.ResetHyperlink()
}
