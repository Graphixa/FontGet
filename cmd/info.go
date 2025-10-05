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
  fontget info "Roboto" -l
  fontget info "Fira Sans" -m`,
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
		showMetadata, _ := cmd.Flags().GetBool("metadata")

		// If no specific flags are set, show all info
		showAll := !showLicense && !showMetadata
		if showAll {
			showLicense = true
			showMetadata = true
		}

		output.GetVerbose().Info("Display options - License: %v, Metadata: %v, Show All: %v", showLicense, showMetadata, showAll)
		output.GetDebug().State("Info display flags: license=%v, metadata=%v, showAll=%v", showLicense, showMetadata, showAll)

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

		// Find font in manifest
		output.GetVerbose().Info("Searching for font '%s' in manifest", fontID)
		output.GetDebug().State("Searching %d sources for font '%s'", len(manifest.Sources), fontID)
		var fontSource string
		var font repo.FontInfo
		found := false
		for sourceKey, source := range manifest.Sources {
			output.GetDebug().State("Checking source '%s' with %d fonts", sourceKey, len(source.Fonts))
			if f, ok := source.Fonts[fontID]; ok {
				font = f
				fontSource = sourceKey
				found = true
				output.GetVerbose().Info("Found font '%s' in source '%s'", fontID, sourceKey)
				output.GetDebug().State("Font found in source: %s", sourceKey)
				break
			}
		}
		if !found {
			GetLogger().Error("Font '%s' not found", fontID)
			output.GetVerbose().Error("Font '%s' not found in any source", fontID)
			output.GetDebug().Error("Font lookup failed: '%s' not found in %d sources", fontID, len(manifest.Sources))
			return fmt.Errorf("%s", ui.RenderError(fmt.Sprintf("Font '%s' not found", fontID)))
		}

		// Note: License-only mode now uses cards instead of raw text display

		// Note: Metadata-only mode now uses cards instead of raw text display

		// Display font information using card components
		var cards []components.Card

		// Always show font details card
		category := "Unknown"
		if len(font.Categories) > 0 {
			category = font.Categories[0]
		}
		cards = append(cards, components.FontDetailsCard(font.Name, fontID, category))

		if showLicense {
			licenseURL := ""
			// Handle different sources for license URLs
			if fontSource == "google-fonts" {
				if strings.ToLower(font.License) == "ofl" {
					// For OFL fonts, use the GitHub raw URL
					id := strings.ToLower(strings.ReplaceAll(fontID, " ", ""))
					licenseURL = "https://raw.githubusercontent.com/google/fonts/main/ofl/" + id + "/OFL.txt"
				} else if font.SourceURL != "" {
					// For other Google Fonts, use the source URL with license anchor
					licenseURL = font.SourceURL + "#license"
				}
			} else if font.SourceURL != "" {
				// Fallback to source URL
				licenseURL = font.SourceURL
			}

			cards = append(cards, components.LicenseInfoCard(font.License, licenseURL))
		}

		// Always show source information when showing all info
		if showAll {
			// Get source information from the manifest
			if source, exists := manifest.Sources[fontSource]; exists {
				// Use the source name and description from the manifest
				sourceName := source.Name
				if sourceName == "" {
					sourceName = fontSource // Fallback to source key
				}

				// Use the source URL from the manifest, or fall back to font's source URL
				sourceURL := source.URL
				if sourceURL == "" {
					sourceURL = font.SourceURL
				}

				// Only show source info if we have a URL
				if sourceURL != "" {
					cards = append(cards, components.SourceInfoCard(sourceName, sourceURL, source.Description))
				}
			} else {
				// Fallback for unknown sources
				if font.SourceURL != "" {
					cards = append(cards, components.SourceInfoCard(fontSource, font.SourceURL, ""))
				}
			}
		}

		if showMetadata {
			// Create metadata card with popularity as string
			popularityStr := ""
			if font.Popularity > 0 {
				popularityStr = fmt.Sprintf("%d", font.Popularity)
			}
			lastModifiedStr := font.LastModified.Format("2006-01-02T15:04:05Z")
			cards = append(cards, components.MetadataCard(lastModifiedStr, font.SourceURL, popularityStr))
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
	infoCmd.Flags().BoolP("metadata", "m", false, "Show metadata information only")
}
