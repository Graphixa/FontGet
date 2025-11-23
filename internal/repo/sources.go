package repo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"fontget/internal/config"
	"fontget/internal/ui"
)

// usePopularityScoring controls whether popularity is used in search scoring and sorting
// This gets set from user config, defaults to true (popularity on)
var usePopularityScoring = true

// SetPopularityScoring sets the popularity scoring preference from user config
func SetPopularityScoring() {
	userPrefs := config.GetUserPreferences()
	usePopularityScoring = userPrefs.Configuration.UsePopularitySort
}

// SearchConfig contains all tunable parameters for the search algorithm
type SearchConfig struct {
	BaseScore    int
	MatchBonuses struct {
		ExactMatch      int
		PrefixMatch     int
		ContainsMatch   int
		IDPrefixMatch   int
		IDContainsMatch int
	}
	LengthThreshold         float64
	PopularityDivisor       int
	PopularityEffectiveness string
}

// DefaultSearchConfig returns the default configuration for search scoring
func DefaultSearchConfig() SearchConfig {
	return SearchConfig{
		BaseScore: 50,
		MatchBonuses: struct {
			ExactMatch      int
			PrefixMatch     int
			ContainsMatch   int
			IDPrefixMatch   int
			IDContainsMatch int
		}{
			ExactMatch:      100, // Increased from 50 to 100 for better granularity
			PrefixMatch:     80,  // Increased from 45 to 80
			ContainsMatch:   60,  // Increased from 40 to 60
			IDPrefixMatch:   40,  // Increased from 25 to 40
			IDContainsMatch: 25,  // Increased from 15 to 25
		},
		LengthThreshold:         0.0,    // 0% = no threshold, can tune to 0.25 (25%) later
		PopularityDivisor:       2,      // Reduced from 4 to 2 for strong influence (0-50 points)
		PopularityEffectiveness: "sqrt", // "linear", "sqrt", or "quadratic"
	}
}

// Source priority order for consistent sorting across all commands
// Lower numbers = higher priority
var sourcePriority = map[string]int{
	"Google Fonts":  1,
	"Nerd Fonts":    2,
	"Font Squirrel": 3,
	// Custom sources will have priority 999 (default)
}

// getSourcePriority returns the priority for a given source name
// Lower numbers = higher priority (Google Fonts = 1, custom sources = 999)
func getSourcePriority(sourceName string) int {
	if priority, exists := sourcePriority[sourceName]; exists {
		return priority
	}
	return 999 // Custom sources get lowest priority
}

const (
	// Directory structure
	updateInterval = 24 * time.Hour

	// FontGet-Sources URLs
	googleFontsURL  = "https://raw.githubusercontent.com/Graphixa/FontGet-Sources/main/sources/google-fonts.json"
	nerdFontsURL    = "https://raw.githubusercontent.com/Graphixa/FontGet-Sources/main/sources/nerd-fonts.json"
	fontSquirrelURL = "https://raw.githubusercontent.com/Graphixa/FontGet-Sources/main/sources/font-squirrel.json"
)

// SourceURLs maps source names to their FontGet-Sources URLs
var SourceURLs = map[string]string{
	"google":   googleFontsURL,
	"nerd":     nerdFontsURL,
	"squirrel": fontSquirrelURL,
}

// GoogleFontsResponse represents the response from Google Fonts API
type GoogleFontsResponse struct {
	Kind  string `json:"kind"`
	Items []struct {
		Family       string            `json:"family"`
		Variants     []string          `json:"variants"`
		Subsets      []string          `json:"subsets"`
		Version      string            `json:"version"`
		LastModified string            `json:"lastModified"`
		Files        map[string]string `json:"files"`
		Category     string            `json:"category"`
		Kind         string            `json:"kind"`
	} `json:"items"`
}

// getFontGetDir returns the path to the user's ~/.fontget directory
func getFontGetDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(home, ".fontget"), nil
}

// sanitizeSourceNameForFilename converts a source name to a safe filename
// by converting to lowercase and replacing spaces with underscores
func sanitizeSourceNameForFilename(sourceName string) string {
	// Convert to lowercase
	sanitized := strings.ToLower(sourceName)
	// Replace spaces with underscores
	sanitized = strings.ReplaceAll(sanitized, " ", "_")
	return sanitized
}

func loadSourceDataWithCache(url string, sourceName string, progress ProgressCallback, forceRefresh bool) (*SourceData, error) {
	if progress != nil {
		progress(0, 1, fmt.Sprintf("Loading %s source...", sourceName))
	}

	// Get cache directory
	cacheDir, err := getFontGetDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache directory: %w", err)
	}
	sourcesCacheDir := filepath.Join(cacheDir, "sources")
	if err := os.MkdirAll(sourcesCacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create sources cache directory: %w", err)
	}

	// Create cache filename based on source name
	cacheFile := filepath.Join(sourcesCacheDir, fmt.Sprintf("%s.json", sanitizeSourceNameForFilename(sourceName)))

	// Check if we should use cached data
	useCache := false
	if !forceRefresh {
		if info, err := os.Stat(cacheFile); err == nil {
			// Check if cache is less than 24 hours old
			if time.Since(info.ModTime()) < updateInterval {
				useCache = true
			}
		}
	}

	// Try to load from cache first
	if useCache {
		if progress != nil {
			progress(0, 1, fmt.Sprintf("Loading %s from cache...", sourceName))
		}

		if data, err := os.ReadFile(cacheFile); err == nil {
			var sourceData SourceData
			if err := json.Unmarshal(data, &sourceData); err == nil {
				if progress != nil {
					progress(1, 1, fmt.Sprintf("Loaded %s from cache (%d fonts)", sourceName, sourceData.SourceInfo.TotalFonts))
				}
				return &sourceData, nil
			}
		}
	}

	// Download fresh data
	if progress != nil {
		progress(0, 1, fmt.Sprintf("Downloading %s source...", sourceName))
	}

	// Create HTTP client with timeout to prevent hanging
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download %s source: %w", sourceName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download %s source: HTTP %d", sourceName, resp.StatusCode)
	}

	var sourceData SourceData
	if err := json.NewDecoder(resp.Body).Decode(&sourceData); err != nil {
		return nil, fmt.Errorf("error parsing %s source: %w", sourceName, err)
	}

	// Save to cache
	if data, err := json.MarshalIndent(sourceData, "", "  "); err == nil {
		os.WriteFile(cacheFile, data, 0644)
	}

	if progress != nil {
		progress(1, 1, fmt.Sprintf("Downloaded and cached %s source (%d fonts)", sourceName, sourceData.SourceInfo.TotalFonts))
	}

	return &sourceData, nil
}

// loadSourceDataFromCacheOnly loads source data from cache only (no refresh)
func loadSourceDataFromCacheOnly(_ string, sourceName string) (*SourceData, error) {
	// Get cache directory
	cacheDir, err := getFontGetDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache directory: %w", err)
	}
	sourcesCacheDir := filepath.Join(cacheDir, "sources")

	// Create cache filename based on source name
	cacheFile := filepath.Join(sourcesCacheDir, fmt.Sprintf("%s.json", sanitizeSourceNameForFilename(sourceName)))

	// Check if cache file exists
	if _, err := os.Stat(cacheFile); err != nil {
		return nil, fmt.Errorf("cache file not found: %w", err)
	}

	// Read from cache
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	var sourceData SourceData
	if err := json.Unmarshal(data, &sourceData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache data: %w", err)
	}

	return &sourceData, nil
}

// loadAllSourcesFromCacheOnly loads all enabled sources from cache only (no refresh)
func loadAllSourcesFromCacheOnly(manifest *config.Manifest) (*FontManifest, error) {
	var allSources = make(map[string]SourceInfo)

	totalSources := 0
	enabledSources := 0

	// Count enabled sources
	for _, source := range manifest.Sources {
		if source.Enabled {
			totalSources++
		}
	}

	if totalSources == 0 {
		return nil, fmt.Errorf("no sources are enabled")
	}

	var oldestSourceTime time.Time
	firstSource := true

	for sourceName, sourceConfig := range manifest.Sources {
		if !sourceConfig.Enabled {
			continue
		}

		// Use the URL from the configuration
		sourceURL := sourceConfig.URL

		sourceData, err := loadSourceDataFromCacheOnly(sourceURL, sourceName)
		if err != nil {
			// If cache is not available, skip this source
			continue
		}

		// Add source info with fonts
		sourceInfo := sourceData.SourceInfo
		sourceInfo.Fonts = make(map[string]FontInfo)

		// Add fonts with source prefix, converting FontData to FontInfo
		for fontID, font := range sourceData.Fonts {
			prefixedID := fmt.Sprintf("%s.%s", sourceConfig.Prefix, fontID)

			// Convert Font to FontInfo for compatibility
			fontInfo := FontInfo{
				Name:        font.Name,
				License:     font.License,
				LicenseURL:  font.LicenseURL,
				Version:     font.Version,
				Description: font.Description,
				Categories:  font.Categories,
				Tags:        font.Tags,
				Popularity:  font.Popularity,
				MetadataURL: font.MetadataURL,
				SourceURL:   font.SourceURL,
			}

			// Convert variants to legacy format and preserve variant-file mapping
			fontInfo.VariantFiles = make(map[string]map[string]string)
			for _, variant := range font.Variants {
				fontInfo.Variants = append(fontInfo.Variants, variant.Name)

				// Store variant-specific files
				fontInfo.VariantFiles[variant.Name] = make(map[string]string)
				for fileType, url := range variant.Files {
					fontInfo.VariantFiles[variant.Name][fileType] = url
				}

				// Merge variant files into main files for backward compatibility
				if fontInfo.Files == nil {
					fontInfo.Files = make(map[string]string)
				}
				for fileType, url := range variant.Files {
					fontInfo.Files[fileType] = url
				}
			}

			sourceInfo.Fonts[prefixedID] = fontInfo
		}

		allSources[sourceName] = sourceInfo
		enabledSources++

		// Track the oldest source LastUpdated timestamp
		sourceTime := sourceData.SourceInfo.LastUpdated
		if firstSource || sourceTime.Before(oldestSourceTime) {
			oldestSourceTime = sourceTime
			firstSource = false
		}
	}

	if enabledSources == 0 {
		return nil, fmt.Errorf("no cached sources available")
	}

	// Use the oldest source LastUpdated timestamp, or current time if no sources found
	lastUpdated := oldestSourceTime
	if firstSource {
		lastUpdated = time.Now()
	}

	// Create the font manifest
	fontManifest := &FontManifest{
		Version:     "1.0",
		LastUpdated: lastUpdated,
		Sources:     allSources,
	}

	return fontManifest, nil
}

// loadAllSourcesWithCache loads all enabled sources with optional cache refresh
func loadAllSourcesWithCache(manifest *config.Manifest, progress ProgressCallback, forceRefresh bool) (*FontManifest, error) {
	var allSources = make(map[string]SourceInfo)

	totalSources := 0
	enabledSources := 0

	// Count enabled sources
	for _, source := range manifest.Sources {
		if source.Enabled {
			totalSources++
		}
	}

	if totalSources == 0 {
		return nil, fmt.Errorf("no sources are enabled")
	}

	currentSource := 0
	for sourceName, sourceConfig := range manifest.Sources {
		if !sourceConfig.Enabled {
			continue
		}

		currentSource++
		if progress != nil {
			progress(currentSource-1, totalSources, fmt.Sprintf("Loading %s...", sourceName))
		}

		// Use the URL from the configuration
		sourceURL := sourceConfig.URL

		sourceData, err := loadSourceDataWithCache(sourceURL, sourceName, nil, forceRefresh)
		if err != nil {
			// Log the error but continue with other sources
			if progress != nil {
				progress(currentSource-1, totalSources, fmt.Sprintf("Failed to load %s: %v", sourceName, err))
			}
			continue // Skip this source and continue with others
		}

		// Add source info with fonts
		sourceInfo := sourceData.SourceInfo
		sourceInfo.Fonts = make(map[string]FontInfo)

		// Add fonts with source prefix, converting FontData to FontInfo
		for fontID, font := range sourceData.Fonts {
			prefixedID := fmt.Sprintf("%s.%s", sourceConfig.Prefix, fontID)

			// Convert Font to FontInfo for compatibility
			fontInfo := FontInfo{
				Name:        font.Name,
				License:     font.License,
				LicenseURL:  font.LicenseURL,
				Version:     font.Version,
				Description: font.Description,
				Categories:  font.Categories,
				Tags:        font.Tags,
				Popularity:  font.Popularity,
				MetadataURL: font.MetadataURL,
				SourceURL:   font.SourceURL,
			}

			// Convert variants to legacy format and preserve variant-file mapping
			fontInfo.VariantFiles = make(map[string]map[string]string)
			for _, variant := range font.Variants {
				fontInfo.Variants = append(fontInfo.Variants, variant.Name)

				// Store variant-specific files
				fontInfo.VariantFiles[variant.Name] = make(map[string]string)
				for fileType, url := range variant.Files {
					fontInfo.VariantFiles[variant.Name][fileType] = url
				}

				// Merge variant files into main files for backward compatibility
				if fontInfo.Files == nil {
					fontInfo.Files = make(map[string]string)
				}
				for fileType, url := range variant.Files {
					fontInfo.Files[fileType] = url
				}
			}

			sourceInfo.Fonts[prefixedID] = fontInfo
		}

		allSources[sourceName] = sourceInfo
		enabledSources++
	}

	if progress != nil {
		progress(totalSources, totalSources, fmt.Sprintf("Loaded %d sources", enabledSources))
	}

	// Check if we loaded any sources successfully
	if len(allSources) == 0 {
		return nil, fmt.Errorf("no sources could be loaded successfully")
	}

	// Create combined font manifest
	fontManifest := &FontManifest{
		Version:     "2.0", // New version for FontGet-Sources integration
		LastUpdated: time.Now(),
		Sources:     allSources,
	}

	return fontManifest, nil
}

// ProgressCallback is a function type for reporting progress
type ProgressCallback func(current, total int, message string)

// GetFontInfo retrieves information about a font from the repository
func GetFontInfo(fontID string) (*FontInfo, error) {
	manifest, err := GetManifest(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}

	// Look for font in all sources
	for _, source := range manifest.Sources {
		if font, ok := source.Fonts[fontID]; ok {
			return &font, nil
		}
	}

	return nil, fmt.Errorf("font not found: %s", fontID)
}

// GetFontFiles retrieves the font files for a given font family
func GetFontFiles(fontFamily string) (map[string]string, error) {
	manifest, err := GetManifest(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}

	// Look for font in all sources
	for _, source := range manifest.Sources {
		if font, ok := source.Fonts[fontFamily]; ok {
			return font.Files, nil
		}
	}

	return nil, fmt.Errorf("font not found: %s", fontFamily)
}

// Repository represents a font repository
type Repository struct {
	manifest *FontManifest
}

// GetRepository returns a new Repository instance, showing spinner if sources need updating
func GetRepository() (*Repository, error) {
	// Set popularity scoring preference from user config
	SetPopularityScoring()

	// Check if sources should be refreshed
	shouldRefresh, err := config.ShouldRefreshSources()
	if err != nil {
		// If we can't determine, try cached first
		manifest, err := GetCachedManifest()
		if err != nil {
			// If no cache available, load with refresh
			manifest, err = GetManifest(nil, nil)
			if err != nil {
				return nil, err
			}

			// Update the timestamp
			config.UpdateSourcesLastUpdated()
		}
		return &Repository{manifest: manifest}, nil
	}

	if shouldRefresh {
		// Sources are stale, run spinner while refreshing and building
		var manifest *FontManifest
		err := ui.RunSpinner("Updating Sources...", "Sources Updated", func() error {
			m, e := GetManifest(nil, nil)
			if e != nil {
				return e
			}
			manifest = m
			return nil
		})
		if err != nil {
			return nil, err
		}

		// Update the timestamp
		config.UpdateSourcesLastUpdated()
		return &Repository{manifest: manifest}, nil
	}

	// Sources are fresh, use cached data
	manifest, err := GetCachedManifest()
	if err != nil {
		// If no cache available, load with refresh (no spinner for this case)
		manifest, err = GetManifest(nil, nil)
		if err != nil {
			return nil, err
		}
		// Update the timestamp
		config.UpdateSourcesLastUpdated()
	}
	return &Repository{manifest: manifest}, nil
}

// GetRepositoryWithRefresh forces a refresh of sources and returns a new Repository instance
func GetRepositoryWithRefresh() (*Repository, error) {
	// Set popularity scoring preference from user config
	SetPopularityScoring()

	// Force refresh of sources with spinner
	var manifest *FontManifest
	err := ui.RunSpinner("Updating Sources...", "Sources Updated", func() error {
		m, e := GetManifest(nil, nil)
		if e != nil {
			return e
		}
		manifest = m
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Update the timestamp
	config.UpdateSourcesLastUpdated()
	return &Repository{manifest: manifest}, nil
}

// GetManifest returns the current manifest
func (r *Repository) GetManifest() (*FontManifest, error) {
	return r.manifest, nil
}

// SearchFonts searches for fonts matching the query using advanced search logic
func (r *Repository) SearchFonts(query string, category string) ([]SearchResult, error) {
	// Load popularity preference from config each time
	SetPopularityScoring()
	return r.SearchFontsWithOptions(query, category, usePopularityScoring)
}

// SearchFontsWithOptions searches for fonts matching the query using advanced search logic
// with optional popularity scoring
func (r *Repository) SearchFontsWithOptions(query string, category string, usePopularity bool) ([]SearchResult, error) {
	// Update global popularity setting from config
	SetPopularityScoring()

	// Early return: if query is empty and no category, return no results
	// This prevents empty queries from matching all fonts
	if query == "" && category == "" {
		return []SearchResult{}, nil
	}

	var results []SearchResult

	// If query is empty but category is provided, return all fonts (will be filtered by category later)
	if query == "" {
		// Return all fonts from all sources
		for sourceID, source := range r.manifest.Sources {
			for id, font := range source.Fonts {
				result := r.createSearchResult(id, font, sourceID, source.Name)
				result.Score = 50 // Base score for category-only searches
				result.MatchType = "category-only"
				results = append(results, result)
			}
		}
	} else {
		// Normal search with query
		query = strings.ToLower(query)
		// Search through each source in the repository's manifest using advanced scoring
		for sourceID, source := range r.manifest.Sources {
			for id, font := range source.Fonts {
				// Check both the font name and ID
				fontName := strings.ToLower(font.Name)
				fontID := strings.ToLower(id)

				// Use the advanced scoring algorithm (popularity controlled by global variable)
				score, matchType := r.calculateMatchScoreWithOptions(query, fontName, fontID, font)
				if score > 0 {
					result := r.createSearchResult(id, font, sourceID, source.Name)
					result.Score = score
					result.MatchType = matchType
					results = append(results, result)
				}
			}
		}
		// Sort by score (highest first) only when we have a query
		r.sortResultsByScore(results)
	}

	// Filter by category if specified
	if category != "" {
		results = r.filterByCategory(results, category)
		// Sort by name for category-only searches (after filtering)
		if query == "" {
			sort.Slice(results, func(i, j int) bool {
				return results[i].Name < results[j].Name
			})
		}
	}

	return results, nil
}

// calculateMatchScoreWithOptions calculates a score for how well a font matches the query
// using the new configurable algorithm with base score, source priority, and match bonuses
// Returns both the score and the match type for debugging
func (r *Repository) calculateMatchScoreWithOptions(query, fontName, fontID string, font FontInfo) (int, string) {
	config := DefaultSearchConfig()

	// Phase 1: Base Score only (source priority applied in sorting logic)
	score := config.BaseScore
	matchType := "no-match"

	// Phase 2: Match Quality Bonuses
	if fontName == query {
		score += config.MatchBonuses.ExactMatch
		matchType = "exact"
	} else if strings.HasPrefix(fontName, query) {
		score += config.MatchBonuses.PrefixMatch
		matchType = "prefix"
	} else if strings.Contains(fontName, query) {
		// Only apply contains match if query is substantial (3+ chars)
		if len(query) >= 3 {
			score += config.MatchBonuses.ContainsMatch
			matchType = "contains"
		}
	} else if strings.HasPrefix(fontID, query) {
		score += config.MatchBonuses.IDPrefixMatch
		matchType = "id-prefix"
	} else if strings.Contains(fontID, query) {
		// Only apply ID contains match if query is substantial (4+ chars)
		if len(query) >= 4 {
			score += config.MatchBonuses.IDContainsMatch
			matchType = "id-contains"
		}
	}

	// If no match found, return 0
	if score == config.BaseScore {
		return 0, "no-match"
	}

	// Phase 3: Popularity - only if enabled and font has popularity
	if usePopularityScoring && font.Popularity > 0 {
		// Apply full popularity bonus (no length adjustment here)
		popularityBonus := float64(font.Popularity) / float64(config.PopularityDivisor)
		score += int(popularityBonus)
	}

	return score, matchType
}

// createSearchResult creates a SearchResult from FontInfo
func (r *Repository) createSearchResult(id string, font FontInfo, sourceID, sourceName string) SearchResult {
	return SearchResult{
		Name:       font.Name,
		ID:         id,
		Source:     sourceID,
		SourceName: sourceName,
		License:    font.License,
		Categories: font.Categories,
		Popularity: font.Popularity,
	}
}

// sortResultsByScore sorts results by: Font Name Groups → Source Priority → Individual Score → Popularity
// Exact name matches are grouped together and sorted by source priority within each group
func (r *Repository) sortResultsByScore(results []SearchResult) {
	// Group fonts by exact name and find the highest score in each group
	fontGroups := make(map[string]int) // font name -> highest score in group
	for _, result := range results {
		if result.Score > fontGroups[result.Name] {
			fontGroups[result.Name] = result.Score
		}
	}

	sort.Slice(results, func(i, j int) bool {
		// 1. PRIMARY: Group by highest score in font name group
		scoreI := fontGroups[results[i].Name]
		scoreJ := fontGroups[results[j].Name]
		if scoreI != scoreJ {
			return scoreI > scoreJ
		}

		// 2. SECONDARY: Font Name (alphabetically) - for consistent ordering within groups
		if results[i].Name != results[j].Name {
			return results[i].Name < results[j].Name
		}

		// 3. TERTIARY: Source Priority (Google → Nerd → Squirrel → Custom) - only within exact name groups
		priorityI := getSourcePriority(results[i].SourceName)
		priorityJ := getSourcePriority(results[j].SourceName)
		if priorityI != priorityJ {
			return priorityI < priorityJ
		}

		// 4. QUATERNARY: Individual font score (highest first) - within same source priority
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}

		// 5. FINAL: Popularity (highest first) - ultimate tiebreaker (only if enabled)
		if usePopularityScoring {
			if results[i].Popularity != results[j].Popularity {
				return results[i].Popularity > results[j].Popularity
			}
		}

		// 6. ULTIMATE: Font ID (alphabetically) - final tiebreaker
		return results[i].ID < results[j].ID
	})
}

// filterByCategory filters results by category
func (r *Repository) filterByCategory(results []SearchResult, category string) []SearchResult {
	var filteredResults []SearchResult
	for _, result := range results {
		found := false
		for _, cat := range result.Categories {
			if strings.EqualFold(cat, category) {
				found = true
				break
			}
		}
		if found {
			filteredResults = append(filteredResults, result)
		}
	}
	return filteredResults
}

// GetAllCategories returns all unique categories from all sources in the manifest
func (r *Repository) GetAllCategories() []string {
	categorySet := make(map[string]bool)

	for _, source := range r.manifest.Sources {
		for _, font := range source.Fonts {
			for _, category := range font.Categories {
				if category != "" {
					categorySet[category] = true
				}
			}
		}
	}

	var categories []string
	for cat := range categorySet {
		categories = append(categories, cat)
	}

	sort.Strings(categories)
	return categories
}
