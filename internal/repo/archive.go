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

const (
	// Safety limits for archive extraction.
	// Fonts are typically small; these are generous to avoid false positives while preventing bombs.
	maxExtractFileBytes  int64 = 200 << 20 // 200 MiB per file
	maxExtractTotalBytes int64 = 1 << 30   // 1 GiB total
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
	// Archive formats typically use forward slashes regardless of OS.
	rel := path.Clean("/" + strings.TrimSpace(name))
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" || rel == "." {
		return "", false
	}
	// Reject traversal and absolute paths.
	if strings.HasPrefix(rel, "..") || strings.Contains(rel, "/../") {
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

// extractZIP extracts a ZIP archive and returns the list of extracted font files
func extractZIP(archivePath, destDir string, opts *ExtractOptions) ([]string, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open ZIP file: %w", err)
	}
	defer reader.Close()

	var extractedFiles []string

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	var totalWritten int64

	total := 0
	for _, f := range reader.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if !isFontFile(f.Name) {
			continue
		}
		total++
	}
	if total == 0 {
		total = -1
	}

	done := 0
	for _, file := range reader.File {
		// Skip directories
		if file.FileInfo().IsDir() {
			continue
		}

		// Check if it's a font file
		if !isFontFile(file.Name) {
			continue
		}

		rel, ok := safeArchiveRelPath(file.Name)
		if !ok {
			return nil, fmt.Errorf("unsafe zip entry path: %q", file.Name)
		}
		extractedPath := filepath.Join(destDir, filepath.FromSlash(rel))
		if err := ensureParentDir(extractedPath); err != nil {
			return nil, fmt.Errorf("failed to create destination directory for %s: %w", extractedPath, err)
		}

		if file.UncompressedSize64 > uint64(maxExtractFileBytes) {
			return nil, fmt.Errorf("zip entry too large: %q (%d bytes)", file.Name, file.UncompressedSize64)
		}
		if totalWritten > maxExtractTotalBytes {
			return nil, fmt.Errorf("archive extraction exceeded total limit (%d bytes)", maxExtractTotalBytes)
		}

		// Open the file from the archive
		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s from archive: %w", file.Name, err)
		}

		// Create the destination file (no silent overwrite)
		destFile, err := os.OpenFile(extractedPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			rc.Close()
			return nil, fmt.Errorf("failed to create destination file %s: %w", extractedPath, err)
		}

		// Copy the file contents
		limit := maxExtractFileBytes + 1
		n, copyErr := io.Copy(destFile, io.LimitReader(rc, limit))
		rcCloseErr := rc.Close()
		dstCloseErr := destFile.Close()
		extractErr := copyErr
		if extractErr == nil && rcCloseErr != nil {
			extractErr = rcCloseErr
		}
		if extractErr == nil && dstCloseErr != nil {
			extractErr = dstCloseErr
		}

		if extractErr != nil {
			_ = os.Remove(extractedPath) // best-effort cleanup
			return nil, fmt.Errorf("failed to extract file %s: %w", file.Name, extractErr)
		}
		if n > maxExtractFileBytes {
			_ = os.Remove(extractedPath)
			return nil, fmt.Errorf("zip entry exceeded per-file limit: %q", file.Name)
		}
		totalWritten += n

		extractedFiles = append(extractedFiles, extractedPath)
		done++
		if opts != nil && opts.OnFontFileExtracted != nil {
			opts.OnFontFileExtracted(done, total)
		}
	}
	return extractedFiles, nil
}

// extractTARXZ extracts a TAR.XZ archive and returns the list of extracted font files
func extractTARXZ(archivePath, destDir string, opts *ExtractOptions) ([]string, error) {
	// Open the archive file
	file, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open TAR.XZ file: %w", err)
	}
	defer file.Close()

	// Create XZ reader
	xzReader, err := xz.NewReader(file, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to create XZ reader: %w", err)
	}

	// Create TAR reader
	tarReader := tar.NewReader(xzReader)

	var extractedFiles []string

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	var totalWritten int64
	done := 0
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read TAR header: %w", err)
		}

		// Skip directories
		if header.Typeflag == tar.TypeDir {
			continue
		}

		// Only process regular files
		if header.Typeflag != tar.TypeReg {
			continue
		}

		// Check if it's a font file
		if !isFontFile(header.Name) {
			continue
		}

		rel, ok := safeArchiveRelPath(header.Name)
		if !ok {
			return nil, fmt.Errorf("unsafe tar entry path: %q", header.Name)
		}
		extractedPath := filepath.Join(destDir, filepath.FromSlash(rel))
		if err := ensureParentDir(extractedPath); err != nil {
			return nil, fmt.Errorf("failed to create destination directory for %s: %w", extractedPath, err)
		}
		if header.Size < 0 || header.Size > maxExtractFileBytes {
			return nil, fmt.Errorf("tar entry too large: %q (%d bytes)", header.Name, header.Size)
		}
		if totalWritten > maxExtractTotalBytes {
			return nil, fmt.Errorf("archive extraction exceeded total limit (%d bytes)", maxExtractTotalBytes)
		}

		// Create the destination file
		destFile, err := os.OpenFile(extractedPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to create destination file %s: %w", extractedPath, err)
		}

		// Copy the file contents
		limit := maxExtractFileBytes + 1
		n, copyErr := io.Copy(destFile, io.LimitReader(tarReader, limit))
		dstCloseErr := destFile.Close()
		extractErr := copyErr
		if extractErr == nil && dstCloseErr != nil {
			extractErr = dstCloseErr
		}

		if extractErr != nil {
			_ = os.Remove(extractedPath) // best-effort cleanup
			return nil, fmt.Errorf("failed to extract file %s: %w", header.Name, extractErr)
		}
		if n > maxExtractFileBytes {
			_ = os.Remove(extractedPath)
			return nil, fmt.Errorf("tar entry exceeded per-file limit: %q", header.Name)
		}
		totalWritten += n

		extractedFiles = append(extractedFiles, extractedPath)
		done++
		if opts != nil && opts.OnFontFileExtracted != nil {
			opts.OnFontFileExtracted(done, -1) // tar streams; total unknown without a second pass
		}
	}

	return extractedFiles, nil
}

// extractTARGZ extracts a TAR.GZ archive and returns the list of extracted font files
func extractTARGZ(archivePath, destDir string, opts *ExtractOptions) ([]string, error) {
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

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	var extractedFiles []string
	var totalWritten int64
	done := 0
	for {
		header, err := tarReader.Next()
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
		if !isFontFile(header.Name) {
			continue
		}

		rel, ok := safeArchiveRelPath(header.Name)
		if !ok {
			return nil, fmt.Errorf("unsafe tar entry path: %q", header.Name)
		}
		extractedPath := filepath.Join(destDir, filepath.FromSlash(rel))
		if err := ensureParentDir(extractedPath); err != nil {
			return nil, fmt.Errorf("failed to create destination directory for %s: %w", extractedPath, err)
		}
		if header.Size < 0 || header.Size > maxExtractFileBytes {
			return nil, fmt.Errorf("tar entry too large: %q (%d bytes)", header.Name, header.Size)
		}
		if totalWritten > maxExtractTotalBytes {
			return nil, fmt.Errorf("archive extraction exceeded total limit (%d bytes)", maxExtractTotalBytes)
		}

		destFile, err := os.OpenFile(extractedPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to create destination file %s: %w", extractedPath, err)
		}
		limit := maxExtractFileBytes + 1
		n, copyErr := io.Copy(destFile, io.LimitReader(tarReader, limit))
		dstCloseErr := destFile.Close()
		extractErr := copyErr
		if extractErr == nil && dstCloseErr != nil {
			extractErr = dstCloseErr
		}
		if extractErr != nil {
			_ = os.Remove(extractedPath)
			return nil, fmt.Errorf("failed to extract file %s: %w", header.Name, extractErr)
		}
		if n > maxExtractFileBytes {
			_ = os.Remove(extractedPath)
			return nil, fmt.Errorf("tar entry exceeded per-file limit: %q", header.Name)
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
func extract7Z(archivePath, destDir string, opts *ExtractOptions) ([]string, error) {
	tool, err := find7zTool()
	if err != nil {
		return nil, err
	}

	// Extract into a temp dir first so we can validate paths before copying into destDir.
	tmp, err := os.MkdirTemp("", "fontget-7z-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp extraction directory: %w", err)
	}
	defer os.RemoveAll(tmp)

	// 7z x -y -o<dir> <archive>
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
	walkErr := filepath.WalkDir(tmp, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		// Skip symlinks for safety.
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
			return fmt.Errorf("unsafe 7z extracted path: %q", rel)
		}

		dst := filepath.Join(destDir, filepath.FromSlash(relSafe))
		if err := ensureParentDir(dst); err != nil {
			return err
		}

		srcFile, err := os.Open(p)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			_ = srcFile.Close()
			return err
		}

		limit := maxExtractFileBytes + 1
		n, err := io.Copy(dstFile, io.LimitReader(srcFile, limit))
		_ = dstFile.Close()
		if err != nil {
			_ = os.Remove(dst)
			return err
		}
		if n > maxExtractFileBytes {
			_ = os.Remove(dst)
			return fmt.Errorf("7z entry exceeded per-file limit: %q", relSafe)
		}
		totalWritten += n
		if totalWritten > maxExtractTotalBytes {
			return fmt.Errorf("archive extraction exceeded total limit (%d bytes)", maxExtractTotalBytes)
		}

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
