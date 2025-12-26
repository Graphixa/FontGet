package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"fontget/internal/cmdutils"
	"fontget/internal/components"
	"fontget/internal/config"
	"fontget/internal/output"
	"fontget/internal/platform"
	"fontget/internal/repo"
	"fontget/internal/shared"
	"fontget/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// ImportResult tracks the result of importing fonts
type ImportResult struct {
	Success int
	Skipped int
	Failed  int
	Errors  []string
}

var importCmd = &cobra.Command{
	Use:   "import <manifest-file>",
	Short: "Import fonts from an export manifest file",
	Long: `Import fonts from a FontGet export manifest file.

The manifest file should be a JSON file created by the 'export' command.
Fonts will be installed using their Font IDs, and missing fonts will be skipped with a warning.`,
	Example: `  fontget import fonts.json
  fontget import fonts.json --scope user
  fontget import fonts.json --force`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			fmt.Printf("\n%s\n", ui.RenderError("A manifest file is required"))
			fmt.Printf("Use 'fontget import --help' for more information.\n\n")
			return nil
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		GetLogger().Info("Starting font import operation")

		// Always start with a blank line for consistent spacing
		fmt.Println()

		// Ensure manifest system is initialized
		if err := cmdutils.EnsureManifestInitialized(func() cmdutils.Logger { return GetLogger() }); err != nil {
			return err
		}

		manifestFile := args[0]

		// Check if file exists
		if _, err := os.Stat(manifestFile); os.IsNotExist(err) {
			return fmt.Errorf("manifest file not found: %s", manifestFile)
		}

		// Read manifest file
		data, err := os.ReadFile(manifestFile)
		if err != nil {
			return fmt.Errorf("unable to read manifest file: %v", err)
		}

		var exportManifest ExportManifest
		if err := json.Unmarshal(data, &exportManifest); err != nil {
			return fmt.Errorf("unable to parse manifest file: %v", err)
		}

		// Validate manifest structure
		if exportManifest.Version == "" {
			return fmt.Errorf("invalid manifest: missing version")
		}
		if len(exportManifest.Fonts) == 0 {
			return fmt.Errorf("manifest contains no fonts to import")
		}

		// Get flags
		scope, _ := cmd.Flags().GetString("scope")
		force, _ := cmd.Flags().GetBool("force")

		// Create font manager
		fontManager, err := cmdutils.CreateFontManager(func() cmdutils.Logger { return GetLogger() })
		if err != nil {
			return err
		}

		// Auto-detect scope if not provided
		if scope == "" {
			var err error
			scope, err = platform.AutoDetectScope(fontManager, "user", "machine", GetLogger())
			if err != nil {
				// Should not happen, but handle gracefully
				scope = "user"
			}
		}

		// Convert scope string to InstallationScope
		installScope := platform.UserScope
		if scope != "user" {
			installScope = platform.InstallationScope(scope)
			if installScope != platform.UserScope && installScope != platform.MachineScope {
				err := fmt.Errorf("invalid scope '%s'. Valid options are: 'user' or 'machine'", scope)
				output.GetVerbose().Error("%v", err)
				output.GetDebug().Error("Invalid scope provided: '%s'", scope)
				return err
			}
		}

		// Check elevation
		if err := cmdutils.CheckElevation(cmd, fontManager, installScope); err != nil {
			if errors.Is(err, cmdutils.ErrElevationRequired) {
				return nil
			}
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("checkElevation() failed: %v", err)
			return fmt.Errorf("unable to verify system permissions: %v", err)
		}

		// Get font directory
		fontDir := fontManager.GetFontDir(installScope)

		// Verbose output
		output.GetVerbose().Info("Importing fonts from: %s", manifestFile)
		output.GetVerbose().Info("Manifest version: %s", exportManifest.Version)
		output.GetVerbose().Info("Exported at: %s", exportManifest.ExportedAt)
		output.GetVerbose().Info("Total fonts in manifest: %d", len(exportManifest.Fonts))
		output.GetVerbose().Info("Scope: %s", scope)
		output.GetVerbose().Info("Force mode: %v", force)
		output.GetVerbose().Info("Installing to: %s", fontDir)
		// Verbose section ends with blank line per spacing framework (only if verbose was shown)
		if IsVerbose() {
			fmt.Println()
		}

		// Load config manifest early to check source availability
		configManifest, err := config.LoadManifest()
		if err != nil {
			output.GetVerbose().Warning("Failed to load config manifest: %v", err)
			output.GetDebug().Error("config.LoadManifest() failed: %v", err)
		}

		// Collect fonts to install - deduplicate by Font ID (handles Nerd Fonts with multiple families per Font ID)
		type fontInstallInfo struct {
			fonts       []repo.FontFile
			sourceName  string
			familyNames []string
		}

		fontsByID := make(map[string]*fontInstallInfo)
		var notFoundFonts []string
		var invalidFonts []string

		// Track fonts by source for availability detection
		type sourceFontInfo struct {
			fontID      string
			familyNames []string
		}
		fontsBySource := make(map[string][]sourceFontInfo) // source name -> list of fonts

		// Track fonts that require disabled or missing sources
		var missingSourceFonts = make(map[string][]string)  // source name -> font names
		var disabledSourceFonts = make(map[string][]string) // source name -> font names
		var builtInSources = map[string]bool{
			"Google Fonts":  true,
			"Nerd Fonts":    true,
			"Font Squirrel": true,
		}

		for _, exportedFont := range exportManifest.Fonts {
			// Skip fonts without Font IDs
			if exportedFont.FontID == "" {
				// Handle both old format (FamilyName) and new format (FamilyNames)
				if len(exportedFont.FamilyNames) > 0 {
					invalidFonts = append(invalidFonts, strings.Join(exportedFont.FamilyNames, ", "))
				} else if exportedFont.FamilyName != "" {
					invalidFonts = append(invalidFonts, exportedFont.FamilyName)
				}
				continue
			}

			// Skip if we've already processed this Font ID (deduplication)
			if _, exists := fontsByID[exportedFont.FontID]; exists {
				output.GetDebug().State("Skipping duplicate Font ID: %s", exportedFont.FontID)
				continue
			}

			// Get source name from export file
			sourceName := exportedFont.Source
			if sourceName == "" {
				sourceName = shared.GetSourceNameFromID(exportedFont.FontID)
			}

			// Get family names for tracking
			var familyNames []string
			if len(exportedFont.FamilyNames) > 0 {
				familyNames = exportedFont.FamilyNames
			} else if exportedFont.FamilyName != "" {
				familyNames = []string{exportedFont.FamilyName}
			}

			// Check source availability BEFORE trying to get font from repository
			if sourceName != "" && configManifest != nil {
				sourceConfig, exists := configManifest.Sources[sourceName]
				if !exists {
					// Source doesn't exist - add to missing sources and skip
					if len(familyNames) > 0 {
						missingSourceFonts[sourceName] = append(missingSourceFonts[sourceName], strings.Join(familyNames, ", "))
					}
					// Track for later reporting, but don't try to get font
					fontsBySource[sourceName] = append(fontsBySource[sourceName], sourceFontInfo{
						fontID:      exportedFont.FontID,
						familyNames: familyNames,
					})
					continue
				} else if !sourceConfig.Enabled {
					// Source exists but is disabled - add to disabled sources and skip
					if len(familyNames) > 0 {
						disabledSourceFonts[sourceName] = append(disabledSourceFonts[sourceName], strings.Join(familyNames, ", "))
					}
					// Track for later reporting, but don't try to get font
					fontsBySource[sourceName] = append(fontsBySource[sourceName], sourceFontInfo{
						fontID:      exportedFont.FontID,
						familyNames: familyNames,
					})
					continue
				}
			}

			// Track this font by source for availability detection
			if sourceName != "" {
				fontsBySource[sourceName] = append(fontsBySource[sourceName], sourceFontInfo{
					fontID:      exportedFont.FontID,
					familyNames: familyNames,
				})
			}

			// Get font files from repository (only if source is enabled)
			fonts, err := repo.GetFontByID(exportedFont.FontID)
			if err != nil {
				output.GetDebug().State("Font ID %s not found in repository: %v", exportedFont.FontID, err)
				// Handle both old format (FamilyName) and new format (FamilyNames)
				if len(familyNames) > 0 {
					notFoundFonts = append(notFoundFonts, strings.Join(familyNames, ", "))
				}
				continue
			}

			if len(fonts) == 0 {
				// Handle both old format (FamilyName) and new format (FamilyNames)
				if len(familyNames) > 0 {
					notFoundFonts = append(notFoundFonts, strings.Join(familyNames, ", "))
				}
				continue
			}

			// Use family names we already extracted, or fallback to repository
			if len(familyNames) == 0 {
				// Fallback to font name from repository
				if len(fonts) > 0 {
					familyNames = []string{fonts[0].Name}
				}
			}

			fontsByID[exportedFont.FontID] = &fontInstallInfo{
				fonts:       fonts,
				sourceName:  sourceName,
				familyNames: familyNames,
			}
		}

		// Convert to FontToInstall slice
		var fontsToInstall []FontToInstall
		for fontID, info := range fontsByID {
			// Use first family name as primary name for FontToInstall (for display purposes)
			primaryName := info.familyNames[0]
			if len(info.familyNames) > 1 {
				// For multiple families, use comma-separated list
				primaryName = strings.Join(info.familyNames, ", ")
			}

			// Add to collection (fonts will be checked for installation status during installFont)
			fontsToInstall = append(fontsToInstall, FontToInstall{
				Fonts:      info.fonts,
				SourceName: info.sourceName,
				FontName:   primaryName, // This will be used for display (comma-separated for Nerd Fonts)
				FontID:     fontID,      // Store font ID for checking if already installed
			})
		}

		// Sort fonts alphabetically by Font ID to match export order
		// This matches the sorting logic in export.go for consistent ordering
		sort.Slice(fontsToInstall, func(i, j int) bool {
			// Both should have Font IDs (fonts without IDs are filtered out earlier)
			// Sort by Font ID alphabetically
			return fontsToInstall[i].FontID < fontsToInstall[j].FontID
		})

		// Also check for fonts that weren't found but weren't already caught by source availability check
		// (This handles edge cases where source is enabled but font still isn't found)
		if configManifest != nil {
			for sourceName, fontList := range fontsBySource {
				// Skip sources we've already handled (disabled/missing)
				if _, alreadyHandled := missingSourceFonts[sourceName]; alreadyHandled {
					continue
				}
				if _, alreadyHandled := disabledSourceFonts[sourceName]; alreadyHandled {
					continue
				}

				// Check if any fonts from this source were not found
				var missingFromSource []string
				for _, fontInfo := range fontList {
					// Check if this font was found (exists in fontsByID)
					if _, found := fontsByID[fontInfo.fontID]; !found {
						if len(fontInfo.familyNames) > 0 {
							missingFromSource = append(missingFromSource, strings.Join(fontInfo.familyNames, ", "))
						}
					}
				}

				// Add to notFoundFonts if there are any missing fonts from enabled sources
				if len(missingFromSource) > 0 {
					notFoundFonts = append(notFoundFonts, missingFromSource...)
				}
			}
		}

		// Debug output
		// Log import parameters and status (always log to file, not conditional on flags)
		GetLogger().Info("Import parameters - Scope: %s, Force: %v", scope, force)
		GetLogger().Info("Manifest contains %d fonts", len(exportManifest.Fonts))
		if len(invalidFonts) > 0 {
			GetLogger().Info("Skipping %d fonts without Font IDs", len(invalidFonts))
		}
		if len(notFoundFonts) > 0 {
			GetLogger().Info("Skipping %d fonts not found in repository", len(notFoundFonts))
		}
		if len(disabledSourceFonts) > 0 {
			for sourceName, fontNames := range disabledSourceFonts {
				GetLogger().Info("Source '%s' is disabled - %d fonts affected: %v", sourceName, len(fontNames), fontNames)
			}
		}
		if len(missingSourceFonts) > 0 {
			for sourceName, fontNames := range missingSourceFonts {
				GetLogger().Info("Source '%s' is missing - %d fonts affected: %v", sourceName, len(fontNames), fontNames)
			}
		}
		GetLogger().Info("Installing %d fonts", len(fontsToInstall))

		// If no fonts to install, exit
		if len(fontsToInstall) == 0 {
			fmt.Printf("%s\n", ui.WarningText.Render("No fonts to install. All fonts in the manifest are either invalid or not found in the repository."))
			fmt.Println()
			return nil
		}

		// Initialize status tracking
		status := &InstallationStatus{
			Details: make([]string, 0),
		}

		// For debug mode: bypass TUI and use plain text output
		if IsDebug() {
			return importFontsInDebugMode(fontManager, fontsToInstall, installScope, force, fontDir, status, scope)
		}

		// Create operation items for unified progress
		var operationItems []components.OperationItem
		for _, fontGroup := range fontsToInstall {
			// Use FontName which already contains comma-separated family names for Nerd Fonts
			var variantNames []string
			for _, font := range fontGroup.Fonts {
				variantNames = append(variantNames, font.Variant)
			}

			operationItems = append(operationItems, components.OperationItem{
				Name:          fontGroup.FontName, // Already contains comma-separated names for Nerd Fonts
				SourceName:    fontGroup.SourceName,
				Status:        "pending",
				StatusMessage: "Pending",
				Variants:      variantNames,
				Scope:         "",
			})
		}

		// Determine title based on scope
		title := "Importing Fonts"
		if installScope == platform.MachineScope {
			title = "Importing Fonts for All Users"
		}

		// Run unified progress for download and install
		verbose, _ := cmd.Flags().GetBool("verbose")
		debug, _ := cmd.Flags().GetBool("debug")
		progressErr := components.RunProgressBar(
			title,
			operationItems,
			verbose, // Verbose mode: show operational details and file/variant listings
			debug,   // Debug mode: show technical details
			func(send func(msg tea.Msg), cancelChan <-chan struct{}) error {
				// Process each font group
				for itemIndex, fontGroup := range fontsToInstall {
					send(components.ItemUpdateMsg{
						Index:   itemIndex,
						Status:  "in_progress",
						Message: "Downloading from " + fontGroup.SourceName,
					})

					percent := float64(itemIndex) / float64(len(fontsToInstall)) * 100
					send(components.ProgressUpdateMsg{Percent: percent})

					// Install the font
					result, err := installFont(
						fontGroup.Fonts,
						fontGroup.FontID,
						fontManager,
						installScope,
						force,
						fontDir,
					)

					if err != nil {
						status.Failed += result.Failed
						GetLogger().Error("Failed to process font %s: %v", fontGroup.FontName, err)
						errorMsg := err.Error()
						if len(errorMsg) > 100 {
							errorMsg = errorMsg[:97] + "..."
						}
						send(components.ItemUpdateMsg{
							Index:        itemIndex,
							Status:       "failed",
							Message:      "Operation failed",
							ErrorMessage: errorMsg,
						})
						continue
					}

					// Update status
					status.Installed += result.Success
					status.Skipped += result.Skipped
					status.Failed += result.Failed
					status.Errors = append(status.Errors, result.Errors...)

					// Get error message if failed
					var errorMsg string
					if result.Status == "failed" && len(result.Errors) > 0 {
						errorMsg = result.Errors[0]
					}

					// Use "Installed" message (font name is already shown in the item name)
					// Don't pass scope for single-scope operations (cleaner output, title already shows scope)
					send(components.ItemUpdateMsg{
						Index:        itemIndex,
						Status:       result.Status,
						Message:      "Installed", // Simple message like add command
						ErrorMessage: errorMsg,
						Scope:        "", // Empty for single-scope operations (cleaner output)
					})

					percent = float64(itemIndex+1) / float64(len(fontsToInstall)) * 100
					send(components.ProgressUpdateMsg{Percent: percent})
				}

				return nil
			},
		)

		if progressErr != nil {
			GetLogger().Error("Failed to import fonts: %v", progressErr)
			return progressErr
		}

		// Show source availability warnings at the bottom (after progress bar, before status report)
		if !IsDebug() {
			// Show disabled sources
			for sourceName, fontNames := range disabledSourceFonts {
				fmt.Printf("%s\n", ui.WarningText.Render(fmt.Sprintf("The following fonts require '%s' which is currently disabled:", sourceName)))
				for _, fontName := range fontNames {
					fmt.Printf("  - %s\n", fontName)
				}
				// Blank line before help text (within section, per spacing framework)
				fmt.Println()
				fmt.Printf("Enable this source via 'fontget sources manage' to import these fonts.\n")
				// Section ends with blank line (per spacing framework)
				fmt.Println()
			}

			// Show missing sources
			for sourceName, fontNames := range missingSourceFonts {
				fmt.Printf("%s\n", ui.WarningText.Render(fmt.Sprintf("The following fonts require '%s' which is not available in your sources:", sourceName)))
				for _, fontName := range fontNames {
					fmt.Printf("  - %s\n", fontName)
				}
				if builtInSources[sourceName] {
					fmt.Printf("Run 'fontget sources update' to refresh sources, or enable this source via 'fontget sources manage'.\n")
				} else {
					fmt.Printf("Add this source via 'fontget sources manage' to import these fonts.\n")
				}
				fmt.Println()
			}

			// Show invalid fonts (no Font ID)
			if len(invalidFonts) > 0 {
				fmt.Printf("%s\n", ui.WarningText.Render("The following fonts in the manifest have no Font ID and will be skipped:"))
				for _, fontName := range invalidFonts {
					fmt.Printf("  - %s\n", fontName)
				}
				fmt.Println()
			}

			// Show not found fonts (found in repository but not available - different from source issues)
			if len(notFoundFonts) > 0 {
				// Filter out fonts that are already covered by source availability messages
				var remainingNotFound []string
				for _, fontName := range notFoundFonts {
					// Check if this font is already covered by source availability messages
					covered := false
					for _, fontNames := range disabledSourceFonts {
						for _, fn := range fontNames {
							if fn == fontName {
								covered = true
								break
							}
						}
						if covered {
							break
						}
					}
					if !covered {
						for _, fontNames := range missingSourceFonts {
							for _, fn := range fontNames {
								if fn == fontName {
									covered = true
									break
								}
							}
							if covered {
								break
							}
						}
					}
					if !covered {
						remainingNotFound = append(remainingNotFound, fontName)
					}
				}

				if len(remainingNotFound) > 0 {
					fmt.Printf("%s\n", ui.WarningText.Render("The following fonts were not found in the repository:"))
					for _, fontName := range remainingNotFound {
						fmt.Printf("  - %s\n", fontName)
					}
					fmt.Printf("Try running 'fontget sources update' to refresh the repository.\n")
					fmt.Println()
				}
			}
		}

		// Print status report last (only shown in verbose mode)
		output.PrintStatusReport(output.StatusReport{
			Success:      status.Installed,
			Skipped:      status.Skipped,
			Failed:       status.Failed,
			SuccessLabel: "Installed",
			SkippedLabel: "Skipped",
			FailedLabel:  "Failed",
		}, IsVerbose())

		GetLogger().Info("Import complete - Installed: %d, Skipped: %d, Failed: %d",
			status.Installed, status.Skipped, status.Failed)

		return nil
	},
}

// importFontsInDebugMode processes fonts with plain text output (no TUI) for easier parsing/logging
func importFontsInDebugMode(fontManager platform.FontManager, fontsToInstall []FontToInstall, installScope platform.InstallationScope, force bool, fontDir string, status *InstallationStatus, _ string) error {
	output.GetDebug().State("Starting font import operation")
	output.GetDebug().State("Total fonts: %d", len(fontsToInstall))

	scopeLabel := "user scope"
	if installScope == platform.MachineScope {
		scopeLabel = "machine scope"
	}

	// Process each font
	for i, fontGroup := range fontsToInstall {
		output.GetDebug().State("Importing font %d/%d: %s", i+1, len(fontsToInstall), fontGroup.FontName)
		output.GetDebug().State("Installing font %s in %s (directory: %s)", fontGroup.FontName, scopeLabel, fontDir)

		output.GetDebug().State("Calling installFont(%s, %s, %s, %v, %s)", fontGroup.FontID, scopeLabel, fontDir, force, "...")

		result, err := installFont(
			fontGroup.Fonts,
			fontGroup.FontID,
			fontManager,
			installScope,
			force,
			fontDir,
		)

		if err != nil {
			output.GetDebug().State("Error installing font %s in %s: %v", fontGroup.FontName, scopeLabel, err)
			if result != nil {
				status.Failed += result.Failed
				status.Skipped += result.Skipped
				status.Errors = append(status.Errors, result.Errors...)
			}
			continue
		}

		// Update status
		status.Installed += result.Success
		status.Skipped += result.Skipped
		status.Failed += result.Failed
		status.Errors = append(status.Errors, result.Errors...)

		// Show detailed result information in debug mode
		if len(result.Details) > 0 {
			installedCount := result.Success
			skippedCount := result.Skipped
			failedCount := result.Failed
			var installedFiles, skippedFiles, failedFiles []string
			idx := 0
			if installedCount > 0 && idx < len(result.Details) {
				installedFiles = result.Details[idx : idx+installedCount]
				idx += installedCount
			}
			if skippedCount > 0 && idx < len(result.Details) {
				skippedFiles = result.Details[idx : idx+skippedCount]
				idx += skippedCount
			}
			if failedCount > 0 && idx < len(result.Details) {
				failedFiles = result.Details[idx : idx+failedCount]
			}

			if len(installedFiles) > 0 {
				output.GetDebug().State("Installed variants:")
				for _, file := range installedFiles {
					output.GetDebug().State(" - %s", file)
				}
			}
			if len(skippedFiles) > 0 {
				output.GetDebug().State("Skipped variants:")
				for _, file := range skippedFiles {
					output.GetDebug().State(" - %s", file)
				}
			}
			if len(failedFiles) > 0 {
				output.GetDebug().State("Failed variants:")
				for _, file := range failedFiles {
					output.GetDebug().State(" - %s", file)
				}
			}
		}

		// Show success message with comma-separated family names
		if result.Status == "completed" && result.Success > 0 {
			fmt.Printf("Installed %s\n", fontGroup.FontName)
		}
		output.GetDebug().State("Font %s in %s completed: %s - %s (Installed: %d, Skipped: %d, Failed: %d)",
			fontGroup.FontName, scopeLabel, result.Status, result.Message, result.Success, result.Skipped, result.Failed)
	}

	output.GetDebug().State("Operation complete - Installed: %d, Skipped: %d, Failed: %d",
		status.Installed, status.Skipped, status.Failed)

	// Print status report
	output.PrintStatusReport(output.StatusReport{
		Success:      status.Installed,
		Skipped:      status.Skipped,
		Failed:       status.Failed,
		SuccessLabel: "Installed",
		SkippedLabel: "Skipped",
		FailedLabel:  "Failed",
	}, IsVerbose())

	GetLogger().Info("Import complete - Installed: %d, Skipped: %d, Failed: %d",
		status.Installed, status.Skipped, status.Failed)
	return nil
}

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.Flags().StringP("scope", "s", "", "Installation scope override (user or machine)")
	importCmd.Flags().BoolP("force", "f", false, "Force installation even if font is already installed")
}
