package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fontget/internal/platform"
	"fontget/internal/repo"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// InstallationStatus tracks the status of font installations
type InstallationStatus struct {
	Installed int
	Skipped   int
	Failed    int
	Details   []string
}

// findSimilarFonts returns a list of font names that are similar to the given name
func findSimilarFonts(fontName string, allFonts []string) []string {
	// Use the search function to get scored results
	results, err := repo.SearchFonts(fontName, false)
	if err != nil {
		GetLogger().Error("Failed to search for similar fonts: %v", err)
		return nil
	}

	// Create a map to track unique font names (case-insensitive)
	seen := make(map[string]bool)
	var similar []string

	// Process results in order of score
	for _, result := range results {
		// Normalize the font name for deduplication
		normalizedName := strings.ToLower(result.Name)
		if !seen[normalizedName] {
			seen[normalizedName] = true
			similar = append(similar, result.Name)
		}
	}

	// Limit to 5 suggestions
	if len(similar) > 5 {
		similar = similar[:5]
	}

	GetLogger().Debug("Found %d similar fonts for %s", len(similar), fontName)
	return similar
}

var addCmd = &cobra.Command{
	Use:          "add <font-id>",
	Short:        "Add a font to your system",
	SilenceUsage: true,
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
		GetLogger().Info("Starting font installation operation")

		// Double check args to prevent panic
		if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
			return nil // Args validator will have already shown the help
		}

		// Create font manager
		fontManager, err := platform.NewFontManager()
		if err != nil {
			GetLogger().Error("Failed to initialize font manager: %v", err)
			return fmt.Errorf("failed to initialize font manager: %w", err)
		}

		// Get scope from flag
		scope, _ := cmd.Flags().GetString("scope")
		force, _ := cmd.Flags().GetBool("force")

		GetLogger().Info("Installation parameters - Scope: %s, Force: %v", scope, force)

		// Convert scope string to InstallationScope
		installScope := platform.UserScope
		if scope != "user" {
			installScope = platform.InstallationScope(scope)
			if installScope != platform.UserScope && installScope != platform.MachineScope {
				GetLogger().Error("Invalid scope '%s'", scope)
				return fmt.Errorf("invalid scope '%s'. Must be 'user' or 'machine'", scope)
			}
		}

		// Check elevation
		if err := checkElevation(cmd, fontManager, installScope); err != nil {
			GetLogger().Error("Elevation check failed: %v", err)
			return err
		}

		// Get font names from args and split by comma
		fontNames := strings.Split(args[0], ",")
		for i, name := range fontNames {
			fontNames[i] = strings.TrimSpace(name)
		}

		GetLogger().Info("Processing %d font(s): %v", len(fontNames), fontNames)

		// Get font directory for the specified scope
		fontDir := fontManager.GetFontDir(installScope)
		GetLogger().Debug("Using font directory: %s", fontDir)

		// Initialize status tracking
		status := InstallationStatus{
			Details: make([]string, 0),
		}

		// Create color functions
		green := color.New(color.FgGreen).SprintFunc()
		yellow := color.New(color.FgYellow).SprintFunc()
		red := color.New(color.FgRed).SprintFunc()
		bold := color.New(color.Bold).SprintFunc()
		cyan := color.New(color.FgCyan).SprintFunc()

		// Get all available fonts for suggestions
		allFonts := repo.GetAllFonts()
		if len(allFonts) == 0 {
			GetLogger().Warn("Could not get list of available fonts for suggestions")
			fmt.Println(red("Warning: Could not get list of available fonts for suggestions"))
		}

		// Process each font
		for _, fontName := range fontNames {
			GetLogger().Info("Processing font: %s", fontName)
			fmt.Printf("\n%s\n", bold(fontName))

			// Get font information from repository
			fonts, err := repo.GetFont(fontName)
			if err != nil {
				// This is a query error, not an installation failure
				msg := fmt.Sprintf("Font '%s' not found in the Google Fonts repository", fontName)
				GetLogger().Error("Font not found: %s", fontName)
				fmt.Println(red(msg))

				// Try to find similar fonts
				similar := findSimilarFonts(fontName, allFonts)
				if len(similar) > 0 {
					GetLogger().Info("Found %d similar fonts for %s", len(similar), fontName)
					fmt.Println(cyan("\nDid you mean one of these fonts?"))
					for _, s := range similar {
						fmt.Printf("  - %s\n", s)
					}
					fmt.Println() // Add a blank line after suggestions
				} else {
					GetLogger().Info("No similar fonts found for %s", fontName)
					fmt.Println(yellow("\nNo similar fonts found. Try using the search command to find available fonts:"))
					fmt.Println("  fontget search <query>")
					fmt.Println()
				}
				continue // Skip to next font
			}

			GetLogger().Debug("Found %d font files for %s", len(fonts), fontName)

			// Download and install each font file
			for _, font := range fonts {
				// Check if font is already installed (unless force flag is set)
				if !force {
					fontPath := filepath.Join(fontDir, font.Name)
					if _, err := os.Stat(fontPath); err == nil {
						status.Skipped++
						msg := fmt.Sprintf("  - \"%s\" is already installed (Skipped)", font.Name)
						GetLogger().Info("Font already installed: %s", font.Name)
						fmt.Println(yellow(msg))
						continue
					}
				}

				// Download font to temp directory
				tempDir := filepath.Join(os.TempDir(), "Fontget", "fonts")
				GetLogger().Debug("Downloading font to temp directory: %s", tempDir)
				fontPath, err := repo.DownloadFont(&font, tempDir)
				if err != nil {
					status.Failed++
					msg := fmt.Sprintf("  - \"%s\" (Failed to download) - %v", font.Name, err)
					GetLogger().Error("Failed to download font %s: %v", font.Name, err)
					fmt.Println(red(msg))
					continue
				}

				// Install the font
				GetLogger().Debug("Installing font: %s", font.Name)
				if err := fontManager.InstallFont(fontPath, installScope, force); err != nil {
					os.Remove(fontPath) // Clean up temp file
					status.Failed++
					msg := fmt.Sprintf("  - \"%s\" (Failed to install) - %v", font.Name, err)
					GetLogger().Error("Failed to install font %s: %v", font.Name, err)
					fmt.Println(red(msg))
					continue
				}

				// Clean up temp file
				os.Remove(fontPath)
				status.Installed++
				msg := fmt.Sprintf("  - \"%s\" (Installed)", font.Name)
				GetLogger().Info("Successfully installed font: %s", font.Name)
				fmt.Println(green(msg))
			}
		}

		// Print status report
		fmt.Printf("\n%s\n", bold("Status Report"))
		fmt.Println("---------------------------------------------")
		fmt.Printf("%s: %d  |  %s: %d  |  %s: %d\n\n",
			green("Installed"), status.Installed,
			yellow("Skipped"), status.Skipped,
			red("Failed"), status.Failed)

		GetLogger().Info("Installation complete - Installed: %d, Skipped: %d, Failed: %d",
			status.Installed, status.Skipped, status.Failed)

		// Only return error if there were actual installation or download failures
		if status.Failed > 0 {
			return &FontInstallationError{
				FailedCount: status.Failed,
				TotalCount:  len(fontNames),
			}
		}

		return nil
	},
}

// FontInstallationError is a custom error type for font installation failures
type FontInstallationError struct {
	FailedCount int
	TotalCount  int
}

// Error implements the error interface
func (e *FontInstallationError) Error() string {
	return fmt.Sprintf("one or more font files failed to install")
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().String("scope", "user", "Installation scope (user or machine)")
	addCmd.Flags().Bool("force", false, "Force installation even if font is already installed")
}
