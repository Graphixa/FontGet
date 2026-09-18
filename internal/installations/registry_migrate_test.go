package installations

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"fontget/internal/testutil"
)

func TestMigrateV1_0ToV1_1_renamesNerdFontIDs(t *testing.T) {
	reg := &Registry{
		SchemaVersion: "1.0",
		Installations: map[string]*Installation{
			"nerd.cascadia-code": {
				FontID:      "nerd.cascadia-code",
				CatalogName: "Cascadia Code",
				Scope:       "user",
				InstalledAt: time.Now().UTC(),
			},
			"nerd.jetbrains-mono": {
				FontID:      "nerd.jetbrains-mono",
				CatalogName: "JetBrainsMono Nerd Font",
				Scope:       "user",
				InstalledAt: time.Now().UTC(),
			},
		},
	}
	if err := applyFontIDRenames(reg, nerdFontsV1ToV2JSON); err != nil {
		t.Fatal(err)
	}
	if _, ok := reg.Installations["nerd.cascadia-code"]; ok {
		t.Fatal("legacy cascadia-code key should be gone")
	}
	inst := reg.Installations["nerd.caskaydia-cove"]
	if inst == nil {
		t.Fatal("missing nerd.caskaydia-cove")
	}
	if inst.FontID != "nerd.caskaydia-cove" {
		t.Fatalf("FontID = %q", inst.FontID)
	}
	if inst.CatalogName != "CaskaydiaCove Nerd Font" {
		t.Fatalf("CatalogName = %q", inst.CatalogName)
	}
	if reg.Installations["nerd.jetbrains-mono"] == nil {
		t.Fatal("unchanged id should remain")
	}
}

func TestMigrateV1_0ToV1_1_destinationExistsDropsFrom(t *testing.T) {
	reg := &Registry{
		SchemaVersion: "1.0",
		Installations: map[string]*Installation{
			"nerd.cascadia-code": {
				FontID:      "nerd.cascadia-code",
				CatalogName: "old",
				Scope:       "user",
			},
			"nerd.caskaydia-cove": {
				FontID:      "nerd.caskaydia-cove",
				CatalogName: "keep",
				Scope:       "user",
			},
		},
	}
	if err := applyFontIDRenames(reg, nerdFontsV1ToV2JSON); err != nil {
		t.Fatal(err)
	}
	if _, ok := reg.Installations["nerd.cascadia-code"]; ok {
		t.Fatal("from key should be dropped when to exists")
	}
	if got := reg.Installations["nerd.caskaydia-cove"].CatalogName; got != "keep" {
		t.Fatalf("CatalogName = %q want keep", got)
	}
}

func TestLoad_appliesNerdIDRenamesAndPersists(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)
	dir := filepath.Join(home, ".fontget")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, FileName)
	raw := []byte(`{
		"schema_version": "1.0",
		"created": "2024-01-01T00:00:00Z",
		"last_updated": "2024-01-01T00:00:00Z",
		"installations": {
			"nerd.cascadia-code": {
				"font_id": "nerd.cascadia-code",
				"catalog_name": "Cascadia Code",
				"scope": "user",
				"installed_at": "2024-01-01T00:00:00Z",
				"families": []
			}
		}
	}`)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	reg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if reg.SchemaVersion != "1.1" {
		t.Fatalf("schema = %q", reg.SchemaVersion)
	}
	if reg.FindByFontID("nerd.caskaydia-cove") == nil {
		t.Fatal("expected renamed install")
	}
	if reg.FindByFontID("nerd.cascadia-code") != nil {
		t.Fatal("legacy id should be gone")
	}

	// Second load is idempotent.
	reg2, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if reg2.FindByFontID("nerd.caskaydia-cove") == nil {
		t.Fatal("rename should stick")
	}
}
