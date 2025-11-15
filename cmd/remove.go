package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fontget/internal/components"
	"fontget/internal/output"
	"fontget/internal/platform"
	"fontget/internal/repo"
	"fontget/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// RemovalStatus tracks the status of font removals.
// This is kept separate from OperationStatus for command-specific clarity and backward compatibility.
// It provides clearer field names (Removed vs Success) for the remove command context.
type RemovalStatus struct {
	Removed int
	Skipped int
	Failed  int
	Details []string
}

// RemoveResult tracks the result of removing a single font
type RemoveResult struct {
	Success int
	Skipped int
	Failed  int
	Status  string // "completed", "failed", "skipped"
	Message string
	Details []string // Font display names
	Errors  []string
}

// List of critical system fonts to not remove (filenames and families, case-insensitive, no extension)
var criticalSystemFonts = map[string]bool{
	// Windows core fonts
	"arial":                 true,
	"arialbold":             true,
	"arialitalic":           true,
	"arialbolditalic":       true,
	"calibri":               true,
	"calibribold":           true,
	"calibriitalic":         true,
	"calibribolditalic":     true,
	"segoeui":               true,
	"segoeuibold":           true,
	"segoeuiitalic":         true,
	"segoeuibolditalic":     true,
	"times":                 true,
	"timesnewroman":         true,
	"timesnewromanpsmt":     true,
	"courier":               true,
	"tahoma":                true,
	"verdana":               true,
	"symbol":                true,
	"wingdings":             true,
	"consolas":              true,
	"georgia":               true,
	"georgiabold":           true,
	"georgiaitalic":         true,
	"georgiabolditalic":     true,
	"comicsansms":           true,
	"comicsansmsbold":       true,
	"impact":                true,
	"trebuchetms":           true,
	"trebuchetmsbold":       true,
	"trebuchetmsitalic":     true,
	"trebuchetmsbolditalic": true,
	"palatino":              true,
	"palatinolinotype":      true,
	"bookantiqua":           true,
	"centurygothic":         true,
	"franklingothic":        true,
	"gillsans":              true,
	"gillsansmt":            true,

	// macOS core fonts
	"cambria":              true,
	"sfnsdisplay":          true,
	"sfnsrounded":          true,
	"sfnstext":             true,
	"geneva":               true,
	"monaco":               true,
	"lucida grande":        true,
	"menlo":                true,
	"helvetica":            true,
	"helveticaneue":        true,
	"myriad":               true,
	"myriadpro":            true,
	"myriadset":            true,
	"myriadsemibold":       true,
	"myriadsemibolditalic": true,
	"sanfrancisco":         true,
	"sfprodisplay":         true,
	"sfprotext":            true,
	"sfprorounded":         true,
	"athelas":              true,
	"seravek":              true,
	"seraveklight":         true,
	"seravekmedium":        true,
	"seraveksemibold":      true,
	"seravekbold":          true,
	"applegaramond":        true,
	"garamond":             true,
	"garamonditalic":       true,
	"garamondbold":         true,
	"garamondbolditalic":   true,
	"optima":               true,
	"optimabold":           true,
	"optimaitalic":         true,
	"optimabolditalic":     true,
	"futura":               true,
	"futurabold":           true,
	"futuraitalic":         true,
	"futurabolditalic":     true,

	// Linux system fonts
	"ubuntu":              true,
	"ubuntumono":          true,
	"ubuntubold":          true,
	"ubuntuitalic":        true,
	"ubuntubolditalic":    true,
	"cantarell":           true,
	"cantarellbold":       true,
	"cantarellitalic":     true,
	"cantarellbolditalic": true,
}

// isCriticalSystemFont checks if a font is a critical system font
func isCriticalSystemFont(fontName string) bool {
	name := strings.ToLower(fontName)
	name = strings.ReplaceAll(name, " ", "")
	name = strings.ReplaceAll(name, "-", "")
	name = strings.ReplaceAll(name, "_", "")
	name = strings.TrimSuffix(name, filepath.Ext(name))
	return criticalSystemFonts[name]
}

// findFontFamilyFiles returns a list of font files that belong to the given font family
func findFontFamilyFiles(fontFamily string, fontManager platform.FontManager, scope platform.InstallationScope) []string {
	// Get the font directory
	fontDir := fontManager.GetFontDir(scope)

	// Get all installed fonts
	var matchingFonts []string
	filepath.Walk(fontDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			// Extract family name from the font file
			family, _ := parseFontName(info.Name())

			// Normalize both names for comparison
			normalizedFamily := normalizeFontName(family)
			normalizedQuery := normalizeFontName(fontFamily)

			// Only match if the normalized names are exactly equal
			if normalizedFamily == normalizedQuery {
				matchingFonts = append(matchingFonts, info.Name())
			}
		}
		return nil
	})

	return matchingFonts
}

// normalizeFontName normalizes a font name for comparison
func normalizeFontName(name string) string {
	// Convert to lowercase
	name = strings.ToLower(name)
	// Remove spaces and special characters
	name = strings.ReplaceAll(name, " ", "")
	name = strings.ReplaceAll(name, "-", "")
	name = strings.ReplaceAll(name, "_", "")
	return name
}

// extractFontDisplayNameFromPath extracts the proper display name from a font file path
// Uses font metadata (SFNT name table) for accurate font names, falls back to filename parsing
func extractFontDisplayNameFromPath(fontPath string) string {
	// Try to extract metadata from the font file first (most accurate)
	if metadata, err := platform.ExtractFontMetadata(fontPath); err == nil {
		if metadata.FamilyName != "" {
			// Use FormatFontNameWithVariant to properly format the name with style
			return FormatFontNameWithVariant(metadata.FamilyName, metadata.StyleName)
		}
	}

	// Fallback to filename parsing if metadata extraction fails
	filename := filepath.Base(fontPath)
	return GetDisplayNameFromFilename(filename)
}

// extractFontFamilyNameFromPath extracts just the font family name (without variant) from a font file path
// Uses font metadata (SFNT name table) for accurate font names, falls back to filename parsing
func extractFontFamilyNameFromPath(fontPath string) string {
	// Try to extract metadata from the font file first (most accurate)
	if metadata, err := platform.ExtractFontMetadata(fontPath); err == nil {
		if metadata.FamilyName != "" {
			return metadata.FamilyName
		}
	}

	// Fallback to filename parsing if metadata extraction fails
	filename := filepath.Base(fontPath)
	// Extract just the family name (without variant)
	family, _ := parseFontName(filename)
	// Convert to proper display format (e.g., "RobotoMono" -> "Roboto Mono")
	return convertCamelCaseToSpaced(family)
}

// removeFont handles the core removal logic for a single font
func removeFont(
	fontName string,
	fontManager platform.FontManager,
	scope platform.InstallationScope,
	fontDir string,
	repository *repo.Repository,
	isCriticalSystemFont func(string) bool,
) (*RemoveResult, error) {
	result := &RemoveResult{
		Details: make([]string, 0),
		Errors:  make([]string, 0),
	}

	// Find font files in the specified scope
	matchingFonts := findFontFamilyFiles(fontName, fontManager, scope)

	// If no direct matches, try repository search
	if len(matchingFonts) == 0 && repository != nil {
		results, err := repository.SearchFonts(fontName, "false")
		if err == nil && len(results) > 0 {
			matchingFonts = findFontFamilyFiles(results[0].Name, fontManager, scope)
		}
	}

	if len(matchingFonts) == 0 {
		result.Status = "failed"
		result.Message = "Font not found"
		return result, fmt.Errorf("font not found: %s", fontName)
	}

	// Remove each matching font file
	for _, matchingFont := range matchingFonts {
		// Construct full font path for metadata extraction
		fontPath := filepath.Join(fontDir, matchingFont)

		// Check for protected system fonts
		if isCriticalSystemFont != nil && isCriticalSystemFont(matchingFont) {
			result.Skipped++
			fontDisplayName := extractFontDisplayNameFromPath(fontPath)
			result.Details = append(result.Details, fontDisplayName+" (Skipped)")
			continue
		}

		fontDisplayName := extractFontDisplayNameFromPath(fontPath)

		// Remove font
		err := fontManager.RemoveFont(matchingFont, scope)

		if err != nil {
			result.Failed++
			var errorMsg string
			errStr := err.Error()
			if containsAny(errStr, []string{"in use", "access denied", "permission"}) {
				errorMsg = "Font is in use or access denied"
			} else {
				errorMsg = "Failed to remove existing font"
			}
			result.Errors = append(result.Errors, errorMsg)
			result.Details = append(result.Details, fontDisplayName+" (Failed)")
			output.GetDebug().State("fontManager.RemoveFont() failed for %s: %v", matchingFont, err)
			continue
		}

		result.Success++
		result.Details = append(result.Details, fontDisplayName)
	}

	// Determine final status
	if result.Success > 0 {
		result.Status = "completed"
		result.Message = "Removed"
	} else if result.Failed > 0 {
		result.Status = "failed"
		result.Message = "Removal failed"
	} else if result.Skipped > 0 {
		result.Status = "skipped"
		result.Message = "Protected system font"
	}

	return result, nil
}

// containsAny checks if a string contains any of the given substrings (case-insensitive)
func containsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if strings.Contains(strings.ToLower(s), strings.ToLower(substr)) {
			return true
		}
	}
	return false
}

// parseFontName extracts a simple family and style from a filename (fallback for removal lookups)
func parseFontName(filename string) (family, style string) {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	if strings.Contains(name, " ") {
		parts := strings.Split(name, " ")
		if len(parts) > 1 {
			return parts[0], strings.Join(parts[1:], " ")
		}
		return name, "Regular"
	}
	parts := strings.Split(name, "-")
	if len(parts) == 1 {
		return parts[0], "Regular"
	}
	family = strings.Join(parts[:len(parts)-1], "-")
	style = parts[len(parts)-1]
	return
}

// buildInstalledFontsCache builds a cache of installed font names for efficient suggestions
func buildInstalledFontsCache(fontManager platform.FontManager) []string {
	var installedFonts []string
	scopes := []platform.InstallationScope{platform.UserScope, platform.MachineScope}

	for _, scope := range scopes {
		fontDir := fontManager.GetFontDir(scope)
		filepath.Walk(fontDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && (strings.HasSuffix(strings.ToLower(info.Name()), ".ttf") ||
				strings.HasSuffix(strings.ToLower(info.Name()), ".otf")) {
				// Extract font family name
				family, _ := parseFontName(info.Name())
				if family != "" {
					installedFonts = append(installedFonts, family)
				}
			}
			return nil
		})
	}

	// Remove duplicates
	seen := make(map[string]bool)
	var uniqueFonts []string
	for _, font := range installedFonts {
		if !seen[font] {
			seen[font] = true
			uniqueFonts = append(uniqueFonts, font)
		}
	}

	return uniqueFonts
}

// showInstalledFontNotFoundWithSuggestionsCached displays font not found error with suggestions using cached data
func showInstalledFontNotFoundWithSuggestionsCached(fontName string, installedFonts []string) {
	fmt.Printf("\n%s\n", ui.FeedbackError.Render(fmt.Sprintf("Font '%s' not found.", fontName)))

	// Find similar fonts from cache
	similar := findSimilarFonts(fontName, installedFonts, true) // true = installed fonts

	// Only show suggestions if there are matches (>= 1)
	if len(similar) > 0 {
		// Display similar fonts in table format like add command
		fmt.Printf("%s\n\n", ui.FeedbackText.Render("Did you mean one of these installed fonts?"))

		// Use consistent column widths and apply styling to the entire formatted string
		fmt.Printf("%s\n", ui.TableHeader.Render(GetSearchTableHeader()))
		fmt.Printf("%s\n", GetTableSeparator())

		// Display each similar font as a table row
		for _, font := range similar {
			// For installed fonts, we don't have full metadata, so use placeholders
			fmt.Printf("%s %-*s %-*s %-*s %-*s\n",
				ui.TableSourceName.Render(fmt.Sprintf("%-*s", TableColName, truncateString(font, TableColName))),
				TableColID, truncateString("N/A", TableColID),
				TableColLicense, truncateString("N/A", TableColLicense),
				TableColCategories, truncateString("N/A", TableColCategories),
				TableColSource, truncateString("System", TableColSource))
		}

		fmt.Printf("\n")
	} else {
		// No matches found - just show the error message, no additional guidance
		fmt.Println()
	}
}

var removeCmd = &cobra.Command{
	Use:          "remove <font-id> [<font-id2> <font-id3> ...]",
	Aliases:      []string{"uninstall"},
	Short:        "Remove fonts from your system",
	SilenceUsage: true,
	Long: `Remove fonts from your system. 
	
You can specify multiple fonts by separating them with spaces.
Font names with spaces should be wrapped in quotes. Comma-separated lists are also supported.

You can specify the removal scope using the --scope flag:
  - user (default): Remove font from current user
  - machine: Remove font system-wide (requires elevation)
  - all: Remove from both user and machine scopes (requires elevation)

Use --force to override critical system font protection.
`,
	Example: `  fontget remove "Roboto"
  fontget remove "Open Sans" "Fira Sans" "Noto Sans"
  fontget remove roboto firasans notosans
  fontget remove "roboto, firasans, notosans"
  fontget remove "Open Sans" -s machine -f
  fontget remove "Roboto" -s user
  fontget remove "opensans" --force`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
			fmt.Printf("\n%s\n\n", ui.RenderError("A font ID is required"))
			return cmd.Help()
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		GetLogger().Info("Starting font removal operation")

		// Debug-level information for developers
		// Note: Suppressed to avoid TUI interference (consistent with add command)
		// output.GetDebug().Message("Debug mode enabled - showing detailed diagnostic information")

		if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
			return nil
		}

		fontManager, err := platform.NewFontManager()
		if err != nil {
			GetLogger().Error("Failed to initialize font manager: %v", err)
			return fmt.Errorf("failed to initialize font manager: %w", err)
		}

		scopeFlag, _ := cmd.Flags().GetString("scope")
		forceFlag, _ := cmd.Flags().GetBool("force")
		// Note: Scope will be logged after auto-detection in debug mode

		status := RemovalStatus{Details: make([]string, 0)}

		r, err := repo.GetRepository()
		if err != nil {
			GetLogger().Error("Failed to initialize repository: %v", err)
			return fmt.Errorf("failed to initialize repository: %w", err)
		}

		// Auto-detect scope if not explicitly provided
		if scopeFlag == "" {
			isElevated, err := fontManager.IsElevated()
			if err != nil {
				GetLogger().Warn("Failed to detect elevation status: %v", err)
				output.GetVerbose().Warning("Failed to detect elevation status: %v", err)
				// Default to user scope if we can't detect elevation
				scopeFlag = "user"
			} else if isElevated {
				scopeFlag = "all"
				GetLogger().Info("Auto-detected elevated privileges, defaulting to 'all' scope")
				// Note: Suppressed to avoid TUI interference (consistent with add command)
			} else {
				scopeFlag = "user"
				GetLogger().Info("Auto-detected user privileges, defaulting to 'user' scope")
				// Note: Suppressed to avoid TUI interference (consistent with add command)
				// output.GetVerbose().Info("Auto-detected user privileges, defaulting to 'user' scope")
			}
		}

		// Log removal parameters after scope is determined (for debug mode)
		GetLogger().Info("Removal parameters - Scope: %s, Force: %v", scopeFlag, forceFlag)
		// Format font list as a bulleted list
		GetLogger().Info("Removing %d Font(s):", len(args))
		for _, fontName := range args {
			GetLogger().Info("  - %s", fontName)
		}

		// Verbose-level information for users - show operational details before progress bar
		// (After scope auto-detection so we can display the actual scope being used)
		if IsVerbose() && !IsDebug() {
			// Format scope label for display
			scopeDisplay := scopeFlag
			if scopeFlag == "all" {
				scopeDisplay = "all (user and machine)"
			}
			output.GetVerbose().Info("Scope: %s", scopeDisplay)
			output.GetVerbose().Info("Force mode: %v", forceFlag)
			output.GetVerbose().Info("Removing %d font(s)", len(args))
			// Add single blank line after verbose output (matches add command behavior)
			// This creates the blank line between verbose info and "not found" message when no fonts found
			fmt.Println()
		}

		// Determine which scopes to check
		var scopes []platform.InstallationScope
		var scopeLabel []string
		if scopeFlag == "all" {
			scopes = []platform.InstallationScope{platform.UserScope, platform.MachineScope}
			scopeLabel = []string{"user", "machine"}
		} else {
			s := platform.InstallationScope(scopeFlag)
			if s != platform.UserScope && s != platform.MachineScope {
				GetLogger().Error("Invalid scope '%s'", scopeFlag)
				return fmt.Errorf("invalid scope '%s'. Must be 'user', 'machine', or 'all'", scopeFlag)
			}
			scopes = []platform.InstallationScope{s}
			scopeLabel = []string{scopeFlag}
		}

		// Process font names from arguments
		fontNames := ParseFontNames(args)

		// Note: Header will be shown only for successful operations, not for not found cases
		// This matches the add command behavior

		// Pre-evaluate all fonts: separate found fonts from not found fonts
		// This ensures we only show found fonts in the progress bar
		type FontInfo struct {
			SearchName string // Original search name (e.g., "open sans")
			ProperName string // Proper display name (e.g., "Open Sans")
		}
		var foundFonts []FontInfo
		var notFoundFonts []string
		fontNameMap := make(map[string]string) // Maps search name to proper name for found fonts

		for _, fontName := range fontNames {
			properName := ""
			fontFound := false

			// Try to find the font and extract its proper name
			for _, scopeType := range scopes {
				matchingFonts := findFontFamilyFiles(fontName, fontManager, scopeType)
				if len(matchingFonts) > 0 {
					fontDir := fontManager.GetFontDir(scopeType)
					firstFontPath := filepath.Join(fontDir, matchingFonts[0])
					properName = extractFontFamilyNameFromPath(firstFontPath)
					fontFound = true
					break
				}
			}
			// If no direct matches, try repository search
			if !fontFound {
				results, err := r.SearchFonts(fontName, "false")
				if err == nil && len(results) > 0 {
					searchName := results[0].Name
					for _, scopeType := range scopes {
						matchingFonts := findFontFamilyFiles(searchName, fontManager, scopeType)
						if len(matchingFonts) > 0 {
							fontDir := fontManager.GetFontDir(scopeType)
							firstFontPath := filepath.Join(fontDir, matchingFonts[0])
							properName = extractFontFamilyNameFromPath(firstFontPath)
							fontFound = true
							break
						}
					}
				}
			}

			if fontFound && properName != "" {
				// Font found - add to found list
				foundFonts = append(foundFonts, FontInfo{
					SearchName: fontName,
					ProperName: properName,
				})
				fontNameMap[fontName] = properName
			} else {
				// Font not found - add to not found list
				notFoundFonts = append(notFoundFonts, fontName)
			}
		}

		// Show not found fonts in debug mode (before processing)
		if IsDebug() && len(notFoundFonts) > 0 {
			scopeDisplay := scopeFlag
			if scopeDisplay == "" {
				scopeDisplay = "user"
			}
			GetLogger().Info("The following font(s) were not found installed in the '%s' scope:", scopeDisplay)
			for _, fontName := range notFoundFonts {
				GetLogger().Info("  - %s", fontName)
				// Check if font exists in machine scope (only for user scope operations)
				if len(scopes) == 1 && scopes[0] == platform.UserScope {
					machineFonts := findFontFamilyFiles(fontName, fontManager, platform.MachineScope)
					if len(machineFonts) == 0 {
						// Try repository search
						results, err := r.SearchFonts(fontName, "false")
						if err == nil && len(results) > 0 {
							searchName := results[0].Name
							machineFonts = findFontFamilyFiles(searchName, fontManager, platform.MachineScope)
						}
					}
					if len(machineFonts) > 0 {
						// Extract proper name for display
						fontDir := fontManager.GetFontDir(platform.MachineScope)
						firstFontPath := filepath.Join(fontDir, machineFonts[0])
						properName := extractFontFamilyNameFromPath(firstFontPath)
						GetLogger().Info("    '%s' is available for removal in the 'machine' scope, use \"--scope machine\" to remove", properName)
					}
				}
			}
		}

		// If no fonts to remove, show not found message and exit (don't show progress bar)
		if len(foundFonts) == 0 {
			// Show not found fonts message
			if len(notFoundFonts) > 0 {
				// In verbose mode, verbose output already ends with \n, so we need one more for blank line
				// In non-verbose mode, we need to add a blank line before the message
				// Match add command behavior: only add blank line in non-verbose mode
				if !IsVerbose() {
					fmt.Println()
				}
				// Format scope for display
				scopeDisplay := scopeFlag
				if scopeDisplay == "" {
					scopeDisplay = "user"
				}
				// Check if fonts are available in the opposite scope (only for single scope operations)
				fontsInOppositeScope := make(map[string]string) // Maps search name to proper name in opposite scope
				oppositeScopeName := ""
				if len(scopes) == 1 {
					var oppositeScope platform.InstallationScope
					if scopes[0] == platform.UserScope {
						oppositeScope = platform.MachineScope
						oppositeScopeName = "machine"
					} else if scopes[0] == platform.MachineScope {
						oppositeScope = platform.UserScope
						oppositeScopeName = "user"
					}

					// Check each not found font in the opposite scope
					if oppositeScope != "" {
						for _, fontName := range notFoundFonts {
							// Try direct match first
							matchingFonts := findFontFamilyFiles(fontName, fontManager, oppositeScope)
							if len(matchingFonts) == 0 {
								// Try repository search
								results, err := r.SearchFonts(fontName, "false")
								if err == nil && len(results) > 0 {
									searchName := results[0].Name
									matchingFonts = findFontFamilyFiles(searchName, fontManager, oppositeScope)
								}
							}
							if len(matchingFonts) > 0 {
								// Extract proper name for display
								fontDir := fontManager.GetFontDir(oppositeScope)
								firstFontPath := filepath.Join(fontDir, matchingFonts[0])
								properName := extractFontFamilyNameFromPath(firstFontPath)
								fontsInOppositeScope[fontName] = properName
							}
						}
					}
				}

				// Separate fonts into two groups: truly not found vs found in opposite scope
				trulyNotFound := []string{}
				for _, fontName := range notFoundFonts {
					if _, foundInOpposite := fontsInOppositeScope[fontName]; !foundInOpposite {
						trulyNotFound = append(trulyNotFound, fontName)
					}
				}

				// Display truly not found fonts (only if there are any)
				if len(trulyNotFound) > 0 {
					fmt.Printf("%s\n", ui.FeedbackError.Render(fmt.Sprintf("The following font(s) were not found installed in the '%s' scope:", scopeDisplay)))
					for _, fontName := range trulyNotFound {
						fmt.Printf("%s\n", ui.FeedbackError.Render(fmt.Sprintf("  - %s", fontName)))
					}
				}

				// Display fonts found in opposite scope as a separate section
				if len(fontsInOppositeScope) > 0 {
					// Add blank line before the opposite scope section (if we showed "not found" section)
					if len(trulyNotFound) > 0 {
						fmt.Println()
					}
					// Use FeedbackWarning (yellow, no bold) for the header message
					fmt.Printf("%s\n", ui.FeedbackWarning.Render(fmt.Sprintf("The following font(s) are installed in the '%s' scope, use '--scope %s' to remove them:", oppositeScopeName, oppositeScopeName)))
					for _, fontName := range notFoundFonts {
						if properName, foundInOpposite := fontsInOppositeScope[fontName]; foundInOpposite {
							fmt.Printf("  - %s\n", properName)
						}
					}
					// Add blank line after the list before prompt
					fmt.Println()
				} else {
					// No fonts in opposite scope, but we might have truly not found fonts
					// Add blank line before hint message
					fmt.Println()
					// Show appropriate hint message
					if len(trulyNotFound) > 0 {
						// Some fonts are truly not found
						fmt.Printf("Try using 'fontget list' to show currently installed fonts.\n")
						// Add blank line after message before prompt
						fmt.Println()
					}
				}
			}
			return nil
		}

		// For --all scope, require elevation upfront
		if len(scopes) == 2 {
			// Check elevation first
			if err := checkElevation(cmd, fontManager, platform.MachineScope); err != nil {
				if errors.Is(err, ErrElevationRequired) {
					return nil // Already printed user-friendly message
				}
				GetLogger().Error("Elevation check failed for --scope all: %v", err)
				return err
			}

			// Create operation items only for found fonts - use proper names from the start
			var operationItems []components.OperationItem
			for _, fontInfo := range foundFonts {
				operationItems = append(operationItems, components.OperationItem{
					Name:          fontInfo.ProperName,
					Status:        "pending",
					StatusMessage: "Pending",
					Variants:      []string{},
					Scope:         "",
				})
			}

			// Check if flags are set
			list, _ := cmd.Flags().GetBool("list")
			verbose, _ := cmd.Flags().GetBool("verbose")
			debug, _ := cmd.Flags().GetBool("debug")

			// For debug mode: bypass TUI and use plain text output
			if IsDebug() {
				output.GetDebug().State("Starting font removal operation")
				output.GetDebug().State("Total fonts: %d", len(foundFonts))

				// Process each found font across all scopes
				for i, fontInfo := range foundFonts {
					output.GetDebug().State("Removing font %d/%d: %s", i+1, len(foundFonts), fontInfo.SearchName)

					// Process each scope separately
					for j, scopeType := range scopes {
						scopeLabelName := scopeLabel[j]
						fontDir := fontManager.GetFontDir(scopeType)
						output.GetDebug().State("Removing font %s in %s scope (directory: %s)", fontInfo.SearchName, scopeLabelName, fontDir)

						result, err := removeFont(
							fontInfo.SearchName,
							fontManager,
							scopeType,
							fontDir,
							r,
							isCriticalSystemFont,
						)

						if err != nil {
							output.GetDebug().State("Error removing font %s in %s scope: %v", fontInfo.SearchName, scopeLabelName, err)
							if result != nil {
								status.Failed += result.Failed
								status.Skipped += result.Skipped
								if len(result.Details) > 0 {
									// Format failed variants as a list with hyphens
									output.GetDebug().State("Failed variants:")
									for _, detail := range result.Details {
										output.GetDebug().State("  - %s", detail)
									}
								}
							}
							continue
						}

						status.Removed += result.Success
						status.Skipped += result.Skipped
						status.Failed += result.Failed

						// Show detailed result information in debug mode
						if len(result.Details) > 0 {
							output.GetDebug().State("Removed variants:")
							for _, detail := range result.Details {
								// Remove status suffixes like " (Skipped)" or " (Failed)" for cleaner output
								variantName := strings.TrimSuffix(strings.TrimSuffix(detail, " (Skipped)"), " (Failed)")
								output.GetDebug().State("  - %s", variantName)
							}
						}
						output.GetDebug().State("Font %s in %s scope completed: %s - %s (Removed: %d, Skipped: %d, Failed: %d)",
							fontInfo.SearchName, scopeLabelName, result.Status, result.Message, result.Success, result.Skipped, result.Failed)
					}
				}

				output.GetDebug().State("Operation complete - Removed: %d, Skipped: %d, Failed: %d",
					status.Removed, status.Skipped, status.Failed)

				// Print status report
				fmt.Println()
				PrintStatusReport(StatusReport{
					Success:      status.Removed,
					Skipped:      status.Skipped,
					Failed:       status.Failed,
					SuccessLabel: "Removed",
					SkippedLabel: "Skipped",
					FailedLabel:  "Failed",
				})

				GetLogger().Info("Removal complete - Removed: %d, Skipped: %d, Failed: %d",
					status.Removed, status.Skipped, status.Failed)
				return nil
			}

			// Run unified progress for font removal (TUI mode)
			progressErr := components.RunProgressBar(
				"Removing Fonts",
				operationItems,
				list,    // List mode: show file listings
				verbose, // Verbose mode: show operational details
				debug,   // Debug mode: show technical details
				func(send func(msg tea.Msg)) error {
					// Process each found font across all scopes
					for i, fontInfo := range foundFonts {
						// Use proper font name from pre-extracted map
						properFontName := fontInfo.ProperName
						fontName := fontInfo.SearchName

						send(components.ItemUpdateMsg{
							Index:   i,
							Name:    properFontName, // Use proper name from the start
							Status:  "in_progress",
							Message: "Finding fonts...",
						})

						GetLogger().Info("Removing font: %s", fontName)

						var allRemovedVariants []string
						var fontStatus string = "completed"
						var statusMessage string = "Removed"
						var finalScopeLabel string = ""
						var allErrors []string // Collect errors from all scopes

						// Process each scope
						for j, scopeType := range scopes {
							scopeLabelName := scopeLabel[j]
							fontDir := fontManager.GetFontDir(scopeType)

							// Remove the font using the removeFont helper
							result, err := removeFont(
								fontName,
								fontManager,
								scopeType,
								fontDir,
								r,
								isCriticalSystemFont,
							)

							if err != nil {
								// Font not found - mark as failed and continue with other fonts
								if strings.Contains(err.Error(), "not found") {
									fontStatus = "failed"
									statusMessage = "Font not found"
									allErrors = append(allErrors, "Font not found")
									status.Failed++
									// Continue to next font instead of exiting
									break // Break out of scope loop, continue to next font
								}
								// Other errors - continue to next scope
								continue
							}

							// Update status
							status.Removed += result.Success
							status.Skipped += result.Skipped
							status.Failed += result.Failed

							// Collect errors
							allErrors = append(allErrors, result.Errors...)

							// Collect variants for display
							allRemovedVariants = append(allRemovedVariants, result.Details...)

							// Determine overall status
							if result.Status == "failed" {
								fontStatus = "failed"
								statusMessage = result.Message
							} else if result.Status == "skipped" && fontStatus != "failed" {
								fontStatus = "skipped"
								statusMessage = result.Message
							}

							if result.Success > 0 {
								finalScopeLabel = scopeLabelName
							}
						}

						// Get first error message if status is failed
						var errorMsg string
						if fontStatus == "failed" && len(allErrors) > 0 {
							errorMsg = allErrors[0]
						}

						// Mark as completed - use proper font name if we found it
						updateMsg := components.ItemUpdateMsg{
							Index:        i,
							Status:       fontStatus,
							Message:      statusMessage,
							ErrorMessage: errorMsg,
							Variants:     allRemovedVariants,
							Scope:        finalScopeLabel,
						}
						if properFontName != "" {
							updateMsg.Name = properFontName
						}
						send(updateMsg)

						// Update progress percentage
						percent := float64(i+1) / float64(len(foundFonts)) * 100
						send(components.ProgressUpdateMsg{Percent: percent})
					}

					return nil
				},
			)

			if progressErr != nil {
				GetLogger().Error("Failed to process font removal: %v", progressErr)
				return progressErr
			}

			GetLogger().Info("Removal complete - Removed: %d, Skipped: %d, Failed: %d",
				status.Removed, status.Skipped, status.Failed)

			// Show not found fonts right after progress bar output (before status report)
			if len(notFoundFonts) > 0 {
				// Progress bar ends with \n - add \n to create blank line before "not found" messages
				fmt.Println()
				// Format scope for display
				scopeDisplay := scopeFlag
				if scopeDisplay == "" {
					scopeDisplay = "user"
				}
				// Format message to match add command style
				fmt.Printf("%s\n", ui.FeedbackError.Render(fmt.Sprintf("The following font(s) were not found installed in the '%s' scope:", scopeDisplay)))
				for _, fontName := range notFoundFonts {
					fmt.Printf("%s\n", ui.FeedbackError.Render(fmt.Sprintf("  - %s", fontName)))
				}
				// Add blank line before "Try using..." message
				fmt.Println()
				fmt.Printf("Try using 'fontget list' to show currently installed fonts.\n")
				// Add blank line after "not found" messages before prompt
				// (Status report will only show if there were operations, so we need spacing here)
				fmt.Println()
			}

			// Print status report last (after error messages)
			PrintStatusReport(StatusReport{
				Success:      status.Removed,
				Skipped:      status.Skipped,
				Failed:       status.Failed,
				SuccessLabel: "Removed",
				SkippedLabel: "Skipped",
				FailedLabel:  "Failed",
			})

			// Don't return error for removal failures since we already show detailed status report
			// This prevents duplicate error messages while maintaining proper exit codes
			return nil
		}

		// Handle single scope operations (user or machine) - use TUI like add command
		// Create operation items only for found fonts - use proper names from the start
		var operationItems []components.OperationItem
		for _, fontInfo := range foundFonts {
			operationItems = append(operationItems, components.OperationItem{
				Name:          fontInfo.ProperName,
				Status:        "pending",
				StatusMessage: "Pending",
				Variants:      []string{},
				Scope:         "",
			})
		}

		// Check if flags are set
		list, _ := cmd.Flags().GetBool("list")
		verbose, _ := cmd.Flags().GetBool("verbose")
		debug, _ := cmd.Flags().GetBool("debug")

		// For debug mode: bypass TUI and use plain text output
		if IsDebug() {
			output.GetDebug().State("Starting font removal operation")
			output.GetDebug().State("Total fonts: %d", len(foundFonts))

			// Process each found font
			for i, fontInfo := range foundFonts {
				output.GetDebug().State("Removing font %d/%d: %s", i+1, len(foundFonts), fontInfo.SearchName)

				// Process each scope separately
				for j, scopeType := range scopes {
					scopeLabelName := scopeLabel[j]
					fontDir := fontManager.GetFontDir(scopeType)
					output.GetDebug().State("Removing font %s in %s scope (directory: %s)", fontInfo.SearchName, scopeLabelName, fontDir)

					result, err := removeFont(
						fontInfo.SearchName,
						fontManager,
						scopeType,
						fontDir,
						r,
						isCriticalSystemFont,
					)

					if err != nil {
						output.GetDebug().State("Error removing font %s in %s scope: %v", fontInfo.SearchName, scopeLabelName, err)
						if result != nil {
							status.Failed += result.Failed
							status.Skipped += result.Skipped
							if len(result.Details) > 0 {
								// Format failed variants as a list with hyphens
								output.GetDebug().State("Failed variants:")
								for _, detail := range result.Details {
									output.GetDebug().State("  - %s", detail)
								}
							}
						}
						continue
					}

					status.Removed += result.Success
					status.Skipped += result.Skipped
					status.Failed += result.Failed

					// Show detailed result information in debug mode
					if len(result.Details) > 0 {
						output.GetDebug().State("Removed variants:")
						for _, detail := range result.Details {
							// Remove status suffixes like " (Skipped)" or " (Failed)" for cleaner output
							variantName := strings.TrimSuffix(strings.TrimSuffix(detail, " (Skipped)"), " (Failed)")
							output.GetDebug().State("  - %s", variantName)
						}
					}
					output.GetDebug().State("Font %s in %s scope completed: %s - %s (Removed: %d, Skipped: %d, Failed: %d)",
						fontInfo.SearchName, scopeLabelName, result.Status, result.Message, result.Success, result.Skipped, result.Failed)
				}
			}

			output.GetDebug().State("Operation complete - Removed: %d, Skipped: %d, Failed: %d",
				status.Removed, status.Skipped, status.Failed)

			// Print status report
			fmt.Println()
			PrintStatusReport(StatusReport{
				Success:      status.Removed,
				Skipped:      status.Skipped,
				Failed:       status.Failed,
				SuccessLabel: "Removed",
				SkippedLabel: "Skipped",
				FailedLabel:  "Failed",
			})

			GetLogger().Info("Removal complete - Removed: %d, Skipped: %d, Failed: %d",
				status.Removed, status.Skipped, status.Failed)
			return nil
		}

		// Run unified progress for font removal (TUI mode)
		fontsInOppositeScope := []string{} // Track fonts that still exist in opposite scope after removal
		progressErr := components.RunProgressBar(
			"Removing Fonts",
			operationItems,
			list,    // List mode: show file listings
			verbose, // Verbose mode: show operational details
			debug,   // Debug mode: show technical details
			func(send func(msg tea.Msg)) error {
				// Process each found font
				for i, fontInfo := range foundFonts {
					// Use proper font name from pre-extracted map
					properFontName := fontInfo.ProperName
					fontName := fontInfo.SearchName

					send(components.ItemUpdateMsg{
						Index:   i,
						Name:    properFontName, // Use proper name from the start
						Status:  "in_progress",
						Message: "Finding fonts...",
					})

					GetLogger().Info("Processing font: %s", fontName)

					var allRemovedVariants []string
					var fontStatus string = "completed"
					var statusMessage string = "Removed"
					var finalScopeLabel string = ""
					var allErrors []string // Collect errors from all scopes
					var percent float64    // Progress percentage

					// Special handling for user scope only: check both scopes for better UX
					if len(scopes) == 1 && scopes[0] == platform.UserScope {
						// Check both scopes efficiently
						machineFonts := findFontFamilyFiles(fontName, fontManager, platform.MachineScope)
						userFonts := findFontFamilyFiles(fontName, fontManager, platform.UserScope)

						// If no direct matches, try repository search to find the font
						if len(machineFonts) == 0 && len(userFonts) == 0 {
							results, err := r.SearchFonts(fontName, "false")
							if err == nil && len(results) > 0 {
								searchName := results[0].Name
								machineFonts = findFontFamilyFiles(searchName, fontManager, platform.MachineScope)
								userFonts = findFontFamilyFiles(searchName, fontManager, platform.UserScope)
							}
						}

						// Handle different scenarios
						if len(userFonts) == 0 && len(machineFonts) == 0 {
							// Font not found in either scope - mark as failed and continue
							fontStatus = "failed"
							statusMessage = "Font not found"
							allErrors = append(allErrors, "Font not found")
							status.Failed++
							// Send update with proper name and update progress
							send(components.ItemUpdateMsg{
								Index:        i,
								Name:         properFontName, // Use proper name from map
								Status:       fontStatus,
								Message:      statusMessage,
								ErrorMessage: "Font not found",
							})
							// Update progress percentage
							percent = float64(i+1) / float64(len(foundFonts)) * 100
							send(components.ProgressUpdateMsg{Percent: percent})
							// Continue to next font instead of exiting
							continue
						} else if len(userFonts) == 0 && len(machineFonts) > 0 {
							// Font only exists in machine scope
							status.Skipped++
							fontStatus = "skipped"
							statusMessage = "Only installed in machine scope"
							// Send update before continuing
							send(components.ItemUpdateMsg{
								Index:   i,
								Name:    properFontName,
								Status:  fontStatus,
								Message: statusMessage,
							})
							// Update progress percentage
							percent = float64(i+1) / float64(len(fontNames)) * 100
							send(components.ProgressUpdateMsg{Percent: percent})
							continue
						}
						// Note: We'll check if font still exists in opposite scope AFTER removal
						// Don't track here to avoid duplicates

						// properFontName already set from fontNameMap above

						// Process removal from user scope
						fontDir := fontManager.GetFontDir(platform.UserScope)
						result, err := removeFont(
							fontName,
							fontManager,
							platform.UserScope,
							fontDir,
							r,
							isCriticalSystemFont,
						)

						if err != nil {
							if strings.Contains(err.Error(), "not found") {
								// Font not found - mark as failed and continue
								fontStatus = "failed"
								statusMessage = "Font not found"
								allErrors = append(allErrors, "Font not found")
								status.Failed++
								// Continue to next font instead of exiting
							} else {
								fontStatus = "failed"
								statusMessage = err.Error()
								// Add error message
								allErrors = append(allErrors, err.Error())
								if result != nil {
									status.Failed += result.Failed
									allErrors = append(allErrors, result.Errors...)
								}
							}
						} else {

							status.Removed += result.Success
							status.Skipped += result.Skipped
							status.Failed += result.Failed

							// Collect errors
							allErrors = append(allErrors, result.Errors...)

							// Collect variants for display (only in verbose/list mode)
							if verbose || list {
								allRemovedVariants = append(allRemovedVariants, result.Details...)
							}

							if result.Status == "failed" {
								fontStatus = "failed"
								statusMessage = result.Message
							} else if result.Status == "skipped" && fontStatus != "failed" {
								fontStatus = "skipped"
								statusMessage = result.Message
							}

							if result.Success > 0 {
								finalScopeLabel = "user scope"
								// After successful removal from user scope, check if font still exists in machine scope
								machineFonts := findFontFamilyFiles(fontName, fontManager, platform.MachineScope)
								if len(machineFonts) == 0 {
									// Try repository search
									results, err := r.SearchFonts(fontName, "false")
									if err == nil && len(results) > 0 {
										searchName := results[0].Name
										machineFonts = findFontFamilyFiles(searchName, fontManager, platform.MachineScope)
									}
								}
								if len(machineFonts) > 0 {
									// Font still exists in machine scope - track for display
									fontsInOppositeScope = append(fontsInOppositeScope, properFontName)
								}
							}
						}
					} else {
						// Handle machine scope or all scopes
						for j, scopeType := range scopes {
							scopeLabelName := scopeLabel[j]
							fontDir := fontManager.GetFontDir(scopeType)

							// Extract proper font family name from first matching font file BEFORE removal
							if properFontName == "" {
								matchingFonts := findFontFamilyFiles(fontName, fontManager, scopeType)
								if len(matchingFonts) > 0 {
									firstFontPath := filepath.Join(fontDir, matchingFonts[0])
									properFontName = extractFontFamilyNameFromPath(firstFontPath)
								}
							}

							// Remove the font using the removeFont helper
							result, err := removeFont(
								fontName,
								fontManager,
								scopeType,
								fontDir,
								r,
								isCriticalSystemFont,
							)

							if err != nil {
								if strings.Contains(err.Error(), "not found") {
									// Font not found - mark as failed and continue
									fontStatus = "failed"
									statusMessage = "Font not found"
									allErrors = append(allErrors, "Font not found")
									status.Failed++
									// Continue to next scope or font instead of exiting
									continue
								}
								fontStatus = "failed"
								statusMessage = err.Error()
								// Add error message
								allErrors = append(allErrors, err.Error())
								if result != nil {
									status.Failed += result.Failed
									allErrors = append(allErrors, result.Errors...)
								}
								continue
							}

							// Update status
							status.Removed += result.Success
							status.Skipped += result.Skipped
							status.Failed += result.Failed

							// Collect errors
							allErrors = append(allErrors, result.Errors...)

							// Collect variants for display (only in verbose/list mode)
							if verbose || list {
								allRemovedVariants = append(allRemovedVariants, result.Details...)
							}

							// Determine overall status
							if result.Status == "failed" {
								fontStatus = "failed"
								statusMessage = result.Message
							} else if result.Status == "skipped" && fontStatus != "failed" {
								fontStatus = "skipped"
								statusMessage = result.Message
							}

							if result.Success > 0 {
								finalScopeLabel = scopeLabelName + " scope"
								// After successful removal, check if font still exists in opposite scope (only for single scope operations)
								if len(scopes) == 1 {
									var oppositeScope platform.InstallationScope
									if scopeType == platform.UserScope {
										oppositeScope = platform.MachineScope
									} else if scopeType == platform.MachineScope {
										oppositeScope = platform.UserScope
									}
									if oppositeScope != "" {
										oppositeFonts := findFontFamilyFiles(fontName, fontManager, oppositeScope)
										if len(oppositeFonts) == 0 {
											// Try repository search
											results, err := r.SearchFonts(fontName, "false")
											if err == nil && len(results) > 0 {
												searchName := results[0].Name
												oppositeFonts = findFontFamilyFiles(searchName, fontManager, oppositeScope)
											}
										}
										if len(oppositeFonts) > 0 {
											// Font still exists in opposite scope - track for display
											fontsInOppositeScope = append(fontsInOppositeScope, properFontName)
										}
									}
								}
							}
						}
					}

					// Get first error message if status is failed
					var errorMsg string
					if fontStatus == "failed" && len(allErrors) > 0 {
						errorMsg = allErrors[0]
					}

					// Mark as completed - use proper font name if we found it
					updateMsg := components.ItemUpdateMsg{
						Index:        i,
						Status:       fontStatus,
						Message:      statusMessage,
						ErrorMessage: errorMsg,
						Variants:     allRemovedVariants,
						Scope:        finalScopeLabel,
					}
					if properFontName != "" {
						updateMsg.Name = properFontName
					}
					send(updateMsg)

					// Update progress percentage
					percent = float64(i+1) / float64(len(foundFonts)) * 100
					send(components.ProgressUpdateMsg{Percent: percent})
				}

				return nil
			},
		)

		if progressErr != nil {
			GetLogger().Error("Failed to process font removal: %v", progressErr)
			return progressErr
		}

		GetLogger().Info("Removal complete - Removed: %d, Skipped: %d, Failed: %d",
			status.Removed, status.Skipped, status.Failed)

		// Show not found fonts right after progress bar output (before status report)
		if len(notFoundFonts) > 0 {
			// Progress bar ends with \n - add \n to create blank line before "not found" messages
			fmt.Println()
			// Format scope for display
			scopeDisplay := scopeFlag
			if scopeDisplay == "" {
				scopeDisplay = "user"
			}
			// Format message to match add command style
			fmt.Printf("%s\n", ui.FeedbackError.Render(fmt.Sprintf("The following font(s) were not found installed in the '%s' scope:", scopeDisplay)))
			for _, fontName := range notFoundFonts {
				fmt.Printf("%s\n", ui.FeedbackError.Render(fmt.Sprintf("  - %s", fontName)))
			}
			// Add blank line before "Try using..." message
			fmt.Println()
			fmt.Printf("Try using 'fontget list' to show currently installed fonts.\n")
			// Only add blank line after "Try using..." if there are no fonts in opposite scope
			// (If there are fonts in opposite scope, that section will handle spacing)
			if len(fontsInOppositeScope) == 0 {
				fmt.Println()
			}
		}

		// Show fonts that still exist in opposite scope after removal (before status report)
		if len(fontsInOppositeScope) > 0 {
			// Determine opposite scope name for display
			var oppositeScopeName string
			if len(scopes) == 1 {
				if scopes[0] == platform.UserScope {
					oppositeScopeName = "machine"
				} else if scopes[0] == platform.MachineScope {
					oppositeScopeName = "user"
				}
			}

			if oppositeScopeName != "" {
				// Remove duplicates from the list
				seen := make(map[string]bool)
				uniqueFonts := []string{}
				for _, fontName := range fontsInOppositeScope {
					if !seen[fontName] {
						seen[fontName] = true
						uniqueFonts = append(uniqueFonts, fontName)
					}
				}

				// Add blank line before opposite scope messages
				// (If "not found" section was shown, "Try using..." already ended with \n, so this creates the blank line)
				// (If no "not found" section, progress bar ended with \n, so this creates the blank line)
				fmt.Println()
				// Use FeedbackWarning (yellow, no bold) for the header message
				fmt.Printf("%s\n", ui.FeedbackWarning.Render(fmt.Sprintf("The following font(s) are installed in the '%s' scope, use '--scope %s' to remove them:", oppositeScopeName, oppositeScopeName)))
				for _, fontName := range uniqueFonts {
					fmt.Printf("  - %s\n", fontName)
				}
				// Add blank line after the list, but only if not in verbose mode
				// (In verbose mode, PrintStatusReport will add spacing)
				if !IsVerbose() {
					fmt.Println()
				}
			}
		}

		// Print status report last (after error messages and machine scope info and only if there were actual operations)
		PrintStatusReport(StatusReport{
			Success:      status.Removed,
			Skipped:      status.Skipped,
			Failed:       status.Failed,
			SuccessLabel: "Removed",
			SkippedLabel: "Skipped",
			FailedLabel:  "Failed",
		})

		// If no status report and no other messages, add blank line before prompt
		// (Status report already ends with \n\n, so no need to add another blank line)
		if !IsVerbose() && len(notFoundFonts) == 0 && len(fontsInOppositeScope) == 0 {
			// Progress bar ends with \n - add \n to create blank line before prompt
			fmt.Println()
		}

		// Don't return error for removal failures since we already show detailed status report
		// This prevents duplicate error messages while maintaining proper exit codes
		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
	removeCmd.Flags().StringP("scope", "s", "", "Installation scope (user, machine, or all)")
	removeCmd.Flags().BoolP("force", "f", false, "Force removal of critical system fonts")
}
