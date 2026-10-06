package cmd

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"fontget/internal/shared"
)

func TestBackupZipCommentIsSet(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "out.zip")
	f, err := os.Create(outPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	if err := zw.SetComment(shared.BackupZipComment); err != nil {
		t.Fatal(err)
	}
	w, err := zw.Create("Other/Demo/Demo-Regular.ttf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("not-a-real-font-but-ok-for-zip")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	r, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if !shared.IsFontGetBackupComment(r.Comment) {
		t.Fatalf("comment = %q, want FontGet backup marker", r.Comment)
	}
}
