package cmd

import (
	"fmt"
	"strings"

	"fontget/internal/components"
	"fontget/internal/output"
	"fontget/internal/ui"
	"fontget/internal/update"
	"fontget/internal/version"

	"github.com/spf13/cobra"
)

var (
	updateCheckOnly bool
	updateAutoYes   bool
	updateVersion   string
)

var updateCmd = &cobra.Command{
	Use:          "update",
	Short:        "Update FontGet to the latest version",
	SilenceUsage: true,
	Long: `Check for updates and optionally install the latest version of FontGet.

By default, this command checks for updates and prompts for confirmation
before installing. Use flags to customize behavior.

The update system automatically:
- Checks GitHub Releases for the latest version
- Verifies checksums for security
- Handles binary replacement safely
- Rolls back on failure`,
	Example: `  fontget update              # Check and prompt for update
  fontget update --check      # Only check, don't install
  fontget update -y            # Auto-confirm update
  fontget update --version 1.2.3  # Update to specific version`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := GetLogger()
		if logger != nil {
			logger.Info("Starting update operation")
			logger.Info("Parameters - checkOnly: %t, autoYes: %t, version: %s", updateCheckOnly, updateAutoYes, updateVersion)
		}

		output.GetVerbose().Info("Starting FontGet update operation")
		output.GetDebug().State("Update command initiated")

		// If --version flag is set, update to specific version
		if updateVersion != "" {
			return handleUpdateToVersion(updateVersion, updateAutoYes)
		}

		// If --check flag is set, only check for updates
		if updateCheckOnly {
			return handleCheckOnly()
		}

		// Default: check and prompt for update
		return handleUpdateFlow(updateAutoYes)
	},
}

func init() {
	updateCmd.Flags().BoolVarP(&updateCheckOnly, "check", "c", false, "Only check for updates, don't install")
	updateCmd.Flags().BoolVarP(&updateAutoYes, "yes", "y", false, "Skip confirmation prompt and auto-confirm update")
	updateCmd.Flags().StringVar(&updateVersion, "version", "", "Update to specific version (e.g., 1.2.3)")

	rootCmd.AddCommand(updateCmd)
}

// handleCheckOnly only checks for updates without installing
func handleCheckOnly() error {
	logger := GetLogger()
	output.GetVerbose().Info("Checking for updates (check-only mode)")
	output.GetDebug().State("Calling update.CheckForUpdates()")

	result, err := update.CheckForUpdates()
	if err != nil {
		if logger != nil {
			logger.Error("Failed to check for updates: %v", err)
		}
		output.GetVerbose().Error("%v", err)
		output.GetDebug().Error("update.CheckForUpdates() failed: %v", err)
		fmt.Printf("%s\n", ui.FeedbackError.Render(fmt.Sprintf("Unable to check for updates: %v", err)))
		return fmt.Errorf("update check failed: %w", err)
	}

	if logger != nil {
		logger.Info("Update check complete - Available: %t, Current: %s, Latest: %s", result.Available, result.Current, result.Latest)
	}

	output.GetVerbose().Info("Update check complete")
	output.GetDebug().State("Update check result - Available: %t, Current: %s, Latest: %s, NeedsUpdate: %t", result.Available, result.Current, result.Latest, result.NeedsUpdate)

	if !result.Available {
		fmt.Printf("%s\n", ui.FeedbackInfo.Render("No releases found on GitHub."))
		return nil
	}

	if !result.NeedsUpdate {
		fmt.Printf("%s\n", ui.FeedbackSuccess.Render(fmt.Sprintf("FontGet is up to date (v%s)", result.Current)))
		return nil
	}

	// Show update available message
	fmt.Printf("\n%s\n", ui.PageTitle.Render("Update Available"))
	fmt.Printf("%s\n", ui.FeedbackInfo.Render(fmt.Sprintf("FontGet v%s is available (you have v%s).", result.Latest, result.Current)))
	fmt.Printf("%s\n", ui.FeedbackText.Render("Run 'fontget update' to upgrade."))

	// Show release notes if available
	if result.Release != nil && result.Release.ReleaseNotes != "" {
		notes := formatReleaseNotes(result.Release.ReleaseNotes)
		if notes != "" {
			fmt.Printf("\n%s\n", ui.PageSubtitle.Render("Release Notes:"))
			// Avoid trailing blank line after notes
			fmt.Printf("%s\n", ui.FeedbackText.Render(notes))
		}
	}

	return nil
}

// handleUpdateFlow handles the full update flow with confirmation
func handleUpdateFlow(autoYes bool) error {
	logger := GetLogger()
	output.GetVerbose().Info("Checking for updates")
	output.GetDebug().State("Calling update.CheckForUpdates()")

	// Show spinner while checking
	output.GetVerbose().Info("Checking for updates...")
	output.GetDebug().State("Starting update check")

	result, err := update.CheckForUpdates()
	if err != nil {
		if logger != nil {
			logger.Error("Failed to check for updates: %v", err)
		}
		output.GetVerbose().Error("%v", err)
		output.GetDebug().Error("update.CheckForUpdates() failed: %v", err)
		fmt.Printf("%s\n", ui.FeedbackError.Render(fmt.Sprintf("Unable to check for updates: %v", err)))
		return fmt.Errorf("update check failed: %w", err)
	}

	if logger != nil {
		logger.Info("Update check complete - Available: %t, Current: %s, Latest: %s", result.Available, result.Current, result.Latest)
	}

	output.GetVerbose().Info("Update check complete")
	output.GetDebug().State("Update check result - Available: %t, Current: %s, Latest: %s, NeedsUpdate: %t", result.Available, result.Current, result.Latest, result.NeedsUpdate)

	if !result.Available {
		fmt.Printf("%s\n", ui.FeedbackInfo.Render("No releases found on GitHub."))
		return nil
	}

	if !result.NeedsUpdate {
		fmt.Printf("%s\n", ui.FeedbackSuccess.Render(fmt.Sprintf("FontGet is up to date (v%s)", result.Current)))
		return nil
	}

	// Show update information
	fmt.Printf("%s\n", ui.FeedbackInfo.Render(fmt.Sprintf("Current Installed Version: %s", result.Current)))
	fmt.Printf("%s\n", ui.FeedbackInfo.Render(fmt.Sprintf("Version Available:        %s", result.Latest)))

	// Show release notes if available
	if result.Release != nil && result.Release.ReleaseNotes != "" {
		notes := formatReleaseNotes(result.Release.ReleaseNotes)
		if notes != "" {
			fmt.Printf("\n%s\n", ui.PageSubtitle.Render("Release Notes:"))
			// Avoid trailing blank line after notes
			fmt.Printf("%s\n", ui.FeedbackText.Render(notes))
		}
	}

	// Prompt for confirmation unless auto-yes
	if !autoYes {
		output.GetVerbose().Info("Prompting for update confirmation")
		output.GetDebug().State("Showing confirmation dialog")

		confirmed, err := components.RunConfirm(
			"Update FontGet",
			fmt.Sprintf("Do you want to update FontGet from %s to %s?", result.Current, result.Latest),
		)
		if err != nil {
			if logger != nil {
				logger.Error("Confirmation dialog failed: %v", err)
			}
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("components.RunConfirm() failed: %v", err)
			fmt.Printf("%s\n", ui.FeedbackError.Render("Confirmation dialog failed"))
			return fmt.Errorf("unable to show confirmation dialog: %w", err)
		}

		if !confirmed {
			if logger != nil {
				logger.Info("User cancelled update")
			}
			output.GetVerbose().Info("User cancelled update")
			output.GetDebug().State("User chose not to update")
			fmt.Printf("%s\n", ui.FeedbackWarning.Render("Update cancelled. No changes have been made."))
			return nil
		}
	} else {
		output.GetVerbose().Info("Auto-confirming update (--yes flag)")
		output.GetDebug().State("Skipping confirmation (auto-yes)")
	}

	// Perform the update
	return performUpdate(result.Current, result.Latest)
}

// handleUpdateToVersion handles update to a specific version
func handleUpdateToVersion(targetVersion string, autoYes bool) error {
	logger := GetLogger()
	if logger != nil {
		logger.Info("Updating to specific version: %s", targetVersion)
	}

	output.GetVerbose().Info("Updating to version: %s", targetVersion)
	output.GetDebug().State("Target version: %s", targetVersion)

	currentVersion := version.GetVersion()

	// Check if target version is different from current
	if targetVersion == currentVersion {
		fmt.Printf("%s\n", ui.FeedbackInfo.Render(fmt.Sprintf("FontGet is already at version %s", currentVersion)))
		return nil
	}

	// Show update information
	fmt.Printf("\n%s\n", ui.PageTitle.Render("Update to Specific Version"))
	fmt.Printf("%s\n", ui.FeedbackInfo.Render(fmt.Sprintf("Current Installed Version: %s", currentVersion)))
	fmt.Printf("%s\n", ui.FeedbackInfo.Render(fmt.Sprintf("Version Available:        %s", targetVersion)))

	// Prompt for confirmation unless auto-yes
	if !autoYes {
		output.GetVerbose().Info("Prompting for update confirmation")
		output.GetDebug().State("Showing confirmation dialog")

		confirmed, err := components.RunConfirm(
			"Update FontGet",
			fmt.Sprintf("Do you want to update FontGet from %s to %s?", currentVersion, targetVersion),
		)
		if err != nil {
			if logger != nil {
				logger.Error("Confirmation dialog failed: %v", err)
			}
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("components.RunConfirm() failed: %v", err)
			fmt.Printf("%s\n", ui.FeedbackError.Render("Confirmation dialog failed"))
			return fmt.Errorf("unable to show confirmation dialog: %w", err)
		}

		if !confirmed {
			if logger != nil {
				logger.Info("User cancelled update")
			}
			output.GetVerbose().Info("User cancelled update")
			output.GetDebug().State("User chose not to update")
			fmt.Printf("%s\n", ui.FeedbackWarning.Render("Update cancelled. No changes have been made."))
			return nil
		}
	} else {
		output.GetVerbose().Info("Auto-confirming update (--yes flag)")
		output.GetDebug().State("Skipping confirmation (auto-yes)")
	}

	// Perform the update to specific version
	output.GetVerbose().Info("Starting update to version %s", targetVersion)
	output.GetDebug().State("Calling update.UpdateToVersion(%s)", targetVersion)

	err := ui.RunSpinner(
		fmt.Sprintf("Updating FontGet from %s to %s...", currentVersion, targetVersion),
		fmt.Sprintf("Successfully updated to FontGet %s", targetVersion),
		func() error {
			return update.UpdateToVersion(targetVersion)
		},
	)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to update to version %s: %v", targetVersion, err)
		}
		output.GetVerbose().Error("%v", err)
		output.GetDebug().Error("update.UpdateToVersion() failed: %v", err)
		fmt.Printf("%s\n", ui.FeedbackError.Render(fmt.Sprintf("Update failed: %v", err)))
		return fmt.Errorf("update failed: %w", err)
	}

	if logger != nil {
		logger.Info("Successfully updated to version %s", targetVersion)
	}

	output.GetVerbose().Info("Update complete")
	output.GetDebug().State("Update successful")
	fmt.Printf("%s\n", ui.FeedbackSuccess.Render(fmt.Sprintf("Successfully updated to FontGet %s", targetVersion)))

	return nil
}

// performUpdate performs the actual update operation
func performUpdate(currentVersion, latestVersion string) error {
	logger := GetLogger()
	output.GetVerbose().Info("Starting update to version %s", latestVersion)
	output.GetDebug().State("Calling update.UpdateToLatest()")

	// Show spinner while performing the update
	err := ui.RunSpinner(
		fmt.Sprintf("Updating FontGet from %s to %s...", currentVersion, latestVersion),
		fmt.Sprintf("Successfully updated to FontGet %s", latestVersion),
		func() error {
			return update.UpdateToLatest()
		},
	)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to update: %v", err)
		}
		output.GetVerbose().Error("%v", err)
		output.GetDebug().Error("update.UpdateToLatest() failed: %v", err)
		fmt.Printf("%s\n", ui.FeedbackError.Render(fmt.Sprintf("Update failed: %v", err)))
		return fmt.Errorf("update failed: %w", err)
	}

	if logger != nil {
		logger.Info("Successfully updated from %s to %s", currentVersion, latestVersion)
	}

	output.GetVerbose().Info("Update complete")
	output.GetDebug().State("Update successful")
	fmt.Printf("%s\n", ui.FeedbackSuccess.Render(fmt.Sprintf("Successfully updated to FontGet %s", latestVersion)))

	return nil
}

// formatReleaseNotes formats release notes for display (first 10 lines)
func formatReleaseNotes(notes string) string {
	if notes == "" {
		return ""
	}

	lines := strings.Split(notes, "\n")

	// Trim trailing empty lines to avoid extra blank space at the end
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	maxLines := 10
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		return strings.Join(lines, "\n") + "\n..."
	}

	return strings.Join(lines, "\n")
}
