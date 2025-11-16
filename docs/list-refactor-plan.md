# List Command Refactor Plan: Font Matching Feature

## Overview

This document tracks the implementation of font matching functionality for the `list` command, enabling display of repository information (Font ID, License, Categories, Source) for installed fonts.

## Goals

1. Match installed fonts to repository fonts by family name
2. Display Font ID, License, Categories, and Source for matched fonts
3. Cache matches for performance
4. Update cache when sources are updated
5. Maintain 120-character table width limit
6. Follow verbose/debug output guidelines

## Table Format Changes

### Current Format
- Columns: Name (42), ID (34), Type (10), Installed (20), Scope (10)
- Total: 120 characters

### New Format
- Columns: Name (30), Font ID (28), License (8), Categories (16), Type (8), Scope (8), Source (16)
- Total: 30 + 28 + 8 + 16 + 8 + 8 + 16 + 6 spaces = 120 characters ✓

## Implementation Tasks

### 1. Cache Structure (`internal/repo/font_matches.go`)

**New File**: `internal/repo/font_matches.go`

**Cache Structure**:
```go
type FontMatchCache struct {
    Version     string                `json:"version"`
    LastUpdated time.Time             `json:"last_updated"`
    Matches     map[string]FontMatch  `json:"matches"` // family name -> FontMatch
}

type FontMatch struct {
    FontID     string   `json:"font_id"`
    License    string   `json:"license"`
    Categories []string `json:"categories"`
    Source     string   `json:"source"`
}
```

**Functions**:
- `GetFontMatchCachePath() string` - Returns path to `~/.fontget/font-matches.json`
- `LoadFontMatchCache() (*FontMatchCache, error)` - Load cache from disk
- `SaveFontMatchCache(cache *FontMatchCache) error` - Save cache to disk
- `InvalidateFontMatchCache() error` - Delete cache file (called on sources update)

### 2. Matching Algorithm (`internal/repo/font_matches.go`)

**Function**: `MatchInstalledFontToRepository(familyName string, manifest *FontManifest) (*FontMatch, error)`

**Algorithm**:
1. Normalize family name (lowercase, trim spaces)
2. Iterate through sources in priority order (Google Fonts → Nerd Fonts → Font Squirrel → Custom)
3. For each source, search fonts:
   - Exact match: `strings.EqualFold(font.Name, familyName)`
   - Normalized match: `strings.EqualFold(normalize(font.Name), normalize(familyName))`
   - ID match: Check if font ID (without prefix) matches
4. Return first match found (respects priority order)
5. Return `nil` if no match found (unmatched fonts)

**Normalization**:
- Convert to lowercase
- Remove spaces, hyphens, underscores
- Used for fuzzy matching

### 3. Batch Matching (`internal/repo/font_matches.go`)

**Function**: `MatchAllInstalledFonts(familyNames []string, manifest *FontManifest) (map[string]*FontMatch, error)`

**Process**:
1. Load cache if exists
2. Check if cache is valid (sources not updated since cache creation)
3. For each family name:
   - Check cache first
   - If not in cache, perform matching
   - Store result in cache
4. Save updated cache
5. Return matches map

**Cache Validation**:
- Compare `cache.LastUpdated` with `manifest.LastUpdated`
- If manifest is newer, invalidate cache and regenerate

### 4. Extend ParsedFont Struct (`cmd/list.go`)

**Changes**:
```go
type ParsedFont struct {
    Name        string
    Family      string
    Style       string
    Type        string
    InstallDate time.Time
    Scope       string
    // New fields
    FontID      string   // Repository Font ID (e.g., "google.roboto")
    License     string   // License information
    Categories  []string // Font categories
    Source      string   // Source name (e.g., "Google Fonts")
}
```

### 5. Update List Command (`cmd/list.go`)

**Changes**:
1. After collecting fonts, call `MatchAllInstalledFonts()` to get matches
2. Populate `ParsedFont` fields from matches
3. Update table header: `GetListTableHeader()` to include new columns
4. Update table row formatting to display new columns
5. Handle unmatched fonts (empty strings for FontID, License, Categories, Source)

**Table Header**:
```go
func GetListTableHeader() string {
    return fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s %-*s %-*s",
        TableColListName, "Name",
        TableColListID, "ID",
        TableColLicense, "License",
        TableColCategories, "Categories",
        TableColType, "Type",
        TableColScope, "Scope",
        TableColSource, "Source")
}
```

**Table Constants** (update in `cmd/shared.go`):
```go
TableColListName = 30
TableColListID = 28
TableColLicense = 8
TableColCategories = 16
TableColType = 8
TableColScope = 8
TableColSource = 16
```

### 6. Cache Invalidation on Sources Update (`cmd/sources_update.go`)

**Changes**:
- In `updateNextSource()` or completion handler, call `repo.InvalidateFontMatchCache()`
- This ensures cache is cleared when sources are updated

### 7. Verbose/Debug Output

**Verbose Messages** (user-relevant):
- "Matching installed fonts to repository..."
- "Found X matches out of Y installed fonts"
- "Loading font match cache from: %s"
- "Cache invalidated, regenerating matches..."

**Debug Messages** (developer-focused):
- "Calling MatchInstalledFontToRepository() for family: %s"
- "Cache hit for family: %s -> FontID: %s"
- "Cache miss for family: %s, performing matching..."
- "Font match found: %s -> %s (source: %s, priority: %d)"
- "No match found for family: %s"

### 8. Error Handling

- Cache file corruption: Log warning, regenerate cache
- Matching errors: Log debug message, continue with unmatched font
- Manifest loading errors: Return error (can't match without manifest)

## Files to Modify

1. **New Files**:
   - `internal/repo/font_matches.go` - Matching logic and cache management

2. **Modified Files**:
   - `cmd/list.go` - Extend ParsedFont, integrate matching, update display
   - `cmd/shared.go` - Update table column constants and header function
   - `cmd/sources_update.go` - Add cache invalidation on sources update

## Testing Strategy

1. **Test with matched fonts**: Install fonts via `fontget add`, verify they show Font ID, License, etc.
2. **Test with unmatched fonts**: Install fonts manually, verify they show blank fields
3. **Test cache**: Run `list` twice, verify second run uses cache (check debug output)
4. **Test cache invalidation**: Run `sources update`, verify cache is cleared
5. **Test priority order**: Install font that exists in multiple sources, verify highest priority source is shown
6. **Test table formatting**: Verify all columns fit within 120 characters

## Success Criteria

- ✅ `fontget list` shows Font ID, License, Categories, Source for matched fonts
- ✅ Unmatched fonts display correctly with blank fields
- ✅ Cache improves performance on subsequent runs
- ✅ Cache is invalidated when sources are updated
- ✅ Table fits within 120-character width
- ✅ Verbose/debug output follows guidelines
- ✅ Matching respects repository priority order

## Future Enhancements

- Allow `remove` command to accept Font IDs
- Fuzzy matching for slight name variations
- Cache statistics (hit rate, etc.)

