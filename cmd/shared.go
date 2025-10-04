package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
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

// ErrElevationRequired is a sentinel error used to indicate we've already printed
// user-facing elevation instructions and no further error output is needed.
var ErrElevationRequired = errors.New("elevation required")

// printElevationHelp prints platform-specific elevation instructions
func printElevationHelp(cmd *cobra.Command, platform string) {
	fmt.Println()

	// Build the exact command the user ran (prefer root name over executable filename)
	fullCmd := strings.TrimSpace(fmt.Sprintf("%s %s", cmd.Root().Name(), strings.Join(os.Args[1:], " ")))

	switch platform {
	case "windows":
		// Error line in red
		cmd.Println(ui.FeedbackError.Render("This operation requires administrator privileges."))
		fmt.Println()
		// Guidance in normal feedback text
		cmd.Println(ui.FeedbackText.Render("To run as administrator:"))
		cmd.Println(ui.FeedbackText.Render("  1. Right-click on Command Prompt or PowerShell."))
		cmd.Println(ui.FeedbackText.Render("  2. Select 'Run as administrator'."))
		cmd.Println(ui.FeedbackText.Render(fmt.Sprintf("  3. Run: %s", fullCmd)))
	case "darwin", "linux":
		// Error line in red
		cmd.Println(ui.FeedbackError.Render("This operation requires root privileges."))
		fmt.Println()
		// Guidance in normal feedback text
		cmd.Println(ui.FeedbackText.Render("To run as root, prepend 'sudo' to your command, for example:"))
		cmd.Println(ui.FeedbackText.Render(fmt.Sprintf("  sudo %s", fullCmd)))
	default:
		cmd.Println(ui.FeedbackError.Render("This operation requires elevated privileges. Please re-run as administrator or root."))
	}
	fmt.Println()
}

// checkElevation checks if the current process has elevated privileges
// and prints help if elevation is required but not present
func checkElevation(cmd *cobra.Command, fontManager platform.FontManager, scope platform.InstallationScope) error {
	if fontManager.RequiresElevation(scope) {
		// Check if already elevated
		elevated, err := fontManager.IsElevated()
		if err != nil {
			return fmt.Errorf("failed to check elevation status: %w", err)
		}

		if !elevated {
			// Print help message
			printElevationHelp(cmd, runtime.GOOS)
			// Prevent Cobra and callers from printing duplicate error messages
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			return ErrElevationRequired
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
		// Do not duplicate user-facing elevation errors; we've already printed instructions
		if errors.Is(err, ErrElevationRequired) {
			p.Stop("")
			return err
		}
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

// runProgressBarWithOptions runs a progress bar with configurable options
func runProgressBarWithOptions(msg string, totalSteps int, fn func(updateProgress func()) error, hideWhenFinished bool, showHeader bool) error {
	return components.RunWithProgressOptions(msg, totalSteps, fn, hideWhenFinished, showHeader)
}

// Table formatting constants for consistent table widths across all commands
const (
	// Font Search/Add/Remove Tables (5 columns, total: 120 chars - uses full 120-char terminals)
	TableColName       = 36 // Font name (wider for longer names)
	TableColID         = 34 // Font ID (wider for longer IDs like "nerd.font-name")
	TableColLicense    = 12 // License (slightly wider)
	TableColCategories = 16 // Categories (wider for multiple categories)
	TableColSource     = 18 // Source (wider for source names)

	// Font List Tables (5 columns, total: 120 chars)
	TableColListName = 54 // Font family name (wider)
	TableColStyle    = 22 // Font style/variant (wider)
	TableColType     = 10 // File type
	TableColDate     = 20 // Installation date
	TableColScope    = 10 // Scope (user/machine)

	// Sources Management Tables (2 columns, total: 120 chars)
	TableColStatus     = 10  // Checkbox/status
	TableColSourceName = 109 // Source name with tags (much wider)

	// Total table width (uses full 120-char terminals for maximum space utilization)
	TableTotalWidth = 120
)

// GetSearchTableHeader returns a formatted table header for font search/add/remove tables
func GetSearchTableHeader() string {
	return fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s",
		TableColName, "Name",
		TableColID, "ID",
		TableColLicense, "License",
		TableColCategories, "Categories",
		TableColSource, "Source")
}

// GetDynamicSearchTableHeader returns a table header with dynamic column widths based on data
func GetDynamicSearchTableHeader(names, ids, licenses, categories, sources []string) string {
	// Calculate maximum widths needed for each column
	maxName := TableColName
	maxID := TableColID
	maxLicense := TableColLicense
	maxCategories := TableColCategories
	maxSource := TableColSource

	// Check all data arrays
	for _, name := range names {
		if len(name) > maxName {
			maxName = len(name)
		}
	}
	for _, id := range ids {
		if len(id) > maxID {
			maxID = len(id)
		}
	}
	for _, license := range licenses {
		if len(license) > maxLicense {
			maxLicense = len(license)
		}
	}
	for _, category := range categories {
		if len(category) > maxCategories {
			maxCategories = len(category)
		}
	}
	for _, source := range sources {
		if len(source) > maxSource {
			maxSource = len(source)
		}
	}

	// Calculate total width needed
	totalWidth := maxName + maxID + maxLicense + maxCategories + maxSource + 4 // +4 for spaces

	// If total width exceeds reasonable maximum (160 chars), use fixed widths
	if totalWidth > 160 {
		return GetSearchTableHeader()
	}

	// Return dynamic header
	return fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s",
		maxName, "Name",
		maxID, "ID",
		maxLicense, "License",
		maxCategories, "Categories",
		maxSource, "Source")
}

// GetDynamicSearchTableSeparator returns a separator line with dynamic width
func GetDynamicSearchTableSeparator(names, ids, licenses, categories, sources []string) string {
	// Calculate maximum widths needed for each column
	maxName := TableColName
	maxID := TableColID
	maxLicense := TableColLicense
	maxCategories := TableColCategories
	maxSource := TableColSource

	// Check all data arrays
	for _, name := range names {
		if len(name) > maxName {
			maxName = len(name)
		}
	}
	for _, id := range ids {
		if len(id) > maxID {
			maxID = len(id)
		}
	}
	for _, license := range licenses {
		if len(license) > maxLicense {
			maxLicense = len(license)
		}
	}
	for _, category := range categories {
		if len(category) > maxCategories {
			maxCategories = len(category)
		}
	}
	for _, source := range sources {
		if len(source) > maxSource {
			maxSource = len(source)
		}
	}

	// Calculate total width needed
	totalWidth := maxName + maxID + maxLicense + maxCategories + maxSource + 4 // +4 for spaces

	// If total width exceeds reasonable maximum (160 chars), use fixed width
	if totalWidth > 160 {
		return GetTableSeparator()
	}

	return strings.Repeat("-", totalWidth)
}

// GetTableSeparator returns a table separator line with consistent width
func GetTableSeparator() string {
	return strings.Repeat("-", TableTotalWidth)
}

// GetListTableHeader returns a formatted table header for font list tables
func GetListTableHeader() string {
	return fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s",
		TableColListName, "Name",
		TableColStyle, "Style",
		TableColType, "Type",
		TableColDate, "Installed",
		TableColScope, "Scope")
}

// GetSourcesTableHeader returns a formatted table header for sources management tables
func GetSourcesTableHeader() string {
	return fmt.Sprintf("%-*s %-*s",
		TableColStatus, "Status",
		TableColSourceName, "Name")
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
