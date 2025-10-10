package cmd

import (
	"fmt"
	"strings"

	"fontget/internal/components"
	"fontget/internal/config"
	"fontget/internal/output"
	"fontget/internal/repo"
	"fontget/internal/ui"

	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info <font-id> [flags]",
	Short: "Display detailed information about a font",
	Long:  "Show comprehensive information about a font including variants, license, and metadata.",
	Example: `  fontget info "Noto Sans"
  fontget info "Roboto" -l`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
			fmt.Printf("\n%s\n\n", ui.RenderError("A font ID is required"))
			return cmd.Help()
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

		// Debug-level information for developers
		output.GetDebug().Message("Debug mode enabled - showing detailed diagnostic information")

		// Ensure manifest system is initialized (fixes missing sources.json bug)
		if err := config.EnsureManifestExists(); err != nil {
			return fmt.Errorf("failed to initialize sources: %v", err)
		}

		// Double check args to prevent panic
		if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
			return nil // Args validator will have already shown the help
		}

		fontID := args[0]
		GetLogger().Info("Retrieving info for font: %s", fontID)
		output.GetVerbose().Info("Retrieving information for font: %s", fontID)
		output.GetDebug().State("Starting font info lookup for: %s", fontID)

		// Get flags
		showLicense, _ := cmd.Flags().GetBool("license")

		// If no specific flags are set, show all info (both font details and license)
		// If -l flag is set, show only license
		showAll := !showLicense

		output.GetVerbose().Info("Display options - License: %v, Show All: %v", showLicense, showAll)
		output.GetDebug().State("Info display flags: license=%v, showAll=%v", showLicense, showAll)

		// Get repository
		output.GetVerbose().Info("Initializing repository for font lookup")
		output.GetDebug().State("Calling repo.GetRepository()")
		r, err := repo.GetRepository()
		if err != nil {
			GetLogger().Error("Failed to initialize repository: %v", err)
			output.GetVerbose().Error("Failed to initialize repository: %v", err)
			output.GetDebug().Error("Repository initialization failed: %v", err)
			return fmt.Errorf("failed to initialize repository: %w", err)
		}

		// Get manifest
		output.GetVerbose().Info("Retrieving font manifest")
		output.GetDebug().State("Calling r.GetManifest()")
		manifest, err := r.GetManifest()
		if err != nil {
			GetLogger().Error("Failed to get manifest: %v", err)
			output.GetVerbose().Error("Failed to get manifest: %v", err)
			output.GetDebug().Error("Manifest retrieval failed: %v", err)
			return fmt.Errorf("failed to get manifest: %w", err)
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
				output.GetDebug().Error("FindFontMatches failed: %v", matchErr)
				return fmt.Errorf("failed to search for font: %w", matchErr)
			}

			if len(matches) == 0 {
				// No matches found, show similar fonts
				GetLogger().Error("Font '%s' not found", fontID)
				output.GetVerbose().Error("Font '%s' not found in any source", fontID)
				output.GetDebug().Error("Font lookup failed: '%s' not found in %d sources", fontID, len(manifest.Sources))

				// Try to find similar fonts using the same logic as add command
				output.GetVerbose().Info("Searching for similar fonts to '%s'", fontID)
				output.GetDebug().State("Calling findSimilarFonts for font: %s", fontID)

				// Get all available fonts for suggestions (use same method as add command)
				allFonts := repo.GetAllFontsCached()
				if len(allFonts) == 0 {
					output.GetDebug().Error("Could not get list of available fonts for suggestions")
					return fmt.Errorf("%s", ui.RenderError(fmt.Sprintf("Font '%s' not found", fontID)))
				}

				// Find similar fonts using the same method as add command
				similar := findSimilarFonts(fontID, allFonts, false) // false = repository fonts
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
			category := "Unknown"
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
