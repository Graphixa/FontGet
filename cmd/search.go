package cmd

import (
	"fmt"
	"strings"

	"fontget/internal/config"
	"fontget/internal/output"
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

		// Debug-level information for developers
		output.GetDebug().Message("Debug mode enabled - showing detailed diagnostic information")

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

		// Verbose-level information for users
		output.GetVerbose().Info("Search parameters - Query: %s, Category: %s, Refresh: %v", query, category, refresh)
		output.GetDebug().State("Starting font search with parameters: query='%s', category='%s', refresh=%v", query, category, refresh)

		// Print styled title first
		fmt.Printf("\n%s\n", ui.PageTitle.Render("Font Search Results"))

		// Get repository (handles source updates internally with spinner if needed)
		var r *repo.Repository
		var err error
		if refresh {
			// Force refresh of font manifest before search
			output.GetVerbose().Info("Forcing refresh of font manifest before search")
			output.GetDebug().State("Using GetRepositoryWithRefresh() to force source updates")
			r, err = repo.GetRepositoryWithRefresh()
		} else {
			output.GetVerbose().Info("Using cached font manifest for search")
			output.GetDebug().State("Using GetRepository() with cached sources")
			r, err = repo.GetRepository()
		}
		if err != nil {
			GetLogger().Error("Failed to initialize repository: %v", err)
			output.GetVerbose().Error("Failed to initialize repository: %v", err)
			output.GetDebug().Error("Repository initialization failed: %v", err)
			return fmt.Errorf("failed to initialize repository: %w", err)
		}

		// Search fonts
		output.GetVerbose().Info("Searching fonts with query: '%s' and category: '%s'", query, category)
		output.GetDebug().State("Calling r.SearchFonts(query='%s', category='%s')", query, category)
		results, err := r.SearchFonts(query, category)
		if err != nil {
			GetLogger().Error("Failed to search fonts: %v", err)
			output.GetVerbose().Error("Search operation failed: %v", err)
			output.GetDebug().Error("Font search failed: %v", err)
			return fmt.Errorf("failed to search fonts: %w", err)
		}

		output.GetVerbose().Info("Search completed - found %d results", len(results))
		output.GetDebug().State("Search returned %d font results", len(results))
		// Build the search result message
		searchMsg := fmt.Sprintf("Found %d fonts matching: '%s'", len(results), ui.TableSourceName.Render(query))
		if category != "" {
			searchMsg += fmt.Sprintf(" | Filtered by category: '%s'", ui.TableSourceName.Render(category))
		}
		fmt.Printf("\n%s\n\n", searchMsg)

		// Collect data for dynamic table sizing
		var names, ids, licenses, categories, sources []string
		for _, result := range results {
			// Format categories
			categoriesStr := "N/A"
			if len(result.Categories) > 0 {
				categoriesStr = strings.Join(result.Categories, ", ")
			}

			// Format license
			license := "N/A"
			if result.License != "" {
				license = result.License
			}

			names = append(names, result.Name)
			ids = append(ids, result.ID)
			licenses = append(licenses, license)
			categories = append(categories, categoriesStr)
			sources = append(sources, result.SourceName)
		}

		// Calculate dynamic column widths
		maxName := TableColName
		maxID := TableColID
		maxLicense := TableColLicense
		maxCategories := TableColCategories
		maxSource := TableColSource

		for _, name := range names {
			if len(name) > maxName {
				maxName = len(name)
			}
		}
		for _, id := range ids {
			if len(id) > maxID {
				maxID = len(id)
			}
		}
		for _, license := range licenses {
			if len(license) > maxLicense {
				maxLicense = len(license)
			}
		}
		for _, category := range categories {
			if len(category) > maxCategories {
				maxCategories = len(category)
			}
		}
		for _, source := range sources {
			if len(source) > maxSource {
				maxSource = len(source)
			}
		}

		// Calculate total width needed
		totalWidth := maxName + maxID + maxLicense + maxCategories + maxSource + 4 // +4 for spaces

		// If total width exceeds reasonable maximum (160 chars), use fixed widths
		if totalWidth > 160 {
			fmt.Println(ui.TableHeader.Render(GetSearchTableHeader()))
			fmt.Println(GetTableSeparator())

			for i := range results {
				fmt.Printf("%s %-*s %-*s %-*s %-*s\n",
					ui.TableSourceName.Render(fmt.Sprintf("%-*s", TableColName, truncateString(names[i], TableColName))),
					TableColID, truncateString(ids[i], TableColID),
					TableColLicense, truncateString(licenses[i], TableColLicense),
					TableColCategories, truncateString(categories[i], TableColCategories),
					TableColSource, truncateString(sources[i], TableColSource))
			}
		} else {
			// Use dynamic widths
			fmt.Println(ui.TableHeader.Render(fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s",
				maxName, "Name",
				maxID, "ID",
				maxLicense, "License",
				maxCategories, "Categories",
				maxSource, "Source")))
			fmt.Println(strings.Repeat("-", totalWidth))

			for i := range results {
				fmt.Printf("%s %-*s %-*s %-*s %-*s\n",
					ui.TableSourceName.Render(fmt.Sprintf("%-*s", maxName, names[i])),
					maxID, ids[i],
					maxLicense, licenses[i],
					maxCategories, categories[i],
					maxSource, sources[i])
			}
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
