package repo

import (
	"archive/tar"
	"archive/zip"
	"fmt"
	"io"
	"os"
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

	switch archiveType {
	case ArchiveTypeZIP:
		return extractZIP(archivePath, destDir, opts)
	case ArchiveTypeTARXZ:
		return extractTARXZ(archivePath, destDir, opts)
	default:
		return nil, fmt.Errorf("unsupported archive format: %s", filepath.Ext(archivePath))
	}
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

		// Create the full path for the extracted file
		extractedPath := filepath.Join(destDir, filepath.Base(file.Name))

		// Open the file from the archive
		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s from archive: %w", file.Name, err)
		}

		// Create the destination file
		destFile, err := os.Create(extractedPath)
		if err != nil {
			rc.Close()
			return nil, fmt.Errorf("failed to create destination file %s: %w", extractedPath, err)
		}

		// Copy the file contents
		_, err = io.Copy(destFile, rc)
		rc.Close()
		destFile.Close()

		if err != nil {
			os.Remove(extractedPath) // Clean up on error
			return nil, fmt.Errorf("failed to extract file %s: %w", file.Name, err)
		}

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

		// Create the full path for the extracted file
		extractedPath := filepath.Join(destDir, filepath.Base(header.Name))

		// Create the destination file
		destFile, err := os.Create(extractedPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create destination file %s: %w", extractedPath, err)
		}

		// Copy the file contents
		_, err = io.Copy(destFile, tarReader)
		destFile.Close()

		if err != nil {
			os.Remove(extractedPath) // Clean up on error
			return nil, fmt.Errorf("failed to extract file %s: %w", header.Name, err)
		}

		extractedFiles = append(extractedFiles, extractedPath)
		done++
		if opts != nil && opts.OnFontFileExtracted != nil {
			opts.OnFontFileExtracted(done, -1) // tar streams; total unknown without a second pass
		}
	}

	return extractedFiles, nil
}
