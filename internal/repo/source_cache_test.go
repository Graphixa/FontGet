package repo

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"fontget/internal/testutil"
)

func catalogJSON(totalFonts int, fontIDs ...string) []byte {
	fonts := make(map[string]Font, len(fontIDs))
	for _, id := range fontIDs {
		fonts[id] = Font{Name: id, Family: id, License: "OFL", Variants: []FontVariant{}}
	}
	body, err := json.Marshal(SourceData{
		SourceInfo: SourceInfo{
			Name:       "Test",
			Version:    "1",
			TotalFonts: totalFonts,
		},
		Fonts: fonts,
	})
	if err != nil {
		panic(err)
	}
	return body
}

func TestFontCountFromCatalogJSON_UsesTotalFonts(t *testing.T) {
	body := catalogJSON(42, "one")
	n, err := FontCountFromCatalogJSON(body)
	if err != nil {
		t.Fatal(err)
	}
	if n != 42 {
		t.Fatalf("count = %d, want 42 from source_info.total_fonts (not len(fonts))", n)
	}
}

func TestFontCountFromCatalogJSON_FallbackToFontsLen(t *testing.T) {
	body := catalogJSON(0, "a", "b", "c")
	n, err := FontCountFromCatalogJSON(body)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("count = %d, want 3 from fonts map when total_fonts is 0", n)
	}
}

func TestFontCountFromCatalogJSON_RejectsInvalidJSON(t *testing.T) {
	if _, err := FontCountFromCatalogJSON([]byte("not-json")); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestWriteSourceCacheAtomic_WritesRawBodyAndReplaces(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)

	old := []byte(`{"source_info":{"total_fonts":1},"fonts":{"old":{}}}`)
	newer := []byte(`{"source_info":{"total_fonts":9},"fonts":{"n1":{},"n2":{}}}`)

	path, err := WriteSourceCacheAtomic("Alpha Source", old)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, old) {
		t.Fatalf("first write mismatch\ngot  %s\nwant %s", got, old)
	}

	if _, err := WriteSourceCacheAtomic("Alpha Source", newer); err != nil {
		t.Fatal(err)
	}
	got, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, newer) {
		t.Fatalf("replace write should store raw body, not pretty-printed JSON\ngot  %s\nwant %s", got, newer)
	}
	if bytes.Contains(got, []byte("\n  ")) {
		t.Fatal("cache file should not be MarshalIndent pretty-printed")
	}
}

func TestPersistSourceCatalog_DoesNotWipeOtherSources(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)

	otherOld := catalogJSON(2, "x", "y")
	if _, err := WriteSourceCacheAtomic("Other", otherOld); err != nil {
		t.Fatal(err)
	}
	targetOld := catalogJSON(1, "old")
	if _, err := WriteSourceCacheAtomic("Target", targetOld); err != nil {
		t.Fatal(err)
	}

	targetNew := catalogJSON(4, "n1", "n2", "n3", "n4")
	n, path, err := PersistSourceCatalog("Target", targetNew)
	if err != nil {
		t.Fatal(err)
	}
	if n != 4 {
		t.Fatalf("font count = %d, want 4", n)
	}
	got, _ := os.ReadFile(path)
	if !bytes.Equal(got, targetNew) {
		t.Fatalf("target cache not updated to new data")
	}
	otherPath, err := SourceCachePath("Other")
	if err != nil {
		t.Fatal(err)
	}
	gotOther, err := os.ReadFile(otherPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotOther, otherOld) {
		t.Fatal("unrelated source cache must be left unchanged (no wipe-before-download)")
	}
}

func TestPruneStaleSourceCaches_RemovesDisabledKeepsEnabled(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)

	if _, err := WriteSourceCacheAtomic("Keep Me", catalogJSON(1, "a")); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteSourceCacheAtomic("Drop Me", catalogJSON(1, "b")); err != nil {
		t.Fatal(err)
	}

	if err := PruneStaleSourceCaches([]string{"Keep Me"}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(mustCachePath(t, "Keep Me")); err != nil {
		t.Fatalf("enabled source cache should remain: %v", err)
	}
	if _, err := os.Stat(mustCachePath(t, "Drop Me")); !os.IsNotExist(err) {
		t.Fatal("disabled/removed source cache should be pruned")
	}
}

func TestLoadSourceDataWithCache_FailedDownloadKeepsPrevious(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)

	old := catalogJSON(7, "kept")
	if _, err := WriteSourceCacheAtomic("Target", old); err != nil {
		t.Fatal(err)
	}
	other := catalogJSON(2, "x", "y")
	if _, err := WriteSourceCacheAtomic("Other", other); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusBadGateway)
	}))
	t.Cleanup(srv.Close)

	_, err := loadSourceDataWithCache(srv.URL, "Target", nil, true)
	if err == nil {
		t.Fatal("expected download error")
	}

	got, err := os.ReadFile(mustCachePath(t, "Target"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, old) {
		t.Fatal("failed download must keep previous cache")
	}
	gotOther, err := os.ReadFile(mustCachePath(t, "Other"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotOther, other) {
		t.Fatal("failed download must not wipe other source caches")
	}
}

func TestLoadSourceDataWithCache_WritesRawDownloadedBody(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)

	newBody := catalogJSON(4, "n1", "n2", "n3", "n4")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(newBody)
	}))
	t.Cleanup(srv.Close)

	old := catalogJSON(1, "old")
	if _, err := WriteSourceCacheAtomic("Target", old); err != nil {
		t.Fatal(err)
	}

	data, err := loadSourceDataWithCache(srv.URL, "Target", nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if data.SourceInfo.TotalFonts != 4 {
		t.Fatalf("TotalFonts = %d, want 4", data.SourceInfo.TotalFonts)
	}

	got, err := os.ReadFile(mustCachePath(t, "Target"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, newBody) {
		t.Fatalf("cache should be raw download bytes\ngot  %s\nwant %s", got, newBody)
	}
}

func TestCachedSourceFontCount(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)

	if n := CachedSourceFontCount("Missing"); n != 0 {
		t.Fatalf("missing cache count = %d, want 0", n)
	}
	if _, err := WriteSourceCacheAtomic("Present", catalogJSON(11, "a")); err != nil {
		t.Fatal(err)
	}
	if n := CachedSourceFontCount("Present"); n != 11 {
		t.Fatalf("cached count = %d, want 11", n)
	}
}

func mustCachePath(t *testing.T, name string) string {
	t.Helper()
	p, err := SourceCachePath(name)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestWriteSourceCacheAtomic_CleansTempOnSuccess(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)

	if _, err := WriteSourceCacheAtomic("Alpha", catalogJSON(1, "a")); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(home, ".fontget", "sources")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			t.Fatalf("leftover temp file after successful write: %s", e.Name())
		}
	}
}
