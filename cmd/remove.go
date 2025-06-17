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

// List of critical system fonts (filenames and families, case-insensitive, no extension)
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
	Use:   "remove <font-name>",
	Short: "Remove a font from your system",
	Long: `Remove a font from your system. You can specify the installation scope using the --scope flag:
  - user (default): Remove font from current user
  - machine: Remove font system-wide (requires elevation)`,
	Example: `  fontget remove "Roboto"
  fontget remove "Open Sans" --scope machine
  fontget remove "roboto, firasans, notosans"`,
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

		// Double check args to prevent panic
		if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
			return nil // Args validator will have already shown the help
		}

		// Create font manager
		fontManager, err := platform.NewFontManager()
		if err != nil {
			GetLogger().Error("Failed to initialize font manager: %v", err)
			return fmt.Errorf("failed to initialize font manager: %w", err)
		}

		// Get scope from flag
		scope, _ := cmd.Flags().GetString("scope")
		GetLogger().Info("Removal parameters - Scope: %s", scope)

		// Convert scope string to InstallationScope
		installScope := platform.UserScope
		if scope != "user" {
			installScope = platform.InstallationScope(scope)
			if installScope != platform.UserScope && installScope != platform.MachineScope {
				GetLogger().Error("Invalid scope '%s'", scope)
				return fmt.Errorf("invalid scope '%s'. Must be 'user' or 'machine'", scope)
			}
		}

		// Check elevation
		if err := checkElevation(cmd, fontManager, installScope); err != nil {
			GetLogger().Error("Elevation check failed: %v", err)
			return err
		}

		// Get font names from args and split by comma
		fontNames := strings.Split(args[0], ",")
		for i, name := range fontNames {
			fontNames[i] = strings.TrimSpace(name)
		}

		GetLogger().Info("Processing %d font(s): %v", len(fontNames), fontNames)

		// Initialize status tracking
		status := RemovalStatus{
			Details: make([]string, 0),
		}

		// Create color functions
		green := color.New(color.FgGreen).SprintFunc()
		yellow := color.New(color.FgYellow).SprintFunc()
		red := color.New(color.FgRed).SprintFunc()
		bold := color.New(color.Bold).SprintFunc()
		cyan := color.New(color.FgCyan).SprintFunc()

		// Get repository for font ID lookup
		r, err := repo.GetRepository()
		if err != nil {
			GetLogger().Error("Failed to initialize repository: %v", err)
			return fmt.Errorf("failed to initialize repository: %w", err)
		}

		// Process each font
		for _, fontName := range fontNames {
			GetLogger().Info("Processing font: %s", fontName)
			fmt.Printf("\n%s\n", bold(fontName))

			// Step 1: Try to find fonts by family name
			matchingFonts := findFontFamilyFiles(fontName, fontManager, installScope)

			// Step 2: If no matches found, try to find by font ID in manifest
			if len(matchingFonts) == 0 {
				// Search for the font in the manifest
				results, err := r.SearchFonts(fontName, "false")
				if err != nil {
					GetLogger().Error("Failed to search manifest: %v", err)
					continue
				}

				// If we found a match in the manifest, try to find the installed files
				if len(results) > 0 {
					// Use the first match's family name to find installed files
					matchingFonts = findFontFamilyFiles(results[0].Name, fontManager, installScope)
				}
			}

			// If still no matches, show suggestions
			if len(matchingFonts) == 0 {
				status.Skipped++
				msg := fmt.Sprintf("  - \"%s\" is not installed (Skipped)", fontName)
				GetLogger().Info("Font not installed: %s", fontName)
				fmt.Println(yellow(msg))

				// Try to find similar installed fonts
				similar := findSimilarInstalledFonts(fontName, fontManager, installScope)
				if len(similar) > 0 {
					GetLogger().Info("Found %d similar installed fonts for %s", len(similar), fontName)
					fmt.Println(cyan("\nDid you mean one of these installed fonts?"))
					for _, s := range similar {
						fmt.Printf("  - %s\n", s)
					}
					fmt.Println() // Add a blank line after suggestions
				}
				continue
			}

			// Remove each matching font
			success := true
			for _, matchingFont := range matchingFonts {
				if isCriticalSystemFont(matchingFont) || isCriticalSystemFont(fontName) {
					status.Skipped++
					msg := fmt.Sprintf("  - \"%s\" is a critical system font and will not be removed (Skipped)", matchingFont)
					GetLogger().Warn("Attempted to remove critical system font: %s", matchingFont)
					fmt.Println(yellow(msg))
					continue
				}
				err := fontManager.RemoveFont(matchingFont, installScope)
				if err != nil {
					success = false
					status.Failed++
					msg := fmt.Sprintf("  - \"%s\" (Failed to remove) - %v", matchingFont, err)
					GetLogger().Error("Failed to remove font %s: %v", matchingFont, err)
					fmt.Println(red(msg))
				} else {
					status.Removed++
					msg := fmt.Sprintf("  - \"%s\" (Removed)", matchingFont)
					GetLogger().Info("Successfully removed font: %s", matchingFont)
					fmt.Println(green(msg))
				}
			}

			if !success {
				status.Failed++
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
			return fmt.Errorf("one or more fonts failed to remove")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
	removeCmd.Flags().String("scope", "user", "Installation scope (user or machine)")
}
