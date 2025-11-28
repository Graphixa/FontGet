package cmd

import (
	"encoding/json"
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

	"github.com/spf13/cobra"
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
	FamilyName  string   `json:"family_name,omitempty"` // Deprecated: use FamilyNames instead
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

		// Determine output file
		if outputFile == "" {
			if len(args) > 0 {
				outputFile = args[0]
			} else {
				outputFile = generateDefaultExportFilename()
			}
		} else {
			// If -o flag is provided, handle directory vs file path (similar to winget)
			info, err := os.Stat(outputFile)
			if err == nil {
				// Path exists
				if info.IsDir() {
					// It's an existing directory, use date-based default filename in that directory
					outputFile = filepath.Join(outputFile, generateDefaultExportFilename())
				}
				// If it's an existing file, use it as-is (will check and prompt later)
			} else if os.IsNotExist(err) {
				// Path doesn't exist - determine if it should be a directory or file
				if strings.HasSuffix(outputFile, string(filepath.Separator)) {
					// Ends with path separator, definitely a directory
					if err := os.MkdirAll(outputFile, 0755); err == nil {
						outputFile = filepath.Join(outputFile, generateDefaultExportFilename())
					}
				} else if !strings.HasSuffix(strings.ToLower(outputFile), ".json") {
					// No .json extension, treat as directory
					if err := os.MkdirAll(outputFile, 0755); err == nil {
						outputFile = filepath.Join(outputFile, generateDefaultExportFilename())
					}
				}
				// If it has .json extension, treat as file path (will create parent dirs if needed later)
			}
		}

		// Validate and normalize output path (with overwrite confirmation)
		outputFile, err = validateAndNormalizeExportPath(outputFile)
		if err != nil {
			// Check if this is a cancellation (user chose not to overwrite)
			if strings.Contains(err.Error(), "export cancelled") {
				// User cancelled - show friendly message and return nil (no error)
				fmt.Printf("%s\n", ui.FeedbackWarning.Render("Export cancelled - file already exists."))
				fmt.Println()
				return nil
			}
			return err
		}

		// Validate flags
		if exportAll && onlyMatched {
			return fmt.Errorf("cannot use --all and --matched together")
		}
		if matchFilter != "" && sourceFilter != "" {
			return fmt.Errorf("cannot use --match and --source together")
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

		// Collect fonts from all scopes
		scopes := []platform.InstallationScope{platform.UserScope, platform.MachineScope}

		// For debug mode, do everything without spinner
		if IsDebug() {
			return performFullExport(fontManager, scopes, outputFile, matchFilter, sourceFilter, exportAll, onlyMatched)
		}

		// Use pin spinner for normal/verbose mode - wrap all the work
		var exportedFonts []ExportedFont

		err = ui.RunSpinner("Exporting fonts...", "Exported fonts", func() error {
			var err error
			exportedFonts, _, err = performFullExportWithResult(fontManager, scopes, outputFile, matchFilter, sourceFilter, exportAll, onlyMatched)
			return err
		})

		if err != nil {
			return err
		}

		// Check if we have fonts to export
		if len(exportedFonts) == 0 {
			// Start with a blank line for consistent spacing
			fmt.Println()
			fmt.Printf("%s\n", ui.FeedbackWarning.Render("No fonts found matching the specified criteria."))
			fmt.Println()
			return nil
		}

		// Log completion
		GetLogger().Info("Export operation complete - Exported %d font families to %s", len(exportedFonts), outputFile)

		// Show success message
		fmt.Printf("%s\n", ui.FeedbackSuccess.Render(fmt.Sprintf("Successfully exported %d font families to %s", len(exportedFonts), outputFile)))
		fmt.Println()

		return nil
	},
}

// generateDefaultExportFilename generates a date-based export filename
func generateDefaultExportFilename() string {
	now := time.Now()
	dateStr := now.Format("2006-01-02")
	return fmt.Sprintf("fontget-export-%s.json", dateStr)
}

// validateAndNormalizeExportPath validates and normalizes the export output path with overwrite confirmation
func validateAndNormalizeExportPath(outputPath string) (string, error) {
	// Normalize path separators
	outputPath = filepath.Clean(outputPath)

	// Ensure it has .json extension
	if !strings.HasSuffix(strings.ToLower(outputPath), ".json") {
		outputPath = outputPath + ".json"
	}

	// Final validation: ensure it's an absolute or relative path
	absPath, err := filepath.Abs(outputPath)
	if err != nil {
		return "", fmt.Errorf("invalid output path: %v", err)
	}

	// Check if the final file path already exists and prompt for confirmation
	if _, err := os.Stat(absPath); err == nil {
		// File exists - prompt for confirmation before overwriting
		confirmed, err := components.RunConfirm(
			"File Already Exists",
			fmt.Sprintf("File already exists. Overwrite '%s'?", filepath.Base(absPath)),
		)
		if err != nil {
			return "", fmt.Errorf("unable to show confirmation dialog: %v", err)
		}

		if !confirmed {
			return "", fmt.Errorf("export cancelled - file already exists: %s", absPath)
		}
	}

	return absPath, nil
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
		fmt.Printf("%s\n", ui.FeedbackWarning.Render("No fonts found matching the specified criteria."))
		fmt.Println()
		return nil
	}

	output.GetDebug().State("Export operation complete - Exported: %d font families", len(exportedFonts))
	fmt.Printf("%s\n", ui.FeedbackSuccess.Render(fmt.Sprintf("Successfully exported %d font families to %s", len(exportedFonts), outputFile)))
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

// filterFontsForExport applies match/source/exportAll filters to font families and groups them by Font ID.
//
// It filters font families based on match string, source name, and export flags, then groups
// them by Font ID (or family name for fonts without Font IDs when exportAll is true).
// System fonts are always excluded from exports.
//
// Parameters:
//   - params: FilterFontsForExportParams containing families, filters, and export flags
//
// Returns:
//   - fontIDGroups: Map of Font ID (or family name) to font group
//   - skippedSystem: Count of skipped system fonts
//   - skippedUnmatched: Count of skipped fonts without Font IDs (when onlyMatched is true)
//   - skippedByFilter: Count of fonts skipped due to match/source filters
func filterFontsForExport(params FilterFontsForExportParams) (fontIDGroups map[string]*fontIDGroup, skippedSystem, skippedUnmatched, skippedByFilter int) {
	fontIDGroups = make(map[string]*fontIDGroup)

	for _, familyName := range params.Names {
		fontGroup := params.Families[familyName]
		rep := fontGroup[0]

		// Always exclude system fonts
		if shared.IsCriticalSystemFont(familyName) {
			skippedSystem++
			continue
		}

		// Apply filters
		if params.MatchFilter != "" && !strings.Contains(strings.ToLower(familyName), strings.ToLower(params.MatchFilter)) {
			skippedByFilter++
			continue
		}

		if params.SourceFilter != "" && rep.Source != params.SourceFilter {
			skippedByFilter++
			continue
		}

		// Only matched fonts
		if params.OnlyMatched && rep.FontID == "" {
			skippedUnmatched++
			continue
		}

		// Handle fonts without Font ID (when exportAll is true)
		if rep.FontID == "" {
			if !params.ExportAll {
				// Skip fonts without Font ID unless --all is specified
				skippedUnmatched++
				continue
			}
			// For --all mode, group fonts without Font ID by family name
			// Use family name as the key since there's no Font ID
			group, exists := fontIDGroups[familyName]
			if !exists {
				group = &fontIDGroup{
					familyNames: make([]string, 0),
					source:      rep.Source,     // Will be empty for unmatched fonts
					license:     rep.License,    // Will be empty for unmatched fonts
					categories:  rep.Categories, // Will be empty for unmatched fonts
					variants:    make(map[string]bool),
					scope:       rep.Scope,
					hasFontID:   false, // This group is keyed by family name, not Font ID
				}
				fontIDGroups[familyName] = group
			}
			// Add family name to group
			group.familyNames = append(group.familyNames, familyName)

			// Collect variants from this family
			for _, font := range fontGroup {
				group.variants[font.Style] = true
			}
			continue
		}

		// Get or create group for this Font ID
		group, exists := fontIDGroups[rep.FontID]
		if !exists {
			group = &fontIDGroup{
				familyNames: make([]string, 0),
				source:      rep.Source,
				license:     rep.License,
				categories:  rep.Categories,
				variants:    make(map[string]bool),
				scope:       rep.Scope,
				hasFontID:   true, // This group has a Font ID
			}
			fontIDGroups[rep.FontID] = group
		}

		// Add family name to group
		group.familyNames = append(group.familyNames, familyName)

		// Collect variants from this family
		for _, font := range fontGroup {
			group.variants[font.Style] = true
		}
	}

	return fontIDGroups, skippedSystem, skippedUnmatched, skippedByFilter
}

// buildExportManifest builds export manifest from filtered fonts.
//
// It converts font ID groups into ExportedFont entries, sorts them for consistent output,
// and creates a complete ExportManifest with metadata about the export operation.
//
// Parameters:
//   - fontIDGroups: Map of Font ID to font group (from filterFontsForExport)
//   - matchFilter: Match filter string used (for metadata)
//   - sourceFilter: Source filter string used (for metadata)
//   - onlyMatched: Whether only matched fonts were exported (for metadata)
//   - skippedSystem: Count of skipped system fonts (for metadata)
//   - skippedUnmatched: Count of skipped unmatched fonts (for metadata)
//   - skippedByFilter: Count of fonts skipped by filters (for metadata)
//
// Returns:
//   - *ExportManifest: Complete export manifest ready for JSON serialization
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
		Version:    "1.0",
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		ExportedBy: "fontget",
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
		return nil, 0, fmt.Errorf("unable to read installed fonts: %v", err)
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
			return nil, 0, fmt.Errorf("unable to match fonts to repository: %v", err)
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
		return nil, 0, fmt.Errorf("unable to marshal manifest: %v", err)
	}

	// Ensure parent directory exists (skip if outputFile is just a filename)
	if dir := filepath.Dir(outputFile); dir != "." && dir != outputFile {
		if err := os.MkdirAll(dir, 0755); err != nil {
			GetLogger().Error("Failed to create export directory: %v", err)
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("os.MkdirAll() failed for parent directory: %v", err)
			return nil, 0, fmt.Errorf("unable to create directory for export file: %v", err)
		}
	}

	if err := os.WriteFile(outputFile, jsonData, 0644); err != nil {
		GetLogger().Error("Failed to write export file: %v", err)
		output.GetVerbose().Error("%v", err)
		output.GetDebug().Error("os.WriteFile() failed: %v", err)
		return nil, 0, fmt.Errorf("unable to write export file: %v", err)
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
}
