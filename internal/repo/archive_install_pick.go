package repo

import (
	"path/filepath"
	"sort"
	"strings"

	"fontget/internal/output"
)

// PickInstallableFontPathsFromArchive chooses installable paths from validated extract paths:
// known-source path rules (when prefix matches), otherwise directory buckets with soft desktop/web
// scoring and low-confidence fallback to the full set; then applyArchiveInstallPolicy (web filter,
// static vs variable, TTF-before-OTF sort).
func PickInstallableFontPathsFromArchive(valid []string, archiveSourcePrefix string) []string {
	candidates := pickArchiveCandidates(valid, archiveSourcePrefix)
	return applyArchiveInstallPolicy(candidates)
}

func pickArchiveCandidates(paths []string, prefix string) []string {
	if len(paths) == 0 {
		return nil
	}
	p := strings.ToLower(strings.TrimSpace(prefix))
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
