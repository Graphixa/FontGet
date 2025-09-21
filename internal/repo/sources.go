package repo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fontget/internal/config"
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

// loadAllSources loads all enabled sources and merges them into a combined manifest
func loadAllSources(sourcesConfig *config.SourcesConfig, progress ProgressCallback) (*FontManifest, error) {
	return loadAllSourcesWithCache(sourcesConfig, progress, false)
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

			// Convert variants to legacy format
			for _, variant := range font.Variants {
				fontInfo.Variants = append(fontInfo.Variants, variant.Name)
				// Merge variant files into main files
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

// GetRepository returns a new Repository instance
func GetRepository() (*Repository, error) {
	manifest, err := GetManifest(nil, nil)
	if err != nil {
		return nil, err
	}
	return &Repository{manifest: manifest}, nil
}

// GetManifest returns the current manifest
func (r *Repository) GetManifest() (*FontManifest, error) {
	return r.manifest, nil
}

// SearchFonts searches for fonts matching the query
func (r *Repository) SearchFonts(query string, category string) ([]SearchResult, error) {
	// Use the better search implementation from search.go
	results, err := SearchFonts(query, false)
	if err != nil {
		return nil, err
	}

	// Filter by category if specified
	if category != "" {
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
		results = filteredResults
	}

	return results, nil
}
