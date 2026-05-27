package update

import (
	"strings"
	"testing"
)

func TestReleaseArchiveName(t *testing.T) {
	got := ReleaseArchiveName("1.2.3", "linux", "amd64")
	want := "fontget_1.2.3_linux_amd64.tar.gz"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}

	got = ReleaseArchiveName("1.2.3", "windows", "arm64")
	want = "fontget_1.2.3_windows_arm64.zip"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestReleaseArchiveAssetSuffixMatchesGoReleaserName(t *testing.T) {
	name := ReleaseArchiveName("9.9.9", "linux", "amd64")
	if !strings.HasSuffix(name, "_linux_amd64.tar.gz") {
		t.Fatalf("archive %q should end with _linux_amd64.tar.gz for self-update matching", name)
	}
}
