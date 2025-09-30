package cmd

import (
	"fmt"
	"strings"

	"fontget/internal/config"
	"fontget/internal/repo"
	"fontget/internal/ui"

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
			fmt.Printf("\n%s\n\n", ui.RenderError("Either a search query or category is required"))
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

		// Ensure manifest system is initialized (fixes missing sources.json bug)
		if err := config.EnsureManifestExists(); err != nil {
			return fmt.Errorf("failed to initialize sources: %v", err)
		}

		// Get arguments (already validated by Args function)
		category, _ := cmd.Flags().GetString("category")
		refresh, _ := cmd.Flags().GetBool("refresh")
		var query string
		if len(args) > 0 {
			query = args[0]
		}

		GetLogger().Info("Search parameters - Query: %s, Category: %s, Refresh: %v", query, category, refresh)

		// Print styled title first
		fmt.Printf("\n%s\n", ui.PageTitle.Render("Font Search Results"))

		// Get repository (handles source updates internally with spinner if needed)
		var r *repo.Repository
		var err error
		if refresh {
			// Force refresh of font manifest before search
			r, err = repo.GetRepositoryWithRefresh()
		} else {
			r, err = repo.GetRepository()
		}
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
		// Build the search result message
		searchMsg := fmt.Sprintf("Found %d fonts matching: '%s'", len(results), ui.TableSourceName.Render(query))
		if category != "" {
			searchMsg += fmt.Sprintf(" | Filtered by category: '%s'", ui.TableSourceName.Render(category))
		}
		fmt.Printf("\n%s\n\n", searchMsg)

		// Define column widths (match list command style)
		columns := map[string]int{
			"Name":       30, // For display name
			"ID":         30, // For longer font IDs
			"License":    12, // For license type
			"Categories": 15, // For categories
			"Source":     12, // For source name
		}

		// Print header with mauve color
		header := fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s",
			columns["Name"], "Name",
			columns["ID"], "ID",
			columns["License"], "License",
			columns["Categories"], "Categories",
			columns["Source"], "Source")
		fmt.Println(ui.TableHeader.Render(header))
		fmt.Println(ui.FeedbackText.Render(strings.Repeat("-", len(header))))

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
				ui.TableSourceName.Render(fmt.Sprintf("%-*s", columns["Name"], name)),
				columns["ID"], id,
				columns["License"], license,
				columns["Categories"], categories,
				columns["Source"], result.SourceName)
		}

		// Show when FontGet last updated sources
		if lastUpdated, err := config.GetSourcesLastUpdated(); err == nil && !lastUpdated.IsZero() {
			fmt.Printf("\n%s: %s\n", ui.FeedbackText.Render("Sources Last Updated"), lastUpdated.Format("Mon, 02 Jan 2006 15:04:05 MST"))
		}

		GetLogger().Info("Font search operation completed successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.Flags().StringP("category", "c", "", "Filter by font category (Sans Serif, Serif, Display, Handwriting, Monospace, Other)")

	// Hidden flag for development/testing only
	searchCmd.Flags().Bool("refresh", false, "Force refresh of font manifest before search")
	searchCmd.Flags().MarkHidden("refresh")

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
