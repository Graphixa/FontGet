package cmd

import (
	"fmt"
	"strings"

	"fontget/internal/repo"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for fonts that are downloadable with FontGet",
	Long:  "Searches for fonts from Google Fonts and other added sources.",
	Example: `  fontget search fira
  fontget search "Fira Sans"
  fontget search -c "Sans Serif"
  fontget search "roboto" -c "Sans Serif"`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Get flags
		category, _ := cmd.Flags().GetString("category")

		// Get query from args
		var query string
		if len(args) > 0 {
			query = args[0]
		}

		// Validate query
		if query == "" && category == "" {
			red := color.New(color.FgRed).SprintFunc()
			fmt.Printf("\n%s\n\n", red("Either a search query or category is required"))
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
		GetLogger().Info("Starting font search operation")

		// Double check args to prevent panic
		category, _ := cmd.Flags().GetString("category")
		var query string
		if len(args) > 0 {
			query = args[0]
		}
		if query == "" && category == "" {
			return nil // Args validator will have already shown the help
		}

		GetLogger().Info("Search parameters - Query: %s, Category: %s", query, category)

		// Get repository
		r, err := repo.GetRepository()
		if err != nil {
			GetLogger().Error("Failed to initialize repository: %v", err)
			return fmt.Errorf("failed to initialize repository: %w", err)
		}

		// Search fonts
		results, err := r.SearchFonts(query, category)
		if err != nil {
			GetLogger().Error("Failed to search fonts: %v", err)
			return fmt.Errorf("failed to search fonts: %w", err)
		}

		// Print results in a table format
		yellow := color.New(color.FgYellow, color.Bold).SprintFunc()
		cyan := color.New(color.FgCyan).SprintFunc()

		// Print search summary
		fmt.Printf("\nFound %d fonts matching '%s'", len(results), yellow(query))
		if category != "" {
			fmt.Printf(" in category '%s'", yellow(category))
		}
		fmt.Println("\n")

		// Define column widths (match list command style)
		columns := map[string]int{
			"Name":       30, // For display name
			"ID":         30, // For longer font IDs
			"License":    12, // For license type
			"Categories": 15, // For categories
			"Source":     12, // For source name
		}

		// Print header (plain, no color)
		header := fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s",
			columns["Name"], "Name",
			columns["ID"], "ID",
			columns["License"], "License",
			columns["Categories"], "Categories",
			columns["Source"], "Source")
		fmt.Println(header)
		fmt.Println(strings.Repeat("-", len(header)))

		for _, result := range results {
			// Format categories
			categories := "N/A"
			if len(result.Categories) > 0 {
				categories = strings.Join(result.Categories, ", ")
			}

			// Format license
			license := "N/A"
			if result.License != "" {
				license = result.License
			}

			// Truncate long names with ellipsis
			name := result.Name
			if len(name) > columns["Name"]-3 {
				name = name[:columns["Name"]-3] + "..."
			}

			// Truncate long IDs with ellipsis
			id := result.ID
			if len(id) > columns["ID"]-3 {
				id = id[:columns["ID"]-3] + "..."
			}

			// Truncate categories if too long
			if len(categories) > columns["Categories"]-3 {
				categories = categories[:columns["Categories"]-3] + "..."
			}

			// Print row: pad first, then apply color only to font name (like list command)
			fmt.Printf("%s %-*s %-*s %-*s %-*s\n",
				yellow(fmt.Sprintf("%-*s", columns["Name"], name)),
				columns["ID"], id,
				columns["License"], license,
				columns["Categories"], categories,
				columns["Source"], result.SourceName)
		}

		// Print manifest info with colors
		manifest, err := r.GetManifest()
		if err != nil {
			GetLogger().Error("Failed to get manifest: %v", err)
			return fmt.Errorf("failed to get manifest: %w", err)
		}

		fmt.Printf("\n%s: %s\n", cyan("Manifest last updated"), manifest.LastUpdated.Format("Mon, 02 Jan 2006 15:04:05 MST"))
		fmt.Printf("%s: %d\n\n", cyan("Total fonts available"), countTotalFonts(manifest))

		GetLogger().Info("Font search operation completed successfully")
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
	searchCmd.Flags().StringP("category", "c", "", "Filter by font category (Sans Serif, Serif, Display, Handwriting, Monospace, Other)")

	// Register completion for both short and long category flags
	categories := []string{
		"Sans Serif",
		"Serif",
		"Display",
		"Handwriting",
		"Monospace",
		"Other",
	}
	searchCmd.RegisterFlagCompletionFunc("category", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return categories, cobra.ShellCompDirectiveNoFileComp
	})
	searchCmd.RegisterFlagCompletionFunc("c", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return categories, cobra.ShellCompDirectiveNoFileComp
	})
}
