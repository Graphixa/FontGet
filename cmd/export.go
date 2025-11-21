package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"fontget/internal/config"
	"fontget/internal/output"
	"fontget/internal/platform"
	"fontget/internal/repo"
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
	Use:   "export [output-file]",
	Short: "Export installed fonts to a manifest file",
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
		// Always start with a blank line for consistent spacing
		fmt.Println()

		// Ensure manifest system is initialized
		if err := config.EnsureManifestExists(); err != nil {
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("config.EnsureManifestExists() failed: %v", err)
			return fmt.Errorf("unable to load font repository: %v", err)
		}

		fm, err := platform.NewFontManager()
		if err != nil {
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("platform.NewFontManager() failed: %v", err)
			return fmt.Errorf("unable to access system fonts: %v", err)
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
				outputFile = "fonts-export.json"
			}
		} else {
			// If -o flag is provided, handle directory vs file path (similar to winget)
			info, err := os.Stat(outputFile)
			if err == nil {
				// Path exists
				if info.IsDir() {
					// It's an existing directory, use default filename in that directory
					outputFile = filepath.Join(outputFile, "fonts-export.json")
				}
				// If it's an existing file, use it as-is (will overwrite)
			} else if os.IsNotExist(err) {
				// Path doesn't exist - determine if it should be a directory or file
				if strings.HasSuffix(outputFile, string(filepath.Separator)) {
					// Ends with path separator, definitely a directory
					if err := os.MkdirAll(outputFile, 0755); err == nil {
						outputFile = filepath.Join(outputFile, "fonts-export.json")
					}
				} else if !strings.HasSuffix(strings.ToLower(outputFile), ".json") {
					// No .json extension, treat as directory
					if err := os.MkdirAll(outputFile, 0755); err == nil {
						outputFile = filepath.Join(outputFile, "fonts-export.json")
					}
				}
				// If it has .json extension, treat as file path (will create parent dirs if needed later)
			}
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
		if IsVerbose() && !IsDebug() {
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
			fmt.Println()
		}

		// Collect fonts from all scopes
		scopes := []platform.InstallationScope{platform.UserScope, platform.MachineScope}

		// For debug mode, do everything without spinner
		if IsDebug() {
			return performFullExport(fm, scopes, outputFile, matchFilter, sourceFilter, exportAll, onlyMatched)
		}

		// Use pin spinner for normal/verbose mode - wrap all the work
		var exportedFonts []ExportedFont

		err = ui.RunSpinner("Exporting fonts...", "Exported fonts", func() error {
			var err error
			exportedFonts, _, err = performFullExportWithResult(fm, scopes, outputFile, matchFilter, sourceFilter, exportAll, onlyMatched)
			return err
		})

		if err != nil {
			return err
		}

		// Check if we have fonts to export
		if len(exportedFonts) == 0 {
			fmt.Printf("%s\n", ui.PageTitle.Render("Export Fonts"))
			fmt.Printf("%s\n", ui.FeedbackWarning.Render("No fonts found matching the specified criteria."))
			fmt.Println()
			return nil
		}

		// Show success message
		fmt.Printf("%s\n", ui.FeedbackSuccess.Render(fmt.Sprintf("Successfully exported %d font families to %s", len(exportedFonts), outputFile)))
		fmt.Println()

		return nil
	},
}

// performFullExport performs the complete export process (for debug mode)
func performFullExport(fm platform.FontManager, scopes []platform.InstallationScope, outputFile, matchFilter, sourceFilter string, exportAll, onlyMatched bool) error {
	exportedFonts, _, err := performFullExportWithResult(fm, scopes, outputFile, matchFilter, sourceFilter, exportAll, onlyMatched)
	if err != nil {
		return err
	}

	if len(exportedFonts) == 0 {
		fmt.Printf("%s\n", ui.PageTitle.Render("Export Fonts"))
		fmt.Printf("%s\n", ui.FeedbackWarning.Render("No fonts found matching the specified criteria."))
		fmt.Println()
		return nil
	}

	output.GetDebug().State("Export completed successfully: %d font families exported to %s", len(exportedFonts), outputFile)
	fmt.Printf("%s\n", ui.FeedbackSuccess.Render(fmt.Sprintf("Successfully exported %d font families to %s", len(exportedFonts), outputFile)))
	fmt.Println()
	return nil
}

// performFullExportWithResult performs the complete export process and returns the results
func performFullExportWithResult(fm platform.FontManager, scopes []platform.InstallationScope, outputFile, matchFilter, sourceFilter string, exportAll, onlyMatched bool) ([]ExportedFont, int, error) {
	// Collect fonts from all scopes
	output.GetVerbose().Info("Collecting installed fonts...")
	fonts, err := collectFonts(scopes, fm)
	if err != nil {
		output.GetVerbose().Error("%v", err)
		output.GetDebug().Error("collectFonts() failed: %v", err)
		return nil, 0, fmt.Errorf("unable to read installed fonts: %v", err)
	}

	output.GetVerbose().Info("Found %d font files", len(fonts))
	output.GetDebug().State("Collected %d font files", len(fonts))

	// Group by family
	families := groupByFamily(fonts)
	output.GetVerbose().Info("Grouped into %d font families", len(families))

	// Match installed fonts to repository
	var names []string
	for k := range families {
		names = append(names, k)
	}
	sort.Strings(names)

	output.GetVerbose().Info("Matching installed fonts to repository...")
	output.GetDebug().State("Total installed font families to match: %d", len(names))
	matches, err := repo.MatchAllInstalledFonts(names, IsCriticalSystemFont)
	if err != nil {
		output.GetVerbose().Error("%v", err)
		output.GetDebug().Error("repo.MatchAllInstalledFonts() failed: %v", err)
		// Continue without matches if exportAll is true
		if !exportAll {
			return nil, 0, fmt.Errorf("unable to match fonts to repository: %v", err)
		}
		matches = make(map[string]*repo.InstalledFontMatch)
	}
	output.GetVerbose().Info("Matched %d font families to repository entries", len(matches))
	output.GetDebug().State("Matched %d font families to repository entries", len(matches))

	// Populate match data
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

	// Build export manifest - group by Font ID to handle Nerd Fonts (one Font ID = multiple families)
	type fontIDGroup struct {
		familyNames []string
		source      string
		license     string
		categories  []string
		variants    map[string]bool
		scope       string
		hasFontID   bool // Track if this group has a Font ID or is keyed by family name
	}

	fontIDGroups := make(map[string]*fontIDGroup)
	skippedSystem := 0
	skippedUnmatched := 0
	skippedByFilter := 0

	for _, familyName := range names {
		fontGroup := families[familyName]
		rep := fontGroup[0]

		// Always exclude system fonts
		if IsCriticalSystemFont(familyName) {
			skippedSystem++
			continue
		}

		// Apply filters
		if matchFilter != "" && !strings.Contains(strings.ToLower(familyName), strings.ToLower(matchFilter)) {
			skippedByFilter++
			continue
		}

		if sourceFilter != "" && rep.Source != sourceFilter {
			skippedByFilter++
			continue
		}

		// Only matched fonts
		if onlyMatched && rep.FontID == "" {
			skippedUnmatched++
			continue
		}

		// Handle fonts without Font ID (when exportAll is true)
		if rep.FontID == "" {
			if !exportAll {
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

	// Build and write manifest
	manifest := ExportManifest{
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

	// Write manifest
	output.GetVerbose().Info("Writing export file...")
	jsonData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		output.GetVerbose().Error("%v", err)
		output.GetDebug().Error("json.MarshalIndent() failed: %v", err)
		return nil, 0, fmt.Errorf("unable to marshal manifest: %v", err)
	}

	// Ensure parent directory exists (skip if outputFile is just a filename)
	if dir := filepath.Dir(outputFile); dir != "." && dir != outputFile {
		if err := os.MkdirAll(dir, 0755); err != nil {
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("os.MkdirAll() failed for parent directory: %v", err)
			return nil, 0, fmt.Errorf("unable to create directory for export file: %v", err)
		}
	}

	if err := os.WriteFile(outputFile, jsonData, 0644); err != nil {
		output.GetVerbose().Error("%v", err)
		output.GetDebug().Error("os.WriteFile() failed: %v", err)
		return nil, 0, fmt.Errorf("unable to write export file: %v", err)
	}
	output.GetVerbose().Info("Export file written successfully")

	return exportedFonts, totalVariants, nil
}

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.Flags().StringP("output", "o", "", "Output file path (default: fonts-export.json)")
	exportCmd.Flags().StringP("match", "m", "", "Export fonts that match the specified string")
	exportCmd.Flags().StringP("source", "s", "", "Filter by font source (e.g., 'Google Fonts')")
	exportCmd.Flags().BoolP("all", "a", false, "Export all installed fonts (including those without Font IDs)")
	exportCmd.Flags().Bool("matched", false, "Export only fonts that match repository entries (default, cannot be used with --all)")
}
