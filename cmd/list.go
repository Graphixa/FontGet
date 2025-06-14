package cmd

import (
	"fmt"
	"os"
	"runtime"

	"fontget/internal/errors"
	"fontget/internal/platform"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed fonts",
	Long: `List all installed fonts on your system.
You can specify the installation scope using the --scope flag:
  - user (default): List fonts installed for current user
  - machine: List system-wide installed fonts (requires elevation)`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create font manager
		fontManager, err := platform.NewFontManager()
		if err != nil {
			return fmt.Errorf("failed to initialize font manager: %w", err)
		}

		// Convert scope string to InstallationScope
		installScope := platform.UserScope
		if scope != "user" {
			installScope = platform.InstallationScope(scope)
			if installScope != platform.UserScope && installScope != platform.MachineScope {
				return fmt.Errorf("invalid scope '%s'. Must be 'user' or 'machine'", scope)
			}
		}

		// Check if elevation is required
		if fontManager.RequiresElevation(installScope) {
			elevated, err := fontManager.IsElevated()
			if err != nil {
				return fmt.Errorf("failed to check elevation status: %w", err)
			}
			if !elevated {
				errors.PrintElevationHelp(cmd, runtime.GOOS)
				return errors.ElevationRequired(runtime.GOOS)
			}
		}

		// Get font directory for the specified scope
		fontDir := fontManager.GetFontDir(installScope)

		// List all font files in the directory
		files, err := os.ReadDir(fontDir)
		if err != nil {
			return fmt.Errorf("failed to read font directory: %w", err)
		}

		if len(files) == 0 {
			fmt.Printf("No fonts found in %s\n", fontDir)
			return nil
		}

		fmt.Printf("Installed fonts in %s:\n", fontDir)
		for _, file := range files {
			if !file.IsDir() {
				fmt.Printf("  - %s\n", file.Name())
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringVar(&scope, "scope", "user", "Installation scope (user or machine)")
}
