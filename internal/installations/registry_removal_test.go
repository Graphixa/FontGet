package installations

import (
	"path/filepath"
	"testing"

	"fontget/internal/platform"
)

func TestShouldConsultRegistryForRemoval_dot(t *testing.T) {
	if !ShouldConsultRegistryForRemoval("nerd.hack", nil, nil) {
		t.Fatal("dotted name should consult registry path")
	}
}

func TestShouldConsultRegistryForRemoval_registryHitWithoutDot(t *testing.T) {
	reg := &Registry{
		Installations: map[string]*Installation{
			"nodot": {FontID: "nodot", Families: GroupInstalledFiles([]InstalledFontFile{{Path: "/x.ttf"}})},
		},
	}
	if !ShouldConsultRegistryForRemoval("nodot", reg, func(string) bool { return false }) {
		t.Fatal("existing install key without dot should consult registry")
	}
}

func TestInstallationHasBasenamesUnderDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	userFonts := filepath.Join(home, "Library", "Fonts")
	if err := RecordInstallation(RecordParams{
		FontID:      "nerd.meslo",
		CatalogName: "Meslo",
		Scope:       "user",
		Files: []InstalledFontFile{
			{
				Path: filepath.Join(userFonts, "MesloLGLNerdFont-Regular.ttf"),
				SFNT: SFNTSnapshot{Family: "MesloLGL Nerd Font"},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}
	reg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if InstallationHasBasenamesUnderDir(reg, "nerd.meslo", "/wrong/dir") {
		t.Fatal("expected false when registry paths do not match font dir")
	}
	if !InstallationHasBasenamesUnderDir(reg, "nerd.meslo", userFonts) {
		t.Fatal("expected true when paths are under font dir")
	}
}

func TestInstallationRegistryResolvable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	userFonts := filepath.Join(home, "Library", "Fonts")
	if err := RecordInstallation(RecordParams{
		FontID:      "nerd.x",
		CatalogName: "X",
		Scope:       "user",
		Files: []InstalledFontFile{
			{
				Path: filepath.Join(userFonts, "a.ttf"),
				SFNT: SFNTSnapshot{Family: "A Nerd"},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}
	reg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	dirFn := func(s platform.InstallationScope) string {
		if s == platform.UserScope {
			return userFonts
		}
		return "/System/Library/Fonts"
	}
	ok, pn := InstallationRegistryResolvable("nerd.x", reg, []platform.InstallationScope{platform.UserScope}, dirFn, repoIsFontIDNever)
	if !ok || pn != "X" {
		t.Fatalf("got ok=%v pn=%q", ok, pn)
	}
}

func repoIsFontIDNever(string) bool { return false }
