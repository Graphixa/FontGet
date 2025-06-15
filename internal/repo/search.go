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
	SourceName string   `json:"source_name"`
	License    string   `json:"license"`
	Categories []string `json:"categories,omitempty"`
	Score      int      `json:"-"` // Internal score for sorting
}

// SearchFonts searches for fonts matching the query with priority-based matching
func SearchFonts(query string, exactMatch bool) ([]SearchResult, error) {
	manifest, err := GetManifest(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}

	query = strings.ToLower(query)
	var results []SearchResult

	// Search through each source
	for sourceID, source := range manifest.Sources {
		for id, font := range source.Fonts {
			// Check both the font name and ID
			fontName := strings.ToLower(font.Name)
			fontID := strings.ToLower(id)

			if exactMatch {
				if fontName == query || fontID == query {
					result := createSearchResult(id, font, sourceID, source.Name)
					result.Score = 100 // Highest score for exact matches
					results = append(results, result)
				}
			} else {
				score := calculateMatchScore(query, fontName, fontID, font)
				if score > 0 {
					result := createSearchResult(id, font, sourceID, source.Name)
					result.Score = score
					results = append(results, result)
				}
			}
		}
	}

	// Sort results by score (highest first)
	sortResultsByScore(results)

	return results, nil
}

// calculateMatchScore calculates a score for how well a font matches the query
func calculateMatchScore(query, fontName, fontID string, font FontInfo) int {
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

// sortResultsByScore sorts results by their score in descending order, then alphabetically by name
func sortResultsByScore(results []SearchResult) {
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			// First sort by score
			if results[i].Score < results[j].Score {
				results[i], results[j] = results[j], results[i]
			} else if results[i].Score == results[j].Score {
				// If scores are equal, sort alphabetically by name
				if results[i].Name > results[j].Name {
					results[i], results[j] = results[j], results[i]
				}
			}
		}
	}
}

// createSearchResult creates a SearchResult from FontInfo
func createSearchResult(id string, font FontInfo, sourceID, sourceName string) SearchResult {
	return SearchResult{
		Name:       font.Name, // Use the name directly from metadata
		ID:         id,
		Source:     sourceID,
		SourceName: sourceName,
		License:    font.License,
		Categories: font.Categories,
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
