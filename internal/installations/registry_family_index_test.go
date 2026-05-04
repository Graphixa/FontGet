package installations

import "testing"

func TestFamilyInstallationsIndex_ambiguousFamily_twoInstallations(t *testing.T) {
	reg := &Registry{
		Installations: map[string]*Installation{
			"a.first": {
				FontID: "a.first",
				Families: GroupInstalledFiles([]InstalledFontFile{
					{Path: "/tmp/shared.ttf", SFNT: SFNTSnapshot{Family: "Shared Fam"}},
				}),
			},
			"b.second": {
				FontID: "b.second",
				Families: GroupInstalledFiles([]InstalledFontFile{
					{Path: "/tmp/other.ttf", SFNT: SFNTSnapshot{Family: "Shared Fam"}},
				}),
			},
		},
	}
	idx := reg.FamilyInstallationsIndex()
	k := "shared fam"
	if len(idx[k]) != 2 {
		t.Fatalf("want 2 installations for ambiguous family, got %d: %+v", len(idx[k]), idx[k])
	}
}

func TestFamilyInstallationsIndex_singleInstallation(t *testing.T) {
	reg := &Registry{
		Installations: map[string]*Installation{
			"x.y": {
				FontID: "x.y",
				Families: GroupInstalledFiles([]InstalledFontFile{
					{Path: "/a.ttf", SFNT: SFNTSnapshot{Family: "OnlyFam"}},
					{Path: "/b.ttf", SFNT: SFNTSnapshot{Family: "OnlyFam"}},
				}),
			},
		},
	}
	idx := reg.FamilyInstallationsIndex()
	if len(idx["onlyfam"]) != 1 {
		t.Fatalf("got %#v", idx["onlyfam"])
	}
}
