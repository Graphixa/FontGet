package cmd

import (
	"fmt"
	"math"
	"strings"

	"fontget/internal/config"
	"fontget/internal/output"
	"fontget/internal/repo"
	"fontget/internal/ui"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for available fonts",
	Long: `Search for fonts from all configured sources.

The search query matches font names. Use the --category flag to filter by
font category (e.g., "Sans Serif", "Serif", "Monospace").

Examples:
  fontget search fira              # Search for fonts matching "fira"
  fontget search -c "Sans Serif"   # List all fonts in "Sans Serif" category
  fontget search -c                # List all available categories`,
	Example: `  fontget search fira
  fontget search "Fira Sans"
  fontget search -c "Sans Serif"
  fontget search "roboto" -c "Sans Serif"
  fontget search -c`,
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
		// Note: Suppressed to avoid TUI interference
		// output.GetDebug().Message("Debug mode enabled - showing detailed diagnostic information")

		// Ensure manifest system is initialized (fixes missing sources.json bug)
		if err := config.EnsureManifestExists(); err != nil {
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("config.EnsureManifestExists() failed: %v", err)
			return fmt.Errorf("unable to load font repository: %v", err)
		}

		// Get arguments (already validated by Args function)
		category, _ := cmd.Flags().GetString("category")
		refresh, _ := cmd.Flags().GetBool("refresh")
		var query string
		if len(args) > 0 {
			query = args[0]
		}

		// Handle category-only mode (show all categories) - early return
		// Check if category flag was provided but no value was given (NoOptDefVal = "list")
		// AND no arguments were provided (meaning user just used -c without a value)
		if (cmd.Flags().Changed("category") || cmd.Flags().Changed("c")) && category == "list" && len(args) == 0 {
			// Show all available categories
			return showAllCategories()
		}

		// If category is "list" but we have arguments, it means the user provided a category value
		// but NoOptDefVal is overriding it. We need to extract the actual category from args
		if category == "list" && len(args) > 0 {
			// The first argument is actually the category value
			category = args[0]
			// Remove the first argument from args since it's the category, not a query
			if len(args) > 1 {
				query = args[1]
			} else {
				query = ""
			}
		}

		GetLogger().Info("Search parameters - Query: %s, Category: %s, Refresh: %v", query, category, refresh)

		// Verbose-level information for users - show operational details
		if IsVerbose() && !IsDebug() {
			output.GetVerbose().Info("Search parameters - Query: %s, Category: %s, Refresh: %v", query, category, refresh)
		}
		output.GetDebug().State("Starting font search with parameters: query='%s', category='%s', refresh=%v", query, category, refresh)

		// Print styled title first
		fmt.Printf("\n%s\n", ui.PageTitle.Render("Search Results"))

		// Get repository (handles source updates internally with spinner if needed)
		var r *repo.Repository
		var err error
		if refresh {
			// Force refresh of font manifest before search
			if IsVerbose() && !IsDebug() {
				output.GetVerbose().Info("Forcing refresh of font manifest before search")
			}
			output.GetDebug().State("Using GetRepositoryWithRefresh() to force source updates")
			r, err = repo.GetRepositoryWithRefresh()
		} else {
			if IsVerbose() && !IsDebug() {
				output.GetVerbose().Info("Using cached font manifest for search")
			}
			output.GetDebug().State("Using GetRepository() with cached sources")
			r, err = repo.GetRepository()
		}
		if err != nil {
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("repo.GetRepository() failed: %v", err)
			return fmt.Errorf("unable to load font repository: %v", err)
		}

		// Search fonts
		if IsVerbose() && !IsDebug() {
			output.GetVerbose().Info("Searching fonts with query: '%s' and category: '%s'", query, category)
		}
		output.GetDebug().State("Calling r.SearchFonts(query='%s', category='%s')", query, category)
		results, err := r.SearchFonts(query, category)
		if err != nil {
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("r.SearchFonts() failed: %v", err)
			return fmt.Errorf("unable to search fonts: %v", err)
		}

		if IsVerbose() && !IsDebug() {
			output.GetVerbose().Info("Search completed - found %d results", len(results))
		}
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
				// Add detailed score breakdown for debugging (only when --debug flag is enabled)
				// This shows the base score + match type + popularity breakdown
				baseScore := 50 // Base score from config
				// Check if popularity scoring is enabled
				userPrefs := config.GetUserPreferences()
				usePopularity := userPrefs.Configuration.UsePopularitySort
				var popularityBonus int
				if usePopularity {
					popularityBonus = results[i].Popularity / 2 // Correct popularity calculation (divisor = 2)
				} else {
					popularityBonus = 0 // No popularity bonus when disabled
				}

				// Calculate match bonus by reverse-engineering from final score
				// Account for length penalty by estimating the original score before penalty
				queryLength := len(query)
				fontLength := len(results[i].Name)
				lengthRatio := float64(queryLength) / float64(fontLength)

				// Estimate original score before length penalty
				var estimatedOriginalScore int
				if lengthRatio < 0.5 {
					// Length penalty was applied, estimate original score
					lengthPenaltyEffectiveness := math.Sqrt(lengthRatio)
					estimatedOriginalScore = int(float64(results[i].Score) / lengthPenaltyEffectiveness)
				} else {
					// No length penalty applied
					estimatedOriginalScore = results[i].Score
				}

				matchBonus := estimatedOriginalScore - baseScore - popularityBonus

				// Use the actual match type from the search results
				matchType := results[i].MatchType

				output.GetDebug().State("Font '%s': Position=%d/%d, Base=%d, Match=%s(+%d), Popularity=%d(+%d), Final=%d",
					results[i].Name, i+1, len(results), baseScore, matchType, matchBonus, results[i].Popularity, popularityBonus, results[i].Score)
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
				// Add detailed score breakdown for debugging (only when --debug flag is enabled)
				// This shows the base score + match type + popularity breakdown
				baseScore := 50 // Base score from config
				// Check if popularity scoring is enabled
				userPrefs := config.GetUserPreferences()
				usePopularity := userPrefs.Configuration.UsePopularitySort
				var popularityBonus int
				if usePopularity {
					popularityBonus = results[i].Popularity / 2 // Correct popularity calculation (divisor = 2)
				} else {
					popularityBonus = 0 // No popularity bonus when disabled
				}

				// Calculate match bonus by reverse-engineering from final score
				// Account for length penalty by estimating the original score before penalty
				queryLength := len(query)
				fontLength := len(results[i].Name)
				lengthRatio := float64(queryLength) / float64(fontLength)

				// Estimate original score before length penalty
				var estimatedOriginalScore int
				if lengthRatio < 0.5 {
					// Length penalty was applied, estimate original score
					lengthPenaltyEffectiveness := math.Sqrt(lengthRatio)
					estimatedOriginalScore = int(float64(results[i].Score) / lengthPenaltyEffectiveness)
				} else {
					// No length penalty applied
					estimatedOriginalScore = results[i].Score
				}

				matchBonus := estimatedOriginalScore - baseScore - popularityBonus

				// Use the actual match type from the search results
				matchType := results[i].MatchType

				output.GetDebug().State("Font '%s': Position=%d/%d, Base=%d, Match=%s(+%d), Popularity=%d(+%d), Final=%d",
					results[i].Name, i+1, len(results), baseScore, matchType, matchBonus, results[i].Popularity, popularityBonus, results[i].Score)
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
	searchCmd.Flags().StringP("category", "c", "", "Filter by font category (use without value to see all available categories)")
	searchCmd.Flags().Lookup("category").NoOptDefVal = "list"

	// Hidden flag for development/testing only
	searchCmd.Flags().Bool("refresh", false, "Force refresh of font manifest before search")
	searchCmd.Flags().MarkHidden("refresh")

	// Register completion for both short and long category flags
	searchCmd.RegisterFlagCompletionFunc("category", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Get repository to extract categories
		r, err := repo.GetRepository()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		// Get all unique categories from sources
		categories := r.GetAllCategories()
		return categories, cobra.ShellCompDirectiveNoFileComp
	})
	searchCmd.RegisterFlagCompletionFunc("c", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Get repository to extract categories
		r, err := repo.GetRepository()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		// Get all unique categories from sources
		categories := r.GetAllCategories()
		return categories, cobra.ShellCompDirectiveNoFileComp
	})
}

// showAllCategories displays all available categories from all sources
func showAllCategories() error {
	// Ensure manifest system is initialized
	if err := config.EnsureManifestExists(); err != nil {
		output.GetVerbose().Error("%v", err)
		output.GetDebug().Error("config.EnsureManifestExists() failed: %v", err)
		return fmt.Errorf("unable to load font repository: %v", err)
	}

	// Get repository
	r, err := repo.GetRepository()
	if err != nil {
		output.GetVerbose().Error("%v", err)
		output.GetDebug().Error("repo.GetRepository() failed: %v", err)
		return fmt.Errorf("unable to load font repository: %v", err)
	}

	// Get all categories
	categories := r.GetAllCategories()

	if len(categories) == 0 {
		fmt.Printf("\n%s\n\n", ui.RenderError("No categories found in any sources"))
		return nil
	}

	// Display categories
	fmt.Printf("\n%s\n", ui.PageTitle.Render("Available Categories"))
	fmt.Printf("\nFound %d categories across all sources:\n\n", len(categories))

	// Display categories in a proper 3-column table format using consistent table styling
	// Calculate column width: (120 total - 2 spaces between columns) / 3 columns = 39 chars per column
	columnWidth := 39

	// Display in 3 columns with proper table alignment
	for i := 0; i < len(categories); i += 3 {
		// Print up to 3 categories per row
		for j := 0; j < 3 && i+j < len(categories); j++ {
			category := categories[i+j]
			if j == 0 {
				fmt.Printf("  %-*s", columnWidth, ui.TableSourceName.Render(category))
			} else {
				fmt.Printf("  %-*s", columnWidth, ui.TableSourceName.Render(category))
			}
		}
		fmt.Println() // New line after each row
	}

	// Show usage example
	fmt.Printf("\n%s\n", ui.FeedbackText.Render("Usage: fontget search -c \"Category Name\""))
	fmt.Printf("Example: fontget search -c \"Sans Serif\"\n\n")

	return nil
}
