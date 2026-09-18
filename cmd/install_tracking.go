package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fontget/internal/installations"
	"fontget/internal/output"
	"fontget/internal/platform"
	"fontget/internal/repo"
	"fontget/internal/version"
)

// installTracker persists per-file install progress for one package.
type installTracker struct {
	fontID            string
	scope             platform.InstallationScope
	fontDir           string
	catalogName       string
	variantByBasename map[string]string
	installSrc        string
	expected          []string // all package basenames for this attempt
}

func newInstallTracker(fontID string, fontFiles []repo.FontFile, scope platform.InstallationScope, fontDir string, expected []string) *installTracker {
	t := &installTracker{
		fontID:            fontID,
		scope:             scope,
		fontDir:           fontDir,
		variantByBasename: make(map[string]string),
		expected:          dedupeBasenameList(expected),
	}
	for _, ff := range fontFiles {
		b := filepath.Base(strings.TrimSpace(ff.Path))
		if b == "" {
			b = filepath.Base(strings.TrimSpace(ff.Variant))
		}
		if b != "" {
			t.variantByBasename[b] = strings.TrimSpace(ff.Variant)
		}
		if t.catalogName == "" {
			t.catalogName = strings.TrimSpace(ff.Name)
		}
	}
	if fontID != "" {
		if meta, metaErr := repo.MatchRepositoryFontByID(fontID); metaErr == nil && meta != nil {
			t.installSrc = strings.TrimSpace(meta.Source)
		}
	}
	return t
}

func dedupeBasenameList(in []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		s = filepath.Base(s)
		key := strings.ToLower(s)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, s)
	}
	return out
}

func remainingBasenames(expected, present []string) []string {
	have := make(map[string]struct{})
	for _, p := range present {
		have[strings.ToLower(filepath.Base(strings.TrimSpace(p)))] = struct{}{}
	}
	var rem []string
	for _, e := range expected {
		if _, ok := have[strings.ToLower(e)]; !ok {
			rem = append(rem, e)
		}
	}
	return rem
}

func installedFaceFromPath(full, catalogVariant string) (installations.InstalledFontFile, error) {
	md, err := platform.ExtractFontMetadata(full)
	if err != nil {
		return installations.InstalledFontFile{}, err
	}
	fam := strings.TrimSpace(md.TypographicFamily)
	if fam == "" {
		fam = strings.TrimSpace(md.FamilyName)
	}
	style := strings.TrimSpace(md.TypographicStyle)
	if style == "" {
		style = strings.TrimSpace(md.StyleName)
	}
	return installations.InstalledFontFile{
		Path:           full,
		CatalogVariant: catalogVariant,
		SFNT: installations.SFNTSnapshot{
			Family:   fam,
			Style:    style,
			FullName: strings.TrimSpace(md.FullName),
		},
	}, nil
}

func (t *installTracker) facesForBasenames(basenames []string) ([]installations.InstalledFontFile, error) {
	if t == nil {
		return nil, nil
	}
	var files []installations.InstalledFontFile
	for _, base := range basenames {
		base = strings.TrimSpace(base)
		if base == "" {
			continue
		}
		full := filepath.Join(t.fontDir, base)
		if _, err := os.Stat(full); err != nil {
			return nil, fmt.Errorf("tracked file missing: %s", base)
		}
		face, err := installedFaceFromPath(full, t.variantByBasename[base])
		if err != nil {
			return nil, fmt.Errorf("%s: %w", base, err)
		}
		files = append(files, face)
	}
	return files, nil
}

// persistInstallState writes complete or incomplete_install tracking for present basenames.
func (t *installTracker) persistInstallState(present []string, lastErrors []string) error {
	if t == nil || strings.TrimSpace(t.fontID) == "" {
		return nil
	}
	present = dedupeBasenameList(present)
	remaining := remainingBasenames(t.expected, present)
	files, err := t.facesForBasenames(present)
	if err != nil {
		return err
	}
	status := ""
	if len(remaining) > 0 {
		status = installations.StatusIncompleteInstall
	} else if len(files) == 0 {
		return fmt.Errorf("installation registry: no files to record")
	}
	return installations.UpsertInstallation(installations.UpsertParams{
		FontID:             t.fontID,
		CatalogName:        t.catalogName,
		InstallationSource: t.installSrc,
		Scope:              string(t.scope),
		FontGetVersion:     version.GetVersion(),
		Files:              files,
		Status:             status,
		Remaining:          remaining,
		LastErrors:         lastErrors,
	})
}

// persistRemoveState writes incomplete_remove or deletes the record when nothing remains.
func persistRemoveState(fontID, scope, fontDir string, stillPresent []string, remaining []string, lastErrors []string) error {
	fontID = strings.TrimSpace(fontID)
	if fontID == "" {
		return nil
	}
	stillPresent = dedupeBasenameList(stillPresent)
	remaining = dedupeBasenameList(remaining)
	if len(stillPresent) == 0 && len(remaining) == 0 {
		return installations.RemoveInstallation(fontID)
	}
	var files []installations.InstalledFontFile
	for _, base := range stillPresent {
		full := filepath.Join(fontDir, base)
		face, err := installedFaceFromPath(full, "")
		if err != nil {
			output.GetDebug().Warning("remove tracking metadata for %s: %v", base, err)
			files = append(files, installations.InstalledFontFile{Path: full})
			continue
		}
		files = append(files, face)
	}
	catalogName := ""
	installSrc := ""
	fontGetVer := version.GetVersion()
	if reg, err := installations.Load(); err == nil {
		if inst := reg.FindByFontID(fontID); inst != nil {
			catalogName = inst.CatalogName
			installSrc = inst.InstallationSource
			if inst.FontGetVersion != "" {
				fontGetVer = inst.FontGetVersion
			}
		}
	}
	return installations.UpsertInstallation(installations.UpsertParams{
		FontID:             fontID,
		CatalogName:        catalogName,
		InstallationSource: installSrc,
		Scope:              scope,
		FontGetVersion:     fontGetVer,
		Files:              files,
		Status:             installations.StatusIncompleteRemove,
		Remaining:          remaining,
		LastErrors:         lastErrors,
	})
}

// packageBasenamesFromRegistry returns tracked basenames for fontID under fontDir.
func packageBasenamesFromRegistry(fontID, fontDir string) []string {
	reg, err := installations.Load()
	if err != nil {
		return nil
	}
	inst := reg.FindByFontID(fontID)
	if inst == nil {
		return nil
	}
	out := inst.BasenamesForDir(fontDir)
	for _, r := range inst.Remaining {
		r = filepath.Base(strings.TrimSpace(r))
		if r == "" {
			continue
		}
		full := filepath.Join(fontDir, r)
		if _, err := os.Stat(full); err == nil {
			out = append(out, r)
		}
	}
	return dedupeBasenameList(out)
}

// reconcileTrackedPresent returns tracked Files under fontDir that still exist.
// Confirmed missing files are dropped. Unexpected stat/read errors are returned.
func reconcileTrackedPresent(fontID, fontDir string) ([]string, error) {
	fontID = strings.TrimSpace(fontID)
	if fontID == "" {
		return nil, nil
	}
	reg, err := installations.Load()
	if err != nil {
		return nil, err
	}
	inst := reg.FindByFontID(fontID)
	if inst == nil {
		return nil, nil
	}
	var present []string
	for _, base := range inst.BasenamesForDir(fontDir) {
		full := filepath.Join(fontDir, base)
		if _, err := os.Stat(full); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("reconcile tracked inventory: %s: %w", base, err)
		}
		present = append(present, base)
	}
	return dedupeBasenameList(present), nil
}
