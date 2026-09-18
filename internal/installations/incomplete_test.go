package installations

import (
	"path/filepath"
	"testing"

	"fontget/internal/testutil"
)

func TestInstallationIncompleteStatus(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	fontDir := t.TempDir()
	face := filepath.Join(fontDir, "A.ttf")
	if err := UpsertInstallation(UpsertParams{
		FontID:    "pkg.a",
		Scope:     "user",
		Files:     []InstalledFontFile{{Path: face, SFNT: SFNTSnapshot{Family: "A"}}},
		Status:    StatusIncompleteInstall,
		Remaining: []string{"B.ttf"},
	}); err != nil {
		t.Fatal(err)
	}
	reg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	inst := reg.FindByFontID("pkg.a")
	if inst == nil || inst.IsComplete() || !inst.IsIncomplete() {
		t.Fatalf("incomplete install: %+v", inst)
	}
	if err := UpsertInstallation(UpsertParams{
		FontID: "pkg.a",
		Scope:  "user",
		Files: []InstalledFontFile{
			{Path: face, SFNT: SFNTSnapshot{Family: "A"}},
			{Path: filepath.Join(fontDir, "B.ttf"), SFNT: SFNTSnapshot{Family: "B"}},
		},
	}); err != nil {
		t.Fatal(err)
	}
	reg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	inst = reg.FindByFontID("pkg.a")
	if inst == nil || !inst.IsComplete() {
		t.Fatalf("complete install: %+v", inst)
	}
}
