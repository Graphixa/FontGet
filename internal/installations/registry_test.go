package installations

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRegistryPath_underHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	p := RegistryPath()
	if filepath.Dir(p) != filepath.Join(home, ".fontget") {
		t.Fatalf("RegistryPath dir = %q want %q", filepath.Dir(p), filepath.Join(home, ".fontget"))
	}
	if filepath.Base(p) != FileName {
		t.Fatalf("basename = %q want %q", filepath.Base(p), FileName)
	}
}

func TestCurrentRegistrySchemaVersion(t *testing.T) {
	if CurrentRegistrySchemaVersion() != schemaVersion {
		t.Fatalf("got %q", CurrentRegistrySchemaVersion())
	}
}

func TestLoad_migratesSchemaVersionAndPersists(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".fontget")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, FileName)
	raw := []byte(`{
		"schema_version": "1",
		"created": "2024-01-01T00:00:00Z",
		"last_updated": "2024-01-01T00:00:00Z",
		"installations": {}
	}`)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	reg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if reg.SchemaVersion != schemaVersion {
		t.Fatalf("Load: got schema_version %q want %q", reg.SchemaVersion, schemaVersion)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"schema_version": "`+schemaVersion+`"`) {
		t.Fatalf("migrated file should contain current schema: %s", string(data))
	}
}

func TestLoad_unknownSchemaVersion_errors(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".fontget")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, FileName)
	raw := []byte(`{
		"schema_version": "0.5.test-unreadable",
		"created": "2024-01-01T00:00:00Z",
		"last_updated": "2024-01-01T00:00:00Z",
		"installations": {}
	}`)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for unmigratable schema_version")
	}
}

func TestLoad_missingFile_isEmpty(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	reg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(reg.Installations) != 0 {
		t.Fatalf("want empty installations")
	}
}

func TestRecord_roundTripAndUpsert(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	err := RecordInstallation(RecordParams{
		FontID:         "nerd.testfont",
		CatalogName:    "Test Font",
		Scope:          "user",
		FontGetVersion: "1.0.0",
		Files: []InstalledFontFile{
			{Path: "/tmp/a.ttf", SFNT: SFNTSnapshot{Family: "Test Nerd Font"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	reg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	inst := reg.FindByFontID("NERD.TestFont")
	if inst == nil || inst.CatalogName != "Test Font" || len(inst.FlatFiles()) != 1 {
		t.Fatalf("FindByFontID: %#v", inst)
	}

	err = RecordInstallation(RecordParams{
		FontID:         "nerd.testfont",
		CatalogName:    "Test Font",
		Scope:          "user",
		FontGetVersion: "1.0.1",
		Files: []InstalledFontFile{
			{Path: "/tmp/b.otf", SFNT: SFNTSnapshot{Family: "Test Nerd Font Mono"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	reg2, _ := Load()
	inst2 := reg2.FindByFontID("nerd.testfont")
	ff := inst2.FlatFiles()
	if len(ff) != 1 || ff[0].Path != "/tmp/b.otf" {
		t.Fatalf("upsert files: %#v", ff)
	}
}

func TestPathIndex_andNormalizePathKey(t *testing.T) {
	reg := &Registry{
		SchemaVersion: schemaVersion,
		Created:       time.Now().UTC(),
		LastUpdated:   time.Now().UTC(),
		Installations: map[string]*Installation{
			"google.x": {
				FontID:   "google.x",
				Families: GroupInstalledFiles([]InstalledFontFile{{Path: "/Library/Fonts/X.ttf", SFNT: SFNTSnapshot{Family: "X"}}}),
			},
		},
	}
	idx := reg.PathIndex()
	k := NormalizePathKey("/Library/Fonts/X.ttf")
	if idx[k] == nil {
		t.Fatalf("path index miss for key %q", k)
	}
}

func TestBasenamesForDir(t *testing.T) {
	inst := &Installation{
		Families: GroupInstalledFiles([]InstalledFontFile{
			{Path: filepath.Join("/Users/u/Library/Fonts", "a.ttf"), SFNT: SFNTSnapshot{Family: "A"}},
			{Path: "/other/b.ttf", SFNT: SFNTSnapshot{Family: "B"}},
		}),
	}
	dir := filepath.Join("/Users", "u", "Library", "Fonts")
	got := inst.BasenamesForDir(dir)
	if len(got) != 1 || got[0] != "a.ttf" {
		t.Fatalf("got %#v", got)
	}
}

func TestDirContainsFontFile_caseInsensitiveDir(t *testing.T) {
	parent := filepath.Join("/tmp", "FontDir")
	file := filepath.Join(parent, "X.ttf")
	if !DirContainsFontFile(parent, file) {
		t.Fatal("expected match")
	}
	if DirContainsFontFile(parent, filepath.Join("/tmp", "Other", "y.ttf")) {
		t.Fatal("expected no match")
	}
}

func TestLoad_corruptJSON(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".fontget")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for corrupt json")
	}
}

func TestRemoveInstallation(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := RecordInstallation(RecordParams{
		FontID:      "a.b",
		CatalogName: "B",
		Scope:       "user",
		Files:       []InstalledFontFile{{Path: "/x.ttf"}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := RemoveInstallation("a.b"); err != nil {
		t.Fatal(err)
	}
	reg, _ := Load()
	if reg.FindByFontID("a.b") != nil {
		t.Fatal("expected removal")
	}
}

func TestJSONFieldNamesRegistryShape(t *testing.T) {
	reg := Registry{
		SchemaVersion: schemaVersion,
		Created:       time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		LastUpdated:   time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		Installations: map[string]*Installation{
			"x": {
				FontID:      "x.y",
				CatalogName: "Y",
				Scope:       "user",
				InstalledAt: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
				Families: GroupInstalledFiles([]InstalledFontFile{
					{Path: "/p.ttf", SFNT: SFNTSnapshot{Family: "Fam"}},
				}),
				FontGetVersion: "9",
			},
		},
	}
	b, err := json.Marshal(reg)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"schema_version", "created", "last_updated", "installations"} {
		if _, ok := raw[key]; !ok {
			t.Fatalf("missing json key %q in %s", key, string(b))
		}
	}
	var insts map[string]json.RawMessage
	if err := json.Unmarshal(raw["installations"], &insts); err != nil {
		t.Fatal(err)
	}
	var one map[string]json.RawMessage
	if err := json.Unmarshal(insts["x"], &one); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"font_id", "catalog_name", "families"} {
		if _, ok := one[key]; !ok {
			t.Fatalf("installation missing %q: %s", key, string(insts["x"]))
		}
	}
	var famGroups []map[string]json.RawMessage
	if err := json.Unmarshal(one["families"], &famGroups); err != nil {
		t.Fatal(err)
	}
	if len(famGroups) == 0 {
		t.Fatal("expected non-empty families")
	}
	if _, ok := famGroups[0]["files"]; !ok {
		t.Fatalf("family group should expose files key: %s", string(one["families"]))
	}
}

func TestFamilyGroup_JSON_roundTrip(t *testing.T) {
	const raw = `{"family":"F","files":[{"path":"/a.ttf","style":"Regular","full_name":"F Regular"}]}`
	var g FamilyGroup
	if err := json.Unmarshal([]byte(raw), &g); err != nil {
		t.Fatal(err)
	}
	if len(g.Files) != 1 || g.Files[0].Path != "/a.ttf" {
		t.Fatalf("got %+v", g.Files)
	}
	out, err := json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	var again FamilyGroup
	if err := json.Unmarshal(out, &again); err != nil {
		t.Fatal(err)
	}
	if len(again.Files) != 1 || again.Files[0].Path != "/a.ttf" {
		t.Fatalf("round-trip: %+v", again.Files)
	}
}
