package cmd

import (
	"fmt"
	"fontget/internal/platform"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
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

	// Remove variation parameters (e.g., [wght], [wdth,wght])
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
	style = strings.Title(parts[len(parts)-1])
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
	Long:  "Lists all installed fonts on your system, with options to filter by family, type, and installation scope.",
	Example: `  fontget list
  fontget list --scope machine
  fontget list -s all
  fontget list -f "Roboto"
  fontget list -t TTF
  fontget list -s all -t TTF`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		GetLogger().Info("Starting font list operation")

		// Create font manager
		fontManager, err := platform.NewFontManager()
		if err != nil {
			GetLogger().Error("Failed to initialize font manager: %v", err)
			return fmt.Errorf("failed to initialize font manager: %w", err)
		}

		// Get flags
		scope, _ := cmd.Flags().GetString("scope")
		family, _ := cmd.Flags().GetString("family")
		fontType, _ := cmd.Flags().GetString("type")

		GetLogger().Info("List command parameters - Scope: %s, Family: %s, Type: %s", scope, family, fontType)

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
		for _, installScope := range scopes {
			// Check elevation for machine scope
			if installScope == platform.MachineScope {
				GetLogger().Debug("Checking elevation for machine scope")
				if err := checkElevation(cmd, fontManager, installScope); err != nil {
					GetLogger().Error("Elevation check failed: %v", err)
					return err
				}
			}

			// Get font directory for the specified scope
			fontDir := fontManager.GetFontDir(installScope)

			// List fonts in this directory
			fonts, err := listFonts(fontDir, installScope)
			if err != nil {
				return err
			}
			allFonts = append(allFonts, fonts...)
		}

		if len(allFonts) == 0 {
			GetLogger().Info("No fonts found in the specified scope(s)")
			fmt.Printf("No fonts found in the specified scope(s)\n")
			return nil
		}

		// Apply filters
		var filteredFonts []FontFile
		for _, font := range allFonts {
			// Filter by family if specified
			if family != "" && !strings.EqualFold(font.Family, family) {
				continue
			}

			// Filter by type if specified
			if fontType != "" && !strings.EqualFold(font.Type, fontType) {
				continue
			}

			filteredFonts = append(filteredFonts, font)
		}

		if len(filteredFonts) == 0 {
			GetLogger().Info("No fonts found matching the specified filters")
			fmt.Printf("No fonts found matching the specified filters\n")
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

		// Print header
		fmt.Printf("\nInstalled fonts:\n\n")

		// Define column widths
		columns := map[string]int{
			"Name":  45, // For display name
			"Style": 18, // For font style
			"Type":  10, // For file type
			"Date":  20, // For installation date
			"Scope": 10, // For installation scope
		}

		// Print header
		header := fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s",
			columns["Name"], "Name",
			columns["Style"], "Style",
			columns["Type"], "Type",
			columns["Date"], "Installed",
			columns["Scope"], "Scope")
		fmt.Println(header)
		fmt.Println(strings.Repeat("-", len(header)))

		// Print each family
		for _, family := range familyNames {
			fonts := families[family]

			// Sort fonts by style
			sort.Slice(fonts, func(i, j int) bool {
				return fonts[i].Style < fonts[j].Style
			})

			// Print family header
			fmt.Printf("%-*s %-*s %-*s %-*s %-*s\n",
				columns["Name"], "Font Family: "+family,
				columns["Style"], "",
				columns["Type"], "",
				columns["Date"], "",
				columns["Scope"], "")

			// Print each font in the family
			for _, font := range fonts {
				// Format the line with bullet point
				fmt.Printf(" - %-*s %-*s %-*s %-*s %-*s\n",
					columns["Name"]-3, font.Name,
					columns["Style"], font.Style,
					columns["Type"], font.Type,
					columns["Date"], font.InstallDate.Format("2006-01-02 15:04"),
					columns["Scope"], font.Scope)
			}
		}

		GetLogger().Info("Font list operation completed successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	// Add flags
	listCmd.Flags().StringP("scope", "s", "user", "Installation scope (user, machine, or all)")
	listCmd.Flags().StringP("family", "f", "", "Filter by font family name")
	listCmd.Flags().StringP("type", "t", "", "Filter by font type (TTF, OTF, etc.)")
}
