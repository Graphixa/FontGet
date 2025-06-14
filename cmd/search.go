package cmd

import (
	"fmt"
	"strings"

	"fontget/internal/repo"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for fonts in the Google Fonts repository",
	Long: `Search for fonts in the Google Fonts repository.
Use quotes for exact matches: "Fira Sans"
Without quotes for partial matches: fira`,
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
		var results []repo.SearchResult
		normalizedQuery := strings.ToLower(query)

		for _, source := range manifest.Sources {
			for id, font := range source.Fonts {
				if isExactMatch {
					if strings.ToLower(font.Name) == normalizedQuery {
						results = append(results, repo.SearchResult{
							Name:     font.Name,
							ID:       id,
							Source:   source.Name,
							Variants: font.Variants,
						})
					}
				} else {
					if strings.Contains(strings.ToLower(font.Name), normalizedQuery) {
						results = append(results, repo.SearchResult{
							Name:     font.Name,
							ID:       id,
							Source:   source.Name,
							Variants: font.Variants,
						})
					}
				}
			}
		}

		if len(results) == 0 {
			fmt.Printf("No fonts found matching '%s'\n", query)
			return nil
		}

		// Print results in a table format
		fmt.Printf("\n%-20s %-30s %-15s %s\n", "Name", "ID", "Source", "Variants")
		fmt.Println(strings.Repeat("-", 80))
		for _, result := range results {
			fmt.Printf("%-20s %-30s %-15s %s\n",
				result.Name,
				result.ID,
				result.Source,
				strings.Join(result.Variants, ", "))
		}
		fmt.Println()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
