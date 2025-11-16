# Font Matching Feature: Repository Integration for Installed Fonts

## Goal

Enhance the `list` command to match installed fonts with fonts in the repository sources, enabling:
1. Display of license information for installed fonts (when matched to a repository font)
2. Display of source information (e.g., "Google Fonts", "Nerd Fonts") for installed fonts
3. Display of Font ID (e.g., `google.roboto`, `nerd-fonts.fira-code`) for matched fonts
4. Future capability: Allow the `remove` command to accept Font IDs instead of just font names

## Context

### Current State

**Installed Fonts:**
- The `list` command currently identifies fonts by:
  - Font file paths on the system
  - SFNT metadata extraction (family name, style name) via `platform.ExtractFontMetadata()`
  - File modification dates
  - Installation scope (user/machine)
- See `cmd/list.go` for current implementation

**Repository Fonts:**
- Fonts in sources are identified by Font IDs in the format: `{source}.{fontname}`
  - Examples: `google.roboto`, `nerd-fonts.fira-code`, `font-squirrel.abel`
- Font IDs are stored in the repository manifest (see `internal/repo/`)
- Each font has metadata including:
  - Name (display name)
  - License information
  - Source name
  - Variants/files
- See `internal/repo/font.go` for Font ID structure and `GetFontByID()` function

**Current Matching:**
- The `add` command can match font names to Font IDs using `repo.FindFontMatches()` and `repo.GetFontByID()`
- The `remove` command currently only works with font family names (not Font IDs)
- There is no current mechanism to match an installed font back to its repository Font ID

### Reference: winget Pattern

Similar to how Windows Package Manager (winget) matches installed software to packages in their repository using App IDs, we need to:
1. Match installed fonts to repository fonts using available metadata
2. Store or compute this association
3. Display repository information when available
4. Allow operations using Font IDs

## Challenges

1. **Matching Strategy:**
   - Installed fonts only have file names and SFNT metadata (family name, style name)
   - Repository fonts have Font IDs, names, and potentially different naming conventions
   - Need to match: installed font family name → repository font name → Font ID
   - Handle edge cases: fonts installed manually (not from repository), fonts with different names in repository vs. installed

2. **Matching Accuracy:**
   - Exact name matching (case-insensitive, normalized)
   - Fuzzy matching for variations (e.g., "Roboto" vs "Roboto Mono")
   - Handling of font variants (Regular, Bold, Italic, etc.)

3. **Data Storage:**
   - Should we cache matches? Where?
   - Should we store Font ID associations with installed fonts?
   - How to handle fonts that can't be matched?

4. **Performance:**
   - Matching all installed fonts against all repository fonts could be slow
   - Need efficient lookup strategy

## Investigation Tasks

1. **Understand Font ID Structure:**
   - Review `internal/repo/font.go` to understand Font ID format and structure
   - Review `internal/repo/search.go` to understand how font matching currently works
   - Identify what metadata is available for matching

2. **Review Current List Command:**
   - Study `cmd/list.go` to understand current font collection and display
   - Identify where license/source info should be displayed
   - Understand the `ParsedFont` struct and how to extend it

3. **Review Font Metadata:**
   - Study `platform.ExtractFontMetadata()` to understand what metadata is available from installed fonts
   - Compare with repository font metadata structure
   - Identify best matching fields

4. **Design Matching Algorithm:**
   - Propose a matching strategy (exact match, fuzzy match, fallback)
   - Consider performance implications
   - Handle edge cases (unmatched fonts, multiple matches)

5. **Design Data Model:**
   - Extend `ParsedFont` struct or create new structure to include:
     - Matched Font ID (if found)
     - Source name
     - License information
   - Decide on caching strategy

6. **Design UI/UX:**
   - How to display license info in list output
   - How to display source info
   - How to display Font ID (always? only in verbose mode?)
   - Formatting considerations

## Implementation Plan Request

Please provide:
1. **Matching Strategy:** Detailed algorithm for matching installed fonts to repository fonts
2. **Data Model Changes:** Proposed changes to `ParsedFont` or new structures
3. **Implementation Steps:** Step-by-step plan for implementation
4. **Code Locations:** Specific files and functions that need modification
5. **Testing Strategy:** How to test matching accuracy and edge cases
6. **Performance Considerations:** How to ensure matching is fast enough for large font collections

## Files to Review

- `cmd/list.go` - Current list command implementation
- `internal/repo/font.go` - Font ID structure and repository access
- `internal/repo/search.go` - Font matching logic
- `internal/platform/platform.go` - Font metadata extraction
- `cmd/add.go` - Example of Font ID usage (lines 409-436)
- `cmd/remove.go` - Current removal logic (to understand future Font ID integration)

## Success Criteria

1. `fontget list` shows license information for matched fonts
2. `fontget list` shows source information (e.g., "[Google Fonts]") for matched fonts
3. `fontget list --verbose` shows Font IDs for matched fonts
4. Matching is accurate and handles edge cases gracefully
5. Performance is acceptable for typical font collections (100-500 fonts)
6. Unmatched fonts still display correctly (without license/source info)

## Future Enhancement

Once this is implemented, the `remove` command should be enhanced to:
- Accept Font IDs as input (e.g., `fontget remove google.roboto`)
- Use the matching logic to find installed fonts by Font ID
- This will be a separate task, but the matching infrastructure should support it

