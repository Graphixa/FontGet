package repo

import (
	"archive/zip"
	"fmt"
	"path"
	"strings"
)

// InspectZIP reads the ZIP central directory and returns entry metadata without extracting.
func InspectZIP(archivePath string) ([]ArchiveEntry, error) {
	return InspectZIPWithPolicy(archivePath, DefaultExtractionPolicy())
}

// InspectZIPWithPolicy is like InspectZIP but enforces MaxArchiveEntries from policy.
func InspectZIPWithPolicy(archivePath string, policy ExtractionPolicy) ([]ArchiveEntry, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open ZIP file: %w", err)
	}
	defer reader.Close()

	if policy.MaxArchiveEntries > 0 && len(reader.File) > policy.MaxArchiveEntries {
		return nil, fmt.Errorf("%w: %d entries (limit %d)", ErrArchiveEntryCountLimit, len(reader.File), policy.MaxArchiveEntries)
	}

	entries := make([]ArchiveEntry, 0, len(reader.File))
	for _, f := range reader.File {
		isDir := f.FileInfo().IsDir() || strings.HasSuffix(f.Name, "/")
		e := ArchiveEntry{
			Name:             f.Name,
			IsDir:            isDir,
			UncompressedSize: f.UncompressedSize64,
		}
		if !isDir {
			if rel, ok := safeArchiveRelPath(f.Name); ok {
				e.NormalizedPath = rel
			}
		}
		e.Extension = strings.ToLower(path.Ext(f.Name))
		classifyArchiveEntryType(&e)
		entries = append(entries, e)
	}
	return entries, nil
}
