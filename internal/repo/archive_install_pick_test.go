package repo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTryKnownSourcePaths_fontshare(t *testing.T) {
	web := filepath.Join("Kihim_Complete", "Fonts", "WEB", "fonts", "Kihim-Regular.ttf")
	otf := filepath.Join("Kihim_Complete", "Fonts", "OTF", "Kihim-Regular.otf")
	got, ok := tryKnownSourcePaths("fontshare", []string{web, otf})
	if !ok || len(got) != 1 || got[0] != otf {
		t.Fatalf("fontshare: got %#v ok=%v want [%q]", got, ok, otf)
	}
	got2, ok2 := tryKnownSourcePaths("fontshare", []string{web})
	if ok2 || got2 != nil {
		t.Fatalf("fontshare WEB-only: got %#v ok=%v want nil,false", got2, ok2)
	}
}

func TestTryKnownSourcePaths_league(t *testing.T) {
	root := filepath.Join("fanwood-master", "Fanwood.otf")
	web := filepath.Join("fanwood-master", "webfonts", "fanwood-webfont.ttf")
	got, ok := tryKnownSourcePaths("league", []string{root, web})
	if !ok || len(got) != 1 || got[0] != root {
		t.Fatalf("league mixed: got %#v ok=%v", got, ok)
	}
	got2, ok2 := tryKnownSourcePaths("league", []string{web})
	if ok2 || got2 != nil {
		t.Fatalf("league web-only: got %#v ok=%v want nil,false", got2, ok2)
	}
	got3, ok3 := tryKnownSourcePaths("league", []string{root})
	if !ok3 || len(got3) != 1 {
		t.Fatalf("league desktop-only: got %#v ok=%v", got3, ok3)
	}
}

func TestTryKnownSourcePaths_unknownPrefix(t *testing.T) {
	p := filepath.Join("a", "b.otf")
	got, ok := tryKnownSourcePaths("squirrel", []string{p})
	if ok || got != nil {
		t.Fatalf("unknown: got %#v ok=%v", got, ok)
	}
}

func touchFont(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPickAgnosticArchiveCandidates_separationFallback(t *testing.T) {
	tmp := t.TempDir()
	// Two buckets with similar average scores → full-set fallback.
	d1 := filepath.Join(tmp, "desktop")
	d2 := filepath.Join(tmp, "almost")
	touchFont(t, filepath.Join(d1, "a.otf"))
	touchFont(t, filepath.Join(d1, "b.otf"))
	touchFont(t, filepath.Join(d2, "c.otf"))
	paths := []string{
		filepath.Join(d1, "a.otf"),
		filepath.Join(d1, "b.otf"),
		filepath.Join(d2, "c.otf"),
	}
	out := pickAgnosticArchiveCandidates(paths)
	if len(out) != len(paths) {
		t.Fatalf("expected full fallback len=%d got %d", len(paths), len(out))
	}
}

func TestPickInstallableFontPathsFromArchive_fontshareThenPolicy(t *testing.T) {
	tmp := t.TempDir()
	web := filepath.Join(tmp, "Kihim_Complete", "Fonts", "WEB", "fonts", "Kihim-Regular.ttf")
	otf := filepath.Join(tmp, "Kihim_Complete", "Fonts", "OTF", "Kihim-Regular.otf")
	touchFont(t, web)
	touchFont(t, otf)
	got := PickInstallableFontPathsFromArchive([]string{web, otf}, "fontshare")
	if len(got) != 1 || got[0] != otf {
		t.Fatalf("got %#v want [%q]", got, otf)
	}
}

func TestPickInstallableFontPathsFromArchive_unknownUsesAgnostic(t *testing.T) {
	tmp := t.TempDir()
	good := filepath.Join(tmp, "good", "a.otf")
	bad := filepath.Join(tmp, "webfonts", "b.otf")
	touchFont(t, good)
	touchFont(t, bad)
	got := PickInstallableFontPathsFromArchive([]string{good, bad}, "")
	if len(got) != 1 || got[0] != good {
		t.Fatalf("got %#v want [%q]", got, good)
	}
}
