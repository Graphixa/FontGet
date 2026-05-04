package repo

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"

	"fontget/internal/normalize"
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

// FontIndex is an exported type alias for fontIndex to allow external packages to use it
// while maintaining internal implementation details.
type FontIndex = *fontIndex

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
			normalizedName := normalize.FontKey(font.Name)
			index.byName[normalizedName] = append(index.byName[normalizedName], entry)

			// Index by font ID name (without prefix)
			if strings.Contains(fontID, ".") {
				idParts := strings.Split(fontID, ".")
				if len(idParts) > 1 {
					idName := strings.Join(idParts[1:], ".")
					normalizedIDName := normalize.FontKey(idName)
					index.byIDName[normalizedIDName] = append(index.byIDName[normalizedIDName], entry)
				}
			} else {
				// Font ID without prefix
				normalizedFontID := normalize.FontKey(fontID)
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

	normalizedFamily := normalize.FontKey(familyName)

	// Extract base name (removes common suffixes like " Nerd Font")
	// This allows matching "JetBrainsMono Nerd Font" to "nerd.jetbrains-mono"
	baseFontName := normalize.BaseFamilyName(familyName)
	normalizedBaseName := normalize.FontKey(baseFontName)
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

	sort.Slice(sourceList, func(i, j int) bool {
		return sourceList[i].priority < sourceList[j].priority
	})

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

	matches := make(map[string]*InstalledFontMatch, len(familyNames))
	n := len(familyNames)
	if n == 0 {
		return matches, nil
	}

	// Parallelize O(1) per-family index lookups. Each index slot is written once (no map contention).
	const minParallel = 32
	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		workers = 1
	}
	if workers > 8 {
		workers = 8
	}
	if n < minParallel || workers < 2 {
		for _, familyName := range familyNames {
			match, err := MatchInstalledFontToRepository(familyName, index, isProtectedFont)
			if err != nil {
				output.GetDebug().Error("Matching failed for family %s: %v", familyName, err)
				continue
			}
			if match != nil {
				matches[familyName] = match
			}
		}
	} else {
		ordered := make([]*InstalledFontMatch, n)
		jobs := make(chan int, workers*4)
		var wg sync.WaitGroup
		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := range jobs {
					familyName := familyNames[i]
					match, err := MatchInstalledFontToRepository(familyName, index, isProtectedFont)
					if err != nil {
						output.GetDebug().Error("Matching failed for family %s: %v", familyName, err)
						continue
					}
					ordered[i] = match
				}
			}()
		}
		for i := 0; i < n; i++ {
			jobs <- i
		}
		close(jobs)
		wg.Wait()
		for i, familyName := range familyNames {
			if m := ordered[i]; m != nil {
				matches[familyName] = m
			}
		}
	}

	// Show summary of matching results
	matchCount := len(matches)
	output.GetDebug().State("Matching complete: %d matches found out of %d families", matchCount, len(familyNames))

	return matches, nil
}

// BuildFontIndexForMatching builds a font index for repository matching.
// This is exported for use by commands that need to match fonts efficiently.
// Returns nil if manifest cannot be loaded.
func BuildFontIndexForMatching() (FontIndex, error) {
	manifest, err := GetCachedManifest()
	if err != nil {
		return nil, err
	}
	return buildFontIndex(manifest), nil
}

// IsFontIDInCachedManifest reports whether fontID matches a font entry in the on-disk manifest (exact ID, case-insensitive).
func IsFontIDInCachedManifest(fontID string) bool {
	fontID = strings.TrimSpace(fontID)
	if fontID == "" {
		return false
	}
	manifest, err := GetCachedManifest()
	if err != nil || manifest == nil {
		return false
	}
	_, _, _, ok := lookupFontByIDInManifest(manifest, fontID)
	return ok
}

// lookupFontByIDInManifest resolves font ID across sources using getSourcesInPriorityOrder:
// higher-priority sources (lower priority number) win when duplicate IDs exist across sources.
func lookupFontByIDInManifest(manifest *FontManifest, fontID string) (canonicalID string, font FontInfo, sourceName string, ok bool) {
	want := strings.ToLower(strings.TrimSpace(fontID))
	if manifest == nil || manifest.Sources == nil || want == "" {
		return "", FontInfo{}, "", false
	}
	for _, sn := range getSourcesInPriorityOrder(manifest) {
		source, exists := manifest.Sources[sn]
		if !exists || source.Fonts == nil {
			continue
		}
		for id, f := range source.Fonts {
			if strings.ToLower(strings.TrimSpace(id)) != want {
				continue
			}
			return id, f, sn, true
		}
	}
	return "", FontInfo{}, "", false
}

// IsFontIDInManifest reports whether fontID exists in manifest using the same resolution rules as lookupFontByIDInManifest (no extra manifest load).
func IsFontIDInManifest(manifest *FontManifest, fontID string) bool {
	if manifest == nil {
		return false
	}
	_, _, _, ok := lookupFontByIDInManifest(manifest, fontID)
	return ok
}

// MatchRepositoryFontByID returns repository metadata for an exact font ID if it exists in the cached manifest.
// Returns (nil, nil) when the ID is not found (not an error).
// Uses GetCachedManifest only (no network / refresh) so it is safe for list-time merges.
func MatchRepositoryFontByID(fontID string) (*InstalledFontMatch, error) {
	if strings.TrimSpace(fontID) == "" {
		return nil, nil
	}
	manifest, err := GetCachedManifest()
	if err != nil {
		return nil, err
	}
	id, info, sn, ok := lookupFontByIDInManifest(manifest, fontID)
	if !ok {
		return nil, nil
	}
	return &InstalledFontMatch{
		FontID:     id,
		License:    info.License,
		Categories: info.Categories,
		Source:     sn,
	}, nil
}

// MatchFontFamilyToFontID checks if an installed font family name matches a specific Font ID.
// This is more accurate than string matching and works for all Font ID variants.
// Returns true if the font's Font ID matches the target Font ID.
func MatchFontFamilyToFontID(familyName string, targetFontID string, index FontIndex) bool {
	if index == nil {
		return false
	}

	match, err := MatchInstalledFontToRepository(familyName, index, nil)
	if err != nil || match == nil {
		return false
	}

	// Check if the matched Font ID matches our target Font ID (case-insensitive)
	return strings.EqualFold(match.FontID, targetFontID)
}
