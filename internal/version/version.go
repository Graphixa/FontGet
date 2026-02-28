package version

import (
	"os/exec"
	"runtime/debug"
	"strings"
	"sync"
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

var (
	// Cached git commit hash (detected at runtime)
	runtimeGitCommit string
	runtimeGitOnce   sync.Once
)

// GetVersion returns the current FontGet version in normalized form.
// Normalization rules:
// - Strip leading 'v' prefix (v1.2.3 -> 1.2.3)
// - Strip build metadata (+dirty, +meta) EXCEPT for dev builds with commit hash
// - Fallback to "dev" when no version info is available
func GetVersion() string {
	if Version != "dev" {
		// Dev build with date and commit (e.g., "dev-20260228020445-6c41181") - return as-is
		if strings.HasPrefix(Version, "dev-") {
			return Version
		}
		// For dev builds with commit hash (e.g., "dev+ae04b20", "2.0.0-dev+ae04b20"), preserve the full string
		if strings.Contains(Version, "-dev+") || strings.HasPrefix(Version, "dev+") {
			return strings.TrimPrefix(Version, "v") // Only strip 'v' prefix, keep rest
		}
		return normalizeVersionString(Version)
	}

	// Try to get version from build info (for go install)
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "(devel)" && info.Main.Version != "" {
			return normalizeVersionString(info.Main.Version)
		}
	}

	// For local builds, try to get commit hash (from build-time or runtime)
	if Version == "dev" {
		commit := getGitCommit()
		if commit != "" && commit != "unknown" {
			// Use commit hash in build metadata format: dev+abc1234
			// This is SemVer compliant and informative
			if len(commit) >= 7 {
				return "dev+" + commit[:7]
			}
			return "dev+" + commit
		}
	}

	return "dev"
}

// getGitCommit returns the git commit hash, preferring build-time value,
// falling back to runtime detection if not set.
func getGitCommit() string {
	// If set at build time, use that
	if GitCommit != "unknown" && GitCommit != "" {
		return GitCommit
	}

	// Otherwise, try to detect at runtime (once, cached)
	runtimeGitOnce.Do(func() {
		// Try to get git commit hash from the repository
		// This works when the binary is run from within the git repo
		cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
		// Set working directory to the binary's location or current directory
		output, err := cmd.Output()
		if err == nil && len(output) > 0 {
			runtimeGitCommit = strings.TrimSpace(string(output))
		} else {
			runtimeGitCommit = ""
		}
	})

	return runtimeGitCommit
}

// GetFullVersion returns a detailed version string
func GetFullVersion() string {
	version := GetVersion()

	var parts []string
	parts = append(parts, "FontGet "+version)

	commit := getGitCommit()
	if commit != "" && commit != "unknown" {
		if len(commit) > 7 {
			parts = append(parts, "commit "+commit[:7])
		} else {
			parts = append(parts, "commit "+commit)
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
