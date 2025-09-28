package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fontget/internal/config"
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

// getFontDisplayName extracts a proper display name from the font file path
func getFontDisplayName(fontPath, fontName, variant string) string {
	// Get the base filename without extension
	baseName := filepath.Base(fontPath)
	ext := filepath.Ext(baseName)
	fileName := strings.TrimSuffix(baseName, ext)

	// For Nerd Fonts and similar fonts, use the actual filename
	// which contains the proper variant information
	if strings.Contains(fileName, "NerdFont") || strings.Contains(fileName, "Nerd") {
		// Convert filename to display name
		// e.g., "HackNerdFontMono-Regular" -> "Hack Nerd Font Mono Regular"
		displayName := strings.ReplaceAll(fileName, "NerdFont", " Nerd Font ")
		displayName = strings.ReplaceAll(displayName, "-", " ")
		displayName = strings.ReplaceAll(displayName, "  ", " ") // Clean up double spaces
		displayName = strings.TrimSpace(displayName)
		return displayName
	}

	// For other fonts, use the original formatting
	return formatFontNameWithVariant(fontName, variant)
}

// getSourceName extracts the source name from a font ID (e.g., "google.roboto" -> "Google Fonts")
func getSourceName(fontID string) string {
	// Extract source prefix from font ID (e.g., "google.roboto" -> "google")
	parts := strings.Split(fontID, ".")
	if len(parts) < 2 {
		return "Unknown Source"
	}

	sourcePrefix := parts[0]

	// Load sources configuration to get the display name
	sourcesConfig, err := config.LoadSourcesConfig()
	if err != nil {
		// Fallback to capitalized prefix if we can't load config
		return strings.Title(sourcePrefix)
	}

	// Find the source with matching prefix
	for sourceName, sourceConfig := range sourcesConfig.Sources {
		if sourceConfig.Prefix == sourcePrefix {
			return sourceName
		}
	}

	// Fallback to capitalized prefix if not found
	return strings.Title(sourcePrefix)
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
			var sourceName string
			var err error

			if strings.Contains(fontName, ".") {
				// This is a specific font ID, use it directly
				fonts, err = repo.GetFontByID(fontName)
				sourceName = getSourceName(fontName)
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
					sourceName = matches[0].Source
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

					for _, s := range similar {
						fmt.Printf("  - %s\n", ui.FeedbackWarning.Render(s))
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

			// Show installation header for this font
			fontDisplayName := formatFontNameWithVariant(fonts[0].Name, fonts[0].Variant)
			if len(fonts) > 1 {
				fontDisplayName = fonts[0].Name // Use base name for multiple variants
			}

			// Show the header first
			installHeader := fmt.Sprintf("\nInstalling '%s' from '%s'\n", fontDisplayName, sourceName)
			fmt.Printf("%s\n", ui.FeedbackInfo.Render(installHeader))

			// Phase 1: Download with progress bar
			downloadHeader := "Downloading font files..."

			// For single-font archives (like Nerd Fonts), show more detailed progress
			var totalSteps int
			if len(fonts) == 1 {
				totalSteps = 3 // Download, Extract, Complete
			} else {
				totalSteps = len(fonts) // One step per font file
			}

			// Store downloaded font paths for installation
			type downloadedFont struct {
				font      repo.FontFile
				fontPaths []string
			}
			var downloadedFonts []downloadedFont

			progressErr := runProgressBar(downloadHeader, totalSteps, func(updateProgress func()) error {
				// Download each font file
				for _, font := range fonts {
					// Download font to temp directory (handles archives automatically)
					tempDir := filepath.Join(os.TempDir(), "Fontget", "fonts")
					GetLogger().Debug("Downloading font to temp directory: %s", tempDir)

					if len(fonts) == 1 {
						// For single archives, show detailed progress
						updateProgress() // Step 1: Starting download
						time.Sleep(200 * time.Millisecond)
					}

					fontPaths, downloadErr := repo.DownloadAndExtractFont(&font, tempDir)

					if downloadErr != nil {
						status.Failed++
						GetLogger().Error("Failed to download font %s: %v", font.Name, downloadErr)
						updateProgress() // Update progress even on failure
						continue
					}

					// Store for installation phase
					downloadedFonts = append(downloadedFonts, downloadedFont{
						font:      font,
						fontPaths: fontPaths,
					})

					if len(fonts) == 1 {
						// For single archives, show extraction progress
						updateProgress() // Step 2: Extracting
						time.Sleep(200 * time.Millisecond)
						updateProgress() // Step 3: Complete
						time.Sleep(200 * time.Millisecond)
					} else {
						// For multiple files, one step per file
						updateProgress()
						time.Sleep(50 * time.Millisecond)
					}
				}
				return nil
			})

			if progressErr != nil {
				GetLogger().Error("Failed to download font %s: %v", fontDisplayName, progressErr)
				continue // Skip to next font
			}

			// Phase 2: Install downloaded fonts (normal terminal output)
			for _, downloaded := range downloadedFonts {
				for _, fontPath := range downloaded.fontPaths {
					// Check if font is already installed (unless force flag is set)
					if !force {
						expectedPath := filepath.Join(fontDir, filepath.Base(fontPath))
						if _, err := os.Stat(expectedPath); err == nil {
							status.Skipped++
							fontDisplayName := getFontDisplayName(fontPath, downloaded.font.Name, downloaded.font.Variant)
							msg := fmt.Sprintf("  %s %s %s", ui.FeedbackSuccess.Render("✓"), ui.TableRow.Render(fontDisplayName), ui.FeedbackWarning.Render("Skipped - already installed"))
							GetLogger().Info("Font already installed: %s", fontDisplayName)
							fmt.Println(ui.ContentText.Render(msg))
							os.Remove(fontPath) // Clean up temp file
							continue
						}
					}

					// Install the font
					fontDisplayName := getFontDisplayName(fontPath, downloaded.font.Name, downloaded.font.Variant)
					GetLogger().Debug("Installing font: %s", fontDisplayName)

					installErr := fontManager.InstallFont(fontPath, installScope, force)

					if installErr != nil {
						os.Remove(fontPath) // Clean up temp file
						status.Failed++
						msg := fmt.Sprintf("  %s %s %s - %s", ui.FeedbackError.Render("✗"), ui.TableRow.Render(fontDisplayName), ui.FeedbackError.Render("Failed"), installErr.Error())
						GetLogger().Error("Failed to install font %s: %v", fontDisplayName, installErr)
						fmt.Println(ui.TableRow.Render(msg))
						continue
					}

					// Clean up temp file
					os.Remove(fontPath)
					status.Installed++
					msg := fmt.Sprintf("  %s %s %s", ui.FeedbackSuccess.Render("✓"), ui.TableRow.Render(fontDisplayName), ui.FeedbackSuccess.Render("Installed to "+scope+" scope"))
					GetLogger().Info("Successfully installed font: %s to %s scope", fontDisplayName, scope)
					fmt.Println(ui.ContentText.Render(msg))
				}
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
