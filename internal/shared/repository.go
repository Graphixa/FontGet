// Package shared provides general-purpose, domain-agnostic utilities.
//
// This file contains business logic for font query resolution and source name extraction.
// These functions contain pure business logic with no CLI dependencies and can be used
// by any package that needs to resolve font queries or extract source information.

package shared

import (
	"fmt"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"fontget/internal/config"
	"fontget/internal/repo"
)

// PlaceholderNA is used for missing table data
const PlaceholderNA = "N/A"

// FontResolutionResult contains the result of resolving a font query
type FontResolutionResult struct {
	Fonts              []repo.FontFile
	SourceName         string
	FontID             string
	HasMultipleMatches bool
	Matches            []repo.FontMatch // Only populated if HasMultipleMatches is true
}

// ResolveFontQuery resolves a font query (Font ID or name) to FontFile list.
// Returns an error if font is not found, or sets HasMultipleMatches if multiple matches exist.
//
// NOTE: This function is tested via integration tests (see cmd/integration_test.go)
// because it depends on package-level repo functions that are difficult to mock.
func ResolveFontQuery(fontName string) (*FontResolutionResult, error) {
	result := &FontResolutionResult{}

	// Check if this is already a specific font ID (contains a dot like "google.roboto")
	if strings.Contains(fontName, ".") {
		// This is a specific font ID, use it directly
		// Convert to lowercase for consistent matching
		fontID := strings.ToLower(fontName)
		fonts, err := repo.GetFontByID(fontID)
		if err != nil {
			return nil, fmt.Errorf("font not found: %s", fontName)
		}
		result.Fonts = fonts
		result.FontID = fontID
		result.SourceName = GetSourceNameFromID(fontID)
		return result, nil
	}

	// Find all matches across sources
	matches, err := repo.FindFontMatches(fontName)
	if err != nil {
		return nil, fmt.Errorf("font not found: %s", fontName)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("font not found: %s", fontName)
	}

	if len(matches) == 1 {
		// Single match, proceed normally
		fontID := matches[0].ID
		fonts, err := repo.GetFontByID(fontID)
		if err != nil {
			return nil, fmt.Errorf("font not found: %s", fontName)
		}
		result.Fonts = fonts
		result.FontID = fontID
		result.SourceName = matches[0].Source
		return result, nil
	}

	// Multiple matches - return them for caller to handle
	result.HasMultipleMatches = true
	result.Matches = matches
	return result, nil
}

// GetSourceNameFromID extracts the source name from a font ID
func GetSourceNameFromID(fontID string) string {
	// Extract source prefix from font ID (e.g., "google.roboto" -> "google")
	parts := strings.Split(fontID, ".")
	if len(parts) < 2 {
		return "Unknown Source"
	}

	sourcePrefix := strings.ToLower(parts[0])

	// Load manifest to get the display name
	manifest, err := config.LoadManifest()
	if err != nil {
		// Fallback to capitalized prefix if we can't load manifest
		return cases.Title(language.English).String(sourcePrefix)
	}

	// Find the source with matching prefix (case-insensitive)
	for sourceName, sourceConfig := range manifest.Sources {
		if strings.ToLower(sourceConfig.Prefix) == sourcePrefix {
			return sourceName
		}
	}

	// Fallback to capitalized prefix if not found
	return cases.Title(language.English).String(sourcePrefix)
}

// FindMatchesInRepository finds font matches using an already-loaded repository (performance optimization).
// This is useful when you already have a repository loaded and want to avoid reloading it.
func FindMatchesInRepository(repository *repo.Repository, fontName string) []repo.FontMatch {
	// Get the manifest from the repository
	manifest, err := repository.GetManifest()
	if err != nil {
		return nil
	}

	// Normalize font name for comparison
	fontName = strings.ToLower(fontName)
	fontNameNoSpaces := strings.ReplaceAll(fontName, " ", "")

	var matches []repo.FontMatch

	// Search through all sources
	for sourceName, source := range manifest.Sources {
		for id, font := range source.Fonts {
			// Check both the font name and ID with case-insensitive comparison
			fontNameLower := strings.ToLower(font.Name)
			idLower := strings.ToLower(id)
			fontNameNoSpacesLower := strings.ReplaceAll(fontNameLower, " ", "")
			idNoSpacesLower := strings.ReplaceAll(idLower, " ", "")

			// Check for exact match
			if fontNameLower == fontName ||
				fontNameNoSpacesLower == fontNameNoSpaces ||
				idLower == fontName ||
				idNoSpacesLower == fontNameNoSpaces {
				matches = append(matches, repo.FontMatch{
					ID:       id,
					Name:     font.Name,
					Source:   sourceName,
					FontInfo: font,
				})
			}
		}
	}

	return matches
}
