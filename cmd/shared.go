package cmd

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"fontget/internal/components"
	"fontget/internal/platform"
	"fontget/internal/ui"

	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	pinpkg "github.com/yarlson/pin"
)

// printElevationHelp prints platform-specific elevation instructions
func printElevationHelp(cmd *cobra.Command, platform string) {
	fmt.Println()
	switch platform {
	case "windows":
		cmd.Println("This operation requires administrator privileges.")
		cmd.Println("To run as administrator:")
		cmd.Println("  1. Right-click on Command Prompt or PowerShell.")
		cmd.Println("  2. Select 'Run as administrator'.")
		cmd.Printf("  3. Run: %s\n", cmd.CommandPath())
	case "darwin", "linux":
		cmd.Println("This operation requires root privileges.")
		cmd.Println("To run as root, prepend 'sudo' to your command, for example:")
		cmd.Printf("  sudo %s\n", cmd.CommandPath())
	default:
		cmd.Println("This operation requires elevated privileges. Please re-run as administrator or root.")
	}
	fmt.Println()
}

// checkElevation checks if the current process has elevated privileges
// and prints help if elevation is required but not present
func checkElevation(cmd *cobra.Command, fontManager platform.FontManager, scope platform.InstallationScope) error {
	if fontManager.RequiresElevation(scope) {
		// Create elevation manager
		elevationManager, err := platform.NewElevationManager()
		if err != nil {
			return fmt.Errorf("failed to initialize elevation manager: %w", err)
		}

		// Check if already elevated
		elevated, err := elevationManager.IsElevated()
		if err != nil {
			return fmt.Errorf("failed to check elevation status: %w", err)
		}

		if !elevated {
			// Print help message
			printElevationHelp(cmd, runtime.GOOS)
			return fmt.Errorf("this command requires elevated privileges. Please follow the instructions above to re-run as administrator/root")
		}
	}
	return nil
}

// Common color functions to reduce duplication across commands
var (
	// Basic colors
	Red    = color.New(color.FgRed).SprintFunc()
	Green  = color.New(color.FgGreen).SprintFunc()
	Yellow = color.New(color.FgYellow).SprintFunc()
	Cyan   = color.New(color.FgCyan).SprintFunc()
	Bold   = color.New(color.Bold).SprintFunc()
	White  = color.New(color.FgWhite).SprintFunc()

	// Combined colors
	BoldRed    = color.New(color.FgRed, color.Bold).SprintFunc()
	BoldGreen  = color.New(color.FgGreen, color.Bold).SprintFunc()
	BoldYellow = color.New(color.FgYellow, color.Bold).SprintFunc()
	BoldCyan   = color.New(color.FgCyan, color.Bold).SprintFunc()
)

// GetColorFunctions returns a map of commonly used color functions
func GetColorFunctions() map[string]func(a ...interface{}) string {
	return map[string]func(a ...interface{}) string{
		"red":        Red,
		"green":      Green,
		"yellow":     Yellow,
		"cyan":       Cyan,
		"bold":       Bold,
		"white":      White,
		"boldRed":    BoldRed,
		"boldGreen":  BoldGreen,
		"boldYellow": BoldYellow,
		"boldCyan":   BoldCyan,
	}
}

// StatusReport represents a status report for operations
type StatusReport struct {
	Success      int
	Skipped      int
	Failed       int
	SuccessLabel string
	SkippedLabel string
	FailedLabel  string
}

// PrintStatusReport prints a formatted status report if there were actual operations
func PrintStatusReport(report StatusReport) {
	if report.Success > 0 || report.Skipped > 0 || report.Failed > 0 {
		fmt.Printf("\n%s\n", ui.ReportTitle.Render("Status Report"))
		fmt.Println("---------------------------------------------")
		fmt.Printf("%s: %d  |  %s: %d  |  %s: %d\n\n",
			ui.FeedbackSuccess.Render(report.SuccessLabel), report.Success,
			ui.FeedbackWarning.Render(report.SkippedLabel), report.Skipped,
			ui.FeedbackError.Render(report.FailedLabel), report.Failed)
	}
}

// ParseFontNames parses comma-separated font names from command line arguments
func ParseFontNames(args []string) []string {
	var fontNames []string
	for _, arg := range args {
		// Split by comma and trim whitespace
		names := strings.Split(arg, ",")
		for _, name := range names {
			name = strings.TrimSpace(name)
			if name != "" {
				fontNames = append(fontNames, name)
			}
		}
	}
	return fontNames
}

// Custom error types for better error handling

// FontNotFoundError represents when a font is not found
type FontNotFoundError struct {
	FontName    string
	Suggestions []string
}

func (e *FontNotFoundError) Error() string {
	if len(e.Suggestions) > 0 {
		return fmt.Sprintf("font '%s' not found. Did you mean: %s?", e.FontName, strings.Join(e.Suggestions, ", "))
	}
	return fmt.Sprintf("font '%s' not found", e.FontName)
}

// FontInstallationError represents font installation failures
type FontInstallationError struct {
	FailedCount int
	TotalCount  int
	Details     []string
}

func (e *FontInstallationError) Error() string {
	return fmt.Sprintf("failed to install %d out of %d fonts", e.FailedCount, e.TotalCount)
}

// FontRemovalError represents font removal failures
type FontRemovalError struct {
	FailedCount int
	TotalCount  int
	Details     []string
}

func (e *FontRemovalError) Error() string {
	return fmt.Sprintf("failed to remove %d out of %d fonts", e.FailedCount, e.TotalCount)
}

// ConfigurationError represents configuration-related errors
type ConfigurationError struct {
	Field string
	Value string
	Hint  string
}

func (e *ConfigurationError) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("configuration error in field '%s' with value '%s': %s", e.Field, e.Value, e.Hint)
	}
	return fmt.Sprintf("configuration error in field '%s' with value '%s'", e.Field, e.Value)
}

// ElevationError represents elevation-related errors
type ElevationError struct {
	Operation string
	Platform  string
}

func (e *ElevationError) Error() string {
	return fmt.Sprintf("elevation required for operation '%s' on platform '%s'", e.Operation, e.Platform)
}

// runSpinner runs a lightweight spinner while fn executes. Always stops with a green check style.
func runSpinner(msg, doneMsg string, fn func() error) error {
	// Pre-style the message with lipgloss using Pink color for font names
	styledMsg := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#f5c2e7")). // Pink - matches TableSourceName
		Bold(true).
		Render(msg)

	// Configure spinner with pre-styled message
	p := pinpkg.New(styledMsg,
		pinpkg.WithSpinnerColor(pinkToPin("#cba6f7")),    // spinner mauve
		pinpkg.WithDoneSymbol(' '),                       // No done symbol - we'll handle it ourselves
		pinpkg.WithDoneSymbolColor(pinkToPin("#a6e3a1")), // green check
	)

	// Start spinner; it auto-disables animation when output is piped
	cancel := p.Start(context.Background())
	defer cancel()

	err := fn()
	if err != nil {
		// Show failure with red X, but return the error
		p.Fail(err.Error())
		return err
	}
	// Style the completion message with the same color
	styledDoneMsg := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6adc8")). // Gray
		Render(doneMsg)

	if doneMsg == "" {
		styledDoneMsg = styledMsg
	}
	p.Stop(styledDoneMsg)
	return nil
}

// pinkToPin maps hex-ish choice to nearest pin color (simple mapping to keep code local)
func pinkToPin(hex string) pinpkg.Color {
	switch strings.ToLower(hex) {
	case "#a6e3a1":
		return pinpkg.ColorGreen
	case "#cba6f7":
		return pinpkg.ColorMagenta
	case "#b4befe":
		return pinpkg.ColorBlue
	case "#a6adc8":
		return pinpkg.ColorCyan
	default:
		return pinpkg.ColorDefault
	}
}

// runProgressBar runs a progress bar using the UI component
func runProgressBar(msg string, totalSteps int, fn func(updateProgress func()) error) error {
	return components.RunWithProgress(msg, totalSteps, fn)
}
