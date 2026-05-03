package repo

import (
	"path/filepath"
	"sort"
	"strings"

	"fontget/internal/output"
	"fontget/internal/platform"
)

// ArchiveExtractPreference controls how we pick installable files from an extracted archive
// when both variable (fvar) and static fonts are present. This is a maintainer switch only —
// not user-facing config; flip the default in archiveExtractPreference when product policy changes.
type ArchiveExtractPreference int

const (
	// ArchivePreferStaticFonts installs static (non-fvar) desktop cuts when available, and only
	// falls back to variable fonts when the archive has no valid static TTF/OTF left.
	ArchivePreferStaticFonts ArchiveExtractPreference = iota
	// ArchivePreferVariableFonts installs variable fonts only when any are present; otherwise statics.
	ArchivePreferVariableFonts
)

// archiveExtractPreference is the active policy for archive-based installs.
// Set to ArchivePreferVariableFonts to prefer a single VF over multiple statics when both exist.
const archiveExtractPreference = ArchivePreferStaticFonts

// archiveFontPickItem is one candidate path after extraction, before install ordering.
type archiveFontPickItem struct {
	path  string
	isVar bool
	isTTF bool
}

// applyArchiveInstallPolicy applies the final install policy on a candidate path set:
// 1) Drop common webfont-kit paths when desktop OTF/TTF remain.
// 2) Apply archiveExtractPreference (static vs variable).
// 3) Sort with .ttf before .otf, then path (case-insensitive).
func applyArchiveInstallPolicy(paths []string) []string {
	paths = filterArchiveExtractedDesktopFonts(paths)
	if len(paths) == 0 {
		return nil
	}

	items := make([]archiveFontPickItem, 0, len(paths))
	for _, p := range paths {
		isVar, err := platform.FontFileIsVariableFont(p)
		if err != nil {
			output.GetDebug().State("variable font probe failed for %q (treating as static): %v", p, err)
			isVar = false
		}
		ext := strings.ToLower(filepath.Ext(p))
		items = append(items, archiveFontPickItem{path: p, isVar: isVar, isTTF: ext == ".ttf"})
	}

	var vars, statics []archiveFontPickItem
	for _, it := range items {
		if it.isVar {
			vars = append(vars, it)
		} else {
			statics = append(statics, it)
		}
	}

	switch archiveExtractPreference {
	case ArchivePreferVariableFonts:
		if len(vars) > 0 {
			return sortExtractedFontPaths(vars)
		}
		return sortExtractedFontPaths(statics)
	default: // ArchivePreferStaticFonts
		if len(statics) > 0 {
			return sortExtractedFontPaths(statics)
		}
		if len(vars) > 0 {
			return sortExtractedFontPaths(vars)
		}
		return nil
	}
}

// sortExtractedFontPaths returns paths sorted with .ttf before .otf, then by path (case-insensitive).
func sortExtractedFontPaths(items []archiveFontPickItem) []string {
	sort.Slice(items, func(i, j int) bool {
		if items[i].isTTF != items[j].isTTF {
			return items[i].isTTF
		}
		return strings.ToLower(items[i].path) < strings.ToLower(items[j].path)
	})
	out := make([]string, len(items))
	for i := range items {
		out[i] = items[i].path
	}
	return out
}

// isWebfontKitArchivePath detects paths inside common @font-face / web kit trees so we do not
// install duplicate web formats when the same archive already contains desktop OTF/TTF.
func isWebfontKitArchivePath(path string) bool {
	s := strings.ToLower(filepath.ToSlash(path))
	switch {
	case strings.Contains(s, "/webfonts/"):
		return true
	case strings.Contains(s, "/fonts/web/"): // e.g. Fontshare .../Fonts/WEB/fonts/...
		return true
	case strings.Contains(s, "/font/web/"):
		return true
	case strings.Contains(s, "/static/webfonts/"):
		return true
	}
	base := strings.ToLower(filepath.Base(path))
	if strings.Contains(base, "-webfont.") {
		return true
	}
	if strings.Contains(base, "_webfont.") {
		return true
	}
	return false
}

// filterArchiveExtractedDesktopFonts drops webfont-kit paths when at least one desktop
// candidate remains. If everything would be removed, returns paths unchanged.
func filterArchiveExtractedDesktopFonts(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if !isWebfontKitArchivePath(p) {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return paths
	}
	return out
}
