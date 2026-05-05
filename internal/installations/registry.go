// Package installations persists FontGet install provenance for list/remove matching.
package installations

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"fontget/internal/config"
)

// FileName is the install registry basename under GetAppConfigDir (distinct from manifest.json).
const FileName = "installation_registry.json"

var mu sync.Mutex

// Registry is the on-disk installation registry root object.
type Registry struct {
	SchemaVersion string                   `json:"schema_version"`
	Created       time.Time                `json:"created"`
	LastUpdated   time.Time                `json:"last_updated"`
	Installations map[string]*Installation `json:"installations"`
}

// SFNTSnapshot is name-table data captured at install time for one file (matches list semantics).
type SFNTSnapshot struct {
	Family   string `json:"family,omitempty"`    // typographic family when present, else Name ID 1 family
	Style    string `json:"style,omitempty"`     // typographic style when present, else Name ID 2 subfamily
	FullName string `json:"full_name,omitempty"` // Name ID 4 when available
}

// InstalledFontFile is the flat row shape used at record time and inside cmd/list merge logic.
// On disk, rows are grouped under FamilyGroup for readability (large archive installs).
type InstalledFontFile struct {
	Path           string       `json:"path"`
	CatalogVariant string       `json:"catalog_variant,omitempty"` // variant label from FontGet-Sources for this archive member
	SFNT           SFNTSnapshot `json:"sfnt,omitempty"`
}

// FamilyGroup is one preferred SFNT family string and its installed files (paths on disk).
type FamilyGroup struct {
	Family string          `json:"family"`
	Files  []InstalledFace `json:"files"`
}

// InstalledFace is one installed file under a family group; family string lives on the parent group only.
type InstalledFace struct {
	Path           string `json:"path"`
	Style          string `json:"style,omitempty"`
	FullName       string `json:"full_name,omitempty"`
	CatalogVariant string `json:"catalog_variant,omitempty"`
}

// Installation is one catalog install record (map key: lowercase Font ID).
type Installation struct {
	FontID               string        `json:"font_id"`
	CatalogName          string        `json:"catalog_name,omitempty"`
	InstallationSource   string        `json:"installation_source,omitempty"` // manifest Sources map key
	Scope                string        `json:"scope"`
	InstalledAt          time.Time     `json:"installed_at"`
	FontGetVersion       string        `json:"fontget_version,omitempty"`
	Families             []FamilyGroup `json:"families"`
}

// Bump when the persisted JSON contract changes incompatibly.
const schemaVersion = "1.0"

func normalizeFamilyGroups(in []FamilyGroup) []FamilyGroup {
	if len(in) == 0 {
		return nil
	}
	out := make([]FamilyGroup, 0, len(in))
	for _, g := range in {
		fam := strings.TrimSpace(g.Family)
		var faces []InstalledFace
		seen := make(map[string]struct{})
		for _, fc := range g.Files {
			p := strings.TrimSpace(fc.Path)
			if p == "" {
				continue
			}
			p = filepath.Clean(p)
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			faces = append(faces, InstalledFace{
				Path:           p,
				Style:          strings.TrimSpace(fc.Style),
				FullName:       strings.TrimSpace(fc.FullName),
				CatalogVariant: strings.TrimSpace(fc.CatalogVariant),
			})
		}
		if len(faces) == 0 {
			continue
		}
		sortFaces(faces)
		out = append(out, FamilyGroup{Family: fam, Files: faces})
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Family) < strings.ToLower(out[j].Family)
	})
	return out
}

func sortFaces(faces []InstalledFace) {
	sort.Slice(faces, func(i, j int) bool {
		si, sj := styleSortKey(faces[i].Style), styleSortKey(faces[j].Style)
		if si != sj {
			return si < sj
		}
		return faces[i].Path < faces[j].Path
	})
}

func styleSortKey(style string) int {
	switch strings.TrimSpace(style) {
	case "Regular":
		return 0
	case "Italic":
		return 1
	case "Bold":
		return 2
	case "Bold Italic":
		return 3
	default:
		return 100
	}
}

// RegistryPath returns the absolute path to installation_registry.json.
func RegistryPath() string {
	return filepath.Join(config.GetAppConfigDir(), FileName)
}

// Load reads the registry from disk. Missing file yields an empty registry (no error).
// Invalid JSON returns an error (caller should not overwrite without user intent).
func Load() (*Registry, error) {
	mu.Lock()
	defer mu.Unlock()
	return loadUnlocked()
}

func loadUnlocked() (*Registry, error) {
	path := RegistryPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			now := time.Now().UTC()
			return &Registry{
				SchemaVersion: schemaVersion,
				Created:       now,
				LastUpdated:   now,
				Installations: make(map[string]*Installation),
			}, nil
		}
		return nil, fmt.Errorf("read installation registry: %w", err)
	}
	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parse installation registry %q: %w", path, err)
	}
	if reg.Installations == nil {
		reg.Installations = make(map[string]*Installation)
	}
	migrated, err := applyRegistryMigrations(&reg)
	if err != nil {
		return nil, err
	}
	for _, inst := range reg.Installations {
		normalizeLoadedInstallation(inst)
	}
	if migrated {
		if err := saveUnlocked(&reg); err != nil {
			return nil, fmt.Errorf("persist migrated installation registry: %w", err)
		}
	}
	return &reg, nil
}

func normalizeLoadedInstallation(inst *Installation) {
	if inst == nil {
		return
	}
	inst.FontID = strings.TrimSpace(inst.FontID)
	inst.CatalogName = strings.TrimSpace(inst.CatalogName)
	inst.InstallationSource = strings.TrimSpace(inst.InstallationSource)
	inst.Scope = strings.TrimSpace(inst.Scope)
	inst.FontGetVersion = strings.TrimSpace(inst.FontGetVersion)
	inst.Families = normalizeFamilyGroups(inst.Families)
}

// Save writes the registry atomically.
func Save(reg *Registry) error {
	if reg == nil {
		return fmt.Errorf("nil registry")
	}
	mu.Lock()
	defer mu.Unlock()
	return saveUnlocked(reg)
}

func saveUnlocked(reg *Registry) error {
	reg.LastUpdated = time.Now().UTC()
	if reg.Created.IsZero() {
		reg.Created = reg.LastUpdated
	}
	reg.SchemaVersion = schemaVersion
	path := RegistryPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("mkdir for installation registry: %w", err)
	}
	payload, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal installation registry: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o600); err != nil {
		return fmt.Errorf("write installation registry temp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		if remErr := os.Remove(tmp); remErr != nil && !os.IsNotExist(remErr) {
			return fmt.Errorf("rename installation registry: %w (temp file cleanup: %v)", err, remErr)
		}
		return fmt.Errorf("rename installation registry: %w", err)
	}
	return nil
}

// RecordParams describes a successful install to persist.
type RecordParams struct {
	FontID               string
	CatalogName          string // catalog display name at install time
	InstallationSource   string // manifest Sources map key (repository bundle), if known
	Scope                string // "user" or "machine"
	FontGetVersion       string
	Files                []InstalledFontFile // one row per installed file (grouped on save)
}

// RecordInstallation upserts one installation keyed by lowercase Font ID.
func RecordInstallation(p RecordParams) error {
	if strings.TrimSpace(p.FontID) == "" {
		return fmt.Errorf("empty font_id")
	}
	if len(p.Files) == 0 {
		return fmt.Errorf("empty files")
	}
	key := strings.ToLower(strings.TrimSpace(p.FontID))

	mu.Lock()
	defer mu.Unlock()

	reg, err := loadUnlocked()
	if err != nil {
		return err
	}

	flat := normalizeInstalledFiles(p.Files)
	inst := &Installation{
		FontID:               p.FontID,
		CatalogName:          strings.TrimSpace(p.CatalogName),
		InstallationSource:   strings.TrimSpace(p.InstallationSource),
		Scope:                strings.TrimSpace(p.Scope),
		InstalledAt:          time.Now().UTC(),
		FontGetVersion:       strings.TrimSpace(p.FontGetVersion),
		Families:             GroupInstalledFiles(flat),
	}
	reg.Installations[key] = inst
	return saveUnlocked(reg)
}

func normalizeInstalledFiles(in []InstalledFontFile) []InstalledFontFile {
	seen := make(map[string]struct{})
	var out []InstalledFontFile
	for _, f := range in {
		p := strings.TrimSpace(f.Path)
		if p == "" {
			continue
		}
		p = filepath.Clean(p)
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, InstalledFontFile{
			Path:           p,
			CatalogVariant: strings.TrimSpace(f.CatalogVariant),
			SFNT: SFNTSnapshot{
				Family:   strings.TrimSpace(f.SFNT.Family),
				Style:    strings.TrimSpace(f.SFNT.Style),
				FullName: strings.TrimSpace(f.SFNT.FullName),
			},
		})
	}
	return out
}

// GroupInstalledFiles buckets flat rows by PreferredFamily and sorts families/files deterministically.
// Exported for maintenance generators; RecordInstallation uses this after normalization.
func GroupInstalledFiles(flat []InstalledFontFile) []FamilyGroup {
	return groupFlatInstalledFiles(flat)
}

func groupFlatInstalledFiles(flat []InstalledFontFile) []FamilyGroup {
	buckets := make(map[string][]InstalledFace)
	var order []string
	seenFam := make(map[string]struct{})
	for _, f := range flat {
		fam := PreferredFamily(f.SFNT)
		if _, ok := seenFam[fam]; !ok {
			seenFam[fam] = struct{}{}
			order = append(order, fam)
		}
		buckets[fam] = append(buckets[fam], InstalledFace{
			Path:           f.Path,
			Style:          f.SFNT.Style,
			FullName:       f.SFNT.FullName,
			CatalogVariant: f.CatalogVariant,
		})
	}
	groups := make([]FamilyGroup, 0, len(order))
	for _, fam := range order {
		faces := buckets[fam]
		sortFaces(faces)
		groups = append(groups, FamilyGroup{Family: fam, Files: faces})
	}
	sort.Slice(groups, func(i, j int) bool {
		return strings.ToLower(groups[i].Family) < strings.ToLower(groups[j].Family)
	})
	return normalizeFamilyGroups(groups)
}

// PreferredFamily returns the family string used for list matching (same preference as list command).
func PreferredFamily(s SFNTSnapshot) string {
	return strings.TrimSpace(s.Family)
}

func (inst *Installation) eachFace(fn func(InstalledFontFile)) {
	if inst == nil {
		return
	}
	for _, g := range inst.Families {
		fam := strings.TrimSpace(g.Family)
		for _, face := range g.Files {
			fn(InstalledFontFile{
				Path:           face.Path,
				CatalogVariant: face.CatalogVariant,
				SFNT: SFNTSnapshot{
					Family:   fam,
					Style:    strings.TrimSpace(face.Style),
					FullName: strings.TrimSpace(face.FullName),
				},
			})
		}
	}
}

// FlatFiles expands grouped storage into a flat slice (deterministic: family order then face order).
func (inst *Installation) FlatFiles() []InstalledFontFile {
	if inst == nil {
		return nil
	}
	var out []InstalledFontFile
	inst.eachFace(func(f InstalledFontFile) {
		out = append(out, f)
	})
	return out
}

// HasFaces reports whether any installed file paths are recorded.
func (inst *Installation) HasFaces() bool {
	if inst == nil {
		return false
	}
	for _, g := range inst.Families {
		if len(g.Files) > 0 {
			return true
		}
	}
	return false
}

// RemoveInstallation deletes the record for fontID (case-insensitive).
func RemoveInstallation(fontID string) error {
	key := strings.ToLower(strings.TrimSpace(fontID))
	if key == "" {
		return nil
	}
	mu.Lock()
	defer mu.Unlock()
	reg, err := loadUnlocked()
	if err != nil {
		return err
	}
	delete(reg.Installations, key)
	return saveUnlocked(reg)
}

// FindByFontID returns the installation for fontID or nil.
func (reg *Registry) FindByFontID(fontID string) *Installation {
	if reg == nil || reg.Installations == nil {
		return nil
	}
	key := strings.ToLower(strings.TrimSpace(fontID))
	return reg.Installations[key]
}

// PathIndex maps normalized absolute paths (see NormalizePathKey) to installations.
func (reg *Registry) PathIndex() map[string]*Installation {
	out := make(map[string]*Installation)
	if reg == nil || reg.Installations == nil {
		return out
	}
	for _, inst := range reg.Installations {
		if inst == nil {
			continue
		}
		inst.eachFace(func(f InstalledFontFile) {
			k := NormalizePathKey(f.Path)
			if k != "" {
				out[k] = inst
			}
		})
	}
	return out
}

// FamilyInstallationsIndex maps lowercase preferred SFNT family to installations that declare at least one face with that family.
// Multiple installations under one key indicate ambiguous family-key merge; callers should fall back to path index only.
func (reg *Registry) FamilyInstallationsIndex() map[string][]*Installation {
	out := make(map[string][]*Installation)
	if reg == nil || reg.Installations == nil {
		return out
	}
	for _, inst := range reg.Installations {
		if inst == nil {
			continue
		}
		seenFam := make(map[string]struct{})
		inst.eachFace(func(f InstalledFontFile) {
			k := strings.ToLower(strings.TrimSpace(PreferredFamily(f.SFNT)))
			if k == "" {
				return
			}
			if _, dup := seenFam[k]; dup {
				return
			}
			seenFam[k] = struct{}{}
			out[k] = appendUniqueInstallation(out[k], inst)
		})
	}
	return out
}

func appendUniqueInstallation(slice []*Installation, inst *Installation) []*Installation {
	for _, x := range slice {
		if x == inst {
			return slice
		}
	}
	return append(slice, inst)
}

// NormalizePathKey canonicalizes a path for cross-checking with ParsedFont.Path.
func NormalizePathKey(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	p = filepath.Clean(p)
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	return strings.ToLower(p)
}

func dedupeStrings(in []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// canonicalFontDir normalizes a font directory for comparison (Clean + EvalSymlinks when possible).
func canonicalFontDir(d string) string {
	d = filepath.Clean(strings.TrimSpace(d))
	if d == "" {
		return ""
	}
	if sy, err := filepath.EvalSymlinks(d); err == nil && sy != "" {
		d = filepath.Clean(sy)
	}
	return d
}

// DirContainsFontFile reports whether parent is the directory of p after canonicalization.
func DirContainsFontFile(parent, fontFile string) bool {
	parent = canonicalFontDir(parent)
	if parent == "" {
		return false
	}
	p := filepath.Clean(strings.TrimSpace(fontFile))
	if p == "" {
		return false
	}
	if sy, err := filepath.EvalSymlinks(p); err == nil && sy != "" {
		p = filepath.Clean(sy)
	}
	dp := canonicalFontDir(filepath.Dir(p))
	return strings.EqualFold(dp, parent)
}

// BasenamesForDir returns base filenames for paths under fontDir (for platform RemoveFont).
func (inst *Installation) BasenamesForDir(fontDir string) []string {
	if inst == nil {
		return nil
	}
	fontDirCanon := canonicalFontDir(fontDir)
	var out []string
	inst.eachFace(func(f InstalledFontFile) {
		if DirContainsFontFile(fontDirCanon, f.Path) {
			out = append(out, filepath.Base(f.Path))
		}
	})
	return dedupeStrings(out)
}
