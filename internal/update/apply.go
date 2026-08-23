package update

import (
	"fmt"
	"os"
	"path/filepath"
)

// replaceExecutable writes newBinary over destPath using a same-directory temp
// file, renaming the current binary to destPath+".old" so a running process
// can still be replaced on Windows. On failure the original file is restored.
func replaceExecutable(destPath string, newBinary []byte) error {
	if destPath == "" {
		return fmt.Errorf("empty executable path")
	}
	if len(newBinary) == 0 {
		return fmt.Errorf("empty replacement binary")
	}

	abs, err := filepath.Abs(destPath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("failed to stat current executable: %w", err)
	}
	mode := info.Mode()

	dir := filepath.Dir(abs)
	tmp, err := os.CreateTemp(dir, "fontget-update-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmp.Name()
	cleanupTmp := true
	defer func() {
		if cleanupTmp {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(newBinary); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write replacement binary: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close replacement binary: %w", err)
	}
	// #nosec G302 -- must stay executable; permissions are copied from the replaced binary
	if err := os.Chmod(tmpName, mode); err != nil && currentGOOS != "windows" {
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}

	oldPath := abs + ".old"
	_ = os.Remove(oldPath)
	if err := os.Rename(abs, oldPath); err != nil {
		return fmt.Errorf("failed to move current binary aside: %w", err)
	}
	if err := os.Rename(tmpName, abs); err != nil {
		_ = os.Rename(oldPath, abs)
		return fmt.Errorf("failed to install replacement binary: %w", err)
	}
	cleanupTmp = false
	return nil
}
