package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fontget/internal/config"
	"fontget/internal/repo"
	"fontget/internal/sources"
	"fontget/internal/testutil"
)

func testCatalog(totalFonts int, ids ...string) []byte {
	fonts := make(map[string]repo.Font, len(ids))
	for _, id := range ids {
		fonts[id] = repo.Font{Name: id, Family: id, License: "OFL", Variants: []repo.FontVariant{}}
	}
	body, err := json.Marshal(repo.SourceData{
		SourceInfo: repo.SourceInfo{Name: "Test", Version: "1", TotalFonts: totalFonts},
		Fonts:      fonts,
	})
	if err != nil {
		panic(err)
	}
	return body
}

func writeSourcesManifest(t *testing.T, home string, extras map[string]config.SourceConfig) {
	t.Helper()
	now := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	src := make(map[string]config.SourceConfig)
	for name, info := range sources.DefaultSources() {
		src[name] = config.SourceConfig{
			URL:      info.URL,
			Prefix:   info.Prefix,
			Enabled:  false,
			Filename: info.Filename,
			Priority: info.Priority,
		}
	}
	for name, cfg := range extras {
		src[name] = cfg
	}
	m := config.Manifest{
		Version:        "1",
		Created:        now,
		LastUpdated:    now,
		FontGetVersion: "0.0.0",
		Sources:        src,
		CachePolicy:    config.CachePolicy{AutoUpdateDays: 7, CheckOnStartup: false},
	}
	data, err := json.MarshalIndent(&m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	cfgDir := filepath.Join(home, ".fontget")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "manifest.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
}

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	runErr := fn()
	_ = w.Close()
	os.Stdout = old
	out, readErr := io.ReadAll(r)
	_ = r.Close()
	if readErr != nil {
		t.Fatal(readErr)
	}
	return string(out), runErr
}

func TestRunSourcesUpdateVerbose_NoWipeKeepsFailedCacheAndCountsTotalFonts(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)

	okBody := testCatalog(42, "one")
	failOld := testCatalog(7, "kept")
	staleOld := testCatalog(3, "gone")

	var okHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			okHits++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(okBody)
		case "/fail":
			http.Error(w, "unavailable", http.StatusBadGateway)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	writeSourcesManifest(t, home, map[string]config.SourceConfig{
		"Alpha": {URL: srv.URL + "/ok", Prefix: "alpha", Enabled: true, Filename: "alpha.json", Priority: 10},
		"Beta":  {URL: srv.URL + "/fail", Prefix: "beta", Enabled: true, Filename: "beta.json", Priority: 11},
	})

	alphaOld := testCatalog(1, "old-alpha")
	if _, err := repo.WriteSourceCacheAtomic("Alpha", alphaOld); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.WriteSourceCacheAtomic("Beta", failOld); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.WriteSourceCacheAtomic("Stale Source", staleOld); err != nil {
		t.Fatal(err)
	}

	out, err := captureStdout(t, runSourcesUpdateVerbose)
	if err != nil {
		t.Fatalf("runSourcesUpdateVerbose: %v\n%s", err, out)
	}

	alphaPath, err := repo.SourceCachePath("Alpha")
	if err != nil {
		t.Fatal(err)
	}
	gotAlpha, err := os.ReadFile(alphaPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotAlpha, okBody) {
		t.Fatalf("successful source should be replaced with downloaded body")
	}

	gotBeta, err := os.ReadFile(mustCmdCachePath(t, "Beta"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotBeta, failOld) {
		t.Fatal("failed source must keep previous cache (no wipe-before-download)")
	}

	if _, err := os.Stat(mustCmdCachePath(t, "Stale Source")); !os.IsNotExist(err) {
		t.Fatal("cache for a source that is no longer enabled should be pruned")
	}

	// 42 from new Alpha total_fonts + 7 from Beta's previous cache.
	if !strings.Contains(out, "49") {
		t.Fatalf("expected total fonts 49 from total_fonts/download pass, got output:\n%s", out)
	}
	if strings.Contains(out, "Refreshing font data cache") {
		t.Fatal("verbose update must not start a second GetManifest refresh")
	}
	// HEAD + GET for the successful source only; no second catalog download.
	if okHits > 2 {
		t.Fatalf("ok source HTTP hits = %d, want HEAD+GET only (no second catalog fetch)", okHits)
	}
}

func mustCmdCachePath(t *testing.T, name string) string {
	t.Helper()
	p, err := repo.SourceCachePath(name)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
