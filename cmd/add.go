package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fontget/internal/platform"
	"fontget/internal/repo"
	"fontget/internal/ui"

	"github.com/spf13/cobra"
)

// InstallationStatus tracks the status of font installations
type InstallationStatus struct {
	Installed int
	Skipped   int
	Failed    int
	Details   []string
}

// showMultipleMatchesAndExit displays search results and instructs user to use specific font ID
func showMultipleMatchesAndExit(fontName string, matches []repo.FontMatch) {

	fmt.Printf("\n%s\n", ui.FeedbackInfo.Render(fmt.Sprintf("Multiple fonts found matching '%s'.", fontName)))
	fmt.Printf("%s\n\n", ui.FeedbackText.Render("Please specify the exact font ID to install from a specific source."))

	fmt.Printf("%s\n", ui.ContentText.Render("Suggestions:"))

	// Show examples for each match
	for _, match := range matches {
		fmt.Printf("  - fontget add %s\n", ui.CommandExample.Render(match.ID))
	}

	fmt.Printf("\n")
	// Use consistent column widths and apply styling to the entire formatted string
	headerLine := fmt.Sprintf("%-30s %-30s %-12s %-15s %-20s",
		"Name", "ID", "License", "Categories", "Source")
	fmt.Printf("%s\n", ui.TableHeader.Render(headerLine))
	fmt.Printf("%s\n", strings.Repeat("-", 127))

	for _, match := range matches {
		// Get categories (first one if available)
		categories := "N/A"
		if len(match.FontInfo.Categories) > 0 {
			categories = match.FontInfo.Categories[0]
		}

		// Get license
		license := match.FontInfo.License
		if license == "" {
			license = "N/A"
		}

		// Format the data line consistently with yellow font name
		fmt.Printf("%s %-30s %-12s %-15s %-20s\n",
			ui.TableSourceName.Render(fmt.Sprintf("%-30s", truncateString(match.Name, 30))),
			truncateString(match.ID, 30),
			truncateString(license, 12),
			truncateString(categories, 15),
			truncateString(match.Source, 20))
	}

	fmt.Printf("\n")
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// formatFontNameWithVariant formats a font name with its variant for display
func formatFontNameWithVariant(fontName, variant string) string {
	if variant == "" || variant == "regular" {
		return fontName
	}
	// Clean up the variant name for display
	cleanVariant := strings.ReplaceAll(variant, " ", "")
	cleanVariant = strings.ReplaceAll(cleanVariant, "-", "")
	cleanVariant = strings.ReplaceAll(cleanVariant, "_", "")

	// Remove the font name from the variant if it's duplicated
	// Try with spaces removed from font name too
	cleanFontName := strings.ReplaceAll(fontName, " ", "")
	cleanFontName = strings.ReplaceAll(cleanFontName, "-", "")
	cleanFontName = strings.ReplaceAll(cleanFontName, "_", "")

	if strings.HasPrefix(strings.ToLower(cleanVariant), strings.ToLower(cleanFontName)) {
		cleanVariant = cleanVariant[len(cleanFontName):]
	} else if strings.HasPrefix(strings.ToLower(cleanVariant), strings.ToLower(fontName)) {
		cleanVariant = cleanVariant[len(fontName):]
	}

	// Capitalize first letter of variant
	if len(cleanVariant) > 0 {
		cleanVariant = strings.ToUpper(cleanVariant[:1]) + cleanVariant[1:]
	}

	if cleanVariant != "" && cleanVariant != "Regular" {
		return fmt.Sprintf("%s %s", fontName, cleanVariant)
	}
	return fontName
}

// findSimilarFonts returns a list of font names that are similar to the given name
func findSimilarFonts(fontName string, allFonts []string) []string {
	// Simple fuzzy matching on the allFonts list (much faster)
	queryLower := strings.ToLower(fontName)
	queryNorm := strings.ReplaceAll(queryLower, " ", "")
	var similar []string

	// Simple substring matching for speed
	for _, font := range allFonts {
		fontLower := strings.ToLower(font)
		fontNorm := strings.ReplaceAll(fontLower, " ", "")
		// Skip exact equals (case/space-insensitive) to avoid suggesting the same query back
		if fontLower == queryLower || fontNorm == queryNorm {
			continue
		}
		if strings.Contains(fontLower, queryLower) || strings.Contains(queryLower, fontLower) {
			similar = append(similar, font)
			if len(similar) >= 5 {
				break
			}
		}
	}

	// If no substring matches, try partial word matches
	if len(similar) == 0 {
		words := strings.Fields(queryLower)
		for _, font := range allFonts {
			fontLower := strings.ToLower(font)
			fontNorm := strings.ReplaceAll(fontLower, " ", "")
			if fontLower == queryLower || fontNorm == queryNorm {
				continue
			}
			for _, word := range words {
				if len(word) > 2 && strings.Contains(fontLower, word) {
					similar = append(similar, font)
					if len(similar) >= 5 {
						break
					}
				}
			}
			if len(similar) >= 5 {
				break
			}
		}
	}

	GetLogger().Debug("Found %d similar fonts for %s", len(similar), fontName)
	return similar
}

var addCmd = &cobra.Command{
	Use:          "add <font-id> [<font-id2> <font-id3> ...]",
	Aliases:      []string{"install"},
	Short:        "Install fonts on your system",
	SilenceUsage: true,
	Long: `Install fonts from available font repositories.

You can specify multiple fonts by separating them with spaces. 
Font names with spaces should be wrapped in quotes. Comma-separated lists are also supported.

You can specify the installation scope using the --scope flag:
  - user (default): Install font for current user
  - machine: Install font system-wide (requires elevation)

`,
	Example: `  fontget add "Roboto"
  fontget add "Open Sans" "Fira Sans" "Noto Sans"
  fontget add roboto firasans notosans
  fontget add "roboto, firasans, notosans"
  fontget add "Open Sans" -s machine
  fontget add "roboto" -f`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Only handle empty query case
		if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
			fmt.Printf("\n%s\n\n", ui.RenderError("A font ID is required"))
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

		// Auto-detect scope if not explicitly provided
		if scope == "" {
			isElevated, err := fontManager.IsElevated()
			if err != nil {
				GetLogger().Warn("Failed to detect elevation status: %v", err)
				// Default to user scope if we can't detect elevation
				scope = "user"
			} else if isElevated {
				scope = "machine"
				GetLogger().Info("Auto-detected elevated privileges, defaulting to 'machine' scope")
				fmt.Println(ui.FeedbackInfo.Render("Auto-detected administrator privileges - installing system-wide"))
			} else {
				scope = "user"
				GetLogger().Info("Auto-detected user privileges, defaulting to 'user' scope")
			}
		}

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

		// Process font names from arguments
		fontNames := ParseFontNames(args)

		GetLogger().Info("Processing %d font(s): %v", len(fontNames), fontNames)

		// Get font directory for the specified scope
		fontDir := fontManager.GetFontDir(installScope)
		GetLogger().Debug("Using font directory: %s", fontDir)

		// Initialize status tracking
		status := InstallationStatus{
			Details: make([]string, 0),
		}

		// Get all available fonts for suggestions (use cached version for speed)
		allFonts := repo.GetAllFontsCached()
		if len(allFonts) == 0 {
			GetLogger().Warn("Could not get list of available fonts for suggestions")
			fmt.Println(ui.RenderError("Warning: Could not get list of available fonts for suggestions"))
		}

		// Process each font
		for _, fontName := range fontNames {
			GetLogger().Info("Processing font: %s", fontName)

			// Check if this is already a specific font ID (contains a dot like "google.roboto")
			var fonts []repo.FontFile
			var err error

			if strings.Contains(fontName, ".") {
				// This is a specific font ID, use it directly
				fonts, err = repo.GetFontByID(fontName)
			} else {
				// Find all matches across sources
				matches, matchErr := repo.FindFontMatches(fontName)
				if matchErr != nil {
					err = matchErr
				} else if len(matches) == 0 {
					err = fmt.Errorf("font not found: %s", fontName)
				} else if len(matches) == 1 {
					// Single match, proceed normally
					fonts, err = repo.GetFontByID(matches[0].ID)
				} else {
					// Multiple matches - show search results and prompt for specific ID
					showMultipleMatchesAndExit(fontName, matches)
					return nil // Exit the command after showing the options
				}
			}

			if err != nil {
				// This is a query error, not an installation failure
				GetLogger().Error("Font not found: %s", fontName)

				// Try to find similar fonts
				similar := findSimilarFonts(fontName, allFonts)
				if len(similar) > 0 {
					GetLogger().Info("Found %d similar fonts for %s", len(similar), fontName)
					fmt.Printf("\n%s\n\n", ui.FeedbackError.Render(fmt.Sprintf("Font '%s' not found.", fontName)))
					fmt.Printf("%s\n", ui.FeedbackText.Render("Did you mean one of these fonts?"))
					fmt.Printf("\n%s\n", ui.FeedbackText.Render("Suggestions:"))
					for _, s := range similar {
						fmt.Printf("  - %s\n", ui.TableSourceName.Render(s))
					}
					fmt.Println() // Add a blank line after suggestions
				} else {
					GetLogger().Info("No similar fonts found for %s", fontName)
					fmt.Printf("\n%s\n", ui.FeedbackError.Render("No similar fonts found."))
					fmt.Printf("%s\n", ui.FeedbackText.Render("Try using the search command to find available fonts."))
					fmt.Printf("\n%s\n", ui.FeedbackText.Render("Example:"))
					fmt.Printf("  %s\n", ui.CommandExample.Render("fontget search \"roboto\""))
					fmt.Println()
				}
				continue // Skip to next font
			}

			GetLogger().Debug("Found %d font files for %s", len(fonts), fontName)

			// Download and install each font file
			for _, font := range fonts {
				// Check if font is already installed (unless force flag is set)
				if !force {
					fontPath := filepath.Join(fontDir, font.Path)
					if _, err := os.Stat(fontPath); err == nil {
						status.Skipped++
						fontDisplayName := formatFontNameWithVariant(font.Name, font.Variant)
						msg := fmt.Sprintf("  - \"%s\" is already installed (%s)", fontDisplayName, ui.FeedbackWarning.Render("Skipped"))
						GetLogger().Info("Font already installed: %s", fontDisplayName)
						fmt.Println(ui.ContentText.Render(msg))
						continue
					}
				}

				// Download font to temp directory
				tempDir := filepath.Join(os.TempDir(), "Fontget", "fonts")
				GetLogger().Debug("Downloading font to temp directory: %s", tempDir)
				fontPath, err := repo.DownloadFont(&font, tempDir)
				if err != nil {
					status.Failed++
					fontDisplayName := formatFontNameWithVariant(font.Name, font.Variant)
					msg := fmt.Sprintf("  - \"%s\" (Failed to download) - %v", fontDisplayName, err)
					GetLogger().Error("Failed to download font %s: %v", fontDisplayName, err)
					fmt.Println(ui.RenderError(msg))
					continue
				}

				// Install the font
				fontDisplayName := formatFontNameWithVariant(font.Name, font.Variant)
				GetLogger().Debug("Installing font: %s", fontDisplayName)
				if err := fontManager.InstallFont(fontPath, installScope, force); err != nil {
					os.Remove(fontPath) // Clean up temp file
					status.Failed++
					msg := fmt.Sprintf("  - \"%s\" (%s) - %v", fontDisplayName, ui.FeedbackError.Render("Failed to install"), err)
					GetLogger().Error("Failed to install font %s: %v", fontDisplayName, err)
					fmt.Println(ui.TableRow.Render(msg))
					continue
				}

				// Clean up temp file
				os.Remove(fontPath)
				status.Installed++
				msg := fmt.Sprintf("  - \"%s\" (%s to %s scope)", fontDisplayName, ui.FeedbackSuccess.Render("Installed"), scope)
				GetLogger().Info("Successfully installed font: %s to %s scope", fontDisplayName, scope)
				fmt.Println(ui.ContentText.Render(msg))
			}
		}

		// Print status report only if there were actual operations
		PrintStatusReport(StatusReport{
			Success:      status.Installed,
			Skipped:      status.Skipped,
			Failed:       status.Failed,
			SuccessLabel: "Installed",
			SkippedLabel: "Skipped",
			FailedLabel:  "Failed",
		})

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
// FontInstallationError is now defined in shared.go

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringP("scope", "s", "", "Installation scope (user or machine)")
	addCmd.Flags().BoolP("force", "f", false, "Force installation even if font is already installed")
}
