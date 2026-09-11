package repo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fontget/internal/config"
	"fontget/internal/sources"
	"fontget/internal/testutil"
)

const testVariantURL = "https://example.invalid/roboto.ttf"

func catalogWithVariantURLs() []byte {
	body, err := json.Marshal(SourceData{
		SourceInfo: SourceInfo{
			Name:       "Google Fonts",
			Version:    "1",
			TotalFonts: 1,
		},
		Fonts: map[string]Font{
			"roboto": {
				Name:       "Roboto",
				License:    "OFL",
				Categories: []string{"Sans Serif"},
				Variants: []FontVariant{{
					Name:  "regular",
					Files: map[string]string{"ttf": testVariantURL},
				}},
			},
		},
	})
	if err != nil {
		panic(err)
	}
	return body
}

func setupCachedGoogleHome(t *testing.T, body []byte) {
	t.Helper()
	home := t.TempDir()
	testutil.SetHome(t, home)
	InvalidateCachedManifests()
	t.Cleanup(InvalidateCachedManifests)

	now := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	src := make(map[string]config.SourceConfig)
	for name, info := range sources.DefaultSources() {
		cfg := config.SourceConfig{
			URL:      info.URL,
			Prefix:   info.Prefix,
			Enabled:  false,
			Filename: info.Filename,
			Priority: info.Priority,
		}
		if name == "Google Fonts" {
			cfg.Enabled = true
		}
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
	data, err := json.Marshal(&m)
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
	if _, err := WriteSourceCacheAtomic("Google Fonts", body); err != nil {
		t.Fatal(err)
	}
}

func TestGetCachedManifest_MemoizedSecondCallDoesNotRereadJSON(t *testing.T) {
	setupCachedGoogleHome(t, catalogWithVariantURLs())

	m1, err := GetCachedManifest()
	if err != nil {
		t.Fatalf("first load: %v", err)
	}
	info, ok := m1.Sources["Google Fonts"].Fonts["google.roboto"]
	if !ok || info.Name != "Roboto" {
		t.Fatalf("first load missing google.roboto: %+v", m1.Sources)
	}

	cachePath, err := SourceCachePath("Google Fonts")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(cachePath); err != nil {
		t.Fatal(err)
	}

	m2, err := GetCachedManifest()
	if err != nil {
		t.Fatalf("second load after deleting cache file: %v", err)
	}
	info2, ok := m2.Sources["Google Fonts"].Fonts["google.roboto"]
	if !ok || info2.Name != "Roboto" {
		t.Fatal("second GetCachedManifest should return the in-process memo, not re-read JSON")
	}
}

func TestMatchRepositoryFontByID_SecondCallUsesMemo(t *testing.T) {
	setupCachedGoogleHome(t, catalogWithVariantURLs())

	first, err := MatchRepositoryFontByID("google.roboto")
	if err != nil || first == nil || first.FontID != "google.roboto" {
		t.Fatalf("first match: match=%+v err=%v", first, err)
	}

	cachePath, err := SourceCachePath("Google Fonts")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(cachePath); err != nil {
		t.Fatal(err)
	}

	second, err := MatchRepositoryFontByID("google.roboto")
	if err != nil || second == nil || second.FontID != "google.roboto" {
		t.Fatalf("second MatchRepositoryFontByID should use memo: match=%+v err=%v", second, err)
	}
}

func TestMatchingCatalogOmitsVariantURLs_FullKeepsThem(t *testing.T) {
	setupCachedGoogleHome(t, catalogWithVariantURLs())

	slim, err := getCachedManifestMatching()
	if err != nil {
		t.Fatal(err)
	}
	slimInfo := slim.Sources["Google Fonts"].Fonts["google.roboto"]
	if len(slimInfo.Files) != 0 || len(slimInfo.VariantFiles) != 0 {
		t.Fatalf("matching catalog should omit variant URLs, files=%v variant_files=%v", slimInfo.Files, slimInfo.VariantFiles)
	}
	if slimInfo.Name != "Roboto" || slimInfo.License != "OFL" {
		t.Fatalf("matching catalog missing list fields: %+v", slimInfo)
	}

	full, err := GetCachedManifest()
	if err != nil {
		t.Fatal(err)
	}
	fullInfo := full.Sources["Google Fonts"].Fonts["google.roboto"]
	if fullInfo.Files["ttf"] != testVariantURL {
		t.Fatalf("full catalog should keep download URLs, files=%v", fullInfo.Files)
	}
	if fullInfo.VariantFiles["regular"]["ttf"] != testVariantURL {
		t.Fatalf("full catalog should keep variant URLs, variant_files=%v", fullInfo.VariantFiles)
	}

	files, err := GetFontByIDCached("google.roboto")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 || files[0].DownloadURL != testVariantURL {
		t.Fatalf("install path should still see variant URLs, files=%+v", files)
	}
}

func TestWriteSourceCacheAtomic_InvalidatesMemo(t *testing.T) {
	setupCachedGoogleHome(t, catalogWithVariantURLs())
	if _, err := GetCachedManifest(); err != nil {
		t.Fatal(err)
	}

	newer := catalogJSON(1, "open-sans")
	if _, err := WriteSourceCacheAtomic("Google Fonts", newer); err != nil {
		t.Fatal(err)
	}
	m, err := GetCachedManifest()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Sources["Google Fonts"].Fonts["google.roboto"]; ok {
		t.Fatal("memo should have been cleared after cache write")
	}
	if _, ok := m.Sources["Google Fonts"].Fonts["google.open-sans"]; !ok {
		t.Fatalf("expected rewritten catalog, fonts=%v", m.Sources["Google Fonts"].Fonts)
	}
}
