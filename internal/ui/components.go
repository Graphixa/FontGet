package ui

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	pinpkg "github.com/yarlson/pin"
)

// Utility functions for consistent UI rendering

// RenderTitleWithSubtitle renders a title with optional subtitle
func RenderTitleWithSubtitle(title, subtitle string) string {
	if subtitle == "" {
		return PageTitle.Render(title) + "\n"
	}
	return PageTitle.Render(title) + "\n" +
		Text.Render(subtitle) + "\n"
}

// RenderStatusReport renders a status report with consistent styling
func RenderStatusReport(title string, items map[string]int) string {
	var content strings.Builder

	content.WriteString("\n")
	content.WriteString(TextBold.Render(title))
	content.WriteString("\n")
	content.WriteString("---------------------------------------------")
	content.WriteString("\n")

	// Status items
	var statusItems []string
	for label, count := range items {
		var style lipgloss.Style
		switch label {
		case "Installed", "Updated", "Success":
			style = SuccessText
		case "Failed", "Error":
			style = ErrorText
		case "Skipped", "Warning":
			style = WarningText
		default:
			style = Text
		}
		statusItems = append(statusItems, fmt.Sprintf("%s: %d", style.Render(label), count))
	}

	content.WriteString(strings.Join(statusItems, "  |  "))
	content.WriteString("\n")

	return content.String()
}

// RenderCommandHelp renders command help with consistent styling
func RenderCommandHelp(commands []string) string {
	var helpItems []string
	for _, cmd := range commands {
		helpItems = append(helpItems, TextBold.Render(cmd))
	}
	return strings.Join(helpItems, "  ")
}

// RenderSearchResults renders search results with consistent formatting
func RenderSearchResults(query string, count int) string {
	return RenderTitleWithSubtitle(
		"Font Search Results",
		fmt.Sprintf("Found %d fonts matching '%s'", count, TableSourceName.Render(query)),
	)
}

// RenderLoadingScreen renders a loading screen
func RenderLoadingScreen(message string) string {
	return fmt.Sprintf("\n%s\n\n%s\n\n%s",
		PageTitle.Render("FontGet"),
		Text.Render(message),
		TextBold.Render("Please wait..."),
	)
}

// RenderErrorScreen renders an error screen
func RenderErrorScreen(title, message string) string {
	return fmt.Sprintf("\n%s\n\n%s\n\n%s",
		PageTitle.Render(title),
		RenderError(message),
		TextBold.Render("Press 'Q' to quit"),
	)
}

// RenderSuccessScreen renders a success screen
func RenderSuccessScreen(title, message string) string {
	return fmt.Sprintf("\n%s\n\n%s\n\n%s",
		PageTitle.Render(title),
		RenderSuccess(message),
		TextBold.Render("Press 'Q' to quit"),
	)
}

// RunSpinner runs a pin spinner while the provided function executes
// Always stops with a green check symbol on success
// If doneMsg is empty string, the spinner line will be cleared (hidden) after completion
func RunSpinner(msg, doneMsg string, fn func() error) error {
	// Configure spinner with colors from styles.go
	p := pinpkg.New(msg,
		pinpkg.WithSpinnerColor(hexToPinColor(SpinnerColor)),
		pinpkg.WithDoneSymbol('✓'),
		pinpkg.WithDoneSymbolColor(hexToPinColor(SpinnerDoneColor)),
	)
	// Start spinner; it auto-disables animation when output is piped
	cancel := p.Start(context.Background())
	defer cancel()

	err := fn()
	if err != nil {
		// Show failure with red X, but return the error
		p.Fail("✗ " + err.Error())
		return err
	}
	// If doneMsg is empty, clear the spinner line instead of showing a message
	if doneMsg == "" {
		// Stop the spinner first (cancel() is idempotent, so defer will be safe)
		cancel()
		// Clear the line using ANSI escape code to ensure it's completely hidden
		// \r moves to start of line, \033[2K clears the entire line
		// Add a newline so subsequent output appears on a fresh line
		fmt.Print("\r\033[2K\n")
		return nil
	}
	p.Stop(doneMsg)
	// Add blank line after spinner completion (per spacing framework)
	fmt.Println()
	return nil
}

// hexToPinColor dynamically maps hex color strings to the closest matching pin package color
// Uses RGB distance calculation to find the nearest ANSI color
// This approach is dynamic and works with any hex color without requiring a static map
// Improved to include more ANSI color options for better theme color matching
func hexToPinColor(hex string) pinpkg.Color {
	if hex == "" {
		return pinpkg.ColorDefault
	}

	// Parse hex to RGB
	r, g, b := parseHexColor(hex)

	// Define extended ANSI colors as RGB values
	// Includes standard and bright variants for better color matching
	ansiColors := []struct {
		color   pinpkg.Color
		r, g, b int
	}{
		// Standard colors
		{pinpkg.ColorRed, 255, 0, 0},       // Red
		{pinpkg.ColorGreen, 0, 255, 0},     // Green
		{pinpkg.ColorYellow, 255, 255, 0},  // Yellow
		{pinpkg.ColorBlue, 0, 0, 255},      // Blue
		{pinpkg.ColorMagenta, 255, 0, 255}, // Magenta
		{pinpkg.ColorCyan, 0, 255, 255},    // Cyan
		// Note: The pin package may support additional colors, but we map to these standard ones
		// For better matching, we use perceptual color distance
	}

	// Find the closest color using perceptual color distance (weighted RGB)
	// This gives better results for human perception than simple Euclidean distance
	bestColor := pinpkg.ColorDefault
	minDistance := 999999.0 // Large initial value

	for _, ansi := range ansiColors {
		// Calculate weighted RGB distance (perceptual)
		// Red and green are weighted more heavily for human perception
		dr := float64(r - ansi.r)
		dg := float64(g - ansi.g)
		db := float64(b - ansi.b)
		// Weighted distance: red and green contribute more to perceived color difference
		distance := 0.3*dr*dr + 0.59*dg*dg + 0.11*db*db

		if distance < minDistance {
			minDistance = distance
			bestColor = ansi.color
		}
	}

	return bestColor
}

// SimpleProgressBar provides a simple inline progress bar without TUI
// It renders the title on one line and updates the progress bar inline using carriage returns
type SimpleProgressBar struct {
	title      string
	barWidth   int
	startColor string
	endColor   string
}

// NewSimpleProgressBar creates a new simple progress bar
func NewSimpleProgressBar(title string) *SimpleProgressBar {
	startColor, endColor := GetProgressBarGradient()
	return &SimpleProgressBar{
		title:      title,
		barWidth:   15,
		startColor: startColor,
		endColor:   endColor,
	}
}

// Run executes the operation and updates the progress bar
// The update function is called with a callback that accepts a percentage (0-100)
func (p *SimpleProgressBar) Run(operation func(update func(percent float64)) error) error {
	// Print the title on its own line
	fmt.Println(p.title)

	// Print initial empty progress bar line so updates don't overwrite the title
	fmt.Print("\n")

	// Run the operation with progress updates
	err := operation(func(percent float64) {
		p.update(percent)
	})

	// Add newline after progress bar (operation should have already called update(100.0))
	fmt.Println()

	return err
}

// update renders the progress bar inline using carriage return
func (p *SimpleProgressBar) update(percent float64) {
	// Clamp percent to 0-100
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	// Calculate filled and empty portions
	filled := int(float64(p.barWidth) * percent / 100.0)
	empty := p.barWidth - filled

	// Build the progress bar with gradient
	var bar strings.Builder

	// Filled portion with gradient
	for i := 0; i < filled; i++ {
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

		// Interpolate color
		color := interpolateHexColor(p.startColor, p.endColor, ratio)
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
		bar.WriteString(style.Render("█"))
	}

	// Empty portion
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086"))
	for i := 0; i < empty; i++ {
		bar.WriteString(emptyStyle.Render("░"))
	}

	// Render with carriage return to overwrite the line, with square brackets
	fmt.Fprintf(os.Stdout, "\r[%s] %3.0f%%", bar.String(), percent)
	os.Stdout.Sync()
}

// interpolateHexColor interpolates between two hex colors
func interpolateHexColor(startHex, endHex string, ratio float64) string {
	// Parse start color
	startR, startG, startB := parseHexColor(startHex)
	// Parse end color
	endR, endG, endB := parseHexColor(endHex)

	// Interpolate
	r := int(float64(startR) + (float64(endR)-float64(startR))*ratio)
	g := int(float64(startG) + (float64(endG)-float64(startG))*ratio)
	b := int(float64(startB) + (float64(endB)-float64(startB))*ratio)

	// Clamp to valid range
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

// parseHexColor parses a hex color string (e.g., "#ff00ff") into RGB values
func parseHexColor(hex string) (r, g, b int) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0
	}
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return r, g, b
}
