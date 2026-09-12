package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fontget/internal/cmdutils"
	"fontget/internal/components"
	"fontget/internal/installations"
	"fontget/internal/output"
	"fontget/internal/platform"
	"fontget/internal/repo"
	"fontget/internal/shared"
	"fontget/internal/ui"
	"fontget/internal/version"

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
	// Safety check
	if len(notFoundFonts) == 0 {
		return
	}

	// Show all not-found fonts first
	if len(notFoundFonts) == 1 {
		// Single font - use the original function for consistency
		allFonts := repo.GetAllFontsCached()
		var similar []string
		if len(allFonts) > 0 {
			similar = shared.FindSimilarFonts(notFoundFonts[0], allFonts, false) // false = repository fonts
		}
		// Ensure we always show output, even if similar fonts list is empty
		showFontNotFoundWithSuggestions(notFoundFonts[0], similar)
		return
	}

	// Multiple fonts - show grouped format
	fmt.Println()
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
		fmt.Printf("%s\n", ui.Text.Render("Did you mean one of these fonts?"))
		fmt.Println()

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
			matches := shared.FindMatchesInRepository(repository, suggestion)
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
			// Build table rows
			var tableRows [][]string
			for _, match := range uniqueMatches {
				categories := shared.PlaceholderNA
				if len(match.FontInfo.Categories) > 0 {
					categories = match.FontInfo.Categories[0]
				}

				license := match.FontInfo.License
				if license == "" {
					license = shared.PlaceholderNA
				}

				row := []string{
					match.Name,
					match.ID,
					categories,
					license,
					match.Source,
				}
				tableRows = append(tableRows, row)
			}

			// Render table with priority configuration
			tableConfig := components.TableConfig{
				Columns: []components.ColumnConfig{
					{Header: "Font Name", Truncatable: true, Hideable: false, MinWidth: 18, Priority: 2, PercentWidth: 26.0},
					{Header: "Font ID", Truncatable: false, Hideable: false, Priority: 1, PercentWidth: 34.0}, // Highest priority, don't trim
					{Header: "Categories", Truncatable: true, MaxWidth: 14, Hideable: true, Priority: 3, PercentWidth: 15.0},
					{Header: "License", Truncatable: true, MaxWidth: 8, Hideable: true, Priority: 4, PercentWidth: 10.0},
					{Header: "Source", Truncatable: true, MaxWidth: 14, Hideable: true, Priority: 5, PercentWidth: 15.0}, // Lowest priority
				},
				Rows:     tableRows,
				Width:    0,   // Auto-detect terminal width
				MaxWidth: 120, // Maximum width
				Mode:     components.TableModeStatic,
				Padding:  1, // Default padding
			}

			fmt.Println(components.RenderStaticTable(tableConfig))
			fmt.Println()
		} else {
			// No matches found, show general guidance
			fmt.Printf("%s\n", ui.Text.Render("Try using the search command to find available fonts."))
			fmt.Println()
		}
	} else {
		// No suggestions at all, show general guidance
		fmt.Println()
		fmt.Printf("%s\n", ui.Text.Render("Try using the search command to find available fonts."))
		fmt.Println()
	}
}

// showFontNotFoundWithSuggestions displays font not found error with suggestions in table format
func showFontNotFoundWithSuggestions(fontName string, similar []string) {
	fmt.Printf("%s\n", ui.ErrorText.Render(fmt.Sprintf("\nFont '%s' not found.", fontName)))

	// If no similar fonts found, show general guidance
	if len(similar) == 0 {
		fmt.Printf("%s\n", ui.Text.Render("Try using the search command to find available fonts."))
		fmt.Println()
		return
	}

	// Load repository for detailed font information
	repository, err := repo.GetRepository()
	if err != nil {
		// If we can't load repository, show simple list (like remove command)
		fmt.Printf("%s\n", ui.Text.Render("Did you mean one of these fonts?"))
		fmt.Println()
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
		matches := shared.FindMatchesInRepository(repository, suggestion)
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
		fmt.Printf("%s\n", ui.Text.Render("Did you mean one of these fonts?"))
		fmt.Println()

		// Build table rows
		var tableRows [][]string
		for _, match := range uniqueMatches {
			// Get categories (first one if available)
			categories := shared.PlaceholderNA
			if len(match.FontInfo.Categories) > 0 {
				categories = match.FontInfo.Categories[0]
			}

			// Get license
			license := match.FontInfo.License
			if license == "" {
				license = shared.PlaceholderNA
			}

			row := []string{
				match.Name,
				match.ID,
				categories,
				license,
				match.Source,
			}
			tableRows = append(tableRows, row)
		}

		// Render table with priority configuration
		tableConfig := components.TableConfig{
			Columns: []components.ColumnConfig{
				{Header: "Font Name", Truncatable: true, Hideable: false, MinWidth: 18, Priority: 2, PercentWidth: 26.0},
				{Header: "Font ID", Truncatable: false, Hideable: false, Priority: 1, PercentWidth: 34.0}, // Highest priority, don't trim
				{Header: "Categories", Truncatable: true, MaxWidth: 14, Hideable: true, Priority: 3, PercentWidth: 15.0},
				{Header: "License", Truncatable: true, MaxWidth: 8, Hideable: true, Priority: 4, PercentWidth: 10.0},
				{Header: "Source", Truncatable: true, MaxWidth: 14, Hideable: true, Priority: 5, PercentWidth: 15.0}, // Lowest priority
			},
			Rows:     tableRows,
			Width:    0,   // Auto-detect terminal width
			MaxWidth: 120, // Maximum width
			Mode:     components.TableModeStatic,
			Padding:  1, // Default padding
		}

		fmt.Println(components.RenderStaticTable(tableConfig))
		fmt.Println()
	} else {
		// Fallback: if similar font names were found but couldn't be resolved to matches
		// This suggests the font exists in our cache but is no longer available from sources
		fmt.Printf("%s\n", ui.ErrorText.Render(fmt.Sprintf("Font '%s' was not able to be downloaded and installed.", fontName)))
		fmt.Printf("%s\n", ui.Text.Render("It may have been removed from the font source."))
		fmt.Printf("\n%s\n", ui.Text.Render("Please refresh FontGet sources using:"))
		fmt.Printf("  %s\n", ui.Text.Render("fontget sources update"))
		fmt.Printf("\n%s\n", ui.Text.Render("Try using the search command to find other available fonts:"))
		fmt.Printf("  %s\n", ui.Text.Render("fontget search \"font name\""))
		fmt.Println()
	}
}

// resolveAndValidateFonts resolves font queries and validates them, returning fonts to install and not found fonts.
func resolveAndValidateFonts(fontNames []string) (fontsToInstall []FontToInstall, notFoundFonts []string) {
	// Initialize as empty slice (not nil) to distinguish between "no fonts" and "multiple matches" cases
	fontsToInstall = []FontToInstall{}
	notFoundFonts = []string{}
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
		fmt.Printf("%s\n", ui.Text.Render("Try using 'fontget search' to find available fonts."))
		// Section ends with blank line per spacing framework
		fmt.Println()
	}
}

// showMultipleMatchesAndExit displays search results and instructs user to use specific font ID
func showMultipleMatchesAndExit(fontName string, matches []repo.FontMatch) {

	fmt.Printf("\n%s\n", ui.InfoText.Render(fmt.Sprintf("Multiple fonts found matching '%s'.", ui.QueryText.Render(fontName))))
	fmt.Printf("%s\n", ui.Text.Render("Please specify the exact font ID to install from a specific source."))
	fmt.Println()

	// Build table rows
	var tableRows [][]string
	for _, match := range matches {
		// Get categories (first one if available)
		categories := shared.PlaceholderNA
		if len(match.FontInfo.Categories) > 0 {
			categories = match.FontInfo.Categories[0]
		}

		// Get license
		license := match.FontInfo.License
		if license == "" {
			license = shared.PlaceholderNA
		}

		row := []string{
			match.Name,
			match.ID,
			categories,
			license,
			match.Source,
		}
		tableRows = append(tableRows, row)
	}

	// Render table with priority configuration
	tableConfig := components.TableConfig{
		Columns: []components.ColumnConfig{
			{Header: "Font Name", Truncatable: true, Hideable: false, MinWidth: 18, Priority: 2, PercentWidth: 26.0},
			{Header: "Font ID", Truncatable: false, Hideable: false, Priority: 1, PercentWidth: 34.0}, // Highest priority, don't trim
			{Header: "Categories", Truncatable: true, MaxWidth: 14, Hideable: true, Priority: 3, PercentWidth: 15.0},
			{Header: "License", Truncatable: true, MaxWidth: 8, Hideable: true, Priority: 4, PercentWidth: 10.0},
			{Header: "Source", Truncatable: true, MaxWidth: 14, Hideable: true, Priority: 5, PercentWidth: 15.0}, // Lowest priority
		},
		Rows:     tableRows,
		Width:    0,   // Auto-detect terminal width
		MaxWidth: 120, // Maximum width
		Mode:     components.TableModeStatic,
		Padding:  1, // Default padding
	}

	fmt.Println(components.RenderStaticTable(tableConfig))
	fmt.Println()
}

var addCmd = &cobra.Command{
	Use:           "add <font-id> [<font-id2> <font-id3> ...]",
	Aliases:       []string{"install"},
	Short:         "Install fonts from configured sources",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `Install one or multiple fonts in a single command.

Fonts can be specified by name (e.g., "Roboto") or Font ID (e.g., "google.roboto").
Font names with spaces must be wrapped in quotes (e.g., "Open Sans").

Use --scope to set installation location:
  - user (default): Install for current user only
  - machine: Install system-wide (requires administrator privileges)`,
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
			fmt.Printf("%s\n", ui.Text.Render("Use 'fontget add --help' for more information."))
			fmt.Println()
			return shared.AlreadyPrinted(fmt.Errorf("a font ID is required"))
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		GetLogger().Info("Starting font installation operation")

		// Always start with a blank line for consistent spacing from command prompt
		// Note: Only print blank line if we're not in a case where we'll show error messages
		// This prevents extra blank lines when showing "not found" errors

		// Ensure manifest system is initialized (fixes missing sources.json bug)
		if err := cmdutils.EnsureManifestInitialized(func() cmdutils.Logger { return GetLogger() }); err != nil {
			return err
		}

		// Double check args to prevent panic
		if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
			return shared.AlreadyPrinted(fmt.Errorf("a font ID is required"))
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
				return shared.AlreadyPrinted(err)
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
			return shared.AlreadyPrinted(fmt.Errorf("multiple fonts match; specify a font ID"))
		}

		// Check if flags are set
		verbose, _ := cmd.Flags().GetBool("verbose")
		debug, _ := cmd.Flags().GetBool("debug")

		// If no fonts to install, show not found message with suggestions and exit
		// Do this BEFORE verbose output to avoid extra blank lines
		if len(fontsToInstall) == 0 {
			if len(notFoundFonts) > 0 {
				if IsDebug() {
					output.GetDebug().Error("No fonts found to install. The following font(s) were not found in any source:")
					for _, fontName := range notFoundFonts {
						output.GetDebug().Error(" - %s", fontName)
					}
				} else {
					defer func() {
						if r := recover(); r != nil {
							fmt.Fprintf(os.Stdout, "Font(s) not found: %v\n", notFoundFonts)
							fmt.Fprintf(os.Stdout, "Try using the search command to find available fonts.\n")
						}
					}()
					os.Stdout.Sync()
					showGroupedFontNotFoundWithSuggestions(notFoundFonts)
					os.Stdout.Sync()
				}
				return shared.AlreadyPrinted(&shared.FontNotFoundError{FontName: strings.Join(notFoundFonts, ", ")})
			}
			fmt.Printf("%s\n", ui.ErrorText.Render("No fonts specified or found."))
			return shared.AlreadyPrinted(fmt.Errorf("no fonts specified or found"))
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
		if output.IsVerboseOutputEnabled() {
			fmt.Println()
		}

		// Initialize status tracking
		status := &InstallationStatus{
			Details: make([]string, 0),
		}

		// Track detailed operations for each font (for verbose mode)
		var operationDetails []FontOperationDetails

		// No need for separate header - the progress bar will show the title

		staging, err := platform.NewOperationStaging()
		if err != nil {
			return fmt.Errorf("failed to create operation staging: %w", err)
		}
		defer func() {
			if cleanupErr := staging.Cleanup(); cleanupErr != nil {
				output.GetDebug().State("Failed to cleanup operation staging: %v", cleanupErr)
			}
		}()

		opCtx := cmd.Context()
		if opCtx == nil {
			opCtx = context.Background()
		}

		packagesFailed := 0
		cancelled := false

		operationItems := setupInstallationProgressBar(fontsToInstall)

		title := OpInstallingFonts
		if installScope == platform.MachineScope {
			title = OpInstallingFontsAllUsers
		}

		if !output.IsVerboseOutputEnabled() {
			fmt.Println()
		}

		suppressVerboseDownloads := components.UseInteractiveRenderer() && !IsDebug()

		progressErr := components.RunProgressBar(
			title,
			operationItems,
			verbose,
			debug,
			func(send func(msg tea.Msg), cancelChan <-chan struct{}) error {
				ctx, cancel := context.WithCancel(opCtx)
				defer cancel()
				go func() {
					select {
					case <-cancelChan:
						cancel()
					case <-ctx.Done():
					}
				}()

				for itemIndex, fontGroup := range fontsToInstall {
					if err := ctx.Err(); err != nil {
						cancelled = true
						return shared.ErrOperationCancelled
					}

					send(components.ItemUpdateMsg{
						Index:   itemIndex,
						Status:  "in_progress",
						Message: "Downloading from " + fontGroup.SourceName,
					})

					percent := float64(itemIndex) / float64(len(fontsToInstall)) * 100
					send(components.ProgressUpdateMsg{Percent: percent})

					lastStep := ""
					lastPctBucket := -1
					onProgress := func(step string, stepPct float64) {
						bucket := int(shared.Clamp01(stepPct) * 20.0)
						if step == lastStep && bucket == lastPctBucket {
							return
						}
						lastStep = step
						lastPctBucket = bucket

						msg := step + "..."
						if step == installStepDownload {
							msg = "Downloading from " + fontGroup.SourceName
						}

						send(components.ItemUpdateMsg{
							Index:   itemIndex,
							Status:  "in_progress",
							Message: msg,
						})
						send(components.ProgressUpdateMsg{
							Percent: OverallInstallPercent(itemIndex, len(fontsToInstall), step, stepPct),
						})
					}
					result, err := installFont(
						ctx,
						fontGroup.Fonts,
						fontGroup.FontID,
						fontManager,
						installScope,
						force,
						fontDir,
						staging,
						suppressVerboseDownloads,
						onProgress,
					)

					if err != nil {
						packagesFailed++
						if result != nil {
							if result.Status != InstallStatusFailed {
								result.Status = InstallStatusFailed
							}
							if result.Failed == 0 && result.Success == 0 {
								status.Failed++
							} else {
								status.Failed += result.Failed
							}
							status.Errors = append(status.Errors, result.Errors...)
						} else {
							status.Failed++
						}
						GetLogger().Error("Failed to process font %s: %v", fontGroup.FontName, err)
						errorMsg := err.Error()
						send(components.ItemUpdateMsg{
							Index:        itemIndex,
							Status:       InstallStatusFailed,
							Message:      "Operation failed",
							ErrorMessage: errorMsg,
						})
						if errors.Is(err, shared.ErrOperationCancelled) || errors.Is(err, context.Canceled) {
							cancelled = true
							return shared.ErrOperationCancelled
						}
						continue
					}

					if result.Status == InstallStatusFailed {
						packagesFailed++
					}

					status.Installed += result.Success
					status.Skipped += result.Skipped
					status.Failed += result.Failed
					status.Errors = append(status.Errors, result.Errors...)

					installedFiles, skippedFiles, failedFiles := processInstallResult(result)
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

					finalStatus := result.Status
					var variantsWithStatus []string
					if verbose {
						variantsWithStatus = variantLinesForVerboseProgress(fontGroup.Fonts)
					}
					var errorMsg string
					if finalStatus == InstallStatusFailed && len(result.Errors) > 0 {
						errorMsg = result.Errors[0]
					}
					scopeLabel := InstallScopeLabelUser
					if installScope == platform.MachineScope {
						scopeLabel = InstallScopeLabelMachine
					}

					send(components.ItemUpdateMsg{
						Index:        itemIndex,
						Status:       finalStatus,
						Message:      "Installed",
						ErrorMessage: errorMsg,
						Variants:     variantsWithStatus,
						Scope:        scopeLabel,
					})

					send(components.ProgressUpdateMsg{Percent: OverallInstallPercent(itemIndex, len(fontsToInstall), installStepCompleted, 1)})
				}

				return nil
			},
		)

		if progressErr != nil {
			if errors.Is(progressErr, shared.ErrOperationCancelled) || cancelled {
				fmt.Printf("%s\n", ui.WarningText.Render("Installation cancelled."))
				fmt.Println()
				return shared.AlreadyPrinted(shared.ErrOperationCancelled)
			}
			GetLogger().Error("Failed to install fonts: %v", progressErr)
			return progressErr
		}

		handleNotFoundFonts(notFoundFonts, IsDebug())

		showSummary := output.IsVerboseOutputEnabled() || packagesFailed > 0 || status.Failed > 0 || len(notFoundFonts) > 0 || !components.UseInteractiveRenderer()
		output.PrintStatusReport(output.StatusReport{
			Success:      status.Installed,
			Skipped:      status.Skipped,
			Failed:       status.Failed,
			SuccessLabel: "Installed",
			SkippedLabel: "Skipped",
			FailedLabel:  "Failed",
		}, showSummary)

		GetLogger().Info("Installation complete - Installed: %d, Skipped: %d, Failed: %d",
			status.Installed, status.Skipped, status.Failed)

		if len(notFoundFonts) > 0 {
			packagesFailed++
		}
		if packagesFailed > 0 || status.Failed > 0 {
			return shared.AlreadyPrinted(&shared.FontInstallationError{
				FailedCount: packagesFailed,
				TotalCount:  len(fontsToInstall) + len(notFoundFonts),
			})
		}
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
			context.Background(),
			fontGroup.Fonts,
			fontGroup.FontID,
			fontManager,
			installScope,
			force,
			fontDir,
			nil,
			false,
			nil,
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
	}, output.IsVerboseOutputEnabled())

	GetLogger().Info("Installation complete - Installed: %d, Skipped: %d, Failed: %d",
		status.Installed, status.Skipped, status.Failed)
	return nil
}

// variantLinesForVerboseProgress returns one human-readable label per manifest variant for the progress TUI
// (avoids listing both style names and on-disk filenames).
func variantLinesForVerboseProgress(fonts []repo.FontFile) []string {
	lines := make([]string, 0, len(fonts))
	for i := range fonts {
		lines = append(lines, formatRepoFontVariantLine(&fonts[i]))
	}
	return lines
}

func formatRepoFontVariantLine(f *repo.FontFile) string {
	v := strings.TrimSpace(f.Variant)
	name := strings.TrimSpace(f.Name)
	if v != "" {
		if name != "" {
			return strings.TrimSpace(fmt.Sprintf("%s %s", name, humanizeFontStyleLabel(v)))
		}
		return humanizeFontStyleLabel(v)
	}
	if f.Path != "" {
		return filepath.Base(f.Path)
	}
	return "font"
}

func humanizeFontStyleLabel(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "_", " "))
	s = strings.ReplaceAll(s, "-", " ")
	fields := strings.Fields(s)
	for i, w := range fields {
		if w == "" {
			continue
		}
		allDigit := true
		for _, r := range w {
			if r < '0' || r > '9' {
				allDigit = false
				break
			}
		}
		if allDigit {
			continue
		}
		fields[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
	}
	return strings.Join(fields, " ")
}

func archiveSourcePrefixFromFontID(fontID string) string {
	fontID = strings.TrimSpace(fontID)
	i := strings.IndexByte(fontID, '.')
	if i <= 0 || i >= len(fontID)-1 {
		return ""
	}
	return strings.ToLower(fontID[:i])
}

// cloneDownloadOptsForProgress returns a shallow copy of downloadOpts (or zero)
// with ArchiveSourcePrefix set from archiveSourcePrefix so progress callbacks
// can be attached without dropping fields like OnResponseHeaders.
func cloneDownloadOptsForProgress(downloadOpts *repo.DownloadFontOptions, archiveSourcePrefix, archiveFontID string) *repo.DownloadFontOptions {
	var base repo.DownloadFontOptions
	if downloadOpts != nil {
		base = *downloadOpts
	}
	base.ArchiveSourcePrefix = archiveSourcePrefix
	base.ArchiveFontID = archiveFontID
	return &base
}

// downloadFontVariants downloads all variants of a font family
func downloadFontVariants(ctx context.Context, fontFiles []repo.FontFile, staging *platform.OperationStaging, fontID string, archiveSourcePrefix string, downloadOpts *repo.DownloadFontOptions, onProgress StepProgressFunc) ([]string, error) {
	start := time.Now()
	var allFontPaths []string

	total := len(fontFiles)
	for i, fontFile := range fontFiles {
		if ctx != nil {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		if onProgress != nil && total > 0 {
			onProgress(installStepDownload, float64(i)/float64(total))
		}
		opts := downloadOpts
		if opts == nil {
			opts = &repo.DownloadFontOptions{}
		} else {
			cpy := *opts
			opts = &cpy
		}
		opts.Context = ctx
		opts.ArchiveSourcePrefix = archiveSourcePrefix
		opts.ArchiveFontID = fontID
		if onProgress != nil {
			opts.OnBytesDownloaded = func(doneBytes int64, totalBytes int64) {
				if totalBytes > 0 {
					onProgress(installStepDownload, float64(doneBytes)/float64(totalBytes))
				}
			}
			opts.OnExtractProgress = func(done int, total int) {
				if total > 0 {
					onProgress(installStepExtract, float64(done)/float64(total))
					return
				}
				onProgress(installStepExtract, float64(done)/float64(done+12))
			}
		}

		variantDir, err := staging.VariantDir(fontID, fontFile.Variant)
		if err != nil {
			return nil, err
		}

		output.GetDebug().State("Calling repo.DownloadAndExtractFont() for variant: %s from %s", fontFile.Variant, fontFile.DownloadURL)
		fontPaths, err := repo.DownloadAndExtractFont(&fontFile, variantDir, opts)
		if err != nil {
			output.GetDebug().State("repo.DownloadAndExtractFont() failed for variant %s: %v", fontFile.Variant, err)
			return nil, err
		}
		allFontPaths = append(allFontPaths, fontPaths...)
		if len(fontPaths) > 1 {
			output.GetDebug().State("Extracted %d file(s) from variant: %s", len(fontPaths), fontFile.Variant)
		}
	}

	if onProgress != nil {
		onProgress(installStepDownload, 1)
		onProgress(installStepExtract, 1)
	}
	output.GetDebug().State("downloadFontVariants: files=%d extracted=%d total=%dms", len(fontFiles), len(allFontPaths), time.Since(start).Milliseconds())
	return allFontPaths, nil
}

// installDownloadedFonts installs downloaded font files to system
func installDownloadedFonts(ctx context.Context, fontPaths []string, fontManager platform.FontManager, installScope platform.InstallationScope, fontDir string, force bool, onProgress StepProgressFunc) (installed, skipped, failed int, details []string, errs []string, downloadSize int64, mutations []platform.FileMutation, err error) {
	start := time.Now()
	var installedFiles []string
	var skippedFiles []string
	var failedFiles []string

	batchOpts := &platform.InstallFontOptions{SkipPostInstallCacheRefresh: true}

	total := len(fontPaths)
	for i, fontPath := range fontPaths {
		if ctx != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				err = ctxErr
				break
			}
		}
		if onProgress != nil && total > 0 {
			onProgress(installStepInstall, float64(i)/float64(total))
		}
		fontDisplayName := filepath.Base(fontPath)

		if fileInfo, statErr := os.Stat(fontPath); statErr == nil {
			downloadSize += fileInfo.Size()
		}

		if !force {
			expectedPath := filepath.Join(fontDir, fontDisplayName)
			if _, statErr := os.Stat(expectedPath); statErr == nil {
				output.GetDebug().State("Font already installed, skipping: %s", fontDisplayName)
				skipped++
				_ = os.Remove(fontPath)
				skippedFiles = append(skippedFiles, fontDisplayName)
				continue
			}
		}

		var mut platform.FileMutation
		batchOpts.Mutation = &mut
		if testInstallFailPoint == "register" {
			batchOpts.FailPoint = platform.InstallFailRegister
		} else {
			batchOpts.FailPoint = ""
		}

		output.GetDebug().State("Installing font file: %s to %s (scope: %s)", fontDisplayName, fontDir, installScope)
		installErr := fontManager.InstallFont(fontPath, installScope, force, batchOpts)

		if mut.DestPath != "" {
			mutations = append(mutations, mut)
		}

		if installErr != nil {
			_ = os.Remove(fontPath)
			failed++
			errorMsg := makeUserFriendlyError(fontDisplayName, installErr)
			errs = append(errs, errorMsg)
			failedFiles = append(failedFiles, fontDisplayName)
			output.GetDebug().Error("fontManager.InstallFont() failed for %s: %v", fontDisplayName, installErr)
			err = installErr
			break
		}

		if testInstallFailPoint == "after-first" && len(mutations) == 1 {
			failed++
			err = fmt.Errorf("injected failure after first file mutation")
			failedFiles = append(failedFiles, fontDisplayName)
			break
		}
		if testInstallFailPoint == "after-later" && len(mutations) >= 2 {
			failed++
			err = fmt.Errorf("injected failure after later file mutation")
			failedFiles = append(failedFiles, fontDisplayName)
			break
		}

		installedPath := filepath.Join(fontDir, fontDisplayName)
		if _, statErr := os.Stat(installedPath); statErr == nil {
			if _, metaErr := platform.ExtractFontMetadata(installedPath); metaErr != nil {
				_ = os.Remove(fontPath)
				failed++
				errorMsg := makeUserFriendlyError(fontDisplayName, fmt.Errorf("installed file is not a valid font: %w", metaErr))
				errs = append(errs, errorMsg)
				failedFiles = append(failedFiles, fontDisplayName)
				output.GetDebug().Warning("Installed file failed validation: %s (%v)", fontDisplayName, metaErr)
				err = metaErr
				break
			}
		}

		output.GetDebug().State("Successfully installed font: %s", fontDisplayName)
		_ = os.Remove(fontPath)
		installed++
		installedFiles = append(installedFiles, fontDisplayName)
	}

	if onProgress != nil {
		onProgress(installStepInstall, 1)
	}

	if err == nil && installed > 0 {
		if onProgress != nil {
			onProgress(installStepFinalize, 0)
		}
		if flushErr := fontManager.FlushFontCache(installScope); flushErr != nil {
			errStr := flushErr.Error()
			isDarwinNonCritical := strings.Contains(strings.ToLower(errStr), "failed to refresh font cache (non-critical")
			if isDarwinNonCritical {
				output.GetDebug().Warning("Font cache refresh failed (non-critical on macOS 14+): %v", flushErr)
			} else {
				output.GetDebug().Error("Post-install font cache flush failed: %v", flushErr)
			}
		}
		if onProgress != nil {
			onProgress(installStepFinalize, 1)
		}
	}

	details = append(details, installedFiles...)
	details = append(details, skippedFiles...)
	details = append(details, failedFiles...)

	output.GetDebug().State("installDownloadedFonts: installed=%d skipped=%d failed=%d total=%dms", installed, skipped, failed, time.Since(start).Milliseconds())
	return installed, skipped, failed, details, errs, downloadSize, mutations, err
}

// testInstallFailPoint injects package-level failures in tests: after-first, after-later, register, provenance.
var testInstallFailPoint string

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
	ctx context.Context,
	fontFiles []repo.FontFile,
	fontID string,
	fontManager platform.FontManager,
	installScope platform.InstallationScope,
	force bool,
	fontDir string,
	staging *platform.OperationStaging,
	suppressVerboseDownloads bool,
	onProgress StepProgressFunc,
) (*InstallResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if onProgress != nil {
		onProgress(installStepPrecheck, 0)
	}
	if !force && fontID != "" && len(fontFiles) > 0 {
		fontName := ""
		if fontFiles[0].Name != "" {
			fontName = fontFiles[0].Name
		}
		alreadyInstalled, checkErr := checkFontsAlreadyInstalled(fontID, fontName, installScope, fontManager)
		if checkErr != nil {
			GetLogger().Warn("Failed to check if font is already installed (ID: %s): %v. Proceeding with installation.", fontID, checkErr)
		} else if alreadyInstalled {
			if onProgress != nil {
				onProgress(installStepPrecheck, 1)
				onProgress(installStepCompleted, 1)
			}
			output.GetDebug().State("Font %s (ID: %s) is already installed, skipping download", fontName, fontID)
			var details []string
			for _, fontFile := range fontFiles {
				details = append(details, fontFile.Variant)
			}
			return buildInstallResult(InstallStatusSkipped, "Already installed", 0, len(fontFiles), 0, details, nil, 0), nil
		}
	}
	if onProgress != nil {
		onProgress(installStepPrecheck, 1)
	}

	if staging == nil {
		created, stErr := platform.NewOperationStaging()
		if stErr != nil {
			return buildInstallResult(InstallStatusFailed, "Failed to create temp directory", 0, 0, len(fontFiles), nil, nil, 0), stErr
		}
		staging = created
		defer func() { _ = staging.Cleanup() }()
	}

	downloadOpts := (*repo.DownloadFontOptions)(nil)
	if suppressVerboseDownloads {
		downloadOpts = &repo.DownloadFontOptions{SuppressVerboseProgressLine: true, Context: ctx}
	} else {
		downloadOpts = &repo.DownloadFontOptions{Context: ctx}
	}
	archivePrefix := archiveSourcePrefixFromFontID(fontID)
	if onProgress != nil {
		onProgress(installStepDownload, 0)
	}
	allFontPaths, downloadErr := downloadFontVariants(ctx, fontFiles, staging, fontID, archivePrefix, downloadOpts, onProgress)
	if downloadErr != nil {
		return buildInstallResult(InstallStatusFailed, "Download failed", 0, 0, len(fontFiles), nil, nil, 0), downloadErr
	}

	unlockDest, lockErr := installations.LockDestination(ctx, fontDir)
	if lockErr != nil {
		return buildInstallResult(InstallStatusFailed, "Failed to lock destination", 0, 0, len(fontFiles), nil, nil, 0), lockErr
	}
	defer unlockDest()

	installed, skipped, failed, details, instErrs, downloadSize, mutations, installErr := installDownloadedFonts(
		ctx, allFontPaths, fontManager, installScope, fontDir, force, onProgress)

	if installErr != nil || failed > 0 {
		rbErr := rollbackPackageMutations(ctx, fontID, string(installScope), mutations, installErr)
		res := buildInstallResult(InstallStatusFailed, "Installation failed", 0, skipped, failed, details, instErrs, downloadSize)
		if rbErr != nil {
			return res, rbErr
		}
		if installErr != nil {
			return res, installErr
		}
		return res, fmt.Errorf("package install incomplete")
	}

	status := InstallStatusCompleted
	message := "Installed"
	if skipped > 0 && installed == 0 && failed == 0 {
		status = InstallStatusSkipped
		message = "Already installed"
	}

	res := buildInstallResult(status, message, installed, skipped, failed, details, instErrs, downloadSize)
	if status == InstallStatusCompleted && installed > 0 {
		if testInstallFailPoint == "provenance" {
			rbErr := rollbackPackageMutations(ctx, fontID, string(installScope), mutations, fmt.Errorf("injected provenance failure"))
			res.Status = InstallStatusFailed
			res.Success = 0
			if rbErr != nil {
				return res, rbErr
			}
			return res, fmt.Errorf("injected provenance failure")
		}
		if recErr := tryRecordInstallationRegistry(fontID, fontFiles, installScope, fontDir, res); recErr != nil {
			rbErr := rollbackPackageMutations(ctx, fontID, string(installScope), mutations, recErr)
			res.Status = InstallStatusFailed
			res.Success = 0
			if rbErr != nil {
				return res, rbErr
			}
			return res, recErr
		}
		for _, mut := range mutations {
			_ = platform.CommitMutation(mut)
		}
	}
	return res, nil
}

func rollbackPackageMutations(ctx context.Context, fontID, scope string, mutations []platform.FileMutation, cause error) error {
	recCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 60*time.Second)
	defer cancel()
	_ = recCtx

	var outstanding []string
	var completed []string
	var recErrs []string
	if cause != nil {
		recErrs = append(recErrs, cause.Error())
	}
	var backupPaths []string
	var affected []string
	for i := len(mutations) - 1; i >= 0; i-- {
		mut := mutations[i]
		affected = append(affected, mut.DestPath)
		if mut.BackupPath != "" {
			backupPaths = append(backupPaths, mut.BackupPath)
		}
		if err := platform.RollbackMutation(mut); err != nil {
			outstanding = append(outstanding, "restore "+mut.DestPath)
			recErrs = append(recErrs, err.Error())
			continue
		}
		completed = append(completed, "restore "+mut.DestPath)
		if mut.BackupPath != "" {
			_ = os.Remove(mut.BackupPath)
		}
	}
	if len(outstanding) == 0 {
		return nil
	}
	path, saveErr := installations.SaveRecoveryRecord(installations.RecoveryRecord{
		PackageID:        fontID,
		Scope:            scope,
		AffectedPaths:    affected,
		BackupPaths:      backupPaths,
		CompletedSteps:   completed,
		OutstandingSteps: outstanding,
		Errors:           recErrs,
	})
	msg := fmt.Sprintf("recovery required for %s; backups retained", fontID)
	if path != "" {
		msg = fmt.Sprintf("%s; recovery record: %s", msg, path)
		if len(backupPaths) > 0 {
			msg = fmt.Sprintf("%s; backups: %s", msg, strings.Join(backupPaths, ", "))
		}
		fmt.Printf("%s\n", ui.ErrorText.Render(msg))
	}
	if saveErr != nil {
		return fmt.Errorf("%w: %v: %v", shared.ErrRecoveryRequired, cause, saveErr)
	}
	return fmt.Errorf("%w: %s", shared.ErrRecoveryRequired, msg)
}

func tryRecordInstallationRegistry(fontID string, fontFiles []repo.FontFile, installScope platform.InstallationScope, fontDir string, result *InstallResult) error {
	if fontID == "" || result == nil {
		return nil
	}
	if result.Status != InstallStatusCompleted || result.Success <= 0 || result.Failed != 0 {
		return nil
	}
	if result.Success > len(result.Details) {
		return fmt.Errorf("installation registry: details shorter than success count")
	}
	catalogName := ""
	variantByBasename := make(map[string]string)
	for _, ff := range fontFiles {
		b := filepath.Base(strings.TrimSpace(ff.Path))
		if b == "" {
			continue
		}
		variantByBasename[b] = strings.TrimSpace(ff.Variant)
	}
	if len(fontFiles) > 0 {
		catalogName = strings.TrimSpace(fontFiles[0].Name)
	}
	installedBasenames := result.Details[:result.Success]
	nonEmptyBasenames := 0
	for _, base := range installedBasenames {
		if strings.TrimSpace(base) != "" {
			nonEmptyBasenames++
		}
	}
	var files []installations.InstalledFontFile
	for _, base := range installedBasenames {
		base = strings.TrimSpace(base)
		if base == "" {
			continue
		}
		full := filepath.Join(fontDir, base)
		md, err := platform.ExtractFontMetadata(full)
		if err != nil {
			output.GetDebug().Warning("installation registry: skipping %s (metadata: %v)", full, err)
			continue
		}
		fam := strings.TrimSpace(md.TypographicFamily)
		if fam == "" {
			fam = strings.TrimSpace(md.FamilyName)
		}
		style := strings.TrimSpace(md.TypographicStyle)
		if style == "" {
			style = strings.TrimSpace(md.StyleName)
		}
		fullName := strings.TrimSpace(md.FullName)
		files = append(files, installations.InstalledFontFile{
			Path:           full,
			CatalogVariant: variantByBasename[base],
			SFNT: installations.SFNTSnapshot{
				Family:   fam,
				Style:    style,
				FullName: fullName,
			},
		})
	}
	if len(files) != nonEmptyBasenames {
		return fmt.Errorf("installation registry: incomplete metadata (%d/%d faces)", len(files), nonEmptyBasenames)
	}
	if len(files) == 0 {
		return fmt.Errorf("installation registry: no files to record")
	}
	installSrc := ""
	if meta, metaErr := repo.MatchRepositoryFontByID(fontID); metaErr == nil && meta != nil {
		installSrc = strings.TrimSpace(meta.Source)
	} else if metaErr != nil {
		output.GetDebug().Warning("installation registry: catalog lookup for installation_source failed: %v", metaErr)
	}
	return installations.RecordInstallation(installations.RecordParams{
		FontID:             fontID,
		CatalogName:        catalogName,
		InstallationSource: installSrc,
		Scope:              string(installScope),
		FontGetVersion:     version.GetVersion(),
		Files:              files,
	})
}

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
