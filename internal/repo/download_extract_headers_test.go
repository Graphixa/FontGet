package repo

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadAndExtractFont_ZipServedAsTTF_UsesHeadersAndMagic(t *testing.T) {
	// Build a small zip payload containing a .ttf file.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("Exotica-Regular.ttf")
	if err != nil {
		t.Fatalf("zip create: %v", err)
	}
	wantPayload := []byte("fake font bytes")
	if _, err := w.Write(wantPayload); err != nil {
		t.Fatalf("zip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// URL path ends with .ttf but we serve a zip archive.
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="exotica.zip"`)
		_, _ = w.Write(buf.Bytes())
	}))
	t.Cleanup(srv.Close)

	tmp := t.TempDir()
	font := &FontFile{
		Path:        "exotica.ttf", // misleading path
		DownloadURL: srv.URL + "/exotica.ttf",
	}

	paths, err := DownloadAndExtractFont(font, tmp, nil)
	if err != nil {
		t.Fatalf("DownloadAndExtractFont error: %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("got %d paths, want 1: %v", len(paths), paths)
	}
	if filepath.Ext(paths[0]) != ".ttf" {
		t.Fatalf("extracted path %q does not look like a font", paths[0])
	}

	f, err := os.Open(paths[0])
	if err != nil {
		t.Fatalf("open extracted: %v", err)
	}
	defer f.Close()
	got, _ := io.ReadAll(f)
	if !bytes.Equal(got, wantPayload) {
		t.Fatalf("extracted payload mismatch: got %q want %q", string(got), string(wantPayload))
	}
}

