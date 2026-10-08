package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fontget/internal/installations"
	"fontget/internal/platform"
	"fontget/internal/version"
)

// localOverlap is a staged local candidate whose origin lived under the destination font dir.
type localOverlap struct {
	Origin string
	Staged string
	Base   string
}

func localOverlapsForFontDir(cands []platform.LocalFontCandidate, fontDir string) []localOverlap {
	var out []localOverlap
	for _, c := range cands {
		origin := strings.TrimSpace(c.OriginDiskPath)
		if origin == "" {
			continue
		}
		if !installations.DirContainsFontFile(fontDir, origin) {
			continue
		}
		out = append(out, localOverlap{
			Origin: origin,
			Staged: c.DiskPath,
			Base:   c.Basename,
		})
	}
	return out
}

func overlapBasenameHit(overlaps []localOverlap, basenames []string) bool {
	if len(overlaps) == 0 || len(basenames) == 0 {
		return false
	}
	want := make(map[string]struct{}, len(basenames))
	for _, b := range basenames {
		want[strings.ToLower(filepath.Base(b))] = struct{}{}
	}
	for _, o := range overlaps {
		if _, ok := want[strings.ToLower(o.Base)]; ok {
			return true
		}
		if _, ok := want[strings.ToLower(filepath.Base(o.Origin))]; ok {
			return true
		}
	}
	return false
}

func snapshotInstallation(fontID string) *installations.Installation {
	reg, err := installations.Load()
	if err != nil {
		return nil
	}
	inst := reg.FindByFontID(fontID)
	if inst == nil {
		return nil
	}
	cp := *inst
	cp.Families = append([]installations.FamilyGroup(nil), inst.Families...)
	for i := range cp.Families {
		cp.Families[i].Files = append([]installations.InstalledFace(nil), inst.Families[i].Files...)
	}
	cp.Remaining = append([]string(nil), inst.Remaining...)
	cp.LastErrors = append([]string(nil), inst.LastErrors...)
	return &cp
}

func copyFileExact(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	tmp := dest + ".fontget-restore-" + fmt.Sprintf("%d", time.Now().UnixNano())
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	_ = os.Remove(dest)
	if err := os.Rename(tmp, dest); err != nil {
		if copyErr := copyFileViaRead(tmp, dest); copyErr != nil {
			_ = os.Remove(tmp)
			return copyErr
		}
		_ = os.Remove(tmp)
	}
	return nil
}

func copyFileViaRead(src, dest string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dest, data, 0o644)
}

// recoverLocalOverlaps runs synchronously under the destination lock. It must
// finish before staging cleanup, even when the user cancelled the install.
func recoverLocalOverlaps(
	overlaps []localOverlap,
	fontManager platform.FontManager,
	installScope platform.InstallationScope,
	fontDir string,
	group localFontGroup,
	prior *installations.Installation,
	opErr error,
) error {
	// Never inherit the cancelled operation context. No background worker is
	// left mutating files after the destination lock is released.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	reg, registryErr := installations.Load()
	var recoveryErrors []error
	var current *installations.Installation
	if registryErr != nil {
		recoveryErrors = append(recoveryErrors, fmt.Errorf("load recovery registry: %w", registryErr))
	} else {
		current = reg.FindByFontID(group.FontID)
	}
	var restoredFiles []installations.InstalledFontFile
	var registrationPending []string
	for _, o := range overlaps {
		if err := copyFileExact(o.Staged, o.Origin); err != nil {
			recoveryErrors = append(recoveryErrors, fmt.Errorf("restore %s: %w", o.Origin, err))
			registrationPending = append(registrationPending, filepath.Base(o.Origin))
			continue
		}
		if err := fontManager.InstallFont(o.Origin, installScope, true, &platform.InstallFontOptions{Context: ctx}); err != nil {
			recoveryErrors = append(recoveryErrors, fmt.Errorf("re-register %s: %w", o.Origin, err))
			registrationPending = append(registrationPending, filepath.Base(o.Origin))
		}
		face, err := installedFaceFromPath(o.Origin, "")
		if err != nil {
			recoveryErrors = append(recoveryErrors, fmt.Errorf("restored font metadata: %w", err))
			registrationPending = append(registrationPending, filepath.Base(o.Origin))
			continue
		}
		restoredFiles = append(restoredFiles, face)
	}

	// Keep both earlier files and files completed by this attempt. Upsert replaces
	// the entire record, so restoring only the old snapshot would lose new faces.
	var files []installations.InstalledFontFile
	byPath := map[string]int{}
	addFace := func(f installations.InstalledFontFile) {
		if !installations.DirContainsFontFile(fontDir, f.Path) {
			return
		}
		info, err := os.Stat(f.Path)
		if err != nil || !info.Mode().IsRegular() {
			return
		}
		f.Path = platform.CanonicalPath(f.Path)
		if i, ok := byPath[f.Path]; ok {
			files[i] = f
			return
		}
		byPath[f.Path] = len(files)
		files = append(files, f)
	}
	var expected, lastErrors []string
	catalogName, installSrc := group.FamilyName, group.InstallSrc
	fontGetVer := version.GetVersion()
	priorIncomplete := prior != nil && prior.IsIncomplete()
	for _, inst := range []*installations.Installation{prior, current} {
		if inst == nil {
			continue
		}
		if inst.CatalogName != "" {
			catalogName = inst.CatalogName
		}
		if inst.InstallationSource != "" {
			installSrc = inst.InstallationSource
		}
		for _, f := range inst.FlatFiles() {
			if installations.DirContainsFontFile(fontDir, f.Path) {
				expected = append(expected, filepath.Base(f.Path))
				addFace(f)
			}
		}
		expected = append(expected, inst.Remaining...)
		lastErrors = append(lastErrors, inst.LastErrors...)
	}
	for _, c := range group.Candidates {
		expected = append(expected, c.Basename)
	}
	for _, f := range restoredFiles {
		addFace(f)
	}
	var present []string
	for _, f := range files {
		present = append(present, filepath.Base(f.Path))
	}
	remaining := remainingBasenames(dedupeBasenameList(expected), present)
	remaining = dedupeBasenameList(append(remaining, registrationPending...))
	status := ""
	if len(remaining) > 0 || len(recoveryErrors) > 0 || priorIncomplete {
		status = installations.StatusIncompleteInstall
		if opErr != nil {
			lastErrors = append(lastErrors, opErr.Error())
		}
		for _, err := range recoveryErrors {
			lastErrors = append(lastErrors, err.Error())
		}
	}
	// Restore bytes even if the registry is unreadable, but never overwrite it
	// with a guessed record. The returned error retains staging for manual repair.
	if registryErr != nil {
		return errors.Join(recoveryErrors...)
	}
	if err := installations.UpsertInstallation(installations.UpsertParams{
		FontID: group.FontID, CatalogName: catalogName, InstallationSource: installSrc,
		Scope: string(installScope), FontGetVersion: fontGetVer, Files: files,
		Status: status, Remaining: remaining, LastErrors: lastErrors,
	}); err != nil {
		recoveryErrors = append(recoveryErrors, fmt.Errorf("restore installation registry: %w", err))
	}
	return errors.Join(recoveryErrors...)
}

// localRecoveryError must survive cancellation handling: the retained bytes and
// manual recovery instructions are more urgent than normal retry advice.
type localRecoveryError struct {
	operation  error
	recovery   error
	stagedRoot string
}

func (e *localRecoveryError) Error() string {
	if e.stagedRoot == "" {
		return fmt.Sprintf("%v; recovery failed: %v (manual recovery required)", e.operation, e.recovery)
	}
	return fmt.Sprintf("%v; recovery failed: %v; preserved staged copy at %s (manual recovery required)", e.operation, e.recovery, e.stagedRoot)
}

func (e *localRecoveryError) Unwrap() []error { return []error{e.operation, e.recovery} }

func wrapLocalRecoveryError(opErr error, staging *platform.OperationStaging, recoverErr error) error {
	if recoverErr == nil {
		return opErr
	}
	e := &localRecoveryError{operation: opErr, recovery: recoverErr}
	if staging != nil {
		staging.Retain()
		e.stagedRoot = staging.Root
	}
	return e
}
