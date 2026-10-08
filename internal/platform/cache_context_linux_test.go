package platform

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFontCacheCommandHonoursRecoveryDeadline(t *testing.T) {
	dir := t.TempDir()
	// Replace the child shell with sleep so cancelling the command cannot leave a
	// descendant holding the output pipes open.
	if err := os.WriteFile(filepath.Join(dir, "fc-cache"), []byte("#!/bin/sh\nexec /bin/sleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	manager := &linuxFontManager{userFontDir: t.TempDir()}
	if err := manager.updateFontCacheContext(ctx, UserScope); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline lost: %v", err)
	}
}
