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

func collectFonts(scopes []platform.InstallationScope, fm platform.FontManager) ([]ParsedFont, error) {
	var parsed []ParsedFont
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
			md, err := platform.ExtractFontMetadata(p)
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
				// Debug: per-file parsed details removed for cleaner output
			} else {
				// Fallback to filename parsing (minimal)
				base := strings.TrimSuffix(name, filepath.Ext(name))
				family = base
				style = "Regular"
				// Debug: per-file fallback details removed for cleaner output
			}
			parsed = append(parsed, ParsedFont{
				Name:        name,
				Family:      family,
				Style:       style,
				Type:        strings.ToUpper(strings.TrimPrefix(filepath.Ext(name), ".")),
				InstallDate: info.ModTime(),
				Scope:       string(scope),
			})
		}
	}
	output.GetVerbose().Info("Scan complete: parsed %d files across %d scope(s)", len(parsed), len(scopes))
	return parsed, nil
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

You can filter the results by providing an optional query string to filter font family names.

By default, fonts from both user and machine scopes are shown.

You can filter to a specific scope using the --scope flag:
  - user: Show fonts installed for current user only
  - machine: Show fonts installed system-wide only

`,
	Example: `  fontget list
  fontget list "jet"
  fontget list roboto -t ttf
  fontget list "fira" -f
  fontget list -s user`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Query is optional - no validation needed
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		output.GetDebug().Message("List command start")
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

		scope, _ := cmd.Flags().GetString("scope")
		typeFilter, _ := cmd.Flags().GetString("type")
		showVariants, _ := cmd.Flags().GetBool("full")

		// Get query from positional argument
		var familyFilter string
		if len(args) > 0 {
			familyFilter = args[0]
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
					output.GetVerbose().Error("%v", err)
					output.GetDebug().Error("checkElevation() failed: %v", err)
					return fmt.Errorf("unable to verify system permissions: %v", err)
				}
			}
			scopes = []platform.InstallationScope{installScope}
		}
		// Debug: initial parameter dump removed to reduce noise

		fonts, err := collectFonts(scopes, fm)
		if err != nil {
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("collectFonts() failed: %v", err)
			return fmt.Errorf("unable to read installed fonts: %v", err)
		}
		output.GetDebug().State("Collected %d font files before filtering", len(fonts))

		// Apply filters
		filtered := make([]ParsedFont, 0, len(fonts))
		for _, f := range fonts {
			if familyFilter != "" && !strings.Contains(strings.ToLower(f.Family), strings.ToLower(familyFilter)) {
				continue
			}
			if typeFilter != "" && !strings.EqualFold(f.Type, typeFilter) {
				continue
			}
			filtered = append(filtered, f)
		}
		output.GetDebug().State("After filtering: %d font files remaining", len(filtered))
		if len(filtered) == 0 {
			fmt.Printf("\n%s\n", ui.PageTitle.Render("Installed Fonts"))

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

		families := groupByFamily(filtered)
		var names []string
		for k := range families {
			names = append(names, k)
		}
		sort.Strings(names)
		output.GetDebug().State("Grouped %d font files into %d unique families", len(filtered), len(names))

		// Match installed fonts to repository
		output.GetVerbose().Info("Matching installed fonts to repository...")
		output.GetDebug().State("Matching %d font families against repository", len(names))
		matches, err := repo.MatchAllInstalledFonts(names, IsCriticalSystemFont)
		if err != nil {
			output.GetDebug().Error("Font matching failed: %v", err)
			// Continue without matches (fonts will show blank fields)
			matches = make(map[string]*repo.InstalledFontMatch)
		} else {
			matchCount := 0
			for _, match := range matches {
				if match != nil {
					matchCount++
				}
			}
			output.GetVerbose().Info("Found %d matches out of %d installed fonts", matchCount, len(names))
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

		// Header
		fmt.Printf("\n%s\n", ui.PageTitle.Render("Installed Fonts"))

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
			group := families[fam]
			output.GetDebug().State("Family '%s': %d files", fam, len(group))
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
				output.GetDebug().State("Family '%s': %d unique variants", fam, len(styles))
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
	listCmd.Flags().BoolP("full", "f", false, "Show font styles in hierarchical view")
}
