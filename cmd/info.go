package cmd

import (
	"fmt"
	"strings"

	"fontget/internal/config"
	"fontget/internal/license"
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

		// If only --license is set, just cat the license text and return
		if showLicense && !showMetadata {
			// Find license URL
			licenseURLFromPackage := license.GetLicenseURL(fontID, fontSource)
			if licenseURLFromPackage != "" {
				licenseText, err := license.FetchLicenseText(licenseURLFromPackage)
				if err != nil {
					license.HandleLicenseError(fontID, err)
					return nil
				}
				fmt.Println() // Add blank line for visual separation
				_ = license.DisplayLicenseText(licenseText)
				return nil
			}
			license.HandleLicenseError(fontID, nil)
			return nil
		}

		// If only --metadata is set, just cat the metadata and return
		if showMetadata && !showLicense {
			// Fetch metadata from the metadata URL
			if font.MetadataURL != "" {
				metadataText, err := license.FetchLicenseText(font.MetadataURL) // Reuse the same HTTP fetching function
				if err != nil {
					fmt.Printf("Metadata not found for \"%s\". Error: %v\n", fontID, err)
					return nil
				}
				fmt.Println() // Add blank line for visual separation
				_ = license.DisplayLicenseText(metadataText)
				return nil
			}
			fmt.Printf("Metadata URL not available for \"%s\"\n", fontID)
			return nil
		}

		// Helper for colored headers

		// Print font information
		fmt.Printf("\n%s %s\n", ui.FormLabel.Render("Font Name:"), font.Name)

		// Always show category as it's a single value
		if len(font.Categories) > 0 {
			fmt.Printf("\n%s %s\n", ui.FormLabel.Render("Category:"), font.Categories[0])
		}

		if showLicense {
			licenseURL := ""
			// Always show the raw license URL for Google Fonts OFL fonts
			if fontSource == "google-fonts" && strings.ToLower(font.License) == "ofl" {
				id := strings.ToLower(strings.ReplaceAll(fontID, " ", ""))
				licenseURL = "https://raw.githubusercontent.com/google/fonts/main/ofl/" + id + "/OFL.txt"
			} else if font.SourceURL != "" && strings.Contains(font.SourceURL, "fonts.google.com") {
				licenseURL = font.SourceURL + "#license"
			}
			if licenseURL != "" {
				fmt.Printf("\n%s %s - %s\n", ui.FormLabel.Render("License:"), font.License, licenseURL)
			} else {
				fmt.Printf("\n%s %s\n", ui.FormLabel.Render("License:"), font.License)
			}
		}

		// Always show files when showing all info
		if showAll {
			fmt.Printf("\n%s\n", ui.FormLabel.Render("Files:"))
			for variant, url := range font.Files {
				fmt.Printf("  - %s: %s\n", variant, url)
			}
		}

		if showMetadata {
			fmt.Printf("\n%s\n", ui.FormLabel.Render("Metadata:"))
			fmt.Printf(" - Last Modified: %s\n", font.LastModified)
			if font.Description != "" {
				fmt.Printf(" - Description: %s\n", font.Description)
			}
			fmt.Printf(" - Source URL: %s\n", font.SourceURL)
			fmt.Printf(" - Metadata URL: %s\n", font.MetadataURL)
			fmt.Printf(" - Popularity: %d\n", font.Popularity)
		}

		fmt.Println()
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
