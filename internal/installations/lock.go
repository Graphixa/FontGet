package installations

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"fontget/internal/network"
)

func registryLockPath() string {
	return RegistryPath() + ".lock"
}

func withRegistryFileLock(ctx context.Context, fn func() error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	unlock, err := lockFile(ctx, registryLockPath())
	if err != nil {
		return err
	}
	defer unlock()
	return fn()
}

func lockFile(ctx context.Context, path string) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("%w: lock dir: %v", network.ErrLocalFailure, err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("%w: open lock: %v", network.ErrLocalFailure, err)
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		if err := ctx.Err(); err != nil {
			_ = f.Close()
			return nil, err
		}
		if err := tryExclusiveLock(f); err == nil {
			return func() {
				_ = unlockFile(f)
				_ = f.Close()
			}, nil
		}
		if time.Now().After(deadline) {
			_ = f.Close()
			return nil, fmt.Errorf("timed out waiting for lock %s", path)
		}
		t := time.NewTimer(50 * time.Millisecond)
		select {
		case <-ctx.Done():
			t.Stop()
			_ = f.Close()
			return nil, ctx.Err()
		case <-t.C:
		}
	}
}

// LockDestination serializes conflicting destination mutations in fontDir. Not held during downloads.
func LockDestination(ctx context.Context, fontDir string) (func(), error) {
	return lockFile(ctx, filepath.Join(fontDir, ".fontget-install.lock"))
}
