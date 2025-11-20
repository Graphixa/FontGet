package ui

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	pinpkg "github.com/yarlson/pin"
)

// Utility functions for consistent UI rendering

// RenderTitleWithSubtitle renders a title with optional subtitle
func RenderTitleWithSubtitle(title, subtitle string) string {
	if subtitle == "" {
		return PageTitle.Render(title) + "\n"
	}
	return PageTitle.Render(title) + "\n" +
		FeedbackText.Render(subtitle) + "\n"
}

// RenderStatusReport renders a status report with consistent styling
func RenderStatusReport(title string, items map[string]int) string {
	var content strings.Builder

	content.WriteString("\n")
	content.WriteString(ReportTitle.Render(title))
	content.WriteString("\n")
	content.WriteString("---------------------------------------------")
	content.WriteString("\n")

	// Status items
	var statusItems []string
	for label, count := range items {
		var style lipgloss.Style
		switch label {
		case "Installed", "Updated", "Success":
			style = FeedbackSuccess
		case "Failed", "Error":
			style = FeedbackError
		case "Skipped", "Warning":
			style = FeedbackWarning
		default:
			style = ContentText
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
		helpItems = append(helpItems, CommandLabel.Render(cmd))
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
		FeedbackText.Render(message),
		CommandLabel.Render("Please wait..."),
	)
}

// RenderErrorScreen renders an error screen
func RenderErrorScreen(title, message string) string {
	return fmt.Sprintf("\n%s\n\n%s\n\n%s",
		PageTitle.Render(title),
		RenderError(message),
		CommandLabel.Render("Press 'Q' to quit"),
	)
}

// RenderSuccessScreen renders a success screen
func RenderSuccessScreen(title, message string) string {
	return fmt.Sprintf("\n%s\n\n%s\n\n%s",
		PageTitle.Render(title),
		RenderSuccess(message),
		CommandLabel.Render("Press 'Q' to quit"),
	)
}

// RunSpinner runs a pin spinner while the provided function executes
// Always stops with a green check symbol on success
func RunSpinner(msg, doneMsg string, fn func() error) error {
	// Configure spinner with colors from styles.go
	// The pin package auto-detects terminal capabilities and disables colors when output is piped
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
	// Use plain text for completion message (no styling)
	if doneMsg == "" {
		doneMsg = msg
	}
	p.Stop(doneMsg)
	return nil
}

// hexToPinColor maps hex color strings to pin package color constants
// Uses PinColorMap from styles.go for the color mapping
// The pin package uses its own color constants and doesn't accept hex strings directly
func hexToPinColor(hex string) pinpkg.Color {
	hexLower := strings.ToLower(hex)
	colorName, exists := PinColorMap[hexLower]
	if !exists {
		return pinpkg.ColorDefault
	}

	switch colorName {
	case "green":
		return pinpkg.ColorGreen
	case "magenta":
		return pinpkg.ColorMagenta
	case "blue":
		return pinpkg.ColorBlue
	case "cyan":
		return pinpkg.ColorCyan
	default:
		return pinpkg.ColorDefault
	}
}

// ResetTerminalAfterBubbleTea resets terminal state and reinitializes lipgloss color profile
// after a Bubble Tea program exits. This prevents escape sequences from being printed literally
// and ensures colors render correctly.
//
// Bubble Tea may change terminal state, so this function:
// 1. Resets all terminal attributes
// 2. Flushes output to ensure proper formatting
// 3. Reinitializes lipgloss to use ANSI256 color profile for better compatibility
//
// Call this function after RunProgressBar or any other Bubble Tea program exits, before
// printing any styled output with lipgloss.
func ResetTerminalAfterBubbleTea() {
	// Reset all terminal attributes and move to new line
	fmt.Print("\033[0m\n")
	// Flush output to ensure proper formatting
	os.Stdout.Sync()

	// Reinitialize lipgloss color profile after Bubble Tea exits
	// Force ANSI256 color profile for better compatibility (avoids RGB codes that some terminals don't support)
	lipgloss.SetColorProfile(termenv.ANSI256)
}

// NewWarningStyleAfterBubbleTea creates a fresh warning style that works correctly
// after Bubble Tea exits. Uses ANSI256 color index instead of RGB codes for better
// terminal compatibility.
//
// This should be called after ResetTerminalAfterBubbleTea() to ensure the style
// uses the correct color profile.
func NewWarningStyleAfterBubbleTea() func(string) string {
	// Create a fresh warning style using ANSI256 color index (yellow, index 11)
	// This avoids RGB color codes that some terminals don't support
	// ANSI256 color 11 is a bright yellow that works well for warnings
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("11")) // ANSI256 bright yellow (index 11)
	return func(s string) string {
		return style.Render(s)
	}
}
