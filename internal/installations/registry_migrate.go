package installations

import (
	"fmt"
	"strings"
)

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
//
// Example for bumping from "1.0" to "2.0":
//
//	case "2.0":
//		return []registryMigrationStep{
//			{from: "", to: "1.0"},
//			{from: "1", to: "1.0"},
//			{from: "1.0", to: "2.0", fn: migrateV1_0ToV2_0},
//		}
func buildRegistryMigrations() []registryMigrationStep {
	switch schemaVersion {
	case "1.0":
		return []registryMigrationStep{
			{from: "", to: schemaVersion},
			{from: "1", to: schemaVersion},
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
