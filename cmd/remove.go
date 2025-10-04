package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fontget/internal/output"
	"fontget/internal/platform"
	"fontget/internal/repo"
	"fontget/internal/ui"

	"github.com/spf13/cobra"
)

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

// extractFontDisplayNameFromFilename converts a font filename to a proper display name
// e.g., "RobotoMono-Bold.ttf" -> "Roboto Mono Bold"
func extractFontDisplayNameFromFilename(filename string) string {
	// Get the base filename without extension
	baseName := filepath.Base(filename)
	ext := filepath.Ext(baseName)
	fileName := strings.TrimSuffix(baseName, ext)

	// Handle fonts with dashes separating base name from variant
	if strings.Contains(fileName, "-") {
		parts := strings.Split(fileName, "-")
		if len(parts) >= 2 {
			baseFontName := parts[0]
			variantPart := strings.Join(parts[1:], "-")

			// Convert camelCase to proper spacing for base name
			// e.g., "RobotoMono" -> "Roboto Mono"
			baseDisplayName := convertCamelCaseToSpaced(baseFontName)

			// Convert variant to proper case
			// e.g., "BoldItalic" -> "BoldItalic" (already proper)
			variantDisplay := convertCamelCaseToSpaced(variantPart)

			return fmt.Sprintf("%s %s", baseDisplayName, variantDisplay)
		}
	}

	// If no dashes, just convert camelCase to spaced
	return convertCamelCaseToSpaced(fileName)
}

// convertCamelCaseToSpaced converts camelCase to spaced format
// e.g., "RobotoMono" -> "Roboto Mono"
func convertCamelCaseToSpaced(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, ' ')
		}
		result = append(result, r)
	}
	return string(result)
}

// showInstalledFontNotFoundWithSuggestions displays font not found error with suggestions for installed fonts
func showInstalledFontNotFoundWithSuggestions(fontName string, fontManager platform.FontManager) {
	fmt.Printf("\n%s\n", ui.FeedbackError.Render(fmt.Sprintf("Font '%s' not found.", fontName)))

	// Get all installed fonts to suggest alternatives
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

	// Find similar fonts
	similar := findSimilarInstalledFonts(fontName, installedFonts)

	// If no similar fonts found, show general guidance
	if len(similar) == 0 {
		fmt.Printf("%s\n", ui.FeedbackText.Render("Try using the list command to see installed fonts."))
		fmt.Printf("\n%s\n", ui.FeedbackText.Render("Example:"))
		fmt.Printf("  %s\n", ui.CommandExample.Render("fontget list"))
		fmt.Println()
		return
	}

	// Display similar fonts in a clean format
	fmt.Printf("%s\n\n", ui.FeedbackText.Render("Did you mean one of these installed fonts?"))

	for _, font := range similar {
		fmt.Printf("  - %s\n", ui.TableSourceName.Render(font))
	}

	fmt.Printf("\n%s\n", ui.FeedbackText.Render("Use the exact font name to remove it:"))
	fmt.Printf("  %s\n", ui.CommandExample.Render(fmt.Sprintf("fontget remove \"%s\"", similar[0])))
	fmt.Println()
}

// findSimilarInstalledFonts returns a list of installed font names that are similar to the given name
func findSimilarInstalledFonts(fontName string, installedFonts []string) []string {
	queryLower := strings.ToLower(fontName)
	queryNorm := strings.ReplaceAll(queryLower, " ", "")
	queryNorm = strings.ReplaceAll(queryNorm, "-", "")
	queryNorm = strings.ReplaceAll(queryNorm, "_", "")

	var similar []string
	seen := make(map[string]bool)

	// Simple substring matching for speed
	for _, font := range installedFonts {
		if len(similar) >= 5 { // Limit to 5 suggestions
			break
		}

		fontLower := strings.ToLower(font)
		fontNorm := strings.ReplaceAll(fontLower, " ", "")
		fontNorm = strings.ReplaceAll(fontNorm, "-", "")
		fontNorm = strings.ReplaceAll(fontNorm, "_", "")

		// Skip exact equals and already found fonts
		if fontLower == queryLower || fontNorm == queryNorm || seen[font] {
			continue
		}

		if strings.Contains(fontLower, queryLower) || strings.Contains(queryLower, fontLower) {
			similar = append(similar, font)
			seen[font] = true
		}
	}

	// If no substring matches and we still need more, try partial word matches
	if len(similar) < 5 {
		words := strings.Fields(queryLower)
		for _, font := range installedFonts {
			if len(similar) >= 5 || seen[font] {
				break
			}

			fontLower := strings.ToLower(font)
			for _, word := range words {
				if len(word) > 2 && strings.Contains(fontLower, word) {
					similar = append(similar, font)
					seen[font] = true
					break
				}
			}
		}
	}

	return similar
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
	similar := findSimilarInstalledFonts(fontName, installedFonts)

	// If no similar fonts found, show general guidance
	if len(similar) == 0 {
		fmt.Printf("%s\n", ui.FeedbackText.Render("Try using the list command to see installed fonts."))
		fmt.Printf("\n%s\n", ui.FeedbackText.Render("Example:"))
		fmt.Printf("  %s\n", ui.CommandExample.Render("fontget list"))
		fmt.Println()
		return
	}

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
		output.GetDebug().Message("Debug mode enabled - showing detailed diagnostic information")

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

		// Verbose-level information for users
		output.GetVerbose().Info("Removal parameters - Scope: %s, Force: %v", scopeFlag, forceFlag)
		output.GetVerbose().Info("Processing %d font(s): %v", len(args), args)

		GetLogger().Info("Processing %d font(s): %v", len(args), args)

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
				output.GetVerbose().Info("Auto-detected elevated privileges, defaulting to 'all' scope")
				fmt.Println(ui.FormLabel.Render("Auto-detected administrator privileges - removing from all scopes"))
			} else {
				scopeFlag = "user"
				GetLogger().Info("Auto-detected user privileges, defaulting to 'user' scope")
				output.GetVerbose().Info("Auto-detected user privileges, defaulting to 'user' scope")
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
		fontNames := ParseFontNames(args)

		GetLogger().Info("Processing %d font(s): %v", len(fontNames), fontNames)

		// Cache installed fonts for suggestions (similar to add command)
		var installedFontsCache []string
		if len(fontNames) > 0 {
			// Only build cache if we might need suggestions
			installedFontsCache = buildInstalledFontsCache(fontManager)
		}

		// Note: Header will be shown only for successful operations, not for not found cases
		// This matches the add command behavior

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

			// Process fonts with simple single-status-report approach
			for _, fontName := range fontNames {
				GetLogger().Info("Processing font: %s", fontName)
				output.GetVerbose().Info("Processing font: %s", fontName)
				output.GetDebug().State("Starting font removal process for: %s", fontName)

				// Show removal header for successful operations
				fmt.Println()
				headerMessage := fmt.Sprintf("Removing '%s' from system", fontName)
				fmt.Println(ui.FeedbackInfo.Render(headerMessage))

				for i, scope := range scopes {
					label := scopeLabel[i]
					GetLogger().Info("Checking scope: %s", label)
					output.GetVerbose().Info("Checking scope: %s", label)
					output.GetDebug().State("Checking font files in %s scope", label)

					matchingFonts := findFontFamilyFiles(fontName, fontManager, scope)
					output.GetDebug().State("Found %d matching font files in %s scope", len(matchingFonts), label)

					if len(matchingFonts) == 0 {
						output.GetVerbose().Info("No direct matches found, searching repository for: %s", fontName)
						results, err := r.SearchFonts(fontName, "false")
						if err == nil && len(results) > 0 {
							output.GetVerbose().Info("Found %d search results, using first match: %s", len(results), results[0].Name)
							matchingFonts = findFontFamilyFiles(results[0].Name, fontManager, scope)
							output.GetDebug().State("After repository search, found %d matching font files", len(matchingFonts))
						}
					}

					if len(matchingFonts) == 0 {
						if isCriticalSystemFont(fontName) {
							msg := fmt.Sprintf("  - \"%s\" is a protected system font and cannot be removed (Skipped)", fontName)
							GetLogger().Error("Attempted to remove protected system font: %s", fontName)
							output.GetVerbose().Warning("Protected system font detected: %s", fontName)
							output.GetDebug().State("Font %s is in critical system font list", fontName)
							fmt.Println(ui.RenderError(msg))
							status.Skipped++
							continue
						}
						// Font not found in this scope - show suggestions and return early
						GetLogger().Info("Font not installed in %s scope: %s", label, fontName)
						output.GetVerbose().Info("Font not found in %s scope: %s", label, fontName)
						output.GetDebug().State("No font files found for %s in %s scope", fontName, label)
						showInstalledFontNotFoundWithSuggestionsCached(fontName, installedFontsCache)
						// Don't show status report for not found fonts (like add command)
						return nil
					}

					// Show removal header for successful operations (like add command)
					fmt.Println()
					headerMessage := fmt.Sprintf("Removing '%s' from system", fontName)
					fmt.Println(ui.FeedbackInfo.Render(headerMessage))

					success := true
					output.GetVerbose().Info("Found %d font files to process in %s scope", len(matchingFonts), label)
					for _, matchingFont := range matchingFonts {
						output.GetDebug().State("Processing font file: %s", matchingFont)

						if isCriticalSystemFont(matchingFont) {
							status.Skipped++
							fontDisplayName := extractFontDisplayNameFromFilename(matchingFont)
							msg := fmt.Sprintf("  %s %s - %s", ui.FeedbackSuccess.Render("✓"), ui.TableRow.Render(fontDisplayName), ui.FeedbackWarning.Render("[Skipped] protected system font"))
							GetLogger().Error("Attempted to remove protected system font: %s", matchingFont)
							output.GetVerbose().Warning("Skipping protected system font: %s", matchingFont)
							output.GetDebug().State("Font %s is in critical system font list", matchingFont)
							fmt.Println(ui.ContentText.Render(msg))
							continue
						}

						// Extract proper font name and variant from the font file
						fontDisplayName := extractFontDisplayNameFromFilename(matchingFont)

						// Remove font directly without spinner
						output.GetVerbose().Info("Removing font file: %s from %s scope", matchingFont, label)
						err := fontManager.RemoveFont(matchingFont, scope)

						if err != nil {
							success = false
							status.Failed++

							// Provide more specific error messages
							var errorMsg string
							if strings.Contains(strings.ToLower(err.Error()), "in use") ||
								strings.Contains(strings.ToLower(err.Error()), "access denied") ||
								strings.Contains(strings.ToLower(err.Error()), "permission") {
								errorMsg = "[Failed] font is in use or access denied"
							} else {
								errorMsg = "[Failed] to remove existing font"
							}

							msg := fmt.Sprintf("  %s %s - %s", ui.FeedbackError.Render("✗"), ui.TableRow.Render(fontDisplayName), ui.FeedbackError.Render(errorMsg))
							GetLogger().Error("Failed to remove font %s from %s scope: %v", matchingFont, label, err)
							output.GetVerbose().Error("Failed to remove font %s: %v", matchingFont, err)
							output.GetDebug().Error("Font removal failed for %s in %s scope: %v", matchingFont, label, err)
							fmt.Println(ui.ContentText.Render(msg))
						} else {
							status.Removed++
							msg := fmt.Sprintf("  %s %s - %s", ui.FeedbackSuccess.Render("✓"), ui.TableRow.Render(fontDisplayName), ui.FeedbackSuccess.Render("[Removed] to "+label+" scope"))
							GetLogger().Info("Successfully removed font: %s from %s scope", matchingFont, label)
							output.GetVerbose().Success("Successfully removed font: %s from %s scope", matchingFont, label)
							output.GetDebug().State("Font removal completed successfully for %s", matchingFont)
							fmt.Println(ui.ContentText.Render(msg))
						}
					}
					if !success {
						status.Failed++
					}
				}
			}

			// Print status report only if there were actual operations
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

			// Don't return error for removal failures since we already show detailed status report
			// This prevents duplicate error messages while maintaining proper exit codes
			return nil
		}

		// Handle single scope operations (user or machine)
		removedInAnyScope := false
		for _, fontName := range fontNames {
			GetLogger().Info("Processing font: %s", fontName)

			// Track if this font was identified as a protected system font
			protectedFontEncountered := false

			// Special handling for user scope only: check both scopes for better UX
			if len(scopes) == 1 && scopes[0] == platform.UserScope {
				// Check both scopes efficiently
				machineFonts := findFontFamilyFiles(fontName, fontManager, platform.MachineScope)
				userFonts := findFontFamilyFiles(fontName, fontManager, platform.UserScope)

				// If no direct matches, try repository search once
				if len(machineFonts) == 0 && len(userFonts) == 0 {
					results, err := r.SearchFonts(fontName, "false")
					if err == nil && len(results) > 0 {
						// Try with the first search result
						searchName := results[0].Name
						machineFonts = findFontFamilyFiles(searchName, fontManager, platform.MachineScope)
						userFonts = findFontFamilyFiles(searchName, fontManager, platform.UserScope)
					}
				}

				// Handle different scenarios
				if len(userFonts) == 0 && len(machineFonts) == 0 {
					// Font not found in either scope - show suggestions and return early
					GetLogger().Info("Font not installed in any scope: %s", fontName)
					showInstalledFontNotFoundWithSuggestionsCached(fontName, installedFontsCache)
					// Don't show status report for not found fonts (like add command)
					return nil
				} else if len(userFonts) == 0 && len(machineFonts) > 0 {
					// Font only exists in machine scope
					msg := fmt.Sprintf("  - \"%s\" is only installed in machine scope (Skipped)", fontName)
					GetLogger().Info("Font only installed in machine scope: %s", fontName)
					fmt.Println(ui.TableSourceName.Render(msg))
					fmt.Println(ui.FormLabel.Render("  - Use --scope machine or run as administrator to remove system-wide fonts"))
					status.Skipped++
					continue
				} else if len(userFonts) > 0 && len(machineFonts) > 0 {
					// Font exists in both scopes - remove from user and inform about machine
					fmt.Println(ui.FormLabel.Render("  - Font also installed in machine scope"))
				}

				// Show removal header for successful operations (like add command)
				fmt.Println()
				headerMessage := fmt.Sprintf("Removing '%s' from system", fontName)
				fmt.Println(ui.FeedbackInfo.Render(headerMessage))

				// Remove from user scope
				success := true
				for _, matchingFont := range userFonts {
					if isCriticalSystemFont(matchingFont) {
						status.Skipped++
						fontDisplayName := extractFontDisplayNameFromFilename(matchingFont)
						msg := fmt.Sprintf("  %s %s - %s", ui.FeedbackSuccess.Render("✓"), ui.TableRow.Render(fontDisplayName), ui.FeedbackWarning.Render("[Skipped] protected system font"))
						GetLogger().Error("Attempted to remove protected system font: %s", matchingFont)
						fmt.Println(ui.ContentText.Render(msg))
						continue
					}

					// Extract proper font name and variant from the font file
					fontDisplayName := extractFontDisplayNameFromFilename(matchingFont)

					// Remove font directly without spinner
					err := fontManager.RemoveFont(matchingFont, platform.UserScope)

					if err != nil {
						success = false
						status.Failed++

						// Provide more specific error messages
						var errorMsg string
						if strings.Contains(strings.ToLower(err.Error()), "in use") ||
							strings.Contains(strings.ToLower(err.Error()), "access denied") ||
							strings.Contains(strings.ToLower(err.Error()), "permission") {
							errorMsg = "[Failed] font is in use or access denied"
						} else {
							errorMsg = "[Failed] to remove existing font"
						}

						msg := fmt.Sprintf("  %s %s - %s", ui.FeedbackError.Render("✗"), ui.TableRow.Render(fontDisplayName), ui.FeedbackError.Render(errorMsg))
						GetLogger().Error("Failed to remove font %s from user scope: %v", matchingFont, err)
						output.GetVerbose().Error("Failed to remove font %s: %v", matchingFont, err)
						fmt.Println(ui.ContentText.Render(msg))
					} else {
						removedInAnyScope = true
						status.Removed++
						msg := fmt.Sprintf("  %s %s - %s", ui.FeedbackSuccess.Render("✓"), ui.TableRow.Render(fontDisplayName), ui.FeedbackSuccess.Render("[Removed] from user scope"))
						GetLogger().Info("Successfully removed font: %s from user scope", matchingFont)
						fmt.Println(ui.ContentText.Render(msg))
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
						fmt.Println(ui.RenderError(msg))
						status.Skipped++
						protectedFontSkipped = true
						protectedFontEncountered = true
						continue
					}

					// Elevation check for machine scope
					if scope == platform.MachineScope {
						if err := checkElevation(cmd, fontManager, scope); err != nil {
							if errors.Is(err, ErrElevationRequired) {
								return nil // Already printed user-friendly message
							}
							GetLogger().Error("Elevation check failed: %v", err)
							return err
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
						// Font not found in this scope - show suggestions and return early
						GetLogger().Info("Font not installed in %s scope: %s", label, fontName)
						showInstalledFontNotFoundWithSuggestionsCached(fontName, installedFontsCache)
						// Don't show status report for not found fonts (like add command)
						return nil
					}

					// Show removal header for successful operations (like add command)
					fmt.Println()
					headerMessage := fmt.Sprintf("Removing '%s' from system", fontName)
					fmt.Println(ui.FeedbackInfo.Render(headerMessage))

					success := true
					for _, matchingFont := range matchingFonts {
						if isCriticalSystemFont(matchingFont) {
							status.Skipped++
							fontDisplayName := extractFontDisplayNameFromFilename(matchingFont)
							msg := fmt.Sprintf("  %s %s - %s", ui.FeedbackSuccess.Render("✓"), ui.TableRow.Render(fontDisplayName), ui.FeedbackWarning.Render("[Skipped] protected system font"))
							GetLogger().Error("Attempted to remove protected system font: %s", matchingFont)
							fmt.Println(ui.ContentText.Render(msg))
							continue
						}

						// Extract proper font name and variant from the font file
						fontDisplayName := extractFontDisplayNameFromFilename(matchingFont)

						// Remove font directly without spinner
						err := fontManager.RemoveFont(matchingFont, scope)

						if err != nil {
							success = false
							status.Failed++

							// Provide more specific error messages
							var errorMsg string
							if strings.Contains(strings.ToLower(err.Error()), "in use") ||
								strings.Contains(strings.ToLower(err.Error()), "access denied") ||
								strings.Contains(strings.ToLower(err.Error()), "permission") {
								errorMsg = "[Failed] font is in use or access denied"
							} else {
								errorMsg = "[Failed] to remove existing font"
							}

							msg := fmt.Sprintf("  %s %s - %s", ui.FeedbackError.Render("✗"), ui.TableRow.Render(fontDisplayName), ui.FeedbackError.Render(errorMsg))
							GetLogger().Error("Failed to remove font %s from %s scope: %v", matchingFont, label, err)
							output.GetVerbose().Error("Failed to remove font %s: %v", matchingFont, err)
							fmt.Println(ui.ContentText.Render(msg))
						} else {
							removedInAnyScope = true
							status.Removed++
							msg := fmt.Sprintf("  %s %s - %s", ui.FeedbackSuccess.Render("✓"), ui.TableRow.Render(fontDisplayName), ui.FeedbackSuccess.Render("[Removed] from "+label+" scope"))
							GetLogger().Info("Successfully removed font: %s from %s scope", matchingFont, label)
							fmt.Println(ui.ContentText.Render(msg))
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
					fmt.Println(ui.FormLabel.Render(fmt.Sprintf("  - Not found in %s scope. Try --scope %s if you installed it there.", scopeLabel[0], otherScope)))
				}
			}
		}

		GetLogger().Info("Removal complete - Removed: %d, Skipped: %d, Failed: %d",
			status.Removed, status.Skipped, status.Failed)

		// Print status report only if there were actual operations
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
	},
}

// FontRemovalError represents an error that occurred during font removal
// FontRemovalError is now defined in shared.go

func init() {
	rootCmd.AddCommand(removeCmd)
	removeCmd.Flags().StringP("scope", "s", "", "Installation scope (user, machine, or all)")
	removeCmd.Flags().BoolP("force", "f", false, "Force removal of critical system fonts")
}
