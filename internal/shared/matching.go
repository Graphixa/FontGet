package shared

import (
	"fmt"
	"strings"

	"fontget/internal/repo"
)

// FindSimilarFonts finds similar fonts using fuzzy matching.
// This is a unified version that works for both repository fonts and installed fonts.
func FindSimilarFonts(fontName string, allFonts []string, isInstalledFonts bool) []string {
	if isInstalledFonts {
		// For installed fonts, use simpler matching for speed
		queryLower := strings.ToLower(fontName)
		queryNorm := strings.ReplaceAll(queryLower, " ", "")
		queryNorm = strings.ReplaceAll(queryNorm, "-", "")
		queryNorm = strings.ReplaceAll(queryNorm, "_", "")

		var similar []string
		seen := make(map[string]bool)
		similar = findMatchesInInstalledFonts(queryLower, queryNorm, allFonts, similar, seen, 5)
		return similar
	}

	// For repository fonts, use sophisticated scoring algorithm
	// Use sophisticated scoring (popularity controlled by switch in sources.go)
	similar, err := findSimilarFontsWithScoring(fontName, false)
	if err != nil {
		// Fallback to simple matching if sophisticated scoring fails
		queryLower := strings.ToLower(fontName)
		queryNorm := strings.ReplaceAll(queryLower, " ", "")
		queryNorm = strings.ReplaceAll(queryNorm, "-", "")
		queryNorm = strings.ReplaceAll(queryNorm, "_", "")

		var fallbackSimilar []string

		// Separate font names from font IDs for prioritized matching
		var fontNames []string // "Open Sans", "Roboto", etc.
		var fontIDs []string   // "google.roboto", "nerd.fira-code", etc.

		for _, font := range allFonts {
			if strings.Contains(font, ".") {
				fontIDs = append(fontIDs, font)
			} else {
				fontNames = append(fontNames, font)
			}
		}

		// Phase 1: Check font names first (higher priority)
		fallbackSimilar = findMatchesInList(queryLower, queryNorm, fontNames, fallbackSimilar, 5)

		// Phase 2: If we need more results, check font IDs
		if len(fallbackSimilar) < 5 {
			remaining := 5 - len(fallbackSimilar)
			fallbackSimilar = findMatchesInList(queryLower, queryNorm, fontIDs, fallbackSimilar, remaining)
		}

		return fallbackSimilar
	}

	return similar
}

// findSimilarFontsWithScoring finds similar fonts using the sophisticated scoring algorithm.
// This provides better matching with position-based scoring and optional popularity support.
func findSimilarFontsWithScoring(fontName string, _ bool) ([]string, error) {
	// Get repository for sophisticated scoring
	r, err := repo.GetRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize repository: %w", err)
	}

	// Use the repository's search function (popularity controlled by switch in sources.go)
	results, err := r.SearchFonts(fontName, "")
	if err != nil {
		return nil, fmt.Errorf("failed to search fonts: %w", err)
	}

	// Convert SearchResults to font names/IDs for display
	var similar []string
	for _, result := range results {
		// Use the full ID to preserve source information for proper sorting
		similar = append(similar, result.ID)

		// Limit to 5 results
		if len(similar) >= 5 {
			break
		}
	}

	return similar, nil
}

// findMatchesInInstalledFonts performs fuzzy matching on installed fonts (simplified for speed)
func findMatchesInInstalledFonts(queryLower, queryNorm string, fontList []string, existing []string, seen map[string]bool, maxResults int) []string {
	similar := existing

	// Simple substring matching for speed
	for _, font := range fontList {
		if len(similar) >= maxResults {
			break
		}

		fontLower := strings.ToLower(font)
		fontNorm := strings.ReplaceAll(fontLower, " ", "")
		fontNorm = strings.ReplaceAll(fontNorm, "-", "")
		fontNorm = strings.ReplaceAll(fontNorm, "_", "")

		// Skip exact equals and already found fonts
		if fontLower == queryLower || fontNorm == queryNorm || seen[font] {
			continue
		}

		if strings.Contains(fontLower, queryLower) || strings.Contains(queryLower, fontLower) {
			similar = append(similar, font)
			seen[font] = true
		}
	}

	// If no substring matches and we still need more, try partial word matches
	if len(similar) < maxResults {
		words := strings.Fields(queryLower)
		for _, font := range fontList {
			if len(similar) >= maxResults || seen[font] {
				break
			}

			fontLower := strings.ToLower(font)
			for _, word := range words {
				if len(word) > 2 && strings.Contains(fontLower, word) {
					similar = append(similar, font)
					seen[font] = true
					break
				}
			}
		}
	}

	return similar
}

// findMatchesInList performs fuzzy matching on a specific list of fonts (for repository fonts)
func findMatchesInList(queryLower, queryNorm string, fontList []string, existing []string, maxResults int) []string {
	similar := existing
	seen := make(map[string]bool)

	// Simple substring matching for speed
	for _, font := range fontList {
		if len(similar) >= maxResults {
			break
		}

		fontLower := strings.ToLower(font)
		fontNorm := strings.ReplaceAll(fontLower, " ", "")

		// Skip exact equals and already found fonts
		if fontLower == queryLower || fontNorm == queryNorm || seen[font] {
			continue
		}

		if strings.Contains(fontLower, queryLower) || strings.Contains(queryLower, fontLower) {
			similar = append(similar, font)
			seen[font] = true
		}
	}

	return similar
}
