package logging

import (
	"path/filepath"
	"testing"
)

func TestSetGlobalGetLoggerCloseGlobal(t *testing.T) {
	t.Cleanup(func() { _ = CloseGlobal() })

	cfg := DefaultConfig()
	path := filepath.Join(t.TempDir(), "fontget.log")
	l, err := NewWithPath(cfg, path)
	if err != nil {
		t.Fatal(err)
	}

	SetGlobal(l)
	if g := GetLogger(); g != l {
		t.Fatalf("GetLogger: want same pointer as SetGlobal, got %v vs %v", g, l)
	}

	dir, err := ActiveLogDir()
	if err != nil {
		t.Fatal(err)
	}
	if dir != filepath.Dir(path) {
		t.Fatalf("ActiveLogDir: got %q want %q", dir, filepath.Dir(path))
	}

	if err := CloseGlobal(); err != nil {
		t.Fatal(err)
	}
	if GetLogger() != nil {
		t.Fatal("GetLogger after CloseGlobal: want nil")
	}
	if _, err := ActiveLogDir(); err == nil {
		t.Fatal("ActiveLogDir after CloseGlobal: want error")
	}
}

func TestSetGlobalReplacesPrevious(t *testing.T) {
	t.Cleanup(func() { _ = CloseGlobal() })

	dir := t.TempDir()
	l1, err := NewWithPath(DefaultConfig(), filepath.Join(dir, "a.log"))
	if err != nil {
		t.Fatal(err)
	}
	l2, err := NewWithPath(DefaultConfig(), filepath.Join(dir, "b.log"))
	if err != nil {
		t.Fatal(err)
	}

	SetGlobal(l1)
	SetGlobal(l2)
	if GetLogger() != l2 {
		t.Fatal("expected second logger as global")
	}
	// l1 should have been closed by SetGlobal; closing global should not double-panic
	if err := CloseGlobal(); err != nil {
		t.Fatal(err)
	}
}
