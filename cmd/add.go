package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fontget/internal/config"
	"fontget/internal/output"
	"fontget/internal/platform"
	"fontget/internal/repo"
	"fontget/internal/ui"

	"github.com/spf13/cobra"
)

// Configuration options for add command behavior
const (
	// HideProgressBarWhenFinished controls whether the progress bar disappears after completion
	// Set to true for cleaner output (recommended), false to keep progress bar visible for debugging
	HideProgressBarWhenFinished = true

	// ShowProgressBarHeader controls whether the progress bar displays its own header text
	// Set to false when you're displaying your own header (cleaner), true to show both
	ShowProgressBarHeader = false
)

// InstallationStatus tracks the status of font installations
type InstallationStatus struct {
	Installed int
	Skipped   int
	Failed    int
	Details   []string
}

// showFontNotFoundWithSuggestions displays font not found error with suggestions in table format
func showFontNotFoundWithSuggestions(fontName string, similar []string) {
	fmt.Printf("\n%s\n", ui.FeedbackError.Render(fmt.Sprintf("Font '%s' not found.", fontName)))

	// If no similar fonts found, show general guidance
	if len(similar) == 0 {
		fmt.Printf("%s\n", ui.FeedbackText.Render("Try using the search command to find available fonts."))
		fmt.Printf("\n%s\n", ui.FeedbackText.Render("Example:"))
		fmt.Printf("  %s\n", ui.CommandExample.Render("fontget search \"roboto\""))
		fmt.Println()
		return
	}

	// Collect unique matches from all suggestions
	seenIDs := make(map[string]bool)
	var uniqueMatches []repo.FontMatch

	for _, suggestion := range similar {
		// Try to find matches for the suggestion to get proper font IDs
		matches, err := repo.FindFontMatches(suggestion)
		if err == nil && len(matches) > 0 {
			// Add all unique matches from this suggestion
			for _, match := range matches {
				if !seenIDs[match.ID] {
					uniqueMatches = append(uniqueMatches, match)
					seenIDs[match.ID] = true
				}
			}
		}
	}

	// If we found matches, display them in table format
	if len(uniqueMatches) > 0 {
		fmt.Printf("%s\n\n", ui.FeedbackText.Render("Did you mean one of these fonts?"))
		// Use consistent column widths and apply styling to the entire formatted string
		fmt.Printf("%s\n", ui.TableHeader.Render(GetSearchTableHeader()))
		fmt.Printf("%s\n", GetTableSeparator())

		// Display each unique match as a table row
		for _, match := range uniqueMatches {
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
			fmt.Printf("%s %-*s %-*s %-*s %-*s\n",
				ui.TableSourceName.Render(fmt.Sprintf("%-*s", TableColName, truncateString(match.Name, TableColName))),
				TableColID, truncateString(match.ID, TableColID),
				TableColLicense, truncateString(license, TableColLicense),
				TableColCategories, truncateString(categories, TableColCategories),
				TableColSource, truncateString(match.Source, TableColSource))
		}
	} else {
		// Fallback: if similar font names were found but couldn't be resolved to matches
		// This suggests the font exists in our cache but is no longer available from sources
		fmt.Printf("%s\n", ui.FeedbackError.Render(fmt.Sprintf("Font '%s' was not able to be downloaded and installed.", fontName)))
		fmt.Printf("%s\n", ui.FeedbackText.Render("It may have been removed from the font source."))
		fmt.Printf("\n%s\n", ui.FeedbackText.Render("Please refresh FontGet sources using:"))
		fmt.Printf("  %s\n", ui.CommandExample.Render("fontget sources update"))
		fmt.Printf("\n%s\n", ui.FeedbackText.Render("Try using the search command to find other available fonts:"))
		fmt.Printf("  %s\n", ui.CommandExample.Render("fontget search \"roboto\""))
	}

	fmt.Printf("\n")
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
	fmt.Printf("%s\n", ui.TableHeader.Render(GetSearchTableHeader()))
	fmt.Printf("%s\n", GetTableSeparator())

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
		fmt.Printf("%s %-*s %-*s %-*s %-*s\n",
			ui.TableSourceName.Render(fmt.Sprintf("%-*s", TableColName, truncateString(match.Name, TableColName))),
			TableColID, truncateString(match.ID, TableColID),
			TableColLicense, truncateString(license, TableColLicense),
			TableColCategories, truncateString(categories, TableColCategories),
			TableColSource, truncateString(match.Source, TableColSource))
	}

	fmt.Printf("\n")
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

	sourcePrefix := strings.ToLower(parts[0]) // Ensure lowercase for consistent matching

	// Load manifest to get the display name
	manifest, err := config.LoadManifest()
	if err != nil {
		// Fallback to capitalized prefix if we can't load manifest
		return strings.Title(sourcePrefix)
	}

	// Find the source with matching prefix (case-insensitive)
	for sourceName, sourceConfig := range manifest.Sources {
		if strings.ToLower(sourceConfig.Prefix) == sourcePrefix {
			return sourceName
		}
	}

	// Fallback to capitalized prefix if not found
	return strings.Title(sourcePrefix)
}

// findSimilarFonts returns a list of font names that are similar to the given name
// Prioritizes actual font names (no dots) over font IDs (with dots like "google.roboto")
func findSimilarFonts(fontName string, allFonts []string) []string {
	queryLower := strings.ToLower(fontName)
	queryNorm := strings.ReplaceAll(queryLower, " ", "")

	// Separate font names from font IDs for prioritized matching
	var fontNames []string // "Open Sans", "Roboto", etc.
	var fontIDs []string   // "google.roboto", "nerd.fira-code", etc.

	for _, font := range allFonts {
		if strings.Contains(font, ".") {
			fontIDs = append(fontIDs, font)
		} else {
			fontNames = append(fontNames, font)
		}
	}

	var similar []string

	// Phase 1: Check font names first (higher priority)
	similar = findMatchesInList(queryLower, queryNorm, fontNames, similar, 5)

	// Phase 2: If we need more results, check font IDs
	if len(similar) < 5 {
		remaining := 5 - len(similar)
		similar = findMatchesInList(queryLower, queryNorm, fontIDs, similar, remaining)
	}

	GetLogger().Debug("Found %d similar fonts for %s", len(similar), fontName)
	return similar
}

// findMatchesInList performs fuzzy matching on a specific list of fonts
func findMatchesInList(queryLower, queryNorm string, fontList []string, existing []string, maxResults int) []string {
	similar := existing

	// Simple substring matching for speed
	for _, font := range fontList {
		if len(similar) >= maxResults {
			break
		}

		fontLower := strings.ToLower(font)
		fontNorm := strings.ReplaceAll(fontLower, " ", "")

		// Skip exact equals and already found fonts
		if fontLower == queryLower || fontNorm == queryNorm {
			continue
		}

		// Skip if already in results
		alreadyFound := false
		for _, existing := range similar {
			if strings.ToLower(existing) == fontLower {
				alreadyFound = true
				break
			}
		}
		if alreadyFound {
			continue
		}

		if strings.Contains(fontLower, queryLower) || strings.Contains(queryLower, fontLower) {
			similar = append(similar, font)
		}
	}

	// If no substring matches and we still need more, try partial word matches
	if len(similar) < maxResults {
		words := strings.Fields(queryLower)
		for _, font := range fontList {
			if len(similar) >= maxResults {
				break
			}

			fontLower := strings.ToLower(font)
			fontNorm := strings.ReplaceAll(fontLower, " ", "")

			// Skip exact equals and already found fonts
			if fontLower == queryLower || fontNorm == queryNorm {
				continue
			}

			// Skip if already in results
			alreadyFound := false
			for _, existing := range similar {
				if strings.ToLower(existing) == fontLower {
					alreadyFound = true
					break
				}
			}
			if alreadyFound {
				continue
			}

			for _, word := range words {
				if len(word) > 2 && strings.Contains(fontLower, word) {
					similar = append(similar, font)
					break
				}
			}
		}
	}

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

		// Ensure manifest system is initialized (fixes missing sources.json bug)
		if err := config.EnsureManifestExists(); err != nil {
			return fmt.Errorf("failed to initialize sources: %v", err)
		}

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

		// Debug-level information for developers
		output.GetDebug().Message("Debug mode enabled - showing detailed diagnostic information")

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
			if errors.Is(err, ErrElevationRequired) {
				return nil // Already printed user-friendly message
			}
			GetLogger().Error("Elevation check failed: %v", err)
			return err
		}

		// Process font names from arguments
		fontNames := ParseFontNames(args)

		GetLogger().Info("Processing %d font(s): %v", len(fontNames), fontNames)

		// Get font directory for the specified scope
		fontDir := fontManager.GetFontDir(installScope)
		GetLogger().Debug("Using font directory: %s", fontDir)

		// Verbose-level information for users
		output.GetVerbose().Info("Installing fonts to: %s", fontDir)

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
				// Convert to lowercase for consistent matching
				fontID := strings.ToLower(fontName)
				fonts, err = repo.GetFontByID(fontID)
				sourceName = getSourceName(fontID)
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
				GetLogger().Info("Found %d similar fonts for %s", len(similar), fontName)
				showFontNotFoundWithSuggestions(fontName, similar)
				continue // Skip to next font
			}

			GetLogger().Debug("Found %d font files for %s", len(fonts), fontName)

			// Show installation header for this font
			fontDisplayName := formatFontNameWithVariant(fonts[0].Name, fonts[0].Variant)
			if len(fonts) > 1 {
				fontDisplayName = fonts[0].Name // Use base name for multiple variants
			}

			// Phase 1: Download with progress bar (combined header)
			// Add blank line before each font header
			fmt.Println()

			headerMessage := fmt.Sprintf("Downloading and Installing '%s' from '%s'", fontDisplayName, sourceName)
			fmt.Println(ui.FeedbackInfo.Render(headerMessage))
			downloadHeader := "" // Don't pass header to progress bar since we already displayed it

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

			progressErr := runProgressBarWithOptions(downloadHeader, totalSteps, func(updateProgress func()) error {
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
			}, HideProgressBarWhenFinished, ShowProgressBarHeader)

			if progressErr != nil {
				GetLogger().Error("Failed to download font %s: %v", fontDisplayName, progressErr)
				continue // Skip to next font
			}

			// Add blank line after progress bar
			fmt.Println()

			// Phase 2: Install downloaded fonts (normal terminal output)
			for _, downloaded := range downloadedFonts {
				for _, fontPath := range downloaded.fontPaths {
					// Check if font is already installed (unless force flag is set)
					if !force {
						expectedPath := filepath.Join(fontDir, filepath.Base(fontPath))
						if _, err := os.Stat(expectedPath); err == nil {
							status.Skipped++
							fontDisplayName := getFontDisplayName(fontPath, downloaded.font.Name, downloaded.font.Variant)
							msg := fmt.Sprintf("  %s %s - %s", ui.FeedbackSuccess.Render("✓"), ui.TableRow.Render(fontDisplayName), ui.FeedbackWarning.Render("[Skipped] already installed"))
							GetLogger().Info("Font already installed: %s", fontDisplayName)

							// Verbose mode shows additional info about skipped fonts
							output.GetVerbose().Detail("Info", "Font exists at: %s", expectedPath)

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

						// Simple, clean error message for all platforms
						msg := fmt.Sprintf("  %s %s - %s", ui.FeedbackError.Render("✗"), ui.TableRow.Render(fontDisplayName), ui.FeedbackError.Render("[Failed] to overwrite existing font"))
						fmt.Println(ui.ContentText.Render(msg))

						// Show detailed error in verbose mode (user-friendly, no timestamps)
						output.GetVerbose().Detail("Error", "%s", installErr.Error())

						GetLogger().Error("Failed to install font %s: %v", fontDisplayName, installErr)
						continue
					}

					// Clean up temp file
					os.Remove(fontPath)
					status.Installed++
					msg := fmt.Sprintf("  %s %s - %s", ui.FeedbackSuccess.Render("✓"), ui.TableRow.Render(fontDisplayName), ui.FeedbackSuccess.Render("[Installed] to "+scope+" scope"))
					GetLogger().Info("Successfully installed font: %s to %s scope", fontDisplayName, scope)

					// Verbose mode shows installation path
					finalPath := filepath.Join(fontDir, filepath.Base(fontPath))
					output.GetVerbose().Detail("Info", "Installed to: %s", finalPath)

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

		// Don't return error for installation failures since we already show detailed status report
		// This prevents duplicate error messages while maintaining proper exit codes
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
