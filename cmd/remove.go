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

		// For --all scope, require elevation upfront
		if len(scopes) == 2 {
			// Check elevation first
			if err := checkElevation(cmd, fontManager, platform.MachineScope); err != nil {
				if errors.Is(err, ErrElevationRequired) {
					return nil // Already printed user-friendly message
				}
				GetLogger().Error("Elevation check failed for --scope all: %v", err)
				fmt.Println(ui.RenderError("This operation requires administrator privileges."))
				fmt.Println("To run as administrator:")
				fmt.Println("  1. Right-click on Command Prompt or PowerShell.")
				fmt.Println("  2. Select 'Run as administrator'.")
				fmt.Printf("  3. Run: fontget remove --scope all %s\n", strings.Join(fontNames, " "))
				return fmt.Errorf("elevation required for --scope all")
			}

			// Process fonts with simple single-status-report approach
			for _, fontName := range fontNames {
				GetLogger().Info("Processing font: %s", fontName)
				output.GetVerbose().Info("Processing font: %s", fontName)
				output.GetDebug().State("Starting font removal process for: %s", fontName)

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
						msg := fmt.Sprintf("  - Not found in %s scope", label)
						GetLogger().Info("Font not installed in %s scope: %s", label, fontName)
						output.GetVerbose().Info("Font not found in %s scope: %s", label, fontName)
						output.GetDebug().State("No font files found for %s in %s scope", fontName, label)
						fmt.Println(ui.TableSourceName.Render(msg))
						status.Skipped++
						continue
					}

					success := true
					output.GetVerbose().Info("Found %d font files to process in %s scope", len(matchingFonts), label)
					for _, matchingFont := range matchingFonts {
						output.GetDebug().State("Processing font file: %s", matchingFont)

						if isCriticalSystemFont(matchingFont) {
							status.Skipped++
							msg := fmt.Sprintf("  ✓ \"%s\" (%s)", matchingFont, ui.FeedbackWarning.Render("Skipped - protected system font"))
							GetLogger().Error("Attempted to remove protected system font: %s", matchingFont)
							output.GetVerbose().Warning("Skipping protected system font: %s", matchingFont)
							output.GetDebug().State("Font %s is in critical system font list", matchingFont)
							fmt.Println(ui.ContentText.Render(msg))
							continue
						}

						// Use spinner for removal operation
						removeMsg := fmt.Sprintf("Removing %s from %s scope...", matchingFont, label)
						output.GetVerbose().Info("Removing font file: %s from %s scope", matchingFont, label)
						err := runSpinner(removeMsg, "", func() error {
							return fontManager.RemoveFont(matchingFont, scope)
						})

						if err != nil {
							success = false
							status.Failed++
							msg := fmt.Sprintf("  ✗ \"%s\" (%s) - %v", matchingFont, ui.FeedbackError.Render("Failed"), err)
							GetLogger().Error("Failed to remove font %s from %s scope: %v", matchingFont, label, err)
							output.GetVerbose().Error("Failed to remove font %s: %v", matchingFont, err)
							output.GetDebug().Error("Font removal failed for %s in %s scope: %v", matchingFont, label, err)
							fmt.Println(ui.RenderError(msg))
						} else {
							status.Removed++
							msg := fmt.Sprintf("  ✓ \"%s\" (%s from %s scope)", matchingFont, ui.FeedbackSuccess.Render("Removed"), label)
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
					fmt.Println(ui.TableSourceName.Render(msg))
					status.Skipped++
					continue
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

				// Remove from user scope
				success := true
				for _, matchingFont := range userFonts {
					if isCriticalSystemFont(matchingFont) {
						status.Skipped++
						msg := fmt.Sprintf("  ✓ \"%s\" (%s)", matchingFont, ui.FeedbackWarning.Render("Skipped - protected system font"))
						GetLogger().Error("Attempted to remove protected system font: %s", matchingFont)
						fmt.Println(ui.ContentText.Render(msg))
						continue
					}

					// Use spinner for removal operation
					removeMsg := fmt.Sprintf("Removing %s from user scope...", matchingFont)
					err := runSpinner(removeMsg, "", func() error {
						return fontManager.RemoveFont(matchingFont, platform.UserScope)
					})

					if err != nil {
						success = false
						status.Failed++
						msg := fmt.Sprintf("  ✗ \"%s\" (%s) - %v", matchingFont, ui.FeedbackError.Render("Failed"), err)
						GetLogger().Error("Failed to remove font %s from user scope: %v", matchingFont, err)
						fmt.Println(ui.RenderError(msg))
					} else {
						removedInAnyScope = true
						status.Removed++
						msg := fmt.Sprintf("  ✓ \"%s\" (%s from user scope)", matchingFont, ui.FeedbackSuccess.Render("Removed"))
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
								fmt.Println(ui.RenderError("  - Skipped machine scope due to missing elevation"))
								continue
							}
							GetLogger().Error("Elevation check failed: %v", err)
							fmt.Println(ui.RenderError("  - Skipped machine scope due to missing elevation"))
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
						fmt.Println(ui.TableSourceName.Render(msg))
						status.Skipped++
						continue
					}

					success := true
					for _, matchingFont := range matchingFonts {
						if isCriticalSystemFont(matchingFont) {
							status.Skipped++
							msg := fmt.Sprintf("  ✓ \"%s\" (%s)", matchingFont, ui.FeedbackWarning.Render("Skipped - protected system font"))
							GetLogger().Error("Attempted to remove protected system font: %s", matchingFont)
							fmt.Println(ui.ContentText.Render(msg))
							continue
						}

						// Use spinner for removal operation
						removeMsg := fmt.Sprintf("Removing %s from %s scope...", matchingFont, label)
						err := runSpinner(removeMsg, "", func() error {
							return fontManager.RemoveFont(matchingFont, scope)
						})

						if err != nil {
							success = false
							status.Failed++
							msg := fmt.Sprintf("  ✗ \"%s\" (%s) - %v", matchingFont, ui.FeedbackError.Render("Failed"), err)
							GetLogger().Error("Failed to remove font %s from %s scope: %v", matchingFont, label, err)
							fmt.Println(ui.RenderError(msg))
						} else {
							removedInAnyScope = true
							status.Removed++
							msg := fmt.Sprintf("  ✓ \"%s\" (%s from %s scope)", matchingFont, ui.FeedbackSuccess.Render("Removed"), label)
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
// FontRemovalError is now defined in shared.go

func init() {
	rootCmd.AddCommand(removeCmd)
	removeCmd.Flags().StringP("scope", "s", "", "Installation scope (user, machine, or all)")
	removeCmd.Flags().BoolP("force", "f", false, "Force removal of critical system fonts")
}
