package cmd

import (
	"fmt"
	"strings"

	"fontget/internal/components"
	"fontget/internal/output"
	"fontget/internal/ui"
	"fontget/internal/update"
	"fontget/internal/version"

	"github.com/blang/semver"
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
		fmt.Printf("%s\n", ui.ErrorText.Render(fmt.Sprintf("Unable to check for updates: %v", err)))
		return fmt.Errorf("update check failed: %w", err)
	}

	if logger != nil {
		logger.Info("Update check complete - Available: %t, Current: %s, Latest: %s", result.Available, result.Current, result.Latest)
	}

	output.GetVerbose().Info("Update check complete")
	output.GetDebug().State("Update check result - Available: %t, Current: %s, Latest: %s, NeedsUpdate: %t", result.Available, result.Current, result.Latest, result.NeedsUpdate)

	if !result.Available {
		fmt.Printf("%s\n", ui.InfoText.Render("No releases found on GitHub."))
		return nil
	}

	if !result.NeedsUpdate {
		fmt.Printf("%s\n", ui.SuccessText.Render(fmt.Sprintf("FontGet is up to date (v%s)", result.Current)))
		return nil
	}

	// Show update information (no PageTitle for non-TUI commands)
	fmt.Printf("%s\n", ui.InfoText.Render(fmt.Sprintf("Installed Version: v%s", result.Current)))
	fmt.Printf("%s\n", ui.InfoText.Render(fmt.Sprintf("Update Version: v%s", result.Latest)))

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
		fmt.Printf("%s\n", ui.ErrorText.Render(fmt.Sprintf("Unable to check for updates: %v", err)))
		return fmt.Errorf("update check failed: %w", err)
	}

	if logger != nil {
		logger.Info("Update check complete - Available: %t, Current: %s, Latest: %s", result.Available, result.Current, result.Latest)
	}

	output.GetVerbose().Info("Update check complete")
	output.GetDebug().State("Update check result - Available: %t, Current: %s, Latest: %s, NeedsUpdate: %t", result.Available, result.Current, result.Latest, result.NeedsUpdate)

	if !result.Available {
		fmt.Printf("%s\n", ui.InfoText.Render("No releases found on GitHub."))
		return nil
	}

	if !result.NeedsUpdate {
		fmt.Printf("%s\n", ui.SuccessText.Render(fmt.Sprintf("FontGet is up to date (v%s)", result.Current)))
		return nil
	}

	// Show update information (no PageTitle for non-TUI commands)
	fmt.Printf("%s\n", ui.InfoText.Render(fmt.Sprintf("Installed Version: v%s", result.Current)))
	fmt.Printf("%s\n", ui.InfoText.Render(fmt.Sprintf("Version Available: v%s", result.Latest)))

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
			fmt.Printf("%s\n", ui.ErrorText.Render("Confirmation dialog failed"))
			return fmt.Errorf("unable to show confirmation dialog: %w", err)
		}

		if !confirmed {
			if logger != nil {
				logger.Info("User cancelled update")
			}
			output.GetVerbose().Info("User cancelled update")
			output.GetDebug().State("User chose not to update")
			fmt.Printf("%s\n", ui.WarningText.Render("Update cancelled. No changes have been made."))
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

	// If target equals current, treat as no-op and exit early
	if targetVersion == currentVersion {
		fmt.Printf("%s\n", ui.InfoText.Render(fmt.Sprintf("FontGet is already at v%s (no changes made).", currentVersion)))
		return nil
	}

	// Parse versions to detect downgrades (ignore parse errors gracefully)
	isDowngrade := false
	if curr, errCurr := semver.Parse(currentVersion); errCurr == nil {
		if tgt, errTgt := semver.Parse(targetVersion); errTgt == nil {
			if tgt.LT(curr) {
				isDowngrade = true
			}
		}
	}

	// Show update information (no PageTitle for non-TUI commands)
	fmt.Printf("%s\n", ui.InfoText.Render(fmt.Sprintf("Installed Version: v%s", currentVersion)))
	fmt.Printf("%s\n", ui.InfoText.Render(fmt.Sprintf("Version Available: v%s", targetVersion)))

	// Warn on downgrade
	if isDowngrade {
		fmt.Printf("%s\n", ui.RenderWarning("Warning: You are attempting to downgrade FontGet. This may cause configuration or cache issues."))
	}

	// For downgrades, always require confirmation (ignore --yes)
	mustConfirm := isDowngrade || !autoYes
	if mustConfirm {
		output.GetVerbose().Info("Prompting for update confirmation")
		output.GetDebug().State("Showing confirmation dialog")

		confirmTitle := "Update FontGet"
		confirmMessage := fmt.Sprintf("Do you want to update FontGet from v%s to v%s?", currentVersion, targetVersion)
		if isDowngrade {
			confirmTitle = "Downgrade FontGet"
			confirmMessage = fmt.Sprintf(
				"You are about to downgrade FontGet from v%s to v%s.\n\nThis may cause:\n- Configuration incompatibilities\n- Cache or index issues\n- Behavior changes in scripts using new flags\n\nDo you want to continue?",
				currentVersion, targetVersion,
			)
		}

		confirmed, err := components.RunConfirm(
			confirmTitle,
			confirmMessage,
		)
		if err != nil {
			if logger != nil {
				logger.Error("Confirmation dialog failed: %v", err)
			}
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("components.RunConfirm() failed: %v", err)
			fmt.Printf("%s\n", ui.ErrorText.Render("Confirmation dialog failed"))
			return fmt.Errorf("unable to show confirmation dialog: %w", err)
		}

		if !confirmed {
			if logger != nil {
				logger.Info("User cancelled update")
			}
			output.GetVerbose().Info("User cancelled update")
			output.GetDebug().State("User chose not to update")
			fmt.Printf("%s\n", ui.WarningText.Render("Update cancelled. No changes have been made."))
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
		fmt.Sprintf("Updating FontGet from v%s to v%s...", currentVersion, targetVersion),
		fmt.Sprintf("Successfully updated to FontGet v%s", targetVersion),
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
		// Spinner already showed the error message; just return it for Cobra to report once.
		return fmt.Errorf("update failed: %w", err)
	}

	if logger != nil {
		logger.Info("Successfully updated to version %s", targetVersion)
	}

	output.GetVerbose().Info("Update complete")
	output.GetDebug().State("Update successful")
	fmt.Printf("%s\n", ui.SuccessText.Render(fmt.Sprintf("Successfully updated to FontGet v%s", targetVersion)))
	fmt.Printf("%s\n", ui.Text.Render("Run 'fontget version --release-notes' to see what's new."))

	return nil
}

// performUpdate performs the actual update operation
func performUpdate(currentVersion, latestVersion string) error {
	logger := GetLogger()
	output.GetVerbose().Info("Starting update to version %s", latestVersion)
	output.GetDebug().State("Calling update.UpdateToLatest()")

	// Show spinner while performing the update
	err := ui.RunSpinner(
		fmt.Sprintf("Updating FontGet from v%s to v%s...", currentVersion, latestVersion),
		fmt.Sprintf("Successfully updated to FontGet v%s", latestVersion),
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
		// Spinner already showed the error message; just return it for Cobra to report once.
		return fmt.Errorf("update failed: %w", err)
	}

	if logger != nil {
		logger.Info("Successfully updated from %s to %s", currentVersion, latestVersion)
	}

	output.GetVerbose().Info("Update complete")
	output.GetDebug().State("Update successful")
	fmt.Printf("%s\n", ui.SuccessText.Render(fmt.Sprintf("Successfully updated to FontGet v%s", latestVersion)))
	fmt.Printf("%s\n", ui.Text.Render("Run 'fontget version --release-notes' to see what's new."))

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
