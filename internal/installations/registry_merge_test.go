package installations

import "testing"

func TestResolveInstallationForCatalogMerge_ambiguousFamilySkipped(t *testing.T) {
	reg := &Registry{
		Installations: map[string]*Installation{
			"id.one": {
				FontID: "id.one",
				Families: GroupInstalledFiles([]InstalledFontFile{
					{Path: "/tmp/one.ttf", SFNT: SFNTSnapshot{Family: "DupFam"}},
				}),
			},
			"id.two": {
				FontID: "id.two",
				Families: GroupInstalledFiles([]InstalledFontFile{
					{Path: "/tmp/two.ttf", SFNT: SFNTSnapshot{Family: "DupFam"}},
				}),
			},
		},
	}
	if got := ResolveInstallationForCatalogMerge(reg, "", "DupFam"); got != nil {
		t.Fatalf("ambiguous SFNT family: expected nil, got %q", got.FontID)
	}
}

func TestResolveInstallationForCatalogMerge_pathWins(t *testing.T) {
	reg := &Registry{
		Installations: map[string]*Installation{
			"id.one": {
				FontID: "id.one",
				Families: GroupInstalledFiles([]InstalledFontFile{
					{Path: "/tmp/one.ttf", SFNT: SFNTSnapshot{Family: "Fam"}},
				}),
			},
		},
	}
	got := ResolveInstallationForCatalogMerge(reg, "/tmp/one.ttf", "Fam")
	if got == nil || got.FontID != "id.one" {
		t.Fatalf("path lookup: got %#v", got)
	}
}
