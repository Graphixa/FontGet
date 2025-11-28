package cmdutils

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"fontget/internal/platform"
	"fontget/internal/ui"

	"github.com/spf13/cobra"
)

// ErrElevationRequired is a sentinel error used to indicate we've already printed
// user-facing elevation instructions and no further error output is needed.
var ErrElevationRequired = errors.New("elevation required")

// PrintElevationHelp prints platform-specific elevation instructions
func PrintElevationHelp(cmd *cobra.Command, platform string) {
	// No leading blank line - commands already start with a blank line per spacing framework

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

// CheckElevation checks if the current process has elevated privileges
// and prints help if elevation is required but not present
func CheckElevation(cmd *cobra.Command, fontManager platform.FontManager, scope platform.InstallationScope) error {
	if fontManager.RequiresElevation(scope) {
		// Check if already elevated
		elevated, err := fontManager.IsElevated()
		if err != nil {
			return fmt.Errorf("failed to check elevation status: %w", err)
		}

		if !elevated {
			// Print help message
			PrintElevationHelp(cmd, runtime.GOOS)
			// Prevent Cobra and callers from printing duplicate error messages
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			return ErrElevationRequired
		}
	}
	return nil
}
