package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"fontget/internal/platform"
)

type removeTrackingFM struct {
	dir   string
	calls []string
}

func (m *removeTrackingFM) InstallFont(string, platform.InstallationScope, bool, *platform.InstallFontOptions) error {
	return nil
}
func (m *removeTrackingFM) RemoveFont(name string, _ platform.InstallationScope, _ *platform.RemoveFontOptions) error {
	m.calls = append(m.calls, name)
	return nil
}
func (m *removeTrackingFM) GetFontDir(platform.InstallationScope) string { return m.dir }
func (m *removeTrackingFM) RequiresElevation(platform.InstallationScope) bool {
	return false
}
func (m *removeTrackingFM) IsElevated() (bool, error)                   { return true, nil }
func (m *removeTrackingFM) FlushFontCache(platform.InstallationScope) error { return nil }
func (m *removeTrackingFM) GetElevationCommand() (string, []string, error) {
	return "", nil, nil
}

func TestRemoveFontFiles_missingFilesCountAsRemoved(t *testing.T) {
	dir := t.TempDir()
	fm := &removeTrackingFM{dir: dir}
	removed, skipped, failed, _, _ := removeFontFiles(RemoveFontFilesParams{
		Ctx:           context.Background(),
		MatchingFonts: []string{"Lekton.ttf", "Lekton-Bold.ttf"},
		FontManager:   fm,
		Scope:         platform.UserScope,
		FontDir:       dir,
	})
	if removed != 2 || skipped != 0 || failed != 0 {
		t.Fatalf("removed=%d skipped=%d failed=%d", removed, skipped, failed)
	}
	if len(fm.calls) != 0 {
		t.Fatalf("RemoveFont should not run for absent files, got %v", fm.calls)
	}
}

func TestRemoveFontFiles_presentStillCallsRemove(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Lekton.ttf"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	fm := &removeTrackingFM{dir: dir}
	removed, _, failed, _, _ := removeFontFiles(RemoveFontFilesParams{
		Ctx:           context.Background(),
		MatchingFonts: []string{"Lekton.ttf", "Gone.ttf"},
		FontManager:   fm,
		Scope:         platform.UserScope,
		FontDir:       dir,
	})
	if removed != 2 || failed != 0 {
		t.Fatalf("removed=%d failed=%d", removed, failed)
	}
	// unregister + delete for present file only
	if len(fm.calls) != 2 || fm.calls[0] != "Lekton.ttf" || fm.calls[1] != "Lekton.ttf" {
		t.Fatalf("calls=%v", fm.calls)
	}
}
