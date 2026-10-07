package cmd

import (
	"context"
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

// recoverLocalOverlaps restores destination-overlapping inputs from staged copies under the
// destination lock. Uses a non-cancelled context so user cancel cannot abort recovery.
func recoverLocalOverlaps(
	overlaps []localOverlap,
	fontManager platform.FontManager,
	installScope platform.InstallationScope,
	fontDir string,
	group localFontGroup,
	prior *installations.Installation,
	mutated bool,
) error {
	if len(overlaps) == 0 || !mutated {
		return nil
	}
	recoverCtx := context.Background()
	_ = recoverCtx

	var first error
	var restoredFiles []installations.InstalledFontFile
	for _, o := range overlaps {
		if _, err := os.Stat(o.Staged); err != nil {
			if first == nil {
				first = fmt.Errorf("staged copy missing for %s: %w", o.Origin, err)
			}
			continue
		}
		if err := copyFileExact(o.Staged, o.Origin); err != nil {
			if first == nil {
				first = fmt.Errorf("restore %s: %w", o.Origin, err)
			}
			continue
		}
		if err := fontManager.InstallFont(o.Origin, installScope, true, nil); err != nil {
			// File bytes are restored; registration may still fail on some platforms.
			if first == nil {
				first = fmt.Errorf("re-register %s: %w", o.Origin, err)
			}
		}
		face := installations.InstalledFontFile{
			Path: o.Origin,
			SFNT: installations.SFNTSnapshot{Family: group.FamilyName},
		}
		if md, err := platform.ExtractFontMetadata(o.Origin); err == nil && md != nil {
			fam := strings.TrimSpace(md.TypographicFamily)
			if fam == "" {
				fam = strings.TrimSpace(md.FamilyName)
			}
			style := strings.TrimSpace(md.TypographicStyle)
			if style == "" {
				style = strings.TrimSpace(md.StyleName)
			}
			face.SFNT = installations.SFNTSnapshot{
				Family:   fam,
				Style:    style,
				FullName: strings.TrimSpace(md.FullName),
			}
		}
		restoredFiles = append(restoredFiles, face)
	}
	if len(restoredFiles) == 0 && first != nil {
		return first
	}

	catalogName := group.FamilyName
	installSrc := group.InstallSrc
	fontGetVer := version.GetVersion()
	if prior != nil {
		if prior.CatalogName != "" {
			catalogName = prior.CatalogName
		}
		if prior.InstallationSource != "" {
			installSrc = prior.InstallationSource
		}
		if prior.FontGetVersion != "" {
			fontGetVer = prior.FontGetVersion
		}
		// Prefer prior face list for files we restored by basename match; keep other prior faces that still exist.
		byBase := map[string]installations.InstalledFontFile{}
		for _, f := range restoredFiles {
			byBase[strings.ToLower(filepath.Base(f.Path))] = f
		}
		var merged []installations.InstalledFontFile
		for _, f := range prior.FlatFiles() {
			base := strings.ToLower(filepath.Base(f.Path))
			if rep, ok := byBase[base]; ok {
				merged = append(merged, rep)
				delete(byBase, base)
				continue
			}
			if _, err := os.Stat(f.Path); err == nil {
				merged = append(merged, f)
			}
		}
		for _, f := range byBase {
			merged = append(merged, f)
		}
		restoredFiles = merged
	}
	if len(restoredFiles) > 0 {
		if err := installations.UpsertInstallation(installations.UpsertParams{
			FontID:             group.FontID,
			CatalogName:        catalogName,
			InstallationSource: installSrc,
			Scope:              string(installScope),
			FontGetVersion:     fontGetVer,
			Files:              restoredFiles,
		}); err != nil && first == nil {
			first = fmt.Errorf("restore installation registry: %w", err)
		}
	}
	return first
}

func wrapLocalRecoveryError(opErr error, staging *platform.OperationStaging, recoverErr error) error {
	if recoverErr == nil {
		return opErr
	}
	preserved := ""
	if staging != nil {
		staging.Retain()
		preserved = staging.Root
	}
	if preserved == "" {
		return fmt.Errorf("%w; recovery failed: %v (manual recovery required)", opErr, recoverErr)
	}
	return fmt.Errorf("%w; recovery failed: %v; preserved staged copy at %s (manual recovery required)", opErr, recoverErr, preserved)
}
