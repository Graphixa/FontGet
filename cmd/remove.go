package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fontget/internal/platform"
	"fontget/internal/repo"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	scope string
	force bool
)

// promptYesNo asks the user a yes/no question and returns true for yes, false for no
func promptYesNo(message string) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(message + " (y/n): ")
		response, err := reader.ReadString('\n')
		if err != nil {
			return false, fmt.Errorf("failed to read user input: %w", err)
		}
		response = strings.ToLower(strings.TrimSpace(response))
		if response == "y" || response == "yes" {
			return true, nil
		}
		if response == "n" || response == "no" {
			return false, nil
		}
		fmt.Println("Please answer 'y' or 'n'")
	}
}

// RemovalStatus tracks the status of font removals
type RemovalStatus struct {
	Removed int
	Skipped int
	Failed  int
	Details []string
}

// List of critical system fonts to not remove (filenames and families, case-insensitive, no extension)
var criticalSystemFonts = map[string]bool{
	// Windows core fonts
	"arial":             true,
	"arialbold":         true,
	"arialitalic":       true,
	"arialbolditalic":   true,
	"calibri":           true,
	"calibribold":       true,
	"calibriitalic":     true,
	"calibribolditalic": true,
	"segoeui":           true,
	"segoeuibold":       true,
	"segoeuiitalic":     true,
	"segoeuibolditalic": true,
	"times":             true,
	"timesnewroman":     true,
	"timesnewromanpsmt": true,
	"courier":           true,
	"tahoma":            true,
	"verdana":           true,
	"symbol":            true,
	"wingdings":         true,
	// macOS core fonts
	"sfnsdisplay":   true,
	"sfnsrounded":   true,
	"sfnstext":      true,
	"geneva":        true,
	"monaco":        true,
	"lucida grande": true,
	"menlo":         true,
	"helvetica":     true,
	"helveticaneue": true,
	// Linux common fonts
	"dejavusans":       true,
	"dejavusansmono":   true,
	"dejavuserif":      true,
	"liberationsans":   true,
	"liberationserif":  true,
	"liberationmono":   true,
	"nimbusroman":      true,
	"nimbussans":       true,
	"nimbusmono":       true,
	"ubuntu":           true,
	"ubuntumono":       true,
	"ubuntubold":       true,
	"ubuntuitalic":     true,
	"ubuntubolditalic": true,
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

// findSimilarInstalledFonts returns a list of installed font names that are similar to the given name
func findSimilarInstalledFonts(fontName string, fontManager platform.FontManager, scope platform.InstallationScope) []string {
	// Get the font directory
	fontDir := fontManager.GetFontDir(scope)

	// Get all installed fonts
	var installedFonts []string
	filepath.Walk(fontDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			installedFonts = append(installedFonts, info.Name())
		}
		return nil
	})

	// Find similar fonts using string similarity
	var similar []string
	normalizedQuery := normalizeFontName(fontName)
	for _, font := range installedFonts {
		family, _ := parseFontName(font)
		normalizedFamily := normalizeFontName(family)

		// Check if the normalized family name contains the query or vice versa
		if strings.Contains(normalizedFamily, normalizedQuery) || strings.Contains(normalizedQuery, normalizedFamily) {
			similar = append(similar, font)
		}
	}

	// Limit to 5 suggestions
	if len(similar) > 5 {
		similar = similar[:5]
	}

	return similar
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
  - all: Remove from both user and machine scopes
  
Fonts are removed under both the user and machine scopes by default.

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
			red := color.New(color.FgRed).SprintFunc()
			fmt.Printf("\n%s\n\n", red("A font ID is required"))
			return cmd.Help()
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		GetLogger().Info("Starting font removal operation")

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
		GetLogger().Info("Removal parameters - Scope: %s, Force: %v", scopeFlag, forceFlag)

		GetLogger().Info("Processing %d font(s): %v", len(args), args)

		status := RemovalStatus{Details: make([]string, 0)}
		green := color.New(color.FgGreen).SprintFunc()
		yellow := color.New(color.FgYellow).SprintFunc()
		red := color.New(color.FgRed).SprintFunc()
		bold := color.New(color.Bold).SprintFunc()
		cyan := color.New(color.FgCyan).SprintFunc()

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
				// Default to user scope if we can't detect elevation
				scopeFlag = "user"
			} else if isElevated {
				scopeFlag = "all"
				GetLogger().Info("Auto-detected elevated privileges, defaulting to 'all' scope")
				fmt.Println(cyan("Auto-detected administrator privileges - removing from all scopes"))
			} else {
				scopeFlag = "user"
				GetLogger().Info("Auto-detected user privileges, defaulting to 'user' scope")
			}
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
		var fontNames []string
		for _, arg := range args {
			// Split each argument by comma in case user provides comma-separated list
			names := strings.Split(arg, ",")
			for _, name := range names {
				name = strings.TrimSpace(name)
				if name != "" {
					fontNames = append(fontNames, name)
				}
			}
		}

		GetLogger().Info("Processing %d font(s): %v", len(fontNames), fontNames)

		// For --all scope, require elevation upfront
		if len(scopes) == 2 {
			// Check elevation first
			if err := checkElevation(cmd, fontManager, platform.MachineScope); err != nil {
				GetLogger().Error("Elevation check failed for --scope all: %v", err)
				fmt.Println(red("This operation requires administrator privileges."))
				fmt.Println("To run as administrator:")
				fmt.Println("  1. Right-click on Command Prompt or PowerShell.")
				fmt.Println("  2. Select 'Run as administrator'.")
				fmt.Printf("  3. Run: fontget remove --scope all %s\n", strings.Join(fontNames, " "))
				return fmt.Errorf("elevation required for --scope all")
			}

			// Process fonts with simple single-status-report approach
			for _, fontName := range fontNames {
				GetLogger().Info("Processing font: %s", fontName)
				fmt.Printf("\n%s\n", bold(fontName))

				for i, scope := range scopes {
					label := scopeLabel[i]
					GetLogger().Info("Checking scope: %s", label)

					matchingFonts := findFontFamilyFiles(fontName, fontManager, scope)
					if len(matchingFonts) == 0 {
						results, err := r.SearchFonts(fontName, "false")
						if err == nil && len(results) > 0 {
							matchingFonts = findFontFamilyFiles(results[0].Name, fontManager, scope)
						}
					}

					if len(matchingFonts) == 0 {
						if isCriticalSystemFont(fontName) {
							msg := fmt.Sprintf("  - \"%s\" is a protected system font and cannot be removed (Skipped)", fontName)
							GetLogger().Error("Attempted to remove protected system font: %s", fontName)
							fmt.Println(red(msg))
							status.Skipped++
							continue
						}
						msg := fmt.Sprintf("  - Not found in %s scope", label)
						GetLogger().Info("Font not installed in %s scope: %s", label, fontName)
						fmt.Println(yellow(msg))
						status.Skipped++
						continue
					}

					success := true
					for _, matchingFont := range matchingFonts {
						if isCriticalSystemFont(matchingFont) {
							status.Skipped++
							msg := fmt.Sprintf("  - \"%s\" is a protected system font and cannot be removed (Skipped)", matchingFont)
							GetLogger().Error("Attempted to remove protected system font: %s", matchingFont)
							fmt.Println(red(msg))
							continue
						}
						err := fontManager.RemoveFont(matchingFont, scope)
						if err != nil {
							success = false
							status.Failed++
							msg := fmt.Sprintf("  - \"%s\" (Failed to remove from %s scope) - %v", matchingFont, label, err)
							GetLogger().Error("Failed to remove font %s from %s scope: %v", matchingFont, label, err)
							fmt.Println(red(msg))
						} else {
							status.Removed++
							msg := fmt.Sprintf("  - \"%s\" (Removed from %s scope)", matchingFont, label)
							GetLogger().Info("Successfully removed font: %s from %s scope", matchingFont, label)
							fmt.Println(green(msg))
						}
					}
					if !success {
						status.Failed++
					}
				}
			}

			// Print simple status report
			fmt.Printf("\n%s\n", bold("Status Report"))
			fmt.Println("---------------------------------------------")
			fmt.Printf("%s: %d  |  %s: %d  |  %s: %d\n\n",
				green("Removed"), status.Removed,
				yellow("Skipped"), status.Skipped,
				red("Failed"), status.Failed)

			GetLogger().Info("Removal complete - Removed: %d, Skipped: %d, Failed: %d",
				status.Removed, status.Skipped, status.Failed)

			// Only return error if there were actual removal failures
			if status.Failed > 0 {
				return &FontRemovalError{
					FailedCount: status.Failed,
					TotalCount:  len(fontNames),
				}
			}

			return nil
		}

		// Handle single scope operations (user or machine)
		removedInAnyScope := false
		for _, fontName := range fontNames {
			GetLogger().Info("Processing font: %s", fontName)
			fmt.Printf("\n%s\n", bold(fontName))

			// Track if this font was identified as a protected system font
			protectedFontEncountered := false

			// Special handling for user scope only: check both scopes for better UX
			if len(scopes) == 1 && scopes[0] == platform.UserScope {
				// Check if font exists in machine scope for better user feedback
				machineFonts := findFontFamilyFiles(fontName, fontManager, platform.MachineScope)
				if len(machineFonts) == 0 {
					// Try with search results
					results, err := r.SearchFonts(fontName, "false")
					if err == nil && len(results) > 0 {
						machineFonts = findFontFamilyFiles(results[0].Name, fontManager, platform.MachineScope)
					}
				}

				// Check user scope
				userFonts := findFontFamilyFiles(fontName, fontManager, platform.UserScope)
				if len(userFonts) == 0 {
					// Try with search results
					results, err := r.SearchFonts(fontName, "false")
					if err == nil && len(results) > 0 {
						userFonts = findFontFamilyFiles(results[0].Name, fontManager, platform.UserScope)
					}
				}

				// Handle different scenarios
				if len(userFonts) == 0 && len(machineFonts) == 0 {
					// Font not found in either scope
					msg := fmt.Sprintf("  - \"%s\" is not installed in any scope (Skipped)", fontName)
					GetLogger().Info("Font not installed in any scope: %s", fontName)
					fmt.Println(yellow(msg))
					status.Skipped++
					continue
				} else if len(userFonts) == 0 && len(machineFonts) > 0 {
					// Font only exists in machine scope
					msg := fmt.Sprintf("  - \"%s\" is only installed in machine scope (Skipped)", fontName)
					GetLogger().Info("Font only installed in machine scope: %s", fontName)
					fmt.Println(yellow(msg))
					fmt.Println(cyan("  - Use --scope machine or run as administrator to remove system-wide fonts"))
					status.Skipped++
					continue
				} else if len(userFonts) > 0 && len(machineFonts) > 0 {
					// Font exists in both scopes - remove from user and inform about machine
					fmt.Println(cyan("  - Font also installed in machine scope"))
				}

				// Remove from user scope
				success := true
				for _, matchingFont := range userFonts {
					if isCriticalSystemFont(matchingFont) {
						status.Skipped++
						msg := fmt.Sprintf("  - \"%s\" is a protected system font and cannot be removed (Skipped)", matchingFont)
						GetLogger().Error("Attempted to remove protected system font: %s", matchingFont)
						fmt.Println(red(msg))
						continue
					}
					err := fontManager.RemoveFont(matchingFont, platform.UserScope)
					if err != nil {
						success = false
						status.Failed++
						msg := fmt.Sprintf("  - \"%s\" (Failed to remove from user scope) - %v", matchingFont, err)
						GetLogger().Error("Failed to remove font %s from user scope: %v", matchingFont, err)
						fmt.Println(red(msg))
					} else {
						removedInAnyScope = true
						status.Removed++
						msg := fmt.Sprintf("  - \"%s\" (Removed from user scope)", matchingFont)
						GetLogger().Info("Successfully removed font: %s from user scope", matchingFont)
						fmt.Println(green(msg))
					}
				}
				if !success {
					status.Failed++
				}
			} else {
				// Handle machine scope
				for i, scope := range scopes {
					label := scopeLabel[i]
					GetLogger().Info("Checking scope: %s", label)

					// Check for protected system font first
					protectedFontSkipped := false
					if isCriticalSystemFont(fontName) {
						msg := fmt.Sprintf("  - \"%s\" is a protected system font and cannot be removed (Skipped)", fontName)
						GetLogger().Error("Attempted to remove protected system font: %s", fontName)
						fmt.Println(red(msg))
						status.Skipped++
						protectedFontSkipped = true
						protectedFontEncountered = true
						continue
					}

					// Elevation check for machine scope
					if scope == platform.MachineScope {
						if err := checkElevation(cmd, fontManager, scope); err != nil {
							GetLogger().Error("Elevation check failed: %v", err)
							fmt.Println(red("  - Skipped machine scope due to missing elevation"))
							continue
						}
					}

					matchingFonts := findFontFamilyFiles(fontName, fontManager, scope)
					if len(matchingFonts) == 0 {
						results, err := r.SearchFonts(fontName, "false")
						if err == nil && len(results) > 0 {
							matchingFonts = findFontFamilyFiles(results[0].Name, fontManager, scope)
						}
					}

					if len(matchingFonts) == 0 && !protectedFontSkipped {
						msg := fmt.Sprintf("  - \"%s\" is not installed in %s scope (Skipped)", fontName, label)
						GetLogger().Info("Font not installed in %s scope: %s", label, fontName)
						fmt.Println(yellow(msg))
						status.Skipped++
						continue
					}

					success := true
					for _, matchingFont := range matchingFonts {
						if isCriticalSystemFont(matchingFont) {
							status.Skipped++
							msg := fmt.Sprintf("  - \"%s\" is a protected system font and cannot be removed (Skipped)", matchingFont)
							GetLogger().Error("Attempted to remove protected system font: %s", matchingFont)
							fmt.Println(red(msg))
							continue
						}
						err := fontManager.RemoveFont(matchingFont, scope)
						if err != nil {
							success = false
							status.Failed++
							msg := fmt.Sprintf("  - \"%s\" (Failed to remove from %s scope) - %v", matchingFont, label, err)
							GetLogger().Error("Failed to remove font %s from %s scope: %v", matchingFont, label, err)
							fmt.Println(red(msg))
						} else {
							removedInAnyScope = true
							status.Removed++
							msg := fmt.Sprintf("  - \"%s\" (Removed from %s scope)", matchingFont, label)
							GetLogger().Info("Successfully removed font: %s from %s scope", matchingFont, label)
							fmt.Println(green(msg))
						}
					}
					if !success {
						status.Failed++
					}
				}
			}

			if !removedInAnyScope && !protectedFontEncountered {
				// Suggest the other scope if not found (only for non-user-scope operations)
				if len(scopes) == 1 && scopes[0] != platform.UserScope {
					otherScope := "user"
					if scopes[0] == platform.UserScope {
						otherScope = "machine"
					}
					fmt.Println(cyan(fmt.Sprintf("  - Not found in %s scope. Try --scope %s if you installed it there.", scopeLabel[0], otherScope)))
				}
			}
		}

		// Print status report
		fmt.Printf("\n%s\n", bold("Status Report"))
		fmt.Println("---------------------------------------------")
		fmt.Printf("%s: %d  |  %s: %d  |  %s: %d\n\n",
			green("Removed"), status.Removed,
			yellow("Skipped"), status.Skipped,
			red("Failed"), status.Failed)

		GetLogger().Info("Removal complete - Removed: %d, Skipped: %d, Failed: %d",
			status.Removed, status.Skipped, status.Failed)

		// Only return error if there were actual removal failures
		if status.Failed > 0 {
			return &FontRemovalError{
				FailedCount: status.Failed,
				TotalCount:  len(fontNames),
			}
		}

		return nil
	},
}

// FontRemovalError represents an error that occurred during font removal
type FontRemovalError struct {
	FailedCount int
	TotalCount  int
}

func (e *FontRemovalError) Error() string {
	return fmt.Sprintf("failed to remove %d out of %d fonts", e.FailedCount, e.TotalCount)
}

func init() {
	rootCmd.AddCommand(removeCmd)
	removeCmd.Flags().StringP("scope", "s", "", "Installation scope (user, machine, or all)")
	removeCmd.Flags().BoolP("force", "f", false, "Force removal of critical system fonts")
}
