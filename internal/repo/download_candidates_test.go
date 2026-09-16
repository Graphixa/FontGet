package repo

import (
	"archive/zip"
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRankDownloadCandidates(t *testing.T) {
	cases := []struct {
		name string
		in   map[string]string
		want []string // keys in order
		urls []string // expected URLs in order (optional; len 0 skips)
	}{
		{
			name: "ttf preferred over zip",
			in: map[string]string{
				"zip": "https://example.com/a.zip",
				"ttf": "https://example.com/a.ttf",
			},
			want: []string{"ttf", "zip"},
		},
		{
			name: "nerd tar_xz before zip",
			in: map[string]string{
				"zip":    "https://example.com/Iosevka.zip",
				"tar_xz": "https://example.com/Iosevka.tar.xz",
			},
			want: []string{"tar_xz", "zip"},
			urls: []string{
				"https://example.com/Iosevka.tar.xz",
				"https://example.com/Iosevka.zip",
			},
		},
		{
			name: "tar.xz alias ranks as tar_xz",
			in: map[string]string{
				"zip":    "https://example.com/a.zip",
				"tar.xz": "https://example.com/a.tar.xz",
			},
			want: []string{"tar_xz", "zip"},
		},
		{
			name: "xz alias",
			in: map[string]string{
				"xz":  "https://example.com/a.tar.xz",
				"zip": "https://example.com/a.zip",
			},
			want: []string{"tar_xz", "zip"},
		},
		{
			name: "zip only",
			in:   map[string]string{"zip": "https://example.com/a.zip"},
			want: []string{"zip"},
		},
		{
			name: "ttf only",
			in:   map[string]string{"ttf": "https://example.com/a.ttf"},
			want: []string{"ttf"},
		},
		{
			name: "duplicate urls collapsed",
			in: map[string]string{
				"tar_xz": "https://example.com/same.tar.xz",
				"tar.xz": "https://example.com/same.tar.xz",
				"zip":    "https://example.com/same.tar.xz",
			},
			want: []string{"tar_xz"},
		},
		{
			name: "unknown keys ignored",
			in: map[string]string{
				"woff2": "https://example.com/a.woff2",
				"zip":   "https://example.com/a.zip",
			},
			want: []string{"zip"},
		},
		{
			name: "empty map",
			in:   nil,
			want: nil,
		},
		{
			name: "ttf then otf then archives",
			in: map[string]string{
				"7z":     "https://example.com/a.7z",
				"zip":    "https://example.com/a.zip",
				"otf":    "https://example.com/a.otf",
				"ttf":    "https://example.com/a.ttf",
				"tar_xz": "https://example.com/a.tar.xz",
			},
			want: []string{"ttf", "otf", "tar_xz", "zip", "7z"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := rankDownloadCandidates(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("len=%d want %d: %#v", len(got), len(tc.want), got)
			}
			for i := range tc.want {
				if got[i].Key != tc.want[i] {
					t.Fatalf("key[%d]=%q want %q (full %#v)", i, got[i].Key, tc.want[i], got)
				}
			}
			if len(tc.urls) > 0 {
				for i := range tc.urls {
					if got[i].URL != tc.urls[i] {
						t.Fatalf("url[%d]=%q want %q", i, got[i].URL, tc.urls[i])
					}
				}
			}
			if pick := pickDownloadURLFromFileMap(tc.in); len(tc.want) == 0 {
				if pick != "" {
					t.Fatalf("pick=%q want empty", pick)
				}
			} else if pick != got[0].URL {
				t.Fatalf("pick=%q want first %q", pick, got[0].URL)
			}
		})
	}
}

func TestConvertFontInfoToFontFiles_nerdPrefersTarXZ(t *testing.T) {
	info := FontInfo{
		Name:     "Iosevka",
		Variants: []string{"Iosevka Regular"},
		VariantFiles: map[string]map[string]string{
			"Iosevka Regular": {
				"zip":    "https://github.com/ryanoasis/nerd-fonts/releases/download/v3.5.1/Iosevka.zip",
				"tar_xz": "https://github.com/ryanoasis/nerd-fonts/releases/download/v3.5.1/Iosevka.tar.xz",
			},
		},
	}
	files, err := convertFontInfoToFontFiles(info, "nerd.iosevka")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("got %d files", len(files))
	}
	f := files[0]
	if !strings.HasSuffix(f.DownloadURL, "Iosevka.tar.xz") {
		t.Fatalf("primary URL=%q want tar.xz", f.DownloadURL)
	}
	if len(f.DownloadCandidates) != 2 {
		t.Fatalf("candidates=%v", f.DownloadCandidates)
	}
	if !strings.HasSuffix(f.DownloadCandidates[0], ".tar.xz") || !strings.HasSuffix(f.DownloadCandidates[1], ".zip") {
		t.Fatalf("candidate order=%v", f.DownloadCandidates)
	}
}

func TestDownloadAndExtractFont_formatRetryToSecondCandidate(t *testing.T) {
	// First URL is a broken ZIP; second is a valid tiny ZIP with a font-like payload.
	var goodBuf bytes.Buffer
	zw := zip.NewWriter(&goodBuf)
	w, err := zw.Create("OkNerdFont-Regular.ttf")
	if err != nil {
		t.Fatal(err)
	}
	// Minimal TTF header so ValidateFontFile passes; metadata may still fail — use non-nerd prefix.
	payload := append([]byte{0x00, 0x01, 0x00, 0x00}, bytes.Repeat([]byte("f"), 64)...)
	if _, err := w.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		switch {
		case strings.HasSuffix(r.URL.Path, "/bad.zip"):
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("nope"))
		case strings.HasSuffix(r.URL.Path, "/good.zip"):
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(goodBuf.Bytes())
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	tmp := t.TempDir()
	font := &FontFile{
		Name:        "RetryFont",
		Variant:     "Regular",
		Path:        "bad.zip",
		DownloadURL: srv.URL + "/bad.zip",
		DownloadCandidates: []string{
			srv.URL + "/bad.zip",
			srv.URL + "/good.zip",
		},
	}

	// Non-nerd: invalid sfnt metadata is dropped; may yield "no valid font files".
	// For retry proof we only need first candidate fail and second be attempted.
	_, err = DownloadAndExtractFont(font, tmp, nil)
	// Second candidate is a zip with ttf magic but not parseable SFNT — expect validation error
	// AFTER retry (hits >= 2), not a single-candidate failure.
	if hits < 2 {
		t.Fatalf("expected retry to second URL, hits=%d err=%v", hits, err)
	}
	if err == nil {
		// If platform metadata somehow accepts our stub, success is fine.
		return
	}
	if !strings.Contains(err.Error(), "after 2 format candidates") &&
		!strings.Contains(err.Error(), "candidates exhausted") &&
		!strings.Contains(err.Error(), "no valid font files") &&
		!strings.Contains(err.Error(), "failed to extract") {
		t.Fatalf("unexpected err after retry: %v", err)
	}
	// Staging from failed attempts should not leave the bad archive behind.
	if _, statErr := os.Stat(filepath.Join(tmp, "bad.zip")); !os.IsNotExist(statErr) {
		t.Fatalf("bad.zip should be cleaned after failed attempt")
	}
}

func TestDownloadAndExtractFont_allCandidatesFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	tmp := t.TempDir()
	font := &FontFile{
		Name:        "FailFont",
		Variant:     "Regular",
		DownloadURL: srv.URL + "/a.tar.xz",
		DownloadCandidates: []string{
			srv.URL + "/a.tar.xz",
			srv.URL + "/a.zip",
		},
	}
	_, err := DownloadAndExtractFont(font, tmp, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrCandidatesExhausted) {
		t.Fatalf("want ErrCandidatesExhausted, got: %v", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "/a.tar.xz") || !strings.Contains(msg, "/a.zip") {
		t.Fatalf("want candidate outcomes, got: %v", err)
	}
	entries, _ := os.ReadDir(tmp)
	for _, e := range entries {
		if e.Name() == "extracted" {
			sub, _ := os.ReadDir(filepath.Join(tmp, "extracted"))
			if len(sub) > 0 {
				t.Fatalf("extracted staging not empty: %v", sub)
			}
		}
	}
}
