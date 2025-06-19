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
	"georgia":       true,
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
	Use:     "remove <font-name>",
	Aliases: []string{"uninstall"},
	Short:   "Remove a font from your system",
	Long: `Remove a font from your system. You can specify the installation scope using the --scope flag:
  - user (default): Remove font from current user
  - machine: Remove font system-wide (requires elevation)
  - all: Remove from both user and machine scopes`,
	Example: `  fontget remove "Roboto"
  fontget remove "Open Sans" --scope machine
  fontget remove "roboto, firasans, notosans"
  fontget remove "Roboto" --scope all`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
			red := color.New(color.FgRed).SprintFunc()
			fmt.Printf("\n%s\n\n", red("A font name is required"))
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
		GetLogger().Info("Removal parameters - Scope: %s", scopeFlag)

		// Determine which scopes to check
		var scopes []platform.InstallationScope
		var scopeLabel []string
		if scopeFlag == "all" || scopeFlag == "" {
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

		// Get font names from args and split by comma
		fontNames := strings.Split(args[0], ",")
		for i, name := range fontNames {
			fontNames[i] = strings.TrimSpace(name)
		}

		GetLogger().Info("Processing %d font(s): %v", len(fontNames), fontNames)

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

		// Track per-scope results
		perScopeResults := make(map[string][]string)

		for _, fontName := range fontNames {
			GetLogger().Info("Processing font: %s", fontName)
			fmt.Printf("\n%s\n", bold(fontName))

			removedInAnyScope := false
			for i, scope := range scopes {
				label := scopeLabel[i]
				GetLogger().Info("Checking scope: %s", label)

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

				if len(matchingFonts) == 0 {
					msg := fmt.Sprintf("  - \"%s\" is not installed in %s scope (Skipped)", fontName, label)
					GetLogger().Info("Font not installed in %s scope: %s", label, fontName)
					fmt.Println(yellow(msg))
					perScopeResults[label] = append(perScopeResults[label], msg)
					continue
				}

				success := true
				for _, matchingFont := range matchingFonts {
					if isCriticalSystemFont(matchingFont) || isCriticalSystemFont(fontName) {
						msg := fmt.Sprintf("  - \"%s\" is a critical system font and will not be removed (Skipped)", matchingFont)
						GetLogger().Warn("Attempted to remove critical system font: %s", matchingFont)
						fmt.Println(yellow(msg))
						perScopeResults[label] = append(perScopeResults[label], msg)
						continue
					}
					err := fontManager.RemoveFont(matchingFont, scope)
					if err != nil {
						success = false
						msg := fmt.Sprintf("  - \"%s\" (Failed to remove from %s scope) - %v", matchingFont, label, err)
						GetLogger().Error("Failed to remove font %s from %s scope: %v", matchingFont, label, err)
						fmt.Println(red(msg))
						perScopeResults[label] = append(perScopeResults[label], msg)
					} else {
						removedInAnyScope = true
						msg := fmt.Sprintf("  - \"%s\" (Removed from %s scope)", matchingFont, label)
						GetLogger().Info("Successfully removed font: %s from %s scope", matchingFont, label)
						fmt.Println(green(msg))
						perScopeResults[label] = append(perScopeResults[label], msg)
					}
				}
				if !success {
					status.Failed++
				}
			}

			if !removedInAnyScope {
				// Suggest the other scope if not found
				if len(scopes) == 1 {
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
		for _, label := range scopeLabel {
			if msgs, ok := perScopeResults[label]; ok {
				fmt.Printf("%s scope:\n", strings.Title(label))
				for _, msg := range msgs {
					fmt.Println(msg)
				}
			}
		}
		fmt.Printf("%s: %d  |  %s: %d  |  %s: %d\n\n",
			green("Removed"), status.Removed,
			yellow("Skipped"), status.Skipped,
			red("Failed"), status.Failed)

		GetLogger().Info("Removal complete - Removed: %d, Skipped: %d, Failed: %d",
			status.Removed, status.Skipped, status.Failed)

		if status.Failed > 0 {
			return fmt.Errorf("one or more fonts failed to remove")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
	removeCmd.Flags().String("scope", "all", "Installation scope (user, machine, or all)")
}
