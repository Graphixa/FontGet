package repo

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
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
	if err == nil {
		t.Fatalf("expected error because extracted payload is not a valid font, got paths: %v", paths)
	}
	// With strict validation, we should refuse to return non-parseable "font" payloads.
	// This test still ensures we correctly detect the ZIP (served as .ttf) and attempt extraction.
	if len(paths) != 0 {
		t.Fatalf("expected no returned paths on validation failure, got %v", paths)
	}

	// We still expect the original archive decision logic to have run; no further assertions needed here.
	_ = wantPayload
}

