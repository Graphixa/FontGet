package platform

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fontget/internal/testutil"
)

func writeMinimalTTF(t *testing.T, path, family, style string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, testutil.MinimalTTF(family, style), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeZip(t *testing.T, zipPath string, comment string, files map[string][]byte) {
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

func TestDiscoverAndDedupe_SingleFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "Solo-Regular.ttf")
	writeMinimalTTF(t, p, "Solo", "Regular")
	res, err := DiscoverAndDedupeLocalFonts(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Kept) != 1 {
		t.Fatalf("kept=%d", len(res.Kept))
	}
	if !res.Kept[0].LooseFile || res.Kept[0].Family != "Solo" {
		t.Fatalf("got %+v", res.Kept[0])
	}
}

func TestDiscoverAndDedupe_FolderNested(t *testing.T) {
	root := t.TempDir()
	writeMinimalTTF(t, filepath.Join(root, "OpenSans", "OpenSans-Regular.ttf"), "Open Sans", "Regular")
	writeMinimalTTF(t, filepath.Join(root, "Abeezee", "ABeeZee-Regular.ttf"), "ABeeZee", "Regular")
	_ = os.WriteFile(filepath.Join(root, ".DS_Store"), []byte("x"), 0o644)
	_ = os.MkdirAll(filepath.Join(root, "__MACOSX"), 0o755)

	res, err := DiscoverAndDedupeLocalFonts(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Kept) != 2 {
		t.Fatalf("kept=%d warnings=%v", len(res.Kept), res.Warnings)
	}
}

func TestDiscoverAndDedupe_HashPrefersLooseOverZip(t *testing.T) {
	root := t.TempDir()
	data := testutil.MinimalTTF("DupFam", "Regular")
	loose := filepath.Join(root, "DupFam", "Dup-Regular.ttf")
	writeMinimalTTF(t, loose, "DupFam", "Regular")
	// overwrite with exact same bytes as zip will use
	if err := os.WriteFile(loose, data, 0o644); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(root, "DupFam", "DupFam.zip")
	writeZip(t, zipPath, "", map[string][]byte{"Dup-Regular.ttf": data})

	res, err := DiscoverAndDedupeLocalFonts(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Kept) != 1 {
		t.Fatalf("kept=%d hashSkip=%d", len(res.Kept), res.HashDupesSkipped)
	}
	if !res.Kept[0].LooseFile {
		t.Fatal("expected loose file preferred")
	}
	if res.HashDupesSkipped < 1 {
		t.Fatalf("expected hash dupe skip, got %d", res.HashDupesSkipped)
	}
	if res.NestedZipFonts < 1 {
		t.Fatalf("expected nested zip fonts counted, got %d", res.NestedZipFonts)
	}
}

func TestDiscoverAndDedupe_SFNTFaceDedupe(t *testing.T) {
	root := t.TempDir()
	// Same family/style, different filenames so hash buckets differ; face dedupe keeps one.
	writeMinimalTTF(t, filepath.Join(root, "A-Regular.ttf"), "SameFace", "Regular")
	writeMinimalTTF(t, filepath.Join(root, "B-Regular.ttf"), "SameFace", "Regular")

	res, err := DiscoverAndDedupeLocalFonts(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Kept) != 1 {
		t.Fatalf("kept=%d faceSkip=%d", len(res.Kept), res.FaceDupesSkipped)
	}
	if res.FaceDupesSkipped < 1 {
		t.Fatalf("faceSkip=%d", res.FaceDupesSkipped)
	}
}

func TestDiscoverAndDedupe_BasenameConflict(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "pack1", "Regular.ttf")
	b := filepath.Join(root, "pack2", "Regular.ttf")
	writeMinimalTTF(t, a, "FamilyOne", "Regular")
	writeMinimalTTF(t, b, "FamilyTwo", "Regular")

	res, err := DiscoverAndDedupeLocalFonts(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Kept) != 1 {
		t.Fatalf("kept=%d conflict=%d", len(res.Kept), res.ConflictsSkipped)
	}
	if res.ConflictsSkipped != 1 {
		t.Fatalf("conflicts=%d warnings=%v", res.ConflictsSkipped, res.Warnings)
	}
	found := false
	for _, w := range res.Warnings {
		if strings.Contains(w, "conflicts with kept") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing conflict warning: %v", res.Warnings)
	}
}

func TestDiscoverAndDedupe_TopLevelZipBackupComment(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "backup.zip")
	data := testutil.MinimalTTF("BackupFam", "Regular")
	writeZip(t, zipPath, fontGetBackupZipComment, map[string][]byte{
		"Google Fonts/BackupFam/BackupFam-Regular.ttf": data,
		"nested/inner.zip": []byte("PK\x03\x04dummy"),
	})

	res, err := DiscoverAndDedupeLocalFonts(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if !res.FontGetBackup {
		t.Fatal("expected FontGet backup detection")
	}
	if len(res.Kept) != 1 {
		t.Fatalf("kept=%d", len(res.Kept))
	}
	if !res.Kept[0].FromZip {
		t.Fatal("expected zip candidate")
	}
	skippedNested := false
	for _, w := range res.Warnings {
		if strings.Contains(w, "nested zip member") {
			skippedNested = true
		}
	}
	if !skippedNested {
		t.Fatalf("expected nested zip member skip warning: %v", res.Warnings)
	}
}

func TestDiscoverAndDedupe_Empty(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hi"), 0o644)
	_, err := DiscoverAndDedupeLocalFonts(dir)
	if err == nil || !strings.Contains(err.Error(), "no font files") {
		t.Fatalf("err=%v", err)
	}
}

func TestExtractZipEntryToTemp_Basename(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "f.zip")
	data := testutil.MinimalTTF("TmpFam", "Bold")
	writeZip(t, zipPath, "", map[string][]byte{"dir/MyFont-Bold.ttf": data})
	out, err := ExtractZipEntryToTemp(zipPath, "dir/MyFont-Bold.ttf", "MyFont-Bold.ttf")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(filepath.Dir(out))
	if filepath.Base(out) != "MyFont-Bold.ttf" {
		t.Fatalf("basename=%s", out)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(data) {
		t.Fatal("content mismatch")
	}
}

func TestDiscoverAndDedupe_DirectCollectionRejected(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"face.ttc", "face.otc"} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("ttcf"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := DiscoverAndDedupeLocalFonts(p)
		if err == nil || !strings.Contains(err.Error(), "font collections (.ttc and .otc) are not supported") {
			t.Fatalf("%s: err=%v", name, err)
		}
	}
}

func TestDiscoverAndDedupe_ZipOnlyCollections(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "cols.zip")
	writeZip(t, zipPath, "", map[string][]byte{
		"a.ttc": []byte("ttcf"),
		"b.otc": []byte("otcf"),
	})
	_, err := DiscoverAndDedupeLocalFonts(zipPath)
	if err == nil || !strings.Contains(err.Error(), "no font files") {
		t.Fatalf("err=%v", err)
	}
}

func TestIsLocalInstallFontExt_NoCollections(t *testing.T) {
	if IsLocalInstallFontExt(".ttc") || IsLocalInstallFontExt(".otc") {
		t.Fatal("collections must not be local install extensions")
	}
	if !IsLocalInstallFontExt(".ttf") || !IsLocalInstallFontExt(".otf") {
		t.Fatal("ttf/otf must remain installable")
	}
	if !IsLocalCollectionFontExt(".ttc") || !IsLocalCollectionFontExt(".otc") {
		t.Fatal("expected collection helpers")
	}
}
