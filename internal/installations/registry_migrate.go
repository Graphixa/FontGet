package installations

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed migrations/001-nerd-fonts-v1-to-v2.json
var nerdFontsV1ToV2JSON []byte

// registryMigrationStep advances schema_version by one hop. Steps are applied in order until
// reg.SchemaVersion matches schemaVersion (see schemaVersion in registry.go).
type registryMigrationStep struct {
	from string
	to   string
	fn   func(*Registry) error
}

var registryMigrationChain []registryMigrationStep

func init() {
	registryMigrationChain = buildRegistryMigrations()
}

// buildRegistryMigrations defines every allowed schema_version transition for this binary.
// When you bump schemaVersion, add a new switch case and chain older versions → newer (one hop per `from`).
func buildRegistryMigrations() []registryMigrationStep {
	switch schemaVersion {
	case "1.1":
		return []registryMigrationStep{
			{from: "", to: "1.0"},
			{from: "1", to: "1.0"},
			{from: "1.0", to: "1.1", fn: migrateV1_0ToV1_1},
		}
	default:
		panic(fmt.Sprintf("installations: schemaVersion %q has no migration definition — edit buildRegistryMigrations in registry_migrate.go", schemaVersion))
	}
}

func registryMigrationFrom(cur string) *registryMigrationStep {
	for i := range registryMigrationChain {
		if registryMigrationChain[i].from == cur {
			return &registryMigrationChain[i]
		}
	}
	return nil
}

// applyRegistryMigrations advances reg.SchemaVersion (and optional data) until it equals schemaVersion.
// It returns whether any step ran (caller may persist to disk).
func applyRegistryMigrations(reg *Registry) (changed bool, err error) {
	if reg == nil {
		return false, fmt.Errorf("nil registry")
	}
	for {
		cur := strings.TrimSpace(reg.SchemaVersion)
		if cur == schemaVersion {
			return changed, nil
		}
		step := registryMigrationFrom(cur)
		if step == nil {
			return changed, fmt.Errorf("installation registry schema_version %q cannot be migrated (FontGet expects %s)", reg.SchemaVersion, schemaVersion)
		}
		if step.fn != nil {
			if err := step.fn(reg); err != nil {
				return changed, fmt.Errorf("migrate installation registry from schema_version %q: %w", cur, err)
			}
		}
		reg.SchemaVersion = step.to
		changed = true
	}
}

// CurrentRegistrySchemaVersion returns the schema_version string this binary reads and writes.
func CurrentRegistrySchemaVersion() string {
	return schemaVersion
}

type fontIDRenameDoc struct {
	Renames []fontIDRename `json:"renames"`
}

type fontIDRename struct {
	From        string `json:"from"`
	To          string `json:"to"`
	CatalogName string `json:"catalog_name"`
}

func migrateV1_0ToV1_1(reg *Registry) error {
	return applyFontIDRenames(reg, nerdFontsV1ToV2JSON)
}

func applyFontIDRenames(reg *Registry, raw []byte) error {
	var doc fontIDRenameDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("parse font id renames: %w", err)
	}
	if reg.Installations == nil {
		reg.Installations = make(map[string]*Installation)
	}
	for _, r := range doc.Renames {
		from := strings.ToLower(strings.TrimSpace(r.From))
		to := strings.ToLower(strings.TrimSpace(r.To))
		if from == "" || to == "" || from == to {
			continue
		}
		inst, ok := reg.Installations[from]
		if !ok {
			continue
		}
		if _, exists := reg.Installations[to]; exists {
			// Destination already tracked — drop the legacy key.
			delete(reg.Installations, from)
			continue
		}
		delete(reg.Installations, from)
		inst.FontID = to
		if name := strings.TrimSpace(r.CatalogName); name != "" {
			inst.CatalogName = name
		}
		reg.Installations[to] = inst
	}
	return nil
}
