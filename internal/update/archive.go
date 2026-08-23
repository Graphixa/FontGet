package update

import (
	"fmt"
	"runtime"
)

// currentGOOS / currentGOARCH are the platform used to select a release archive.
// Tests override these so fixtures can target a fixed GOOS/GOARCH.
var (
	currentGOOS   = runtime.GOOS
	currentGOARCH = runtime.GOARCH
)

// ReleaseArchiveName returns the GoReleaser archive filename for a version and platform.
// Matches archives.name_template: fontget_{{ .Version }}_{{ .Os }}_{{ .Arch }}
// (Windows archives use .zip; others use .tar.gz).
func ReleaseArchiveName(version, goos, goarch string) string {
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("fontget_%s_%s_%s%s", version, goos, goarch, ext)
}

// ReleaseArchiveAssetSuffix returns the filename suffix that self-update matches
// on GitHub release assets for the current platform (e.g. _linux_amd64.tar.gz).
func ReleaseArchiveAssetSuffix() string {
	ext := ".tar.gz"
	if currentGOOS == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("_%s_%s%s", currentGOOS, currentGOARCH, ext)
}

func currentArchiveName(version string) string {
	return ReleaseArchiveName(version, currentGOOS, currentGOARCH)
}

func currentExecutableName() string {
	if currentGOOS == "windows" {
		return "fontget.exe"
	}
	return "fontget"
}
