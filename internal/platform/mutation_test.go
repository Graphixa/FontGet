package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPlaceFontFileCreates(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.ttf")
	dst := filepath.Join(dir, "dst.ttf")
	if err := os.WriteFile(src, []byte("new-font-bytes"), 0644); err != nil {
		t.Fatal(err)
	}
	var mut FileMutation
	if _, err := placeFontFile(src, dst, false, &InstallFontOptions{Mutation: &mut}); err != nil {
		t.Fatal(err)
	}
	if !mut.Created || mut.Replaced {
		t.Fatalf("mutation = %+v", mut)
	}
	got, _ := os.ReadFile(dst)
	if string(got) != "new-font-bytes" {
		t.Fatalf("dest = %q", got)
	}
}

func TestPlaceFontFileForceReplaceLeavesOldOnInjectedFail(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.ttf")
	dst := filepath.Join(dir, "dst.ttf")
	if err := os.WriteFile(src, []byte("new-bytes-here"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("old-bytes-here"), 0644); err != nil {
		t.Fatal(err)
	}
	var mut FileMutation
	_, err := placeFontFile(src, dst, true, &InstallFontOptions{Mutation: &mut, FailPoint: InstallFailReplace})
	if err == nil {
		t.Fatal("expected injected replace failure")
	}
	got, _ := os.ReadFile(dst)
	if string(got) != "old-bytes-here" {
		t.Fatalf("original bytes lost: %q", got)
	}
	if mut.BackupPath != "" {
		t.Fatal("force replace must not create backups")
	}
}

func TestPlaceFontFileForceOverwrites(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.ttf")
	dst := filepath.Join(dir, "dst.ttf")
	_ = os.WriteFile(src, []byte("new-bytes-here"), 0644)
	_ = os.WriteFile(dst, []byte("old-bytes-here"), 0644)
	mut, err := placeFontFile(src, dst, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !mut.Replaced || mut.BackupPath != "" {
		t.Fatalf("mutation = %+v", mut)
	}
	got, _ := os.ReadFile(dst)
	if string(got) != "new-bytes-here" {
		t.Fatalf("dest = %q", got)
	}
	if err := RollbackMutation(mut); err != nil {
		t.Fatal(err)
	}
	// No backup: contents stay; registration undo is a no-op without hooks.
	got, _ = os.ReadFile(dst)
	if string(got) != "new-bytes-here" {
		t.Fatalf("force replace leaves new bytes: %q", got)
	}
}

func TestPlaceFontFileCopyAfterWriteTracksPartial(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.ttf")
	dst := filepath.Join(dir, "dst.ttf")
	if err := os.WriteFile(src, []byte("partial-new-bytes"), 0644); err != nil {
		t.Fatal(err)
	}
	var mut FileMutation
	_, err := placeFontFile(src, dst, false, &InstallFontOptions{Mutation: &mut, FailPoint: InstallFailCopyAfterWrite})
	if err == nil {
		t.Fatal("expected copy-after-write failure")
	}
	if !mut.Created || mut.DestPath != dst {
		t.Fatalf("partial dest must be tracked: %+v", mut)
	}
	if err := RollbackMutation(mut); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatal("partial dest must be removed by rollback")
	}
}

func TestCheckDestinationCollisions(t *testing.T) {
	a := filepath.Join("dir-a", "Foo.ttf")
	b := filepath.Join("dir-b", "Foo.ttf")
	c := filepath.Join("dir-a", "Bar.ttf")
	if err := CheckDestinationCollisions([]string{a, b}); err == nil {
		t.Fatal("expected collision")
	}
	if err := CheckDestinationCollisions([]string{a, c}); err != nil {
		t.Fatal(err)
	}
}

func TestPlaceFontFileSkipWithoutForce(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.ttf")
	dst := filepath.Join(dir, "dst.ttf")
	_ = os.WriteFile(src, []byte("new"), 0644)
	_ = os.WriteFile(dst, []byte("old"), 0644)
	if _, err := placeFontFile(src, dst, false, nil); err == nil {
		t.Fatal("expected already installed")
	}
	got, _ := os.ReadFile(dst)
	if string(got) != "old" {
		t.Fatalf("must not touch dest: %q", got)
	}
}
