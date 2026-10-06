package cmd

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

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
