package repo

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/xi2/xz"
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

// tarStreamOpener opens a compressed TAR and returns a tar.Reader plus a closer for underlying resources.
type tarStreamOpener func(archivePath string) (*tar.Reader, io.Closer, error)

func openTARXZStream(archivePath string) (*tar.Reader, io.Closer, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open TAR.XZ file: %w", err)
	}
	xzReader, err := xz.NewReader(file, 0)
	if err != nil {
		_ = file.Close()
		return nil, nil, fmt.Errorf("failed to create XZ reader: %w", err)
	}
	return tar.NewReader(xzReader), file, nil
}

func openTARGZStream(archivePath string) (*tar.Reader, io.Closer, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open TAR.GZ file: %w", err)
	}
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		_ = file.Close()
		return nil, nil, fmt.Errorf("failed to create GZIP reader: %w", err)
	}
	return tar.NewReader(gzReader), &multiCloser{gzReader, file}, nil
}

type multiCloser struct {
	a, b io.Closer
}

func (m *multiCloser) Close() error {
	errA := m.a.Close()
	errB := m.b.Close()
	if errA != nil {
		return errA
	}
	return errB
}

// InspectTARXZWithPolicy walks TAR.XZ headers without writing files.
func InspectTARXZWithPolicy(archivePath string, policy ExtractionPolicy) ([]ArchiveEntry, error) {
	return inspectCompressedTARWithPolicy(archivePath, policy, openTARXZStream)
}

// InspectTARGZWithPolicy walks TAR.GZ headers without writing files.
func InspectTARGZWithPolicy(archivePath string, policy ExtractionPolicy) ([]ArchiveEntry, error) {
	return inspectCompressedTARWithPolicy(archivePath, policy, openTARGZStream)
}

func inspectCompressedTARWithPolicy(archivePath string, policy ExtractionPolicy, open tarStreamOpener) ([]ArchiveEntry, error) {
	tr, closer, err := open(archivePath)
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	var entries []ArchiveEntry
	entryCount := 0
	for {
		header, err := tr.Next()
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

		isDir := header.Typeflag == tar.TypeDir || strings.HasSuffix(header.Name, "/")
		var size uint64
		if header.Size > 0 {
			size = uint64(header.Size)
		}
		e := ArchiveEntry{
			Name:             header.Name,
			IsDir:            isDir,
			UncompressedSize: size,
		}
		if !isDir {
			if rel, ok := safeArchiveRelPath(header.Name); ok {
				e.NormalizedPath = rel
			}
		}
		e.Extension = strings.ToLower(path.Ext(header.Name))
		classifyArchiveEntryType(&e)
		entries = append(entries, e)

		// Drain regular file bodies so the next header is reachable without writing.
		if header.Typeflag == tar.TypeReg && header.Size > 0 {
			if _, err := io.Copy(io.Discard, tr); err != nil {
				return nil, fmt.Errorf("failed to skip TAR member %q: %w", header.Name, err)
			}
		}
	}
	return entries, nil
}
