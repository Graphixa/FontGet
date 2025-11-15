package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fontget/internal/components"
	"fontget/internal/config"
	"fontget/internal/output"
	"fontget/internal/platform"
	"fontget/internal/repo"
	"fontget/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
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

// InstallationStatus tracks the status of font installations.
// This is kept separate from OperationStatus for command-specific clarity and backward compatibility.
// It provides clearer field names (Installed vs Success) for the add command context.
type InstallationStatus struct {
	Installed int
	Skipped   int
	Failed    int
	Details   []string
	Errors    []string // Track individual error messages
}

// FontOperationDetails tracks detailed information about each font operation
type FontOperationDetails struct {
	FontName       string
	SourceName     string
	TotalVariants  int
	InstalledFiles []string
	SkippedFiles   []string
	FailedFiles    []string
	TempDir        string
	DownloadSize   int64 // Total size of downloaded files in bytes
}

// InstallResult tracks the result of installing a single font
type InstallResult struct {
	Success      int
	Skipped      int
	Failed       int
	Status       string // "completed", "failed", "skipped"
	Message      string
	Details      []string // Categorized: installed files, then skipped, then failed
	Errors       []string
	DownloadSize int64 // Total size of downloaded files in bytes
}

// FontToInstall represents a font family to be installed
type FontToInstall struct {
	Fonts      []repo.FontFile
	SourceName string
	FontName   string
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

	// Load repository for detailed font information
	repository, err := repo.GetRepository()
	if err != nil {
		// If we can't load repository, show simple list (like remove command)
		fmt.Printf("%s\n\n", ui.FeedbackText.Render("Did you mean one of these fonts?"))
		for _, font := range similar {
			fmt.Printf("  - %s\n", ui.TableSourceName.Render(font))
		}
		fmt.Println()
		return
	}

	// Collect unique matches from all suggestions using the loaded repository
	seenIDs := make(map[string]bool)
	var uniqueMatches []repo.FontMatch

	for _, suggestion := range similar {
		// Use the already-loaded repository instead of FindFontMatches
		matches := findMatchesInRepository(repository, suggestion)
		if len(matches) > 0 {
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

// findMatchesInRepository finds font matches using an already-loaded repository (performance optimization)
func findMatchesInRepository(repository *repo.Repository, fontName string) []repo.FontMatch {
	// Get the manifest from the repository
	manifest, err := repository.GetManifest()
	if err != nil {
		return nil
	}

	// Normalize font name for comparison
	fontName = strings.ToLower(fontName)
	fontNameNoSpaces := strings.ReplaceAll(fontName, " ", "")

	var matches []repo.FontMatch

	// Search through all sources
	for sourceName, source := range manifest.Sources {
		for id, font := range source.Fonts {
			// Check both the font name and ID with case-insensitive comparison
			fontNameLower := strings.ToLower(font.Name)
			idLower := strings.ToLower(id)
			fontNameNoSpacesLower := strings.ReplaceAll(fontNameLower, " ", "")
			idNoSpacesLower := strings.ReplaceAll(idLower, " ", "")

			// Check for exact match
			if fontNameLower == fontName ||
				fontNameNoSpacesLower == fontNameNoSpaces ||
				idLower == fontName ||
				idNoSpacesLower == fontNameNoSpaces {
				matches = append(matches, repo.FontMatch{
					ID:       id,
					Name:     font.Name,
					Source:   sourceName,
					FontInfo: font,
				})
			}
		}
	}

	return matches
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
		// GetLogger().Info("Starting font installation operation")

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

		// GetLogger().Info("Installation parameters - Scope: %s, Force: %v", scope, force)

		// Debug-level information for developers
		// Note: Suppressed to avoid TUI interference
		// output.GetDebug().Message("Debug mode enabled - showing detailed diagnostic information")

		// Auto-detect scope if not explicitly provided
		if scope == "" {
			isElevated, err := fontManager.IsElevated()
			if err != nil {
				GetLogger().Warn("Failed to detect elevation status: %v", err)
				// Default to user scope if we can't detect elevation
				scope = "user"
			} else if isElevated {
				scope = "machine"
				// GetLogger().Info("Auto-detected elevated privileges, defaulting to 'machine' scope")
				// Note: Suppressed to avoid TUI interference
			} else {
				scope = "user"
				// GetLogger().Info("Auto-detected user privileges, defaulting to 'user' scope")
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

		// GetLogger().Info("Processing %d font(s): %v", len(fontNames), fontNames)

		// Get font directory for the specified scope
		fontDir := fontManager.GetFontDir(installScope)
		GetLogger().Debug("Using font directory: %s", fontDir)

		// Verbose-level information for users - show operational details before progress bar
		if IsVerbose() && !IsDebug() {
			output.GetVerbose().Info("Installation scope: %s", scope)
			output.GetVerbose().Info("Font directory: %s", fontDir)
			output.GetVerbose().Info("Force mode: %v", force)
			output.GetVerbose().Info("Processing %d font family(s)", len(fontNames))
		}

		// Initialize status tracking
		status := &InstallationStatus{
			Details: make([]string, 0),
		}

		// Track detailed operations for each font (for verbose mode)
		var operationDetails []FontOperationDetails

		// Get all available fonts for suggestions (use cached version for speed)
		allFonts := repo.GetAllFontsCached()
		if len(allFonts) == 0 {
			GetLogger().Warn("Could not get list of available fonts for suggestions")
			// Note: Suppressed to avoid TUI interference
			// fmt.Println(ui.RenderError("Warning: Could not get list of available fonts for suggestions"))
		}

		// Collect all fonts first
		var fontsToInstall []FontToInstall

		// Process each font name to collect all fonts
		for _, fontName := range fontNames {
			// GetLogger().Info("Processing font: %s", fontName)

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
				similar := findSimilarFonts(fontName, allFonts, false) // false = repository fonts
				// GetLogger().Info("Found %d similar fonts for %s", len(similar), fontName)
				showFontNotFoundWithSuggestions(fontName, similar)
				continue // Skip to next font
			}

			GetLogger().Debug("Found %d font files for %s", len(fonts), fontName)

			// Add to collection
			fontsToInstall = append(fontsToInstall, FontToInstall{
				Fonts:      fonts,
				SourceName: sourceName,
				FontName:   fontName,
			})
		}

		// If no fonts to install, exit
		if len(fontsToInstall) == 0 {
			return nil
		}

		// No need for separate header - the progress bar will show the title

		// Check if flags are set
		verbose, _ := cmd.Flags().GetBool("verbose")
		debug, _ := cmd.Flags().GetBool("debug")

		// Determine scope label for verbose output
		scopeLabel := "user scope"
		if installScope == platform.MachineScope {
			scopeLabel = "machine scope"
		}

		// For debug mode: bypass TUI and use plain text output for easier parsing/logging
		if IsDebug() {
			return installFontsInDebugMode(fontManager, fontsToInstall, installScope, force, fontDir, status)
		}

		// For verbose mode: process fonts one at a time with TUI + details
		if IsVerbose() && !IsDebug() {
			return installFontsOneAtATime(fontManager, fontsToInstall, installScope, force, fontDir, status, scopeLabel)
		}

		// Create operation items for unified progress - one item per font family
		var operationItems []components.OperationItem
		for _, fontGroup := range fontsToInstall {
			// Group variants by font name
			fontName := fontGroup.Fonts[0].Name
			var variantNames []string
			for _, font := range fontGroup.Fonts {
				variantNames = append(variantNames, font.Variant)
			}

			operationItems = append(operationItems, components.OperationItem{
				Name:          fontName,
				SourceName:    fontGroup.SourceName,
				Status:        "pending",
				StatusMessage: "Pending",
				Variants:      variantNames,
				Scope:         "",
			})
		}

		// Run unified progress for download and install
		progressErr := components.RunProgressBar(
			"Installing Fonts",
			operationItems,
			list,    // List mode: show file/variant listings
			verbose, // Verbose mode: show operational details
			debug,   // Debug mode: show technical details
			func(send func(msg tea.Msg)) error {
				// Process each font group (one per font family)
				for itemIndex, fontGroup := range fontsToInstall {
					// Start downloading - update status to show we're working on this item
					send(components.ItemUpdateMsg{
						Index:   itemIndex,
						Status:  "in_progress",
						Message: "Downloading from " + fontGroup.SourceName,
					})

					// Update progress based on items started (not completed yet)
					// This shows progress as we work through items, but won't reach 100% until done
					percent := float64(itemIndex) / float64(len(fontsToInstall)) * 100
					send(components.ProgressUpdateMsg{Percent: percent})

					// Install the font using the installFont helper
					result, err := installFont(
						fontGroup.Fonts,
						fontManager,
						installScope,
						force,
						fontDir,
					)

					if err != nil {
						status.Failed += result.Failed
						GetLogger().Error("Failed to process font %s: %v", fontGroup.FontName, err)
						// Create a brief error message from the error
						errorMsg := err.Error()
						if len(errorMsg) > 100 {
							errorMsg = errorMsg[:97] + "..."
						}
						send(components.ItemUpdateMsg{
							Index:        itemIndex,
							Status:       "failed",
							Message:      "Operation failed",
							ErrorMessage: errorMsg,
						})
						continue
					}

					// Update status
					status.Installed += result.Success
					status.Skipped += result.Skipped
					status.Failed += result.Failed
					status.Errors = append(status.Errors, result.Errors...)

					// Store details for verbose mode - need to categorize files
					// Result.Details contains: installed files, then skipped, then failed
					installedCount := result.Success
					skippedCount := result.Skipped
					failedCount := result.Failed

					var installedFiles, skippedFiles, failedFiles []string
					idx := 0
					if installedCount > 0 && idx < len(result.Details) {
						installedFiles = result.Details[idx : idx+installedCount]
						idx += installedCount
					}
					if skippedCount > 0 && idx < len(result.Details) {
						skippedFiles = result.Details[idx : idx+skippedCount]
						idx += skippedCount
					}
					if failedCount > 0 && idx < len(result.Details) {
						failedFiles = result.Details[idx : idx+failedCount]
					}

					fontDetails := FontOperationDetails{
						FontName:       fontGroup.FontName,
						SourceName:     fontGroup.SourceName,
						TotalVariants:  len(fontGroup.Fonts),
						InstalledFiles: installedFiles,
						SkippedFiles:   skippedFiles,
						FailedFiles:    failedFiles,
						DownloadSize:   result.DownloadSize,
					}
					operationDetails = append(operationDetails, fontDetails)

					// Determine final status based on what happened
					scope := "user scope"
					if installScope == platform.MachineScope {
						scope = "machine scope"
					}

					// Determine status based on results
					finalStatus := result.Status

					// Build variants list with status information for verbose mode only
					var variantsWithStatus []string
					if verbose {
						// Verbose mode: Combine all file statuses with inline status
						for _, file := range fontDetails.InstalledFiles {
							variantsWithStatus = append(variantsWithStatus, file+" (installed)")
						}
						for _, file := range fontDetails.SkippedFiles {
							variantsWithStatus = append(variantsWithStatus, file+" (skipped - already installed)")
						}
						for _, file := range fontDetails.FailedFiles {
							variantsWithStatus = append(variantsWithStatus, file+" (failed)")
						}
					} else if list {
						// List mode: Just show variant names without status - collect ALL variants
						variantsWithStatus = append(variantsWithStatus, fontDetails.InstalledFiles...)
						variantsWithStatus = append(variantsWithStatus, fontDetails.SkippedFiles...)
						variantsWithStatus = append(variantsWithStatus, fontDetails.FailedFiles...)
					}
					// else: Default mode - don't show variants at all

					// Get first error message if status is failed
					var errorMsg string
					if finalStatus == "failed" && len(result.Errors) > 0 {
						errorMsg = result.Errors[0]
					}

					send(components.ItemUpdateMsg{
						Index:        itemIndex,
						Status:       finalStatus,
						Message:      "Installed", // Message is overridden by View() based on status
						ErrorMessage: errorMsg,
						Variants:     variantsWithStatus,
						Scope:        scope,
					})

					// Update progress percentage - now based on actual completion
					percent = float64(itemIndex+1) / float64(len(fontsToInstall)) * 100
					send(components.ProgressUpdateMsg{Percent: percent})
				}

				return nil
			},
		)

		if progressErr != nil {
			GetLogger().Error("Failed to install fonts: %v", progressErr)
			return progressErr
		}

		// Verbose-level information after progress bar - show final operational details and errors
		if IsVerbose() && !IsDebug() {
			// Show variant details for each font (verbose only, not debug)
			for _, details := range operationDetails {
				if details.TotalVariants > 0 {
					// Extract variant name helper function
					extractVariant := func(file, fontName string) string {
						displayName := GetDisplayNameFromFilename(file)
						variantName := displayName
						if strings.HasPrefix(displayName, fontName) {
							variantName = strings.TrimSpace(strings.TrimPrefix(displayName, fontName))
							if variantName == "" {
								variantName = "Regular"
							}
						}
						return variantName
					}

					output.GetVerbose().DisplayFontOperationDetails(
						details.FontName,
						details.SourceName,
						details.InstalledFiles,
						details.SkippedFiles,
						details.FailedFiles,
						FormatFileSize(details.DownloadSize),
						fontDir,
						scopeLabel,
						extractVariant,
					)
				}
			}
			fmt.Println() // Blank line before status report

			// Show error details in verbose mode - one inline message per failed font
			if len(status.Errors) > 0 {
				// Get the unique font families that failed by tracking what was processed
				// We need to map errors back to the font families
				if len(fontsToInstall) > 0 && status.Failed > 0 {
					// Find which font families failed and count variants per family
					failedFamilies := make(map[string]int)
					for _, err := range status.Errors {
						// Extract base font name from error message
						parts := strings.Split(err, " ")
						if len(parts) > 0 {
							fontFile := strings.TrimSpace(parts[0])
							fontFile = strings.TrimSuffix(fontFile, ".ttf")
							fontParts := strings.Split(fontFile, "-")
							if len(fontParts) > 0 {
								familyName := fontParts[0]
								failedFamilies[familyName]++
							}
						}
					}

					// Show one error message per failed font family with variant count
					for name, variantCount := range failedFamilies {
						var forceMsg string
						variantText := ""
						if variantCount > 1 {
							variantText = fmt.Sprintf(" (%d variants)", variantCount)
						}
						if force {
							forceMsg = fmt.Sprintf("\"%s\"%s could not be removed for reinstallation as it's being used by another application. Try closing the app or process using this font and try again.", name, variantText)
						} else {
							forceMsg = fmt.Sprintf("\"%s\"%s could not be installed as it's in use by another application. Try closing the app or process using this font and try again.", name, variantText)
						}
						output.GetVerbose().Error("%s", forceMsg)
					}
				}
			}
		}

		// Add blank line after progress bar output (for clean spacing before command prompt)
		if !IsVerbose() {
			fmt.Println()
		}

		// Print status report after progress bar completes (this should be last)
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

// installFontsInDebugMode processes fonts with plain text output (no TUI) for easier parsing/logging
func installFontsInDebugMode(fontManager platform.FontManager, fontsToInstall []FontToInstall, installScope platform.InstallationScope, force bool, fontDir string, status *InstallationStatus) error {
	output.GetDebug().State("Starting font installation operation")
	output.GetDebug().State("Total fonts: %d", len(fontsToInstall))

	// Process each font
	for i, fontGroup := range fontsToInstall {
		output.GetDebug().State("Processing font %d/%d: %s", i+1, len(fontsToInstall), fontGroup.FontName)

		result, err := installFont(
			fontGroup.Fonts,
			fontManager,
			installScope,
			force,
			fontDir,
		)

		if err != nil {
			output.GetDebug().State("Error processing font %s: %v", fontGroup.FontName, err)
			if result != nil {
				status.Failed += result.Failed
				status.Skipped += result.Skipped
				status.Errors = append(status.Errors, result.Errors...)
			}
			continue
		}

		// Update status
		status.Installed += result.Success
		status.Skipped += result.Skipped
		status.Failed += result.Failed
		status.Errors = append(status.Errors, result.Errors...)

		output.GetDebug().State("Font %s completed: %s - %s", fontGroup.FontName, result.Status, result.Message)
	}

	output.GetDebug().State("Operation complete - Installed: %d, Skipped: %d, Failed: %d",
		status.Installed, status.Skipped, status.Failed)

	// Print status report
	fmt.Println()
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
	return nil
}

// installFontsOneAtATime processes fonts one at a time with TUI + variant details
func installFontsOneAtATime(fontManager platform.FontManager, fontsToInstall []FontToInstall, installScope platform.InstallationScope, force bool, fontDir string, status *InstallationStatus, scopeLabel string) error {
	// Process each font one at a time with its own TUI
	for _, fontGroup := range fontsToInstall {
		fontName := fontGroup.Fonts[0].Name

		// Create operation item for just this one font
		var variantNames []string
		for _, font := range fontGroup.Fonts {
			variantNames = append(variantNames, font.Variant)
		}

		operationItem := components.OperationItem{
			Name:          fontName,
			SourceName:    fontGroup.SourceName,
			Status:        "pending",
			StatusMessage: "Pending",
			Variants:      variantNames,
			Scope:         scopeLabel,
		}

		// Store result outside closure for verbose output
		var result *InstallResult

		// Run TUI for this one font
		progressErr := components.RunProgressBar(
			"Installing Fonts",
			[]components.OperationItem{operationItem},
			false, // List mode
			true,  // Verbose mode
			false, // Debug mode
			func(send func(msg tea.Msg)) error {
				// Install the font using the installFont helper
				var err error
				result, err = installFont(
					fontGroup.Fonts,
					fontManager,
					installScope,
					force,
					fontDir,
				)

				if err != nil {
					status.Failed += result.Failed
					// Create a brief error message from the error
					errorMsg := err.Error()
					if len(errorMsg) > 100 {
						errorMsg = errorMsg[:97] + "..."
					}
					send(components.ItemUpdateMsg{
						Index:        0,
						Status:       "failed",
						Message:      "Operation failed",
						ErrorMessage: errorMsg,
					})
					return nil
				}

				// Update status
				status.Installed += result.Success
				status.Skipped += result.Skipped
				status.Failed += result.Failed
				status.Errors = append(status.Errors, result.Errors...)

				// Store download size for verbose display
				_ = result.DownloadSize // Will be used in verbose display

				// Get first error message if status is failed
				var errorMsg string
				if result.Status == "failed" && len(result.Errors) > 0 {
					errorMsg = result.Errors[0]
				}

				// Update TUI
				send(components.ItemUpdateMsg{
					Index:        0,
					Status:       result.Status,
					Message:      result.Message,
					ErrorMessage: errorMsg,
					Variants:     []string{},
					Scope:        scopeLabel,
				})

				// Update progress to 100%
				send(components.ProgressUpdateMsg{Percent: 100})
				return nil
			})

		if progressErr != nil {
			return progressErr
		}

		// Categorize files from result for verbose output
		if result == nil {
			continue
		}

		installedCount := result.Success
		skippedCount := result.Skipped
		failedCount := result.Failed

		var installedFiles, skippedFiles, failedFiles []string
		idx := 0
		if installedCount > 0 && idx < len(result.Details) {
			installedFiles = result.Details[idx : idx+installedCount]
			idx += installedCount
		}
		if skippedCount > 0 && idx < len(result.Details) {
			skippedFiles = result.Details[idx : idx+skippedCount]
			idx += skippedCount
		}
		if failedCount > 0 && idx < len(result.Details) {
			failedFiles = result.Details[idx : idx+failedCount]
		}

		// After TUI completes, append variant details (only in verbose mode)
		if IsVerbose() && !IsDebug() {
			output.GetVerbose().DisplayFontOperationDetails(
				fontName,
				fontGroup.SourceName,
				installedFiles,
				skippedFiles,
				failedFiles,
				FormatFileSize(result.DownloadSize),
				fontDir,
				scopeLabel,
				extractVariantNameFromFile,
			)
		}
	}

	// Show Status Report
	fmt.Println()
	PrintStatusReport(StatusReport{
		Success:      status.Installed,
		Skipped:      status.Skipped,
		Failed:       status.Failed,
		SuccessLabel: "Installed",
		SkippedLabel: "Skipped",
		FailedLabel:  "Failed",
	})

	return nil
}

// extractVariantNameFromFile extracts just the variant name from a font filename
func extractVariantNameFromFile(filename string, fontName string) string {
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)

	// Check if this is just the base font name (e.g., "ABeeZee.ttf" or "RobotoMono.ttf")
	// Normalize by removing spaces for comparison
	normalizedFilename := strings.ReplaceAll(nameWithoutExt, " ", "")
	normalizedFontName := strings.ReplaceAll(fontName, " ", "")
	if strings.EqualFold(normalizedFilename, normalizedFontName) {
		return "Regular"
	}

	// Try to extract variant after font name + hyphen
	// e.g., "ABeeZee-Italic.ttf" → fontName is "ABeeZee" → variant is "Italic"
	if strings.HasPrefix(strings.ToLower(nameWithoutExt), strings.ToLower(normalizedFontName)+"-") {
		variantPart := strings.TrimPrefix(nameWithoutExt, normalizedFontName)
		variantPart = strings.TrimPrefix(variantPart, "-")
		if variantPart != "" {
			return convertCamelCaseToSpaced(variantPart)
		}
	}

	// Fallback: check if display name equals font name (i.e., Regular variant)
	displayName := GetDisplayNameFromFilename(filename)
	if strings.EqualFold(displayName, fontName) {
		return "Regular"
	}

	// Extract variant from display name
	if strings.HasPrefix(displayName, fontName) {
		variantPart := strings.TrimSpace(strings.TrimPrefix(displayName, fontName))
		if variantPart != "" {
			return variantPart
		}
	}

	return displayName
}

// installFont handles the core installation logic for a single font
func installFont(
	fontFiles []repo.FontFile,
	fontManager platform.FontManager,
	installScope platform.InstallationScope,
	force bool,
	fontDir string,
) (*InstallResult, error) {
	result := &InstallResult{
		Details: make([]string, 0),
		Errors:  make([]string, 0),
	}

	// Download all variants of this font family
	tempDir := filepath.Join(os.TempDir(), "Fontget", "fonts")
	output.GetDebug().State("Temp directory: %s", tempDir)

	var allFontPaths []string
	var downloadErr error

	// Download each variant - only log errors and unusual cases
	for _, fontFile := range fontFiles {
		output.GetDebug().State("Calling repo.DownloadAndExtractFont() for variant: %s from %s", fontFile.Variant, fontFile.DownloadURL)
		fontPaths, err := repo.DownloadAndExtractFont(&fontFile, tempDir)
		if err != nil {
			downloadErr = err
			output.GetDebug().State("repo.DownloadAndExtractFont() failed for variant %s: %v", fontFile.Variant, err)
			break
		}
		allFontPaths = append(allFontPaths, fontPaths...)
		// Only log if multiple files extracted (unusual case worth noting)
		if len(fontPaths) > 1 {
			output.GetDebug().State("Extracted %d file(s) from variant: %s", len(fontPaths), fontFile.Variant)
		}
	}

	if downloadErr != nil {
		result.Status = "failed"
		result.Message = "Download failed"
		result.Failed = len(fontFiles)
		return result, downloadErr
	}

	// Install downloaded fonts
	var installedFiles []string
	var skippedFiles []string
	var failedFiles []string

	for _, fontPath := range allFontPaths {
		fontDisplayName := filepath.Base(fontPath)

		// Get file size before we potentially remove it
		if fileInfo, err := os.Stat(fontPath); err == nil {
			result.DownloadSize += fileInfo.Size()
		}

		// Check if font is already installed (unless force flag is set)
		if !force {
			expectedPath := filepath.Join(fontDir, fontDisplayName)
			if _, err := os.Stat(expectedPath); err == nil {
				result.Skipped++
				os.Remove(fontPath) // Clean up temp file
				skippedFiles = append(skippedFiles, fontDisplayName)
				continue
			}
		}

		// Install the font
		installErr := fontManager.InstallFont(fontPath, installScope, force)

		if installErr != nil {
			os.Remove(fontPath) // Clean up temp file
			result.Failed++
			errorMsg := makeUserFriendlyError(fontDisplayName, installErr)
			result.Errors = append(result.Errors, errorMsg)
			failedFiles = append(failedFiles, fontDisplayName)
			output.GetDebug().State("fontManager.InstallFont() failed for %s: %v", fontDisplayName, installErr)
			continue
		}

		// Clean up temp file
		os.Remove(fontPath)
		result.Success++
		installedFiles = append(installedFiles, fontDisplayName)
	}

	// Store categorized details
	result.Details = append(result.Details, installedFiles...)
	result.Details = append(result.Details, skippedFiles...)
	result.Details = append(result.Details, failedFiles...)

	// Determine final status
	if result.Success > 0 {
		result.Status = "completed"
		result.Message = "Installed"
	} else if result.Failed > 0 {
		result.Status = "failed"
		result.Message = "Installation failed"
	} else if result.Skipped > 0 {
		result.Status = "skipped"
		result.Message = "Already installed"
	}

	return result, nil
}

// makeUserFriendlyError converts technical error messages to user-friendly explanations
func makeUserFriendlyError(fontName string, err error) string {
	errStr := strings.ToLower(err.Error())

	// Check for common error patterns and provide user-friendly messages
	if strings.Contains(errStr, "cannot access the file because it is being used") {
		return fmt.Sprintf("%s could not be installed as it's in use by another application. Try closing the app or process using this font and try again.", fontName)
	}

	if strings.Contains(errStr, "access denied") {
		return fmt.Sprintf("%s could not be installed due to access denied. You may need administrator privileges.", fontName)
	}

	if strings.Contains(errStr, "already exists") {
		return fmt.Sprintf("%s is already installed.", fontName)
	}

	if strings.Contains(errStr, "file in use") {
		return fmt.Sprintf("%s could not be installed as it's in use by another application. Try closing the app or process using this font and try again.", fontName)
	}

	// For unknown errors, show a simplified version
	return fmt.Sprintf("%s could not be installed. Check logs for details.", fontName)
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringP("scope", "s", "", "Installation scope (user or machine)")
	addCmd.Flags().BoolP("force", "f", false, "Force installation even if font is already installed")
}
