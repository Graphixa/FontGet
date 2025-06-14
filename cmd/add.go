package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fontget/internal/platform"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <font-id>",
	Short: "Add a font to your system",
	Long: `Add a font to your system. You can specify the installation scope using the --scope flag:
  - user (default): Add font for current user only
  - machine: Add font system-wide (requires elevation)`,
	Example: `  fontget add "Roboto"
  fontget add "opensans" --scope machine
  fontget add "roboto" --force
  fontget add "roboto, firasans, notosans"
  `,
	Args: func(cmd *cobra.Command, args []string) error {
		// Only handle empty query case
		if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
			red := color.New(color.FgRed).SprintFunc()
			fmt.Printf("\n%s\n\n", red("A font ID is required"))
			return cmd.Help()
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Double check args to prevent panic
		if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
			return nil // Args validator will have already shown the help
		}

		// Create font manager
		fontManager, err := platform.NewFontManager()
		if err != nil {
			return fmt.Errorf("failed to initialize font manager: %w", err)
		}

		// Get scope from flag
		scope, _ := cmd.Flags().GetString("scope")

		// Convert scope string to InstallationScope
		installScope := platform.UserScope
		if scope != "user" {
			installScope = platform.InstallationScope(scope)
			if installScope != platform.UserScope && installScope != platform.MachineScope {
				return fmt.Errorf("invalid scope '%s'. Must be 'user' or 'machine'", scope)
			}
		}

		// Check elevation
		if err := checkElevation(cmd, fontManager, installScope); err != nil {
			return err
		}

		// Get font name from args
		fontName := args[0]

		// Get font directory for the specified scope
		fontDir := fontManager.GetFontDir(installScope)

		// Check if font is already installed
		fontPath := filepath.Join(fontDir, fontName)
		if _, err := os.Stat(fontPath); err == nil {
			return fmt.Errorf("font already installed: %s", fontName)
		}

		// Install the font
		if err := fontManager.InstallFont(fontPath, installScope); err != nil {
			return fmt.Errorf("failed to install font: %w", err)
		}

		fmt.Printf("Successfully installed font: %s\n", fontName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().String("scope", "user", "Installation scope (user or machine)")
}
