package installations

import "testing"

func TestResolveInstallationForCatalogMergeWithIndexes(t *testing.T) {
	reg := &Registry{
		Installations: map[string]*Installation{
			"id.one": {
				FontID: "id.one",
				Families: GroupInstalledFiles([]InstalledFontFile{
					{Path: "/tmp/one.ttf", SFNT: SFNTSnapshot{Family: "Fam"}},
				}),
			},
			"id.dup.a": {
				FontID: "id.dup.a",
				Families: GroupInstalledFiles([]InstalledFontFile{
					{Path: "/tmp/dup-a.ttf", SFNT: SFNTSnapshot{Family: "DupFam"}},
				}),
			},
			"id.dup.b": {
				FontID: "id.dup.b",
				Families: GroupInstalledFiles([]InstalledFontFile{
					{Path: "/tmp/dup-b.ttf", SFNT: SFNTSnapshot{Family: "DupFam"}},
				}),
			},
			"id.unique": {
				FontID: "id.unique",
				Families: GroupInstalledFiles([]InstalledFontFile{
					{Path: "/tmp/unique.ttf", SFNT: SFNTSnapshot{Family: "OnlyFam"}},
				}),
			},
		},
	}
	byPath := reg.PathIndex()
	byFamily := reg.FamilyInstallationsIndex()

	if got := ResolveInstallationForCatalogMergeWithIndexes(byPath, byFamily, "/tmp/one.ttf", "Other"); got == nil || got.FontID != "id.one" {
		t.Fatalf("path match: got %#v", got)
	}
	if got := ResolveInstallationForCatalogMergeWithIndexes(byPath, byFamily, "", "OnlyFam"); got == nil || got.FontID != "id.unique" {
		t.Fatalf("unique family: got %#v", got)
	}
	if got := ResolveInstallationForCatalogMergeWithIndexes(byPath, byFamily, "", "DupFam"); got != nil {
		t.Fatalf("ambiguous family: expected nil, got %q", got.FontID)
	}
}
