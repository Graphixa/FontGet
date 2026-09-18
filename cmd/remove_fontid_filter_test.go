package cmd

import "testing"

func TestSfntFamilyAllowedForFontID(t *testing.T) {
	cases := []struct {
		family string
		id     string
		want   bool
	}{
		{"Iosevka", "nerd.iosevka", false},
		{"Iosevka Nerd Font", "nerd.iosevka", true},
		{"Iosevka", "fontsource.iosevka", true},
		{"Iosevka Nerd Font", "fontsource.iosevka", false},
		{"Roboto", "google.roboto", true},
	}
	for _, tc := range cases {
		if got := sfntFamilyAllowedForFontID(tc.family, tc.id); got != tc.want {
			t.Fatalf("sfntFamilyAllowedForFontID(%q, %q)=%v want %v", tc.family, tc.id, got, tc.want)
		}
	}
}

func TestCheckFontMatchesFontID_nerdGate(t *testing.T) {
	if checkFontMatchesFontID("Iosevka", "nerd.iosevka", nil) {
		t.Fatal("plain Iosevka must not match nerd.iosevka")
	}
	if checkFontMatchesFontID("Iosevka Nerd Font", "fontsource.iosevka", nil) {
		t.Fatal("nerd face must not match fontsource id")
	}
}
