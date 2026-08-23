package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"path"
	"strings"
)

const maxExtractedBinaryBytes = 80 << 20 // 80 MiB

func extractBinary(archiveBytes []byte, archiveName string) ([]byte, error) {
	lower := strings.ToLower(archiveName)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return extractBinaryFromZip(archiveBytes)
	case strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz"):
		return extractBinaryFromTarGz(archiveBytes)
	default:
		return nil, fmt.Errorf("unsupported archive format: %s", archiveName)
	}
}

func extractBinaryFromZip(archiveBytes []byte) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(archiveBytes), int64(len(archiveBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to read zip archive: %w", err)
	}
	want := strings.ToLower(currentExecutableName())
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if !isSafeArchivePath(f.Name) {
			continue
		}
		if !isReleaseBinaryName(f.Name, want) {
			continue
		}
		if f.UncompressedSize64 > uint64(maxExtractedBinaryBytes) {
			return nil, fmt.Errorf("archive entry too large: %q", f.Name)
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open %s in archive: %w", f.Name, err)
		}
		data, err := readLimited(rc, maxExtractedBinaryBytes)
		_ = rc.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to extract %s: %w", f.Name, err)
		}
		return data, nil
	}
	return nil, fmt.Errorf("archive does not contain %s", currentExecutableName())
}

func extractBinaryFromTarGz(archiveBytes []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archiveBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to read tar.gz archive: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	want := strings.ToLower(currentExecutableName())
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar header: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if !isSafeArchivePath(hdr.Name) {
			continue
		}
		if !isReleaseBinaryName(hdr.Name, want) {
			continue
		}
		if hdr.Size < 0 || hdr.Size > maxExtractedBinaryBytes {
			return nil, fmt.Errorf("archive entry too large: %q", hdr.Name)
		}
		data, err := readLimited(tr, maxExtractedBinaryBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to extract %s: %w", hdr.Name, err)
		}
		return data, nil
	}
	return nil, fmt.Errorf("archive does not contain %s", currentExecutableName())
}

func isReleaseBinaryName(entryName, wantBase string) bool {
	base := strings.ToLower(path.Base(strings.ReplaceAll(entryName, "\\", "/")))
	return base == wantBase
}

func isSafeArchivePath(name string) bool {
	cleaned := path.Clean("/" + strings.ReplaceAll(name, "\\", "/"))
	return !strings.Contains(cleaned, "/../") && !strings.HasPrefix(name, "/") && !strings.Contains(name, "..")
}

func readLimited(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("entry exceeded %d bytes", limit)
	}
	return data, nil
}
