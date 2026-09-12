package platform

import (
	"fmt"
	"os"
	"path/filepath"
)

func PlaceFontFile(src, dest string, force bool, opts *InstallFontOptions) (FileMutation, error) {
	return placeFontFile(src, dest, force, opts)
}

func failPointError(point InstallFailPoint) error {
	return fmt.Errorf("injected failure at %s", point)
}

// placeFontFile copies src to dest, using a same-volume backup and staged file when replacing.
// It never uses an unconditional Remove(old) followed by an unprotected copy as the only strategy.
func placeFontFile(src, dest string, force bool, opts *InstallFontOptions) (FileMutation, error) {
	var mut FileMutation
	mut.DestPath = dest

	if _, err := os.Stat(dest); err == nil {
		if !force {
			return mut, fmt.Errorf("font already installed: %s", filepath.Base(dest))
		}
		if err := replaceExistingFontFile(src, dest, &mut, opts); err != nil {
			if opts != nil && opts.Mutation != nil {
				*opts.Mutation = mut
			}
			return mut, err
		}
	} else if !os.IsNotExist(err) {
		return mut, fmt.Errorf("stat destination: %w", err)
	} else {
		if opts != nil && opts.FailPoint == InstallFailCopy {
			return mut, failPointError(InstallFailCopy)
		}
		if err := copyFile(src, dest); err != nil {
			return mut, fmt.Errorf("copy font file: %w", err)
		}
		mut.Created = true
	}

	if opts != nil && opts.Mutation != nil {
		*opts.Mutation = mut
	}
	return mut, nil
}

func replaceExistingFontFile(src, dest string, mut *FileMutation, opts *InstallFontOptions) error {
	backup := dest + ".fontget-bak"
	if opts != nil && opts.FailPoint == InstallFailBackup {
		return failPointError(InstallFailBackup)
	}
	if err := copyFile(dest, backup); err != nil {
		return fmt.Errorf("backup existing font: %w", err)
	}
	mut.BackupPath = backup
	mut.Replaced = true

	if opts != nil && opts.FailPoint == InstallFailCopy {
		return failPointError(InstallFailCopy)
	}

	staged := dest + ".fontget-new"
	if err := copyFile(src, staged); err != nil {
		_ = os.Remove(backup)
		mut.BackupPath = ""
		mut.Replaced = false
		return fmt.Errorf("stage replacement: %w", err)
	}

	if opts != nil && opts.FailPoint == InstallFailReplace {
		_ = os.Remove(staged)
		return failPointError(InstallFailReplace)
	}

	oldMoved := dest + ".fontget-old"
	_ = os.Remove(oldMoved)
	if err := os.Rename(dest, oldMoved); err != nil {
		_ = os.Remove(staged)
		return fmt.Errorf("cannot replace locked or busy font: %w", err)
	}
	if err := os.Rename(staged, dest); err != nil {
		_ = os.Rename(oldMoved, dest)
		_ = os.Remove(staged)
		return fmt.Errorf("install replacement: %w", err)
	}
	_ = os.Remove(oldMoved)
	return nil
}

// CommitMutation deletes disposable backups after a successful package commit.
func CommitMutation(mut FileMutation) error {
	if mut.BackupPath == "" {
		return nil
	}
	if err := os.Remove(mut.BackupPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// RollbackMutation undoes a single file mutation. Pre-existing files are restored from backup;
// files created by this operation are removed. Untouched fonts are not deleted.
func RollbackMutation(mut FileMutation) error {
	if mut.Replaced && mut.BackupPath != "" {
		if err := copyFile(mut.BackupPath, mut.DestPath); err != nil {
			return fmt.Errorf("restore backup %s: %w", mut.BackupPath, err)
		}
		return nil
	}
	if mut.Created && mut.DestPath != "" {
		if err := os.Remove(mut.DestPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove created file %s: %w", mut.DestPath, err)
		}
	}
	return nil
}
