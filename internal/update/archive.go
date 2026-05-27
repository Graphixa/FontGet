package update

import (
	"fmt"
	"runtime"
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

// ReleaseArchiveAssetSuffix returns the filename suffix that go-github-selfupdate matches
// on GitHub release assets for the current platform (e.g. _linux_amd64.tar.gz).
func ReleaseArchiveAssetSuffix() string {
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("_%s_%s%s", runtime.GOOS, runtime.GOARCH, ext)
}
