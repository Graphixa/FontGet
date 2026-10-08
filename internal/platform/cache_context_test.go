package platform

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallCancelledContextDoesNotMutateDestination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	manager, err := NewFontManager()
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "Absent.ttf")
	err = manager.InstallFont(source, UserScope, true, &InstallFontOptions{Context: ctx})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("must stop before file access: %v", err)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("unexpected source: %v", err)
	}
}
