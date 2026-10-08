package platform

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fontget/internal/backupmeta"
	"fontget/internal/fontkey"
)

const (
	localIngestMaxDepth      = 16
	localIngestMaxZipMembers = 10000
	localIngestMaxFileBytes  = 200 << 20 // 200 MiB
)

// LocalFontCandidate is one font file discovered for local add ingest.
type LocalFontCandidate struct {
	DiskPath  string // absolute path for loose files (staged path after StageLocalCandidate)
	ZipPath   string // archive path when FromZip
	ZipEntry  string // entry name inside ZipPath (forward slashes)
	Basename  string
	Size      int64
	FromZip   bool
	LooseFile bool
	Depth     int

	// OriginDiskPath is the loose-file origin, resolved to its canonical target on staging.
	// Empty for zip-sourced candidates.
	// Used to restore destination-overlapping inputs after failed force install.
	OriginDiskPath string

	SHA256   string
	Family   string
	Style    string
	FullName string
	// MetaReady means Family/Style/FullName were already read, so later passes must not reopen the file.
	MetaReady bool
}

// LocalIngestResult is the outcome of discover + dedupe for one local path.
type LocalIngestResult struct {
	Kept             []LocalFontCandidate
	HashDupesSkipped int
	FaceDupesSkipped int
	ConflictsSkipped int
	NestedZipFonts   int
	FontGetBackup    bool
	Warnings         []string
}

// ErrLocalCollectionUnsupported is returned for a direct .ttc/.otc local add path.
var ErrLocalCollectionUnsupported = fmt.Errorf("font collections (.ttc and .otc) are not supported for local installation")

// IsLocalInstallFontExt reports whether ext is an installable font for local add.
func IsLocalInstallFontExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".ttf", ".otf":
		return true
	default:
		return false
	}
}

// IsLocalCollectionFontExt reports whether ext is a font collection (not supported for local add).
func IsLocalCollectionFontExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".ttc", ".otc":
		return true
	default:
		return false
	}
}

// DedupeLocalCandidates applies hash, face, and basename conflict dedupe to candidates.
func DedupeLocalCandidates(cands []LocalFontCandidate) (kept []LocalFontCandidate, hashSkip, faceSkip, conflictSkip int, warnings []string, err error) {
	return dedupeLocalCandidates(cands)
}

// DiscoverAndDedupeLocalFonts discovers fonts under path (file, dir, or zip) and dedupes them.
func DiscoverAndDedupeLocalFonts(path string) (*LocalIngestResult, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}

	var (
		cands         []LocalFontCandidate
		fontGetBackup bool
		nestedZip     int
		warnings      []string
	)

	ext := filepath.Ext(abs)
	switch {
	case info.IsDir():
		cands, nestedZip, warnings, err = discoverLocalDir(abs)
	case strings.EqualFold(ext, ".zip"):
		cands, fontGetBackup, warnings, err = discoverLocalZip(abs, false)
	case IsLocalCollectionFontExt(ext):
		return nil, ErrLocalCollectionUnsupported
	case IsLocalInstallFontExt(ext):
		cands = []LocalFontCandidate{{
			DiskPath:       abs,
			OriginDiskPath: abs,
			Basename:       filepath.Base(abs),
			Size:           info.Size(),
			LooseFile:      true,
			FromZip:        false,
			Depth:          0,
		}}
	default:
		return nil, fmt.Errorf("not a font file, folder, or zip: %s", abs)
	}
	if err != nil {
		return nil, err
	}
	if len(cands) == 0 {
		return nil, fmt.Errorf("no font files found in %s", abs)
	}

	kept, hashSkip, faceSkip, conflictSkip, moreWarn, err := dedupeLocalCandidates(cands)
	if err != nil {
		return nil, err
	}
	warnings = append(warnings, moreWarn...)

	return &LocalIngestResult{
		Kept:             kept,
		HashDupesSkipped: hashSkip,
		FaceDupesSkipped: faceSkip,
		ConflictsSkipped: conflictSkip,
		NestedZipFonts:   nestedZip,
		FontGetBackup:    fontGetBackup,
		Warnings:         warnings,
	}, nil
}

func discoverLocalDir(root string) ([]LocalFontCandidate, int, []string, error) {
	var (
		cands     []LocalFontCandidate
		nestedZip int
		warnings  []string
	)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		depth := 0
		if rel != "." {
			depth = strings.Count(filepath.ToSlash(rel), "/") + 1
		}
		name := d.Name()
		if d.IsDir() {
			if shouldSkipLocalDir(name) {
				return filepath.SkipDir
			}
			if depth > localIngestMaxDepth {
				return filepath.SkipDir
			}
			return nil
		}
		if shouldSkipLocalFile(name) {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(name))
		if ext == ".zip" {
			zipCands, _, zipWarn, zipErr := discoverLocalZip(path, true)
			warnings = append(warnings, zipWarn...)
			if zipErr != nil {
				warnings = append(warnings, fmt.Sprintf("skipped nested zip %s: %v", path, zipErr))
				return nil
			}
			for i := range zipCands {
				zipCands[i].Depth = depth + zipCands[i].Depth
			}
			nestedZip += len(zipCands)
			cands = append(cands, zipCands...)
			return nil
		}
		if IsLocalCollectionFontExt(ext) {
			warnings = append(warnings, fmt.Sprintf("skipped unsupported file %s", path))
			return nil
		}
		if !IsLocalInstallFontExt(ext) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("skipped %s: %v", path, err))
			return nil
		}
		if info.Size() > localIngestMaxFileBytes {
			warnings = append(warnings, fmt.Sprintf("skipped oversized font %s", path))
			return nil
		}
		cands = append(cands, LocalFontCandidate{
			DiskPath:       path,
			OriginDiskPath: path,
			Basename:       name,
			Size:           info.Size(),
			LooseFile:      true,
			FromZip:        false,
			Depth:          depth,
		})
		return nil
	})
	return cands, nestedZip, warnings, err
}

func discoverLocalZip(zipPath string, nestedFromFolder bool) ([]LocalFontCandidate, bool, []string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, false, nil, err
	}
	defer r.Close()

	fontGetBackup := backupmeta.IsBackupComment(r.Comment)
	var warnings []string
	var cands []LocalFontCandidate
	if len(r.File) > localIngestMaxZipMembers {
		return nil, fontGetBackup, nil, fmt.Errorf("zip has too many entries (%d)", len(r.File))
	}
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := filepath.ToSlash(f.Name)
		base := filepath.Base(name)
		if shouldSkipLocalFile(base) || strings.HasPrefix(name, "__MACOSX/") {
			continue
		}
		ext := strings.ToLower(filepath.Ext(base))
		if ext == ".zip" {
			if !nestedFromFolder {
				warnings = append(warnings, fmt.Sprintf("skipped nested zip member %s", name))
			}
			continue
		}
		if IsLocalCollectionFontExt(ext) {
			warnings = append(warnings, fmt.Sprintf("skipped unsupported file %s", name))
			continue
		}
		if !IsLocalInstallFontExt(ext) {
			continue
		}
		// Guard before uint64->int64 (same pattern as repo.sizeExceedsLimit).
		uSize := f.UncompressedSize64
		if uSize > math.MaxInt64 || int64(uSize) > localIngestMaxFileBytes {
			warnings = append(warnings, fmt.Sprintf("skipped oversized zip entry %s", name))
			continue
		}
		depth := strings.Count(name, "/")
		cands = append(cands, LocalFontCandidate{
			ZipPath:   zipPath,
			ZipEntry:  name,
			Basename:  base,
			Size:      int64(uSize),
			LooseFile: false,
			FromZip:   true,
			Depth:     depth,
		})
	}
	return cands, fontGetBackup, warnings, nil
}

func shouldSkipLocalDir(name string) bool {
	switch strings.ToLower(name) {
	case "__macosx", ".git", "node_modules", ".svn":
		return true
	default:
		return false
	}
}

func shouldSkipLocalFile(name string) bool {
	switch name {
	case ".DS_Store", "Thumbs.db":
		return true
	default:
		return false
	}
}

func dedupeLocalCandidates(cands []LocalFontCandidate) (kept []LocalFontCandidate, hashSkip, faceSkip, conflictSkip int, warnings []string, err error) {
	sortLocalCandidates(cands)

	// Hash only when (size, basename) collide.
	type sizeName struct {
		size int64
		base string
	}
	buckets := make(map[sizeName][]int)
	for i, c := range cands {
		key := sizeName{c.Size, strings.ToLower(c.Basename)}
		buckets[key] = append(buckets[key], i)
	}
	for _, idxs := range buckets {
		if len(idxs) < 2 {
			continue
		}
		for _, i := range idxs {
			sum, herr := hashLocalCandidate(cands[i])
			if herr != nil {
				warnings = append(warnings, fmt.Sprintf("hash failed for %s: %v", CandidateLabel(cands[i]), herr))
				continue
			}
			cands[i].SHA256 = sum
		}
	}

	afterHash := make([]LocalFontCandidate, 0, len(cands))
	seenHash := make(map[string]bool)
	for _, c := range cands {
		if c.SHA256 != "" {
			if seenHash[c.SHA256] {
				hashSkip++
				continue
			}
			seenHash[c.SHA256] = true
		}
		afterHash = append(afterHash, c)
	}

	// SFNT face metadata for survivors. One zip stays open for every member.
	withMeta := make([]LocalFontCandidate, 0, len(afterHash))
	zips := newZipCache()
	defer zips.Close()
	for _, c := range afterHash {
		if c.MetaReady {
			withMeta = append(withMeta, c)
			continue
		}
		md, merr := metadataForLocalCandidate(c, zips)
		if merr != nil {
			warnings = append(warnings, fmt.Sprintf("skipped unreadable font %s: %v", CandidateLabel(c), merr))
			continue
		}
		c.Family = preferredFamily(md)
		c.Style = preferredStyle(md)
		c.FullName = strings.TrimSpace(md.FullName)
		c.MetaReady = true
		withMeta = append(withMeta, c)
	}

	afterFace := make([]LocalFontCandidate, 0, len(withMeta))
	seenFace := make(map[string]bool)
	for _, c := range withMeta {
		fk := faceIdentityKey(c.Family, c.Style)
		if fk == "|" {
			afterFace = append(afterFace, c)
			continue
		}
		if seenFace[fk] {
			faceSkip++
			continue
		}
		seenFace[fk] = true
		afterFace = append(afterFace, c)
	}

	seenBase := make(map[string]LocalFontCandidate)
	for _, c := range afterFace {
		baseKey := strings.ToLower(c.Basename)
		if prev, ok := seenBase[baseKey]; ok {
			conflictSkip++
			warnings = append(warnings, fmt.Sprintf(
				"skipped %s (SFNT %q %q) conflicts with kept %s (SFNT %q %q): same filename, different font",
				CandidateLabel(c), c.Family, c.Style, CandidateLabel(prev), prev.Family, prev.Style,
			))
			continue
		}
		seenBase[baseKey] = c
		kept = append(kept, c)
	}
	return kept, hashSkip, faceSkip, conflictSkip, warnings, nil
}

func sortLocalCandidates(cands []LocalFontCandidate) {
	sort.SliceStable(cands, func(i, j int) bool {
		a, b := cands[i], cands[j]
		if a.LooseFile != b.LooseFile {
			return a.LooseFile // loose first
		}
		if a.Depth != b.Depth {
			return a.Depth < b.Depth
		}
		la, lb := CandidateLabel(a), CandidateLabel(b)
		return la < lb
	})
}

// CandidateLabel returns a human-readable path for a local candidate.
func CandidateLabel(c LocalFontCandidate) string {
	if c.LooseFile {
		return c.DiskPath
	}
	return c.ZipPath + "!" + c.ZipEntry
}

func faceIdentityKey(family, style string) string {
	return fontkey.Key(family) + "|" + fontkey.Key(style)
}

func preferredFamily(md *FontMetadata) string {
	if md == nil {
		return ""
	}
	if fam := strings.TrimSpace(md.TypographicFamily); fam != "" {
		return fam
	}
	return strings.TrimSpace(md.FamilyName)
}

func preferredStyle(md *FontMetadata) string {
	if md == nil {
		return ""
	}
	if style := strings.TrimSpace(md.TypographicStyle); style != "" {
		return style
	}
	return strings.TrimSpace(md.StyleName)
}

// openZipMember opens zipPath and returns the archive reader plus the named entry.
// Caller must close the returned *zip.ReadCloser (which also closes the archive).
func openZipMember(zipPath, entry string) (*zip.ReadCloser, *zip.File, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, nil, err
	}
	for _, f := range r.File {
		if filepath.ToSlash(f.Name) == entry {
			return r, f, nil
		}
	}
	_ = r.Close()
	return nil, nil, fmt.Errorf("zip entry not found: %s", entry)
}

func hashLocalCandidate(c LocalFontCandidate) (string, error) {
	h := sha256.New()
	if c.LooseFile {
		f, err := os.Open(c.DiskPath)
		if err != nil {
			return "", err
		}
		defer f.Close()
		if _, err := io.Copy(h, f); err != nil {
			return "", err
		}
		return hex.EncodeToString(h.Sum(nil)), nil
	}
	zr, zf, err := openZipMember(c.ZipPath, c.ZipEntry)
	if err != nil {
		return "", err
	}
	defer zr.Close()
	rc, err := zf.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()
	if _, err := io.Copy(h, io.LimitReader(rc, localIngestMaxFileBytes+1)); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func metadataForLocalCandidate(c LocalFontCandidate, zips *zipCache) (*FontMetadata, error) {
	if c.LooseFile {
		return ExtractFontMetadata(c.DiskPath)
	}
	zf, err := zips.entry(c.ZipPath, c.ZipEntry)
	if err != nil {
		return nil, err
	}
	data, err := readZipFile(zf)
	if err != nil {
		return nil, err
	}
	return fontMetadataFromBytes(data, c.Basename)
}

// zipCache opens each archive once and reuses its central directory.
type zipCache struct {
	open  map[string]*zip.ReadCloser
	files map[string]map[string]*zip.File
}

func newZipCache() *zipCache {
	return &zipCache{
		open:  map[string]*zip.ReadCloser{},
		files: map[string]map[string]*zip.File{},
	}
}

func (z *zipCache) Close() {
	if z == nil {
		return
	}
	for _, r := range z.open {
		_ = r.Close()
	}
}

func (z *zipCache) entry(zipPath, name string) (*zip.File, error) {
	files := z.files[zipPath]
	if files == nil {
		r, err := zip.OpenReader(zipPath)
		if err != nil {
			return nil, err
		}
		z.open[zipPath] = r
		files = make(map[string]*zip.File, len(r.File))
		for _, f := range r.File {
			files[filepath.ToSlash(f.Name)] = f
		}
		z.files[zipPath] = files
	}
	f := files[name]
	if f == nil {
		return nil, fmt.Errorf("zip entry not found: %s", name)
	}
	return f, nil
}

func readZipFile(zf *zip.File) ([]byte, error) {
	rc, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	data, err := io.ReadAll(io.LimitReader(rc, localIngestMaxFileBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > localIngestMaxFileBytes {
		return nil, fmt.Errorf("zip entry exceeds size limit")
	}
	return data, nil
}

// ExtractZipEntryToTemp extracts one zip member to a temp file for install.
// Caller must remove the returned path.
func ExtractZipEntryToTemp(zipPath, entry, basename string) (string, error) {
	dir, err := os.MkdirTemp("", "fontget-local-*")
	if err != nil {
		return "", err
	}
	named, err := extractZipEntryIntoDir(zipPath, entry, basename, dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	return named, nil
}

// StageLocalCandidates extracts every candidate, opening each zip once.
func StageLocalCandidates(staging *OperationStaging, cands []LocalFontCandidate) ([]LocalFontCandidate, error) {
	if staging == nil {
		return nil, fmt.Errorf("nil operation staging")
	}
	zips := newZipCache()
	defer zips.Close()
	out := make([]LocalFontCandidate, len(cands))
	for i, c := range cands {
		sc, err := stageLocalCandidate(staging, c, zips)
		if err != nil {
			return nil, fmt.Errorf("stage local font %s: %w", CandidateLabel(c), err)
		}
		out[i] = sc
	}
	return out, nil
}

// StageLocalCandidate copies or extracts a candidate into operation staging.
// The returned candidate always points at a loose staged file (FromZip=false).
func StageLocalCandidate(staging *OperationStaging, c LocalFontCandidate) (LocalFontCandidate, error) {
	return stageLocalCandidate(staging, c, nil)
}

func stageLocalCandidate(staging *OperationStaging, c LocalFontCandidate, zips *zipCache) (LocalFontCandidate, error) {
	if staging == nil {
		return LocalFontCandidate{}, fmt.Errorf("nil operation staging")
	}
	dir, err := staging.LocalStageDir(c.Basename)
	if err != nil {
		return LocalFontCandidate{}, err
	}
	var staged string
	if c.FromZip {
		if zips == nil {
			zips = newZipCache()
			defer zips.Close()
		}
		zf, zerr := zips.entry(c.ZipPath, c.ZipEntry)
		if zerr != nil {
			_ = os.RemoveAll(dir)
			return LocalFontCandidate{}, zerr
		}
		staged, err = extractOpenZipEntry(zf, c.Basename, dir)
	} else {
		staged, err = copyFileIntoDir(c.DiskPath, c.Basename, dir)
	}
	if err != nil {
		_ = os.RemoveAll(dir)
		return LocalFontCandidate{}, err
	}
	info, err := os.Stat(staged)
	if err != nil {
		_ = os.RemoveAll(dir)
		return LocalFontCandidate{}, err
	}
	family, style, full := c.Family, c.Style, c.FullName
	if !c.MetaReady {
		md, merr := ExtractFontMetadata(staged)
		if merr != nil {
			_ = os.RemoveAll(dir)
			return LocalFontCandidate{}, merr
		}
		family = preferredFamily(md)
		style = preferredStyle(md)
		full = strings.TrimSpace(md.FullName)
	}
	origin := strings.TrimSpace(c.OriginDiskPath)
	if origin == "" && c.LooseFile && !c.FromZip {
		origin = c.DiskPath
	}
	// Resolve aliases while the origin still exists; recovery must never replace
	// the user's symlink or try to resolve it after force removal.
	if origin != "" {
		origin, err = filepath.EvalSymlinks(origin)
		if err != nil {
			_ = os.RemoveAll(dir)
			return LocalFontCandidate{}, fmt.Errorf("resolve recovery origin: %w", err)
		}
		origin, err = filepath.Abs(origin)
		if err != nil {
			_ = os.RemoveAll(dir)
			return LocalFontCandidate{}, err
		}
	}
	return LocalFontCandidate{
		DiskPath:       CanonicalPath(staged),
		OriginDiskPath: origin,
		Basename:       filepath.Base(staged),
		Size:           info.Size(),
		LooseFile:      true,
		FromZip:        false,
		Depth:          0,
		SHA256:         c.SHA256,
		Family:         family,
		Style:          style,
		FullName:       full,
		MetaReady:      true,
	}, nil
}

func copyFileIntoDir(srcPath, basename, dir string) (string, error) {
	base := basename
	if base == "" {
		base = filepath.Base(srcPath)
	}
	named := filepath.Join(dir, base)
	in, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer in.Close()
	out, err := os.OpenFile(named, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, io.LimitReader(in, localIngestMaxFileBytes+1)); err != nil {
		_ = out.Close()
		return "", err
	}
	if err := out.Close(); err != nil {
		return "", err
	}
	return named, nil
}

func extractZipEntryIntoDir(zipPath, entry, basename, dir string) (string, error) {
	zr, zf, err := openZipMember(zipPath, entry)
	if err != nil {
		return "", err
	}
	defer zr.Close()
	return extractOpenZipEntry(zf, basename, dir)
}

func extractOpenZipEntry(zf *zip.File, basename, dir string) (string, error) {
	rc, err := zf.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	base := basename
	if base == "" {
		base = filepath.Base(zf.Name)
	}
	named := filepath.Join(dir, base)
	out, err := os.OpenFile(named, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, io.LimitReader(rc, localIngestMaxFileBytes+1)); err != nil {
		_ = out.Close()
		return "", err
	}
	if err := out.Close(); err != nil {
		return "", err
	}
	return named, nil
}
