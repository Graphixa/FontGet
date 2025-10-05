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

// groupFontsByFamily analyzes font names to find common base families
func groupFontsByFamily(fontMetadata map[string]*platform.FontMetadata) map[string][]string {
	groups := make(map[string][]string)

	// Collect all unique family names
	familyNames := make([]string, 0)
	for _, metadata := range fontMetadata {
		familyNames = append(familyNames, metadata.FamilyName)
	}

	// Group similar family names together
	for _, familyName := range familyNames {
		baseFamily := findBaseFamilyName(familyName, familyNames)
		groups[baseFamily] = append(groups[baseFamily], familyName)
	}

	return groups
}

// findBaseFamilyName finds the base family name by analyzing common prefixes
func findBaseFamilyName(familyName string, allFamilyNames []string) string {
	// Split the family name into words
	words := strings.Split(familyName, " ")
	if len(words) <= 1 {
		return familyName
	}

	// Find the longest common prefix with other family names
	bestMatch := familyName
	maxCommonWords := 0

	for _, otherFamily := range allFamilyNames {
		if otherFamily == familyName {
			continue
		}

		otherWords := strings.Split(otherFamily, " ")
		commonWords := findCommonPrefix(words, otherWords)

		if len(commonWords) > maxCommonWords && len(commonWords) > 0 {
			maxCommonWords = len(commonWords)
			bestMatch = strings.Join(commonWords, " ")
		}
	}

	return bestMatch
}

// findCommonPrefix finds the common prefix between two word arrays
func findCommonPrefix(words1, words2 []string) []string {
	var common []string
	minLen := len(words1)
	if len(words2) < minLen {
		minLen = len(words2)
	}

	for i := 0; i < minLen; i++ {
		if words1[i] == words2[i] {
			common = append(common, words1[i])
		} else {
			break
		}
	}

	return common
}

// findBestGroup finds the best group for a font family name
func findBestGroup(familyName string, groups map[string][]string) string {
	// Look for exact match first
	for groupName, familyNames := range groups {
		for _, name := range familyNames {
			if name == familyName {
				return groupName
			}
		}
	}

	// If no exact match, return the original name
	return familyName
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

		// For list command, use filename parsing to show individual font files
		// This matches the old behavior where each font file is shown separately
		family, style := parseFontName(file.Name())

		// Create font file entry using filename parsing
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
	Long:  "List installed fonts on your system with filtering options and display modes.",
	Example: `  fontget list
  fontget list --detailed
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

		// Group fonts by analyzing family names dynamically (like Windows/macOS font manager)
		families := make(map[string][]FontFile)

		// First pass: collect all font metadata
		fontMetadata := make(map[string]*platform.FontMetadata)
		for _, f := range filteredFonts {
			fontPath := filepath.Join(fontManager.GetFontDir(platform.InstallationScope(f.Scope)), f.Name)
			metadata, err := platform.ExtractFontMetadata(fontPath)
			if err != nil {
				GetLogger().Debug("Failed to extract metadata for %s, using filename parsing: %v", f.Name, err)
				families[f.Family] = append(families[f.Family], f)
			} else {
				fontMetadata[f.Name] = metadata
			}
		}

		// Second pass: group fonts by analyzing family name patterns
		groupedFamilies := groupFontsByFamily(fontMetadata)

		// Third pass: assign fonts to their groups
		for _, f := range filteredFonts {
			if metadata, exists := fontMetadata[f.Name]; exists {
				groupName := findBestGroup(metadata.FamilyName, groupedFamilies)
				families[groupName] = append(families[groupName], f)
			}
		}

		// Sort families alphabetically
		var familyNames []string
		for family := range families {
			familyNames = append(familyNames, family)
		}
		sort.Strings(familyNames)

		GetLogger().Info("Found %d font families", len(familyNames))
		output.GetVerbose().Info("Found %d font families with %d total fonts", len(familyNames), len(filteredFonts))
		output.GetDebug().State("Font families: %v", familyNames)

		// Display page title
		fmt.Printf("\n%s\n", ui.PageTitle.Render("Installed Fonts"))

		// Check if we should show detailed view (hierarchy under each font)
		detailed, _ := cmd.Flags().GetBool("detailed")

		output.GetVerbose().Info("Using table display with detailed=%v", detailed)
		output.GetDebug().State("Table mode enabled - detailed=%v", detailed)

		// Build the info message about found fonts
		scopeInfo := "across both user and machine scopes"
		if scope == "user" {
			scopeInfo = "in the 'user' scope"
		} else if scope == "machine" {
			scopeInfo = "in the 'machine' scope"
		}

		infoMsg := fmt.Sprintf("Found %d fonts installed %s", len(filteredFonts), scopeInfo)
		fmt.Printf("\n%s\n\n", infoMsg)

		// Always use table format - use the list table format
		fmt.Println(ui.TableHeader.Render(GetListTableHeader()))
		fmt.Println(GetTableSeparator())

		// Print each family
		for i, family := range familyNames {
			fonts := families[family]

			// Sort fonts by style
			sort.Slice(fonts, func(i, j int) bool {
				return fonts[i].Style < fonts[j].Style
			})

			// Get the first font to represent the family (for Type, Date, Scope)
			representativeFont := fonts[0]

			// Font ID is blank for now (until ID matching is implemented)
			fontID := ""

			// Print family row with pink color for font name - using same pattern as search.go
			fmt.Printf("%s %-*s %-*s %-*s %-*s\n",
				ui.TableSourceName.Render(fmt.Sprintf("%-*s", TableColListName, truncateString(family, TableColListName))),
				TableColListID, fontID, // Blank for now
				TableColType, representativeFont.Type,
				TableColDate, representativeFont.InstallDate.Format("2006-01-02 15:04"),
				TableColScope, representativeFont.Scope)

			// If detailed mode, show variants under each font family
			if detailed {
				output.GetDebug().State("Showing detailed variants for family: %s", family)
				for _, font := range fonts {
					// Get the actual style name from metadata for this specific font
					fontPath := filepath.Join(fontManager.GetFontDir(platform.InstallationScope(font.Scope)), font.Name)
					metadata, err := platform.ExtractFontMetadata(fontPath)
					styleName := font.Style // Default to filename-based style
					if err == nil && metadata.StyleName != "" {
						styleName = metadata.StyleName // Use proper style name from metadata
					}

					// Create variant display with arrow and regular console color
					variantDisplay := fmt.Sprintf("  ↳ %s", styleName)
					fmt.Printf("%s %-*s %-*s %-*s %-*s\n",
						fmt.Sprintf("%-*s", TableColListName, variantDisplay), // Regular console color for variants
						TableColListID, "",
						TableColType, "",
						TableColDate, "",
						TableColScope, "")
				}
			}

			// Add spacing after each font family only in detailed mode
			if detailed && i < len(familyNames)-1 {
				fmt.Println()
			}
		}

		// Add final spacing for better terminal appearance
		fmt.Println()

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
	listCmd.Flags().BoolP("detailed", "d", false, "Show detailed hierarchical view of font families with variants")
}
