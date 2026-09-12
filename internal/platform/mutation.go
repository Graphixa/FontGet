package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func PlaceFontFile(src, dest string, force bool, opts *InstallFontOptions) (FileMutation, error) {
	return placeFontFile(src, dest, force, opts)
}

func failPointError(point InstallFailPoint) error {
	return fmt.Errorf("injected failure at %s", point)
}

func exportMutation(opts *InstallFontOptions, mut FileMutation) {
	if opts != nil && opts.Mutation != nil {
		*opts.Mutation = mut
	}
}

// uniqueArtifact returns an operation-owned path next to dest that will not collide with
// fixed legacy suffixes or another concurrent attempt's retained recovery backup.
func uniqueArtifact(dest, kind string) string {
	return fmt.Sprintf("%s.fontget-%s-%d-%d", dest, kind, os.Getpid(), time.Now().UnixNano())
}

func trackArtifact(mut *FileMutation, path string) {
	if path == "" {
		return
	}
	mut.ArtifactPaths = append(mut.ArtifactPaths, path)
}

func removeTracked(mut *FileMutation, path string) {
	if path == "" {
		return
	}
	_ = os.Remove(path)
	out := mut.ArtifactPaths[:0]
	for _, p := range mut.ArtifactPaths {
		if p != path {
			out = append(out, p)
		}
	}
	mut.ArtifactPaths = out
}

// placeFontFile copies src to dest, using unique same-volume staging and backups when replacing.
// New files are staged then renamed so a failed mid-copy cannot leave an untracked partial dest.
func placeFontFile(src, dest string, force bool, opts *InstallFontOptions) (FileMutation, error) {
	var mut FileMutation
	mut.DestPath = dest

	if _, err := os.Stat(dest); err == nil {
		if !force {
			return mut, fmt.Errorf("font already installed: %s", filepath.Base(dest))
		}
		if err := replaceExistingFontFile(src, dest, &mut, opts); err != nil {
			exportMutation(opts, mut)
			return mut, err
		}
	} else if !os.IsNotExist(err) {
		return mut, fmt.Errorf("stat destination: %w", err)
	} else {
		if err := createNewFontFile(src, dest, &mut, opts); err != nil {
			exportMutation(opts, mut)
			return mut, err
		}
	}

	exportMutation(opts, mut)
	return mut, nil
}

func createNewFontFile(src, dest string, mut *FileMutation, opts *InstallFontOptions) error {
	staged := uniqueArtifact(dest, "new")
	trackArtifact(mut, staged)
	if err := copyFile(src, staged); err != nil {
		removeTracked(mut, staged)
		return fmt.Errorf("stage new font: %w", err)
	}
	if opts != nil && opts.FailPoint == InstallFailCopyAfterWrite {
		// Staged bytes exist; leave DestPath unset as Created. Caller still gets mutation via export
		// so staged artifact is cleaned. Simulate "partial dest" by also creating dest then failing.
		partial := dest
		if copyErr := copyFile(staged, partial); copyErr == nil {
			mut.Created = true
			mut.DestPath = dest
		}
		return failPointError(InstallFailCopyAfterWrite)
	}

	if err := os.Rename(staged, dest); err != nil {
		// Cross-volume or busy: fall back to copy into dest, tracking ownership immediately.
		mut.Created = true
		if copyErr := copyFile(staged, dest); copyErr != nil {
			_ = os.Remove(dest)
			mut.Created = false
			removeTracked(mut, staged)
			return fmt.Errorf("install new font: %w", copyErr)
		}
		removeTracked(mut, staged)
		return nil
	}
	removeTracked(mut, staged)
	mut.Created = true
	return nil
}

func replaceExistingFontFile(src, dest string, mut *FileMutation, opts *InstallFontOptions) error {
	backup := uniqueArtifact(dest, "bak")
	if err := copyFile(dest, backup); err != nil {
		return fmt.Errorf("backup existing font: %w", err)
	}
	mut.BackupPath = backup
	mut.Replaced = true
	trackArtifact(mut, backup)

	staged := uniqueArtifact(dest, "new")
	trackArtifact(mut, staged)
	if err := copyFile(src, staged); err != nil {
		removeTracked(mut, staged)
		return fmt.Errorf("stage replacement: %w", err)
	}
	if opts != nil && opts.FailPoint == InstallFailCopyAfterWrite {
		return failPointError(InstallFailCopyAfterWrite)
	}

	if opts != nil && opts.FailPoint == InstallFailReplace {
		removeTracked(mut, staged)
		return failPointError(InstallFailReplace)
	}

	oldMoved := uniqueArtifact(dest, "old")
	trackArtifact(mut, oldMoved)
	if err := os.Rename(dest, oldMoved); err != nil {
		removeTracked(mut, staged)
		removeTracked(mut, oldMoved)
		return fmt.Errorf("cannot replace locked or busy font: %w", err)
	}
	if err := os.Rename(staged, dest); err != nil {
		_ = os.Rename(oldMoved, dest)
		removeTracked(mut, staged)
		removeTracked(mut, oldMoved)
		return fmt.Errorf("install replacement: %w", err)
	}
	removeTracked(mut, staged)
	removeTracked(mut, oldMoved)
	return nil
}

// CommitMutation deletes disposable backups/artifacts after a successful package commit.
func CommitMutation(mut FileMutation) error {
	var first error
	paths := append([]string{}, mut.ArtifactPaths...)
	if mut.BackupPath != "" {
		paths = append(paths, mut.BackupPath)
	}
	seen := map[string]struct{}{}
	for _, p := range paths {
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) && first == nil {
			first = err
		}
	}
	return first
}

// RollbackMutation undoes file and platform registration changes for one mutation.
// Pre-existing fonts are restored from backup; files created by this operation are removed.
func RollbackMutation(mut FileMutation) error {
	if mut.UndoRegistration != nil {
		if err := mut.UndoRegistration(); err != nil {
			return err
		}
	} else if err := undoFontRegistration(mut); err != nil {
		return err
	}

	var fileErr error
	if mut.Replaced && mut.BackupPath != "" {
		if err := copyFile(mut.BackupPath, mut.DestPath); err != nil {
			fileErr = fmt.Errorf("restore backup %s: %w", mut.BackupPath, err)
		}
	} else if mut.Created && mut.DestPath != "" {
		if err := os.Remove(mut.DestPath); err != nil && !os.IsNotExist(err) {
			fileErr = fmt.Errorf("remove created file %s: %w", mut.DestPath, err)
		}
	}

	if mut.RestoreRegistration != nil {
		if restoreErr := mut.RestoreRegistration(); restoreErr != nil && fileErr == nil {
			fileErr = restoreErr
		}
	} else if restoreErr := restorePriorFontRegistration(mut); restoreErr != nil && fileErr == nil {
		fileErr = restoreErr
	}

	// Do not delete BackupPath here: incomplete recovery must retain it. CommitMutation cleans success.
	for _, p := range mut.ArtifactPaths {
		if p == "" || p == mut.BackupPath {
			continue
		}
		_ = os.Remove(p)
	}
	return fileErr
}

// CheckDestinationCollisions returns an error if two paths resolve to the same destination basename.
func CheckDestinationCollisions(destPaths []string) error {
	seen := make(map[string]string, len(destPaths))
	for _, p := range destPaths {
		key := filepath.Clean(p)
		base := filepath.Base(key)
		if prev, ok := seen[base]; ok && prev != key {
			return fmt.Errorf("destination collision: %s and %s both install as %s", prev, key, base)
		}
		if prev, ok := seen[key]; ok {
			return fmt.Errorf("destination collision: duplicate path %s", prev)
		}
		seen[base] = key
		seen[key] = key
	}
	return nil
}
