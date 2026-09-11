package repo

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestSizeExceedsLimit(t *testing.T) {
	cases := []struct {
		size  uint64
		limit int64
		want  bool
	}{
		{0, 0, false},
		{1, 0, true},
		{100, 100, false},
		{101, 100, true},
		{0, -1, true},
		{math.MaxInt64, math.MaxInt64, false},
		{math.MaxInt64 + 1, math.MaxInt64, true},
	}
	for _, tc := range cases {
		if got := sizeExceedsLimit(tc.size, tc.limit); got != tc.want {
			t.Errorf("sizeExceedsLimit(%d, %d)=%v want %v", tc.size, tc.limit, got, tc.want)
		}
	}
}

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

func TestExtractZIP_nerdPackageMode_renamedFamiliesAndCompleteNoto(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "package.zip")
	destDir := filepath.Join(dir, "out")

	// Catalogue names differ from installed family tokens (OFL remaps) and Noto is multi-family.
	files := map[string][]byte{
		"CaskaydiaCoveNerdFont-Regular.ttf": bytes.Repeat([]byte("c"), 40),
		"SauceCodeProNerdFont-Regular.ttf":  bytes.Repeat([]byte("s"), 40),
		"IosevkaNerdFont-Regular.ttf":       bytes.Repeat([]byte("i"), 20),
		"IosevkaNerdFontMono-Regular.ttf":   bytes.Repeat([]byte("i"), 20),
		"IosevkaNerdFontPropo-Regular.ttf":  bytes.Repeat([]byte("i"), 20),
		"NotoSansMNerdFont-Regular.ttf":     bytes.Repeat([]byte("m"), 30),
		"NotoSansNerdFont-Regular.ttf":      bytes.Repeat([]byte("S"), 30),
		"NotoSerifNerdFont-Regular.ttf":     bytes.Repeat([]byte("R"), 30),
		"webfonts/IgnoreMe-Regular.ttf":     bytes.Repeat([]byte("w"), 30),
		"readme.txt":                        []byte("not a font"),
	}
	writeZipWithEntries(t, archivePath, files)

	policy := ExtractionPolicy{
		MaxFileBytes:      200,
		MaxTotalBytes:     500,
		MaxArchiveEntries: 100,
		MaxSelectedFiles:  100,
	}

	cases := []struct {
		name   string
		fontID string
		want   int
	}{
		{"cascadia renamed", "nerd.cascadia-code", 8}, // all desktop fonts; webfonts filtered
		{"source-code renamed", "nerd.source-code-pro", 8},
		{"iosevka all variants", "nerd.iosevka", 8},
		{"noto complete package", "nerd.noto", 8},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := filepath.Join(destDir, tc.name)
			paths, err := ExtractArchiveWithOptions(archivePath, out, &ExtractOptions{
				Policy: &policy,
				Selection: &ArchiveSelectionContext{
					SourcePrefix: "nerd",
					FontName:     "Unrelated Catalogue Name",
					FontID:       tc.fontID,
				},
			})
			if err != nil {
				t.Fatalf("extract: %v", err)
			}
			if len(paths) != tc.want {
				t.Fatalf("want %d extracted, got %d: %v", tc.want, len(paths), paths)
			}
			for _, p := range paths {
				if strings.Contains(strings.ToLower(filepath.ToSlash(p)), "/webfonts/") {
					t.Fatalf("webfont should not be extracted: %q", p)
				}
			}
		})
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
			FontName:     "Noto",
			FontID:       "nerd.noto",
		},
	})
	if !errors.Is(err, ErrArchiveTotalLimit) {
		t.Fatalf("got %v want ErrArchiveTotalLimit", err)
	}
	if _, statErr := os.Stat(destDir); !os.IsNotExist(statErr) {
		entries, _ := os.ReadDir(destDir)
		if len(entries) > 0 {
			t.Fatalf("planning failure must leave dest empty, got %d entries", len(entries))
		}
	}
}

func TestPlanSelectedArchiveBudget_boundaries(t *testing.T) {
	policy := DefaultExtractionPolicy()
	exact := splitBudgetEntries("exact", uint64(policy.MaxTotalBytes), uint64(policy.MaxFileBytes))
	if err := planSelectedArchiveBudget(exact, policy); err != nil {
		t.Fatalf("exact 1.5 GiB should pass: %v", err)
	}

	over := splitBudgetEntries("over", uint64(policy.MaxTotalBytes)+1, uint64(policy.MaxFileBytes))
	if err := planSelectedArchiveBudget(over, policy); !errors.Is(err, ErrArchiveTotalLimit) {
		t.Fatalf("one over limit: got %v", err)
	}

	// Declared total just above the former 1 GiB default must succeed under 1.5 GiB.
	former1GiB := uint64(1 << 30)
	aboveOld := splitBudgetEntries("old", former1GiB+1, uint64(policy.MaxFileBytes))
	if err := planSelectedArchiveBudget(aboveOld, policy); err != nil {
		t.Fatalf("above old 1 GiB should pass under 1.5 GiB: %v", err)
	}

	// Overflow-safe aggregate rejection (three MaxInt64 sizes cannot sum in uint64).
	overflow := []ArchiveEntry{
		{Name: "a.ttf", NormalizedPath: "a.ttf", UncompressedSize: math.MaxInt64},
		{Name: "b.ttf", NormalizedPath: "b.ttf", UncompressedSize: math.MaxInt64},
		{Name: "c.ttf", NormalizedPath: "c.ttf", UncompressedSize: math.MaxInt64},
	}
	wide := ExtractionPolicy{
		MaxFileBytes:  math.MaxInt64,
		MaxTotalBytes: math.MaxInt64,
	}
	if err := planSelectedArchiveBudget(overflow, wide); !errors.Is(err, ErrArchiveTotalLimit) {
		t.Fatalf("overflow: got %v", err)
	}

	perFile := []ArchiveEntry{{
		Name: "huge.ttf", NormalizedPath: "huge.ttf", UncompressedSize: uint64(defaultMaxExtractFileBytes) + 1,
	}}
	if err := planSelectedArchiveBudget(perFile, policy); !errors.Is(err, ErrArchiveEntryTooLarge) {
		t.Fatalf("per-file: got %v", err)
	}
}

func splitBudgetEntries(prefix string, total, maxFile uint64) []ArchiveEntry {
	var out []ArchiveEntry
	remaining := total
	i := 0
	for remaining > 0 {
		chunk := maxFile
		if remaining < chunk {
			chunk = remaining
		}
		name := prefix + string(rune('a'+i)) + ".ttf"
		out = append(out, ArchiveEntry{Name: name, NormalizedPath: name, UncompressedSize: chunk})
		remaining -= chunk
		i++
	}
	return out
}

func TestDefaultExtractionPolicy_onePointFiveGiB(t *testing.T) {
	p := DefaultExtractionPolicy()
	if p.MaxTotalBytes != 1536<<20 {
		t.Fatalf("MaxTotalBytes=%d want %d", p.MaxTotalBytes, 1536<<20)
	}
}

func TestPickArchiveCandidates_nerdPackageMode(t *testing.T) {
	paths := []string{
		"CaskaydiaCoveNerdFont-Regular.ttf",
		"SauceCodeProNerdFont-Regular.ttf",
		"NotoSansNerdFont-Regular.ttf",
		"NotoSerifNerdFont-Regular.ttf",
	}
	out := pickArchiveCandidates(paths, "nerd", "Cascadia Code", "nerd.cascadia-code")
	if len(out) != len(paths) {
		t.Fatalf("package mode should return all fonts, got %#v", out)
	}
}

func TestExtractTARGZ_nerdPackageMode_parityWithZIP(t *testing.T) {
	dir := t.TempDir()
	files := map[string][]byte{
		"CaskaydiaCoveNerdFont-Regular.ttf": []byte("caskaydia"),
		"SauceCodeProNerdFont-Regular.ttf":  []byte("sauce"),
		"webfonts/Skip-Regular.ttf":         []byte("web"),
	}

	zipPath := filepath.Join(dir, "pkg.zip")
	writeZipWithEntries(t, zipPath, files)
	tarPath := filepath.Join(dir, "pkg.tar.gz")
	writeTarGzWithEntries(t, tarPath, files)

	policy := ExtractionPolicy{
		MaxFileBytes:      1000,
		MaxTotalBytes:     1000,
		MaxArchiveEntries: 100,
		MaxSelectedFiles:  100,
	}
	sel := &ArchiveSelectionContext{
		SourcePrefix: "nerd",
		FontID:       "nerd.cascadia-code",
		FontName:     "Cascadia Code",
	}

	var zipProgressTotal int
	zipOut := filepath.Join(dir, "zip-out")
	zipPaths, err := ExtractArchiveWithOptions(zipPath, zipOut, &ExtractOptions{
		Policy:    &policy,
		Selection: sel,
		OnFontFileExtracted: func(done, total int) {
			zipProgressTotal = total
		},
	})
	if err != nil {
		t.Fatalf("zip: %v", err)
	}

	var tarProgressTotal int
	tarOut := filepath.Join(dir, "tar-out")
	tarPaths, err := ExtractArchiveWithOptions(tarPath, tarOut, &ExtractOptions{
		Policy:    &policy,
		Selection: sel,
		OnFontFileExtracted: func(done, total int) {
			tarProgressTotal = total
		},
	})
	if err != nil {
		t.Fatalf("tar.gz: %v", err)
	}

	if len(zipPaths) != 2 || len(tarPaths) != 2 {
		t.Fatalf("want 2 each, zip=%d tar=%d", len(zipPaths), len(tarPaths))
	}
	// ZIP inspects first so progress total is known; Nerd TAR package mode is single-pass (-1).
	if zipProgressTotal != 2 {
		t.Fatalf("zip progress total=%d want 2", zipProgressTotal)
	}
	if tarProgressTotal != -1 {
		t.Fatalf("tar package-mode progress total=%d want -1", tarProgressTotal)
	}

	zipBases := basenames(zipPaths)
	tarBases := basenames(tarPaths)
	sort.Strings(zipBases)
	sort.Strings(tarBases)
	if strings.Join(zipBases, ",") != strings.Join(tarBases, ",") {
		t.Fatalf("parity mismatch zip=%v tar=%v", zipBases, tarBases)
	}
}

func TestExtractTARGZ_planningFailsLeavesNoOutput(t *testing.T) {
	dir := t.TempDir()
	tarPath := filepath.Join(dir, "big.tar.gz")
	destDir := filepath.Join(dir, "out")
	writeTarGzWithEntries(t, tarPath, map[string][]byte{
		"ANerdFont-Regular.ttf": bytes.Repeat([]byte("a"), 80),
		"BNerdFont-Regular.ttf": bytes.Repeat([]byte("b"), 80),
	})
	policy := ExtractionPolicy{
		MaxFileBytes:      200,
		MaxTotalBytes:     100,
		MaxArchiveEntries: 100,
		MaxSelectedFiles:  100,
	}
	_, err := ExtractArchiveWithOptions(tarPath, destDir, &ExtractOptions{
		Policy:    &policy,
		Selection: &ArchiveSelectionContext{SourcePrefix: "nerd", FontID: "nerd.x"},
	})
	if !errors.Is(err, ErrArchiveTotalLimit) {
		t.Fatalf("got %v", err)
	}
	// Package-mode single-pass may write then fail; written fonts are cleaned up on error.
	if entries, _ := os.ReadDir(destDir); len(entries) > 0 {
		t.Fatalf("dest should be empty after budget failure cleanup, got %v", entries)
	}
}

func TestInspectTARXZ_corruptLeavesNoStaging(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.tar.xz")
	// XZ magic but truncated/corrupt payload.
	if err := os.WriteFile(p, []byte{0xFD, 0x37, 0x7A, 0x58, 0x5A, 0x00, 0x00}, 0644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "out")
	_, err := ExtractArchiveWithOptions(p, dest, &ExtractOptions{
		Selection: &ArchiveSelectionContext{SourcePrefix: "nerd", FontID: "nerd.x"},
	})
	if err == nil {
		t.Fatal("expected corrupt archive error")
	}
	if entries, _ := os.ReadDir(dest); len(entries) > 0 {
		t.Fatalf("corrupt extract must not leave staging files: %v", entries)
	}
}

func writeTarGzWithEntries(t *testing.T, path string, files map[string][]byte) {
	t.Helper()
	var tarBuf bytes.Buffer
	tw := tar.NewWriter(&tarBuf)
	for name, data := range files {
		h := &tar.Header{Name: name, Mode: 0644, Size: int64(len(data))}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	var gzBuf bytes.Buffer
	zw := gzip.NewWriter(&gzBuf)
	if _, err := zw.Write(tarBuf.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, gzBuf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
}

func basenames(paths []string) []string {
	out := make([]string, len(paths))
	for i, p := range paths {
		out[i] = strings.ToLower(filepath.Base(p))
	}
	return out
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
