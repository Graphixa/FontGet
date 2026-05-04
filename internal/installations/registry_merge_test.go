package installations

import (
	"strings"
	"testing"
)

type stubMergeRow struct {
	Path, Family, FontID, License, Source string
	Categories                            []string
}

func (s *stubMergeRow) BlankFontID() bool { return strings.TrimSpace(s.FontID) == "" }
func (s *stubMergeRow) PathForMerge() string {
	return strings.TrimSpace(s.Path)
}
func (s *stubMergeRow) FamilyForMerge() string {
	return strings.TrimSpace(s.Family)
}
func (s *stubMergeRow) ApplyRepoCatalog(fontID, license string, categories []string, source string) {
	s.FontID = fontID
	s.License = license
	s.Categories = append([]string(nil), categories...)
	s.Source = source
}

func TestMergeInstallationRegistryIntoFamilyGroups_ambiguousFamilySkipped(t *testing.T) {
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
	row := &stubMergeRow{Family: "DupFam"}
	groups := map[string][]RegistryMergeMutableRow{
		"DupFam": {row},
	}
	MergeInstallationRegistryIntoFamilyGroups(groups, reg, func(string) (string, string, []string, string, bool) {
		t.Error("catalog lookup should not run when family index is ambiguous")
		return "", "", nil, "", false
	})
	if row.FontID != "" {
		t.Fatalf("ambiguous SFNT family: expected FontID still empty, got %q", row.FontID)
	}
}
