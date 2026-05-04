package repo

import (
	"testing"
)

func TestLookupFontByIDInManifest_duplicateAcrossSources_priorityWins(t *testing.T) {
	m := &FontManifest{
		Sources: map[string]SourceInfo{
			"Font Squirrel": {
				Fonts: map[string]FontInfo{
					"dup.examplefont": {License: "squirrel-license", Name: "S"},
				},
			},
			"Google Fonts": {
				Fonts: map[string]FontInfo{
					"dup.examplefont": {License: "google-license", Name: "G"},
				},
			},
		},
	}
	id, info, sn, ok := lookupFontByIDInManifest(m, "dup.examplefont")
	if !ok {
		t.Fatal("expected match")
	}
	if id != "dup.examplefont" {
		t.Fatalf("id = %q", id)
	}
	if info.License != "google-license" || sn != "Google Fonts" {
		t.Fatalf("want Google Fonts winner, got src=%q lic=%q", sn, info.License)
	}

	id2, _, _, ok2 := lookupFontByIDInManifest(m, "Dup.ExampleFont")
	if !ok2 || id2 != "dup.examplefont" {
		t.Fatalf("case-insensitive lookup: ok=%v id=%q", ok2, id2)
	}
}

func TestIsFontIDInCachedManifest_errorsFalse(t *testing.T) {
	if IsFontIDInCachedManifest("") {
		t.Fatal("empty id")
	}
}
