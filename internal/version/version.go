package version

import (
	"runtime/debug"
	"strings"
)

// Version information - can be overridden at build time
var (
	// Version is the current FontGet version
	Version = "dev"

	// GitCommit is the git commit hash (set at build time)
	GitCommit = "unknown"

	// BuildDate is when the binary was built (set at build time)
	BuildDate = "unknown"

	// ManifestVersion is the current manifest schema version
	ManifestVersion = "1.0"
)

// GetVersion returns a clean, user-facing version string:
// - Releases: "1.2.3"
// - Dev builds: "dev"
//
// Commit/build metadata is intentionally NOT included; use GetFullVersion().
func GetVersion() string {
	if v := strings.TrimSpace(Version); v != "" && v != "dev" {
		return normalizeVersionString(v)
	}

	// Try to get version from build info (e.g., go install ...@v1.2.3).
	if info, ok := debug.ReadBuildInfo(); ok {
		mv := strings.TrimSpace(info.Main.Version)
		if mv != "" && mv != "(devel)" {
			norm := normalizeVersionString(mv)
			if looksLikeReleaseVersion(norm) {
				return norm
			}
		}
	}

	return "dev"
}

// GetBuildID returns a short build identifier (typically a git commit hash).
// It prefers build-time injection (GitCommit) and falls back to Go build info (vcs.revision).
func GetBuildID() (id string, dirty bool) {
	if c := strings.TrimSpace(GitCommit); c != "" && c != "unknown" {
		return shortCommit(c), false
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", false
	}
	var rev string
	var modified string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			modified = s.Value
		}
	}
	if strings.TrimSpace(rev) == "" {
		return "", false
	}
	return shortCommit(rev), modified == "true"
}

// GetFullVersion returns a detailed version string
func GetFullVersion() string {
	v := GetVersion()
	if v == "" {
		v = "dev"
	}
	base := "FontGet " + v

	build, dirty := GetBuildID()
	// Stable releases should be clean and short by default: "FontGet 1.2.3".
	// Dev builds (or dirty builds) include build metadata for debugging/support.
	if v == "dev" || dirty {
		if build != "" {
			if dirty {
				return base + " (build " + build + "-dirty)"
			}
			return base + " (build " + build + ")"
		}
		if dirty {
			return base + " (dirty)"
		}
	}
	return base
}

// GetManifestVersion returns the current manifest schema version
func GetManifestVersion() string {
	return ManifestVersion
}

// normalizeVersionString normalizes a version string for display and comparison.
// It:
// - Trims whitespace
// - Strips leading 'v' prefix
// - Strips build metadata suffix (e.g., +dirty)
func normalizeVersionString(v string) string {
	v = strings.TrimSpace(v)
	// Strip leading 'v' prefix
	v = strings.TrimPrefix(v, "v")
	// Strip build metadata (+dirty, +meta, etc.)
	if idx := strings.Index(v, "+"); idx != -1 {
		v = v[:idx]
	}
	return v
}

func looksLikeReleaseVersion(v string) bool {
	// Minimal heuristic: "X.Y.Z" with digits and dots only.
	// Anything else (pseudo-versions, "(devel)", etc.) is treated as dev.
	if v == "" {
		return false
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		for _, r := range p {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

func shortCommit(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 7 {
		return s[:7]
	}
	return s
}
