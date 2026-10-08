package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"fontget/internal/installations"
	"fontget/internal/platform"
	"fontget/internal/shared"
	"fontget/internal/testutil"
	"github.com/spf13/cobra"
)

// Only fail the named normal install; restoration uses the original path.
type failVariantFM struct {
	*copyFontManager
	failBase string
}

func (m *failVariantFM) InstallFont(path string, scope platform.InstallationScope, force bool, opts *platform.InstallFontOptions) error {
	if filepath.Base(path) == m.failBase {
		return os.ErrPermission
	}
	return m.copyFontManager.InstallFont(path, scope, force, opts)
}

type cancelRecoveryFM struct {
	*cancelAfterRemoveFM
	failRecovery       bool
	recoveryContexts   []context.Context
	recoveryContextErr error
}

func (m *cancelRecoveryFM) InstallFont(path string, scope platform.InstallationScope, force bool, opts *platform.InstallFontOptions) error {
	if opts != nil && opts.Context != nil {
		m.recoveryContexts = append(m.recoveryContexts, opts.Context)
		m.recoveryContextErr = opts.Context.Err()
	}
	if m.failRecovery {
		return os.ErrPermission
	}
	return m.copyFontManager.InstallFont(path, scope, force, opts)
}

func seedRecoveryOrigin(t *testing.T, dir, family, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	writeLocalTTF(t, path, family, "Regular")
	if err := installations.UpsertInstallation(installations.UpsertParams{
		FontID: "local." + strings.ToLower(family), Scope: "user",
		Files: []installations.InstalledFontFile{{Path: path, SFNT: installations.SFNTSnapshot{Family: family, Style: "Regular"}}},
	}); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRecoveryKeepsCompletedVariantsAndPendingFailures(t *testing.T) {
	testutil.SetHome(t, t.TempDir())
	dir := t.TempDir()
	origin := seedRecoveryOrigin(t, dir, "Mixed", "A-Regular.ttf")
	bold := filepath.Join(t.TempDir(), "B-Bold.ttf")
	italic := filepath.Join(t.TempDir(), "C-Italic.ttf")
	writeLocalTTF(t, bold, "Mixed", "Bold")
	writeLocalTTF(t, italic, "Mixed", "Italic")
	items, staging := prepareMergedStagedLocals(t, []string{origin, bold, italic}, true)
	// Explicit order makes the failure occur after both Regular and Bold commit.
	slices.SortFunc(items[0].Local.Candidates, func(a, b platform.LocalFontCandidate) int { return strings.Compare(a.Basename, b.Basename) })
	fm := &failVariantFM{copyFontManager: &copyFontManager{dir: dir}, failBase: "C-Italic.ttf"}
	result, err := installLocalFontGroup(context.Background(), *items[0].Local, fm, platform.UserScope, dir, true, localSourceName, nil, staging)
	if err == nil || result.Success != 2 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	reg, err := installations.Load()
	if err != nil {
		t.Fatal(err)
	}
	inst := reg.FindByFontID("local.mixed")
	if inst == nil || !inst.IsIncomplete() {
		t.Fatalf("expected incomplete: %+v", inst)
	}
	bases := inst.BasenamesForDir(dir)
	if len(bases) != 2 || !slices.Contains(bases, "A-Regular.ttf") || !slices.Contains(bases, "B-Bold.ttf") {
		t.Fatalf("lost completed files: %v", bases)
	}
	if !slices.Contains(inst.Remaining, "C-Italic.ttf") || len(inst.LastErrors) == 0 {
		t.Fatalf("lost pending failure: %+v", inst)
	}
	if !fm.registered["A-Regular.ttf"] || !fm.registered["B-Bold.ttf"] {
		t.Fatalf("registration lost: %v", fm.registered)
	}
	if staging.Retained() {
		t.Fatal("successful recovery should allow cleanup")
	}
}

func TestRecoveryPreservesPriorIncompleteState(t *testing.T) {
	testutil.SetHome(t, t.TempDir())
	dir := t.TempDir()
	origin := seedRecoveryOrigin(t, dir, "Prior", "Prior-Regular.ttf")
	reg, err := installations.Load()
	if err != nil {
		t.Fatal(err)
	}
	prior := reg.FindByFontID("local.prior")
	if err := installations.UpsertInstallation(installations.UpsertParams{
		FontID: prior.FontID, Scope: "user", Files: prior.FlatFiles(),
		Status: installations.StatusIncompleteInstall, Remaining: []string{"Prior-Bold.ttf"}, LastErrors: []string{"previous failure"},
	}); err != nil {
		t.Fatal(err)
	}
	items, staging := prepareMergedStagedLocals(t, []string{origin}, true)
	fm := &failNInstallsAfterRemoveFM{copyFontManager: &copyFontManager{dir: dir}, installFailsLeft: 1}
	_, err = installLocalFontGroup(context.Background(), *items[0].Local, fm, platform.UserScope, dir, true, localSourceName, nil, staging)
	if err == nil {
		t.Fatal("expected install failure")
	}
	reg, err = installations.Load()
	if err != nil {
		t.Fatal(err)
	}
	inst := reg.FindByFontID(prior.FontID)
	if inst == nil || !inst.IsIncomplete() || !slices.Contains(inst.Remaining, "Prior-Bold.ttf") || !slices.Contains(inst.LastErrors, "previous failure") {
		t.Fatalf("lost prior incomplete state: %+v", inst)
	}
}

func TestRecoveryRestoresCanonicalOriginWithoutReplacingAlias(t *testing.T) {
	testutil.SetHome(t, t.TempDir())
	dir := t.TempDir()
	origin := seedRecoveryOrigin(t, dir, "Alias", "Original.ttf")
	alias := filepath.Join(t.TempDir(), "Link.ttf")
	if err := os.Symlink(origin, alias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	payload, err := os.ReadFile(origin)
	if err != nil {
		t.Fatal(err)
	}
	items, staging := prepareMergedStagedLocals(t, []string{alias}, true)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fm := &cancelAfterRemoveFM{copyFontManager: &copyFontManager{dir: dir}, cancel: cancel}
	_, _, err = runUnifiedAddSession(ctx, items, fm, platform.UserScope, dir, true, false, true, staging)
	if !IsCancelErr(err) {
		t.Fatalf("expected cancellation: %v", err)
	}
	target, err := os.Readlink(alias)
	if err != nil || target != origin {
		t.Fatalf("alias changed: %q %v", target, err)
	}
	restored, err := os.ReadFile(origin)
	if err != nil || string(restored) != string(payload) {
		t.Fatalf("original not restored: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "Link.ttf")); !os.IsNotExist(err) {
		t.Fatalf("alias basename installed: %v", err)
	}
	reg, err := installations.Load()
	if err != nil {
		t.Fatal(err)
	}
	inst := reg.FindByFontID("local.alias")
	if inst == nil || len(inst.FlatFiles()) != 1 || inst.FlatFiles()[0].Path != platform.CanonicalPath(origin) {
		t.Fatalf("wrong registry target: %+v", inst)
	}
	if !fm.registered["Original.ttf"] {
		t.Fatal("original was not re-registered")
	}
}

func TestAddCommandRecoveryFailureSurvivesCancellation(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	writeSourcesManifest(t, home, nil)
	dir := t.TempDir()
	origin := seedRecoveryOrigin(t, dir, "Boundary", "Boundary-Regular.ttf")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fm := &cancelRecoveryFM{cancelAfterRemoveFM: &cancelAfterRemoveFM{copyFontManager: &copyFontManager{dir: dir}, cancel: cancel}, failRecovery: true}
	command := recoveryTestCommand(fm)
	command.SetArgs([]string{origin})
	out, err := captureStdout(t, func() error { return command.ExecuteContext(ctx) })
	var failure *localRecoveryError
	var displayed *shared.DisplayedError
	if !errors.As(err, &failure) || !errors.As(err, &displayed) || !errors.Is(err, context.Canceled) || !errors.Is(err, os.ErrPermission) {
		t.Fatalf("lost composite error: %v", err)
	}
	if failure.stagedRoot == "" || !strings.Contains(out, failure.stagedRoot) || !strings.Contains(out, "manual recovery required") {
		t.Fatalf("missing diagnostic: %q", out)
	}
	t.Cleanup(func() { _ = os.RemoveAll(failure.stagedRoot) })
	var retainedFont bool
	walkErr := filepath.WalkDir(failure.stagedRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".ttf" {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			retainedFont = string(data) == string(testutil.MinimalTTF("Boundary", "Regular"))
		}
		return nil
	})
	if walkErr != nil || !retainedFont {
		t.Fatalf("staged bytes lost after command cleanup: %v", walkErr)
	}
	if fm.recoveryContextErr != nil {
		t.Fatalf("recovery inherited cancellation: %v", fm.recoveryContextErr)
	}
	if len(fm.recoveryContexts) != 1 {
		t.Fatal("recovery context not supplied")
	}
	if _, ok := fm.recoveryContexts[0].Deadline(); !ok {
		t.Fatal("recovery has no deadline")
	}
	reg, err := installations.Load()
	if err != nil {
		t.Fatal(err)
	}
	inst := reg.FindByFontID("local.boundary")
	if inst == nil || !inst.IsIncomplete() || !slices.Contains(inst.Remaining, "Boundary-Regular.ttf") {
		t.Fatalf("registration failure marked complete: %+v", inst)
	}
}

func recoveryTestCommand(fm platform.FontManager) *cobra.Command {
	cmd := &cobra.Command{Use: "add", SilenceErrors: true, SilenceUsage: true, RunE: addRunE(func() (platform.FontManager, error) { return fm, nil })}
	cmd.Flags().String("scope", "user", "")
	cmd.Flags().Bool("force", true, "")
	cmd.Flags().Bool("yes", true, "")
	cmd.Flags().Bool("verbose", false, "")
	cmd.Flags().Bool("debug", true, "")
	return cmd
}

func TestAddCommandIncompleteCancelAndFailureReturnErrors(t *testing.T) {
	for _, mode := range []string{"cancel", "failure"} {
		t.Run(mode, func(t *testing.T) {
			home := t.TempDir()
			testutil.SetHome(t, home)
			writeSourcesManifest(t, home, nil)
			dir := t.TempDir()
			origin := seedRecoveryOrigin(t, dir, "Boundary", "Boundary-Regular.ttf")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var fm platform.FontManager
			if mode == "cancel" {
				fm = &cancelAfterRemoveFM{copyFontManager: &copyFontManager{dir: dir}, cancel: cancel}
			} else {
				fm = &failNInstallsAfterRemoveFM{copyFontManager: &copyFontManager{dir: dir}, installFailsLeft: 1}
			}
			command := recoveryTestCommand(fm)
			command.SetArgs([]string{origin})
			_, err := captureStdout(t, func() error { return command.ExecuteContext(ctx) })
			var displayed *shared.DisplayedError
			if !errors.As(err, &displayed) {
				t.Fatalf("expected non-zero, already-printed error: %v", err)
			}
			if mode == "cancel" && !errors.Is(err, shared.ErrOperationCancelled) {
				t.Fatalf("lost cancel: %v", err)
			}
			if mode == "failure" {
				var installErr *shared.FontInstallationError
				if !errors.As(err, &installErr) {
					t.Fatalf("lost install error: %v", err)
				}
			}
		})
	}
}

func TestPartialRecoveryRetainsStagingAndTracksRestoredFiles(t *testing.T) {
	testutil.SetHome(t, t.TempDir())
	dir := t.TempDir()
	first := seedRecoveryOrigin(t, dir, "Partial", "A-Regular.ttf")
	second := filepath.Join(dir, "B-Bold.ttf")
	writeLocalTTF(t, second, "Partial", "Bold")
	items, staging := prepareMergedStagedLocals(t, []string{first, second}, true)
	prior := snapshotInstallation("local.partial")
	overlaps := localOverlapsForFontDir(items[0].Local.Candidates, dir)
	// A non-empty directory makes restoring this one target fail on every OS.
	if err := os.Remove(first); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(first, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(first, "blocker"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(second); err != nil {
		t.Fatal(err)
	}
	fm := &copyFontManager{dir: dir}
	unlock, err := installations.LockDestination(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	recoveryErr := recoverLocalOverlaps(overlaps, fm, platform.UserScope, dir, *items[0].Local, prior, os.ErrPermission)
	unlock()
	if recoveryErr == nil {
		t.Fatal("expected partial recovery failure")
	}
	wrapped := wrapLocalRecoveryError(os.ErrPermission, staging, recoveryErr)
	if !strings.Contains(wrapped.Error(), staging.Root) {
		t.Fatalf("missing staging path: %v", wrapped)
	}
	root := staging.Root
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	if err := staging.Cleanup(); err != nil {
		t.Fatal(err)
	}
	for _, o := range overlaps {
		if _, err := os.ReadFile(o.Staged); err != nil {
			t.Fatalf("staged bytes lost: %v", err)
		}
	}
	if !fm.registered["B-Bold.ttf"] {
		t.Fatal("recovery stopped before second origin")
	}
	reg, err := installations.Load()
	if err != nil {
		t.Fatal(err)
	}
	inst := reg.FindByFontID("local.partial")
	if inst == nil || !inst.IsIncomplete() || !slices.Contains(inst.Remaining, "A-Regular.ttf") || !slices.Contains(inst.BasenamesForDir(dir), "B-Bold.ttf") || slices.Contains(inst.BasenamesForDir(dir), "A-Regular.ttf") {
		t.Fatalf("partial recovery state lost: %+v", inst)
	}
}
