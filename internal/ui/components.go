package ui

import (
	"context"
	"fmt"
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
	// Add blank line after spinner completion (per spacing framework)
	fmt.Println()
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
