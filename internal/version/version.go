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

// GetVersion returns the current FontGet version in normalized form.
// Normalization rules:
// - Strip leading 'v' prefix (v1.2.3 -> 1.2.3)
// - Strip build metadata (+dirty, +meta)
// - Fallback to "dev" when no version info is available
func GetVersion() string {
	if Version != "dev" {
		return normalizeVersionString(Version)
	}

	// Try to get version from build info (for go install)
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "(devel)" && info.Main.Version != "" {
			return normalizeVersionString(info.Main.Version)
		}
	}

	return "dev"
}

// GetFullVersion returns a detailed version string
func GetFullVersion() string {
	version := GetVersion()

	var parts []string
	parts = append(parts, "FontGet "+version)

	if GitCommit != "unknown" {
		if len(GitCommit) > 7 {
			parts = append(parts, "commit "+GitCommit[:7])
		} else {
			parts = append(parts, "commit "+GitCommit)
		}
	}

	if BuildDate != "unknown" {
		parts = append(parts, "built "+BuildDate)
	}

	return strings.Join(parts, ", ")
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
