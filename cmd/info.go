package cmd

import (
	"fmt"
	"strings"

	"fontget/internal/repo"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var infoCmd = &cobra.Command{
	Use:   "info <font-id>",
	Short: "Display detailed information about a font",
	Long:  "Shows comprehensive information about a font including its variants, license, metadata, and categories.",
	Example: `  fontget info "Noto Sans"
  fontget info "Roboto" --license
  fontget info "Open Sans" -f
  fontget info "Fira Sans" --metadata
  `,
	Args: cobra.MaximumNArgs(1),
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
		// Check for font ID
		if len(args) == 0 || args[0] == "" {
			red := color.New(color.FgRed).SprintFunc()
			fmt.Printf("\n%s\n\n", red("A font ID is required"))
			fmt.Println(cmd.Long)
			fmt.Println()
			fmt.Println("Usage:")
			fmt.Printf("  %s\n\n", cmd.UseLine())
			fmt.Println("Flags:")
			cmd.Flags().VisitAll(func(flag *pflag.Flag) {
				if flag.Shorthand != "" {
					fmt.Printf("  -%s, --%s\t%s\n", flag.Shorthand, flag.Name, flag.Usage)
				} else {
					fmt.Printf("  --%s\t%s\n", flag.Name, flag.Usage)
				}
			})
			if cmd.Example != "" {
				fmt.Println("\nExamples:")
				fmt.Println(cmd.Example)
			}
			return nil
		}

		fontID := args[0]

		// Get flags
		showLicense, _ := cmd.Flags().GetBool("license")
		showFiles, _ := cmd.Flags().GetBool("files")
		showMetadata, _ := cmd.Flags().GetBool("metadata")

		// If no specific flags are set, show all info
		if !showLicense && !showFiles && !showMetadata {
			showLicense = true
			showFiles = true
			showMetadata = true
		}

		// Get repository
		r, err := repo.GetRepository()
		if err != nil {
			return fmt.Errorf("failed to initialize repository: %w", err)
		}

		// Get manifest
		manifest, err := r.GetManifest()
		if err != nil {
			return fmt.Errorf("failed to get manifest: %w", err)
		}

		// Find font in manifest
		font, exists := manifest.Sources["google-fonts"].Fonts[fontID]
		if !exists {
			red := color.New(color.FgRed).SprintFunc()
			return fmt.Errorf("%s", red(fmt.Sprintf("Font '%s' not found", fontID)))
		}

		// Print font information
		fmt.Printf("\nFont Information for '%s'\n", fontID)
		fmt.Println(strings.Repeat("-", 40))

		// Always show category as it's a single value
		if len(font.Categories) > 0 {
			fmt.Printf("\nCategory: %s\n", font.Categories[0])
		}

		if showLicense {
			fmt.Printf("\nLicense: %s\n", font.License)
		}

		if showFiles {
			fmt.Printf("\nFiles:\n")
			for variant, url := range font.Files {
				fmt.Printf("  - %s: %s\n", variant, url)
			}
		}

		if showMetadata {
			fmt.Printf("\nMetadata:\n")
			fmt.Printf("  Last Modified: %s\n", font.LastModified)
			if font.Description != "" {
				fmt.Printf("  Description: %s\n", font.Description)
			}
			fmt.Printf("  Source URL: %s\n", font.SourceURL)
			fmt.Printf("  Metadata URL: %s\n", font.MetadataURL)
			fmt.Printf("  Popularity: %d\n", font.Popularity)
		}

		fmt.Println()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)

	// Add flags
	infoCmd.Flags().BoolP("license", "l", false, "Show license information")
	infoCmd.Flags().BoolP("files", "f", false, "Show font files")
	infoCmd.Flags().BoolP("metadata", "m", false, "Show metadata information")
}
