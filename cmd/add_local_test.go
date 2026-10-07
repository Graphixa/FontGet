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
	if err := installations.RecordInstallation(installations.RecordParams{
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
	if err := installations.RecordInstallation(installations.RecordParams{
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

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	status, _, runErr := runUnifiedAddSession(ctx, items, fm, platform.UserScope, fontDir, false, false, false, staging)
	if runErr == nil && (status == nil || status.FailedItems == 0) {
		// Lock wait may cancel via context — still must not look like success with zero failures.
		if status != nil && status.FailedItems == 0 && status.Installed > 0 {
			t.Fatal("unexpected success under held lock")
		}
	}
	if status != nil && status.FailedItems == 0 && !IsCancelErr(runErr) {
		t.Fatalf("want FailedItems or cancel, status=%+v err=%v", status, runErr)
	}
	if status != nil && status.FailedItems > 0 {
		if len(status.Errors) == 0 {
			t.Fatal("expected error text preserved")
		}
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
