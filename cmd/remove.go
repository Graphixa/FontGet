package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fontget/internal/cmdutils"
	"fontget/internal/components"
	"fontget/internal/output"
	"fontget/internal/platform"
	"fontget/internal/repo"
	"fontget/internal/shared"
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

// resolveFontNameOrID resolves a Font ID to a font name, or returns the original if it's already a font name
// This allows the remove command to accept both Font IDs (e.g., "google.noto-sans") and font names (e.g., "Noto Sans")
func resolveFontNameOrID(input string, repository *repo.Repository) string {
	// Check if this is a Font ID (contains a dot like "google.noto-sans")
	if !strings.Contains(input, ".") {
		// Not a Font ID, return as-is
		return input
	}

	// This is a Font ID, try to resolve it to a font name
	if repository == nil {
		// No repository available, return original
		return input
	}

	fontID := strings.ToLower(input)
	fonts, err := repo.GetFontByID(fontID)
	if err == nil && len(fonts) > 0 {
		// Successfully resolved Font ID to font name
		return fonts[0].Name
	}

	// Font ID not found in repository, return original (will be handled as font name)
	return input
}

// extractFontDisplayNameFromPath extracts the proper display name from a font file path
// Uses font metadata (SFNT name table) for accurate font names, falls back to filename parsing
func extractFontDisplayNameFromPath(fontPath string) string {
	// Try to extract metadata from the font file first (most accurate)
	if metadata, err := platform.ExtractFontMetadata(fontPath); err == nil {
		if metadata.FamilyName != "" {
			// Use FormatFontNameWithVariant to properly format the name with style
			return shared.FormatFontNameWithVariant(metadata.FamilyName, metadata.StyleName)
		}
	}

	// Fallback to filename parsing if metadata extraction fails
	filename := filepath.Base(fontPath)
	return shared.GetDisplayNameFromFilename(filename)
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
	// Note: convertCamelCaseToSpaced is unexported in shared package
	// We'll use GetDisplayNameFromFilename which handles camelCase conversion
	return shared.GetDisplayNameFromFilename(family + ".ttf")
}

// processRemoveResult processes and categorizes removal result details (removed/skipped/failed variants)
func processRemoveResult(result *RemoveResult) (removedFiles, skippedFiles, failedFiles []string) {
	if result == nil || len(result.Details) == 0 {
		return nil, nil, nil
	}

	for _, detail := range result.Details {
		if strings.Contains(detail, " (Skipped") {
			variantName := strings.TrimSuffix(strings.TrimSuffix(detail, " (Skipped - Protected system font)"), " (Skipped)")
			skippedFiles = append(skippedFiles, variantName)
		} else if strings.Contains(detail, " (Failed)") {
			variantName := strings.TrimSuffix(detail, " (Failed)")
			failedFiles = append(failedFiles, variantName)
		} else {
			removedFiles = append(removedFiles, detail)
		}
	}

	return removedFiles, skippedFiles, failedFiles
}

// logRemoveResultDetails logs detailed variant information in debug mode
func logRemoveResultDetails(result *RemoveResult, fontName, scopeLabel string) {
	if result == nil {
		return
	}

	removedFiles, skippedFiles, failedFiles := processRemoveResult(result)

	if len(removedFiles) > 0 {
		output.GetDebug().State("Removed variants:")
		for _, file := range removedFiles {
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

	output.GetDebug().State("Font %s in %s completed: %s - %s (Removed: %d, Skipped: %d, Failed: %d)",
		fontName, scopeLabel, result.Status, result.Message, result.Success, result.Skipped, result.Failed)
}

// updateRemovalStatus updates removal status from result
func updateRemovalStatus(status *RemovalStatus, result *RemoveResult) {
	if result == nil || status == nil {
		return
	}
	status.Removed += result.Success
	status.Skipped += result.Skipped
	status.Failed += result.Failed
}

// findFontFilesForRemoval finds all font files matching the font name
func findFontFilesForRemoval(fontName string, fontManager platform.FontManager, scope platform.InstallationScope, repository *repo.Repository) ([]string, error) {
	// Resolve Font ID to font name if needed (supports both Font IDs and font names)
	searchName := resolveFontNameOrID(fontName, repository)
	if searchName != fontName {
		output.GetDebug().State("Resolved font name: %s -> %s", fontName, searchName)
	}

	// Find font files in the specified scope
	output.GetDebug().State("Calling findFontFamilyFiles(%s, %s)", searchName, scope)
	matchingFonts := findFontFamilyFiles(searchName, fontManager, scope)
	output.GetDebug().State("Found %d matching font file(s)", len(matchingFonts))

	// If no direct matches, try repository search
	if len(matchingFonts) == 0 && repository != nil {
		output.GetDebug().State("No direct matches found, trying repository search for: %s", searchName)
		results, err := repository.SearchFonts(searchName, "false")
		if err == nil && len(results) > 0 {
			output.GetDebug().State("Repository search found: %s, searching for matching files", results[0].Name)
			matchingFonts = findFontFamilyFiles(results[0].Name, fontManager, scope)
			output.GetDebug().State("Found %d matching font file(s) after repository search", len(matchingFonts))
		} else if err != nil {
			output.GetDebug().State("Repository search failed: %v", err)
		}
	}

	if len(matchingFonts) == 0 {
		return nil, fmt.Errorf("font not found: %s", fontName)
	}

	return matchingFonts, nil
}

// RemoveFontFilesParams contains parameters for removeFontFiles function
type RemoveFontFilesParams struct {
	MatchingFonts        []string
	FontManager          platform.FontManager
	Scope                platform.InstallationScope
	FontDir              string
	IsCriticalSystemFont func(string) bool
}

// removeFontFiles removes font files from system
func removeFontFiles(params RemoveFontFilesParams) (removed, skipped, failed int, details []string, errors []string) {
	for _, matchingFont := range params.MatchingFonts {
		// Construct full font path for metadata extraction
		fontPath := filepath.Join(params.FontDir, matchingFont)

		// Check for protected system fonts - ALWAYS enforced
		// Critical system fonts should never be removable for system stability
		if params.IsCriticalSystemFont != nil && params.IsCriticalSystemFont(matchingFont) {
			skipped++
			fontDisplayName := extractFontDisplayNameFromPath(fontPath)
			details = append(details, fontDisplayName+" (Skipped - Protected system font)")
			continue
		}

		fontDisplayName := extractFontDisplayNameFromPath(fontPath)

		// Remove font
		output.GetDebug().State("Calling fontManager.RemoveFont(%s, %s)", matchingFont, params.Scope)
		err := params.FontManager.RemoveFont(matchingFont, params.Scope)

		if err != nil {
			failed++
			var errorMsg string
			errStr := err.Error()
			if containsAny(errStr, []string{"in use", "access denied", "permission"}) {
				errorMsg = "Font is in use or access denied"
			} else {
				errorMsg = "Failed to remove existing font"
			}
			errors = append(errors, errorMsg)
			details = append(details, fontDisplayName+" (Failed)")
			output.GetDebug().State("fontManager.RemoveFont() failed for %s: %v", matchingFont, err)
			continue
		}

		removed++
		details = append(details, fontDisplayName)
	}

	return removed, skipped, failed, details, errors
}

// buildRemoveResult builds RemoveResult from removal outcomes
func buildRemoveResult(removed, skipped, failed int, details []string, errors []string) *RemoveResult {
	result := &RemoveResult{
		Success: removed,
		Skipped: skipped,
		Failed:  failed,
		Details: details,
		Errors:  errors,
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

	return result
}

// removeFont handles the core removal logic for a single font.
//
// It finds all font files matching the font name (supports both Font IDs and font names),
// removes them from the system, and returns a RemoveResult with the operation outcome.
// Protected system fonts are automatically skipped.
//
// Parameters:
//   - fontName: Font name or Font ID to remove
//   - fontManager: Platform-specific font manager for removal
//   - scope: Installation scope (user or machine)
//   - fontDir: Font directory path
//   - repository: Font repository for resolving Font IDs to names
//
// Returns:
//   - RemoveResult: Contains removed/skipped/failed counts and details
//   - error: Removal error if the operation fails
func removeFont(
	fontName string,
	fontManager platform.FontManager,
	scope platform.InstallationScope,
	fontDir string,
	repository *repo.Repository,
) (*RemoveResult, error) {
	// Find font files for removal
	matchingFonts, err := findFontFilesForRemoval(fontName, fontManager, scope, repository)
	if err != nil {
		// Font not found - return result with failed status
		result := buildRemoveResult(0, 0, 0, nil, nil)
		result.Status = "failed"
		result.Message = "Font not found"
		return result, err
	}

	// Remove font files
	removed, skipped, failed, details, errors := removeFontFiles(RemoveFontFilesParams{
		MatchingFonts:        matchingFonts,
		FontManager:          fontManager,
		Scope:                scope,
		FontDir:              fontDir,
		IsCriticalSystemFont: shared.IsCriticalSystemFont,
	})

	// Build and return result
	return buildRemoveResult(removed, skipped, failed, details, errors), nil
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

var removeCmd = &cobra.Command{
	Use:          "remove <font-id> [<font-id2> <font-id3> ...]",
	Aliases:      []string{"uninstall"},
	Short:        "Remove fonts from your system",
	SilenceUsage: true,
	Long: `Remove one or multiple fonts by name (e.g., "Roboto") or Font ID (e.g., "google.roboto").
Names with spaces must be quoted. Scope: user (default), machine, or all (admin for system-wide).`,
	Example: `  fontget remove "Roboto"
  fontget remove "google.roboto"
  fontget remove "Open Sans" "Fira Sans" "Noto Sans"
  fontget remove "roboto, firasans, notosans"
  fontget remove "Open Sans" -s machine
  fontget remove "Roboto" -s user`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
			fmt.Printf("\n%s\n", ui.RenderError("A font ID is required"))
			fmt.Printf("Use 'fontget remove --help' for more information.\n\n")
			return nil
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		GetLogger().Info("Starting font removal operation")

		// Always start with a blank line for consistent spacing from command prompt
		fmt.Println()

		if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
			return nil
		}

		fontManager, err := platform.NewFontManager()
		if err != nil {
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("platform.NewFontManager() failed: %v", err)
			return fmt.Errorf("unable to access system fonts: %v", err)
		}

		scopeFlag, _ := cmd.Flags().GetString("scope")

		status := &RemovalStatus{Details: make([]string, 0)}

		r, err := repo.GetRepository()
		if err != nil {
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("repo.GetRepository() failed: %v", err)
			return fmt.Errorf("unable to load font repository: %v", err)
		}

		// Auto-detect scope if not explicitly provided
		if scopeFlag == "" {
			var err error
			scopeFlag, err = platform.AutoDetectScope(fontManager, "user", "all", GetLogger())
			if err != nil {
				// Should not happen, but handle gracefully
				scopeFlag = "user"
			}
		}

		// Log removal parameters after scope is determined (for debug mode)
		GetLogger().Info("Removal parameters - Scope: %s", scopeFlag)
		// Format font list as a bulleted list
		GetLogger().Info("Removing %d Font(s):", len(args))
		for _, fontName := range args {
			GetLogger().Info(" - %s", fontName)
		}

		// Verbose-level information for users - show operational details before progress bar
		// (After scope auto-detection so we can display the actual scope being used)
		// Format scope label for display
		scopeDisplay := scopeFlag
		if scopeFlag == "all" {
			scopeDisplay = "all (user and machine)"
		}
		output.GetVerbose().Info("Scope: %s", scopeDisplay)
		output.GetVerbose().Info("Removing %d font(s)", len(args))
		// Verbose section ends with blank line per spacing framework (only if verbose was shown)
		if IsVerbose() {
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
				err := fmt.Errorf("invalid scope '%s'. Valid options are: 'user' or 'machine'", scopeFlag)
				output.GetVerbose().Error("%v", err)
				output.GetDebug().Error("Invalid scope provided: '%s'", scopeFlag)
				return err
			}
			scopes = []platform.InstallationScope{s}
			scopeLabel = []string{scopeFlag}
		}

		// Check elevation for machine scope operations (single scope or all)
		if len(scopes) == 1 && scopes[0] == platform.MachineScope {
			if err := cmdutils.CheckElevation(cmd, fontManager, platform.MachineScope); err != nil {
				if errors.Is(err, cmdutils.ErrElevationRequired) {
					return nil // Already printed user-friendly message
				}
				output.GetVerbose().Error("%v", err)
				output.GetDebug().Error("checkElevation() failed: %v", err)
				return fmt.Errorf("unable to verify system permissions: %v", err)
			}
		}

		// Process font names from arguments
		fontNames := cmdutils.ParseFontNames(args)

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

			// Resolve Font ID to font name if needed (supports both Font IDs and font names)
			searchName := resolveFontNameOrID(fontName, r)

			// Try to find the font and extract its proper name
			// For "all" scope, check all scopes to find the font (don't break early)
			// For single scope, break after first match
			for _, scopeType := range scopes {
				matchingFonts := findFontFamilyFiles(searchName, fontManager, scopeType)
				if len(matchingFonts) > 0 {
					fontDir := fontManager.GetFontDir(scopeType)
					firstFontPath := filepath.Join(fontDir, matchingFonts[0])
					// Extract proper name (use first one found, but continue checking for "all" scope)
					if properName == "" {
						properName = extractFontFamilyNameFromPath(firstFontPath)
					}
					fontFound = true
					// For single scope, break after first match
					if len(scopes) == 1 {
						break
					}
					// For "all" scope, continue checking all scopes
				}
			}
			// If no direct matches, try repository search
			if !fontFound {
				results, err := r.SearchFonts(searchName, "false")
				if err == nil && len(results) > 0 {
					repoSearchName := results[0].Name
					for _, scopeType := range scopes {
						matchingFonts := findFontFamilyFiles(repoSearchName, fontManager, scopeType)
						if len(matchingFonts) > 0 {
							fontDir := fontManager.GetFontDir(scopeType)
							firstFontPath := filepath.Join(fontDir, matchingFonts[0])
							// Extract proper name (use first one found, but continue checking for "all" scope)
							if properName == "" {
								properName = extractFontFamilyNameFromPath(firstFontPath)
							}
							fontFound = true
							// For single scope, break after first match
							if len(scopes) == 1 {
								break
							}
							// For "all" scope, continue checking all scopes
						}
					}
				}
			}

			// If still not found and we're checking a single scope, check the opposite scope
			// This provides better UX by detecting fonts in the opposite scope early
			if !fontFound && len(scopes) == 1 {
				var oppositeScope platform.InstallationScope
				switch scopes[0] {
				case platform.UserScope:
					oppositeScope = platform.MachineScope
				case platform.MachineScope:
					oppositeScope = platform.UserScope
				}

				if oppositeScope != "" {
					// Always add to notFoundFonts - the message handler will check the opposite scope
					// and show the helpful message about using --scope if the font is found there
					notFoundFonts = append(notFoundFonts, fontName)
					continue // Skip adding to foundFonts
				}
			}

			if fontFound && properName != "" {
				// Font found - add to found list
				foundFonts = append(foundFonts, FontInfo{
					SearchName: fontName,
					ProperName: properName,
				})
				fontNameMap[fontName] = properName
			} else if !fontFound {
				// Font not found - add to not found list
				notFoundFonts = append(notFoundFonts, fontName)
				// Log to file and show in debug console
				GetLogger().Error("Font not found: %s", fontName)
				output.GetDebug().Error("Font not found in installed fonts: %s", fontName)
			}
		}

		// If no fonts to remove, show not found message and exit (don't show progress bar)
		if len(foundFonts) == 0 {
			if len(notFoundFonts) > 0 {
				if IsDebug() {
					// In debug mode, show technical details to console
					scopeDisplay := scopeFlag
					if scopeDisplay == "" {
						scopeDisplay = "user"
					}
					output.GetDebug().Error("No fonts found to remove. The following font(s) were not found installed in the '%s' scope:", scopeDisplay)
					// Deduplicate for display
					seenNotFound := make(map[string]bool)
					for _, fontName := range notFoundFonts {
						normalized := normalizeFontName(fontName)
						if !seenNotFound[normalized] {
							seenNotFound[normalized] = true
							output.GetDebug().Error(" - %s", fontName)
						}
					}
				} else {
					// In normal/verbose mode, show user-friendly message with suggestions
					// Deduplicate notFoundFonts
					seenNotFound := make(map[string]bool)
					uniqueNotFound := []string{}
					for _, fontName := range notFoundFonts {
						normalized := normalizeFontName(fontName)
						if !seenNotFound[normalized] {
							seenNotFound[normalized] = true
							uniqueNotFound = append(uniqueNotFound, fontName)
						}
					}
					notFoundFonts = uniqueNotFound

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
						switch scopes[0] {
						case platform.UserScope:
							oppositeScope = platform.MachineScope
							oppositeScopeName = "machine"
						case platform.MachineScope:
							oppositeScope = platform.UserScope
							oppositeScopeName = "user"
						}

						// Check each not found font in the opposite scope
						if oppositeScope != "" {
							for _, fontName := range notFoundFonts {
								// Resolve Font ID to font name if needed
								searchName := resolveFontNameOrID(fontName, r)
								// Try direct match first
								matchingFonts := findFontFamilyFiles(searchName, fontManager, oppositeScope)
								if len(matchingFonts) == 0 {
									// Try repository search
									results, err := r.SearchFonts(searchName, "false")
									if err == nil && len(results) > 0 {
										repoSearchName := results[0].Name
										matchingFonts = findFontFamilyFiles(repoSearchName, fontManager, oppositeScope)
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

					// Display truly not found fonts with suggestions (only if there are any)
					if len(trulyNotFound) > 0 {
						// Get installed font family names for suggestions
						var installedFontNames []string
						for _, scopeType := range scopes {
							fontDir := fontManager.GetFontDir(scopeType)
							fontFiles, err := platform.ListInstalledFonts(fontDir)
							if err == nil {
								// Extract unique family names from installed fonts
								seenFamilies := make(map[string]bool)
								for _, fontFile := range fontFiles {
									fontPath := filepath.Join(fontDir, fontFile)
									familyName := extractFontFamilyNameFromPath(fontPath)
									normalized := normalizeFontName(familyName)
									if !seenFamilies[normalized] && familyName != "" {
										seenFamilies[normalized] = true
										installedFontNames = append(installedFontNames, familyName)
									}
								}
							}
						}

						// Build normalized map of all not-found fonts (including those in opposite scope)
						notFoundNormalized := make(map[string]bool)
						for _, fontName := range notFoundFonts {
							notFoundNormalized[normalizeFontName(fontName)] = true
						}

						// Filter out not found fonts from suggestions (don't suggest fonts that are being removed)
						filteredInstalledFontNames := []string{}
						for _, fontName := range installedFontNames {
							if !notFoundNormalized[normalizeFontName(fontName)] {
								filteredInstalledFontNames = append(filteredInstalledFontNames, fontName)
							}
						}
						installedFontNames = filteredInstalledFontNames

						// Show header first
						fmt.Printf("%s\n", ui.ErrorText.Render(fmt.Sprintf("The following font(s) were not found installed in the '%s' scope:", scopeDisplay)))

						// Show all not-found fonts grouped together
						if len(trulyNotFound) == 1 {
							// Single font - show with suggestions
							fontName := trulyNotFound[0]
							var similar []string
							if len(installedFontNames) > 0 {
								similar = shared.FindSimilarFonts(fontName, installedFontNames, true) // true = installed fonts
							}

							// Filter out any suggestions that are also in the not found list
							filteredSimilar := []string{}
							for _, suggestion := range similar {
								if !notFoundNormalized[normalizeFontName(suggestion)] {
									filteredSimilar = append(filteredSimilar, suggestion)
								}
							}
							similar = filteredSimilar

							if len(similar) > 0 {
								fmt.Printf("%s\n", ui.ErrorText.Render(fmt.Sprintf("Font '%s' not found.", fontName)))
								fmt.Printf("%s\n\n", ui.Text.Render("Did you mean one of these installed fonts?"))
								for _, similarFont := range similar {
									fmt.Printf(" - %s\n", ui.TableSourceName.Render(similarFont))
								}
							} else {
								fmt.Printf("%s\n", ui.ErrorText.Render(fmt.Sprintf(" - %s", fontName)))
							}
						} else {
							// Multiple fonts - show grouped format
							for _, fontName := range trulyNotFound {
								fmt.Printf(" - %s\n", fontName)
							}

							// Collect all suggestions for all fonts (limit to 12 total)
							const maxSuggestions = 12
							allSimilar := []string{}
							seenSimilar := make(map[string]bool)

							for _, fontName := range trulyNotFound {
								if len(allSimilar) >= maxSuggestions {
									break
								}
								var similar []string
								if len(installedFontNames) > 0 {
									similar = shared.FindSimilarFonts(fontName, installedFontNames, true) // true = installed fonts
								}
								// Deduplicate suggestions and filter out not-found fonts
								for _, suggestion := range similar {
									if len(allSimilar) >= maxSuggestions {
										break
									}
									normalizedSuggestion := normalizeFontName(suggestion)
									if !seenSimilar[suggestion] && !notFoundNormalized[normalizedSuggestion] {
										seenSimilar[suggestion] = true
										allSimilar = append(allSimilar, suggestion)
									}
								}
							}

							// Show consolidated suggestions if any
							if len(allSimilar) > 0 {
								fmt.Println()
								fmt.Printf("%s\n\n", ui.Text.Render("Did you mean one of these installed fonts?"))
								for _, similarFont := range allSimilar {
									fmt.Printf(" - %s\n", ui.TableSourceName.Render(similarFont))
								}
							}
						}

						// Add blank line before hint or next section
						if len(fontsInOppositeScope) > 0 {
							fmt.Println()
						} else if len(trulyNotFound) > 0 {
							fmt.Println()
							fmt.Printf("Try using 'fontget list' to show currently installed fonts.\n")
						}
					}

					// Display fonts found in opposite scope as a separate section
					if len(fontsInOppositeScope) > 0 {
						// Use FeedbackWarning (yellow, no bold) for the header message
						fmt.Printf("%s\n", ui.WarningText.Render(fmt.Sprintf("The following font(s) are installed in the '%s' scope, use '--scope %s' to remove them:", oppositeScopeName, oppositeScopeName)))
						for _, fontName := range notFoundFonts {
							if properName, foundInOpposite := fontsInOppositeScope[fontName]; foundInOpposite {
								fmt.Printf(" - %s\n", properName)
							}
						}
						// Section ends with blank line per spacing framework
						fmt.Println()
					} else if len(trulyNotFound) > 0 {
						// Section ends with blank line per spacing framework
						fmt.Println()
					}
				}
			}
			return nil
		}

		// For --all scope, require elevation upfront
		if len(scopes) == 2 {
			// Check elevation first
			if err := cmdutils.CheckElevation(cmd, fontManager, platform.MachineScope); err != nil {
				if errors.Is(err, cmdutils.ErrElevationRequired) {
					return nil // Already printed user-friendly message
				}
				output.GetVerbose().Error("%v", err)
				output.GetDebug().Error("checkElevation() failed for --scope all: %v", err)
				return fmt.Errorf("unable to verify system permissions: %v", err)
			}
		}

		// Handle single scope operations (user or machine) - use TUI like add command
		// For "all" scope: Check each font in each scope individually and only create items
		// for scopes where the font actually exists (using existing detection logic)
		var operationItems []components.OperationItem
		type FontScopeItem struct {
			FontName   string
			ProperName string
			ScopeType  platform.InstallationScope
			ScopeLabel string
			ItemIndex  int
		}
		var fontScopeItems []FontScopeItem

		if len(scopes) > 1 {
			// "all" scope - check each font in each scope individually
			itemIndex := 0
			for _, fontInfo := range foundFonts {
				searchName := resolveFontNameOrID(fontInfo.SearchName, r)
				for j, scopeType := range scopes {
					scopeLabelName := scopeLabel[j]
					// Check if font exists in this scope
					matchingFonts := findFontFamilyFiles(searchName, fontManager, scopeType)
					if len(matchingFonts) == 0 {
						// Try repository search
						results, err := r.SearchFonts(searchName, "false")
						if err == nil && len(results) > 0 {
							repoSearchName := results[0].Name
							matchingFonts = findFontFamilyFiles(repoSearchName, fontManager, scopeType)
						}
					}
					// Only create item if font exists in this scope
					if len(matchingFonts) > 0 {
						// Extract proper name if not already set
						properName := fontInfo.ProperName
						if properName == "" {
							fontDir := fontManager.GetFontDir(scopeType)
							firstFontPath := filepath.Join(fontDir, matchingFonts[0])
							properName = extractFontFamilyNameFromPath(firstFontPath)
						}
						operationItems = append(operationItems, components.OperationItem{
							Name:          properName,
							Status:        "pending",
							StatusMessage: "Pending",
							Variants:      []string{},
							Scope:         scopeLabelName + " scope",
						})
						fontScopeItems = append(fontScopeItems, FontScopeItem{
							FontName:   fontInfo.SearchName,
							ProperName: properName,
							ScopeType:  scopeType,
							ScopeLabel: scopeLabelName,
							ItemIndex:  itemIndex,
						})
						itemIndex++
					}
				}
			}
		} else {
			// Single scope - one item per font
			for _, fontInfo := range foundFonts {
				operationItems = append(operationItems, components.OperationItem{
					Name:          fontInfo.ProperName,
					Status:        "pending",
					StatusMessage: "Pending",
					Variants:      []string{},
					Scope:         "",
				})
			}
		}

		// Check if flags are set
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
					output.GetDebug().State("Removing font %s in %s (directory: %s)", fontInfo.SearchName, scopeLabelName, fontDir)

					output.GetDebug().State("Calling removeFont(%s, %s, %s)", fontInfo.SearchName, scopeLabelName, fontDir)
					result, err := removeFont(
						fontInfo.SearchName,
						fontManager,
						scopeType,
						fontDir,
						r,
					)

					if err != nil {
						output.GetDebug().State("Error removing font %s in %s: %v", fontInfo.SearchName, scopeLabelName, err)
						if result != nil {
							updateRemovalStatus(status, result)
							// Show failed variants if available
							_, _, failedFiles := processRemoveResult(result)
							if len(failedFiles) > 0 {
								output.GetDebug().State("Failed variants:")
								for _, file := range failedFiles {
									output.GetDebug().State(" - %s", file)
								}
							}
						}
						continue
					}

					// Update status
					updateRemovalStatus(status, result)

					// Show detailed result information in debug mode
					logRemoveResultDetails(result, fontInfo.SearchName, scopeLabelName)
				}
			}

			output.GetDebug().State("Operation complete - Removed: %d, Skipped: %d, Failed: %d",
				status.Removed, status.Skipped, status.Failed)

			// Print status report
			output.PrintStatusReport(output.StatusReport{
				Success:      status.Removed,
				Skipped:      status.Skipped,
				Failed:       status.Failed,
				SuccessLabel: "Removed",
				SkippedLabel: "Skipped",
				FailedLabel:  "Failed",
			}, IsVerbose())

			GetLogger().Info("Removal complete - Removed: %d, Skipped: %d, Failed: %d",
				status.Removed, status.Skipped, status.Failed)
			return nil
		}

		// Determine title based on scope
		title := "Removing Fonts"
		if len(scopes) == 1 && scopes[0] == platform.MachineScope {
			title = "Removing Fonts for All Users"
		} else if len(scopes) > 1 {
			// "all" scope - show both scopes in title
			title = "Removing Fonts for both Machine & User scopes"
		}

		// Run unified progress for font removal (TUI mode)
		fontsInOppositeScope := []string{} // Track fonts that still exist in opposite scope after removal
		progressErr := components.RunProgressBar(
			title,
			operationItems,
			verbose, // Verbose mode: show operational details and file/variant listings
			debug,   // Debug mode: show technical details
			func(send func(msg tea.Msg)) error {
				// Process items based on scope mode
				if len(scopes) > 1 {
					// "all" scope - process each font+scope combination individually
					for _, item := range fontScopeItems {
						// Send initial "in_progress" message
						send(components.ItemUpdateMsg{
							Index:   item.ItemIndex,
							Name:    item.ProperName,
							Status:  "in_progress",
							Message: "Removing...",
						})

						GetLogger().Info("Processing font: %s in %s scope", item.FontName, item.ScopeLabel)

						fontDir := fontManager.GetFontDir(item.ScopeType)
						result, err := removeFont(
							item.FontName,
							fontManager,
							item.ScopeType,
							fontDir,
							r,
						)

						// Collect variants for display (only in verbose mode)
						scopeVariants := []string{}
						if verbose {
							scopeVariants = result.Details
						}

						// Determine status
						scopeStatus := "completed"
						scopeMessage := "Removed"
						if err != nil {
							if strings.Contains(err.Error(), "not found") {
								// Font not found in this scope - skip (shouldn't happen since we checked)
								continue
							}
							scopeStatus = "failed"
							scopeMessage = err.Error()
							status.Failed++
							if result != nil {
								status.Failed += result.Failed
								status.Skipped += result.Skipped
							}
						} else if result != nil {
							status.Removed += result.Success
							status.Skipped += result.Skipped
							status.Failed += result.Failed

							switch result.Status {
							case "failed":
								scopeStatus = "failed"
								scopeMessage = result.Message
							case "skipped":
								scopeStatus = "skipped"
								scopeMessage = result.Message
							}
						}

						// Send update for this scope
						errorMsg := ""
						if scopeStatus == "failed" && err != nil {
							errorMsg = err.Error()
						} else if scopeStatus == "failed" && result != nil {
							errorMsg = result.Message
						}
						// For multi-scope operations, show scope in status message
						// For single-scope operations, scope is shown in title, so don't repeat in status
						scopeForDisplay := item.ScopeLabel + " scope"
						send(components.ItemUpdateMsg{
							Index:        item.ItemIndex,
							Name:         item.ProperName,
							Status:       scopeStatus,
							Message:      scopeMessage,
							ErrorMessage: errorMsg,
							Variants:     scopeVariants,
							Scope:        scopeForDisplay, // Always pass scope for multi-scope operations
						})

						// Update progress percentage
						percent := float64(item.ItemIndex+1) / float64(len(operationItems)) * 100
						send(components.ProgressUpdateMsg{Percent: percent})
					}
				} else {
					// Single scope - process each found font
					for i, fontInfo := range foundFonts {
						// Use proper font name from pre-extracted map
						properFontName := fontInfo.ProperName
						fontName := fontInfo.SearchName

						// Send initial "in_progress" message
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
						var allErrors []string // Collect errors from all scopes
						var percent float64    // Progress percentage

						// Special handling for user scope only: check both scopes for better UX
						if len(scopes) == 1 && scopes[0] == platform.UserScope {
							// Resolve Font ID to font name if needed
							searchName := resolveFontNameOrID(fontName, r)
							// Check both scopes efficiently
							machineFonts := findFontFamilyFiles(searchName, fontManager, platform.MachineScope)
							userFonts := findFontFamilyFiles(searchName, fontManager, platform.UserScope)

							// If no direct matches, try repository search to find the font
							if len(machineFonts) == 0 && len(userFonts) == 0 {
								results, err := r.SearchFonts(searchName, "false")
								if err == nil && len(results) > 0 {
									repoSearchName := results[0].Name
									machineFonts = findFontFamilyFiles(repoSearchName, fontManager, platform.MachineScope)
									userFonts = findFontFamilyFiles(repoSearchName, fontManager, platform.UserScope)
								}
							}

							// Handle different scenarios
							if len(userFonts) == 0 && len(machineFonts) == 0 {
								// Font not found in either scope - mark as failed and continue
								fontStatus = "failed"
								statusMessage = "Font not found"
								status.Failed++
								// Send update with proper name and update progress
								// Note: ErrorMessage is set directly, no need to append to allErrors since we continue
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
								percent = float64(i+1) / float64(len(foundFonts)) * 100
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

								// Collect variants for display (only in verbose mode)
								if verbose {
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
									// After successful removal from user scope, check if font still exists in machine scope
									// Resolve Font ID to font name if needed
									searchName := resolveFontNameOrID(fontName, r)
									machineFonts := findFontFamilyFiles(searchName, fontManager, platform.MachineScope)
									if len(machineFonts) == 0 {
										// Try repository search
										results, err := r.SearchFonts(searchName, "false")
										if err == nil && len(results) > 0 {
											repoSearchName := results[0].Name
											machineFonts = findFontFamilyFiles(repoSearchName, fontManager, platform.MachineScope)
										}
									}
									if len(machineFonts) > 0 {
										// Font still exists in machine scope - track for display
										fontsInOppositeScope = append(fontsInOppositeScope, properFontName)
									}
								}
							}

							// Send update message for user scope (single-scope operation)
							// Don't show scope in status - title already indicates it
							errorMsg := ""
							if fontStatus == "failed" && len(allErrors) > 0 {
								errorMsg = allErrors[0]
							}
							send(components.ItemUpdateMsg{
								Index:        i,
								Name:         properFontName,
								Status:       fontStatus,
								Message:      statusMessage,
								ErrorMessage: errorMsg,
								Variants:     allRemovedVariants,
								Scope:        "", // Empty for single-scope operations (cleaner output)
							})
						} else {
							// Handle machine scope (single scope)
							scopeType := scopes[0]
							fontDir := fontManager.GetFontDir(scopeType)

							// Remove the font using the removeFont helper (it will handle Font ID resolution internally)
							result, err := removeFont(
								fontName,
								fontManager,
								scopeType,
								fontDir,
								r,
							)

							if err != nil {
								if strings.Contains(err.Error(), "not found") {
									fontStatus = "failed"
									statusMessage = "Font not found"
									allErrors = append(allErrors, "Font not found")
									status.Failed++
								} else {
									fontStatus = "failed"
									statusMessage = err.Error()
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

								// Collect variants for display (only in verbose mode)
								if verbose {
									allRemovedVariants = append(allRemovedVariants, result.Details...)
								}

								if result.Status == "failed" {
									fontStatus = "failed"
									statusMessage = result.Message
								} else if result.Status == "skipped" && fontStatus != "failed" {
									fontStatus = "skipped"
									statusMessage = result.Message
								}

								// After successful removal, check if font still exists in opposite scope
								if result.Success > 0 {
									var oppositeScope platform.InstallationScope
									switch scopeType {
									case platform.UserScope:
										oppositeScope = platform.MachineScope
									case platform.MachineScope:
										oppositeScope = platform.UserScope
									}
									if oppositeScope != "" {
										// Resolve Font ID to font name if needed
										searchName := resolveFontNameOrID(fontName, r)
										oppositeFonts := findFontFamilyFiles(searchName, fontManager, oppositeScope)
										if len(oppositeFonts) == 0 {
											// Try repository search
											results, err := r.SearchFonts(searchName, "false")
											if err == nil && len(results) > 0 {
												repoSearchName := results[0].Name
												oppositeFonts = findFontFamilyFiles(repoSearchName, fontManager, oppositeScope)
											}
										}
										if len(oppositeFonts) > 0 {
											// Font still exists in opposite scope - track for display
											fontsInOppositeScope = append(fontsInOppositeScope, properFontName)
										}
									}
								}
							}

							// Send update message for single scope
							errorMsg := ""
							if fontStatus == "failed" && len(allErrors) > 0 {
								errorMsg = allErrors[0]
							}
							// For single-scope operations, don't show scope in status (title already shows it)
							// Pass empty scope so progress bar doesn't display it
							send(components.ItemUpdateMsg{
								Index:        i,
								Name:         properFontName,
								Status:       fontStatus,
								Message:      statusMessage,
								ErrorMessage: errorMsg,
								Variants:     allRemovedVariants,
								Scope:        "", // Empty for single-scope operations (cleaner output)
							})
						}

						// Update progress percentage
						percent = float64(i+1) / float64(len(foundFonts)) * 100
						send(components.ProgressUpdateMsg{Percent: percent})
					}
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
			if IsDebug() {
				// In debug mode, show technical details to console
				scopeDisplay := scopeFlag
				if scopeDisplay == "" {
					scopeDisplay = "user"
				}
				output.GetDebug().Error("The following font(s) were not found installed in the '%s' scope:", scopeDisplay)
				// Deduplicate for display
				seenNotFound := make(map[string]bool)
				for _, fontName := range notFoundFonts {
					normalized := normalizeFontName(fontName)
					if !seenNotFound[normalized] {
						seenNotFound[normalized] = true
						output.GetDebug().Error(" - %s", fontName)
					}
				}
			} else {
				// In normal/verbose mode, show user-friendly message with suggestions
				// Progress bar already ends with \n\n, so we start directly
				// Deduplicate notFoundFonts
				seenNotFound := make(map[string]bool)
				uniqueNotFound := []string{}
				for _, fontName := range notFoundFonts {
					normalized := normalizeFontName(fontName)
					if !seenNotFound[normalized] {
						seenNotFound[normalized] = true
						uniqueNotFound = append(uniqueNotFound, fontName)
					}
				}
				notFoundFonts = uniqueNotFound

				// Format scope for display
				scopeDisplay := scopeFlag
				if scopeDisplay == "" {
					scopeDisplay = "user"
				}

				// Get installed font family names for suggestions
				var installedFontNames []string
				for _, scopeType := range scopes {
					fontDir := fontManager.GetFontDir(scopeType)
					fontFiles, err := platform.ListInstalledFonts(fontDir)
					if err == nil {
						// Extract unique family names from installed fonts
						seenFamilies := make(map[string]bool)
						for _, fontFile := range fontFiles {
							fontPath := filepath.Join(fontDir, fontFile)
							familyName := extractFontFamilyNameFromPath(fontPath)
							normalized := normalizeFontName(familyName)
							if !seenFamilies[normalized] && familyName != "" {
								seenFamilies[normalized] = true
								installedFontNames = append(installedFontNames, familyName)
							}
						}
					}
				}

				// Filter out not found fonts from suggestions (don't suggest fonts that are being removed)
				notFoundNormalized := make(map[string]bool)
				for _, fontName := range notFoundFonts {
					notFoundNormalized[normalizeFontName(fontName)] = true
				}
				filteredInstalledFontNames := []string{}
				for _, fontName := range installedFontNames {
					if !notFoundNormalized[normalizeFontName(fontName)] {
						filteredInstalledFontNames = append(filteredInstalledFontNames, fontName)
					}
				}
				installedFontNames = filteredInstalledFontNames

				// Show header first
				fmt.Printf("%s\n", ui.ErrorText.Render(fmt.Sprintf("The following font(s) were not found installed in the '%s' scope:", scopeDisplay)))

				// Show all not-found fonts grouped together
				if len(notFoundFonts) == 1 {
					// Single font - show with suggestions
					fontName := notFoundFonts[0]
					var similar []string
					if len(installedFontNames) > 0 {
						similar = shared.FindSimilarFonts(fontName, installedFontNames, true) // true = installed fonts
					}

					// Filter out any suggestions that are also in the not found list
					filteredSimilar := []string{}
					for _, suggestion := range similar {
						if !notFoundNormalized[normalizeFontName(suggestion)] {
							filteredSimilar = append(filteredSimilar, suggestion)
						}
					}
					similar = filteredSimilar

					if len(similar) > 0 {
						fmt.Printf("%s\n", ui.ErrorText.Render(fmt.Sprintf("Font '%s' not found.", fontName)))
						fmt.Printf("%s\n\n", ui.Text.Render("Did you mean one of these installed fonts?"))
						for _, similarFont := range similar {
							fmt.Printf(" - %s\n", ui.TableSourceName.Render(similarFont))
						}
					} else {
						fmt.Printf("%s\n", ui.ErrorText.Render(fmt.Sprintf(" - %s", fontName)))
					}
				} else {
					// Multiple fonts - show grouped format
					for _, fontName := range notFoundFonts {
						fmt.Printf(" - %s\n", fontName)
					}

					// Collect all suggestions for all fonts (limit to 12 total)
					const maxSuggestions = 12
					allSimilar := []string{}
					seenSimilar := make(map[string]bool)

					for _, fontName := range notFoundFonts {
						if len(allSimilar) >= maxSuggestions {
							break
						}
						var similar []string
						if len(installedFontNames) > 0 {
							similar = shared.FindSimilarFonts(fontName, installedFontNames, true) // true = installed fonts
						}
						// Deduplicate suggestions and filter out not-found fonts
						for _, suggestion := range similar {
							if len(allSimilar) >= maxSuggestions {
								break
							}
							normalizedSuggestion := normalizeFontName(suggestion)
							if !seenSimilar[suggestion] && !notFoundNormalized[normalizedSuggestion] {
								seenSimilar[suggestion] = true
								allSimilar = append(allSimilar, suggestion)
							}
						}
					}

					// Show consolidated suggestions if any
					if len(allSimilar) > 0 {
						fmt.Println()
						fmt.Printf("%s\n\n", ui.Text.Render("Did you mean one of these installed fonts?"))
						for _, similarFont := range allSimilar {
							fmt.Printf(" - %s\n", ui.TableSourceName.Render(similarFont))
						}
					}
				}

				// Add blank line before "Try using..." message (within section)
				fmt.Println()
				fmt.Printf("Try using 'fontget list' to show currently installed fonts.\n")
				// Section ends with blank line per spacing framework
				fmt.Println()
			}
		}

		// Show fonts that still exist in opposite scope after removal (before status report)
		// Skip in debug mode - already shown in debug logs
		if len(fontsInOppositeScope) > 0 && !IsDebug() {
			// Determine opposite scope name for display
			var oppositeScopeName string
			if len(scopes) == 1 {
				switch scopes[0] {
				case platform.UserScope:
					oppositeScopeName = "machine"
				case platform.MachineScope:
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

				// Use FeedbackWarning (yellow, no bold) for the header message
				fmt.Printf("%s\n", ui.WarningText.Render(fmt.Sprintf("The following font(s) are installed in the '%s' scope, use '--scope %s' to remove them:", oppositeScopeName, oppositeScopeName)))
				for _, fontName := range uniqueFonts {
					fmt.Printf(" - %s\n", fontName)
				}
				// Section ends with blank line per spacing framework
				fmt.Println()
			}
		}

		// Print status report last (after error messages and machine scope info and only if there were actual operations)
		output.PrintStatusReport(output.StatusReport{
			Success:      status.Removed,
			Skipped:      status.Skipped,
			Failed:       status.Failed,
			SuccessLabel: "Removed",
			SkippedLabel: "Skipped",
			FailedLabel:  "Failed",
		}, IsVerbose())

		// Don't return error for removal failures since we already show detailed status report
		// This prevents duplicate error messages while maintaining proper exit codes
		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
	removeCmd.Flags().StringP("scope", "s", "", "Installation scope (user, machine, or all)")
}
