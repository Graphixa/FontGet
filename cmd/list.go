package cmd

import (
	"errors"
	"fmt"
	"fontget/internal/config"
	"fontget/internal/output"
	"fontget/internal/platform"
	"fontget/internal/repo"
	"fontget/internal/ui"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type ParsedFont struct {
	Name        string
	Family      string
	Style       string
	Type        string
	InstallDate time.Time
	Scope       string
	// Repository match fields
	FontID     string
	License    string
	Categories []string
	Source     string
}

func collectFonts(scopes []platform.InstallationScope, fm platform.FontManager, typeFilter string) ([]ParsedFont, error) {
	var parsed []ParsedFont
	// Normalize type filter for comparison (uppercase)
	typeFilterUpper := ""
	if typeFilter != "" {
		typeFilterUpper = strings.ToUpper(typeFilter)
	}

	for _, scope := range scopes {
		fontDir := fm.GetFontDir(scope)
		output.GetVerbose().Info("Scanning %s scope: %s", scope, fontDir)
		names, err := platform.ListInstalledFonts(fontDir)
		if err != nil {
			return nil, err
		}
		output.GetVerbose().Info("Found %d files in %s", len(names), fontDir)
		for _, name := range names {
			p := filepath.Join(fontDir, name)
			info, err := os.Stat(p)
			if err != nil {
				continue
			}

			// Optimization 1: Early type filtering - check extension before expensive metadata extraction
			fileExt := strings.ToUpper(strings.TrimPrefix(filepath.Ext(name), "."))
			if typeFilterUpper != "" && fileExt != typeFilterUpper {
				// Skip this file if it doesn't match the type filter
				continue
			}

			// Build ParsedFont struct using extracted function
			parsed = append(parsed, buildParsedFont(p, name, scope, info))
		}
	}
	output.GetVerbose().Info("Scan complete: parsed %d files across %d scope(s)", len(parsed), len(scopes))
	return parsed, nil
}

// buildParsedFont extracts font metadata from a file path and builds a ParsedFont struct
func buildParsedFont(fontPath, fileName string, scope platform.InstallationScope, fileInfo os.FileInfo) ParsedFont {
	// Extract file extension for type
	fileExt := strings.ToUpper(strings.TrimPrefix(filepath.Ext(fileName), "."))

	// Try to extract metadata from the font file
	md, err := platform.ExtractFontMetadata(fontPath)
	family := ""
	style := ""

	if err == nil {
		// Prefer typographic names for display when present
		if md.TypographicFamily != "" {
			family = md.TypographicFamily
		} else {
			family = md.FamilyName
		}
		if md.TypographicStyle != "" {
			style = md.TypographicStyle
		} else {
			style = md.StyleName
		}
	} else {
		// Fallback to filename parsing (minimal)
		base := strings.TrimSuffix(fileName, filepath.Ext(fileName))
		family = base
		style = "Regular"
	}

	return ParsedFont{
		Name:        fileName,
		Family:      family,
		Style:       style,
		Type:        fileExt,
		InstallDate: fileInfo.ModTime(),
		Scope:       string(scope),
	}
}

func groupByFamily(fonts []ParsedFont) map[string][]ParsedFont {
	res := make(map[string][]ParsedFont)
	for _, f := range fonts {
		res[f.Family] = append(res[f.Family], f)
	}
	return res
}

var listCmd = &cobra.Command{
	Use:   "list [query]",
	Short: "List installed fonts",
	Long: `List fonts installed on your system.

By default, shows fonts from both user and system-wide installations.
Results can be filtered by font family name, Font ID, type, or scope.

The query parameter can match either font family names (e.g., "Roboto") or Font IDs (e.g., "google.roboto").

Flags:
  --scope, -s    Filter by installation scope (user, machine)
  --type, -t     Filter by font type (TTF, OTF, etc.)
  --expand, -x   Show all font variants in hierarchical view

`,
	Example: `  fontget list
  fontget list "jet"
  fontget list roboto -t ttf
  fontget list "google.roboto"
  fontget list "fira" -x
  fontget list -s user`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Query is optional - no validation needed
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		GetLogger().Info("Starting font list operation")

		if err := config.EnsureManifestExists(); err != nil {
			GetLogger().Error("Failed to ensure manifest exists: %v", err)
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("config.EnsureManifestExists() failed: %v", err)
			return fmt.Errorf("unable to load font repository: %v", err)
		}

		fm, err := platform.NewFontManager()
		if err != nil {
			GetLogger().Error("Failed to create font manager: %v", err)
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("platform.NewFontManager() failed: %v", err)
			return fmt.Errorf("unable to access system fonts: %v", err)
		}

		scope, _ := cmd.Flags().GetString("scope")
		typeFilter, _ := cmd.Flags().GetString("type")
		showVariants, _ := cmd.Flags().GetBool("expand")

		// Get query from positional argument
		var familyFilter string
		if len(args) > 0 {
			familyFilter = args[0]
		}

		// Log parameters (always log to file)
		scopeDisplay := "all"
		if scope != "" {
			scopeDisplay = scope
		}
		GetLogger().Info("List parameters - Scope: %s, Type filter: %s, Family filter: %s, Show variants: %v", scopeDisplay, typeFilter, familyFilter, showVariants)

		// Verbose output for parameters
		output.GetVerbose().Info("Scope: %s", scopeDisplay)
		if typeFilter != "" {
			output.GetVerbose().Info("Type filter: %s", typeFilter)
		}
		if familyFilter != "" {
			output.GetVerbose().Info("Family filter: %s", familyFilter)
		}

		var scopes []platform.InstallationScope
		// Default to "all" (both scopes) if no scope specified
		if scope == "" {
			scopes = []platform.InstallationScope{platform.UserScope, platform.MachineScope}
		} else {
			// Validate scope - only "user" or "machine" are valid
			installScope := platform.InstallationScope(scope)
			if installScope != platform.UserScope && installScope != platform.MachineScope {
				err := fmt.Errorf("invalid scope '%s'. Valid options are: user, machine", scope)
				GetLogger().Error("Invalid scope provided: %s", scope)
				output.GetVerbose().Error("%v", err)
				output.GetDebug().Error("Invalid scope provided: '%s'", scope)
				return err
			}
			// machine scope requires elevation
			if installScope == platform.MachineScope {
				if err := checkElevation(cmd, fm, installScope); err != nil {
					if errors.Is(err, ErrElevationRequired) {
						return nil
					}
					GetLogger().Error("Failed to check elevation: %v", err)
					output.GetVerbose().Error("%v", err)
					output.GetDebug().Error("checkElevation() failed: %v", err)
					return fmt.Errorf("unable to verify system permissions: %v", err)
				}
			}
			scopes = []platform.InstallationScope{installScope}
		}
		// Debug: initial parameter dump removed to reduce noise

		fonts, err := collectFonts(scopes, fm, typeFilter)
		if err != nil {
			GetLogger().Error("Failed to collect fonts: %v", err)
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("collectFonts() failed: %v", err)
			return fmt.Errorf("unable to read installed fonts: %v", err)
		}
		output.GetDebug().State("Collected %d font files before filtering", len(fonts))

		// Group fonts by family first (before matching to repository)
		families := groupByFamily(fonts)
		var allFamilyNames []string
		for k := range families {
			allFamilyNames = append(allFamilyNames, k)
		}
		sort.Strings(allFamilyNames)
		output.GetDebug().State("Grouped %d font files into %d unique families", len(fonts), len(allFamilyNames))

		// Match installed fonts to repository BEFORE filtering (so Font IDs are available)
		output.GetVerbose().Info("Matching installed fonts to repository...")
		output.GetDebug().State("Matching %d font families against repository", len(allFamilyNames))
		matches, err := repo.MatchAllInstalledFonts(allFamilyNames, IsCriticalSystemFont)
		if err != nil {
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("repo.MatchAllInstalledFonts() failed: %v", err)
			// Continue without matches (fonts will show blank fields)
			matches = make(map[string]*repo.InstalledFontMatch)
		} else {
			matchCount := 0
			for _, match := range matches {
				if match != nil {
					matchCount++
				}
			}
			output.GetVerbose().Info("Found %d matches out of %d installed fonts", matchCount, len(allFamilyNames))
		}

		// Populate match data into ParsedFont structs
		for familyName, fontGroup := range families {
			if match, exists := matches[familyName]; exists && match != nil {
				// Update all fonts in this family group with match data
				for i := range fontGroup {
					fontGroup[i].FontID = match.FontID
					fontGroup[i].License = match.License
					fontGroup[i].Categories = match.Categories
					fontGroup[i].Source = match.Source
				}
				families[familyName] = fontGroup
			}
		}

		// Apply filters (now that Font IDs are populated)
		// Filter by family name and Font ID (type filter already applied during collection)
		// Optimization 2: Cache lowercased strings to avoid repeated ToLower() calls
		filteredFamilies := make(map[string][]ParsedFont)
		familyFilterLower := ""
		if familyFilter != "" {
			familyFilterLower = strings.ToLower(familyFilter)
		}

		for familyName, fontGroup := range families {
			// Get Font ID for this family (from first font in group, all have same Font ID)
			fontID := ""
			if len(fontGroup) > 0 {
				fontID = fontGroup[0].FontID
			}

			// Check if family name or Font ID matches the filter
			matchesFilter := true
			if familyFilterLower != "" {
				// Cache lowercased strings (optimization 2)
				familyLower := strings.ToLower(familyName)
				fontIDLower := ""
				if fontID != "" {
					fontIDLower = strings.ToLower(fontID)
				}

				// Check if filter matches family name OR Font ID
				matchesFilter = strings.Contains(familyLower, familyFilterLower) ||
					(fontIDLower != "" && strings.Contains(fontIDLower, familyFilterLower))
			}

			if !matchesFilter {
				continue
			}

			// Type filter already applied during collection, so include all fonts in family
			filteredFamilies[familyName] = fontGroup

			// Show which fonts matched the filter (useful for debugging)
			if familyFilterLower != "" {
				matchReason := "family name"
				if fontID != "" && strings.Contains(strings.ToLower(fontID), familyFilterLower) {
					matchReason = "Font ID"
				}
				output.GetDebug().State("Filter match: '%s' (matched by %s)", familyName, matchReason)
			}
		}

		// Get sorted list of filtered family names
		var names []string
		for k := range filteredFamilies {
			names = append(names, k)
		}
		sort.Strings(names)
		output.GetDebug().State("After filtering: %d font families remaining", len(names))

		if len(names) == 0 {

			// Show filter info in same format as successful results, just with 0 count
			if familyFilter != "" || typeFilter != "" {
				filterInfo := fmt.Sprintf("Found 0 font families installed matching '%s'", ui.TableSourceName.Render(familyFilter))
				if typeFilter != "" {
					filterInfo += fmt.Sprintf(" | Filtered by type: '%s'", ui.TableSourceName.Render(typeFilter))
				}
				fmt.Printf("\n%s\n\n", filterInfo)
			} else {
				fmt.Printf("\n%s\n\n", ui.FeedbackText.Render("Found 0 font families installed"))
			}
			return nil
		}

		// Log completion
		GetLogger().Info("List operation complete - Found %d font families", len(names))

		// Show filter info if filtering is applied (count shows families, not individual files)
		if familyFilter != "" || typeFilter != "" {
			filterInfo := fmt.Sprintf("Found %d font families installed matching '%s'", len(names), ui.TableSourceName.Render(familyFilter))
			if typeFilter != "" {
				filterInfo += fmt.Sprintf(" | Filtered by type: '%s'", ui.TableSourceName.Render(typeFilter))
			}
			fmt.Printf("\n%s\n\n", filterInfo)
		} else {
			info := fmt.Sprintf("Found %d font families installed", len(names))
			fmt.Printf("%s\n\n", ui.FeedbackText.Render(info))
		}

		fmt.Println(ui.TableHeader.Render(GetListTableHeader()))
		fmt.Println(GetTableSeparator())

		for i, fam := range names {
			group := filteredFamilies[fam]
			sort.Slice(group, func(i, j int) bool { return group[i].Style < group[j].Style })
			rep := group[0]

			// Format Font ID
			fontID := rep.FontID

			// Format License
			license := rep.License

			// Format Categories (first category only, like search command)
			categories := ""
			if len(rep.Categories) > 0 {
				categories = rep.Categories[0]
			}

			// Format Source
			source := rep.Source

			fmt.Printf("%s %-*s %-*s %-*s %-*s %-*s %-*s\n",
				ui.TableSourceName.Render(fmt.Sprintf("%-*s", TableColListName, truncateString(fam, TableColListName))),
				TableColListID, truncateString(fontID, TableColListID),
				TableColListLicense, truncateString(license, TableColListLicense),
				TableColListCategory, truncateString(categories, TableColListCategory),
				TableColType, rep.Type,
				TableColScope, rep.Scope,
				TableColListSource, truncateString(source, TableColListSource),
			)

			if showVariants {
				uniq := map[string]bool{}
				var styles []string
				for _, f := range group {
					if !uniq[f.Style] {
						uniq[f.Style] = true
						styles = append(styles, f.Style)
					}
				}
				sort.Strings(styles)
				for _, s := range styles {
					row := fmt.Sprintf("  ↳ %s", s)
					fmt.Printf("%s %-*s %-*s %-*s %-*s %-*s %-*s\n",
						fmt.Sprintf("%-*s", TableColListName, row),
						TableColListID, "",
						TableColListLicense, "",
						TableColListCategory, "",
						TableColType, "",
						TableColScope, "",
						TableColListSource, "",
					)
				}
				if i < len(names)-1 {
					fmt.Println()
				}
			}
		}

		fmt.Println()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringP("scope", "s", "", "Filter by installation scope (user or machine). Default: show all scopes")
	listCmd.Flags().StringP("type", "t", "", "Filter by font type (TTF, OTF, etc.)")
	listCmd.Flags().BoolP("expand", "x", false, "Show font styles in hierarchical view")
}
