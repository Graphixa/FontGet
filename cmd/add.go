package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fontget/internal/cmdutils"
	"fontget/internal/components"
	"fontget/internal/output"
	"fontget/internal/platform"
	"fontget/internal/repo"
	"fontget/internal/shared"
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

// Status constants for installation operations
const (
	InstallStatusCompleted = "completed"
	InstallStatusFailed    = "failed"
	InstallStatusSkipped   = "skipped"
)

// Operation message constants
const (
	OpInstallingFonts         = "Installing Fonts"
	OpInstallingFontsAllUsers = "Installing Fonts for All Users"
)

// Scope label constants
const (
	InstallScopeLabelUser    = "user scope"
	InstallScopeLabelMachine = "machine scope"
)

// Placeholder constants
const (
	PlaceholderNA = "N/A"
)

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
	FontID     string // Font ID for checking if already installed
}

// showGroupedFontNotFoundWithSuggestions displays all not-found fonts grouped together with consolidated suggestions
func showGroupedFontNotFoundWithSuggestions(notFoundFonts []string) {
	// Show all not-found fonts first
	if len(notFoundFonts) == 1 {
		// Single font - use the original function for consistency
		allFonts := repo.GetAllFontsCached()
		var similar []string
		if len(allFonts) > 0 {
			similar = shared.FindSimilarFonts(notFoundFonts[0], allFonts, false) // false = repository fonts
		}
		showFontNotFoundWithSuggestions(notFoundFonts[0], similar)
		return
	}

	// Multiple fonts - show grouped format
	fmt.Printf("%s\n", ui.ErrorText.Render("The following font(s) were not found:"))
	for _, fontName := range notFoundFonts {
		fmt.Printf("  - %s\n", fontName)
	}

	// Collect all suggestions for all fonts
	allFonts := repo.GetAllFontsCached()
	allSimilar := []string{}
	seenSimilar := make(map[string]bool)

	for _, fontName := range notFoundFonts {
		var similar []string
		if len(allFonts) > 0 {
			similar = shared.FindSimilarFonts(fontName, allFonts, false) // false = repository fonts
		}
		// Deduplicate suggestions
		for _, suggestion := range similar {
			if !seenSimilar[suggestion] {
				seenSimilar[suggestion] = true
				allSimilar = append(allSimilar, suggestion)
			}
		}
	}

	// Show consolidated suggestions if any
	if len(allSimilar) > 0 {
		fmt.Println()
		fmt.Printf("%s\n\n", ui.Text.Render("Did you mean one of these fonts?"))

		// Load repository for detailed font information
		repository, err := repo.GetRepository()
		if err != nil {
			// If we can't load repository, show simple list (limit to 12)
			const maxSuggestions = 12
			for i, font := range allSimilar {
				if i >= maxSuggestions {
					break
				}
				fmt.Printf("  - %s\n", ui.TableSourceName.Render(font))
			}
			fmt.Println()
			return
		}

		// Collect unique matches from all suggestions using the loaded repository
		// Limit to 12 suggestions total to avoid overwhelming output
		const maxSuggestions = 12
		seenIDs := make(map[string]bool)
		var uniqueMatches []repo.FontMatch

		for _, suggestion := range allSimilar {
			if len(uniqueMatches) >= maxSuggestions {
				break
			}
			matches := findMatchesInRepository(repository, suggestion)
			if len(matches) > 0 {
				for _, match := range matches {
					if len(uniqueMatches) >= maxSuggestions {
						break
					}
					if !seenIDs[match.ID] {
						uniqueMatches = append(uniqueMatches, match)
						seenIDs[match.ID] = true
					}
				}
			}
		}

		// Display matches in table format
		if len(uniqueMatches) > 0 {
			fmt.Printf("%s\n", ui.GetSearchTableHeader())
			fmt.Printf("%s\n", ui.GetTableSeparator())

			for _, match := range uniqueMatches {
				categories := PlaceholderNA
				if len(match.FontInfo.Categories) > 0 {
					categories = match.FontInfo.Categories[0]
				}

				license := match.FontInfo.License
				if license == "" {
					license = PlaceholderNA
				}

				fmt.Printf("%s %-*s %-*s %-*s %-*s\n",
					ui.TableSourceName.Render(fmt.Sprintf("%-*s", ui.TableColName, shared.TruncateString(match.Name, ui.TableColName))),
					ui.TableColID, shared.TruncateString(match.ID, ui.TableColID),
					ui.TableColLicense, shared.TruncateString(license, ui.TableColLicense),
					ui.TableColCategories, shared.TruncateString(categories, ui.TableColCategories),
					ui.TableColSource, shared.TruncateString(match.Source, ui.TableColSource))
			}
			fmt.Println()
		} else {
			// No matches found, show general guidance
			fmt.Printf("%s\n", ui.Text.Render("Try using the search command to find available fonts."))
			fmt.Printf("\n%s\n", ui.Text.Render("Example:"))
			fmt.Printf("  %s\n\n", ui.Text.Render("fontget search \"roboto\""))
		}
	} else {
		// No suggestions at all, show general guidance
		fmt.Println()
		fmt.Printf("%s\n", ui.Text.Render("Try using the search command to find available fonts."))
		fmt.Printf("\n%s\n", ui.Text.Render("Example:"))
		fmt.Printf("  %s\n\n", ui.Text.Render("fontget search \"roboto\""))
	}
}

// showFontNotFoundWithSuggestions displays font not found error with suggestions in table format
func showFontNotFoundWithSuggestions(fontName string, similar []string) {
	fmt.Printf("%s\n", ui.ErrorText.Render(fmt.Sprintf("Font '%s' not found.", fontName)))
	// If no similar fonts found, show general guidance
	if len(similar) == 0 {
		fmt.Printf("%s\n", ui.Text.Render("Try using the search command to find available fonts."))
		fmt.Printf("\n%s\n", ui.Text.Render("Example:"))
		fmt.Printf("  %s\n\n", ui.Text.Render("fontget search \"roboto\""))
		return
	}

	// Load repository for detailed font information
	repository, err := repo.GetRepository()
	if err != nil {
		// If we can't load repository, show simple list (like remove command)
		fmt.Printf("%s\n\n", ui.Text.Render("Did you mean one of these fonts?"))
		for _, font := range similar {
			fmt.Printf("  - %s\n", ui.TableSourceName.Render(font))
		}
		fmt.Printf("\n")
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
		fmt.Printf("%s\n\n", ui.Text.Render("Did you mean one of these fonts?"))
		// Use consistent column widths and apply styling to the entire formatted string
		fmt.Printf("%s\n", ui.GetSearchTableHeader())
		fmt.Printf("%s\n", ui.GetTableSeparator())

		// Display each unique match as a table row
		for _, match := range uniqueMatches {
			// Get categories (first one if available)
			categories := PlaceholderNA
			if len(match.FontInfo.Categories) > 0 {
				categories = match.FontInfo.Categories[0]
			}

			// Get license
			license := match.FontInfo.License
			if license == "" {
				license = PlaceholderNA
			}

			// Format the data line consistently with yellow font name
			fmt.Printf("%s %-*s %-*s %-*s %-*s\n",
				ui.TableSourceName.Render(fmt.Sprintf("%-*s", ui.TableColName, shared.TruncateString(match.Name, ui.TableColName))),
				ui.TableColID, shared.TruncateString(match.ID, ui.TableColID),
				ui.TableColLicense, shared.TruncateString(license, ui.TableColLicense),
				ui.TableColCategories, shared.TruncateString(categories, ui.TableColCategories),
				ui.TableColSource, shared.TruncateString(match.Source, ui.TableColSource))
		}
		fmt.Println()
	} else {
		// Fallback: if similar font names were found but couldn't be resolved to matches
		// This suggests the font exists in our cache but is no longer available from sources
		fmt.Printf("%s\n", ui.ErrorText.Render(fmt.Sprintf("Font '%s' was not able to be downloaded and installed.", fontName)))
		fmt.Printf("%s\n", ui.Text.Render("It may have been removed from the font source."))
		fmt.Printf("\n%s\n", ui.Text.Render("Please refresh FontGet sources using:"))
		fmt.Printf("  %s\n", ui.Text.Render("fontget sources update"))
		fmt.Printf("\n%s\n", ui.Text.Render("Try using the search command to find other available fonts:"))
		fmt.Printf("  %s\n\n", ui.Text.Render("fontget search \"roboto\""))
	}
}

// resolveAndValidateFonts resolves font queries and validates them, returning fonts to install and not found fonts.
func resolveAndValidateFonts(fontNames []string) (fontsToInstall []FontToInstall, notFoundFonts []string) {
	for _, fontName := range fontNames {
		GetLogger().Info("Processing font: %s", fontName)

		// Resolve font query (Font ID or name) to FontFile list
		result, err := shared.ResolveFontQuery(fontName)
		if err != nil {
			// This is a query error, not an installation failure
			GetLogger().Error("Font not found: %s", fontName)
			output.GetDebug().Error("Font not found in repository: %s", fontName)
			// Collect for later display instead of showing immediately
			notFoundFonts = append(notFoundFonts, fontName)
			continue // Skip to next font
		}

		// Handle multiple matches case
		if result.HasMultipleMatches {
			// Multiple matches - show search results and prompt for specific ID
			showMultipleMatchesAndExit(fontName, result.Matches)
			return nil, notFoundFonts // Exit early - caller should handle this
		}

		GetLogger().Debug("Found %d font files for %s", len(result.Fonts), fontName)

		// Add to collection (fonts will be checked for installation status during installFont)
		fontsToInstall = append(fontsToInstall, FontToInstall{
			Fonts:      result.Fonts,
			SourceName: result.SourceName,
			FontName:   fontName,
			FontID:     result.FontID, // Store font ID for checking if already installed
		})
	}
	return fontsToInstall, notFoundFonts
}

// setupInstallationProgressBar creates operation items for the progress bar.
func setupInstallationProgressBar(fontsToInstall []FontToInstall) []components.OperationItem {
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
	return operationItems
}

// handleNotFoundFonts displays not found fonts with suggestions.
func handleNotFoundFonts(notFoundFonts []string, isDebug bool) {
	if len(notFoundFonts) == 0 {
		return
	}

	if isDebug {
		// In debug mode, show technical details to console
		output.GetDebug().Error("The following fonts were not found in any source:")
		for _, fontName := range notFoundFonts {
			output.GetDebug().Error(" - %s", fontName)
		}
	} else {
		// In normal/verbose mode, show user-friendly message
		fmt.Printf("%s\n", ui.ErrorText.Render("The following fonts were not found in any source:"))
		for _, fontName := range notFoundFonts {
			fmt.Printf("%s\n", ui.ErrorText.Render(fmt.Sprintf("  - %s", fontName)))
		}
		// Add blank line before "Try using..." message (within section)
		fmt.Println()
		fmt.Printf("Try using 'fontget search' to find available fonts.\n")
		// Section ends with blank line per spacing framework
		fmt.Println()
	}
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

	fmt.Printf("\n%s\n", ui.InfoText.Render(fmt.Sprintf("Multiple fonts found matching '%s'.", ui.QueryText.Render(fontName))))
	fmt.Printf("%s\n\n", ui.Text.Render("Please specify the exact font ID to install from a specific source."))

	// Use consistent column widths and apply styling to the entire formatted string
	fmt.Printf("%s\n", ui.GetSearchTableHeader())
	fmt.Printf("%s\n", ui.GetTableSeparator())

	for _, match := range matches {
		// Get categories (first one if available)
		categories := PlaceholderNA
		if len(match.FontInfo.Categories) > 0 {
			categories = match.FontInfo.Categories[0]
		}

		// Get license
		license := match.FontInfo.License
		if license == "" {
			license = PlaceholderNA
		}

		// Format the data line consistently with yellow font name
		fmt.Printf("%s %-*s %-*s %-*s %-*s\n",
			ui.TableSourceName.Render(fmt.Sprintf("%-*s", ui.TableColName, shared.TruncateString(match.Name, ui.TableColName))),
			ui.TableColID, shared.TruncateString(match.ID, ui.TableColID),
			ui.TableColLicense, shared.TruncateString(license, ui.TableColLicense),
			ui.TableColCategories, shared.TruncateString(categories, ui.TableColCategories),
			ui.TableColSource, shared.TruncateString(match.Source, ui.TableColSource))
	}

	fmt.Printf("\n")
}

var addCmd = &cobra.Command{
	Use:          "add <font-id> [<font-id2> <font-id3> ...]",
	Aliases:      []string{"install"},
	Short:        "Install fonts from configured sources",
	SilenceUsage: true,
	Long: `Install fonts from configured sources. Install one or multiple fonts in a single command.

Fonts can be specified by name (e.g., "Roboto") or Font ID (e.g., "google.roboto").

Font names with spaces must be wrapped in quotes (e.g., "Open Sans").

Installation scope can be specified with the --scope flag:
  - user (default): Install for current user only
  - machine: Install system-wide (requires administrator privileges)

`,
	Example: `  fontget add "Roboto"
  fontget add "google.roboto"
  fontget add "Open Sans" "Fira Sans" "Noto Sans"
  fontget add "roboto firasans notosans"
  fontget add "Open Sans" -s machine
  fontget add "roboto" -f`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Only handle empty query case
		if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
			fmt.Printf("%s\n", ui.RenderError("A font ID is required"))
			fmt.Printf("Use 'fontget add --help' for more information.\n\n")
			return nil
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		GetLogger().Info("Starting font installation operation")

		// Always start with a blank line for consistent spacing from command prompt
		fmt.Println()

		// Ensure manifest system is initialized (fixes missing sources.json bug)
		if err := cmdutils.EnsureManifestInitialized(func() cmdutils.Logger { return GetLogger() }); err != nil {
			return err
		}

		// Double check args to prevent panic
		if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
			return nil // Args validator will have already shown the help
		}

		// Create font manager
		fontManager, err := cmdutils.CreateFontManager(func() cmdutils.Logger { return GetLogger() })
		if err != nil {
			return err
		}

		// Get scope from flag
		scope, _ := cmd.Flags().GetString("scope")
		force, _ := cmd.Flags().GetBool("force")

		// Log installation parameters (always log to file)
		GetLogger().Info("Installation parameters - Scope: %s, Force: %v", scope, force)

		// Debug-level information for developers
		// Note: Suppressed to avoid TUI interference
		// output.GetDebug().Message("Debug mode enabled - showing detailed diagnostic information")

		// Auto-detect scope if not explicitly provided
		if scope == "" {
			var err error
			scope, err = platform.AutoDetectScope(fontManager, "user", "machine", GetLogger())
			if err != nil {
				// Should not happen, but handle gracefully
				scope = "user"
			}
		}

		// Convert scope string to InstallationScope
		installScope := platform.UserScope
		if scope != "user" {
			installScope = platform.InstallationScope(scope)
			if installScope != platform.UserScope && installScope != platform.MachineScope {
				err := fmt.Errorf("invalid scope '%s'. Valid options are: 'user' or 'machine'", scope)
				output.GetVerbose().Error("%v", err)
				output.GetDebug().Error("Invalid scope provided: '%s'", scope)
				return err
			}
		}

		// Check elevation
		if err := cmdutils.CheckElevation(cmd, fontManager, installScope); err != nil {
			if errors.Is(err, cmdutils.ErrElevationRequired) {
				return nil // Already printed user-friendly message
			}
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("checkElevation() failed: %v", err)
			return fmt.Errorf("unable to verify system permissions: %v", err)
		}

		// Process font names from arguments
		fontNames := cmdutils.ParseFontNames(args)

		GetLogger().Info("Processing %d font(s): %v", len(fontNames), fontNames)

		// Get font directory for the specified scope
		fontDir := fontManager.GetFontDir(installScope)
		GetLogger().Debug("Using font directory: %s", fontDir)

		// Get all available fonts for suggestions (use cached version for speed)
		allFonts := repo.GetAllFontsCached()
		if len(allFonts) == 0 {
			GetLogger().Warn("Could not get list of available fonts for suggestions")
			// Note: Suppressed to avoid TUI interference
			// fmt.Println(ui.RenderError("Warning: Could not get list of available fonts for suggestions"))
		}

		// Collect all fonts first
		fontsToInstall, notFoundFonts := resolveAndValidateFonts(fontNames)

		// Check for multiple matches (would have been handled in resolveAndValidateFonts)
		if fontsToInstall == nil {
			return nil // Multiple matches case - already shown
		}

		// Check if flags are set
		verbose, _ := cmd.Flags().GetBool("verbose")
		debug, _ := cmd.Flags().GetBool("debug")

		// If no fonts to install, show not found message with suggestions and exit
		// Do this BEFORE verbose output to avoid extra blank lines
		if len(fontsToInstall) == 0 {
			if len(notFoundFonts) > 0 {
				if IsDebug() {
					// In debug mode, show technical details to console
					output.GetDebug().Error("No fonts found to install. The following font(s) were not found in any source:")
					for _, fontName := range notFoundFonts {
						output.GetDebug().Error(" - %s", fontName)
					}
				} else {
					// In normal/verbose mode, show user-friendly message with suggestions
					showGroupedFontNotFoundWithSuggestions(notFoundFonts)
				}
			}
			return nil
		}

		// Verbose-level information for users - show operational details before progress bar
		// Format scope label for display
		scopeDisplay := scope
		if scope == "" {
			scopeDisplay = "user"
		}
		output.GetVerbose().Info("Scope: %s", scopeDisplay)
		output.GetVerbose().Info("Force mode: %v", force)
		output.GetVerbose().Info("Installing %d font(s)", len(fontNames))
		// Verbose section ends with blank line per spacing framework (only if verbose was shown)
		if IsVerbose() {
			fmt.Println()
		}

		// Initialize status tracking
		status := &InstallationStatus{
			Details: make([]string, 0),
		}

		// Track detailed operations for each font (for verbose mode)
		var operationDetails []FontOperationDetails

		// No need for separate header - the progress bar will show the title

		// Log installation parameters after scope is determined (for debug mode)
		if IsDebug() {
			scopeDisplay := scope
			if scopeDisplay == "" {
				scopeDisplay = "user"
			}
			GetLogger().Info("Installation parameters - Scope: %s, Force: %v", scopeDisplay, force)
			// Format font list as a bulleted list
			GetLogger().Info("Installing %d Font(s):", len(fontNames))
			for _, fontName := range fontNames {
				GetLogger().Info("  - %s", fontName)
			}
			// Show not found fonts in debug mode (before processing)
			if len(notFoundFonts) > 0 {
				GetLogger().Info("The following font(s) were not found in any source:")
				for _, fontName := range notFoundFonts {
					GetLogger().Info("  - %s", fontName)
				}
			}
		}

		// For debug mode: bypass TUI and use plain text output for easier parsing/logging
		if IsDebug() {
			return installFontsInDebugMode(fontManager, fontsToInstall, installScope, force, fontDir, status, scope)
		}

		// Create operation items for unified progress - one item per font family
		operationItems := setupInstallationProgressBar(fontsToInstall)

		// Determine title based on scope
		title := OpInstallingFonts
		if installScope == platform.MachineScope {
			title = OpInstallingFontsAllUsers
		}

		// Run unified progress for download and install
		progressErr := components.RunProgressBar(
			title,
			operationItems,
			verbose, // Verbose mode: show operational details and file/variant listings
			debug,   // Debug mode: show technical details
			func(send func(msg tea.Msg), cancelChan <-chan struct{}) error {
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
						fontGroup.FontID,
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
							Status:       InstallStatusFailed,
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

					// Determine status based on results
					finalStatus := result.Status

					// Build variants list - show in verbose mode
					var variantsWithStatus []string
					if verbose {
						// Verbose mode: Show variant names - collect ALL variants
						variantsWithStatus = append(variantsWithStatus, fontDetails.InstalledFiles...)
						variantsWithStatus = append(variantsWithStatus, fontDetails.SkippedFiles...)
						variantsWithStatus = append(variantsWithStatus, fontDetails.FailedFiles...)
					}
					// Default mode: don't show variants in TUI (variants shown in debug mode only)

					// Get first error message if status is failed
					var errorMsg string
					if finalStatus == InstallStatusFailed && len(result.Errors) > 0 {
						errorMsg = result.Errors[0]
					}

					// Determine scope label for display
					scopeLabel := InstallScopeLabelUser
					if installScope == platform.MachineScope {
						scopeLabel = InstallScopeLabelMachine
					}

					send(components.ItemUpdateMsg{
						Index:        itemIndex,
						Status:       finalStatus,
						Message:      "Installed", // Message is overridden by View() based on status
						ErrorMessage: errorMsg,
						Variants:     variantsWithStatus,
						Scope:        scopeLabel,
					})

					// Update progress percentage - now based on actual completion
					percent = float64(itemIndex+1) / float64(len(fontsToInstall)) * 100
					send(components.ProgressUpdateMsg{Percent: percent})
				}

				return nil
			},
		)

		if progressErr != nil {
			// Check if it was a cancellation
			if errors.Is(progressErr, shared.ErrOperationCancelled) {
				fmt.Printf("%s\n", ui.WarningText.Render("Installation cancelled."))
				fmt.Println()
				return nil // Don't return error for cancellation
			}
			GetLogger().Error("Failed to install fonts: %v", progressErr)
			return progressErr
		}

		// Show not found fonts right after progress bar output (before status report)
		handleNotFoundFonts(notFoundFonts, IsDebug())

		// Note: Error messages for failed installations are already shown in the progress bar
		// No need to duplicate them here - verbose mode should be user-friendly, not technical

		// Print status report after progress bar completes (this should be last)
		output.PrintStatusReport(output.StatusReport{
			Success:      status.Installed,
			Skipped:      status.Skipped,
			Failed:       status.Failed,
			SuccessLabel: "Installed",
			SkippedLabel: "Skipped",
			FailedLabel:  "Failed",
		}, IsVerbose())

		GetLogger().Info("Installation complete - Installed: %d, Skipped: %d, Failed: %d",
			status.Installed, status.Skipped, status.Failed)

		// Don't return error for installation failures since we already show detailed status report
		// This prevents duplicate error messages while maintaining proper exit codes
		return nil
	},
}

// processInstallResult processes and categorizes install result details (installed/skipped/failed variants)
func processInstallResult(result *InstallResult) (installedFiles, skippedFiles, failedFiles []string) {
	if result == nil || len(result.Details) == 0 {
		return nil, nil, nil
	}

	installedCount := result.Success
	skippedCount := result.Skipped
	failedCount := result.Failed

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

	return installedFiles, skippedFiles, failedFiles
}

// logInstallResultDetails logs detailed variant information in debug mode
func logInstallResultDetails(result *InstallResult, fontName, scopeLabel string) {
	if result == nil {
		return
	}

	installedFiles, skippedFiles, failedFiles := processInstallResult(result)

	if len(installedFiles) > 0 {
		output.GetDebug().State("Installed variants:")
		for _, file := range installedFiles {
			output.GetDebug().State(" - %s", file)
		}
	}
	if len(skippedFiles) > 0 {
		output.GetDebug().State("Skipped variants:")
		for _, file := range skippedFiles {
			output.GetDebug().State(" - %s", file)
		}
	}
	if len(failedFiles) > 0 {
		output.GetDebug().State("Failed variants:")
		for _, file := range failedFiles {
			output.GetDebug().State(" - %s", file)
		}
	}

	output.GetDebug().State("Font %s in %s completed: %s - %s (Installed: %d, Skipped: %d, Failed: %d)",
		fontName, scopeLabel, result.Status, result.Message, result.Success, result.Skipped, result.Failed)
}

// updateInstallStatus updates installation status from result
func updateInstallStatus(status *InstallationStatus, result *InstallResult) {
	if result == nil {
		return
	}
	status.Installed += result.Success
	status.Skipped += result.Skipped
	status.Failed += result.Failed
	status.Errors = append(status.Errors, result.Errors...)
}

// installFontsInDebugMode processes fonts with plain text output (no TUI) for easier parsing/logging.
//
// This function is used when --debug flag is enabled. It bypasses the TUI progress bar and uses
// plain text output instead, making it easier to parse logs and debug issues.
//
// It processes each font in fontsToInstall, calls installFont for each, and updates the status
// tracking structure. All output is sent to debug logger for detailed diagnostic information.
func installFontsInDebugMode(fontManager platform.FontManager, fontsToInstall []FontToInstall, installScope platform.InstallationScope, force bool, fontDir string, status *InstallationStatus, _ string) error {
	output.GetDebug().State("Starting font installation operation")
	output.GetDebug().State("Total fonts: %d", len(fontsToInstall))

	// Determine scope label for display
	scopeLabel := InstallScopeLabelUser
	if installScope == platform.MachineScope {
		scopeLabel = InstallScopeLabelMachine
	}

	// Process each font
	for i, fontGroup := range fontsToInstall {
		output.GetDebug().State("Installing font %d/%d: %s", i+1, len(fontsToInstall), fontGroup.FontName)
		output.GetDebug().State("Installing font %s in %s (directory: %s)", fontGroup.FontName, scopeLabel, fontDir)

		result, err := installFont(
			fontGroup.Fonts,
			fontGroup.FontID,
			fontManager,
			installScope,
			force,
			fontDir,
		)

		if err != nil {
			output.GetDebug().State("Error installing font %s in %s: %v", fontGroup.FontName, scopeLabel, err)
			if result != nil {
				updateInstallStatus(status, result)
				// Show failed variants if available
				_, _, failedFiles := processInstallResult(result)
				if len(failedFiles) > 0 {
					output.GetDebug().State("Failed variants:")
					for _, file := range failedFiles {
						output.GetDebug().State(" - %s", file)
					}
				}
			}
			continue
		}

		// Update status
		updateInstallStatus(status, result)

		// Show detailed result information in debug mode
		logInstallResultDetails(result, fontGroup.FontName, scopeLabel)
	}

	output.GetDebug().State("Operation complete - Installed: %d, Skipped: %d, Failed: %d",
		status.Installed, status.Skipped, status.Failed)

	// Print status report
	output.PrintStatusReport(output.StatusReport{
		Success:      status.Installed,
		Skipped:      status.Skipped,
		Failed:       status.Failed,
		SuccessLabel: "Installed",
		SkippedLabel: "Skipped",
		FailedLabel:  "Failed",
	}, IsVerbose())

	GetLogger().Info("Installation complete - Installed: %d, Skipped: %d, Failed: %d",
		status.Installed, status.Skipped, status.Failed)
	return nil
}

// downloadFontVariants downloads all variants of a font family
func downloadFontVariants(fontFiles []repo.FontFile, tempDir string) ([]string, error) {
	var allFontPaths []string

	// Download each variant - only log errors and unusual cases
	for _, fontFile := range fontFiles {
		output.GetDebug().State("Calling repo.DownloadAndExtractFont() for variant: %s from %s", fontFile.Variant, fontFile.DownloadURL)
		fontPaths, err := repo.DownloadAndExtractFont(&fontFile, tempDir)
		if err != nil {
			output.GetDebug().State("repo.DownloadAndExtractFont() failed for variant %s: %v", fontFile.Variant, err)
			return nil, err
		}
		allFontPaths = append(allFontPaths, fontPaths...)
		// Only log if multiple files extracted (unusual case worth noting)
		if len(fontPaths) > 1 {
			output.GetDebug().State("Extracted %d file(s) from variant: %s", len(fontPaths), fontFile.Variant)
		}
	}

	return allFontPaths, nil
}

// installDownloadedFonts installs downloaded font files to system
func installDownloadedFonts(fontPaths []string, fontManager platform.FontManager, installScope platform.InstallationScope, fontDir string, force bool) (installed, skipped, failed int, details []string, errors []string, downloadSize int64) {
	var installedFiles []string
	var skippedFiles []string
	var failedFiles []string

	for _, fontPath := range fontPaths {
		fontDisplayName := filepath.Base(fontPath)

		// Get file size before we potentially remove it
		if fileInfo, err := os.Stat(fontPath); err == nil {
			downloadSize += fileInfo.Size()
		}

		// Check if font is already installed (unless force flag is set)
		if !force {
			expectedPath := filepath.Join(fontDir, fontDisplayName)
			if _, err := os.Stat(expectedPath); err == nil {
				output.GetDebug().State("Font already installed, skipping: %s", fontDisplayName)
				skipped++
				os.Remove(fontPath) // Clean up temp file
				skippedFiles = append(skippedFiles, fontDisplayName)
				continue
			}
		}

		// Install the font
		output.GetDebug().State("Installing font file: %s to %s (scope: %s)", fontDisplayName, fontDir, installScope)
		installErr := fontManager.InstallFont(fontPath, installScope, force)

		if installErr != nil {
			// Check if error is related to font cache refresh (non-critical on macOS 14+)
			errStr := installErr.Error()
			isCacheError := strings.Contains(strings.ToLower(errStr), "cache refresh failed (non-critical)")

			if isCacheError {
				// Font was installed successfully, only cache refresh failed
				// This is non-critical - treat as success
				output.GetDebug().Warning("Font cache refresh failed (non-critical on macOS 14+): %s", fontDisplayName)
				output.GetDebug().State("Font installed successfully, cache refresh is optional. Font will be available after app restart.")
				installed++
				installedFiles = append(installedFiles, fontDisplayName)
				os.Remove(fontPath) // Clean up temp file
				continue
			}

			// Actual installation failure
			os.Remove(fontPath) // Clean up temp file
			failed++
			errorMsg := makeUserFriendlyError(fontDisplayName, installErr)
			errors = append(errors, errorMsg)
			failedFiles = append(failedFiles, fontDisplayName)
			output.GetDebug().Error("fontManager.InstallFont() failed for %s: %v", fontDisplayName, installErr)
			continue
		}

		output.GetDebug().State("Successfully installed font: %s", fontDisplayName)

		// Clean up temp file
		os.Remove(fontPath)
		installed++
		installedFiles = append(installedFiles, fontDisplayName)
	}

	// Store categorized details: installed, then skipped, then failed
	details = append(details, installedFiles...)
	details = append(details, skippedFiles...)
	details = append(details, failedFiles...)

	return installed, skipped, failed, details, errors, downloadSize
}

// buildInstallResult builds InstallResult from installation outcomes
func buildInstallResult(status string, message string, installed, skipped, failed int, details []string, errors []string, downloadSize int64) *InstallResult {
	return &InstallResult{
		Success:      installed,
		Skipped:      skipped,
		Failed:       failed,
		Status:       status,
		Message:      message,
		Details:      details,
		Errors:       errors,
		DownloadSize: downloadSize,
	}
}

// installFont handles the core installation logic for a single font.
//
// It checks if the font is already installed (unless force is true), downloads all font variants,
// installs them to the system, and returns an InstallResult with the operation outcome.
// The function handles cleanup of temporary files automatically via defer.
//
// Parameters:
//   - fontFiles: List of font file variants to install
//   - fontID: Font identifier for checking if already installed
//   - fontManager: Platform-specific font manager for installation
//   - installScope: Installation scope (user or machine)
//   - force: If true, skip already-installed check and force reinstallation
//   - fontDir: Target directory for font installation
//
// Returns:
//   - InstallResult: Contains success/skipped/failed counts and details
//   - error: Installation error if the operation fails
func installFont(
	fontFiles []repo.FontFile,
	fontID string,
	fontManager platform.FontManager,
	installScope platform.InstallationScope,
	force bool,
	fontDir string,
) (*InstallResult, error) {
	// Check if font is already installed BEFORE downloading (unless force flag is set)
	// This saves bandwidth by skipping downloads for already-installed fonts
	if !force && fontID != "" && len(fontFiles) > 0 {
		// Get font name from first font file (all files in a family should have the same name)
		fontName := ""
		if len(fontFiles) > 0 && fontFiles[0].Name != "" {
			fontName = fontFiles[0].Name
		}
		alreadyInstalled, checkErr := checkFontsAlreadyInstalled(fontID, fontName, installScope, fontManager)
		if checkErr != nil {
			// Log warning but continue with installation (fail-safe behavior)
			GetLogger().Warn("Failed to check if font is already installed (ID: %s): %v. Proceeding with installation.", fontID, checkErr)
		} else if alreadyInstalled {
			// Font is already installed - skip download and mark all variants as skipped
			output.GetDebug().State("Font %s (ID: %s) is already installed, skipping download", fontName, fontID)
			var details []string
			for _, fontFile := range fontFiles {
				details = append(details, fontFile.Variant)
			}
			return buildInstallResult(InstallStatusSkipped, "Already installed", 0, len(fontFiles), 0, details, nil, 0), nil
		}
	}

	// Download all variants of this font family
	tempDir, err := platform.GetTempFontsDir()
	if err != nil {
		return buildInstallResult(InstallStatusFailed, "Failed to create temp directory", 0, 0, len(fontFiles), nil, nil, 0), fmt.Errorf("failed to create temp directory: %w", err)
	}
	output.GetDebug().State("Temp directory: %s", tempDir)

	// Ensure cleanup happens even if download fails
	defer func() {
		if cleanupErr := platform.CleanupTempFontsDir(); cleanupErr != nil {
			output.GetDebug().State("Failed to cleanup temp directory: %v", cleanupErr)
			// Don't fail the installation if cleanup fails, just log it
		}
	}()

	allFontPaths, downloadErr := downloadFontVariants(fontFiles, tempDir)
	if downloadErr != nil {
		return buildInstallResult(InstallStatusFailed, "Download failed", 0, 0, len(fontFiles), nil, nil, 0), downloadErr
	}

	// Install downloaded fonts
	installed, skipped, failed, details, errors, downloadSize := installDownloadedFonts(
		allFontPaths, fontManager, installScope, fontDir, force)

	// Determine final status
	status := InstallStatusCompleted
	message := "Installed"
	if failed > 0 && installed == 0 {
		status = InstallStatusFailed
		message = "Installation failed"
	} else if skipped > 0 && installed == 0 && failed == 0 {
		status = InstallStatusSkipped
		message = "Already installed"
	}

	return buildInstallResult(status, message, installed, skipped, failed, details, errors, downloadSize), nil
}

// makeUserFriendlyError converts technical error messages to user-friendly explanations
func makeUserFriendlyError(fontName string, err error) string {
	errStr := strings.ToLower(err.Error())

	// Check for common error patterns and provide user-friendly messages
	if strings.Contains(errStr, "cannot access the file because it is being used") {
		return fmt.Sprintf("%s could not be reinstalled as it's in use by another application. Try closing the app or process using this font and try again.", fontName)
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

// checkFontsAlreadyInstalled checks if a font is already installed in the specified scope.
// It uses the same matching logic as the list command (collectFonts and MatchAllInstalledFonts)
// to match by Font ID (most accurate) and family name (fallback).
// Returns true if the font is already installed, false otherwise.
// Note: This function scans the font directory each time it's called. For multiple fonts,
// checkFontsAlreadyInstalled checks if a font is already installed in the specified scope.
//
// It collects installed fonts from the target scope, matches them against the repository to get
// Font IDs, and checks if the provided fontID matches any installed font.
//
// This function is used to avoid unnecessary downloads when a font is already installed.
// Note: For performance with many fonts, consider pre-collecting fonts and using a cached approach.
//
// Parameters:
//   - fontID: Font identifier to check
//   - fontName: Font name (used for fallback matching if Font ID matching fails)
//   - scope: Installation scope to check (user or machine)
//   - fontManager: Platform-specific font manager
//
// Returns:
//   - bool: true if font is already installed, false otherwise
//   - error: Error if font collection or matching fails
func checkFontsAlreadyInstalled(fontID string, fontName string, scope platform.InstallationScope, fontManager platform.FontManager) (bool, error) {
	// Early return if fontID is empty (can't check without ID)
	if fontID == "" {
		return false, nil
	}

	// Collect installed fonts from the target scope
	// Suppress verbose output since this is an internal check, not a primary operation
	scopes := []platform.InstallationScope{scope}
	fonts, err := collectFonts(scopes, fontManager, "", true)
	if err != nil {
		return false, fmt.Errorf("failed to collect installed fonts: %w", err)
	}

	// Early return if no fonts found
	if len(fonts) == 0 {
		return false, nil
	}

	// Group fonts by family name
	families := groupByFamily(fonts)
	if len(families) == 0 {
		return false, nil
	}

	// Get all family names
	var familyNames []string
	for familyName := range families {
		familyNames = append(familyNames, familyName)
	}

	// Match installed fonts to repository entries
	matches, err := repo.MatchAllInstalledFonts(familyNames, shared.IsCriticalSystemFont)
	if err != nil {
		// If matching fails, we can't determine if font is installed, so return false
		// This allows the installation to proceed (fail-safe)
		// Note: Error is not returned to caller, but this is intentional for fail-safe behavior
		return false, nil
	}

	// Normalize font ID for comparison (case-insensitive) - do this once
	fontIDLower := strings.ToLower(fontID)

	// Check if any installed font matches the target Font ID (most accurate match)
	for _, match := range matches {
		if match != nil {
			// Match by Font ID (most accurate)
			matchIDLower := strings.ToLower(match.FontID)
			if matchIDLower == fontIDLower {
				return true, nil
			}
		}
	}

	// Fallback: check by family name if Font ID didn't match
	// This handles cases where the font might be installed but not matched to repository
	// Note: This fallback may have false positives (e.g., "Roboto" might match "Roboto Mono")
	// but it's acceptable as a fallback for fonts not in the repository
	if fontName != "" {
		fontNameLower := strings.ToLower(fontName)
		fontNameNorm := strings.ReplaceAll(fontNameLower, " ", "")
		fontNameNorm = strings.ReplaceAll(fontNameNorm, "-", "")
		fontNameNorm = strings.ReplaceAll(fontNameNorm, "_", "")

		for familyName := range families {
			familyLower := strings.ToLower(familyName)
			familyNorm := strings.ReplaceAll(familyLower, " ", "")
			familyNorm = strings.ReplaceAll(familyNorm, "-", "")
			familyNorm = strings.ReplaceAll(familyNorm, "_", "")

			// Check for exact match (normalized)
			if familyLower == fontNameLower || familyNorm == fontNameNorm {
				return true, nil
			}
		}
	}

	return false, nil
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringP("scope", "s", "", "Installation scope (user or machine)")
	addCmd.Flags().BoolP("force", "f", false, "Force installation even if font is already installed")
}
