package repo

import (
	"fmt"
	"strings"

	"fontget/internal/output"
)

// InstalledFontMatch represents a matched font from the repository for an installed font
type InstalledFontMatch struct {
	FontID     string
	License    string
	Categories []string
	Source     string
}

// fontIndexEntry represents an entry in the font matching index
type fontIndexEntry struct {
	FontID     string
	License    string
	Categories []string
	Source     string
	Priority   int // Source priority (lower = higher priority)
}

// fontIndex is a lookup structure for fast font matching
// Maps normalized names to font entries, with priority tracking
type fontIndex struct {
	// Map of normalized font names -> entries (multiple entries possible, sorted by priority)
	byName map[string][]fontIndexEntry
	// Map of normalized font ID (without prefix) -> entries
	byIDName map[string][]fontIndexEntry
}

// normalizeFamilyName normalizes a font family name for matching
// - Converts to lowercase
// - Removes spaces, hyphens, underscores
func normalizeFamilyName(name string) string {
	normalized := strings.ToLower(name)
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, "-", "")
	normalized = strings.ReplaceAll(normalized, "_", "")
	return normalized
}

// extractBaseFontName extracts the base font name by removing common suffixes
// This handles cases where installed fonts have suffixes that aren't in the repository Font ID
// Examples:
//   - "JetBrainsMono Nerd Font" -> "JetBrainsMono"
//   - "JetBrainsMono Nerd Font Mono" -> "JetBrainsMono"
//   - "JetBrainsMonoNL Nerd Font" -> "JetBrainsMono" (removes both "NL" and " Nerd Font")
//   - "FiraCode Nerd Font" -> "FiraCode"
//   - "Roboto" -> "Roboto" (no change if no suffix)
//
// Returns the original name if no suffix pattern is found
func extractBaseFontName(familyName string) string {
	lower := strings.ToLower(familyName)

	// First, remove "Nerd Font" suffix patterns
	nerdPatterns := []string{
		" nerd font",
		"nerdfont",
		" nerd",
	}

	var baseName string
	foundNerdPattern := false
	for _, pattern := range nerdPatterns {
		if idx := strings.Index(lower, pattern); idx > 0 {
			// Extract the base name before the pattern
			baseName = familyName[:idx]
			baseName = strings.TrimSpace(baseName)
			foundNerdPattern = true
			break
		}
	}

	// If no Nerd Font pattern found, use the original name
	if !foundNerdPattern {
		baseName = familyName
	}

	// Now remove variant suffixes that might be part of the base name
	// These are common font variant suffixes that don't appear in repository Font IDs
	// Note: We don't remove "Mono" because it's often part of the base font name (e.g., "JetBrainsMono")
	variantSuffixes := []string{
		"NL",           // No Ligatures (e.g., "JetBrainsMonoNL" -> "JetBrainsMono")
		"Propo",        // Proportional variant
		"Proportional", // Proportional variant (full word)
	}

	// Try removing variant suffixes from the end of the base name
	baseLower := strings.ToLower(baseName)
	for _, suffix := range variantSuffixes {
		suffixLower := strings.ToLower(suffix)
		// Check if the base name ends with this suffix (case-insensitive)
		if strings.HasSuffix(baseLower, suffixLower) {
			// Remove the suffix
			baseName = baseName[:len(baseName)-len(suffix)]
			baseName = strings.TrimSpace(baseName)
			baseLower = strings.ToLower(baseName)
		}
	}

	return baseName
}

// buildFontIndex builds an optimized lookup index from the manifest
// This allows O(1) lookups instead of O(n) iterations
func buildFontIndex(manifest *FontManifest) *fontIndex {
	index := &fontIndex{
		byName:   make(map[string][]fontIndexEntry),
		byIDName: make(map[string][]fontIndexEntry),
	}

	if manifest == nil || manifest.Sources == nil {
		return index
	}

	// Get sources in priority order
	sourceOrder := getSourcesInPriorityOrder(manifest)

	// Build index by iterating through sources in priority order
	for _, sourceName := range sourceOrder {
		source, exists := manifest.Sources[sourceName]
		if !exists || source.Fonts == nil {
			continue
		}

		priority := getSourcePriority(sourceName)

		// Index all fonts in this source
		for fontID, font := range source.Fonts {
			entry := fontIndexEntry{
				FontID:     fontID,
				License:    font.License,
				Categories: font.Categories,
				Source:     sourceName,
				Priority:   priority,
			}

			// Index by normalized font name
			normalizedName := normalizeFamilyName(font.Name)
			index.byName[normalizedName] = append(index.byName[normalizedName], entry)

			// Index by font ID name (without prefix)
			if strings.Contains(fontID, ".") {
				idParts := strings.Split(fontID, ".")
				if len(idParts) > 1 {
					idName := strings.Join(idParts[1:], ".")
					normalizedIDName := normalizeFamilyName(idName)
					index.byIDName[normalizedIDName] = append(index.byIDName[normalizedIDName], entry)
				}
			} else {
				// Font ID without prefix
				normalizedFontID := normalizeFamilyName(fontID)
				index.byIDName[normalizedFontID] = append(index.byIDName[normalizedFontID], entry)
			}
		}
	}

	return index
}

// findBestMatch finds the best matching font entry from a list of candidates
// Returns the entry with the highest priority (lowest priority number)
func findBestMatch(candidates []fontIndexEntry) *fontIndexEntry {
	if len(candidates) == 0 {
		return nil
	}

	best := candidates[0]
	for _, candidate := range candidates[1:] {
		if candidate.Priority < best.Priority {
			best = candidate
		}
	}

	return &best
}

// MatchInstalledFontToRepository matches an installed font family name to a repository font
// Uses an optimized index for O(1) lookups instead of O(n) iterations
// Returns the first match found based on repository priority order
func MatchInstalledFontToRepository(familyName string, index *fontIndex, isProtectedFont func(string) bool) (*InstalledFontMatch, error) {
	if index == nil {
		return nil, nil
	}

	// Skip protected system fonts - they should not be matched
	if isProtectedFont != nil && isProtectedFont(familyName) {

		return nil, nil
	}

	normalizedFamily := normalizeFamilyName(familyName)

	// Extract base name (removes common suffixes like " Nerd Font")
	// This allows matching "JetBrainsMono Nerd Font" to "nerd.jetbrains-mono"
	baseFontName := extractBaseFontName(familyName)
	normalizedBaseName := normalizeFamilyName(baseFontName)
	hasSuffix := baseFontName != familyName // True if we extracted a base name (e.g., Nerd Font pattern)

	// If we detected a suffix pattern, infer the expected source name from the pattern
	// This prevents false matches (e.g., "JetBrainsMono Nerd Font" matching "google.jetbrains-mono")
	var expectedSourceName string
	if hasSuffix {
		lower := strings.ToLower(familyName)
		if strings.Contains(lower, "nerd") {
			expectedSourceName = "Nerd Fonts"
		}
		// Future: could add more pattern-based source detection here
	}

	// Try exact match by normalized font name
	if candidates, exists := index.byName[normalizedFamily]; exists {
		// Filter by expected source if we have one
		if expectedSourceName != "" {
			for _, candidate := range candidates {
				if candidate.Source == expectedSourceName {
					return &InstalledFontMatch{
						FontID:     candidate.FontID,
						License:    candidate.License,
						Categories: candidate.Categories,
						Source:     candidate.Source,
					}, nil
				}
			}
		}
		// No source filter or no match with filter, use best priority match
		if best := findBestMatch(candidates); best != nil {
			return &InstalledFontMatch{
				FontID:     best.FontID,
				License:    best.License,
				Categories: best.Categories,
				Source:     best.Source,
			}, nil
		}
	}

	// Try match by font ID name (normalized)
	if candidates, exists := index.byIDName[normalizedFamily]; exists {
		// Filter by expected source if we have one
		if expectedSourceName != "" {
			for _, candidate := range candidates {
				if candidate.Source == expectedSourceName {
					return &InstalledFontMatch{
						FontID:     candidate.FontID,
						License:    candidate.License,
						Categories: candidate.Categories,
						Source:     candidate.Source,
					}, nil
				}
			}
		}
		// No source filter or no match with filter, use best priority match
		if best := findBestMatch(candidates); best != nil {
			return &InstalledFontMatch{
				FontID:     best.FontID,
				License:    best.License,
				Categories: best.Categories,
				Source:     best.Source,
			}, nil
		}
	}

	// If we have a suffix pattern, try base name matching
	if hasSuffix {
		// Try base name match by ID name
		if candidates, exists := index.byIDName[normalizedBaseName]; exists {
			// Only match if source matches expected (prevents false positives)
			if expectedSourceName != "" {
				for _, candidate := range candidates {
					if candidate.Source == expectedSourceName {
						return &InstalledFontMatch{
							FontID:     candidate.FontID,
							License:    candidate.License,
							Categories: candidate.Categories,
							Source:     candidate.Source,
						}, nil
					}
				}
			} else {
				// No expected source, use best priority match
				if best := findBestMatch(candidates); best != nil {
					return &InstalledFontMatch{
						FontID:     best.FontID,
						License:    best.License,
						Categories: best.Categories,
						Source:     best.Source,
					}, nil
				}
			}
		}
	}

	// No match found
	return nil, nil
}

// getSourcesInPriorityOrder returns source names sorted by priority (lower number = higher priority)
func getSourcesInPriorityOrder(manifest *FontManifest) []string {
	type sourcePriority struct {
		name     string
		priority int
	}

	var sourceList []sourcePriority
	for name := range manifest.Sources {
		priority := getSourcePriority(name)
		sourceList = append(sourceList, sourcePriority{
			name:     name,
			priority: priority,
		})
	}

	// Sort by priority (lower number = higher priority)
	for i := 0; i < len(sourceList)-1; i++ {
		for j := i + 1; j < len(sourceList); j++ {
			if sourceList[i].priority > sourceList[j].priority {
				sourceList[i], sourceList[j] = sourceList[j], sourceList[i]
			}
		}
	}

	// Extract names in order
	result := make([]string, len(sourceList))
	for i, s := range sourceList {
		result[i] = s.name
	}

	return result
}

// MatchAllInstalledFonts matches all installed font family names to repository fonts
// Uses an optimized index for fast O(1) lookups instead of O(n) iterations
// isProtectedFont is a function to check if a font is a protected system font (can be nil)
func MatchAllInstalledFonts(familyNames []string, isProtectedFont func(string) bool) (map[string]*InstalledFontMatch, error) {
	// Load manifest
	manifest, err := GetCachedManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
	}

	// Build optimized index once
	index := buildFontIndex(manifest)
	output.GetDebug().State("Built font matching index: %d normalized name entries, %d normalized ID entries", len(index.byName), len(index.byIDName))

	matches := make(map[string]*InstalledFontMatch)

	// Match each family name using the index (O(1) lookups)
	for _, familyName := range familyNames {
		match, err := MatchInstalledFontToRepository(familyName, index, isProtectedFont)
		if err != nil {
			// Log error but continue
			output.GetDebug().Error("Matching failed for family %s: %v", familyName, err)
			continue
		}

		if match != nil {
			matches[familyName] = match
			// Note: We don't log individual matches to reduce debug noise - only show summary at end
			// Base name matches are still logged above as they're useful for debugging matching logic
		}
		// Note: We don't log "no match" cases to reduce debug noise - it's expected that many
		// installed fonts (system fonts, custom fonts, etc.) won't be in the repository
	}

	// Show summary of matching results
	matchCount := len(matches)
	output.GetDebug().State("Matching complete: %d matches found out of %d families", matchCount, len(familyNames))

	return matches, nil
}
