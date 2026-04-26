package repo

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectArchiveTypeFromFile_MagicBytes_AllSupported(t *testing.T) {
	dir := t.TempDir()

	// ZIP saved as .ttf
	{
		p := filepath.Join(dir, "zip-as-ttf.ttf")
		f, err := os.Create(p)
		if err != nil {
			t.Fatalf("zip create: %v", err)
		}
		zw := zip.NewWriter(f)
		w, err := zw.Create("Font-Regular.ttf")
		if err != nil {
			t.Fatalf("zip entry: %v", err)
		}
		_, _ = w.Write([]byte("test"))
		if err := zw.Close(); err != nil {
			t.Fatalf("zip close: %v", err)
		}
		if err := f.Close(); err != nil {
			t.Fatalf("zip file close: %v", err)
		}
		if got := DetectArchiveTypeFromFile(p); got != ArchiveTypeZIP {
			t.Fatalf("zip magic: got %v want %v", got, ArchiveTypeZIP)
		}
	}

	// TAR.XZ (magic bytes only; extraction validates structure)
	{
		p := filepath.Join(dir, "xz-as-ttf.ttf")
		// XZ magic: FD 37 7A 58 5A 00
		if err := os.WriteFile(p, []byte{0xFD, 0x37, 0x7A, 0x58, 0x5A, 0x00, 0, 0}, 0644); err != nil {
			t.Fatalf("xz write: %v", err)
		}
		if got := DetectArchiveTypeFromFile(p); got != ArchiveTypeTARXZ {
			t.Fatalf("xz magic: got %v want %v", got, ArchiveTypeTARXZ)
		}
	}

	// TAR.GZ (gzip magic)
	{
		p := filepath.Join(dir, "gz-as-ttf.ttf")
		// GZIP magic: 1F 8B
		if err := os.WriteFile(p, []byte{0x1F, 0x8B, 0, 0, 0, 0, 0, 0}, 0644); err != nil {
			t.Fatalf("gz write: %v", err)
		}
		if got := DetectArchiveTypeFromFile(p); got != ArchiveTypeTARGZ {
			t.Fatalf("gz magic: got %v want %v", got, ArchiveTypeTARGZ)
		}
	}

	// 7Z magic
	{
		p := filepath.Join(dir, "7z-as-ttf.ttf")
		// 7Z magic: 37 7A BC AF 27 1C
		if err := os.WriteFile(p, []byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C, 0, 0}, 0644); err != nil {
			t.Fatalf("7z write: %v", err)
		}
		if got := DetectArchiveTypeFromFile(p); got != ArchiveType7Z {
			t.Fatalf("7z magic: got %v want %v", got, ArchiveType7Z)
		}
	}
}

func TestExtractArchiveWithOptions_TarGz(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "pack.tar.gz")
	destDir := filepath.Join(dir, "out")

	// Build tar.gz with one font.
	var tarBuf bytes.Buffer
	tw := tar.NewWriter(&tarBuf)
	data := []byte("fake font bytes")
	h := &tar.Header{
		Name: "SomeFont-Regular.ttf",
		Mode: 0644,
		Size: int64(len(data)),
	}
	if err := tw.WriteHeader(h); err != nil {
		t.Fatalf("tar header: %v", err)
	}
	if _, err := tw.Write(data); err != nil {
		t.Fatalf("tar write: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar close: %v", err)
	}

	var gzBuf bytes.Buffer
	zw := gzip.NewWriter(&gzBuf)
	if _, err := zw.Write(tarBuf.Bytes()); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}

	if err := os.WriteFile(archivePath, gzBuf.Bytes(), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	paths, err := ExtractArchiveWithOptions(archivePath, destDir, nil)
	if err != nil {
		t.Fatalf("ExtractArchiveWithOptions: %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("got %d paths, want 1: %v", len(paths), paths)
	}
	if filepath.Base(paths[0]) != "SomeFont-Regular.ttf" {
		t.Fatalf("unexpected extracted file: %q", paths[0])
	}
}

