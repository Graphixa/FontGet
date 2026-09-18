package sources

import (
	"strings"
	"testing"
)

func TestDefaultSources_nerdFontsV2(t *testing.T) {
	if !strings.HasSuffix(NerdFontsURL, "/nerd-fonts-v2.json") {
		t.Fatalf("NerdFontsURL = %q want …/nerd-fonts-v2.json", NerdFontsURL)
	}
	nerd := DefaultSources()["Nerd Fonts"]
	if nerd.Filename != "nerd-fonts-v2.json" {
		t.Fatalf("Filename = %q", nerd.Filename)
	}
	if nerd.URL != NerdFontsURL {
		t.Fatalf("URL = %q want %q", nerd.URL, NerdFontsURL)
	}
}
