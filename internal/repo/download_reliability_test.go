package repo

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fontget/internal/network"
	"fontget/internal/testutil"
)

func TestDownloadAndExtractFont_Skip404ThenSucceed(t *testing.T) {
	payload := testutil.MinimalTTF("TestFamily", "Regular")
	missingHits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/missing.ttf") {
			missingHits++
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "font/ttf")
		_, _ = w.Write(payload)
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	font := &FontFile{
		Name:               "Test",
		Variant:            "Regular",
		DownloadURL:        srv.URL + "/missing.ttf",
		DownloadCandidates: []string{srv.URL + "/missing.ttf", srv.URL + "/ok.ttf"},
	}
	paths, err := DownloadAndExtractFont(font, dir, nil)
	if err != nil {
		t.Fatalf("expected fallback candidate to succeed: %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("paths=%v", paths)
	}
	if missingHits != 1 {
		t.Fatalf("404 URL hit %d times; must not retry with external tools", missingHits)
	}
}

func TestDownloadFont_ChecksumMismatch(t *testing.T) {
	payload := testutil.MinimalTTF("TestFamily", "Regular")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	t.Cleanup(srv.Close)

	wrong := hex.EncodeToString(make([]byte, 32))
	dir := t.TempDir()
	font := &FontFile{Name: "Test", Variant: "Regular", Path: "t.ttf", DownloadURL: srv.URL + "/t.ttf", SHA: wrong}
	_, err := DownloadFont(font, dir, nil)
	if err == nil || !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("want checksum mismatch, got %v", err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatalf("rejected download must be removed, leftover %v", entries)
	}
}

func TestDownloadFont_CorrectChecksum(t *testing.T) {
	payload := testutil.MinimalTTF("TestFamily", "Regular")
	sum := sha256.Sum256(payload)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	font := &FontFile{Name: "Test", Variant: "Regular", Path: "t.ttf", DownloadURL: srv.URL + "/t.ttf", SHA: hex.EncodeToString(sum[:])}
	path, err := DownloadFont(font, dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatal("payload mismatch")
	}
}

func TestDownloadFont_MalformedChecksum(t *testing.T) {
	dir := t.TempDir()
	font := &FontFile{Name: "Test", Path: "t.ttf", DownloadURL: "https://example.com/t.ttf", SHA: "nope"}
	_, err := DownloadFont(font, dir, nil)
	if err == nil || !errors.Is(err, ErrMalformedChecksum) {
		t.Fatalf("want malformed checksum, got %v", err)
	}
}

func TestDownloadAndExtractFont_UnassociatedChecksum(t *testing.T) {
	sum := hex.EncodeToString(make([]byte, 32))
	sum = strings.ReplaceAll(sum, "00", "ab")
	if len(sum) != 64 {
		sum = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	}
	font := &FontFile{
		Name:               "Test",
		SHA:                "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		DownloadURL:        "https://example.com/a.zip",
		DownloadCandidates: []string{"https://example.com/a.zip", "https://example.com/a.tar.xz"},
	}
	_, err := DownloadAndExtractFont(font, t.TempDir(), nil)
	if err == nil || !errors.Is(err, ErrChecksumUnassociated) {
		t.Fatalf("want unassociated checksum, got %v", err)
	}
}

func TestDownloadFont_LocalWriteFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte{0x00, 0x01, 0x00, 0x00})
	}))
	t.Cleanup(srv.Close)
	blocked := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocked, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	font := &FontFile{Name: "Test", Path: "t.ttf", DownloadURL: srv.URL + "/t.ttf"}
	_, err := DownloadFont(font, blocked, nil)
	if err == nil || !errors.Is(err, network.ErrLocalFailure) {
		t.Fatalf("want local failure, got %v", err)
	}
}

func TestCompleteDownloadedFile(t *testing.T) {
	payload := testutil.MinimalTTF("HashFam", "Regular")
	sum := sha256.Sum256(payload)
	dir := t.TempDir()
	okPath := filepath.Join(dir, "ok.ttf")
	if err := os.WriteFile(okPath, payload, 0644); err != nil {
		t.Fatal(err)
	}
	if err := completeDownloadedFile(okPath, hex.EncodeToString(sum[:])); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(dir, "bad.ttf")
	if err := os.WriteFile(bad, payload, 0644); err != nil {
		t.Fatal(err)
	}
	wrong := hex.EncodeToString(make([]byte, 32))
	if err := completeDownloadedFile(bad, wrong); err == nil || !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("want mismatch, got %v", err)
	}
	if _, err := os.Stat(bad); !os.IsNotExist(err) {
		t.Fatal("rejected payload must be removed")
	}
}

func TestDownloadAndExtractFont_RateLimitDoesNotAdvanceCandidate(t *testing.T) {
	old := network.MaxRetryAfterWait
	network.MaxRetryAfterWait = time.Millisecond
	t.Cleanup(func() { network.MaxRetryAfterWait = old })

	hits := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits[r.URL.Path]++
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(srv.Close)

	font := &FontFile{
		Name:               "Rate",
		Variant:            "Regular",
		DownloadURL:        srv.URL + "/a.zip",
		DownloadCandidates: []string{srv.URL + "/a.zip", srv.URL + "/b.zip"},
	}
	_, err := DownloadAndExtractFont(font, t.TempDir(), nil)
	if err == nil || !errors.Is(err, network.ErrRateLimited) {
		t.Fatalf("want rate limited, got %v", err)
	}
	if hits["/b.zip"] != 0 {
		t.Fatalf("must not advance to next candidate on rate limit, hits=%v", hits)
	}
}

func TestDownloadFont_RetryAfterExceedsBudget(t *testing.T) {
	old := network.MaxRetryAfterWait
	network.MaxRetryAfterWait = time.Millisecond
	t.Cleanup(func() { network.MaxRetryAfterWait = old })

	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(srv.Close)

	font := &FontFile{Name: "Test", Path: "t.ttf", DownloadURL: srv.URL + "/t.ttf"}
	_, err := DownloadFont(font, t.TempDir(), nil)
	if err == nil || !errors.Is(err, network.ErrRateLimited) {
		t.Fatalf("want rate limited, got %v", err)
	}
	if hits != 1 {
		t.Fatalf("over-budget 429 must not retry, hits=%d", hits)
	}
}
