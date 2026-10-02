package repo

import (
	"archive/tar"
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// ArchiveType represents the type of archive file
type ArchiveType int

const (
	ArchiveTypeUnknown ArchiveType = iota
	ArchiveTypeZIP
	ArchiveTypeTARXZ
	ArchiveTypeTARGZ
)

// DetectArchiveType detects the archive type based on file extension
func DetectArchiveType(filename string) ArchiveType {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".zip":
		return ArchiveTypeZIP
	case ".xz":
		// Check if it's a .tar.xz file
		if strings.HasSuffix(strings.ToLower(filename), ".tar.xz") {
			return ArchiveTypeTARXZ
		}
	case ".gz":
		// Check if it's a .tar.gz or .tgz file
		lower := strings.ToLower(filename)
		if strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") {
			return ArchiveTypeTARGZ
		}
	}
	return ArchiveTypeUnknown
}

// DetectArchiveTypeFromFile attempts to detect archive type by inspecting file magic bytes.
// This is necessary because some upstreams (notably Font Squirrel) may serve ZIP archives
// behind URLs/paths that end in ".ttf" or ".otf".
//
// Returns ArchiveTypeUnknown when the file doesn't look like a supported archive.
func DetectArchiveTypeFromFile(path string) ArchiveType {
	f, err := os.Open(path)
	if err != nil {
		return ArchiveTypeUnknown
	}
	defer f.Close()

	var hdr [8]byte
	n, readErr := f.Read(hdr[:])
	if readErr != nil && readErr != io.EOF {
		return ArchiveTypeUnknown
	}
	b := hdr[:n]

	// ZIP: PK\x03\x04 (local file header), PK\x05\x06 (empty archive), PK\x07\x08 (spanned)
	if len(b) >= 4 && b[0] == 'P' && b[1] == 'K' {
		if (b[2] == 3 && b[3] == 4) || (b[2] == 5 && b[3] == 6) || (b[2] == 7 && b[3] == 8) {
			return ArchiveTypeZIP
		}
	}

	// XZ magic: FD 37 7A 58 5A 00
	// We only support TAR.XZ in this codebase. Some upstreams can serve a .tar.xz with the wrong
	// filename extension (e.g., ".ttf"), so treat XZ magic as TAR.XZ and let extraction validate.
	if len(b) >= 6 && b[0] == 0xFD && b[1] == 0x37 && b[2] == 0x7A && b[3] == 0x58 && b[4] == 0x5A && b[5] == 0x00 {
		return ArchiveTypeTARXZ
	}

	// GZIP magic: 1F 8B
	// We treat gzip payloads as TAR.GZ for extraction purposes; tar reader will validate.
	if len(b) >= 2 && b[0] == 0x1F && b[1] == 0x8B {
		return ArchiveTypeTARGZ
	}

	return ArchiveTypeUnknown
}

// ExtractArchive extracts an archive file to the specified directory.
func ExtractArchive(archivePath, destDir string) ([]string, error) {
	return ExtractArchiveWithOptions(archivePath, destDir, nil)
}

// ExtractOptions configures ExtractArchiveWithOptions.
type ExtractOptions struct {
	// Context cancels extraction between archive members. Nil uses Background.
	Context context.Context

	// OnFontFileExtracted is called after each font file is extracted.
	// total is the number of font files that will be extracted when known, otherwise -1.
	OnFontFileExtracted func(done int, total int)

	// Policy overrides default extraction limits when non-nil.
	Policy *ExtractionPolicy

	// Selection, when set, enables source-aware / agnostic (or Nerd package-mode) selection
	// before ZIP and compressed-TAR extraction.
	Selection *ArchiveSelectionContext
}

func extractContext(opts *ExtractOptions) context.Context {
	if opts != nil && opts.Context != nil {
		return opts.Context
	}
	return context.Background()
}

// ExtractArchiveWithOptions extracts an archive file to the specified directory, with optional progress callbacks.
func ExtractArchiveWithOptions(archivePath, destDir string, opts *ExtractOptions) ([]string, error) {
	archiveType := DetectArchiveType(archivePath)
	if archiveType == ArchiveTypeUnknown {
		archiveType = DetectArchiveTypeFromFile(archivePath)
	}

	switch archiveType {
	case ArchiveTypeZIP:
		return extractZIP(archivePath, destDir, opts)
	case ArchiveTypeTARXZ:
		return extractTARXZ(archivePath, destDir, opts)
	case ArchiveTypeTARGZ:
		return extractTARGZ(archivePath, destDir, opts)
	default:
		return nil, fmt.Errorf("unsupported archive format: %s", filepath.Ext(archivePath))
	}
}

func safeArchiveRelPath(name string) (string, bool) {
	raw := strings.TrimSpace(name)
	if raw == "" {
		return "", false
	}
	// Normalize separators for inspection; reject absolutes and traversal before Clean
	// collapses them into seemingly-safe relative paths.
	slash := strings.ReplaceAll(raw, "\\", "/")
	if strings.HasPrefix(slash, "/") || strings.HasPrefix(slash, "~") {
		return "", false
	}
	if len(slash) >= 2 && slash[1] == ':' {
		return "", false
	}
	for _, part := range strings.Split(slash, "/") {
		if part == ".." {
			return "", false
		}
	}

	rel := path.Clean("/" + slash)
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" || rel == "." {
		return "", false
	}
	if strings.HasPrefix(rel, "..") || strings.Contains(rel, "/../") {
		return "", false
	}
	if len(rel) >= 2 && rel[1] == ':' {
		return "", false
	}
	return rel, true
}

func ensureParentDir(filePath string) error {
	parent := filepath.Dir(filePath)
	if parent == "." || parent == "" {
		return nil
	}
	return os.MkdirAll(parent, 0755)
}

// extractZIP inspects the central directory, selects install candidates, then extracts only those entries.
// The archive is opened twice on purpose: inspect/select must finish before any bytes are written.
func extractZIP(archivePath, destDir string, opts *ExtractOptions) ([]string, error) {
	policy := resolveExtractionPolicy(opts)

	entries, err := InspectZIPWithPolicy(archivePath, policy)
	if err != nil {
		return nil, err
	}

	ctx := selectionContextFromOpts(opts)
	selected, err := SelectArchiveFontEntries(entries, ctx, policy)
	if err != nil {
		return nil, err
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("no font files selected from archive")
	}

	want := selectedPathSet(selected)

	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open ZIP file: %w", err)
	}
	defer reader.Close()

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	var extractedFiles []string
	var totalWritten int64
	total := len(selected)
	done := 0

	for _, file := range reader.File {
		if err := extractContext(opts).Err(); err != nil {
			return extractedFiles, err
		}
		if file.FileInfo().IsDir() || strings.HasSuffix(file.Name, "/") {
			continue
		}
		rel, ok := safeArchiveRelPath(file.Name)
		if !ok {
			continue // unselected / non-candidate; unsafe paths among selected already rejected
		}
		if _, ok := want[rel]; !ok {
			continue
		}

		extractedPath := filepath.Join(destDir, filepath.FromSlash(rel))
		if err := ensureParentDir(extractedPath); err != nil {
			return nil, fmt.Errorf("failed to create destination directory for %s: %w", extractedPath, err)
		}

		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s from archive: %w", file.Name, err)
		}
		n, extractErr := copyExtractedFileWithDeclaredSize(extractedPath, rc, file.Name, file.UncompressedSize64, policy, totalWritten)
		_ = rc.Close()
		if extractErr != nil {
			return nil, extractErr
		}
		totalWritten += n

		extractedFiles = append(extractedFiles, extractedPath)
		done++
		if opts != nil && opts.OnFontFileExtracted != nil {
			opts.OnFontFileExtracted(done, total)
		}
	}

	if len(extractedFiles) == 0 {
		return nil, fmt.Errorf("no font files extracted from archive")
	}
	return extractedFiles, nil
}

func selectionContextFromOpts(opts *ExtractOptions) ArchiveSelectionContext {
	if opts != nil && opts.Selection != nil {
		return *opts.Selection
	}
	return ArchiveSelectionContext{}
}

func selectedPathSet(selected []ArchiveEntry) map[string]ArchiveEntry {
	want := make(map[string]ArchiveEntry, len(selected)*2)
	for _, e := range selected {
		want[e.NormalizedPath] = e
		want[filepath.ToSlash(e.Name)] = e
	}
	return want
}

// extractTARXZ extracts a TAR.XZ archive.
// Nerd package mode uses a single decompress pass (all desktop fonts).
// Other sources inspect first, then extract only selected members (second decompress).
func extractTARXZ(archivePath, destDir string, opts *ExtractOptions) ([]string, error) {
	policy := resolveExtractionPolicy(opts)
	ctx := selectionContextFromOpts(opts)
	if isNerdPackageSource(ctx.SourcePrefix) {
		return extractCompressedTARPackageMode(archivePath, destDir, opts, policy, openTARXZStream)
	}
	entries, err := InspectTARXZWithPolicy(archivePath, policy)
	if err != nil {
		return nil, err
	}
	return extractSelectedCompressedTAR(archivePath, destDir, opts, policy, entries, openTARXZStream)
}

// extractTARGZ extracts a TAR.GZ archive (same package-mode single-pass rule as TAR.XZ).
func extractTARGZ(archivePath, destDir string, opts *ExtractOptions) ([]string, error) {
	policy := resolveExtractionPolicy(opts)
	ctx := selectionContextFromOpts(opts)
	if isNerdPackageSource(ctx.SourcePrefix) {
		return extractCompressedTARPackageMode(archivePath, destDir, opts, policy, openTARGZStream)
	}
	entries, err := InspectTARGZWithPolicy(archivePath, policy)
	if err != nil {
		return nil, err
	}
	return extractSelectedCompressedTAR(archivePath, destDir, opts, policy, entries, openTARGZStream)
}

// extractCompressedTARPackageMode decompresses once and writes every safe desktop font
// as it is seen. Used for Nerd Fonts where selection is "install the whole package".
// On failure, any files written in this call are removed so callers see a clean dest.
func extractCompressedTARPackageMode(
	archivePath, destDir string,
	opts *ExtractOptions,
	policy ExtractionPolicy,
	open tarStreamOpener,
) ([]string, error) {
	tr, closer, err := open(archivePath)
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	var extractedFiles []string
	var totalWritten int64
	entryCount := 0
	selectedCount := 0
	seenDest := make(map[string]string)
	done := 0

	cleanupWritten := func() {
		for _, p := range extractedFiles {
			_ = os.Remove(p)
		}
	}

	for {
		if err := extractContext(opts).Err(); err != nil {
			cleanupWritten()
			return nil, err
		}
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			cleanupWritten()
			return nil, fmt.Errorf("failed to read TAR header: %w", err)
		}

		entryCount++
		if policy.MaxArchiveEntries > 0 && entryCount > policy.MaxArchiveEntries {
			cleanupWritten()
			return nil, fmt.Errorf("%w: exceeded %d entries", ErrArchiveEntryCountLimit, policy.MaxArchiveEntries)
		}

		if header.Typeflag == tar.TypeDir || strings.HasSuffix(header.Name, "/") {
			continue
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}

		discardBody := func() error {
			if header.Size <= 0 {
				return nil
			}
			if _, err := io.Copy(io.Discard, tr); err != nil {
				return fmt.Errorf("failed to skip TAR member %q: %w", header.Name, err)
			}
			return nil
		}

		if !isFontFile(header.Name) {
			if err := discardBody(); err != nil {
				cleanupWritten()
				return nil, err
			}
			continue
		}

		rel, ok := safeArchiveRelPath(header.Name)
		if !ok {
			if err := discardBody(); err != nil {
				cleanupWritten()
				return nil, err
			}
			continue
		}
		if isWebfontKitArchivePath(rel) {
			if err := discardBody(); err != nil {
				cleanupWritten()
				return nil, err
			}
			continue
		}

		key := destinationCollisionKey(rel)
		if prev, exists := seenDest[key]; exists {
			cleanupWritten()
			return nil, fmt.Errorf("%w: %q and %q", ErrArchivePathCollision, prev, rel)
		}

		selectedCount++
		if policy.MaxSelectedFiles > 0 && selectedCount > policy.MaxSelectedFiles {
			cleanupWritten()
			return nil, fmt.Errorf("%w: selected %d (limit %d)", ErrArchiveSelectedFileLimit, selectedCount, policy.MaxSelectedFiles)
		}
		seenDest[key] = rel

		extractedPath := filepath.Join(destDir, filepath.FromSlash(rel))
		if err := ensureParentDir(extractedPath); err != nil {
			cleanupWritten()
			return nil, fmt.Errorf("failed to create destination directory for %s: %w", extractedPath, err)
		}

		var declared uint64
		if header.Size > 0 {
			declared = uint64(header.Size)
		}
		n, extractErr := copyExtractedFileWithDeclaredSize(extractedPath, tr, header.Name, declared, policy, totalWritten)
		if extractErr != nil {
			cleanupWritten()
			return nil, extractErr
		}
		totalWritten += n
		extractedFiles = append(extractedFiles, extractedPath)
		done++
		if opts != nil && opts.OnFontFileExtracted != nil {
			// Total unknown until stream ends; progress UIs treat -1 as indeterminate.
			opts.OnFontFileExtracted(done, -1)
		}
	}

	if len(extractedFiles) == 0 {
		return nil, fmt.Errorf("no font files selected from archive")
	}
	return extractedFiles, nil
}

func extractSelectedCompressedTAR(
	archivePath, destDir string,
	opts *ExtractOptions,
	policy ExtractionPolicy,
	entries []ArchiveEntry,
	open tarStreamOpener,
) ([]string, error) {
	ctx := selectionContextFromOpts(opts)
	selected, err := SelectArchiveFontEntries(entries, ctx, policy)
	if err != nil {
		return nil, err
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("no font files selected from archive")
	}

	want := selectedPathSet(selected)

	tr, closer, err := open(archivePath)
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	var extractedFiles []string
	var totalWritten int64
	total := len(selected)
	done := 0

	for {
		if err := extractContext(opts).Err(); err != nil {
			return extractedFiles, err
		}
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read TAR header: %w", err)
		}
		if header.Typeflag == tar.TypeDir {
			continue
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}

		rel, ok := safeArchiveRelPath(header.Name)
		if !ok {
			continue
		}
		if _, ok := want[rel]; !ok {
			if header.Size > 0 {
				if _, err := io.Copy(io.Discard, tr); err != nil {
					return nil, fmt.Errorf("failed to skip TAR member %q: %w", header.Name, err)
				}
			}
			continue
		}

		extractedPath := filepath.Join(destDir, filepath.FromSlash(rel))
		if err := ensureParentDir(extractedPath); err != nil {
			return nil, fmt.Errorf("failed to create destination directory for %s: %w", extractedPath, err)
		}

		var declared uint64
		if header.Size > 0 {
			declared = uint64(header.Size)
		}
		n, extractErr := copyExtractedFileWithDeclaredSize(extractedPath, tr, header.Name, declared, policy, totalWritten)
		if extractErr != nil {
			return nil, extractErr
		}
		totalWritten += n

		extractedFiles = append(extractedFiles, extractedPath)
		done++
		if opts != nil && opts.OnFontFileExtracted != nil {
			opts.OnFontFileExtracted(done, total)
		}
	}

	if len(extractedFiles) == 0 {
		return nil, fmt.Errorf("no font files extracted from archive")
	}
	return extractedFiles, nil
}
