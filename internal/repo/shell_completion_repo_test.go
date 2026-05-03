package repo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fontget/internal/config"
	"fontget/internal/sources"
)

func TestGetRepositoryForShellCompletion_WithoutManifest(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfgDir := filepath.Join(home, ".fontget")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Valid manifest with every built-in present but disabled so no cache load runs
	// and shell completion cannot build a font repository.
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
	manifestPath := filepath.Join(cfgDir, "manifest.json")
	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	_, err = GetRepositoryForShellCompletion()
	if err == nil {
		t.Fatal("GetRepositoryForShellCompletion: expected error when no sources are enabled / no cache")
	}
}
