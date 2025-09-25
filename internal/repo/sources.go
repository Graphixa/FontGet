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

	"context"

	"github.com/charmbracelet/lipgloss"
	pinpkg "github.com/yarlson/pin"
)

const (
	// Directory structure
	sourcesDir     = "sources"
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

// getSourcesDir returns the path to the user's ~/.fontget/sources directory
func getSourcesDir() (string, error) {
	fontGetDir, err := getFontGetDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(fontGetDir, sourcesDir), nil
}

// loadSourceData loads a single source file from FontGet-Sources
func loadSourceData(url string, sourceName string, progress ProgressCallback) (*SourceData, error) {
	return loadSourceDataWithCache(url, sourceName, progress, false)
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
	cacheFile := filepath.Join(sourcesCacheDir, fmt.Sprintf("%s.json", strings.ToLower(sourceName)))

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

	resp, err := http.Get(url)
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
func loadSourceDataFromCacheOnly(url string, sourceName string) (*SourceData, error) {
	// Get cache directory
	cacheDir, err := getFontGetDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache directory: %w", err)
	}
	sourcesCacheDir := filepath.Join(cacheDir, "sources")

	// Create cache filename based on source name
	cacheFile := filepath.Join(sourcesCacheDir, fmt.Sprintf("%s.json", strings.ToLower(sourceName)))

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

// loadAllSources loads all enabled sources and merges them into a combined manifest
func loadAllSources(sourcesConfig *config.SourcesConfig, progress ProgressCallback) (*FontManifest, error) {
	return loadAllSourcesWithCache(sourcesConfig, progress, false)
}

// loadAllSourcesFromCacheOnly loads all enabled sources from cache only (no refresh)
func loadAllSourcesFromCacheOnly(sourcesConfig *config.SourcesConfig) (*FontManifest, error) {
	var allSources = make(map[string]SourceInfo)

	totalSources := 0
	enabledSources := 0

	// Count enabled sources
	for _, source := range sourcesConfig.Sources {
		if source.Enabled {
			totalSources++
		}
	}

	if totalSources == 0 {
		return nil, fmt.Errorf("no sources are enabled")
	}

	var oldestSourceTime time.Time
	firstSource := true

	for sourceName, sourceConfig := range sourcesConfig.Sources {
		if !sourceConfig.Enabled {
			continue
		}

		// Use the URL from the configuration
		sourceURL := sourceConfig.Path

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

	// Create the manifest
	manifest := &FontManifest{
		Version:     "1.0",
		LastUpdated: lastUpdated,
		Sources:     allSources,
	}

	return manifest, nil
}

// loadAllSourcesWithCache loads all enabled sources with optional cache refresh
func loadAllSourcesWithCache(sourcesConfig *config.SourcesConfig, progress ProgressCallback, forceRefresh bool) (*FontManifest, error) {
	var allSources = make(map[string]SourceInfo)

	totalSources := 0
	enabledSources := 0

	// Count enabled sources
	for _, source := range sourcesConfig.Sources {
		if source.Enabled {
			totalSources++
		}
	}

	if totalSources == 0 {
		return nil, fmt.Errorf("no sources are enabled")
	}

	currentSource := 0
	for sourceName, sourceConfig := range sourcesConfig.Sources {
		if !sourceConfig.Enabled {
			continue
		}

		currentSource++
		if progress != nil {
			progress(currentSource-1, totalSources, fmt.Sprintf("Loading %s...", sourceName))
		}

		// Use the URL from the configuration
		sourceURL := sourceConfig.Path

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

	// Create combined manifest
	manifest := &FontManifest{
		Version:     "2.0", // New version for FontGet-Sources integration
		LastUpdated: time.Now(),
		Sources:     allSources,
	}

	return manifest, nil
}

// ensureSourcesDir ensures the sources directory exists
func ensureSourcesDir() (string, error) {
	// First ensure .fontget directory exists
	fontGetDir, err := getFontGetDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(fontGetDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create .fontget directory: %w", err)
	}

	// Then ensure sources directory exists
	sourcesDir, err := getSourcesDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(sourcesDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create sources directory: %w", err)
	}

	return sourcesDir, nil
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

// runSpinner runs a lightweight spinner while fn executes. Always stops with a green check style.
func runSpinner(msg, doneMsg string, fn func() error) error {
	// Pre-style the message with lipgloss for any hex color
	styledMsg := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6adc8")). // Gray
		Render(msg)

	// Configure spinner with pre-styled message
	p := pinpkg.New(styledMsg,
		pinpkg.WithSpinnerColor(pinkToPin("#cba6f7")), // spinner mauve
		pinpkg.WithDoneSymbol('✓'),
		pinpkg.WithDoneSymbolColor(pinkToPin("#a6e3a1")), // green check
	)
	// Start spinner; it auto-disables animation when output is piped
	cancel := p.Start(context.Background())
	defer cancel()

	err := fn()
	if err != nil {
		// Show failure with red X, but return the error
		p.Fail("✗ " + err.Error())
		return err
	}
	// Style the completion message with the same color
	styledDoneMsg := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6adc8")). // Gray
		Render(doneMsg)

	if doneMsg == "" {
		styledDoneMsg = styledMsg
	}
	p.Stop(styledDoneMsg)
	return nil
}

// pinkToPin maps hex-ish choice to nearest pin color (simple mapping to keep code local)
func pinkToPin(hex string) pinpkg.Color {
	switch strings.ToLower(hex) {
	case "#a6e3a1":
		return pinpkg.ColorGreen
	case "#cba6f7":
		return pinpkg.ColorMagenta
	case "#b4befe":
		return pinpkg.ColorBlue
	case "#a6adc8":
		return pinpkg.ColorCyan
	default:
		return pinpkg.ColorDefault
	}
}

// GetRepository returns a new Repository instance, showing spinner if sources need updating
func GetRepository() (*Repository, error) {
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
		err := runSpinner("Updating Sources...", "Sources Updated", func() error {
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

// GetManifest returns the current manifest
func (r *Repository) GetManifest() (*FontManifest, error) {
	return r.manifest, nil
}

// SearchFonts searches for fonts matching the query using advanced search logic
func (r *Repository) SearchFonts(query string, category string) ([]SearchResult, error) {
	// Check cache first
	if cachedResults, err := r.getCachedSearchResults(query, category); err == nil && cachedResults != nil {
		return cachedResults, nil
	}

	query = strings.ToLower(query)
	var results []SearchResult

	// Search through each source in the repository's manifest using advanced scoring
	for sourceID, source := range r.manifest.Sources {
		for id, font := range source.Fonts {
			// Check both the font name and ID
			fontName := strings.ToLower(font.Name)
			fontID := strings.ToLower(id)

			// Use the advanced scoring algorithm
			score := r.calculateMatchScore(query, fontName, fontID, font)
			if score > 0 {
				result := r.createSearchResult(id, font, sourceID, source.Name)
				result.Score = score
				results = append(results, result)
			}
		}
	}

	// Sort by score (highest first)
	r.sortResultsByScore(results)

	// Filter by category if specified
	if category != "" {
		results = r.filterByCategory(results, category)
	}

	// Cache the results
	r.cacheSearchResults(query, category, results)

	return results, nil
}

// calculateMatchScore calculates a score for how well a font matches the query
func (r *Repository) calculateMatchScore(query, fontName, fontID string, font FontInfo) int {
	score := 0

	// Check for exact match of the base font name
	if fontName == query {
		score += 100 // Highest score for exact base name match
	} else {
		// Check name matches
		if strings.HasPrefix(fontName, query) {
			score += 80 // High score for prefix matches
		} else if strings.Contains(fontName, query) {
			score += 50 // Medium score for contains matches
		}

		// Check ID matches
		if strings.HasPrefix(fontID, query) {
			score += 40 // High score for ID prefix matches
		} else if strings.Contains(fontID, query) {
			score += 20 // Lower score for ID contains matches
		}

		// Check categories
		for _, category := range font.Categories {
			if strings.Contains(strings.ToLower(category), query) {
				score += 10 // Low score for category matches
			}
		}

		// Check tags
		for _, tag := range font.Tags {
			if strings.Contains(strings.ToLower(tag), query) {
				score += 5 // Lowest score for tag matches
			}
		}
	}

	return score
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
	}
}

// sortResultsByScore sorts results by their score in descending order, then alphabetically by name
func (r *Repository) sortResultsByScore(results []SearchResult) {
	sort.Slice(results, func(i, j int) bool {
		// First sort by score (highest first)
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		// If scores are equal, sort alphabetically by name
		return results[i].Name < results[j].Name
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

// getCachedSearchResults retrieves cached search results if they exist and are valid
func (r *Repository) getCachedSearchResults(query, category string) ([]SearchResult, error) {
	cache, err := NewCache()
	if err != nil {
		return nil, err
	}

	// Create cache filename based on query and category
	cacheKey := fmt.Sprintf("search_%s_%s", strings.ReplaceAll(query, " ", "_"), strings.ReplaceAll(category, " ", "_"))
	cacheFile := filepath.Join(cache.Dir, "search", cacheKey+".json")

	// Check if cache file exists and is less than 1 hour old
	if info, err := os.Stat(cacheFile); err != nil {
		return nil, err
	} else if time.Since(info.ModTime()) > time.Hour {
		return nil, fmt.Errorf("cache expired")
	}

	// Read cache file
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}

	var entry SearchCacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}

	// Verify the query matches (ignore category for now since original struct doesn't have it)
	if entry.Query != query {
		return nil, fmt.Errorf("cache mismatch")
	}

	return entry.Results, nil
}

// cacheSearchResults caches search results for future use
func (r *Repository) cacheSearchResults(query, category string, results []SearchResult) error {
	cache, err := NewCache()
	if err != nil {
		return err
	}

	// Create search cache directory
	searchCacheDir := filepath.Join(cache.Dir, "search")
	if err := os.MkdirAll(searchCacheDir, 0755); err != nil {
		return err
	}

	// Create cache entry
	entry := SearchCacheEntry{
		Query:      query,
		ExactMatch: false, // We don't use exact match in Repository search
		Results:    results,
		Timestamp:  time.Now(),
	}

	// Create cache filename
	cacheKey := fmt.Sprintf("search_%s_%s", strings.ReplaceAll(query, " ", "_"), strings.ReplaceAll(category, " ", "_"))
	cacheFile := filepath.Join(searchCacheDir, cacheKey+".json")

	// Marshal and save
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, data, 0644)
}
