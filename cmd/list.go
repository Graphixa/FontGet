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
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// FontFile represents an installed font file
type FontFile struct {
	Name        string
	Family      string
	Style       string
	Type        string
	InstallDate time.Time
	Scope       string
}

// parseFontName extracts family and style from a font filename
func parseFontName(filename string) (family, style string) {
	// Remove file extension
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Remove variation parameters
	if idx := strings.Index(name, "["); idx != -1 {
		name = name[:idx]
	}

	// Remove "webfont" suffix if present
	name = strings.TrimSuffix(name, "-webfont")

	// For fonts with spaces in their names
	if strings.Contains(name, " ") {
		// Extract the base family name
		parts := strings.Split(name, " ")
		if len(parts) > 0 {
			family = parts[0]
			// The rest is the style
			if len(parts) > 1 {
				style = strings.Join(parts[1:], " ")
			} else {
				style = "Regular"
			}
			return family, style
		}
	}

	// For other fonts, split by hyphens
	parts := strings.Split(name, "-")
	if len(parts) == 1 {
		return parts[0], "Regular"
	}

	// If we have multiple parts, assume the last part is the style
	// and everything else is the family name
	family = strings.Join(parts[:len(parts)-1], "-")
	style = cases.Title(language.English, cases.NoLower).String(parts[len(parts)-1])
	return family, style
}

// listFonts lists fonts in the specified directory and scope
func listFonts(fontDir string, installScope platform.InstallationScope) ([]FontFile, error) {
	GetLogger().Debug("Listing fonts in directory: %s (scope: %s)", fontDir, installScope)

	// List all font files in the directory
	files, err := os.ReadDir(fontDir)
	if err != nil {
		GetLogger().Error("Failed to read font directory %s: %v", fontDir, err)
		return nil, fmt.Errorf("failed to read font directory: %w", err)
	}

	var fontFiles []FontFile
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Get file info
		fileInfo, err := file.Info()
		if err != nil {
			GetLogger().Warn("Failed to get file info for %s: %v", file.Name(), err)
			continue
		}

		// Parse font name
		family, style := parseFontName(file.Name())

		// Create font file entry
		fontFiles = append(fontFiles, FontFile{
			Name:        file.Name(),
			Family:      family,
			Style:       style,
			Type:        strings.ToUpper(strings.TrimPrefix(filepath.Ext(file.Name()), ".")),
			InstallDate: fileInfo.ModTime(),
			Scope:       string(installScope),
		})
	}

	GetLogger().Debug("Found %d fonts in %s", len(fontFiles), fontDir)
	return fontFiles, nil
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed fonts",
	Long:  "List installed fonts on your system with filtering options.",
	Example: `  fontget list
  fontget list -s machine
  fontget list -s all
  fontget list -a "Roboto"
  fontget list -t TTF
  fontget list -s all -t TTF`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		GetLogger().Info("Starting font list operation")

		// Debug-level information for developers
		output.GetDebug().Message("Debug mode enabled - showing detailed diagnostic information")

		// Ensure manifest system is initialized (fixes missing sources.json bug)
		if err := config.EnsureManifestExists(); err != nil {
			return fmt.Errorf("failed to initialize sources: %v", err)
		}

		// Create font manager
		fontManager, err := platform.NewFontManager()
		if err != nil {
			GetLogger().Error("Failed to initialize font manager: %v", err)
			output.GetVerbose().Error("Failed to initialize font manager: %v", err)
			output.GetDebug().Error("Font manager initialization failed: %v", err)
			return fmt.Errorf("failed to initialize font manager: %w", err)
		}

		// Get flags
		scope, _ := cmd.Flags().GetString("scope")
		family, _ := cmd.Flags().GetString("family")
		fontType, _ := cmd.Flags().GetString("type")

		GetLogger().Info("List command parameters - Scope: %s, Family: %s, Type: %s", scope, family, fontType)

		// Verbose-level information for users
		output.GetVerbose().Info("List command parameters - Scope: %s, Family: %s, Type: %s", scope, family, fontType)
		output.GetDebug().State("Starting font listing with parameters: scope='%s', family='%s', type='%s'", scope, family, fontType)

		// Auto-detect scope if not explicitly provided
		if scope == "" {
			isElevated, err := fontManager.IsElevated()
			if err != nil {
				GetLogger().Warn("Failed to detect elevation status: %v", err)
				output.GetVerbose().Warning("Failed to detect elevation status: %v", err)
				output.GetDebug().State("Elevation detection failed: %v", err)
				// Default to user scope if we can't detect elevation
				scope = "user"
			} else if isElevated {
				scope = "all"
				GetLogger().Info("Auto-detected elevated privileges, defaulting to 'all' scope")
				output.GetVerbose().Info("Auto-detected elevated privileges, defaulting to 'all' scope")
				output.GetDebug().State("Elevation detected, using 'all' scope")
				fmt.Println(ui.FormLabel.Render("Auto-detected administrator privileges - listing from all scopes"))
			} else {
				scope = "user"
				GetLogger().Info("Auto-detected user privileges, defaulting to 'user' scope")
				output.GetVerbose().Info("Auto-detected user privileges, defaulting to 'user' scope")
				output.GetDebug().State("No elevation detected, using 'user' scope")
			}
		}

		// Convert scope string to InstallationScope
		var scopes []platform.InstallationScope
		if scope == "all" {
			scopes = []platform.InstallationScope{platform.UserScope, platform.MachineScope}
			GetLogger().Debug("Listing fonts from all scopes")
		} else {
			installScope := platform.UserScope
			if scope != "user" {
				installScope = platform.InstallationScope(scope)
				if installScope != platform.UserScope && installScope != platform.MachineScope {
					GetLogger().Error("Invalid scope '%s'", scope)
					return fmt.Errorf("invalid scope '%s'. Must be 'user', 'machine', or 'all'", scope)
				}
			}
			scopes = []platform.InstallationScope{installScope}
			GetLogger().Debug("Listing fonts from scope: %s", installScope)
		}

		// Collect fonts from all specified scopes
		var allFonts []FontFile
		output.GetVerbose().Info("Scanning %d scope(s) for installed fonts", len(scopes))
		for _, installScope := range scopes {
			output.GetVerbose().Info("Scanning %s scope for fonts", installScope)
			output.GetDebug().State("Processing scope: %s", installScope)

			// Check elevation for machine scope
			if installScope == platform.MachineScope {
				GetLogger().Debug("Checking elevation for machine scope")
				output.GetVerbose().Info("Checking elevation for machine scope access")
				output.GetDebug().State("Machine scope requires elevation check")
				if err := checkElevation(cmd, fontManager, installScope); err != nil {
					if errors.Is(err, ErrElevationRequired) {
						return nil // Already printed user-friendly message
					}
					GetLogger().Error("Elevation check failed: %v", err)
					output.GetVerbose().Error("Elevation check failed: %v", err)
					output.GetDebug().Error("Elevation check failed for machine scope: %v", err)
					return err
				}
			}

			// Get font directory for the specified scope
			fontDir := fontManager.GetFontDir(installScope)
			output.GetVerbose().Info("Scanning font directory: %s", fontDir)
			output.GetDebug().State("Font directory for %s scope: %s", installScope, fontDir)

			// List fonts in this directory
			fonts, err := listFonts(fontDir, installScope)
			if err != nil {
				output.GetVerbose().Error("Failed to scan fonts in %s: %v", fontDir, err)
				output.GetDebug().Error("Font scanning failed for %s: %v", fontDir, err)
				return err
			}
			output.GetVerbose().Info("Found %d fonts in %s scope", len(fonts), installScope)
			output.GetDebug().State("Scanned %d font files in %s scope", len(fonts), installScope)
			allFonts = append(allFonts, fonts...)
		}

		if len(allFonts) == 0 {
			GetLogger().Info("No fonts found in the specified scope(s)")
			fmt.Printf("No fonts found in the specified scope(s)\n")
			return nil
		}

		// Apply filters
		output.GetVerbose().Info("Applying filters - Family: '%s', Type: '%s'", family, fontType)
		output.GetDebug().State("Filtering %d fonts with family='%s', type='%s'", len(allFonts), family, fontType)
		var filteredFonts []FontFile
		for _, font := range allFonts {
			// Filter by family if specified
			if family != "" && !strings.EqualFold(font.Family, family) {
				output.GetDebug().State("Filtering out font %s (family '%s' != '%s')", font.Name, font.Family, family)
				continue
			}

			// Filter by type if specified
			if fontType != "" && !strings.EqualFold(font.Type, fontType) {
				output.GetDebug().State("Filtering out font %s (type '%s' != '%s')", font.Name, font.Type, fontType)
				continue
			}

			filteredFonts = append(filteredFonts, font)
		}

		output.GetVerbose().Info("Filtering completed - %d fonts match criteria", len(filteredFonts))
		output.GetDebug().State("Filtering result: %d fonts match all criteria", len(filteredFonts))

		if len(filteredFonts) == 0 {
			GetLogger().Info("No fonts found matching the specified filters")
			fmt.Printf("\n%s\n", ui.PageTitle.Render("Installed Fonts"))
			fmt.Printf("---------------------------------------------\n")
			fmt.Printf("%s\n", ui.FeedbackWarning.Render("No fonts found matching the specified filters"))
			return nil
		}

		// Group fonts by family
		families := make(map[string][]FontFile)
		for _, f := range filteredFonts {
			families[f.Family] = append(families[f.Family], f)
		}

		// Sort families alphabetically
		var familyNames []string
		for family := range families {
			familyNames = append(familyNames, family)
		}
		sort.Strings(familyNames)

		GetLogger().Info("Found %d font families", len(familyNames))

		// Define column widths
		columns := map[string]int{
			"Name":  45, // For display name
			"Style": 18, // For font style
			"Type":  10, // For file type
			"Date":  20, // For installation date
			"Scope": 10, // For installation scope
		}

		// Print header (match search command style)
		header := fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s",
			columns["Name"], "Name",
			columns["Style"], "Style",
			columns["Type"], "Type",
			columns["Date"], "Installed",
			columns["Scope"], "Scope")
		fmt.Println(ui.TableHeader.Render(header))
		fmt.Println(ui.FeedbackText.Render(strings.Repeat("-", len(header))))

		// Print each family
		for _, family := range familyNames {
			fonts := families[family]

			// Sort fonts by style
			sort.Slice(fonts, func(i, j int) bool {
				return fonts[i].Style < fonts[j].Style
			})

			// Print family header (match search command style)
			fmt.Printf("%-*s %-*s %-*s %-*s %-*s\n",
				columns["Name"], ui.TableSourceName.Render(family),
				columns["Style"], "",
				columns["Type"], "",
				columns["Date"], "",
				columns["Scope"], "")

			// Print each font in the family (match search command style)
			for _, font := range fonts {
				fmt.Printf(" - %-*s %-*s %-*s %-*s %-*s\n",
					columns["Name"]-3, font.Name,
					columns["Style"], font.Style,
					columns["Type"], font.Type,
					columns["Date"], font.InstallDate.Format("2006-01-02 15:04"),
					columns["Scope"], font.Scope)
			}

			// Add a blank line after each family
			fmt.Println("")
		}

		// Print summary
		fmt.Printf("\n%s\n", ui.ReportTitle.Render("Summary"))
		fmt.Printf("---------------------------------------------\n")
		fmt.Printf("%s: %d font(s) in %d family(ies)\n",
			ui.ContentHighlight.Render("Total"), len(filteredFonts), len(familyNames))
		if scope != "" {
			fmt.Printf("%s: %s\n", ui.ContentHighlight.Render("Scope"), scope)
		}
		fmt.Printf("\n")

		GetLogger().Info("Font list operation completed successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	// Add flags
	listCmd.Flags().StringP("scope", "s", "", "Installation scope (user, machine, or all)")
	listCmd.Flags().StringP("family", "a", "", "Filter by font family name")
	listCmd.Flags().StringP("type", "t", "", "Filter by font type (TTF, OTF, etc.)")
}
