package repo

import (
	"archive/zip"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSafeArchiveRelPath(t *testing.T) {
	cases := []struct {
		in   string
		ok   bool
		want string
	}{
		{"normal/path/font.ttf", true, "normal/path/font.ttf"},
		{"../font.ttf", false, ""},
		{"../../font.ttf", false, ""},
		{"/absolute/font.ttf", false, ""},
		{"/etc/font.ttf", false, ""},
		{`C:\Windows\Fonts\font.ttf`, false, ""},
		{"nested/../../../font.ttf", false, ""},
		{"fonts/../../font.ttf", false, ""},
		{"", false, ""},
		{".", false, ""},
	}
	for _, tc := range cases {
		got, ok := safeArchiveRelPath(tc.in)
		if ok != tc.ok {
			t.Errorf("safeArchiveRelPath(%q) ok=%v want %v (got %q)", tc.in, ok, tc.ok, got)
			continue
		}
		if ok && got != tc.want {
			t.Errorf("safeArchiveRelPath(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestCheckDestinationCollisions(t *testing.T) {
	if err := checkDestinationCollisions([]string{"fonts/A.ttf", "fonts/B.ttf"}); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if err := checkDestinationCollisions([]string{"fonts/A.ttf", "fonts/a.ttf"}); !errors.Is(err, ErrArchivePathCollision) {
		t.Fatalf("case collision: got %v", err)
	}
	if err := checkDestinationCollisions([]string{"fonts/A.ttf", "fonts/A.ttf"}); !errors.Is(err, ErrArchivePathCollision) {
		t.Fatalf("exact dup: got %v", err)
	}
}

func TestCopyExtractedFile_exactAndOverTotalBudget(t *testing.T) {
	dir := t.TempDir()
	policy := ExtractionPolicy{
		MaxFileBytes:      1000,
		MaxTotalBytes:     100,
		MaxArchiveEntries: 100,
		MaxSelectedFiles:  100,
	}

	// Exact limit: 100 bytes into empty budget.
	p1 := filepath.Join(dir, "exact.ttf")
	n, err := copyExtractedFile(p1, bytes.NewReader(bytes.Repeat([]byte("a"), 100)), "exact.ttf", policy, 0)
	if err != nil || n != 100 {
		t.Fatalf("exact: n=%d err=%v", n, err)
	}

	// One more byte must fail against remaining=0.
	p2 := filepath.Join(dir, "over.ttf")
	_, err = copyExtractedFile(p2, bytes.NewReader([]byte("x")), "over.ttf", policy, 100)
	if !errors.Is(err, ErrArchiveTotalLimit) {
		t.Fatalf("over total: got %v", err)
	}
	if _, statErr := os.Stat(p2); !os.IsNotExist(statErr) {
		t.Fatalf("partial over-limit file should be removed")
	}
}

func TestCopyExtractedFile_streamExceedsRemaining(t *testing.T) {
	dir := t.TempDir()
	policy := ExtractionPolicy{
		MaxFileBytes:      1000,
		MaxTotalBytes:     50,
		MaxArchiveEntries: 100,
		MaxSelectedFiles:  100,
	}
	p := filepath.Join(dir, "stream.ttf")
	// Declared size under remaining, but stream tries to write more.
	_, err := copyExtractedFileWithDeclaredSize(p, bytes.NewReader(bytes.Repeat([]byte("b"), 200)), "stream.ttf", 10, policy, 0)
	if !errors.Is(err, ErrArchiveTotalLimit) {
		t.Fatalf("got %v want ErrArchiveTotalLimit", err)
	}
	if _, statErr := os.Stat(p); !os.IsNotExist(statErr) {
		t.Fatalf("partial file should be removed")
	}
}

func TestCopyExtractedFile_perFileLimit(t *testing.T) {
	dir := t.TempDir()
	policy := ExtractionPolicy{
		MaxFileBytes:      20,
		MaxTotalBytes:     1000,
		MaxArchiveEntries: 100,
		MaxSelectedFiles:  100,
	}
	p := filepath.Join(dir, "big.ttf")
	_, err := copyExtractedFileWithDeclaredSize(p, bytes.NewReader(bytes.Repeat([]byte("c"), 50)), "big.ttf", 50, policy, 0)
	if !errors.Is(err, ErrArchiveEntryTooLarge) {
		t.Fatalf("got %v want ErrArchiveEntryTooLarge", err)
	}
}

func writeZipWithEntries(t *testing.T, path string, files map[string][]byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, data := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestExtractZIP_selectiveNerdFonts_largeArchiveSmallSelection(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "Noto.zip")
	destDir := filepath.Join(dir, "out")

	// Real Nerd Fonts v3 naming: Noto Sans Mono is abbreviated to NotoSansMNerdFont*.
	files := map[string][]byte{
		"NotoSansMNerdFont-Regular.ttf":     bytes.Repeat([]byte("m"), 40),
		"NotoSansMNerdFontMono-Regular.ttf": bytes.Repeat([]byte("m"), 40),
		"NotoSansNerdFont-Regular.ttf":      bytes.Repeat([]byte("S"), 500),
		"NotoSerifNerdFont-Regular.ttf":     bytes.Repeat([]byte("R"), 500),
	}
	writeZipWithEntries(t, archivePath, files)

	policy := ExtractionPolicy{
		MaxFileBytes:      200,
		MaxTotalBytes:     100, // selected=80 fits; full archive would not
		MaxArchiveEntries: 100,
		MaxSelectedFiles:  100,
	}
	paths, err := ExtractArchiveWithOptions(archivePath, destDir, &ExtractOptions{
		Policy: &policy,
		Selection: &ArchiveSelectionContext{
			SourcePrefix: "nerd",
			FontName:     "Noto Sans Mono",
			FontID:       "nerd.noto-sans-mono",
		},
	})
	if err != nil {
		t.Fatalf("selective extract: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("want 2 extracted files, got %d: %v", len(paths), paths)
	}
	for _, p := range paths {
		base := strings.ToLower(filepath.Base(p))
		if !strings.HasPrefix(base, "notosansmnerdfont") {
			t.Fatalf("unexpected extracted file %q", base)
		}
	}
	if _, err := os.Stat(filepath.Join(destDir, "NotoSansNerdFont-Regular.ttf")); !os.IsNotExist(err) {
		t.Fatalf("unselected filler should not be extracted")
	}
}

func TestExtractZIP_selectedSubsetExceedsBudget(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "Noto.zip")
	destDir := filepath.Join(dir, "out")

	files := map[string][]byte{
		"NotoSansMNerdFont-Regular.ttf": bytes.Repeat([]byte("m"), 80),
		"NotoSansMNerdFont-Bold.ttf":    bytes.Repeat([]byte("m"), 80),
	}
	writeZipWithEntries(t, archivePath, files)

	policy := ExtractionPolicy{
		MaxFileBytes:      200,
		MaxTotalBytes:     100, // selected total 160 > 100
		MaxArchiveEntries: 100,
		MaxSelectedFiles:  100,
	}
	_, err := ExtractArchiveWithOptions(archivePath, destDir, &ExtractOptions{
		Policy: &policy,
		Selection: &ArchiveSelectionContext{
			SourcePrefix: "nerd",
			FontName:     "Noto Sans Mono",
		},
	})
	if !errors.Is(err, ErrArchiveTotalLimit) {
		t.Fatalf("got %v want ErrArchiveTotalLimit", err)
	}
}

func TestExtractZIP_pathCollision(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "collide.zip")
	destDir := filepath.Join(dir, "out")

	// Two names that collide after case folding.
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for _, name := range []string{"Fonts/A.ttf", "Fonts/a.ttf"} {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte("data"))
	}
	_ = zw.Close()
	_ = f.Close()

	policy := DefaultExtractionPolicy()
	policy.MaxTotalBytes = 1000
	_, err = ExtractArchiveWithOptions(archivePath, destDir, &ExtractOptions{
		Policy:    &policy,
		Selection: &ArchiveSelectionContext{},
	})
	if !errors.Is(err, ErrArchivePathCollision) {
		t.Fatalf("got %v want ErrArchivePathCollision", err)
	}
}

func TestExtractZIP_maxSelectedFiles(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "many.zip")
	destDir := filepath.Join(dir, "out")

	files := map[string][]byte{
		"a.ttf": []byte("1"),
		"b.ttf": []byte("2"),
		"c.ttf": []byte("3"),
	}
	writeZipWithEntries(t, archivePath, files)

	policy := ExtractionPolicy{
		MaxFileBytes:      100,
		MaxTotalBytes:     1000,
		MaxArchiveEntries: 100,
		MaxSelectedFiles:  2,
	}
	_, err := ExtractArchiveWithOptions(archivePath, destDir, &ExtractOptions{
		Policy:    &policy,
		Selection: &ArchiveSelectionContext{},
	})
	if !errors.Is(err, ErrArchiveSelectedFileLimit) {
		t.Fatalf("got %v want ErrArchiveSelectedFileLimit", err)
	}
}

func TestInspectZIP_maxArchiveEntries(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "many.zip")
	files := map[string][]byte{
		"a.ttf": []byte("1"),
		"b.ttf": []byte("2"),
		"c.ttf": []byte("3"),
	}
	writeZipWithEntries(t, archivePath, files)

	policy := ExtractionPolicy{
		MaxFileBytes:      100,
		MaxTotalBytes:     1000,
		MaxArchiveEntries: 2,
		MaxSelectedFiles:  100,
	}
	_, err := InspectZIPWithPolicy(archivePath, policy)
	if !errors.Is(err, ErrArchiveEntryCountLimit) {
		t.Fatalf("got %v want ErrArchiveEntryCountLimit", err)
	}
}

func TestExtractZIP_unsafePathRejected(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "bad.zip")
	destDir := filepath.Join(dir, "out")

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("../escape.ttf")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("x"))
	_ = zw.Close()
	_ = f.Close()

	_, err = ExtractArchiveWithOptions(archivePath, destDir, &ExtractOptions{
		Selection: &ArchiveSelectionContext{},
	})
	// Unsafe entries are skipped; with nothing safe left, selection yields no fonts.
	if err == nil {
		t.Fatal("expected error when archive has only unsafe font paths")
	}
	if !strings.Contains(err.Error(), "no font files selected") && !errors.Is(err, ErrArchiveUnsafePath) {
		t.Fatalf("got %v", err)
	}
}

func TestValidateFontFile_headers(t *testing.T) {
	dir := t.TempDir()

	ttf := filepath.Join(dir, "ok.ttf")
	if err := os.WriteFile(ttf, []byte{0x00, 0x01, 0x00, 0x00, 0x00}, 0644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateFontFile(ttf); err != nil {
		t.Fatalf("ttf: %v", err)
	}

	otto := filepath.Join(dir, "ok.otf")
	if err := os.WriteFile(otto, []byte("OTTO"+string([]byte{0})), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateFontFile(otto); err != nil {
		t.Fatalf("otto: %v", err)
	}

	ttc := filepath.Join(dir, "ok.ttc")
	if err := os.WriteFile(ttc, []byte("ttcf"+string([]byte{0})), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateFontFile(ttc); err != nil {
		t.Fatalf("ttc: %v", err)
	}

	zipNamed := filepath.Join(dir, "evil.ttf")
	if err := os.WriteFile(zipNamed, []byte("PK\x03\x04junk"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateFontFile(zipNamed); !errors.Is(err, ErrInvalidFontPayload) {
		t.Fatalf("zip-as-ttf: got %v", err)
	}

	garbage := filepath.Join(dir, "garbage.otf")
	if err := os.WriteFile(garbage, []byte("notafont"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateFontFile(garbage); !errors.Is(err, ErrInvalidFontPayload) {
		t.Fatalf("garbage: got %v", err)
	}

	woff := filepath.Join(dir, "web.ttf")
	if err := os.WriteFile(woff, []byte("wOFF"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateFontFile(woff); !errors.Is(err, ErrInvalidFontPayload) {
		t.Fatalf("woff: got %v", err)
	}
}

func TestTryNerdFontsKnownPaths(t *testing.T) {
	paths := []string{
		"NotoSansMNerdFont-Regular.ttf",
		"NotoSansMNerdFontMono-Regular.ttf",
		"NotoSansNerdFont-Regular.ttf",
		"NotoSerifNerdFont-Regular.ttf",
	}
	got, stem, ok := tryNerdFontsKnownPaths(paths, "Noto Sans Mono", "nerd.noto-sans-mono")
	if !ok || len(got) != 2 || stem != "notosansmono" {
		t.Fatalf("got %#v stem=%q ok=%v", got, stem, ok)
	}
	// Nerd is fail-closed via pickArchiveCandidates, not tryKnownSourcePaths.
	out := pickArchiveCandidates(paths, "nerd", "Noto Sans Mono", "")
	if len(out) != 2 {
		t.Fatalf("via pickArchiveCandidates: got %#v", out)
	}
}

func TestTryNerdFontsKnownPaths_abbreviatedNotoSansM(t *testing.T) {
	// Regression: real Noto.zip uses NotoSansMNerdFont* for Noto Sans Mono (SFNT name shortening).
	paths := []string{
		"NotoSansMNerdFont-Regular.ttf",
		"NotoSansMNerdFont-Bold.ttf",
		"NotoSansMNerdFontMono-Regular.ttf",
		"NotoSansMNerdFontPropo-Regular.ttf",
		"NotoSansNerdFont-Regular.ttf",
		"NotoSansNerdFontPropo-CondensedThin.ttf",
		"NotoSerifNerdFont-Regular.ttf",
	}
	got, stem, ok := tryNerdFontsKnownPaths(paths, "Noto Sans Mono", "nerd.noto-sans-mono")
	if !ok || stem != "notosansmono" {
		t.Fatalf("stem=%q ok=%v got=%#v", stem, ok, got)
	}
	if len(got) != 4 {
		t.Fatalf("want 4 NotoSansM* files, got %d: %#v", len(got), got)
	}
	for _, p := range got {
		b := strings.ToLower(filepath.Base(p))
		if !strings.HasPrefix(b, "notosansmnerdfont") {
			t.Fatalf("selected non-Mono file %q", p)
		}
		// Must not pick plain Noto Sans (notosansnerdfont without trailing m before nerd).
		if strings.HasPrefix(b, "notosansnerdfont") && !strings.HasPrefix(b, "notosansmnerdfont") {
			t.Fatalf("selected Noto Sans instead of Mono: %q", p)
		}
	}
}

func TestTryNerdFontsKnownPaths_fontIDWinsOverStyleBearingName(t *testing.T) {
	paths := []string{
		"NotoSansMNerdFont-Regular.ttf",
		"NotoSansMNerdFontMono-Regular.ttf",
		"NotoSansNerdFont-Regular.ttf",
		"NotoSerifNerdFont-Regular.ttf",
	}
	got, stem, ok := tryNerdFontsKnownPaths(paths, "Noto Sans Mono Regular", "nerd.noto-sans-mono")
	if !ok || stem != "notosansmono" || len(got) != 2 {
		t.Fatalf("got %#v stem=%q ok=%v", got, stem, ok)
	}
}

func TestTryNerdFontsKnownPaths_nameOnlyStripsRegular(t *testing.T) {
	paths := []string{
		"NotoSansMNerdFont-Regular.ttf",
		"NotoSansNerdFont-Regular.ttf",
	}
	got, stem, ok := tryNerdFontsKnownPaths(paths, "Noto Sans Mono Regular", "")
	if !ok || stem != "notosansmono" || len(got) != 1 {
		t.Fatalf("got %#v stem=%q ok=%v", got, stem, ok)
	}
}

func TestTryNerdFontsKnownPaths_notoSansDoesNotTakeMono(t *testing.T) {
	paths := []string{
		"NotoSansMNerdFont-Regular.ttf",
		"NotoSansNerdFont-Regular.ttf",
		"NotoSansNerdFont-Bold.ttf",
	}
	got, stem, ok := tryNerdFontsKnownPaths(paths, "Noto Sans", "nerd.noto-sans")
	if !ok || stem != "notosans" || len(got) != 2 {
		t.Fatalf("got %#v stem=%q ok=%v", got, stem, ok)
	}
	for _, p := range got {
		if strings.Contains(strings.ToLower(filepath.Base(p)), "notosansmnerdfont") {
			t.Fatalf("Mono file selected for Sans stem: %q", p)
		}
	}
}

func TestNerdFamilyKeyMatchesStem_noCrossFamily(t *testing.T) {
	// Full Mono name must not match short Sans stem.
	if nerdFamilyKeyMatchesStem(nerdFamilyKeyFromBasename("notosansmononerdfont-regular.ttf"), "notosans") {
		t.Fatal("notosans must not match NotoSansMonoNerdFont")
	}
	// Exact Mono stem matches full Mono name.
	if !nerdFamilyKeyMatchesStem(nerdFamilyKeyFromBasename("notosansmononerdfont-regular.ttf"), "notosansmono") {
		t.Fatal("notosansmono should match NotoSansMonoNerdFont")
	}
	// Abbreviated Mono name matches Mono stem.
	if !nerdFamilyKeyMatchesStem(nerdFamilyKeyFromBasename("notosansmnerdfont-regular.ttf"), "notosansmono") {
		t.Fatal("notosansmono should match abbreviated NotoSansMNerdFont")
	}
}

func TestMatchNerdPathsForStem_longestAbbreviationWins(t *testing.T) {
	paths := []string{
		"NotoSansNerdFont-Regular.ttf",
		"NotoSansMNerdFont-Regular.ttf",
	}
	got := matchNerdPathsForStem(paths, "notosansmono")
	if len(got) != 1 || !strings.Contains(got[0], "NotoSansM") {
		t.Fatalf("want only NotoSansM, got %#v", got)
	}
}

func TestPickArchiveCandidates_nerdFailClosed(t *testing.T) {
	paths := []string{
		"NotoSansNerdFont-Regular.ttf",
		"NotoSerifNerdFont-Regular.ttf",
		"OtherFamilyNerdFont-Regular.ttf",
	}
	out := pickArchiveCandidates(paths, "nerd", "NoSuchFont", "nerd.no-such-font")
	if out != nil {
		t.Fatalf("expected nil (fail closed), got %#v", out)
	}
}

func TestNerdFontFamilyStem(t *testing.T) {
	if got := nerdFontFamilyStem("Noto Sans Mono Regular", "nerd.noto-sans-mono"); got != "notosansmono" {
		t.Fatalf("prefer id: %q", got)
	}
	if got := nerdFontFamilyStem("Noto Sans Mono", ""); got != "notosansmono" {
		t.Fatalf("name stem: %q", got)
	}
	if got := nerdFontFamilyStem("", "nerd.noto-sans-mono"); got != "notosansmono" {
		t.Fatalf("id stem: %q", got)
	}
	if got := nerdFontFamilyStem("Noto Sans Mono Regular", ""); got != "notosansmono" {
		t.Fatalf("strip Regular: %q", got)
	}
}

func TestStripTrailingFontStyleWords(t *testing.T) {
	if got := stripTrailingFontStyleWords("Noto Sans Mono Regular"); got != "Noto Sans Mono" {
		t.Fatalf("got %q", got)
	}
	if got := stripTrailingFontStyleWords("Fira Code Bold Italic"); got != "Fira Code" {
		t.Fatalf("got %q", got)
	}
}

func TestArchiveEntryBasename_forwardSlashOnWindows(t *testing.T) {
	got := archiveEntryBasename("nested/dir/NotoSansMNerdFont-Regular.ttf")
	if got != "notosansmnerdfont-regular.ttf" {
		t.Fatalf("got %q", got)
	}
}
