package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"fontget/internal/cmdutils"
	"fontget/internal/output"
	"fontget/internal/platform"
	"fontget/internal/repo"
	"fontget/internal/shared"
	"fontget/internal/ui"

	"github.com/spf13/cobra"
)

// ParsedFont represents a parsed font file with metadata
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

var listCmd = &cobra.Command{
	Use:          "list [query]",
	Short:        "List installed fonts",
	SilenceUsage: true,
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

		if err := cmdutils.EnsureManifestInitialized(func() cmdutils.Logger { return GetLogger() }); err != nil {
			return err
		}

		fm, err := cmdutils.CreateFontManager(func() cmdutils.Logger { return GetLogger() })
		if err != nil {
			return err
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
		// Verbose section ends with blank line per spacing framework (only if verbose was shown)
		if IsVerbose() {
			fmt.Println()
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
				if err := cmdutils.CheckElevation(cmd, fm, installScope); err != nil {
					if errors.Is(err, cmdutils.ErrElevationRequired) {
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
		matches, err := repo.MatchAllInstalledFonts(allFamilyNames, shared.IsCriticalSystemFont)
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
			// Verbose section ends with blank line per spacing framework (only if verbose was shown)
			if IsVerbose() {
				fmt.Println()
			}
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
		// Apply filter using helper function
		filteredFamilies := filterFontsByFamilyAndID(families, familyFilter)

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
				fmt.Printf("\n%s\n\n", ui.Text.Render("Found 0 font families installed"))
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
			fmt.Printf("%s\n\n", ui.Text.Render(info))
		}

		fmt.Println(ui.GetListTableHeader())
		fmt.Println(ui.GetTableSeparator())

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
				ui.TableSourceName.Render(fmt.Sprintf("%-*s", ui.TableColListName, shared.TruncateString(fam, ui.TableColListName))),
				ui.TableColListID, shared.TruncateString(fontID, ui.TableColListID),
				ui.TableColListLicense, shared.TruncateString(license, ui.TableColListLicense),
				ui.TableColListCategory, shared.TruncateString(categories, ui.TableColListCategory),
				ui.TableColType, rep.Type,
				ui.TableColScope, rep.Scope,
				ui.TableColListSource, shared.TruncateString(source, ui.TableColListSource),
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
						fmt.Sprintf("%-*s", ui.TableColListName, row),
						ui.TableColListID, "",
						ui.TableColListLicense, "",
						ui.TableColListCategory, "",
						ui.TableColType, "",
						ui.TableColScope, "",
						ui.TableColListSource, "",
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

// collectFonts collects font files from the specified scopes
func collectFonts(scopes []platform.InstallationScope, fm platform.FontManager, typeFilter string, suppressVerbose ...bool) ([]ParsedFont, error) {
	var parsed []ParsedFont
	// Normalize type filter for comparison (uppercase)
	typeFilterUpper := ""
	if typeFilter != "" {
		typeFilterUpper = strings.ToUpper(typeFilter)
	}

	// Check if verbose output should be suppressed (default: false, show verbose)
	shouldSuppressVerbose := false
	if len(suppressVerbose) > 0 {
		shouldSuppressVerbose = suppressVerbose[0]
	}

	for _, scope := range scopes {
		fontDir := fm.GetFontDir(scope)
		if !shouldSuppressVerbose {
			output.GetVerbose().Info("Scanning %s scope: %s", scope, fontDir)
		}
		names, err := platform.ListInstalledFonts(fontDir)
		if err != nil {
			return nil, err
		}
		if !shouldSuppressVerbose {
			output.GetVerbose().Info("Found %d files in %s", len(names), fontDir)
		}
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
	if !shouldSuppressVerbose {
		output.GetVerbose().Info("Scan complete: parsed %d files across %d scope(s)", len(parsed), len(scopes))
		// Verbose section ends with blank line per spacing framework (only if verbose was shown)
		if IsVerbose() {
			fmt.Println()
		}
	}
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

// groupByFamily groups fonts by family name
func groupByFamily(fonts []ParsedFont) map[string][]ParsedFont {
	res := make(map[string][]ParsedFont)
	for _, f := range fonts {
		res[f.Family] = append(res[f.Family], f)
	}
	return res
}

// filterFontsByFamilyAndID filters font families by family name or Font ID
func filterFontsByFamilyAndID(families map[string][]ParsedFont, familyFilter string) map[string][]ParsedFont {
	// Optimization: Cache lowercased strings to avoid repeated ToLower() calls
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
			// Cache lowercased strings (optimization)
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

		filteredFamilies[familyName] = fontGroup
	}

	return filteredFamilies
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringP("scope", "s", "", "Filter by installation scope (user or machine). Default: show all scopes")
	listCmd.Flags().StringP("type", "t", "", "Filter by font type (TTF, OTF, etc.)")
	listCmd.Flags().BoolP("expand", "x", false, "Show font styles in hierarchical view")
}
