package output

import (
	"fmt"

	"fontget/internal/ui"
)

// StatusReport represents a status report for operations
type StatusReport struct {
	Success      int
	Skipped      int
	Failed       int
	SuccessLabel string
	SkippedLabel string
	FailedLabel  string
}

// PrintStatusReport prints a formatted status report if there were actual operations.
// Pass output.IsVerboseOutputEnabled() (verbose && !debug) so the report stays hidden in debug mode.
func PrintStatusReport(report StatusReport, isVerbose bool) {
	// Only show status report when styled verbose output is active
	if isVerbose && (report.Success > 0 || report.Skipped > 0 || report.Failed > 0) {
		// Blank line after Bubble Tea / progress output (previous section may not end with a visual newline)
		fmt.Println()
		fmt.Printf("%s\n", ui.TextBold.Render("Status Report"))
		fmt.Println("---------------------------------------------")
		fmt.Printf("%s: %d  |  %s: %d  |  %s: %d\n\n",
			ui.SuccessText.Render(report.SuccessLabel), report.Success,
			ui.WarningText.Render(report.SkippedLabel), report.Skipped,
			ui.ErrorText.Render(report.FailedLabel), report.Failed)
	}
}
