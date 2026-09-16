package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fontget/internal/installations"
	"fontget/internal/platform"
	"fontget/internal/shared"
	"fontget/internal/testutil"
)

func TestApplyRemoveOutcomeTrackingFailureSetsHadError(t *testing.T) {
	status := &RemovalStatus{}
	applyRemoveOutcome(status, &RemoveResult{Success: 1, Failed: 0}, errors.New("removal tracking failed"))
	if !status.HadError {
		t.Fatal("expected HadError")
	}
	if status.Failed != 0 {
		t.Fatalf("Failed count should stay 0 for tracking-only error, got %d", status.Failed)
	}
	err := removalExitAfterSummary(status, 1, 0)
	if err == nil {
		t.Fatal("expected nonzero exit")
	}
	var displayed *shared.DisplayedError
	if !errors.As(err, &displayed) {
		t.Fatalf("want AlreadyPrinted, got %T", err)
	}
}

func TestApplyRemoveOutcomeLockFailureSetsHadError(t *testing.T) {
	status := &RemovalStatus{}
	applyRemoveOutcome(status, &RemoveResult{Failed: 0, Errors: []string{"lock"}}, errors.New("timed out waiting for lock"))
	if !status.HadError {
		t.Fatal("expected HadError")
	}
	if err := removalExitAfterSummary(status, 1, 0); err == nil {
		t.Fatal("expected nonzero exit")
	}
}

func TestRemovalExitAfterSummarySuccess(t *testing.T) {
	if err := removalExitAfterSummary(&RemovalStatus{Removed: 1}, 1, 0); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveFontLockFailureSurfacesWithoutDebug(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	a := filepath.Join(fontDir, "Alpha-Regular.ttf")
	if err := os.WriteFile(a, testutil.MinimalTTF("Alpha", "Regular"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := installations.RecordInstallation(installations.RecordParams{
		FontID: "test.lockfail",
		Scope:  "user",
		Files:  []installations.InstalledFontFile{{Path: a, SFNT: installations.SFNTSnapshot{Family: "Alpha", Style: "Regular"}}},
	}); err != nil {
		t.Fatal(err)
	}

	holdCtx, holdCancel := context.WithCancel(context.Background())
	defer holdCancel()
	unlock, err := installations.LockDestination(holdCtx, fontDir)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	fm := &removeTrackingFM{dir: fontDir}
	result, remErr := removeFont(ctx, "test.lockfail", fm, platform.UserScope, fontDir, nil, mustLoadReg(t), func(string) bool { return true }, nil)
	if remErr == nil {
		t.Fatal("expected lock failure")
	}
	status := &RemovalStatus{}
	applyRemoveOutcome(status, result, remErr)
	if err := removalExitAfterSummary(status, 1, 0); err == nil {
		t.Fatal("normal path must exit nonzero after lock failure")
	}
}

func TestRemoveFontFilesTrackingFailureSurfacesWithoutDebug(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	a := filepath.Join(fontDir, "Alpha-Regular.ttf")
	if err := os.WriteFile(a, testutil.MinimalTTF("Alpha", "Regular"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := installations.RecordInstallation(installations.RecordParams{
		FontID: "test.trackrm",
		Scope:  "user",
		Files:  []installations.InstalledFontFile{{Path: a, SFNT: installations.SFNTSnapshot{Family: "Alpha", Style: "Regular"}}},
	}); err != nil {
		t.Fatal(err)
	}

	fm := &removeThenBreakRegistryFM{removeTrackingFM: &removeTrackingFM{dir: fontDir}, home: home}
	removed, _, failed, _, _, err := removeFontFiles(RemoveFontFilesParams{
		Ctx:           context.Background(),
		MatchingFonts: []string{"Alpha-Regular.ttf"},
		FontManager:   fm,
		Scope:         platform.UserScope,
		FontDir:       fontDir,
		FontID:        "test.trackrm",
	})
	if err == nil {
		t.Fatal("expected tracking failure")
	}
	if removed != 1 || failed != 0 {
		t.Fatalf("removed=%d failed=%d", removed, failed)
	}
	if !strings.Contains(err.Error(), "tracking") {
		t.Fatalf("got %v", err)
	}
	status := &RemovalStatus{}
	applyRemoveOutcome(status, &RemoveResult{Success: removed, Failed: failed}, err)
	if err := removalExitAfterSummary(status, 1, 0); err == nil {
		t.Fatal("normal path must exit nonzero after tracking failure")
	}
}

type removeThenBreakRegistryFM struct {
	*removeTrackingFM
	home string
}

func (m *removeThenBreakRegistryFM) RemoveFont(name string, _ platform.InstallationScope, _ *platform.RemoveFontOptions) error {
	err := os.Remove(filepath.Join(m.dir, name))
	cfg := filepath.Join(m.home, ".fontget")
	_ = os.RemoveAll(cfg)
	_ = os.WriteFile(cfg, []byte("blocked"), 0644)
	return err
}

func mustLoadReg(t *testing.T) *installations.Registry {
	t.Helper()
	reg, err := installations.Load()
	if err != nil {
		t.Fatal(err)
	}
	return reg
}
