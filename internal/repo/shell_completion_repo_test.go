package repo

import (
	"testing"
)

func TestGetRepositoryForShellCompletion_WithoutManifest(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	_, err := GetRepositoryForShellCompletion()
	if err == nil {
		t.Fatal("GetRepositoryForShellCompletion: expected error when manifest/cache is missing")
	}
}
