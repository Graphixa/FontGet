package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"fontget/internal/installations"
	"fontget/internal/platform"
	"fontget/internal/repo"
	"fontget/internal/testutil"
)

func TestTryRecordInstallationRegistry_skipsFailedPackage(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)

	// Failed extract/install must never write provenance (issue #6 cleanup invariant).
	tryRecordInstallationRegistry("nerd.cascadia-code", []repo.FontFile{{Name: "Cascadia Code"}}, platform.UserScope, t.TempDir(),
		buildInstallResult(InstallStatusFailed, "Download failed", 0, 0, 1, nil, nil, 0))

	regPath := installations.RegistryPath()
	if _, err := os.Stat(regPath); !os.IsNotExist(err) {
		data, _ := os.ReadFile(regPath)
		t.Fatalf("failed package must not create registry file %q: %s", filepath.Base(regPath), data)
	}
}
