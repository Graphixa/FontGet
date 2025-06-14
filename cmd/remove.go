package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fontget/internal/platform"

	"github.com/spf13/cobra"
)

var (
	scope string
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
	Short: "Remove a font from your system",
	Long: `Remove a font from your system. You can specify the installation scope using the --scope flag:
  - user (default): Remove font from current user
  - machine: Remove font system-wide (requires elevation)`,
	Example: `  fontget remove "Roboto"
  fontget remove "Open Sans" --scope machine`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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

		// Get font directory for the specified scope
		fontDir := fontManager.GetFontDir(installScope)

		// Get font name from args
		fontName := args[0]

		// Find all matching font files
		var matchingFiles []string
		err = filepath.Walk(fontDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				// Check if the file name contains the font name (case-insensitive)
				if strings.Contains(strings.ToLower(info.Name()), strings.ToLower(fontName)) {
					matchingFiles = append(matchingFiles, path)
				}
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to search for font files: %w", err)
		}

		if len(matchingFiles) == 0 {
			return fmt.Errorf("no fonts found matching '%s'", fontName)
		}

		// Remove each matching font file
		for _, file := range matchingFiles {
			if err := os.Remove(file); err != nil {
				return fmt.Errorf("failed to remove font file %s: %w", file, err)
			}
			fmt.Printf("Removed: %s\n", filepath.Base(file))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
	removeCmd.Flags().String("scope", "user", "Installation scope (user or machine)")
}
