package repo

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/xi2/xz"
)

// ArchiveType represents the type of archive file
type ArchiveType int

const (
	ArchiveTypeUnknown ArchiveType = iota
	ArchiveTypeZIP
	ArchiveTypeTARXZ
	ArchiveTypeTARGZ
	ArchiveType7Z
)

// DetectArchiveType detects the archive type based on file extension
func DetectArchiveType(filename string) ArchiveType {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".zip":
		return ArchiveTypeZIP
	case ".7z":
		return ArchiveType7Z
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

	// 7Z magic: 37 7A BC AF 27 1C
	if len(b) >= 6 && b[0] == 0x37 && b[1] == 0x7A && b[2] == 0xBC && b[3] == 0xAF && b[4] == 0x27 && b[5] == 0x1C {
		return ArchiveType7Z
	}

	return ArchiveTypeUnknown
}

// ExtractArchive extracts an archive file to the specified directory.
func ExtractArchive(archivePath, destDir string) ([]string, error) {
	return ExtractArchiveWithOptions(archivePath, destDir, nil)
}

// ExtractOptions configures ExtractArchiveWithOptions.
type ExtractOptions struct {
	// OnFontFileExtracted is called after each font file is extracted.
	// total is the number of font files that will be extracted when known, otherwise -1.
	OnFontFileExtracted func(done int, total int)

	// Policy overrides default extraction limits when non-nil.
	Policy *ExtractionPolicy

	// Selection, when set, enables source-aware / agnostic selection before ZIP extraction.
	// TAR/7Z still stream-extract font candidates under the same hard budgets.
	Selection *ArchiveSelectionContext
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
	case ArchiveType7Z:
		return extract7Z(archivePath, destDir, opts)
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

	ctx := ArchiveSelectionContext{}
	if opts != nil && opts.Selection != nil {
		ctx = *opts.Selection
	}

	selected, err := SelectArchiveFontEntries(entries, ctx, policy)
	if err != nil {
		return nil, err
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("no font files selected from archive")
	}

	want := make(map[string]ArchiveEntry, len(selected))
	for _, e := range selected {
		want[e.NormalizedPath] = e
		want[filepath.ToSlash(e.Name)] = e
	}

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

// extractTARXZ extracts a TAR.XZ archive and returns the list of extracted font files.
// Selection before extract is not practical for single-pass streams; hard budgets still apply.
func extractTARXZ(archivePath, destDir string, opts *ExtractOptions) ([]string, error) {
	policy := resolveExtractionPolicy(opts)

	file, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open TAR.XZ file: %w", err)
	}
	defer file.Close()

	xzReader, err := xz.NewReader(file, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to create XZ reader: %w", err)
	}

	tarReader := tar.NewReader(xzReader)
	return extractTARStream(tarReader, destDir, opts, policy)
}

// extractTARGZ extracts a TAR.GZ archive and returns the list of extracted font files.
func extractTARGZ(archivePath, destDir string, opts *ExtractOptions) ([]string, error) {
	policy := resolveExtractionPolicy(opts)

	file, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open TAR.GZ file: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create GZIP reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	return extractTARStream(tarReader, destDir, opts, policy)
}

func extractTARStream(tarReader *tar.Reader, destDir string, opts *ExtractOptions, policy ExtractionPolicy) ([]string, error) {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	var extractedFiles []string
	var totalWritten int64
	entryCount := 0
	done := 0
	seenDest := make(map[string]string)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read TAR header: %w", err)
		}

		entryCount++
		if policy.MaxArchiveEntries > 0 && entryCount > policy.MaxArchiveEntries {
			return nil, fmt.Errorf("%w: exceeded %d entries", ErrArchiveEntryCountLimit, policy.MaxArchiveEntries)
		}

		if header.Typeflag == tar.TypeDir {
			continue
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		if !isFontFile(header.Name) {
			continue
		}

		rel, ok := safeArchiveRelPath(header.Name)
		if !ok {
			return nil, fmt.Errorf("%w: %q", ErrArchiveUnsafePath, header.Name)
		}
		key := destinationCollisionKey(rel)
		if prev, exists := seenDest[key]; exists {
			return nil, fmt.Errorf("%w: %q and %q", ErrArchivePathCollision, prev, rel)
		}
		seenDest[key] = rel

		if policy.MaxSelectedFiles > 0 && done >= policy.MaxSelectedFiles {
			return nil, fmt.Errorf("%w: selected %d (limit %d)", ErrArchiveSelectedFileLimit, done+1, policy.MaxSelectedFiles)
		}

		extractedPath := filepath.Join(destDir, filepath.FromSlash(rel))
		if err := ensureParentDir(extractedPath); err != nil {
			return nil, fmt.Errorf("failed to create destination directory for %s: %w", extractedPath, err)
		}

		var declared uint64
		if header.Size > 0 {
			declared = uint64(header.Size)
		}
		n, extractErr := copyExtractedFileWithDeclaredSize(extractedPath, tarReader, header.Name, declared, policy, totalWritten)
		if extractErr != nil {
			return nil, extractErr
		}
		totalWritten += n

		extractedFiles = append(extractedFiles, extractedPath)
		done++
		if opts != nil && opts.OnFontFileExtracted != nil {
			opts.OnFontFileExtracted(done, -1)
		}
	}

	return extractedFiles, nil
}

// extract7Z extracts a 7Z archive using an external tool (7zz/7z) and returns extracted font files.
// Full inspect-before-extract parity is not available via the CLI; hard output budgets still apply.
func extract7Z(archivePath, destDir string, opts *ExtractOptions) ([]string, error) {
	policy := resolveExtractionPolicy(opts)

	tool, err := find7zTool()
	if err != nil {
		return nil, err
	}

	tmp, err := os.MkdirTemp("", "fontget-7z-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp extraction directory: %w", err)
	}
	defer os.RemoveAll(tmp)

	cmd := exec.Command(tool, "x", "-y", "-o"+tmp, archivePath)
	out, runErr := cmd.CombinedOutput()
	if runErr != nil {
		return nil, fmt.Errorf("7z extraction failed: %w (%s)", runErr, strings.TrimSpace(string(out)))
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	var extractedFiles []string
	done := 0
	var totalWritten int64
	entryCount := 0
	seenDest := make(map[string]string)

	walkErr := filepath.WalkDir(tmp, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		entryCount++
		if policy.MaxArchiveEntries > 0 && entryCount > policy.MaxArchiveEntries {
			return fmt.Errorf("%w: exceeded %d entries", ErrArchiveEntryCountLimit, policy.MaxArchiveEntries)
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if !isFontFile(d.Name()) {
			return nil
		}

		rel, err := filepath.Rel(tmp, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		relSafe, ok := safeArchiveRelPath(rel)
		if !ok {
			return fmt.Errorf("%w: %q", ErrArchiveUnsafePath, rel)
		}
		key := destinationCollisionKey(relSafe)
		if prev, exists := seenDest[key]; exists {
			return fmt.Errorf("%w: %q and %q", ErrArchivePathCollision, prev, relSafe)
		}
		seenDest[key] = relSafe

		if policy.MaxSelectedFiles > 0 && done >= policy.MaxSelectedFiles {
			return fmt.Errorf("%w: selected %d (limit %d)", ErrArchiveSelectedFileLimit, done+1, policy.MaxSelectedFiles)
		}

		dst := filepath.Join(destDir, filepath.FromSlash(relSafe))
		if err := ensureParentDir(dst); err != nil {
			return err
		}

		info, err := os.Stat(p)
		if err != nil {
			return err
		}
		srcFile, err := os.Open(p)
		if err != nil {
			return err
		}
		var declared uint64
		if sz := info.Size(); sz > 0 {
			declared = uint64(sz)
		}
		n, err := copyExtractedFileWithDeclaredSize(dst, srcFile, relSafe, declared, policy, totalWritten)
		_ = srcFile.Close()
		if err != nil {
			return err
		}
		totalWritten += n

		extractedFiles = append(extractedFiles, dst)
		done++
		if opts != nil && opts.OnFontFileExtracted != nil {
			opts.OnFontFileExtracted(done, -1)
		}
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("failed to walk extracted 7z contents: %w", walkErr)
	}

	if len(extractedFiles) == 0 {
		return nil, fmt.Errorf("no font files found in archive")
	}
	return extractedFiles, nil
}

func find7zTool() (string, error) {
	// Prefer 7zz (p7zip), then 7z.
	if p, err := exec.LookPath("7zz"); err == nil {
		return p, nil
	}
	if p, err := exec.LookPath("7z"); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("7z archive extraction requires '7zz' or '7z' on PATH")
}
