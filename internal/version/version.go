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

// GetVersion returns the current FontGet version
func GetVersion() string {
	if Version != "dev" {
		return Version
	}

	// Try to get version from build info (for go install)
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "(devel)" && info.Main.Version != "" {
			return info.Main.Version
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
