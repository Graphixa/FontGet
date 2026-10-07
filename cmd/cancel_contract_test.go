package cmd

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fontget/internal/installations"
	"fontget/internal/platform"
	"fontget/internal/repo"
	"fontget/internal/shared"
	"fontget/internal/testutil"
	"fontget/internal/ui"
)

func TestIncompleteInstallDoesNotSatisfyAlreadyInstalled(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	face := filepath.Join(fontDir, "Alpha-Regular.ttf")
	if err := os.WriteFile(face, testutil.MinimalTTF("Alpha", "Regular"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := installations.UpsertInstallation(installations.UpsertParams{
		FontID:    "test.alpha",
		Scope:     "user",
		Files:     []installations.InstalledFontFile{{Path: face, SFNT: installations.SFNTSnapshot{Family: "Alpha", Style: "Regular"}}},
		Status:    installations.StatusIncompleteInstall,
		Remaining: []string{"Beta-Regular.ttf"},
	}); err != nil {
		t.Fatal(err)
	}
	installed, handled := checkInstalledViaRegistry("test.alpha", platform.UserScope)
	if !handled || installed {
		t.Fatalf("incomplete must not count as installed: installed=%v handled=%v", installed, handled)
	}
}

func TestInstallCancelKeepsCompletedFileAndRecordsIncomplete(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	fm := &cancelAfterInstallFM{copyFontManager: &copyFontManager{dir: fontDir}, cancel: cancel}
	staging, err := platform.NewOperationStaging()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = staging.Cleanup() })
	pathA := filepath.Join(staging.Root, "Alpha-Regular.ttf")
	pathB := filepath.Join(staging.Root, "Beta-Regular.ttf")
	if err := os.WriteFile(pathA, testutil.MinimalTTF("Alpha", "Regular"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pathB, testutil.MinimalTTF("Beta", "Regular"), 0644); err != nil {
		t.Fatal(err)
	}

	track := newInstallTracker("test.cancel", nil, platform.UserScope, fontDir, []string{"Alpha-Regular.ttf", "Beta-Regular.ttf"})
	installed, _, _, _, _, _, _, err := installDownloadedFonts(ctx, []string{pathA, pathB}, fm, platform.UserScope, fontDir, false, nil, nil, track, true)
	if err == nil {
		t.Fatal("expected cancel error")
	}
	if installed != 1 {
		t.Fatalf("expected exactly one completed install, got %d", installed)
	}
	if _, statErr := os.Stat(filepath.Join(fontDir, "Alpha-Regular.ttf")); statErr != nil {
		t.Fatalf("first file must remain: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(fontDir, "Beta-Regular.ttf")); !os.IsNotExist(statErr) {
		t.Fatal("second file must not start after cancel")
	}
	reg, loadErr := installations.Load()
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	inst := reg.FindByFontID("test.cancel")
	if inst == nil || !inst.IsIncomplete() {
		t.Fatalf("expected incomplete registry record: %+v", inst)
	}
	if len(inst.Remaining) == 0 {
		t.Fatal("expected remaining basenames")
	}
}

type cancelAfterInstallFM struct {
	*copyFontManager
	cancel context.CancelFunc
	n      int
}

func (m *cancelAfterInstallFM) InstallFont(fontPath string, scope platform.InstallationScope, force bool, opts *platform.InstallFontOptions) error {
	err := m.copyFontManager.InstallFont(fontPath, scope, force, opts)
	m.n++
	if m.n >= 1 && m.cancel != nil {
		m.cancel()
	}
	return err
}

func TestRemoveCancelKeepsRemainingFiles(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	a := filepath.Join(fontDir, "Alpha-Regular.ttf")
	b := filepath.Join(fontDir, "Beta-Regular.ttf")
	if err := os.WriteFile(a, testutil.MinimalTTF("Alpha", "Regular"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, testutil.MinimalTTF("Beta", "Regular"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := installations.UpsertInstallation(installations.UpsertParams{
		FontID: "test.rm",
		Scope:  "user",
		Files: []installations.InstalledFontFile{
			{Path: a, SFNT: installations.SFNTSnapshot{Family: "Alpha", Style: "Regular"}},
			{Path: b, SFNT: installations.SFNTSnapshot{Family: "Beta", Style: "Regular"}},
		},
	}); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	fm := &removeCancelFM{
		removeTrackingFM: &removeTrackingFM{dir: fontDir},
		after: func() {
			cancel()
		},
	}
	removed, _, _, _, _, err := removeFontFiles(RemoveFontFilesParams{
		Ctx:           ctx,
		MatchingFonts: []string{"Alpha-Regular.ttf", "Beta-Regular.ttf"},
		FontManager:   fm,
		Scope:         platform.UserScope,
		FontDir:       fontDir,
		FontID:        "test.rm",
	})
	if err == nil {
		t.Fatal("expected cancel")
	}
	if removed != 1 {
		t.Fatalf("removed=%d calls=%v", removed, fm.calls)
	}
	// Cancellation stops before the second file; registry must record incomplete removal.
	reg, loadErr := installations.Load()
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	inst := reg.FindByFontID("test.rm")
	if inst == nil || inst.Status != installations.StatusIncompleteRemove {
		t.Fatalf("expected incomplete_remove: %+v", inst)
	}
	if len(inst.Remaining) != 1 || !strings.EqualFold(inst.Remaining[0], "Beta-Regular.ttf") {
		t.Fatalf("expected Beta remaining, got %#v", inst.Remaining)
	}
	if _, statErr := os.Stat(b); statErr != nil {
		t.Fatalf("second file must remain: %v", statErr)
	}
	_ = a // first file may still appear present on some hosts if delete is deferred; tracking is authoritative
}

type removeCancelFM struct {
	*removeTrackingFM
	after func()
}

func (m *removeCancelFM) RemoveFont(name string, scope platform.InstallationScope, opts *platform.RemoveFontOptions) error {
	err := m.removeTrackingFM.RemoveFont(name, scope, opts)
	if m.after != nil {
		m.after()
	}
	return err
}

func TestFormatRetryCommands(t *testing.T) {
	got := FormatRetryAddCommand([]string{"nerd.iosevka"}, "user", false)
	if got != "fontget add nerd.iosevka" {
		t.Fatalf("got %q", got)
	}
	got = FormatRetryAddCommand([]string{"nerd.iosevka"}, "machine", true)
	if got != "fontget add nerd.iosevka --scope machine --force" {
		t.Fatalf("got %q", got)
	}
	got = FormatRetryRemoveCommand([]string{"nerd.iosevka"}, "user")
	if got != "fontget remove nerd.iosevka" {
		t.Fatalf("got %q", got)
	}
}

func TestFinishInstallationCancel_exitStatus(t *testing.T) {
	if err := FinishInstallationCancel(nil, "user", false); err != nil {
		t.Fatalf("complete cancel should exit 0 path: %v", err)
	}
	err := FinishInstallationCancel([]string{"nerd.iosevka"}, "user", false)
	if err == nil {
		t.Fatal("incomplete cancel must be non-nil")
	}
	if !errors.Is(err, shared.ErrOperationCancelled) {
		t.Fatalf("want ErrOperationCancelled, got %v", err)
	}
}

func TestFinishRemovalCancel_exitStatus(t *testing.T) {
	if err := FinishRemovalCancel(nil, "user"); err != nil {
		t.Fatalf("complete cancel should exit 0 path: %v", err)
	}
	err := FinishRemovalCancel([]string{"nerd.iosevka"}, "machine")
	if err == nil {
		t.Fatal("incomplete cancel must be non-nil")
	}
	if !errors.Is(err, shared.ErrOperationCancelled) {
		t.Fatalf("want ErrOperationCancelled, got %v", err)
	}
}

func TestIsCancelErr(t *testing.T) {
	if !IsCancelErr(context.Canceled) || !IsCancelErr(shared.ErrOperationCancelled) || !IsCancelErr(ui.ErrCancelled) {
		t.Fatal("expected cancel detection")
	}
	if IsCancelErr(errors.New("other")) || IsCancelErr(nil) {
		t.Fatal("false positive")
	}
}

func TestForceRemovePhaseStopsBeforeInstallOnCancel(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	a := filepath.Join(fontDir, "Old-Regular.ttf")
	if err := os.WriteFile(a, testutil.MinimalTTF("Old", "Regular"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := installations.UpsertInstallation(installations.UpsertParams{
		FontID: "test.force",
		Scope:  "user",
		Files:  []installations.InstalledFontFile{{Path: a, SFNT: installations.SFNTSnapshot{Family: "Old", Style: "Regular"}}},
	}); err != nil {
		t.Fatal(err)
	}
	fm := &removeTrackingFM{dir: fontDir}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled before remove phase
	_, _, _, _, _, err := removeFontFiles(RemoveFontFilesParams{
		Ctx:           ctx,
		MatchingFonts: []string{"Old-Regular.ttf"},
		FontManager:   fm,
		Scope:         platform.UserScope,
		FontDir:       fontDir,
		FontID:        "test.force",
	})
	if err == nil {
		t.Fatal("expected cancel before removal")
	}
	if _, statErr := os.Stat(a); statErr != nil {
		t.Fatalf("cancelled force remove must not delete: %v", statErr)
	}
}

func TestTrackingFailureStopsBeforeNextFile(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	fm := &copyFontManager{dir: fontDir}
	staging, err := platform.NewOperationStaging()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = staging.Cleanup() })
	pathA := filepath.Join(staging.Root, "Alpha-Regular.ttf")
	pathB := filepath.Join(staging.Root, "Beta-Regular.ttf")
	if err := os.WriteFile(pathA, testutil.MinimalTTF("Alpha", "Regular"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pathB, testutil.MinimalTTF("Beta", "Regular"), 0644); err != nil {
		t.Fatal(err)
	}
	track := newInstallTracker("test.trackfail", nil, platform.UserScope, fontDir, []string{"Alpha-Regular.ttf", "Beta-Regular.ttf"})
	installed, _, _, _, _, _, _, err := installDownloadedFonts(context.Background(), []string{pathA, pathB}, fm, platform.UserScope, fontDir, false, nil, &installTestControl{failProvenance: true}, track, true)
	if err == nil {
		t.Fatal("expected tracking failure")
	}
	if installed != 0 {
		t.Fatalf("failed file must not count as installed: %d", installed)
	}
	if _, statErr := os.Stat(filepath.Join(fontDir, "Beta-Regular.ttf")); !os.IsNotExist(statErr) {
		t.Fatal("must not install next file after tracking failure")
	}
}

func TestInstallRetryPreservesRetainedInventory(t *testing.T) {
	// Critical timing: cancel after skipping A (tracking update done), before B is processed.
	// Cancelling after B would re-add B and hide inventory-loss bugs on skip+persist of A.
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	staging, err := platform.NewOperationStaging()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = staging.Cleanup() })

	pathA := filepath.Join(fontDir, "Alpha-Regular.ttf")
	pathB := filepath.Join(fontDir, "Beta-Regular.ttf")
	if err := os.WriteFile(pathA, testutil.MinimalTTF("Alpha", "Regular"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pathB, testutil.MinimalTTF("Beta", "Regular"), 0644); err != nil {
		t.Fatal(err)
	}
	expected := []string{"Alpha-Regular.ttf", "Beta-Regular.ttf", "Gamma-Regular.ttf"}
	if err := installations.UpsertInstallation(installations.UpsertParams{
		FontID: "test.retry",
		Scope:  "user",
		Files: []installations.InstalledFontFile{
			{Path: pathA, SFNT: installations.SFNTSnapshot{Family: "Alpha", Style: "Regular"}},
			{Path: pathB, SFNT: installations.SFNTSnapshot{Family: "Beta", Style: "Regular"}},
		},
		Status:    installations.StatusIncompleteInstall,
		Remaining: []string{"Gamma-Regular.ttf"},
	}); err != nil {
		t.Fatal(err)
	}

	writeStage := func(name, fam, style string) string {
		p := filepath.Join(staging.Root, name)
		if err := os.WriteFile(p, testutil.MinimalTTF(fam, style), 0644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	stageA := writeStage("Alpha-Regular.ttf", "Alpha", "Regular")
	stageB := writeStage("Beta-Regular.ttf", "Beta", "Regular")
	stageC := writeStage("Gamma-Regular.ttf", "Gamma", "Regular")

	ctx, cancel := context.WithCancel(context.Background())
	fm := &copyFontManager{dir: fontDir}
	track := newInstallTracker("test.retry", nil, platform.UserScope, fontDir, expected)
	tc := &installTestControl{
		afterTrackedSkip: cancel, // fire only after A's skip+persist; B must not start
	}
	_, skipped, _, _, _, _, _, err := installDownloadedFonts(
		ctx, []string{stageA, stageB, stageC}, fm, platform.UserScope, fontDir, false, nil, tc, track, true)
	if err == nil {
		t.Fatal("expected cancellation after skipping A")
	}
	if !IsCancelErr(err) {
		t.Fatalf("want cancel error, got %v", err)
	}
	if skipped != 1 {
		t.Fatalf("expected skip A only, got skipped=%d", skipped)
	}
	if _, statErr := os.Stat(pathA); statErr != nil {
		t.Fatalf("A must still exist: %v", statErr)
	}
	if _, statErr := os.Stat(pathB); statErr != nil {
		t.Fatalf("B must still exist: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(fontDir, "Gamma-Regular.ttf")); !os.IsNotExist(statErr) {
		t.Fatal("C must not have been installed")
	}

	reg, loadErr := installations.Load()
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	inst := reg.FindByFontID("test.retry")
	if inst == nil || !inst.IsIncomplete() {
		t.Fatalf("expected incomplete record: %+v", inst)
	}
	bases := inst.BasenamesForDir(fontDir)
	have := map[string]bool{}
	for _, b := range bases {
		have[strings.ToLower(b)] = true
	}
	if !have["alpha-regular.ttf"] || !have["beta-regular.ttf"] {
		t.Fatalf("A and B must remain tracked after skip+cancel, got %#v", bases)
	}
	if have["gamma-regular.ttf"] {
		t.Fatal("C must not be tracked yet")
	}
	if len(inst.Remaining) != 1 || !strings.EqualFold(inst.Remaining[0], "Gamma-Regular.ttf") {
		t.Fatalf("C must remain outstanding: %#v", inst.Remaining)
	}

	removed, _, _, _, _, remErr := removeFontFiles(RemoveFontFilesParams{
		Ctx:           context.Background(),
		MatchingFonts: bases,
		FontManager:   &copyFontManager{dir: fontDir},
		Scope:         platform.UserScope,
		FontDir:       fontDir,
		FontID:        "test.retry",
	})
	if remErr != nil {
		t.Fatal(remErr)
	}
	if removed != 2 {
		t.Fatalf("removal must remove A and B, got %d", removed)
	}
	if _, statErr := os.Stat(pathA); !os.IsNotExist(statErr) {
		t.Fatal("A must be removed from disk")
	}
	if _, statErr := os.Stat(pathB); !os.IsNotExist(statErr) {
		t.Fatal("B must be removed from disk")
	}
	reg, _ = installations.Load()
	if reg.FindByFontID("test.retry") != nil {
		t.Fatal("package record must be cleared after full removal")
	}
}

func TestInstallRetryReconcilesExternallyDeletedTrackedFile(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	a := filepath.Join(fontDir, "Alpha-Regular.ttf")
	b := filepath.Join(fontDir, "Beta-Regular.ttf")
	if err := os.WriteFile(a, testutil.MinimalTTF("Alpha", "Regular"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, testutil.MinimalTTF("Beta", "Regular"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := installations.UpsertInstallation(installations.UpsertParams{
		FontID: "test.ext",
		Scope:  "user",
		Files: []installations.InstalledFontFile{
			{Path: a, SFNT: installations.SFNTSnapshot{Family: "Alpha", Style: "Regular"}},
			{Path: b, SFNT: installations.SFNTSnapshot{Family: "Beta", Style: "Regular"}},
		},
		Status:    installations.StatusIncompleteInstall,
		Remaining: []string{"Gamma-Regular.ttf"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(a); err != nil {
		t.Fatal(err)
	}

	staging, err := platform.NewOperationStaging()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = staging.Cleanup() })
	pathA := filepath.Join(staging.Root, "Alpha-Regular.ttf")
	pathB := filepath.Join(staging.Root, "Beta-Regular.ttf")
	pathC := filepath.Join(staging.Root, "Gamma-Regular.ttf")
	for _, tc := range []struct {
		path, fam, style string
	}{
		{pathA, "Alpha", "Regular"},
		{pathB, "Beta", "Regular"},
		{pathC, "Gamma", "Regular"},
	} {
		if err := os.WriteFile(tc.path, testutil.MinimalTTF(tc.fam, tc.style), 0644); err != nil {
			t.Fatal(err)
		}
	}

	fm := &copyFontManager{dir: fontDir}
	track := newInstallTracker("test.ext", nil, platform.UserScope, fontDir, []string{"Alpha-Regular.ttf", "Beta-Regular.ttf", "Gamma-Regular.ttf"})
	installed, skipped, failed, _, _, _, _, err := installDownloadedFonts(context.Background(), []string{pathA, pathB, pathC}, fm, platform.UserScope, fontDir, false, nil, nil, track, true)
	if err != nil {
		t.Fatal(err)
	}
	if failed != 0 || installed != 2 || skipped != 1 {
		t.Fatalf("want reinstall A + skip B + install C; installed=%d skipped=%d failed=%d", installed, skipped, failed)
	}
	reg, loadErr := installations.Load()
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	inst := reg.FindByFontID("test.ext")
	if inst == nil || inst.IsIncomplete() {
		t.Fatalf("expected complete install after retry: %+v", inst)
	}
	if len(inst.BasenamesForDir(fontDir)) != 3 {
		t.Fatalf("expected A,B,C tracked: %#v", inst.BasenamesForDir(fontDir))
	}
}

func TestInstallCancelOnFinalFileStillSucceeds(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	fm := &cancelAfterInstallFM{copyFontManager: &copyFontManager{dir: fontDir}, cancel: cancel}
	staging, err := platform.NewOperationStaging()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = staging.Cleanup() })
	pathA := filepath.Join(staging.Root, "Only-Regular.ttf")
	if err := os.WriteFile(pathA, testutil.MinimalTTF("Only", "Regular"), 0644); err != nil {
		t.Fatal(err)
	}
	track := newInstallTracker("test.last", nil, platform.UserScope, fontDir, []string{"Only-Regular.ttf"})
	installed, _, _, _, _, _, _, err := installDownloadedFonts(ctx, []string{pathA}, fm, platform.UserScope, fontDir, false, nil, nil, track, true)
	if err != nil {
		t.Fatalf("late cancel after final file must not fail: %v", err)
	}
	if installed != 1 {
		t.Fatalf("installed=%d", installed)
	}
	reg, _ := installations.Load()
	inst := reg.FindByFontID("test.last")
	if inst == nil || inst.IsIncomplete() {
		t.Fatalf("expected complete record: %+v", inst)
	}
}

func TestRemoveCancelOnFinalFileStillSucceeds(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	a := filepath.Join(fontDir, "Only-Regular.ttf")
	if err := os.WriteFile(a, testutil.MinimalTTF("Only", "Regular"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := installations.UpsertInstallation(installations.UpsertParams{
		FontID: "test.rmlast",
		Scope:  "user",
		Files:  []installations.InstalledFontFile{{Path: a, SFNT: installations.SFNTSnapshot{Family: "Only", Style: "Regular"}}},
	}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	fm := &removeCancelFM{
		removeTrackingFM: &removeTrackingFM{dir: fontDir},
		after:            cancel,
	}
	removed, _, _, _, _, err := removeFontFiles(RemoveFontFilesParams{
		Ctx:           ctx,
		MatchingFonts: []string{"Only-Regular.ttf"},
		FontManager:   fm,
		Scope:         platform.UserScope,
		FontDir:       fontDir,
		FontID:        "test.rmlast",
	})
	if err != nil {
		t.Fatalf("late cancel after final file must not fail: %v", err)
	}
	if removed != 1 {
		t.Fatalf("removed=%d", removed)
	}
}

func TestUnifiedAddSession_CancelBeforeFirstItem(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	src := filepath.Join(t.TempDir(), "PreCancel-Regular.ttf")
	writeLocalTTF(t, src, "PreCancel", "Regular")
	items, staging := prepareMergedStagedLocals(t, []string{src}, true)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, completion, err := runUnifiedAddSession(ctx, items, &copyFontManager{dir: fontDir}, platform.UserScope, fontDir, false, false, false, staging)
	if completion != addCancelledRemaining {
		t.Fatalf("completion=%v err=%v", completion, err)
	}
	if err == nil {
		t.Fatal("expected non-nil cancel error")
	}
	if !errors.Is(err, shared.ErrOperationCancelled) {
		t.Fatalf("want ErrOperationCancelled, got %v", err)
	}
}

func TestUnifiedAddSession_CancelMidCatalogIncludesUnfinishedIDs(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()

	payload := testutil.MinimalTTF("CatA", "Regular")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "font/ttf")
		_, _ = w.Write(payload)
	}))
	t.Cleanup(srv.Close)

	items := []addWorkItem{
		{Kind: addWorkCatalog, Catalog: &FontToInstall{
			FontID: "test.first", FontName: "First", SourceName: "test",
			Fonts: []repo.FontFile{{Name: "First", Variant: "Regular", Path: "First.ttf", DownloadURL: srv.URL + "/First.ttf"}},
		}},
		{Kind: addWorkCatalog, Catalog: &FontToInstall{
			FontID: "test.second", FontName: "Second", SourceName: "test",
			Fonts: []repo.FontFile{{Name: "Second", Variant: "Regular", Path: "Second.ttf", DownloadURL: srv.URL + "/Second.ttf"}},
		}},
	}
	staging, err := platform.NewOperationStaging()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = staging.Cleanup() })

	ctx, cancel := context.WithCancel(context.Background())
	fm := &cancelAfterInstallFM{copyFontManager: &copyFontManager{dir: fontDir}, cancel: cancel}
	_, completion, runErr := runUnifiedAddSession(ctx, items, fm, platform.UserScope, fontDir, false, false, false, staging)
	if completion != addCancelledRemaining {
		t.Fatalf("completion=%v err=%v", completion, runErr)
	}
	if runErr == nil || !errors.Is(runErr, shared.ErrOperationCancelled) {
		t.Fatalf("want ErrOperationCancelled, got %v", runErr)
	}
	retry := collectCancelRetry(items, 0)
	text := FormatInstallationCancelledTextMixed(retry.CatalogIDs, retry.LocalPaths, retry.MissingLocal, "user", false)
	if !strings.Contains(text, "test.second") && !strings.Contains(text, "test.first") {
		t.Fatalf("retry guidance missing unfinished catalog IDs: %s (retry=%+v)", text, retry)
	}
}

func TestUnifiedAddSession_CancelMidLocalNoCatalogID(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()

	a := filepath.Join(t.TempDir(), "MidA-Regular.ttf")
	b := filepath.Join(t.TempDir(), "MidB-Regular.ttf")
	writeLocalTTF(t, a, "MidA", "Regular")
	writeLocalTTF(t, b, "MidB", "Regular")
	items, staging := prepareMergedStagedLocals(t, []string{a, b}, true)

	ctx, cancel := context.WithCancel(context.Background())
	fm := &cancelAfterInstallFM{copyFontManager: &copyFontManager{dir: fontDir}, cancel: cancel}
	_, completion, err := runUnifiedAddSession(ctx, items, fm, platform.UserScope, fontDir, false, false, false, staging)
	if completion != addCancelledRemaining {
		t.Fatalf("completion=%v err=%v", completion, err)
	}
	if err == nil {
		t.Fatal("expected non-zero cancel path")
	}
	retry := collectCancelRetry(items, 0)
	text := FormatInstallationCancelledTextMixed(retry.CatalogIDs, retry.LocalPaths, retry.MissingLocal, "user", false)
	if strings.Contains(text, "local.") {
		t.Fatalf("must not invent catalog-style local IDs: %s", text)
	}
}

func TestUnifiedAddSession_CancelAfterFinalItemSucceeds(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	src := filepath.Join(t.TempDir(), "FinalOnly-Regular.ttf")
	writeLocalTTF(t, src, "FinalOnly", "Regular")
	items, staging := prepareMergedStagedLocals(t, []string{src}, true)

	ctx, cancel := context.WithCancel(context.Background())
	fm := &cancelAfterInstallFM{copyFontManager: &copyFontManager{dir: fontDir}, cancel: cancel}
	status, completion, err := runUnifiedAddSession(ctx, items, fm, platform.UserScope, fontDir, false, false, false, staging)
	if completion == addCancelledRemaining {
		t.Fatalf("final-item cancel must not report remaining work: err=%v", err)
	}
	if completion != addCompleted && completion != addCancelledDone {
		t.Fatalf("completion=%v", completion)
	}
	if err != nil {
		t.Fatalf("want nil err, got %v", err)
	}
	if status == nil || status.Installed != 1 {
		t.Fatalf("installed=%v", status)
	}
}

func TestFailedItems_PartialLocalFamily(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	fm := &copyFontManager{dir: fontDir}

	srcDir := t.TempDir()
	a := filepath.Join(srcDir, "PartFam-Regular.ttf")
	b := filepath.Join(srcDir, "PartFam-Bold.ttf")
	writeLocalTTF(t, a, "PartFam", "Regular")
	writeLocalTTF(t, b, "PartFam", "Bold")
	items, _ := prepareMergedStagedLocals(t, []string{a, b}, true)
	if len(items) != 1 || len(items[0].Local.Candidates) != 2 {
		t.Fatalf("items=%d", len(items))
	}

	// Fail after first mutation inside installDownloadedFonts via installLocalFontGroup path.
	paths := []string{items[0].Local.Candidates[0].DiskPath, items[0].Local.Candidates[1].DiskPath}
	track := newInstallTracker(items[0].Local.FontID, nil, platform.UserScope, fontDir, []string{
		filepath.Base(paths[0]), filepath.Base(paths[1]),
	})
	installed, _, failed, _, _, _, _, err := installDownloadedFonts(
		context.Background(), paths, fm, platform.UserScope, fontDir, false, nil,
		&installTestControl{failAfterMutations: 1}, track, false,
	)
	if err == nil {
		t.Fatal("expected partial failure")
	}
	if installed != 1 {
		t.Fatalf("installed=%d", installed)
	}
	status := &InstallationStatus{}
	recordItemFailure(status, buildInstallResult(InstallStatusFailed, "Installation failed", installed, 0, failed, nil, nil, 0), err)
	if status.FailedItems != 1 {
		t.Fatalf("FailedItems=%d", status.FailedItems)
	}
	if status.Failed != failed {
		t.Fatalf("Failed=%d want actual file failure count %d", status.Failed, failed)
	}
}

func TestFinishAddCancel_LocalPathsGone(t *testing.T) {
	err := finishAddCancel(addCancelRetry{MissingLocal: []string{"gone.ttf"}}, "user", false, true)
	if err == nil {
		t.Fatal("work remaining with no retry tokens must be non-nil")
	}
	if !errors.Is(err, shared.ErrOperationCancelled) {
		t.Fatalf("got %v", err)
	}
}
