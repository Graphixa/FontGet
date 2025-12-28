package cmd

import (
	"fmt"
	"strings"

	"fontget/internal/cmdutils"
	"fontget/internal/components"
	"fontget/internal/output"
	"fontget/internal/repo"
	"fontget/internal/shared"
	"fontget/internal/ui"

	"github.com/spf13/cobra"
)

// Placeholder constants
const (
	InfoPlaceholderUnknown = "Unknown"
)

var infoCmd = &cobra.Command{
	Use:   "info <font-id>",
	Short: "Display detailed information about a font",
	Long: `Display detailed information about a font.

Shows font metadata including name, ID, source, variants, license, and categories.
Use --license to show only license information.`,
	Example: `  fontget info "Noto Sans"
  fontget info "Roboto" -l`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
			fmt.Printf("\n%s\n", ui.RenderError("A font ID is required"))
			fmt.Printf("Use 'fontget info --help' for more information.\n\n")
			return nil
		}
		return nil
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Get repository
		r, err := repo.GetRepository()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		// Get all fonts
		results, err := r.SearchFonts("", "")
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		// Filter and return font names
		var completions []string
		for _, result := range results {
			if strings.HasPrefix(strings.ToLower(result.Name), strings.ToLower(toComplete)) {
				completions = append(completions, result.Name)
			}
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		GetLogger().Info("Starting font info operation")

		// Get font ID from arguments
		if len(args) == 0 {
			return fmt.Errorf("font ID or name is required")
		}
		fontID := args[0]

		// Log font ID parameter (always log to file)
		GetLogger().Info("Font info parameters - Font ID: %s", fontID)

		// Debug-level information for developers
		// Note: Suppressed to avoid TUI interference
		// output.GetDebug().Message("Debug mode enabled - showing detailed diagnostic information")

		// Ensure manifest system is initialized (fixes missing sources.json bug)
		if err := cmdutils.EnsureManifestInitialized(func() cmdutils.Logger { return GetLogger() }); err != nil {
			return err
		}

		// Double check args to prevent panic
		if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
			return nil // Args validator will have already shown the help
		}

		output.GetVerbose().Info("Retrieving information for font: %s", fontID)
		output.GetDebug().State("Starting font info lookup for: %s", fontID)

		// Get flags
		showLicense, _ := cmd.Flags().GetBool("license")

		// If no specific flags are set, show all info (both font details and license)
		// If -l flag is set, show only license
		showAll := !showLicense

		output.GetVerbose().Info("Display options - License: %v, Show All: %v", showLicense, showAll)
		output.GetDebug().State("Info display flags: license=%v, showAll=%v", showLicense, showAll)

		// Get repository (using cached manifest)
		output.GetVerbose().Info("Initializing repository for font lookup")
		r, err := cmdutils.GetRepository(false, GetLogger())
		if err != nil {
			return err
		}

		// Get manifest
		output.GetVerbose().Info("Retrieving font manifest")
		output.GetDebug().State("Calling r.GetManifest()")
		manifest, err := r.GetManifest()
		if err != nil {
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("r.GetManifest() failed: %v", err)
			return fmt.Errorf("unable to load font repository: %v", err)
		}

		// First check if it's an exact font ID match
		output.GetVerbose().Info("Checking for exact font ID match: '%s'", fontID)
		output.GetDebug().State("Searching %d sources for exact font ID '%s'", len(manifest.Sources), fontID)
		var font repo.FontInfo
		found := false
		for sourceKey, source := range manifest.Sources {
			output.GetDebug().State("Checking source '%s' with %d fonts", sourceKey, len(source.Fonts))
			if f, ok := source.Fonts[fontID]; ok {
				font = f
				found = true
				output.GetVerbose().Info("Found exact font ID '%s' in source '%s'", fontID, sourceKey)
				output.GetDebug().State("Font found in source: %s", sourceKey)
				break
			}
		}

		if !found {
			// Try to find multiple matches using the same logic as add command
			output.GetVerbose().Info("No exact match found, searching for multiple matches")
			output.GetDebug().State("Calling repo.FindFontMatches for: %s", fontID)

			matches, matchErr := repo.FindFontMatches(fontID)
			if matchErr != nil {
				output.GetVerbose().Error("%v", matchErr)
				output.GetDebug().Error("repo.FindFontMatches() failed: %v", matchErr)
				return fmt.Errorf("unable to search for font: %v", matchErr)
			}

			if len(matches) == 0 {
				// No matches found, show similar fonts
				GetLogger().Error("Font '%s' not found", fontID)
				output.GetVerbose().Error("Font '%s' not found in any source", fontID)
				output.GetDebug().Error("Font lookup failed: '%s' not found in %d sources", fontID, len(manifest.Sources))

				// Try to find similar fonts using the same logic as add command
				output.GetVerbose().Info("Searching for similar fonts to '%s'", fontID)
				output.GetDebug().State("Calling shared.FindSimilarFonts for font: %s", fontID)

				// Get all available fonts for suggestions (use same method as add command)
				allFonts := repo.GetAllFontsCached()
				if len(allFonts) == 0 {
					output.GetDebug().Error("Could not get list of available fonts for suggestions")
					return fmt.Errorf("%s", ui.RenderError(fmt.Sprintf("Font '%s' not found", fontID)))
				}

				// Find similar fonts using the same method as add command
				similar := shared.FindSimilarFonts(fontID, allFonts, false) // false = repository fonts
				GetLogger().Info("Found %d similar fonts for %s", len(similar), fontID)

				// Show suggestions
				showFontNotFoundWithSuggestions(fontID, similar)
				return nil
			} else if len(matches) == 1 {
				// Single match found, use it
				output.GetVerbose().Info("Found single match for '%s': %s", fontID, matches[0].ID)
				output.GetDebug().State("Using single match: %s from source %s", matches[0].ID, matches[0].Source)

				// Get the font info for the single match from the manifest
				output.GetDebug().State("Looking up font info for ID: %s", matches[0].ID)
				for sourceKey, source := range manifest.Sources {
					if f, ok := source.Fonts[matches[0].ID]; ok {
						font = f
						found = true
						output.GetVerbose().Info("Found font info for '%s' in source '%s'", matches[0].ID, sourceKey)
						output.GetDebug().State("Font info found in source: %s", sourceKey)
						break
					}
				}

				if !found {
					output.GetDebug().Error("Font info not found for ID %s", matches[0].ID)
					return fmt.Errorf("no font information available for %s", matches[0].ID)
				}
			} else {
				// Multiple matches found, show them and ask for specific ID
				output.GetVerbose().Info("Found %d matches for '%s'", len(matches), fontID)
				output.GetDebug().State("Multiple matches found, showing options")

				showMultipleMatchesAndExit(fontID, matches)
				return nil
			}
		}

		// Note: License-only mode now uses cards instead of raw text display

		// Note: Metadata-only mode now uses cards instead of raw text display

		// Display font information using card components
		fmt.Println() // Add space between command and first card
		var cards []components.Card

		// Show font details card only if showing all info
		if showAll {
			category := InfoPlaceholderUnknown
			if len(font.Categories) > 0 {
				category = font.Categories[0]
			}

			// Format tags
			tags := ""
			if len(font.Tags) > 0 {
				tags = strings.Join(font.Tags, ", ")
			}

			// Format last modified date
			lastModified := ""
			if !font.LastModified.IsZero() {
				lastModified = font.LastModified.Format("02/01/2006 - 15:04")
			}

			// Format popularity
			popularity := ""
			if font.Popularity > 0 {
				popularity = fmt.Sprintf("%d", font.Popularity)
			}

			cards = append(cards, components.FontDetailsCard(
				font.Name,
				fontID,
				category,
				tags,
				lastModified,
				font.SourceURL,
				popularity,
			))
		}

		// Show license card if showing all info OR if license-only flag is set
		if showAll || showLicense {
			licenseURL := ""
			// Use the specific license URL if available from the source data
			if font.LicenseURL != "" {
				licenseURL = font.LicenseURL
			} else if font.SourceURL != "" {
				// Fallback to source URL if no specific license URL is provided
				licenseURL = font.SourceURL
			}

			cards = append(cards, components.LicenseInfoCard(font.License, licenseURL))
		}

		// Render all cards
		if len(cards) > 0 {
			cardModel := components.NewCardModel("", cards)
			fmt.Println(cardModel.Render())
		}

		GetLogger().Info("Font info operation completed successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)

	// Add flags
	infoCmd.Flags().BoolP("license", "l", false, "Show license information only")
}
