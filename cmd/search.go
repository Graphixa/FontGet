package cmd

import (
	"fmt"
	"strings"
	"time"

	"fontget/internal/repo"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for fonts in the Google Fonts repository",
	Long: `Search for fonts in the Google Fonts repository.
Use quotes for exact matches: "Fira Sans"
Without quotes for partial matches: fira

The search will automatically update the font manifest if it's older than 24 hours.`,
	Example: `  fontget search fira
  fontget search "Fira Sans"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]
		isExactMatch := strings.HasPrefix(query, "\"") && strings.HasSuffix(query, "\"")

		if isExactMatch {
			// Remove quotes for exact match
			query = strings.Trim(query, "\"")
		}

		// Create progress callback
		progress := func(current, total int, message string) {
			if total > 0 {
				fmt.Printf("\r%s (%d/%d)", message, current, total)
			} else {
				fmt.Printf("\r%s", message)
			}
		}

		// Get manifest with progress updates
		manifest, err := repo.GetManifest(progress)
		if err != nil {
			return fmt.Errorf("failed to get manifest: %w", err)
		}
		fmt.Println() // New line after progress

		// Search through the manifest
		results, err := repo.SearchFonts(query, isExactMatch)
		if err != nil {
			return fmt.Errorf("failed to search fonts: %w", err)
		}

		if len(results) == 0 {
			fmt.Printf("No fonts found matching '%s'\n", query)
			return nil
		}

		// Print results in a table format
		fmt.Printf("\nFound %d fonts matching '%s':\n\n", len(results), query)

		// Calculate column widths
		nameWidth := 35 // Increased for long font names
		sourceWidth := 15
		licenseWidth := 15
		categoriesWidth := 20
		variantsWidth := 30

		// Print header
		fmt.Printf("%-*s %-*s %-*s %-*s %s\n",
			nameWidth, "Name",
			sourceWidth, "Source",
			licenseWidth, "License",
			categoriesWidth, "Categories",
			"Variants")
		fmt.Println(strings.Repeat("-", nameWidth+sourceWidth+licenseWidth+categoriesWidth+variantsWidth+4))

		for _, result := range results {
			// Format categories
			categories := "N/A"
			if len(result.Categories) > 0 {
				categories = strings.Join(result.Categories, ", ")
			}

			// Format variants (limit to 3)
			variants := "N/A"
			if len(result.Variants) > 0 {
				if len(result.Variants) > 3 {
					variants = fmt.Sprintf("%s, ... (%d more)",
						strings.Join(result.Variants[:3], ", "),
						len(result.Variants)-3)
				} else {
					variants = strings.Join(result.Variants, ", ")
				}
			}

			// Format license
			license := "N/A"
			if result.License != "" {
				license = result.License
			}

			// Truncate long names with ellipsis
			name := result.Name
			if len(name) > nameWidth-3 {
				name = name[:nameWidth-3] + "..."
			}

			fmt.Printf("%-*s %-*s %-*s %-*s %s\n",
				nameWidth, name,
				sourceWidth, result.Source,
				licenseWidth, license,
				categoriesWidth, categories,
				variants)
		}
		fmt.Println()

		// Print manifest info
		fmt.Printf("Manifest last updated: %s\n", manifest.LastUpdated.Format(time.RFC1123))
		fmt.Printf("Total fonts available: %d\n", countTotalFonts(manifest))

		return nil
	},
}

// countTotalFonts counts the total number of fonts in the manifest
func countTotalFonts(manifest *repo.FontManifest) int {
	total := 0
	for _, source := range manifest.Sources {
		total += len(source.Fonts)
	}
	return total
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
