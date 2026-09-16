package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fontget/internal/installations"
	"fontget/internal/platform"
	"fontget/internal/shared"
	"fontget/internal/testutil"
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
	installed, _, _, _, _, _, _, err := installDownloadedFonts(ctx, []string{pathA, pathB}, fm, platform.UserScope, fontDir, false, nil, nil, track)
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
	if err := installations.RecordInstallation(installations.RecordParams{
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
	if !IsCancelErr(context.Canceled) || !IsCancelErr(shared.ErrOperationCancelled) {
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
	if err := installations.RecordInstallation(installations.RecordParams{
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
	installed, _, _, _, _, _, _, err := installDownloadedFonts(context.Background(), []string{pathA, pathB}, fm, platform.UserScope, fontDir, false, nil, &installTestControl{failProvenance: true}, track)
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
