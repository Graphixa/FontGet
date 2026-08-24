package repo

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

// destinationCollisionKey returns a platform-stable key for detecting duplicate destinations.
// Paths are slash-normalized and case-folded so Windows case collisions are rejected everywhere.
func destinationCollisionKey(rel string) string {
	return strings.ToLower(filepath.ToSlash(rel))
}

// checkDestinationCollisions rejects exact duplicates and case-only / separator collisions
// among selected destination-relative paths (forward-slash form).
func checkDestinationCollisions(rels []string) error {
	seen := make(map[string]string, len(rels))
	for _, rel := range rels {
		key := destinationCollisionKey(rel)
		if prev, ok := seen[key]; ok {
			return fmt.Errorf("%w: %q and %q", ErrArchivePathCollision, prev, rel)
		}
		seen[key] = rel
	}
	return nil
}

// classifyArchiveEntryType sets Type/Extension from Name and IsDir.
func classifyArchiveEntryType(e *ArchiveEntry) {
	if e.IsDir {
		e.Type = ArchiveEntryDir
		return
	}
	ext := strings.ToLower(path.Ext(e.Name))
	e.Extension = ext
	if isFontFile(e.Name) {
		e.Type = ArchiveEntryFont
		return
	}
	e.Type = ArchiveEntryOther
}

// fontCandidateEntries returns non-directory font-like entries with safe normalized paths.
// Entries with unsafe paths are skipped (they are never selected or extracted).
func fontCandidateEntries(entries []ArchiveEntry) []ArchiveEntry {
	out := make([]ArchiveEntry, 0, len(entries))
	for _, e := range entries {
		if e.IsDir || e.Type != ArchiveEntryFont {
			continue
		}
		rel, ok := safeArchiveRelPath(e.Name)
		if !ok {
			continue
		}
		e.NormalizedPath = rel
		out = append(out, e)
	}
	return out
}

// SelectArchiveFontEntries applies source-aware / agnostic path selection to font candidates,
// enforces MaxSelectedFiles, and rejects colliding destinations.
func SelectArchiveFontEntries(entries []ArchiveEntry, ctx ArchiveSelectionContext, policy ExtractionPolicy) ([]ArchiveEntry, error) {
	candidates := fontCandidateEntries(entries)
	if len(candidates) == 0 {
		return nil, nil
	}

	names := make([]string, len(candidates))
	byPath := make(map[string]ArchiveEntry, len(candidates))
	for i, e := range candidates {
		names[i] = e.NormalizedPath
		byPath[e.NormalizedPath] = e
	}

	pickedNames := pickArchiveCandidates(names, ctx.SourcePrefix, ctx.FontName, ctx.FontID)
	pickedNames = filterArchiveExtractedDesktopFonts(pickedNames)
	if len(pickedNames) == 0 {
		return nil, nil
	}

	if policy.MaxSelectedFiles > 0 && len(pickedNames) > policy.MaxSelectedFiles {
		return nil, fmt.Errorf("%w: selected %d (limit %d)", ErrArchiveSelectedFileLimit, len(pickedNames), policy.MaxSelectedFiles)
	}

	if err := checkDestinationCollisions(pickedNames); err != nil {
		return nil, err
	}

	selected := make([]ArchiveEntry, 0, len(pickedNames))
	seen := make(map[string]struct{}, len(pickedNames))
	for _, n := range pickedNames {
		key := filepath.ToSlash(n)
		e, ok := byPath[key]
		if !ok {
			continue
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		selected = append(selected, e)
	}
	return selected, nil
}
