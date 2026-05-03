package repo

import "testing"

func TestIsWebfontKitArchivePath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"Kihim_Complete/Fonts/OTF/Kihim-Regular.otf", false},
		{"Kihim_Complete/Fonts/WEB/fonts/Kihim-Regular.ttf", true},
		{"fanwood-master/Fanwood.otf", false},
		{"fanwood-master/webfonts/fanwood-webfont.ttf", true},
		{"pkg/dist/Fanwood_Text-webfont.ttf", true},
		{"Some/Font/static/webfonts/x.ttf", true},
		{"plain/Regular.ttf", false},
	}
	for _, tt := range tests {
		if got := isWebfontKitArchivePath(tt.path); got != tt.want {
			t.Errorf("isWebfontKitArchivePath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestFilterArchiveExtractedDesktopFonts(t *testing.T) {
	mixed := []string{
		"a/Fonts/WEB/fonts/Kihim-Regular.ttf",
		"a/Fonts/OTF/Kihim-Regular.otf",
	}
	got := filterArchiveExtractedDesktopFonts(mixed)
	if len(got) != 1 || got[0] != mixed[1] {
		t.Fatalf("filter mixed: got %#v want [%q]", got, mixed[1])
	}

	onlyDesktop := []string{"x/Fanwood.otf", "x/Fanwood Italic.otf"}
	got2 := filterArchiveExtractedDesktopFonts(onlyDesktop)
	if len(got2) != 2 {
		t.Fatalf("filter onlyDesktop: got %#v", got2)
	}

	onlyWeb := []string{"w/webfonts/a.ttf", "w/webfonts/b.ttf"}
	got3 := filterArchiveExtractedDesktopFonts(onlyWeb)
	if len(got3) != 2 || got3[0] != onlyWeb[0] {
		t.Fatalf("filter onlyWeb fallback: got %#v want original", got3)
	}
}
