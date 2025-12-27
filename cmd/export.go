package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

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

const (
	// ExportManifestVersion is the version of the export manifest format
	ExportManifestVersion = "1.0"
	// ExportManifestExportedBy identifies the tool that created the export
	ExportManifestExportedBy = "fontget"
)

// ExportManifest represents the structure of an exported font manifest
type ExportManifest struct {
	Version    string         `json:"version"`
	ExportedAt string         `json:"exported_at"`
	ExportedBy string         `json:"exported_by"`
	Fonts      []ExportedFont `json:"fonts"`
	Metadata   ExportMetadata `json:"metadata"`
}

// ExportedFont represents a single font in the export manifest
type ExportedFont struct {
	FontID      string   `json:"font_id"`
	FamilyName  string   `json:"family_name,omitempty"` // Deprecated: use FamilyNames instead. This field is maintained for backward compatibility with older export manifests. Planned removal: v2.0.0
	FamilyNames []string `json:"family_names"`          // Array of family names (handles Nerd Fonts with multiple families per Font ID)
	Source      string   `json:"source"`
	License     string   `json:"license"`
	Categories  []string `json:"categories"`
	Variants    []string `json:"variants"`
	Scope       string   `json:"scope"`
}

// ExportMetadata contains metadata about the export
type ExportMetadata struct {
	TotalFonts     int    `json:"total_fonts"`
	TotalVariants  int    `json:"total_variants"`
	FilterByMatch  string `json:"filter_by_match,omitempty"`
	FilterBySource string `json:"filter_by_source,omitempty"`
	OnlyMatched    bool   `json:"only_matched"` // Only fonts that match repository entries
}

var exportCmd = &cobra.Command{
	Use:          "export [output-file]",
	Short:        "Export installed fonts to a manifest file",
	SilenceUsage: true,
	Long: `Export installed fonts to a JSON manifest file that can be used to restore fonts on another system.

By default, exports all fonts that match repository entries (Font IDs available).
System fonts are always excluded from exports.

Use flags to filter by match string, source, or include all installed fonts.

The export file can be used with the 'import' command to install the same fonts on another system.

The output file can be specified as a positional argument or using the -o flag.
When using -o, you can specify either a directory (creates fonts-export.json in that directory)
or a full file path.`,
	Example: `  fontget export fonts.json
  fontget export -o D:\Exports
  fontget export -o D:\Exports\my-fonts.json
  fontget export fonts.json --match "Roboto"
  fontget export fonts.json --source "Google Fonts"
  fontget export fonts.json --all`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Output file is optional - default to fonts-export.json
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		GetLogger().Info("Starting font export operation")

		// Always start with a blank line for consistent spacing
		fmt.Println()

		// Debug output for operation start
		output.GetDebug().State("Starting font export operation")

		// Ensure manifest system is initialized
		if err := cmdutils.EnsureManifestInitialized(func() cmdutils.Logger { return GetLogger() }); err != nil {
			return err
		}

		fontManager, err := cmdutils.CreateFontManager(func() cmdutils.Logger { return GetLogger() })
		if err != nil {
			return err
		}

		// Get flags
		outputFile, _ := cmd.Flags().GetString("output")
		matchFilter, _ := cmd.Flags().GetString("match")
		sourceFilter, _ := cmd.Flags().GetString("source")
		exportAll, _ := cmd.Flags().GetBool("all")
		onlyMatched, _ := cmd.Flags().GetBool("matched")
		force, _ := cmd.Flags().GetBool("force")

		// Determine output file path (use arg if provided, otherwise use flag or default)
		var outputPath string
		if len(args) > 0 {
			outputPath = args[0]
		} else if outputFile != "" {
			outputPath = outputFile
		}

		// Validate and normalize output path using shared utility
		defaultFilename := generateDefaultExportFilename()
		normalizedPath, needsConfirm, err := cmdutils.ValidateOutputPath(outputPath, defaultFilename, ".json", force)
		if err != nil {
			// Check if this is a path validation error
			var pathErr *shared.PathValidationError
			if errors.As(err, &pathErr) {
				cmdutils.PrintError(pathErr.Error())
				fmt.Println()
				return nil
			}
			return err
		}

		// Handle file existence confirmation
		if needsConfirm {
			// File exists - show error message
			cmdutils.PrintErrorf("File already exists: %s", normalizedPath)
			cmdutils.PrintInfo("Use --force to overwrite")
			fmt.Println()
			return nil
		}

		outputFile = normalizedPath

		// Validate flags
		if exportAll && onlyMatched {
			cmdutils.PrintError("Cannot use --all and --matched together")
			fmt.Println()
			return nil
		}
		if matchFilter != "" && sourceFilter != "" {
			cmdutils.PrintError("Cannot use --match and --source together")
			fmt.Println()
			return nil
		}

		// Default to only matched fonts if no flags specified
		if !exportAll && !onlyMatched {
			onlyMatched = true
		}

		// Verbose output
		output.GetVerbose().Info("Exporting installed fonts")
		if matchFilter != "" {
			output.GetVerbose().Info("Filter: Match = %s", matchFilter)
		}
		if sourceFilter != "" {
			output.GetVerbose().Info("Filter: Source = %s", sourceFilter)
		}
		if onlyMatched {
			output.GetVerbose().Info("Filter: Only fonts with Font IDs")
		}
		output.GetVerbose().Info("System fonts are always excluded")
		output.GetVerbose().Info("Output file: %s", outputFile)
		// Verbose section ends with blank line per spacing framework (only if verbose was shown)
		if IsVerbose() {
			fmt.Println()
		}

		// Auto-detect accessible scopes
		availableScopes, err := cmdutils.DetectAccessibleScopes(fontManager)
		if err != nil {
			GetLogger().Error("Failed to detect accessible scopes: %v", err)
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("cmdutils.DetectAccessibleScopes() failed: %v", err)
			return fmt.Errorf("unable to detect accessible font scopes: %w", err)
		}

		// Use all accessible scopes
		scopes := availableScopes

		// For debug mode, do everything without progress bar
		if IsDebug() {
			return performFullExport(fontManager, scopes, outputFile, matchFilter, sourceFilter, exportAll, onlyMatched)
		}

		// Use progress bar for normal/verbose mode
		return runExportWithProgressBar(fontManager, scopes, outputFile, matchFilter, sourceFilter, exportAll, onlyMatched)
	},
}

// generateDefaultExportFilename generates a date-based export filename
func generateDefaultExportFilename() string {
	now := time.Now()
	dateStr := now.Format("2006-01-02")
	return fmt.Sprintf("fontget-export-%s.json", dateStr)
}

// runExportWithProgressBar runs the export operation with a progress bar
func runExportWithProgressBar(fontManager platform.FontManager, scopes []platform.InstallationScope, outputFile, matchFilter, sourceFilter string, exportAll, onlyMatched bool) error {
	// Start progress bar immediately to give visual feedback
	progressTitle := "Exporting Fonts"

	// Start with empty items and TotalItems = 0 to show "(0 of 0)" placeholder
	// This prevents layout jumping when the count is updated later
	operationItems := make([]components.OperationItem, 0)

	// Run progress bar
	verbose := IsVerbose()
	debug := IsDebug()
	var exportedFonts []ExportedFont
	var totalFamilies int
	progressErr := components.RunProgressBar(
		progressTitle,
		operationItems,
		verbose, // Verbose mode: show operational details
		debug,   // Debug mode: show technical details
		func(send func(msg tea.Msg), cancelChan <-chan struct{}) error {
			// Check for cancellation
			select {
			case <-cancelChan:
				return shared.ErrOperationCancelled
			default:
			}

			// Phase 1: Collect fonts (0-20% progress)
			send(components.ProgressUpdateMsg{Percent: 5.0})
			output.GetVerbose().Info("Scanning fonts to determine export scope...")
			fonts, collectErr := collectFonts(scopes, fontManager, "", true) // Suppress verbose - we have our own high-level message
			if collectErr != nil {
				return fmt.Errorf("unable to collect fonts: %w", collectErr)
			}
			output.GetDebug().State("Total fonts to export: %d", len(fonts))
			send(components.ProgressUpdateMsg{Percent: 20.0})

			// Check for cancellation
			select {
			case <-cancelChan:
				return shared.ErrOperationCancelled
			default:
			}

			// Phase 2: Group by family (20-30% progress)
			families := groupByFamily(fonts)
			output.GetVerbose().Info("Grouped into %d font families", len(families))
			send(components.ProgressUpdateMsg{Percent: 30.0})

			// Check for cancellation
			select {
			case <-cancelChan:
				return shared.ErrOperationCancelled
			default:
			}

			// Phase 3: Match installed fonts to repository (30-50% progress)
			var names []string
			for k := range families {
				names = append(names, k)
			}
			sort.Strings(names)

			send(components.ProgressUpdateMsg{Percent: 35.0})
			matches, matchErr := cmdutils.MatchInstalledFontsToRepository(names, GetLogger(), shared.IsCriticalSystemFont)
			if matchErr != nil {
				// Continue without matches if exportAll is true
				if !exportAll {
					return fmt.Errorf("unable to match fonts to repository: %w", matchErr)
				}
				matches = make(map[string]*repo.InstalledFontMatch)
			}
			send(components.ProgressUpdateMsg{Percent: 50.0})

			// Check for cancellation
			select {
			case <-cancelChan:
				return shared.ErrOperationCancelled
			default:
			}

			// Phase 4: Populate match data and filter fonts (50-60% progress)
			populateFontMatchData(families, matches)

			fontIDGroups, skippedSystem, skippedUnmatched, skippedByFilter := filterFontsForExport(FilterFontsForExportParams{
				Families:     families,
				Names:        names,
				MatchFilter:  matchFilter,
				SourceFilter: sourceFilter,
				ExportAll:    exportAll,
				OnlyMatched:  onlyMatched,
			})

			// Count total families to export
			totalFamilies = len(fontIDGroups)
			if totalFamilies == 0 {
				// No fonts found - we'll handle this after the progress bar
				return nil
			}

			// Update total items now that we know the count
			// This will make it show "Exporting Fonts (0 of y)" immediately
			send(components.TotalItemsUpdateMsg{TotalItems: totalFamilies})
			send(components.ProgressUpdateMsg{Percent: 60.0})

			// Check for cancellation
			select {
			case <-cancelChan:
				return shared.ErrOperationCancelled
			default:
			}

			// Phase 5: Perform the actual export operation (60-100% progress)
			params := ExportProgressParams{
				FontManager:      fontManager,
				Scopes:           scopes,
				OutputFile:       outputFile,
				MatchFilter:      matchFilter,
				SourceFilter:     sourceFilter,
				ExportAll:        exportAll,
				OnlyMatched:      onlyMatched,
				Families:         families,
				Names:            names,
				Matches:          matches,
				FontIDGroups:     fontIDGroups,
				SkippedSystem:    skippedSystem,
				SkippedUnmatched: skippedUnmatched,
				SkippedByFilter:  skippedByFilter,
				TotalFamilies:    totalFamilies,
			}
			var exportErr error
			exportedFonts, _, exportErr = performExportWithProgress(params, send, cancelChan)
			return exportErr
		},
	)

	if progressErr != nil {
		// Check if it was a cancellation
		if errors.Is(progressErr, shared.ErrOperationCancelled) {
			cmdutils.PrintWarning("Export cancelled.")
			fmt.Println()
			return nil // Don't return error for cancellation
		}
		// Print error with proper styling (Cobra won't print it since SilenceErrors is true)
		cmdutils.PrintErrorf("%v", progressErr)
		fmt.Println()
		return progressErr
	}

	// Check if no fonts were found (this happens if totalFamilies is 0)
	if totalFamilies == 0 {
		cmdutils.PrintWarning("No fonts found matching the specified criteria.")
		fmt.Println()
		return nil
	}

	// Show success message after progress bar completes
	if len(exportedFonts) > 0 {
		// Show destination path with checkmark, matching backup command style
		fmt.Printf("  %s %s\n", ui.SuccessText.Render("✓"), ui.Text.Render(fmt.Sprintf("Font manifest exported to: %s", ui.InfoText.Render(fmt.Sprintf("'%s'", outputFile)))))
		fmt.Println()
	}

	return nil
}

// ExportProgressParams contains parameters for performExportWithProgress
type ExportProgressParams struct {
	FontManager      platform.FontManager
	Scopes           []platform.InstallationScope
	OutputFile       string
	MatchFilter      string
	SourceFilter     string
	ExportAll        bool
	OnlyMatched      bool
	Families         map[string][]ParsedFont
	Names            []string
	Matches          map[string]*repo.InstalledFontMatch
	FontIDGroups     map[string]*fontIDGroup
	SkippedSystem    int
	SkippedUnmatched int
	SkippedByFilter  int
	TotalFamilies    int
}

// performExportWithProgress performs the export operation with progress tracking
func performExportWithProgress(params ExportProgressParams, send func(msg tea.Msg), cancelChan <-chan struct{}) ([]ExportedFont, int, error) {
	// Check for cancellation
	if cancelChan != nil {
		select {
		case <-cancelChan:
			return nil, 0, shared.ErrOperationCancelled
		default:
			// Continue processing
		}
	}

	// Build export manifest
	manifest, totalVariants := buildExportManifest(
		params.FontIDGroups, params.MatchFilter, params.SourceFilter, params.OnlyMatched, params.SkippedSystem, params.SkippedUnmatched, params.SkippedByFilter)

	// Update progress
	if send != nil {
		send(components.ProgressUpdateMsg{Percent: 50.0})
	}

	// Check for cancellation before writing
	if cancelChan != nil {
		select {
		case <-cancelChan:
			return nil, 0, shared.ErrOperationCancelled
		default:
			// Continue processing
		}
	}

	// Write manifest
	output.GetVerbose().Info("Writing export file...")
	jsonData, err := json.MarshalIndent(*manifest, "", "  ")
	if err != nil {
		GetLogger().Error("Failed to marshal export manifest: %v", err)
		output.GetVerbose().Error("%v", err)
		output.GetDebug().Error("json.MarshalIndent() failed: %v", err)
		return nil, 0, fmt.Errorf("unable to marshal manifest: %w", err)
	}

	// Ensure parent directory exists (skip if outputFile is just a filename)
	if dir := filepath.Dir(params.OutputFile); dir != "." && dir != params.OutputFile {
		if err := os.MkdirAll(dir, 0755); err != nil {
			GetLogger().Error("Failed to create export directory: %v", err)
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("os.MkdirAll() failed for parent directory: %v", err)
			return nil, 0, fmt.Errorf("unable to create directory for export file: %w", err)
		}
	}

	if err := os.WriteFile(params.OutputFile, jsonData, 0644); err != nil {
		GetLogger().Error("Failed to write export file: %v", err)
		output.GetVerbose().Error("%v", err)
		output.GetDebug().Error("os.WriteFile() failed: %v", err)
		return nil, 0, fmt.Errorf("unable to write export file: %w", err)
	}
	GetLogger().Info("Export file written successfully: %s", params.OutputFile)
	output.GetVerbose().Info("Export file written successfully")

	// Update progress to 100%
	if send != nil {
		send(components.ProgressUpdateMsg{Percent: 100.0})
		// Mark all items as completed so the count shows correctly
		// The progress bar component counts items with status "completed", "failed", or "skipped"
		for i := 0; i < params.TotalFamilies; i++ {
			send(components.ItemUpdateMsg{
				Index:  i,
				Status: "completed",
			})
		}
	}

	return manifest.Fonts, totalVariants, nil
}

// performFullExport performs the complete export process (for debug mode)
func performFullExport(fontManager platform.FontManager, scopes []platform.InstallationScope, outputFile, matchFilter, sourceFilter string, exportAll, onlyMatched bool) error {
	exportedFonts, _, err := performFullExportWithResult(fontManager, scopes, outputFile, matchFilter, sourceFilter, exportAll, onlyMatched)
	if err != nil {
		return err
	}

	if len(exportedFonts) == 0 {
		// Start with a blank line for consistent spacing
		fmt.Println()
		fmt.Printf("%s\n", ui.WarningText.Render("No fonts found matching the specified criteria."))
		fmt.Println()
		return nil
	}

	output.GetDebug().State("Export operation complete - Exported: %d font families", len(exportedFonts))
	fmt.Printf("  %s %s\n", ui.SuccessText.Render("✓"), ui.Text.Render(fmt.Sprintf("Font manifest exported to: %s", ui.InfoText.Render(fmt.Sprintf("'%s'", outputFile)))))
	fmt.Println()
	return nil
}

// performFullExportWithResult performs the complete export process and returns the results
// fontIDGroup represents a group of fonts with the same Font ID (or family name for unmatched fonts)
type fontIDGroup struct {
	familyNames []string
	source      string
	license     string
	categories  []string
	variants    map[string]bool
	scope       string
	hasFontID   bool // Track if this group has a Font ID or is keyed by family name
}

// populateFontMatchData populates ParsedFont structs with match data
func populateFontMatchData(families map[string][]ParsedFont, matches map[string]*repo.InstalledFontMatch) {
	for familyName, fontGroup := range families {
		if match, exists := matches[familyName]; exists && match != nil {
			for i := range fontGroup {
				fontGroup[i].FontID = match.FontID
				fontGroup[i].License = match.License
				fontGroup[i].Categories = match.Categories
				fontGroup[i].Source = match.Source
			}
			families[familyName] = fontGroup
		}
	}
}

// FilterFontsForExportParams contains parameters for filterFontsForExport function
type FilterFontsForExportParams struct {
	Families     map[string][]ParsedFont
	Names        []string
	MatchFilter  string
	SourceFilter string
	ExportAll    bool
	OnlyMatched  bool
}

// filterFontsForExport filters fonts based on the provided criteria and groups them by Font ID
// Returns: (fontIDGroups, skippedSystem, skippedUnmatched, skippedByFilter)
func filterFontsForExport(params FilterFontsForExportParams) (map[string]*fontIDGroup, int, int, int) {
	fontIDGroups := make(map[string]*fontIDGroup)
	skippedSystem := 0
	skippedUnmatched := 0
	skippedByFilter := 0

	for _, familyName := range params.Names {
		// Skip system fonts
		if shared.IsCriticalSystemFont(familyName) {
			skippedSystem++
			continue
		}

		fontGroup := params.Families[familyName]
		if len(fontGroup) == 0 {
			continue
		}

		// Get match data from first font in group (all should have same match data)
		firstFont := fontGroup[0]
		hasFontID := firstFont.FontID != ""

		// Apply filters
		if params.OnlyMatched && !hasFontID {
			skippedUnmatched++
			continue
		}

		if params.MatchFilter != "" {
			// Check if family name or Font ID matches
			matched := strings.Contains(strings.ToLower(familyName), strings.ToLower(params.MatchFilter))
			if !matched && hasFontID {
				matched = strings.Contains(strings.ToLower(firstFont.FontID), strings.ToLower(params.MatchFilter))
			}
			if !matched {
				skippedByFilter++
				continue
			}
		}

		if params.SourceFilter != "" {
			if firstFont.Source == "" || !strings.Contains(strings.ToLower(firstFont.Source), strings.ToLower(params.SourceFilter)) {
				skippedByFilter++
				continue
			}
		}

		// Determine group key (Font ID if available, otherwise family name)
		groupKey := familyName
		if hasFontID {
			groupKey = firstFont.FontID
		}

		// Get or create group
		group, exists := fontIDGroups[groupKey]
		if !exists {
			group = &fontIDGroup{
				familyNames: []string{},
				source:      firstFont.Source,
				license:     firstFont.License,
				categories:  firstFont.Categories,
				variants:    make(map[string]bool),
				scope:       string(fontGroup[0].Scope),
				hasFontID:   hasFontID,
			}
			fontIDGroups[groupKey] = group
		}

		// Add family name if not already present
		familyExists := false
		for _, existingFamily := range group.familyNames {
			if existingFamily == familyName {
				familyExists = true
				break
			}
		}
		if !familyExists {
			group.familyNames = append(group.familyNames, familyName)
		}

		// Add variants
		for _, font := range fontGroup {
			if font.Style != "" {
				group.variants[font.Style] = true
			}
		}
	}

	return fontIDGroups, skippedSystem, skippedUnmatched, skippedByFilter
}

// buildExportManifest builds an ExportManifest from font ID groups
// Returns:
//   - *ExportManifest: The complete export manifest
//   - int: Total number of font variants exported
func buildExportManifest(fontIDGroups map[string]*fontIDGroup, matchFilter, sourceFilter string, onlyMatched bool, skippedSystem, skippedUnmatched, skippedByFilter int) (*ExportManifest, int) {
	// Convert groups to exported fonts
	var exportedFonts []ExportedFont
	totalVariants := 0

	for key, group := range fontIDGroups {
		// Sort family names for consistent output
		sort.Strings(group.familyNames)

		// Convert variants map to sorted slice
		variants := make([]string, 0, len(group.variants))
		for variant := range group.variants {
			variants = append(variants, variant)
		}
		sort.Strings(variants)
		totalVariants += len(variants)

		// Determine Font ID - use key if group has Font ID, otherwise empty (font without Font ID)
		fontID := key
		if !group.hasFontID {
			fontID = "" // Empty Font ID for fonts that don't match repository
		}

		if fontID != "" {
			output.GetDebug().State("Including Font ID: %s with families: %v (Source: %s)", fontID, group.familyNames, group.source)
		} else {
			output.GetDebug().State("Including font without Font ID: %v (Source: %s)", group.familyNames, group.source)
		}

		exportedFont := ExportedFont{
			FontID:      fontID,
			FamilyNames: group.familyNames,
			Source:      group.source,
			License:     group.license,
			Categories:  group.categories,
			Variants:    variants,
			Scope:       group.scope,
		}

		exportedFonts = append(exportedFonts, exportedFont)
	}

	// Sort exported fonts by Font ID for consistent output
	// Fonts with Font IDs come first, then fonts without Font IDs (sorted by family name)
	sort.Slice(exportedFonts, func(i, j int) bool {
		// If one has Font ID and the other doesn't, Font ID comes first
		if exportedFonts[i].FontID != "" && exportedFonts[j].FontID == "" {
			return true
		}
		if exportedFonts[i].FontID == "" && exportedFonts[j].FontID != "" {
			return false
		}
		// Both have Font IDs or both don't - sort by Font ID or first family name
		if exportedFonts[i].FontID != "" {
			return exportedFonts[i].FontID < exportedFonts[j].FontID
		}
		// Both are without Font ID - sort by first family name
		if len(exportedFonts[i].FamilyNames) > 0 && len(exportedFonts[j].FamilyNames) > 0 {
			return exportedFonts[i].FamilyNames[0] < exportedFonts[j].FamilyNames[0]
		}
		return false
	})

	output.GetVerbose().Info("Building export manifest...")
	output.GetVerbose().Info("Exporting %d font families (%d variants)", len(exportedFonts), totalVariants)
	output.GetDebug().State("Export summary: %d exported, %d skipped (system: %d, unmatched: %d, filtered: %d)", len(exportedFonts), skippedSystem+skippedUnmatched+skippedByFilter, skippedSystem, skippedUnmatched, skippedByFilter)

	manifest := &ExportManifest{
		Version:    ExportManifestVersion,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		ExportedBy: ExportManifestExportedBy,
		Fonts:      exportedFonts,
		Metadata: ExportMetadata{
			TotalFonts:     len(exportedFonts),
			TotalVariants:  totalVariants,
			FilterByMatch:  matchFilter,
			FilterBySource: sourceFilter,
			OnlyMatched:    onlyMatched,
		},
	}

	return manifest, totalVariants
}

func performFullExportWithResult(fontManager platform.FontManager, scopes []platform.InstallationScope, outputFile, matchFilter, sourceFilter string, exportAll, onlyMatched bool) ([]ExportedFont, int, error) {
	output.GetDebug().State("Calling performFullExportWithResult(scopes=%v, outputFile=%s, matchFilter=%s, sourceFilter=%s, exportAll=%v, onlyMatched=%v)", scopes, outputFile, matchFilter, sourceFilter, exportAll, onlyMatched)

	// Collect fonts from all scopes
	output.GetVerbose().Info("Collecting installed fonts...")
	fonts, err := collectFonts(scopes, fontManager, "", true) // Suppress verbose - we have our own high-level message
	if err != nil {
		GetLogger().Error("Failed to collect fonts: %v", err)
		output.GetVerbose().Error("%v", err)
		output.GetDebug().Error("collectFonts() failed: %v", err)
		return nil, 0, fmt.Errorf("unable to read installed fonts: %w", err)
	}

	output.GetVerbose().Info("Found %d font files", len(fonts))
	output.GetDebug().State("Total fonts to export: %d", len(fonts))

	// Group by family
	families := groupByFamily(fonts)
	output.GetVerbose().Info("Grouped into %d font families", len(families))

	// Match installed fonts to repository
	var names []string
	for k := range families {
		names = append(names, k)
	}
	sort.Strings(names)

	matches, err := cmdutils.MatchInstalledFontsToRepository(names, GetLogger(), shared.IsCriticalSystemFont)
	if err != nil {
		// Continue without matches if exportAll is true
		if !exportAll {
			return nil, 0, fmt.Errorf("unable to match fonts to repository: %w", err)
		}
		matches = make(map[string]*repo.InstalledFontMatch)
	}

	// Populate match data
	populateFontMatchData(families, matches)

	// Filter fonts and group by Font ID
	fontIDGroups, skippedSystem, skippedUnmatched, skippedByFilter := filterFontsForExport(FilterFontsForExportParams{
		Families:     families,
		Names:        names,
		MatchFilter:  matchFilter,
		SourceFilter: sourceFilter,
		ExportAll:    exportAll,
		OnlyMatched:  onlyMatched,
	})

	// Build export manifest
	manifest, totalVariants := buildExportManifest(
		fontIDGroups, matchFilter, sourceFilter, onlyMatched, skippedSystem, skippedUnmatched, skippedByFilter)

	// Write manifest
	output.GetVerbose().Info("Writing export file...")
	jsonData, err := json.MarshalIndent(*manifest, "", "  ")
	if err != nil {
		GetLogger().Error("Failed to marshal export manifest: %v", err)
		output.GetVerbose().Error("%v", err)
		output.GetDebug().Error("json.MarshalIndent() failed: %v", err)
		return nil, 0, fmt.Errorf("unable to marshal manifest: %w", err)
	}

	// Ensure parent directory exists (skip if outputFile is just a filename)
	if dir := filepath.Dir(outputFile); dir != "." && dir != outputFile {
		if err := os.MkdirAll(dir, 0755); err != nil {
			GetLogger().Error("Failed to create export directory: %v", err)
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("os.MkdirAll() failed for parent directory: %v", err)
			return nil, 0, fmt.Errorf("unable to create directory for export file: %w", err)
		}
	}

	if err := os.WriteFile(outputFile, jsonData, 0644); err != nil {
		GetLogger().Error("Failed to write export file: %v", err)
		output.GetVerbose().Error("%v", err)
		output.GetDebug().Error("os.WriteFile() failed: %v", err)
		return nil, 0, fmt.Errorf("unable to write export file: %w", err)
	}
	GetLogger().Info("Export file written successfully: %s", outputFile)
	output.GetVerbose().Info("Export file written successfully")

	return manifest.Fonts, totalVariants, nil
}

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.Flags().StringP("output", "o", "", "Output file path (default: fonts-export.json)")
	exportCmd.Flags().StringP("match", "m", "", "Export fonts that match the specified string")
	exportCmd.Flags().StringP("source", "s", "", "Filter by font source (e.g., 'Google Fonts')")
	exportCmd.Flags().BoolP("all", "a", false, "Export all installed fonts (including those without Font IDs)")
	exportCmd.Flags().Bool("matched", false, "Export only fonts that match repository entries (default, cannot be used with --all)")
	exportCmd.Flags().BoolP("force", "f", false, "Force overwrite existing file without confirmation")
}
