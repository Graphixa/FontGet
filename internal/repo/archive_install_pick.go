package repo

import (
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"fontget/internal/output"
)

// PickInstallableFontPathsFromArchive chooses installable paths from validated extract paths:
// known-source path rules (when prefix matches), otherwise directory buckets with soft desktop/web
// scoring and low-confidence fallback to the full set; then applyArchiveInstallPolicy (web filter,
// static vs variable, TTF-before-OTF sort).
func PickInstallableFontPathsFromArchive(valid []string, archiveSourcePrefix string) []string {
	return PickInstallableFontPathsFromArchiveWithContext(valid, ArchiveSelectionContext{
		SourcePrefix: archiveSourcePrefix,
	})
}

// PickInstallableFontPathsFromArchiveWithContext is like PickInstallableFontPathsFromArchive
// but passes FontID/FontName for source-specific selectors (e.g. Nerd Fonts).
func PickInstallableFontPathsFromArchiveWithContext(valid []string, ctx ArchiveSelectionContext) []string {
	candidates := pickArchiveCandidates(valid, ctx.SourcePrefix, ctx.FontName, ctx.FontID)
	return applyArchiveInstallPolicy(candidates)
}

func pickArchiveCandidates(paths []string, prefix, fontName, fontID string) []string {
	if len(paths) == 0 {
		return nil
	}
	p := strings.ToLower(strings.TrimSpace(prefix))

	// Nerd Fonts: fail closed — never fall back to agnostic on multi-family ZIPs (e.g. Noto.zip).
	if p == "nerd" || p == "nerdfonts" {
		picked, stem, ok := tryNerdFontsKnownPaths(paths, fontName, fontID)
		if ok {
			output.GetDebug().State("archive pick: branch=known:nerd stem=%s fontName=%q fontID=%q matched=%d",
				stem, fontName, fontID, len(picked))
			return picked
		}
		output.GetDebug().State("archive pick: branch=known:nerd matched=0 (no agnostic fallback) stem=%s fontName=%q fontID=%q",
			stem, fontName, fontID)
		return nil
	}

	if picked, ok := tryKnownSourcePaths(p, paths); ok {
		output.GetDebug().State("archive pick: branch=known:%s count=%d (before policy)", p, len(picked))
		return picked
	}
	out := pickAgnosticArchiveCandidates(paths)
	output.GetDebug().State("archive pick: branch=agnostic count=%d (before policy)", len(out))
	return out
}

// tryKnownSourcePaths returns paths selected by hard-coded layout rules for a source prefix.
// The bool is true when rules ran and produced a non-empty set (no fallback needed for "known" branch).
// Nerd Fonts are handled separately in pickArchiveCandidates (fail-closed).
func tryKnownSourcePaths(prefixLower string, paths []string) ([]string, bool) {
	switch prefixLower {
	case "fontshare":
		return tryFontshareKnownPaths(paths)
	case "league":
		return tryLeagueKnownPaths(paths)
	default:
		return nil, false
	}
}

func tryFontshareKnownPaths(paths []string) ([]string, bool) {
	var otfTree, ttfTree []string
	for _, p := range paths {
		s := strings.ToLower(filepath.ToSlash(p))
		switch {
		case strings.Contains(s, "/fonts/otf/"):
			otfTree = append(otfTree, p)
		case strings.Contains(s, "/fonts/ttf/"):
			ttfTree = append(ttfTree, p)
		}
	}
	if len(otfTree) > 0 {
		return otfTree, true
	}
	if len(ttfTree) > 0 {
		return ttfTree, true
	}
	return nil, false
}

func tryLeagueKnownPaths(paths []string) ([]string, bool) {
	var noWeb []string
	for _, p := range paths {
		s := strings.ToLower(filepath.ToSlash(p))
		if strings.Contains(s, "/webfonts/") {
			continue
		}
		noWeb = append(noWeb, p)
	}
	if len(noWeb) > 0 {
		return noWeb, true
	}
	return nil, false
}

// tryNerdFontsKnownPaths selects font files belonging to one logical Nerd Font family
// from large multi-family archives (e.g. Noto.zip). Matching tries FontID stem first, then
// cleaned FontName.
//
// Nerd Fonts often abbreviate long family names in filenames for SFNT limits, e.g.
// "Noto Sans Mono" → NotoSansMNerdFont-*.ttf (not NotoSansMonoNerdFont-*). Matching
// accepts exact stems and longest abbreviated prefixes of the stem.
func tryNerdFontsKnownPaths(paths []string, fontName, fontID string) (matched []string, stem string, ok bool) {
	for _, candidate := range nerdFontFamilyStemCandidates(fontName, fontID) {
		hits := matchNerdPathsForStem(paths, candidate)
		if len(hits) > 0 {
			return hits, candidate, true
		}
		if stem == "" {
			stem = candidate
		}
	}
	return nil, stem, false
}

// matchNerdPathsForStem returns archive paths whose filename family token matches stem,
// preferring the longest family token (so notosansmono → NotoSansM*, not NotoSans*).
func matchNerdPathsForStem(paths []string, stem string) []string {
	if stem == "" {
		return nil
	}
	type hit struct {
		path      string
		familyKey string
	}
	var hits []hit
	bestLen := -1
	for _, p := range paths {
		base := archiveEntryBasename(p)
		fam := nerdFamilyKeyFromBasename(base)
		if fam == "" || !nerdFamilyKeyMatchesStem(fam, stem) {
			continue
		}
		hits = append(hits, hit{path: p, familyKey: fam})
		if len(fam) > bestLen {
			bestLen = len(fam)
		}
	}
	if bestLen < 0 {
		return nil
	}
	out := make([]string, 0, len(hits))
	for _, h := range hits {
		if len(h.familyKey) == bestLen {
			out = append(out, h.path)
		}
	}
	return out
}

// archiveEntryBasename returns the final path segment for archive-relative paths
// that always use forward slashes (filepath.Base is wrong for "/" on Windows).
func archiveEntryBasename(p string) string {
	return strings.ToLower(path.Base(filepath.ToSlash(p)))
}

// nerdFamilyKeyFromBasename extracts the compacted family token before "nerdfont"
// (or "nerd") in a Nerd Fonts release basename.
// "NotoSansMNerdFontMono-Regular.ttf" → "notosansm"
func nerdFamilyKeyFromBasename(baseLower string) string {
	baseLower = strings.TrimSuffix(baseLower, path.Ext(baseLower))
	compact := compactAlphanumeric(baseLower)
	for _, marker := range []string{"nerdfont", "nerd"} {
		if i := strings.Index(compact, marker); i > 0 {
			return compact[:i]
		}
	}
	return ""
}

// nerdFamilyKeyMatchesStem reports whether a filename family key belongs to stem.
// Exact match, or key is an abbreviation prefix of stem (Nerd Fonts name shortening).
// A longer/more-specific key than stem does not match (NotoSansM must not match stem notosans).
func nerdFamilyKeyMatchesStem(familyKey, stem string) bool {
	if familyKey == "" || stem == "" {
		return false
	}
	if familyKey == stem {
		return true
	}
	// Abbreviation: NotoSansM for stem notosansmono.
	if len(familyKey) < len(stem) && strings.HasPrefix(stem, familyKey) {
		return true
	}
	return false
}

// nerdFontFamilyStemCandidates returns stems to try in order: FontID first (stable, style-free),
// then cleaned FontName. Duplicates are omitted.
func nerdFontFamilyStemCandidates(fontName, fontID string) []string {
	var out []string
	seen := make(map[string]struct{}, 2)
	add := func(s string) {
		s = finalizeNerdStem(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	add(stemFromFontID(fontID))
	add(stemFromFontName(fontName))
	return out
}

// nerdFontFamilyStem returns the preferred stem (FontID if present, else cleaned FontName).
// Prefer nerdFontFamilyStemCandidates when matching archives.
func nerdFontFamilyStem(fontName, fontID string) string {
	cands := nerdFontFamilyStemCandidates(fontName, fontID)
	if len(cands) == 0 {
		return ""
	}
	return cands[0]
}

func stemFromFontID(fontID string) string {
	id := strings.ToLower(strings.TrimSpace(fontID))
	if id == "" {
		return ""
	}
	if i := strings.IndexByte(id, '.'); i >= 0 && i+1 < len(id) {
		id = id[i+1:]
	}
	return compactAlphanumeric(id)
}

func stemFromFontName(fontName string) string {
	cleaned := stripTrailingFontStyleWords(fontName)
	return compactAlphanumeric(cleaned)
}

func finalizeNerdStem(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	// Strip trailing "nerdfont" / "nerd" if present in the catalog name / ID.
	s = strings.TrimSuffix(s, "nerdfonts")
	s = strings.TrimSuffix(s, "nerdfont")
	s = strings.TrimSuffix(s, "nerd")
	return s
}

// nerdStyleSuffixes are trailing style tokens stripped from FontName before stemming.
var nerdStyleSuffixes = []string{
	"extralight", "semibold", "extrabold",
	"condensed", "oblique", "regular", "medium", "italic",
	"light", "thin", "black", "book", "bold",
}

func stripTrailingFontStyleWords(name string) string {
	fields := strings.Fields(strings.TrimSpace(name))
	for len(fields) > 1 {
		last := strings.ToLower(fields[len(fields)-1])
		last = strings.Trim(last, ",;")
		matched := false
		for _, suf := range nerdStyleSuffixes {
			if last == suf {
				fields = fields[:len(fields)-1]
				matched = true
				break
			}
		}
		if !matched {
			break
		}
	}
	return strings.Join(fields, " ")
}

func compactAlphanumeric(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Agnostic bucket scoring: if the best directory's average score is below
// agnosticBucketWinningScoreMin, or too close to the runner-up (see
// agnosticBucketScoreSeparationMin), we return the full path set instead of
// only the best bucket.
const (
	agnosticBucketScoreSeparationMin = 8.0
	agnosticBucketWinningScoreMin    = 0.0
	agnosticWebPathPenalty           = -30
	agnosticDesktopPathBonus         = 10
	agnosticOTFSegmentBonus          = 5
	agnosticStaticNonWebfontsSegment = 2
)

// archivePathSoftDesktopScore is used only in the agnostic branch: higher means more likely desktop install.
func archivePathSoftDesktopScore(path string) int {
	s := strings.ToLower(filepath.ToSlash(path))
	score := 0
	if isWebfontKitArchivePath(path) {
		score += agnosticWebPathPenalty
	} else {
		score += agnosticDesktopPathBonus
	}
	if strings.Contains(s, "/fonts/otf/") || strings.Contains(s, "/otf/") {
		score += agnosticOTFSegmentBonus
	}
	if strings.Contains(s, "/static/") && !strings.Contains(s, "/webfonts/") {
		score += agnosticStaticNonWebfontsSegment
	}
	return score
}

type archiveDirBucket struct {
	dir    string
	paths  []string
	avg    float64
	maxOne int
}

func pickAgnosticArchiveCandidates(paths []string) []string {
	if len(paths) <= 1 {
		return paths
	}
	byDir := make(map[string][]string)
	for _, p := range paths {
		d := filepath.Dir(p)
		byDir[d] = append(byDir[d], p)
	}
	if len(byDir) == 1 {
		return paths
	}
	buckets := make([]archiveDirBucket, 0, len(byDir))
	for dir, ps := range byDir {
		var sum int
		maxOne := -1 << 30
		for _, p := range ps {
			v := archivePathSoftDesktopScore(p)
			sum += v
			if v > maxOne {
				maxOne = v
			}
		}
		avg := float64(sum) / float64(len(ps))
		buckets = append(buckets, archiveDirBucket{dir: dir, paths: ps, avg: avg, maxOne: maxOne})
	}
	sort.Slice(buckets, func(i, j int) bool {
		if buckets[i].avg != buckets[j].avg {
			return buckets[i].avg > buckets[j].avg
		}
		if buckets[i].maxOne != buckets[j].maxOne {
			return buckets[i].maxOne > buckets[j].maxOne
		}
		return strings.ToLower(buckets[i].dir) < strings.ToLower(buckets[j].dir)
	})
	best := buckets[0]
	lowConfidence := false
	if len(buckets) > 1 {
		second := buckets[1]
		if best.avg-second.avg < agnosticBucketScoreSeparationMin {
			lowConfidence = true
		}
	}
	if best.avg < agnosticBucketWinningScoreMin {
		lowConfidence = true
	}
	if lowConfidence {
		output.GetDebug().State("archive pick: agnostic:fallback full_set=%d (best_bucket_avg=%.2f)", len(paths), best.avg)
		return paths
	}
	output.GetDebug().State("archive pick: agnostic:bucket dir=%s avg=%.2f files=%d", best.dir, best.avg, len(best.paths))
	return best.paths
}
