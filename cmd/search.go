package cmd

import (
	"fmt"
	"math"
	"strings"

	"fontget/internal/cmdutils"
	"fontget/internal/components"
	"fontget/internal/config"
	"fontget/internal/output"
	"fontget/internal/repo"
	"fontget/internal/shared"
	"fontget/internal/ui"

	"github.com/spf13/cobra"
)

// Placeholder constants
const (
	SearchPlaceholderNA = "N/A"
)

// Search scoring constants
const (
	SearchBaseScore             = 50
	SearchMaxWidth              = 160
	SearchTableColumnSpacing    = 4   // Spaces between table columns
	SearchPopularityDivisor     = 2   // Divisor for popularity bonus calculation
	SearchLengthRatioThreshold  = 0.5 // Threshold for length penalty application
	SearchCategoryColumnWidth   = 39  // Column width for category display (3 columns)
	SearchCategoryColumnsPerRow = 3   // Number of categories per row
)

// validateSource checks if a source exists (by ID, name, or prefix) in the repository.
// Returns true if the source exists, false otherwise.
// Checks against:
// - Source ID (manifest key, e.g., "Google Fonts")
// - Source Name (full name, e.g., "Google Fonts")
// - Source Prefix (short ID, e.g., "google")
func validateSource(source string) bool {
	if source == "" {
		return true // Empty source is valid (no filter)
	}

	r, err := repo.GetRepository()
	if err != nil {
		return false
	}

	manifest, err := r.GetManifest()
	if err != nil {
		return false
	}

	// Also check config manifest for Prefix field
	configManifest, err := config.LoadManifest()
	if err == nil {
		for sourceID, sourceInfo := range manifest.Sources {
			// Check source ID (manifest key, full name like "Google Fonts")
			if strings.EqualFold(sourceID, source) {
				return true
			}
			// Check source name (full name like "Google Fonts")
			if strings.EqualFold(sourceInfo.Name, source) {
				return true
			}
			// Check source prefix (short ID like "google", "nerd", "squirrel")
			if configSource, exists := configManifest.Sources[sourceID]; exists {
				if strings.EqualFold(configSource.Prefix, source) {
					return true
				}
			}
		}
	} else {
		// Fallback: check without config manifest (no prefix matching)
		for sourceID, sourceInfo := range manifest.Sources {
			if strings.EqualFold(sourceID, source) || strings.EqualFold(sourceInfo.Name, source) {
				return true
			}
		}
	}

	return false
}

// logDebugScoreBreakdown logs detailed score breakdown for a search result in debug mode.
func logDebugScoreBreakdown(result repo.SearchResult, query string, index, totalResults int, baseScore int, usePopularity bool) {
	var popularityBonus int
	if usePopularity {
		popularityBonus = result.Popularity / SearchPopularityDivisor
	}

	// Calculate match bonus by reverse-engineering from final score
	// Account for length penalty by estimating the original score before penalty
	queryLength := len(query)
	fontLength := len(result.Name)
	lengthRatio := float64(queryLength) / float64(fontLength)

	// Estimate original score before length penalty
	var estimatedOriginalScore int
	if lengthRatio < SearchLengthRatioThreshold {
		// Length penalty was applied, estimate original score
		lengthPenaltyEffectiveness := math.Sqrt(lengthRatio)
		estimatedOriginalScore = int(float64(result.Score) / lengthPenaltyEffectiveness)
	} else {
		// No length penalty applied
		estimatedOriginalScore = result.Score
	}

	matchBonus := estimatedOriginalScore - baseScore - popularityBonus
	matchType := result.MatchType

	output.GetDebug().State("Font '%s': Position=%d/%d, Base=%d, Match=%s(+%d), Popularity=%d(+%d), Final=%d",
		result.Name, index+1, totalResults, baseScore, matchType, matchBonus, result.Popularity, popularityBonus, result.Score)
}

var searchCmd = &cobra.Command{
	Use:          "search <query>",
	Short:        "Search for available fonts",
	SilenceUsage: true,
	Long: `Search for fonts from all configured sources.

Use --category to filter by category (e.g., "Sans Serif", "Serif", "Monospace").
Use --source to filter by source (short ID like "google", "nerd", or full name like "Google Fonts").
Use -c without a value to list categories.
Use -s without a value to list sources.`,
	Example: `  fontget search fira
  fontget search "Fira Sans"
  fontget search -c "Sans Serif"
  fontget search "roboto" -c "Sans Serif"
  fontget search "fira" -s google
  fontget search "fira" -s "Google Fonts"
  fontget search -s google
  fontget search -c
  fontget search -s`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Get flags
		category, _ := cmd.Flags().GetString("category")
		source, _ := cmd.Flags().GetString("source")

		// Get query from args
		var query string
		if len(args) > 0 {
			query = args[0]
		}

		// Validate query - require either a query, category, or source
		if query == "" && category == "" && source == "" {
			fmt.Printf("\n%s\n", ui.RenderError("A search query, category, or source is required"))
			fmt.Printf("Use 'fontget search --help' for more information.\n\n")
			// Return nil since we've already printed the error message
			// This prevents Cobra from printing a duplicate error
			return nil
		}

		// Validate source if provided (check both source ID and source name)
		// Skip validation if source is "list" (NoOptDefVal for listing all sources)
		if source != "" && source != "list" && !validateSource(source) {
			fmt.Printf("\n%s\n", ui.RenderError(fmt.Sprintf("Source '%s' not found. Use 'fontget sources info' to see available sources.", source)))
			fmt.Printf("Use 'fontget search --help' for more information.\n\n")
			return nil
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

		// Debug-level information is logged via output.GetDebug() throughout the function

		// Ensure manifest system is initialized (fixes missing sources.json bug)
		if err := cmdutils.EnsureManifestInitialized(func() cmdutils.Logger { return GetLogger() }); err != nil {
			return err
		}

		// Get arguments (already validated by Args function)
		category, _ := cmd.Flags().GetString("category")
		source, _ := cmd.Flags().GetString("source")
		refresh, _ := cmd.Flags().GetBool("refresh")
		var query string
		if len(args) > 0 {
			query = args[0]
		}

		// Safety check: if query, category, and source are all empty, return early
		// This prevents execution when Args validation fails (defense in depth)
		if query == "" && category == "" && source == "" {
			return nil // Args already showed error and help
		}

		// Log search parameters (always log to file)
		GetLogger().Info("Search parameters - Query: %s, Category: %s, Source: %s, Refresh: %v", query, category, source, refresh)

		// Handle category-only mode (show all categories) - early return
		// Check if category flag was provided but no value was given (NoOptDefVal = "list")
		// AND no arguments were provided (meaning user just used -c without a value)
		if (cmd.Flags().Changed("category") || cmd.Flags().Changed("c")) && category == "list" && len(args) == 0 {
			// Show all available categories
			return showAllCategories()
		}

		// Handle source-only mode (show all sources) - early return
		// Check if source flag was provided but no value was given (NoOptDefVal = "list")
		// AND no arguments were provided (meaning user just used -s without a value)
		if (cmd.Flags().Changed("source") || cmd.Flags().Changed("s")) && source == "list" && len(args) == 0 {
			// Show all available sources
			return showAllSources()
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

		// If source is "list" but we have arguments, it means the user provided a source value
		// but NoOptDefVal is overriding it. We need to extract the actual source from args
		if source == "list" && len(args) > 0 {
			// The first argument is actually the source value
			source = args[0]
			// Remove the first argument from args since it's the source, not a query
			if len(args) > 1 {
				query = args[1]
			} else {
				query = ""
			}
		}

		// Verbose-level information for users - show operational details
		output.GetVerbose().Info("Search parameters - Query: %s, Category: %s, Source: %s, Refresh: %v", query, category, source, refresh)
		// Verbose section ends with blank line per spacing framework (only if verbose was shown)
		if IsVerbose() {
			fmt.Println()
		}
		output.GetDebug().State("Starting font search with parameters: query='%s', category='%s', source='%s', refresh=%v", query, category, source, refresh)

		// Get repository with optional refresh
		r, err := cmdutils.GetRepository(refresh, GetLogger())
		if err != nil {
			return err
		}

		// Handle source-only search (no query, no category)
		// SearchFonts("", "") returns empty, so we need to get all fonts directly
		var results []repo.SearchResult
		if query == "" && category == "" && source != "" {
			// Source-only search: get all fonts from the specified source
			manifest, err := r.GetManifest()
			if err != nil {
				GetLogger().Error("Failed to get manifest: %v", err)
				output.GetVerbose().Error("%v", err)
				output.GetDebug().Error("r.GetManifest() failed: %v", err)
				return fmt.Errorf("unable to get font manifest: %v", err)
			}

			// Get config manifest to resolve source prefix to source name
			configManifest, err := config.LoadManifest()
			if err != nil {
				GetLogger().Error("Failed to load config manifest: %v", err)
				output.GetVerbose().Error("%v", err)
				return fmt.Errorf("unable to load config manifest: %v", err)
			}

			// Find the source name(s) that match the provided source (could be prefix, ID, or name)
			matchingSourceNames := make(map[string]bool)
			for sourceID, sourceInfo := range manifest.Sources {
				// Check if source matches by ID (full name like "Google Fonts")
				if strings.EqualFold(sourceID, source) {
					matchingSourceNames[sourceID] = true
				}
				// Check if source matches by name (full name like "Google Fonts")
				if strings.EqualFold(sourceInfo.Name, source) {
					matchingSourceNames[sourceID] = true
				}
				// Check if source matches by prefix (short ID like "google")
				if configManifest != nil {
					if sourceConfig, exists := configManifest.Sources[sourceID]; exists {
						if strings.EqualFold(sourceConfig.Prefix, source) {
							matchingSourceNames[sourceID] = true
						}
					}
				}
			}

			// Get all fonts from matching sources
			for sourceID := range matchingSourceNames {
				if source, exists := manifest.Sources[sourceID]; exists {
					for id, font := range source.Fonts {
						result := repo.SearchResult{
							Name:       font.Name,
							ID:         id,
							Source:     sourceID,
							SourceName: source.Name,
							License:    font.License,
							Categories: font.Categories,
							Popularity: font.Popularity,
							Score:      50, // Base score for source-only searches
							MatchType:  "source-only",
						}
						results = append(results, result)
					}
				}
			}

			output.GetVerbose().Info("Found %d fonts from source: '%s'", len(results), source)
			output.GetDebug().State("Source-only search returned %d results", len(results))
		} else {
			// Normal search with query and/or category
			output.GetVerbose().Info("Searching fonts with query: '%s' and category: '%s'", query, category)
			output.GetDebug().State("Calling r.SearchFonts(query='%s', category='%s')", query, category)
			results, err = r.SearchFonts(query, category)
			if err != nil {
				GetLogger().Error("Failed to search fonts: %v", err)
				output.GetVerbose().Error("%v", err)
				output.GetDebug().Error("r.SearchFonts() failed: %v", err)
				return fmt.Errorf("unable to search fonts: %v", err)
			}
		}

		// Filter by source if specified (check source ID, name, and prefix)
		if source != "" {
			// Get config manifest to check prefixes
			configManifest, _ := config.LoadManifest()

			filteredResults := []repo.SearchResult{}
			for _, result := range results {
				matched := false

				// Match against Source (manifest key, full name like "Google Fonts")
				if strings.EqualFold(result.Source, source) {
					matched = true
				}
				// Match against SourceName (full name like "Google Fonts")
				if strings.EqualFold(result.SourceName, source) {
					matched = true
				}
				// Match against Prefix (short ID like "google", "nerd", "squirrel")
				if configManifest != nil {
					if sourceConfig, exists := configManifest.Sources[result.Source]; exists {
						if strings.EqualFold(sourceConfig.Prefix, source) {
							matched = true
						}
					}
				}

				if matched {
					filteredResults = append(filteredResults, result)
				}
			}
			results = filteredResults
			output.GetDebug().State("Filtered results by source '%s': %d results remaining", source, len(results))
		}

		// Apply search result limit from config (0 = unlimited)
		// Note: When Bubble Tea tables are implemented, this limit may be evaluated differently
		// for interactive browsing vs static output. Leave comments for future evaluation.
		userPrefs := config.GetUserPreferences()
		limit := userPrefs.Search.ResultLimit
		totalResults := len(results)
		if limit > 0 && len(results) > limit {
			results = results[:limit]
		}

		output.GetVerbose().Info("Search completed - found %d results", len(results))
		// Verbose section ends with blank line per spacing framework (only if verbose was shown)
		if IsVerbose() {
			fmt.Println()
		}
		output.GetDebug().State("Search returned %d font results", len(results))

		// Build the search result message
		var searchMsg string
		if query != "" {
			if limit > 0 && totalResults > limit {
				searchMsg = fmt.Sprintf("Found %d fonts matching: '%s' (showing %d)", totalResults, ui.QueryText.Render(query), len(results))
			} else {
				searchMsg = fmt.Sprintf("Found %d fonts matching: '%s'", len(results), ui.QueryText.Render(query))
			}
			if category != "" {
				searchMsg += fmt.Sprintf(" | Filtered by category: '%s'", ui.QueryText.Render(category))
			}
			if source != "" {
				searchMsg += fmt.Sprintf(" | Filtered by source: '%s'", ui.QueryText.Render(source))
			}
		} else if category != "" {
			// Category-only search
			if limit > 0 && totalResults > limit {
				searchMsg = fmt.Sprintf("Found %d fonts in category: '%s' (showing %d)", totalResults, ui.QueryText.Render(category), len(results))
			} else {
				searchMsg = fmt.Sprintf("Found %d fonts in category: '%s'", len(results), ui.QueryText.Render(category))
			}
			if source != "" {
				searchMsg += fmt.Sprintf(" | Filtered by source: '%s'", ui.QueryText.Render(source))
			}
		} else {
			// Source-only search
			if limit > 0 && totalResults > limit {
				searchMsg = fmt.Sprintf("Found %d fonts (showing %d)", totalResults, len(results))
			} else {
				searchMsg = fmt.Sprintf("Found %d fonts", len(results))
			}
		}
		fmt.Printf("\n%s\n\n", searchMsg)

		// Build table rows with priority: Font ID > Font Name > Source > Categories > License
		var tableRows [][]string
		for i, result := range results {
			// Format categories
			categoriesStr := SearchPlaceholderNA
			if len(result.Categories) > 0 {
				categoriesStr = strings.Join(result.Categories, ", ")
			}

			// Format license
			license := SearchPlaceholderNA
			if result.License != "" {
				license = result.License
			}

			// Build row: Font Name, Font ID, License, Categories, Source
			row := []string{
				result.Name,
				result.ID,
				license,
				categoriesStr,
				result.SourceName,
			}
			tableRows = append(tableRows, row)

			// Add detailed score breakdown for debugging (only when --debug flag is enabled)
			if IsDebug() {
				baseScore := SearchBaseScore
				userPrefs := config.GetUserPreferences()
				usePopularity := userPrefs.Search.EnablePopularitySort
				logDebugScoreBreakdown(results[i], query, i, len(results), baseScore, usePopularity)
			}
		}

		// Render table with priority configuration
		tableConfig := components.TableConfig{
			Columns: []components.ColumnConfig{
				{Header: "Font Name", MinWidth: 32, PercentWidth: 22.0},
				{Header: "Font ID", MinWidth: 39, PercentWidth: 24.0}, // Highest priority, don't trim
				{Header: "License", MinWidth: 3, PercentWidth: 8.0},   // Lowest priority
				{Header: "Categories", PercentWidth: 22.0},
				{Header: "Source", MaxWidth: 14, PercentWidth: 24.0},
			},
			Rows:  tableRows,
			Width: 0, // Auto-detect terminal width
			Mode:  components.TableModeStatic,
		}

		fmt.Println(components.RenderStaticTable(tableConfig))

		// Show when FontGet last updated sources
		if lastUpdated, err := config.GetSourcesLastUpdated(); err == nil && !lastUpdated.IsZero() {
			fmt.Printf("\n%s: %s\n", ui.Text.Render("Sources Last Updated"), lastUpdated.Format("Mon, 02 Jan 2006 15:04:05 MST"))
		}

		GetLogger().Info("Font search operation completed successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.Flags().StringP("category", "c", "", "Filter by font category (use without value to see all available categories)")
	searchCmd.Flags().Lookup("category").NoOptDefVal = "list"
	searchCmd.Flags().StringP("source", "s", "", "Filter by source (short ID like \"google\", \"nerd\", \"squirrel\" or full name like \"Google Fonts\")")
	searchCmd.Flags().Lookup("source").NoOptDefVal = "list"

	// Hidden flag for development/testing only
	searchCmd.Flags().Bool("refresh", false, "Force refresh of font manifest before search")
	searchCmd.Flags().MarkHidden("refresh")

	// Helper function for category completion (shared by both short and long flags)
	categoryCompletionFunc := func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		r, err := repo.GetRepository()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		categories := r.GetAllCategories()
		return categories, cobra.ShellCompDirectiveNoFileComp
	}

	// Register completion for both short and long category flags
	searchCmd.RegisterFlagCompletionFunc("category", categoryCompletionFunc)
	searchCmd.RegisterFlagCompletionFunc("c", categoryCompletionFunc)

	// Helper function for source completion (shared by both short and long flags)
	sourceCompletionFunc := func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		r, err := repo.GetRepository()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		manifest, err := r.GetManifest()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		// Get config manifest to access prefixes
		configManifest, err := config.LoadManifest()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		// Get all source prefixes (short IDs), IDs (full names), and names (full names)
		// Prioritize short IDs (prefixes) first
		var completions []string
		seen := make(map[string]bool)
		toCompleteLower := strings.ToLower(toComplete)

		// First add matching source prefixes (short IDs like "google", "nerd", "squirrel")
		for _, sourceConfig := range configManifest.Sources {
			if strings.HasPrefix(strings.ToLower(sourceConfig.Prefix), toCompleteLower) {
				if !seen[sourceConfig.Prefix] {
					completions = append(completions, sourceConfig.Prefix)
					seen[sourceConfig.Prefix] = true
				}
			}
		}

		// Then add matching source IDs (manifest keys, full names like "Google Fonts")
		for sourceID := range manifest.Sources {
			if strings.HasPrefix(strings.ToLower(sourceID), toCompleteLower) {
				if !seen[sourceID] {
					completions = append(completions, sourceID)
					seen[sourceID] = true
				}
			}
		}

		// Finally add matching source names (full names like "Google Fonts", "Nerd Fonts")
		for sourceID, sourceInfo := range manifest.Sources {
			// Only add if it's different from the ID and matches
			if strings.HasPrefix(strings.ToLower(sourceInfo.Name), toCompleteLower) {
				if !seen[sourceInfo.Name] && !strings.EqualFold(sourceID, sourceInfo.Name) {
					completions = append(completions, sourceInfo.Name)
					seen[sourceInfo.Name] = true
				}
			}
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	}

	// Register completion for both short and long source flags
	searchCmd.RegisterFlagCompletionFunc("source", sourceCompletionFunc)
	searchCmd.RegisterFlagCompletionFunc("s", sourceCompletionFunc)
}

// showAllCategories displays all available categories from all sources
func showAllCategories() error {
	// Ensure manifest system is initialized
	if err := cmdutils.EnsureManifestInitialized(func() cmdutils.Logger { return GetLogger() }); err != nil {
		return err
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

	// Start with a blank line for consistent spacing
	fmt.Println()
	fmt.Printf("Found %d categories across all sources:\n\n", len(categories))

	// Display categories in a proper 3-column table format using consistent table styling
	// Display in 3 columns with proper table alignment
	for i := 0; i < len(categories); i += SearchCategoryColumnsPerRow {
		// Print up to 3 categories per row
		for j := 0; j < SearchCategoryColumnsPerRow && i+j < len(categories); j++ {
			category := categories[i+j]
			fmt.Printf("  %-*s", SearchCategoryColumnWidth, ui.TableSourceName.Render(category))
		}
		fmt.Println() // New line after each row
	}

	// Show usage example
	fmt.Printf("\n%s\n", ui.Text.Render("Usage: fontget search -c \"Category Name\""))
	fmt.Printf("Example: fontget search -c \"Sans Serif\"\n\n")

	return nil
}

// showAllSources displays all available sources in a table format
func showAllSources() error {
	// Ensure manifest system is initialized
	if err := cmdutils.EnsureManifestInitialized(func() cmdutils.Logger { return GetLogger() }); err != nil {
		return err
	}

	// Get repository
	r, err := repo.GetRepository()
	if err != nil {
		output.GetVerbose().Error("%v", err)
		output.GetDebug().Error("repo.GetRepository() failed: %v", err)
		return fmt.Errorf("unable to load font repository: %v", err)
	}

	// Get font manifest to access source names
	manifest, err := r.GetManifest()
	if err != nil {
		output.GetVerbose().Error("%v", err)
		output.GetDebug().Error("r.GetManifest() failed: %v", err)
		return fmt.Errorf("unable to get font manifest: %v", err)
	}

	// Get config manifest to access source prefixes
	configManifest, err := config.LoadManifest()
	if err != nil {
		output.GetVerbose().Error("%v", err)
		output.GetDebug().Error("config.LoadManifest() failed: %v", err)
		return fmt.Errorf("unable to load config manifest: %v", err)
	}

	// Collect all sources with their prefixes and names
	type sourceInfo struct {
		prefix string
		name   string
	}
	var sources []sourceInfo
	seen := make(map[string]bool)

	// Iterate through all sources in the font manifest
	for sourceID, sourceData := range manifest.Sources {
		// Get prefix from config manifest
		prefix := ""
		if configSource, exists := configManifest.Sources[sourceID]; exists {
			prefix = configSource.Prefix
		}
		// If no prefix found, use sourceID as fallback
		if prefix == "" {
			prefix = sourceID
		}

		// Get source name (use Name from sourceData, fallback to sourceID)
		sourceName := sourceData.Name
		if sourceName == "" {
			sourceName = sourceID
		}

		// Only add if we haven't seen this source yet
		key := strings.ToLower(prefix + "|" + sourceName)
		if !seen[key] {
			sources = append(sources, sourceInfo{
				prefix: prefix,
				name:   sourceName,
			})
			seen[key] = true
		}
	}

	if len(sources) == 0 {
		fmt.Printf("\n%s\n\n", ui.RenderError("No sources found"))
		return nil
	}

	// Calculate dynamic column widths with minimums
	const (
		minPrefixWidth = 12
		minNameWidth   = 20
		columnSpacing  = 2
	)

	maxPrefixWidth := minPrefixWidth
	maxNameWidth := minNameWidth

	for _, source := range sources {
		if len(source.prefix) > maxPrefixWidth {
			maxPrefixWidth = len(source.prefix)
		}
		if len(source.name) > maxNameWidth {
			maxNameWidth = len(source.name)
		}
	}

	// Start with a blank line for consistent spacing
	fmt.Println()
	fmt.Printf("Found %d sources:\n\n", len(sources))

	// Print table header
	headerPrefix := "Prefix"
	headerName := "Source Name"
	fmt.Printf("  %-*s  %s\n", maxPrefixWidth, headerPrefix, headerName)
	fmt.Printf("  %s  %s\n", strings.Repeat("─", maxPrefixWidth), strings.Repeat("─", maxNameWidth))

	// Print source rows with truncation if needed
	for _, source := range sources {
		prefixDisplay := shared.TruncateString(source.prefix, maxPrefixWidth)
		nameDisplay := shared.TruncateString(source.name, maxNameWidth)
		fmt.Printf("  %-*s  %s\n", maxPrefixWidth, prefixDisplay, nameDisplay)
	}

	// Show usage example
	fmt.Printf("\n%s\n", ui.Text.Render("Usage: fontget search -s \"source\""))
	fmt.Printf("Example: fontget search -s google\n\n")

	return nil
}
