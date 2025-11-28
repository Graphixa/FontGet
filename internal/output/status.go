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
// isVerbose should be passed from the cmd package's IsVerbose() function to avoid circular dependencies.
func PrintStatusReport(report StatusReport, isVerbose bool) {
	// Only show status report in verbose mode
	if isVerbose && (report.Success > 0 || report.Skipped > 0 || report.Failed > 0) {
		// No leading blank line - previous section already ends with blank line per spacing framework
		fmt.Printf("%s\n", ui.ReportTitle.Render("Status Report"))
		fmt.Println("---------------------------------------------")
		fmt.Printf("%s: %d  |  %s: %d  |  %s: %d\n\n",
			ui.FeedbackSuccess.Render(report.SuccessLabel), report.Success,
			ui.FeedbackWarning.Render(report.SkippedLabel), report.Skipped,
			ui.FeedbackError.Render(report.FailedLabel), report.Failed)
	}
}
