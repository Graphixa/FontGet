package repo

import (
	"net/url"
	"strings"
)

// Known FontGet-Sources / custom-source file-map keys, in download preference order.
// Direct desktop fonts first; among archives prefer compact tar.xz before zip/7z.
const (
	fileKeyTTF   = "ttf"
	fileKeyOTF   = "otf"
	fileKeyTarXZ = "tar_xz" // canonical FontGet-Sources key for .tar.xz
	fileKeyZIP   = "zip"
	fileKey7Z    = "7z"
)

// downloadCandidate is one ranked download URL from a source file map.
type downloadCandidate struct {
	Key string
	URL string
}

// downloadPreferenceTiers lists accepted file-map keys per preference tier.
// Earlier tiers win; within a tier, keys are tried in listed order (aliases after canonical).
var downloadPreferenceTiers = [][]string{
	{fileKeyTTF, fileKeyOTF},
	{fileKeyTarXZ, "tar.xz", "xz"}, // tar.xz / xz are aliases for tar_xz
	{fileKeyZIP},
	{fileKey7Z},
}

// normalizeDownloadFileKey lowercases and maps archive aliases to canonical keys.
func normalizeDownloadFileKey(key string) string {
	k := strings.ToLower(strings.TrimSpace(key))
	switch k {
	case "tar.xz", "xz":
		return fileKeyTarXZ
	default:
		return k
	}
}

// normalizeDownloadURLKey collapses trivial URL differences for dedupe
// (trim space, lowercase scheme/host via url.Parse when possible).
func normalizeDownloadURLKey(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return strings.ToLower(raw)
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	return u.String()
}

// rankDownloadCandidates returns preference-ordered download URLs from a source files map.
// Unknown keys are ignored. Duplicate URLs are collapsed (first occurrence wins).
func rankDownloadCandidates(files map[string]string) []downloadCandidate {
	if len(files) == 0 {
		return nil
	}

	// Lowercase key index; first spelling wins if the same key appears with different casing.
	byLower := make(map[string]string, len(files))
	for rawKey, rawURL := range files {
		u := strings.TrimSpace(rawURL)
		if u == "" {
			continue
		}
		lk := strings.ToLower(strings.TrimSpace(rawKey))
		if _, exists := byLower[lk]; exists {
			continue
		}
		byLower[lk] = u
	}

	var out []downloadCandidate
	seenURL := make(map[string]struct{}, len(byLower))
	seenCanon := make(map[string]struct{}, len(byLower))
	for _, tier := range downloadPreferenceTiers {
		for _, key := range tier {
			lk := strings.ToLower(key)
			u, ok := byLower[lk]
			if !ok {
				continue
			}
			canon := normalizeDownloadFileKey(lk)
			// One candidate per canonical format (aliases share a canon).
			if _, used := seenCanon[canon]; used {
				continue
			}
			ukey := normalizeDownloadURLKey(u)
			if ukey == "" {
				continue
			}
			if _, dup := seenURL[ukey]; dup {
				continue
			}
			seenURL[ukey] = struct{}{}
			seenCanon[canon] = struct{}{}
			out = append(out, downloadCandidate{Key: canon, URL: u})
		}
	}
	return out
}

// pickDownloadURLFromFileMap chooses a download URL from FontGet-Sources variant or top-level files
// using the shared download-format preference policy.
func pickDownloadURLFromFileMap(files map[string]string) string {
	cands := rankDownloadCandidates(files)
	if len(cands) == 0 {
		return ""
	}
	return cands[0].URL
}

// downloadCandidateURLs returns preference-ordered URLs only.
func downloadCandidateURLs(files map[string]string) []string {
	cands := rankDownloadCandidates(files)
	if len(cands) == 0 {
		return nil
	}
	out := make([]string, len(cands))
	for i, c := range cands {
		out[i] = c.URL
	}
	return out
}
