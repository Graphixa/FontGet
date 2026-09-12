package platform

import (
	"os"
	"path/filepath"
	"strings"
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

func TestPlaceFontFileForceRestoresOnReplaceFail(t *testing.T) {
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
	if mut.BackupPath == "" {
		t.Fatal("backup must exist so recovery remains possible")
	}
	bak, _ := os.ReadFile(mut.BackupPath)
	if string(bak) != "old-bytes-here" {
		t.Fatalf("backup = %q", bak)
	}
	if !strings.Contains(filepath.Base(mut.BackupPath), ".fontget-bak-") {
		t.Fatalf("backup should use unique suffix, got %s", mut.BackupPath)
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

func TestUniqueBackupSurvivesSecondAttempt(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.ttf")
	dst := filepath.Join(dir, "dst.ttf")
	_ = os.WriteFile(src, []byte("new-bytes-here"), 0644)
	_ = os.WriteFile(dst, []byte("old-bytes-here"), 0644)

	var mut1 FileMutation
	_, err := placeFontFile(src, dst, true, &InstallFontOptions{Mutation: &mut1, FailPoint: InstallFailReplace})
	if err == nil {
		t.Fatal("expected first failure")
	}
	firstBak := mut1.BackupPath
	firstBytes, _ := os.ReadFile(firstBak)

	var mut2 FileMutation
	_, err = placeFontFile(src, dst, true, &InstallFontOptions{Mutation: &mut2, FailPoint: InstallFailReplace})
	if err == nil {
		t.Fatal("expected second failure")
	}
	if mut2.BackupPath == firstBak {
		t.Fatal("second attempt must not reuse first backup path")
	}
	got, err := os.ReadFile(firstBak)
	if err != nil {
		t.Fatalf("first recovery backup destroyed: %v", err)
	}
	if string(got) != string(firstBytes) {
		t.Fatal("first backup contents changed")
	}
}

func TestCheckDestinationCollisions(t *testing.T) {
	if err := CheckDestinationCollisions([]string{`C:\a\Foo.ttf`, `C:\b\Foo.ttf`}); err == nil {
		t.Fatal("expected collision")
	}
	if err := CheckDestinationCollisions([]string{`C:\a\Foo.ttf`, `C:\a\Bar.ttf`}); err != nil {
		t.Fatal(err)
	}
}

func TestRollbackAndCommitMutation(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.ttf")
	dst := filepath.Join(dir, "dst.ttf")
	if err := os.WriteFile(src, []byte("new-bytes-here"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("old-bytes-here"), 0644); err != nil {
		t.Fatal(err)
	}
	mut, err := placeFontFile(src, dst, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(dst)
	if string(got) != "new-bytes-here" {
		t.Fatalf("replaced = %q", got)
	}
	if err := RollbackMutation(mut); err != nil {
		t.Fatal(err)
	}
	got, _ = os.ReadFile(dst)
	if string(got) != "old-bytes-here" {
		t.Fatalf("restored = %q", got)
	}
	if err := os.WriteFile(dst, []byte("new-bytes-here"), 0644); err != nil {
		t.Fatal(err)
	}
	mut2, err := placeFontFile(src, dst, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := CommitMutation(mut2); err != nil {
		t.Fatal(err)
	}
	if mut2.BackupPath != "" {
		if _, err := os.Stat(mut2.BackupPath); !os.IsNotExist(err) {
			t.Fatalf("backup should be removed after commit: %v", err)
		}
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
