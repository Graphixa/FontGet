package cmd

import (
	"errors"
	"fmt"
	"fontget/internal/config"
	"fontget/internal/output"
	"fontget/internal/platform"
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
				output.GetDebug().State("Parsed: %s -> family='%s' style='%s'", name, family, style)
			} else {
				// Fallback to filename parsing (minimal)
				base := strings.TrimSuffix(name, filepath.Ext(name))
				family = base
				style = "Regular"
				output.GetDebug().State("Fallback parse: %s -> family='%s' style='%s' (%v)", name, family, style, err)
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
	output.GetVerbose().Info("Total parsed fonts: %d", len(parsed))
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

You can specify the installation scope using the --scope flag:
  - user (default): Show fonts installed for current user
  - machine: Show fonts installed system-wide
  - all: Show fonts from both user and machine scopes

`,
	Example: `  fontget list
  fontget list "jet"
  fontget list roboto -t ttf
  fontget list "fira" -f
  fontget list -s all`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Query is optional - no validation needed
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		output.GetDebug().Message("List command start")
		if err := config.EnsureManifestExists(); err != nil {
			return fmt.Errorf("failed to initialize sources: %v", err)
		}

		fm, err := platform.NewFontManager()
		if err != nil {
			return err
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
		if scope == "all" {
			scopes = []platform.InstallationScope{platform.UserScope, platform.MachineScope}
		} else {
			installScope := platform.UserScope
			if scope != "" && scope != "user" {
				installScope = platform.InstallationScope(scope)
				if installScope != platform.UserScope && installScope != platform.MachineScope {
					return fmt.Errorf("invalid scope '%s'", scope)
				}
			}
			// machine scope requires elevation
			if installScope == platform.MachineScope {
				if err := checkElevation(cmd, fm, installScope); err != nil {
					if errors.Is(err, ErrElevationRequired) {
						return nil
					}
					return err
				}
			}
			scopes = []platform.InstallationScope{installScope}
		}
		output.GetDebug().State("Scopes=%v family='%s' type='%s' showVariants=%v", scopes, familyFilter, typeFilter, showVariants)

		fonts, err := collectFonts(scopes, fm)
		if err != nil {
			return err
		}
		output.GetVerbose().Info("Collected %d fonts before filter", len(fonts))

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
		output.GetVerbose().Info("Filtered count=%d", len(filtered))
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
		output.GetVerbose().Info("Family count=%d", len(names))

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
			output.GetDebug().State("Family '%s' files=%d", fam, len(group))
			sort.Slice(group, func(i, j int) bool { return group[i].Style < group[j].Style })
			rep := group[0]
			fmt.Printf("%s %-*s %-*s %-*s %-*s\n",
				ui.TableSourceName.Render(fmt.Sprintf("%-*s", TableColListName, truncateString(fam, TableColListName))),
				TableColListID, "",
				TableColType, rep.Type,
				TableColDate, rep.InstallDate.Format("2006-01-02 15:04"),
				TableColScope, rep.Scope,
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
				output.GetDebug().State("Family '%s' uniqueStyles=%d", fam, len(styles))
				for _, s := range styles {
					row := fmt.Sprintf("  ↳ %s", s)
					fmt.Printf("%s %-*s %-*s %-*s %-*s\n",
						fmt.Sprintf("%-*s", TableColListName, row),
						TableColListID, "",
						TableColType, "",
						TableColDate, "",
						TableColScope, "",
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
	listCmd.Flags().StringP("scope", "s", "", "Installation scope (user, machine, or all)")
	listCmd.Flags().StringP("type", "t", "", "Filter by font type (TTF, OTF, etc.)")
	listCmd.Flags().BoolP("full", "f", false, "Show font styles in hierarchical view")
}
