package repo

import (
	"fmt"
	"strings"
)

// SearchResult represents a font search result
type SearchResult struct {
	Name       string   `json:"name"`
	ID         string   `json:"id"`
	Source     string   `json:"source"`
	License    string   `json:"license"`
	Variants   []string `json:"variants"`
	Categories []string `json:"categories,omitempty"`
	Tags       []string `json:"tags,omitempty"`
}

// SearchFonts searches for fonts matching the query
func SearchFonts(query string, exactMatch bool) ([]SearchResult, error) {
	manifest, err := GetManifest(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}

	query = normalizeFontName(query)
	var results []SearchResult

	// Search through each source
	for sourceID, source := range manifest.Sources {
		for id, font := range source.Fonts {
			// Check both the font name and ID
			normalizedName := normalizeFontName(font.Name)
			normalizedID := normalizeFontName(id)

			if exactMatch {
				if normalizedName == query || normalizedID == query {
					results = append(results, createSearchResult(id, font, sourceID, source.Name))
				}
			} else {
				if strings.Contains(normalizedName, query) || strings.Contains(normalizedID, query) {
					results = append(results, createSearchResult(id, font, sourceID, source.Name))
				}
			}
		}
	}

	return results, nil
}

// createSearchResult creates a SearchResult from FontInfo
func createSearchResult(id string, font FontInfo, sourceID, sourceName string) SearchResult {
	return SearchResult{
		Name:       font.Name,
		ID:         id,
		Source:     sourceName,
		License:    font.License,
		Variants:   font.Variants,
		Categories: font.Categories,
		Tags:       font.Tags,
	}
}

// normalizeFontName normalizes a font name for comparison
func normalizeFontName(name string) string {
	// Convert to lowercase
	name = strings.ToLower(name)
	// Remove spaces and special characters
	name = strings.ReplaceAll(name, " ", "")
	name = strings.ReplaceAll(name, "-", "")
	name = strings.ReplaceAll(name, "_", "")
	return name
}

// formatFontName formats a font name for display
func formatFontName(name string) string {
	// Convert camelCase to Title Case with spaces
	// Example: "FiraSans" -> "Fira Sans"
	var result strings.Builder
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune(' ')
		}
		result.WriteRune(r)
	}
	return result.String()
}
