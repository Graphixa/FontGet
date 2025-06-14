package cmd

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"

	"fontget/internal/errors"
	"fontget/internal/platform"

	"github.com/spf13/cobra"
)

var (
	scope string
	force bool
)

// promptYesNo asks the user a yes/no question and returns true for yes, false for no
func promptYesNo(message string) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(message + " (y/n): ")
		response, err := reader.ReadString('\n')
		if err != nil {
			return false, fmt.Errorf("failed to read user input: %w", err)
		}
		response = strings.ToLower(strings.TrimSpace(response))
		if response == "y" || response == "yes" {
			return true, nil
		}
		if response == "n" || response == "no" {
			return false, nil
		}
		fmt.Println("Please answer 'y' or 'n'")
	}
}

var removeCmd = &cobra.Command{
	Use:   "remove <font-name>",
	Short: "Remove an installed font",
	Long: `Remove a font from your system.
You can specify the installation scope using the --scope flag:
  - user (default): Remove from current user's fonts
  - machine: Remove from system-wide fonts (requires elevation)

Use --force to skip interactive prompts.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fontName := args[0]

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

		// Check if font exists in the requested scope
		fontDir := fontManager.GetFontDir(installScope)
		installedFonts, err := platform.ListInstalledFonts(fontDir)
		if err != nil {
			return fmt.Errorf("error checking installed fonts: %w", err)
		}

		fontFound := false
		for _, installedFont := range installedFonts {
			if installedFont == fontName {
				fontFound = true
				break
			}
		}

		if !fontFound {
			// If not found in requested scope, check other scope
			otherScope := platform.MachineScope
			if installScope == platform.MachineScope {
				otherScope = platform.UserScope
			}
			otherFontDir := fontManager.GetFontDir(otherScope)
			otherFonts, err := platform.ListInstalledFonts(otherFontDir)
			if err != nil {
				return fmt.Errorf("error checking fonts in %s scope: %w", otherScope, err)
			}

			for _, otherFont := range otherFonts {
				if otherFont == fontName {
					if force {
						// If force flag is set, automatically use the other scope
						installScope = otherScope
						fontFound = true
						break
					}
					// Ask user if they want to remove from the other scope
					shouldRemove, err := promptYesNo(fmt.Sprintf("Font '%s' is not installed in %s scope, but is installed in %s scope. Would you like to remove it from %s scope instead?",
						fontName, installScope, otherScope, otherScope))
					if err != nil {
						return err
					}

					if shouldRemove {
						// Update scope to the other scope
						installScope = otherScope
						fontFound = true
						break
					} else {
						return fmt.Errorf("font '%s' is not installed in %s scope", fontName, installScope)
					}
				}
			}

			if !fontFound {
				return fmt.Errorf("font '%s' is not installed in any scope", fontName)
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

		fmt.Printf("Removing font '%s' from %s scope...\n", fontName, installScope)

		// Remove the font
		if err := fontManager.RemoveFont(fontName, installScope); err != nil {
			return fmt.Errorf("failed to remove font: %w", err)
		}

		fmt.Printf("Successfully removed font '%s' from %s scope\n", fontName, installScope)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
	removeCmd.Flags().StringVar(&scope, "scope", "user", "Installation scope (user or machine)")
	removeCmd.Flags().BoolVarP(&force, "force", "f", false, "Skip interactive prompts")
}
