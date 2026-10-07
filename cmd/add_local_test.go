package cmd

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"fontget/internal/installations"
	"fontget/internal/platform"
	"fontget/internal/shared"
	"fontget/internal/testutil"
)

func writeLocalTTF(t *testing.T, path, family, style string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, testutil.MinimalTTF(family, style), 0o644); err != nil {
		t.Fatal(err)
	}
}

// installLocalPath prepares and installs a single local path in its own progress session.
func installLocalPath(
	ctx context.Context,
	path string,
	fontManager platform.FontManager,
	installScope platform.InstallationScope,
	fontDir string,
	force bool,
	yes bool,
	verbose bool,
	debug bool,
) (*localInstallOutcome, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	prep, err := prepareLocalInstall(path, path, installScope, yes)
	if err != nil {
		return &localInstallOutcome{Dupes: prepDupes(prep), Conflicts: prepConflicts(prep)}, err
	}
	out := &localInstallOutcome{
		Dupes:     prep.Dupes,
		Conflicts: prep.Conflicts,
		Groups:    len(prep.Groups),
	}

	items := make([]addWorkItem, 0, len(prep.Groups))
	for i := range prep.Groups {
		g := prep.Groups[i]
		items = append(items, addWorkItem{Kind: addWorkLocal, Local: &g})
	}
	items, err = mergeLocalWorkItems(items)
	if err != nil {
		return out, err
	}
	staging, stErr := platform.NewOperationStaging()
	if stErr != nil {
		return out, stErr
	}
	defer func() { _ = staging.Cleanup() }()
	if err := stageLocalWorkItems(items, staging); err != nil {
		return out, err
	}
	status, _, runErr := runUnifiedAddSession(ctx, items, fontManager, installScope, fontDir, force, verbose, debug, staging)
	if status != nil {
		out.Installed = status.Installed
		out.Skipped = status.Skipped
		out.Failed = status.Failed
		out.Errors = status.Errors
		out.Groups = len(items)
	}
	return out, runErr
}

func prepDupes(p *localPrepareResult) int {
	if p == nil {
		return 0
	}
	return p.Dupes
}

func prepConflicts(p *localPrepareResult) int {
	if p == nil {
		return 0
	}
	return p.Conflicts
}

type localInstallOutcome struct {
	Installed int
	Skipped   int
	Failed    int
	Dupes     int
	Conflicts int
	Errors    []string
	Groups    int
}

func writeLocalZip(t *testing.T, zipPath, comment string, files map[string][]byte) {
	t.Helper()
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	if comment != "" {
		if err := zw.SetComment(comment); err != nil {
			t.Fatal(err)
		}
	}
	for name, data := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestClassifyAddArg(t *testing.T) {
	dir := t.TempDir()
	font := filepath.Join(dir, "A.ttf")
	writeLocalTTF(t, font, "A", "Regular")
	zipPath := filepath.Join(dir, "b.zip")
	writeLocalZip(t, zipPath, "", map[string][]byte{"x.ttf": testutil.MinimalTTF("X", "Regular")})

	if got := classifyAddArg(font); got.Kind != addArgLocal {
		t.Fatalf("font file: %+v", got)
	}
	if got := classifyAddArg(dir); got.Kind != addArgLocal {
		t.Fatalf("dir: %+v", got)
	}
	if got := classifyAddArg(zipPath); got.Kind != addArgLocal {
		t.Fatalf("zip: %+v", got)
	}
	if got := classifyAddArg("google.roboto"); got.Kind != addArgCatalog {
		t.Fatalf("catalog: %+v", got)
	}
}

func TestInstallLocalPath_SingleFilePreservesOriginal(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	fm := &copyFontManager{dir: fontDir}

	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "Solo-Regular.ttf")
	writeLocalTTF(t, src, "Solo", "Regular")

	out, err := installLocalPath(context.Background(), src, fm, platform.UserScope, fontDir, false, true, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if out.Installed != 1 {
		t.Fatalf("installed=%d", out.Installed)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("original must remain: %v", err)
	}
	if _, err := os.Stat(filepath.Join(fontDir, "Solo-Regular.ttf")); err != nil {
		t.Fatalf("dest missing: %v", err)
	}
	reg, err := installations.Load()
	if err != nil {
		t.Fatal(err)
	}
	inst := reg.FindByFontID("local.solo")
	if inst == nil {
		t.Fatal("expected local.solo registry entry")
	}
	if inst.InstallationSource != "local" {
		t.Fatalf("source=%q", inst.InstallationSource)
	}
}

func TestInstallLocalPath_FolderDedupeLooseOverZip(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	fm := &copyFontManager{dir: fontDir}

	root := t.TempDir()
	data := testutil.MinimalTTF("DupFam", "Regular")
	loose := filepath.Join(root, "DupFam", "Dup-Regular.ttf")
	if err := os.MkdirAll(filepath.Dir(loose), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(loose, data, 0o644); err != nil {
		t.Fatal(err)
	}
	writeLocalZip(t, filepath.Join(root, "DupFam", "pack.zip"), "", map[string][]byte{"Dup-Regular.ttf": data})

	out, err := installLocalPath(context.Background(), root, fm, platform.UserScope, fontDir, false, true, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if out.Installed != 1 {
		t.Fatalf("installed=%d dupes=%d", out.Installed, out.Dupes)
	}
	if out.Dupes < 1 {
		t.Fatalf("expected dedupe, got %d", out.Dupes)
	}
	if _, err := os.Stat(loose); err != nil {
		t.Fatalf("loose original must remain: %v", err)
	}
}

func TestInstallLocalPath_BasenameConflict(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	fm := &copyFontManager{dir: fontDir}

	root := t.TempDir()
	writeLocalTTF(t, filepath.Join(root, "a", "Regular.ttf"), "FamilyOne", "Regular")
	writeLocalTTF(t, filepath.Join(root, "b", "Regular.ttf"), "FamilyTwo", "Regular")

	out, err := installLocalPath(context.Background(), root, fm, platform.UserScope, fontDir, false, true, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if out.Installed != 1 || out.Conflicts != 1 {
		t.Fatalf("installed=%d conflicts=%d", out.Installed, out.Conflicts)
	}
	entries, err := os.ReadDir(fontDir)
	if err != nil {
		t.Fatal(err)
	}
	var fonts []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		fonts = append(fonts, e.Name())
	}
	if len(fonts) != 1 {
		t.Fatalf("font files=%v installed=%d conflicts=%d", fonts, out.Installed, out.Conflicts)
	}
}

func TestInstallLocalPath_BackupZip(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	fm := &copyFontManager{dir: fontDir}

	dir := t.TempDir()
	zipPath := filepath.Join(dir, "backup.zip")
	data := testutil.MinimalTTF("BackupFam", "Regular")
	writeLocalZip(t, zipPath, shared.BackupZipComment, map[string][]byte{
		"Other/BackupFam/BackupFam-Regular.ttf": data,
	})

	out, err := installLocalPath(context.Background(), zipPath, fm, platform.UserScope, fontDir, false, true, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if out.Installed != 1 {
		t.Fatalf("installed=%d", out.Installed)
	}
	reg, err := installations.Load()
	if err != nil {
		t.Fatal(err)
	}
	inst := reg.FindByFontID("local.backupfam")
	if inst == nil {
		t.Fatal("missing registry")
	}
	if inst.InstallationSource != "fontget-backup" {
		t.Fatalf("source=%q", inst.InstallationSource)
	}
}

func TestInstallLocalPath_SkipAlreadyInstalledPreservesOriginal(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	fm := &copyFontManager{dir: fontDir}

	src := filepath.Join(t.TempDir(), "Keep-Regular.ttf")
	writeLocalTTF(t, src, "Keep", "Regular")
	// Pre-install destination with same basename
	if err := os.WriteFile(filepath.Join(fontDir, "Keep-Regular.ttf"), testutil.MinimalTTF("Keep", "Regular"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := installLocalPath(context.Background(), src, fm, platform.UserScope, fontDir, false, true, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if out.Skipped != 1 || out.Installed != 0 {
		t.Fatalf("out=%+v", out)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("original deleted on skip: %v", err)
	}
}

func TestInstallLocalPath_LargeBatchRequiresYes(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	fm := &copyFontManager{dir: fontDir}

	root := t.TempDir()
	for i := 0; i < localInstallConfirmThreshold; i++ {
		name := filepath.Join(root, "Font"+strconv.Itoa(i)+"-Regular.ttf")
		writeLocalTTF(t, name, "Fam"+strconv.Itoa(i), "Regular")
	}

	_, err := installLocalPath(context.Background(), root, fm, platform.UserScope, fontDir, false, false, false, false)
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("expected --yes requirement, got %v", err)
	}

	out, err := installLocalPath(context.Background(), root, fm, platform.UserScope, fontDir, false, true, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if out.Installed < localInstallConfirmThreshold {
		t.Fatalf("installed=%d", out.Installed)
	}
}

func TestRunUnifiedAddSession_MixedLocalItemsOneBar(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	fm := &copyFontManager{dir: fontDir}

	a := filepath.Join(t.TempDir(), "Alpha-Regular.ttf")
	b := filepath.Join(t.TempDir(), "Beta-Regular.ttf")
	writeLocalTTF(t, a, "Alpha", "Regular")
	writeLocalTTF(t, b, "Beta", "Regular")

	prepA, err := prepareLocalInstall(a, a, platform.UserScope, true)
	if err != nil {
		t.Fatal(err)
	}
	prepB, err := prepareLocalInstall(b, b, platform.UserScope, true)
	if err != nil {
		t.Fatal(err)
	}
	gA := prepA.Groups[0]
	gB := prepB.Groups[0]
	items := []addWorkItem{
		{Kind: addWorkLocal, Local: &gA},
		{Kind: addWorkLocal, Local: &gB},
	}
	items, err = mergeLocalWorkItems(items)
	if err != nil {
		t.Fatal(err)
	}
	staging, err := platform.NewOperationStaging()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = staging.Cleanup() })
	if err := stageLocalWorkItems(items, staging); err != nil {
		t.Fatal(err)
	}

	status, _, err := runUnifiedAddSession(context.Background(), items, fm, platform.UserScope, fontDir, false, false, false, staging)
	if err != nil {
		t.Fatal(err)
	}
	if status.Installed != 2 {
		t.Fatalf("installed=%d status=%+v", status.Installed, status)
	}
	if _, err := os.Stat(filepath.Join(fontDir, "Alpha-Regular.ttf")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(fontDir, "Beta-Regular.ttf")); err != nil {
		t.Fatal(err)
	}
}

func TestInstallDownloadedFonts_DeleteSourcesFalse(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	fm := &copyFontManager{dir: fontDir}
	src := filepath.Join(t.TempDir(), "NoDelete-Regular.ttf")
	writeLocalTTF(t, src, "NoDelete", "Regular")

	installed, _, _, _, _, _, _, err := installDownloadedFonts(
		context.Background(), []string{src}, fm, platform.UserScope, fontDir, false, nil, nil, nil, false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if installed != 1 {
		t.Fatalf("installed=%d", installed)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("source removed despite deleteSources=false: %v", err)
	}
}

func prepareMergedStagedLocals(t *testing.T, paths []string, forceYes bool) ([]addWorkItem, *platform.OperationStaging) {
	t.Helper()
	var items []addWorkItem
	for _, p := range paths {
		prep, err := prepareLocalInstall(p, p, platform.UserScope, forceYes)
		if err != nil {
			t.Fatal(err)
		}
		for i := range prep.Groups {
			g := prep.Groups[i]
			items = append(items, addWorkItem{Kind: addWorkLocal, Local: &g})
		}
	}
	items, err := mergeLocalWorkItems(items)
	if err != nil {
		t.Fatal(err)
	}
	staging, err := platform.NewOperationStaging()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = staging.Cleanup() })
	if err := stageLocalWorkItems(items, staging); err != nil {
		t.Fatal(err)
	}
	return items, staging
}

func TestMergeLocalForce_SeparateArgsBothRemain(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	fm := &copyFontManager{dir: fontDir}

	srcDir := t.TempDir()
	regPath := filepath.Join(srcDir, "MergeFam-Regular.ttf")
	boldPath := filepath.Join(srcDir, "MergeFam-Bold.ttf")
	writeLocalTTF(t, regPath, "MergeFam", "Regular")
	writeLocalTTF(t, boldPath, "MergeFam", "Bold")

	items, staging := prepareMergedStagedLocals(t, []string{regPath, boldPath}, true)
	if len(items) != 1 {
		t.Fatalf("want 1 merged family item, got %d", len(items))
	}
	if len(items[0].Local.Candidates) != 2 {
		t.Fatalf("want 2 candidates, got %d", len(items[0].Local.Candidates))
	}

	status, _, err := runUnifiedAddSession(context.Background(), items, fm, platform.UserScope, fontDir, true, false, false, staging)
	if err != nil {
		t.Fatal(err)
	}
	if status.Installed != 2 {
		t.Fatalf("installed=%d status=%+v", status.Installed, status)
	}
	if _, err := os.Stat(filepath.Join(fontDir, "MergeFam-Regular.ttf")); err != nil {
		t.Fatalf("regular missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(fontDir, "MergeFam-Bold.ttf")); err != nil {
		t.Fatalf("bold missing: %v", err)
	}
}

func TestMergeLocalForce_ExistingTrackedRemovedOnce(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	fm := &recordingFontManager{copyFontManager: &copyFontManager{dir: fontDir}}

	oldPath := filepath.Join(fontDir, "MergeFam-Old.ttf")
	writeLocalTTF(t, oldPath, "MergeFam", "Light")
	if err := installations.UpsertInstallation(installations.UpsertParams{
		FontID: "local.mergefam",
		Scope:  "user",
		Files: []installations.InstalledFontFile{
			{Path: oldPath, SFNT: installations.SFNTSnapshot{Family: "MergeFam", Style: "Light"}},
		},
	}); err != nil {
		t.Fatal(err)
	}

	srcDir := t.TempDir()
	regPath := filepath.Join(srcDir, "MergeFam-Regular.ttf")
	boldPath := filepath.Join(srcDir, "MergeFam-Bold.ttf")
	writeLocalTTF(t, regPath, "MergeFam", "Regular")
	writeLocalTTF(t, boldPath, "MergeFam", "Bold")

	items, staging := prepareMergedStagedLocals(t, []string{regPath, boldPath}, true)
	status, _, err := runUnifiedAddSession(context.Background(), items, fm, platform.UserScope, fontDir, true, false, false, staging)
	if err != nil {
		t.Fatal(err)
	}
	if status.Installed != 2 {
		t.Fatalf("installed=%d", status.Installed)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatal("old tracked file must be removed")
	}
	removeOps := 0
	for _, op := range fm.ops {
		if strings.HasPrefix(op, "remove:") {
			removeOps++
		}
	}
	if removeOps != 1 {
		t.Fatalf("want one force-remove, got %d ops=%v", removeOps, fm.ops)
	}
	if _, err := os.Stat(filepath.Join(fontDir, "MergeFam-Regular.ttf")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(fontDir, "MergeFam-Bold.ttf")); err != nil {
		t.Fatal(err)
	}
}

func TestMergeLocal_SameFileTwiceInstalledOnce(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	fm := &copyFontManager{dir: fontDir}

	src := filepath.Join(t.TempDir(), "Once-Regular.ttf")
	writeLocalTTF(t, src, "Once", "Regular")
	items, staging := prepareMergedStagedLocals(t, []string{src, src}, true)
	if len(items) != 1 || len(items[0].Local.Candidates) != 1 {
		t.Fatalf("want 1 item / 1 candidate, got items=%d cands=%d", len(items), len(items[0].Local.Candidates))
	}
	status, _, err := runUnifiedAddSession(context.Background(), items, fm, platform.UserScope, fontDir, false, false, false, staging)
	if err != nil {
		t.Fatal(err)
	}
	if status.Installed != 1 {
		t.Fatalf("installed=%d", status.Installed)
	}
}

func TestStageLocal_ForceFromFontDirectory(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	fm := &copyFontManager{dir: fontDir}

	src := filepath.Join(fontDir, "InPlace-Regular.ttf")
	payload := testutil.MinimalTTF("InPlace", "Regular")
	if err := os.WriteFile(src, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := installations.UpsertInstallation(installations.UpsertParams{
		FontID: "local.inplace",
		Scope:  "user",
		Files: []installations.InstalledFontFile{
			{Path: src, SFNT: installations.SFNTSnapshot{Family: "InPlace", Style: "Regular"}},
		},
	}); err != nil {
		t.Fatal(err)
	}

	items, staging := prepareMergedStagedLocals(t, []string{src}, true)
	status, _, err := runUnifiedAddSession(context.Background(), items, fm, platform.UserScope, fontDir, true, false, false, staging)
	if err != nil {
		t.Fatal(err)
	}
	if status.Installed != 1 {
		t.Fatalf("installed=%d", status.Installed)
	}
	got, err := os.ReadFile(filepath.Join(fontDir, "InPlace-Regular.ttf"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatal("installed bytes must match staged original")
	}
}

func TestStageLocal_CleanupOnFailure(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()

	outside := filepath.Join(t.TempDir(), "Outside-Regular.ttf")
	writeLocalTTF(t, outside, "Outside", "Regular")
	outsideBytes, err := os.ReadFile(outside)
	if err != nil {
		t.Fatal(err)
	}

	items, staging := prepareMergedStagedLocals(t, []string{outside}, true)
	root := staging.Root
	if _, err := os.Stat(root); err != nil {
		t.Fatal(err)
	}
	// Inject install failure by using a font manager that cannot place files.
	fm := &failInstallFM{dir: fontDir}
	status, _, runErr := runUnifiedAddSession(context.Background(), items, fm, platform.UserScope, fontDir, false, false, false, staging)
	if runErr == nil && (status == nil || status.FailedItems == 0) {
		t.Fatal("expected install failure")
	}
	_ = staging.Cleanup()
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("staging must be cleaned: %v", err)
	}
	got, err := os.ReadFile(outside)
	if err != nil {
		t.Fatalf("outside source must remain: %v", err)
	}
	if string(got) != string(outsideBytes) {
		t.Fatal("outside source modified")
	}
}

func TestStageLocal_CancelAfterStagingPreservesInput(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()

	src := filepath.Join(t.TempDir(), "CancelStage-Regular.ttf")
	writeLocalTTF(t, src, "CancelStage", "Regular")
	items, staging := prepareMergedStagedLocals(t, []string{src}, true)
	root := staging.Root

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before install loop
	fm := &copyFontManager{dir: fontDir}
	_, completion, err := runUnifiedAddSession(ctx, items, fm, platform.UserScope, fontDir, false, false, false, staging)
	if completion != addCancelledRemaining {
		t.Fatalf("completion=%v err=%v", completion, err)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("input deleted: %v", err)
	}
	_ = staging.Cleanup()
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("staging must be cleaned: %v", err)
	}
}

type failInstallFM struct {
	dir string
}

func (m *failInstallFM) FlushFontCache(scope platform.InstallationScope) error { return nil }
func (m *failInstallFM) InstallFont(fontPath string, scope platform.InstallationScope, force bool, opts *platform.InstallFontOptions) error {
	return os.ErrPermission
}
func (m *failInstallFM) RemoveFont(fontName string, scope platform.InstallationScope, opts *platform.RemoveFontOptions) error {
	return nil
}
func (m *failInstallFM) GetFontDir(scope platform.InstallationScope) string { return m.dir }
func (m *failInstallFM) RequiresElevation(scope platform.InstallationScope) bool {
	return false
}
func (m *failInstallFM) IsElevated() (bool, error) { return true, nil }
func (m *failInstallFM) GetElevationCommand() (string, []string, error) {
	return "", nil, nil
}

func TestFailedItems_LockFailure(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	fm := &copyFontManager{dir: fontDir}

	src := filepath.Join(t.TempDir(), "LockFail-Regular.ttf")
	writeLocalTTF(t, src, "LockFail", "Regular")
	items, staging := prepareMergedStagedLocals(t, []string{src}, true)

	// Hold destination lock so installLocalFontGroup fails before placing files.
	unlock, err := installations.LockDestination(context.Background(), fontDir)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	// Short deadline: LockDestination returns context error (not cancel-of-operation path).
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	status, completion, runErr := runUnifiedAddSession(ctx, items, fm, platform.UserScope, fontDir, false, false, false, staging)
	if completion == addCancelledRemaining {
		t.Fatalf("lock timeout must count as item failure, not cancel-remaining: %v", runErr)
	}
	if status == nil || status.FailedItems < 1 {
		t.Fatalf("want FailedItems>=1, status=%+v err=%v", status, runErr)
	}
	if len(status.Errors) == 0 {
		t.Fatal("expected error text preserved")
	}
}

func TestClassifyAddArg_CollectionIsLocal(t *testing.T) {
	dir := t.TempDir()
	ttc := filepath.Join(dir, "x.ttc")
	if err := os.WriteFile(ttc, []byte("ttcf"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := classifyAddArg(ttc); got.Kind != addArgLocal {
		t.Fatalf("ttc should classify as local: %+v", got)
	}
}

func TestStageLocal_BadZipEntryFails(t *testing.T) {
	staging, err := platform.NewOperationStaging()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = staging.Cleanup() })
	_, err = platform.StageLocalCandidate(staging, platform.LocalFontCandidate{
		ZipPath:  filepath.Join(t.TempDir(), "missing.zip"),
		ZipEntry: "gone.ttf",
		Basename: "gone.ttf",
		FromZip:  true,
	})
	if err == nil {
		t.Fatal("expected zip stage failure")
	}
}

func TestFailedItems_CatalogZeroFileFailure(t *testing.T) {
	status := &InstallationStatus{}
	recordItemFailure(status, buildInstallResult(InstallStatusFailed, "Download failed", 0, 0, 0, nil, nil, 0), os.ErrNotExist)
	if status.FailedItems != 1 {
		t.Fatalf("FailedItems=%d", status.FailedItems)
	}
	if len(status.Errors) == 0 {
		t.Fatal("expected preserved error")
	}
}

type failNInstallsAfterRemoveFM struct {
	*copyFontManager
	removed         bool
	installFailsLeft int
}

func (m *failNInstallsAfterRemoveFM) RemoveFont(name string, scope platform.InstallationScope, opts *platform.RemoveFontOptions) error {
	m.removed = true
	return m.copyFontManager.RemoveFont(name, scope, opts)
}

func (m *failNInstallsAfterRemoveFM) InstallFont(fontPath string, scope platform.InstallationScope, force bool, opts *platform.InstallFontOptions) error {
	if m.removed && m.installFailsLeft > 0 {
		m.installFailsLeft--
		return os.ErrPermission
	}
	return m.copyFontManager.InstallFont(fontPath, scope, force, opts)
}

type cancelAfterRemoveFM struct {
	*copyFontManager
	cancel context.CancelFunc
}

func (m *cancelAfterRemoveFM) RemoveFont(name string, scope platform.InstallationScope, opts *platform.RemoveFontOptions) error {
	err := m.copyFontManager.RemoveFont(name, scope, opts)
	if m.cancel != nil {
		m.cancel()
	}
	return err
}

func TestOverlapRecover_ForceFailRestoresOrigin(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	src := filepath.Join(fontDir, "Recover-Regular.ttf")
	payload := testutil.MinimalTTF("Recover", "Regular")
	if err := os.WriteFile(src, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := installations.UpsertInstallation(installations.UpsertParams{
		FontID: "local.recover",
		Scope:  "user",
		Files: []installations.InstalledFontFile{
			{Path: src, SFNT: installations.SFNTSnapshot{Family: "Recover", Style: "Regular"}},
		},
	}); err != nil {
		t.Fatal(err)
	}

	items, staging := prepareMergedStagedLocals(t, []string{src}, true)
	fm := &failNInstallsAfterRemoveFM{copyFontManager: &copyFontManager{dir: fontDir}, installFailsLeft: 1}
	status, _, err := runUnifiedAddSession(context.Background(), items, fm, platform.UserScope, fontDir, true, false, false, staging)
	if err == nil && (status == nil || status.FailedItems == 0) {
		t.Fatal("expected failure")
	}
	got, readErr := os.ReadFile(src)
	if readErr != nil {
		t.Fatalf("origin must be restored: %v", readErr)
	}
	if string(got) != string(payload) {
		t.Fatal("restored bytes mismatch")
	}
	root := staging.Root
	_ = staging.Cleanup()
	if staging.Retained() {
		t.Fatal("successful recovery must not retain staging")
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("staging must be cleaned after successful recovery: %v", err)
	}
	reg, _ := installations.Load()
	if inst := reg.FindByFontID("local.recover"); inst == nil || !inst.IsComplete() {
		t.Fatalf("registry must be restored: %+v", inst)
	}
}

func TestOverlapRecover_CancelAfterRemoveRestores(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	src := filepath.Join(fontDir, "CancelRec-Regular.ttf")
	payload := testutil.MinimalTTF("CancelRec", "Regular")
	if err := os.WriteFile(src, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := installations.UpsertInstallation(installations.UpsertParams{
		FontID: "local.cancelrec",
		Scope:  "user",
		Files: []installations.InstalledFontFile{
			{Path: src, SFNT: installations.SFNTSnapshot{Family: "CancelRec", Style: "Regular"}},
		},
	}); err != nil {
		t.Fatal(err)
	}

	items, staging := prepareMergedStagedLocals(t, []string{src}, true)
	ctx, cancel := context.WithCancel(context.Background())
	fm := &cancelAfterRemoveFM{copyFontManager: &copyFontManager{dir: fontDir}, cancel: cancel}
	_, completion, err := runUnifiedAddSession(ctx, items, fm, platform.UserScope, fontDir, true, false, false, staging)
	if completion != addCancelledRemaining {
		t.Fatalf("completion=%v err=%v", completion, err)
	}
	if err == nil {
		t.Fatal("incomplete cancel must be non-nil")
	}
	got, readErr := os.ReadFile(src)
	if readErr != nil {
		t.Fatalf("origin must be restored: %v", readErr)
	}
	if string(got) != string(payload) {
		t.Fatal("restored bytes mismatch")
	}
}

func TestOverlapRecover_FailureRetainsStaging(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	src := filepath.Join(fontDir, "Retain-Regular.ttf")
	payload := testutil.MinimalTTF("Retain", "Regular")
	if err := os.WriteFile(src, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := installations.UpsertInstallation(installations.UpsertParams{
		FontID: "local.retain",
		Scope:  "user",
		Files: []installations.InstalledFontFile{
			{Path: src, SFNT: installations.SFNTSnapshot{Family: "Retain", Style: "Regular"}},
		},
	}); err != nil {
		t.Fatal(err)
	}

	items, staging := prepareMergedStagedLocals(t, []string{src}, true)
	// Fail install and recovery re-registration.
	fm := &failNInstallsAfterRemoveFM{copyFontManager: &copyFontManager{dir: fontDir}, installFailsLeft: 2}
	status, _, err := runUnifiedAddSession(context.Background(), items, fm, platform.UserScope, fontDir, true, false, false, staging)
	if status == nil || status.FailedItems < 1 {
		t.Fatalf("expected failed item, status=%+v err=%v", status, err)
	}
	joined := strings.Join(status.Errors, "\n")
	if err != nil {
		joined += "\n" + err.Error()
	}
	if !strings.Contains(joined, "preserved staged copy") && !strings.Contains(joined, "recovery failed") {
		t.Fatalf("want recovery failure diagnostic, got %q", joined)
	}
	root := staging.Root
	_ = staging.Cleanup()
	if !staging.Retained() {
		t.Fatal("staging must be retained when recovery fails")
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("staged tree must remain: %v", err)
	}
	if got, err := os.ReadFile(src); err == nil && string(got) != string(payload) {
		t.Fatal("if origin exists it must match original payload")
	}
}

func TestMergeLocal_CrossFamilyBasenameCollision(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)

	a := filepath.Join(t.TempDir(), "FamilyA", "Regular.ttf")
	b := filepath.Join(t.TempDir(), "FamilyB", "Regular.ttf")
	writeLocalTTF(t, a, "FamilyA", "Regular")
	writeLocalTTF(t, b, "FamilyB", "Regular")

	var items []addWorkItem
	for _, p := range []string{a, b} {
		prep, err := prepareLocalInstall(p, p, platform.UserScope, true)
		if err != nil {
			t.Fatal(err)
		}
		for i := range prep.Groups {
			g := prep.Groups[i]
			items = append(items, addWorkItem{Kind: addWorkLocal, Local: &g})
		}
	}
	_, err := mergeLocalWorkItems(items)
	if err == nil || !strings.Contains(err.Error(), "destination filename collision") {
		t.Fatalf("want collision, got %v", err)
	}

	// Case-only basename clash
	c := filepath.Join(t.TempDir(), "regular.ttf")
	writeLocalTTF(t, c, "FamilyC", "Bold")
	items = nil
	for _, p := range []string{a, c} {
		prep, err := prepareLocalInstall(p, p, platform.UserScope, true)
		if err != nil {
			t.Fatal(err)
		}
		for i := range prep.Groups {
			g := prep.Groups[i]
			items = append(items, addWorkItem{Kind: addWorkLocal, Local: &g})
		}
	}
	_, err = mergeLocalWorkItems(items)
	if err == nil || !strings.Contains(err.Error(), "destination filename collision") {
		t.Fatalf("want case collision, got %v", err)
	}
}

func TestRetryPaths_CaseSensitiveExact(t *testing.T) {
	cmd := FormatRetryAddCommandMixed(
		[]string{"Google.Roboto", "google.roboto"},
		[]string{"A.ttf", "a.ttf", "A.ttf"},
		"user",
		false,
	)
	if !strings.Contains(cmd, "A.ttf") || !strings.Contains(cmd, "a.ttf") {
		t.Fatalf("both case-distinct paths required: %s", cmd)
	}
	if strings.Count(cmd, "A.ttf") != 1 {
		t.Fatalf("exact dup should collapse once: %s", cmd)
	}
	// Catalog IDs collapse case-insensitively to first seen.
	if strings.Count(strings.ToLower(cmd), "google.roboto") != 1 {
		t.Fatalf("catalog ids should dedupe case-insensitively: %s", cmd)
	}
}
