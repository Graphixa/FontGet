package config

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// getConfigVersionFromRaw returns the schema version from a raw config map.
// Checks "Version", legacy "version", and "ConfigVersion" keys.
func getConfigVersionFromRaw(raw map[string]interface{}) string {
	if v, ok := raw["Version"].(string); ok && v != "" {
		return v
	}
	if v, ok := raw["version"].(string); ok && v != "" {
		return v
	}
	if v, ok := raw["ConfigVersion"].(string); ok && v != "" {
		return v
	}
	return ""
}

// copyMatchingKeys copies values from old into new where the key path exists in new.
// Preserves the new schema structure; only overwrites with old values for matching keys.
// Recurses into nested maps. Used when migrating to a new schema so that only
// keys that exist in the new schema are carried over.
func copyMatchingKeys(old, new map[string]interface{}) {
	for key, newVal := range new {
		oldVal, exists := old[key]
		if !exists {
			continue
		}
		oldMap, oldIsMap := oldVal.(map[string]interface{})
		newMap, newIsMap := newVal.(map[string]interface{})
		if oldIsMap && newIsMap {
			copyMatchingKeys(oldMap, newMap)
		} else if !newIsMap {
			// New schema has a scalar (or slice) here; copy old value over
			new[key] = oldVal
		}
	}
}

// applyExplicitMigrationRules applies known renames and structural changes from old to new.
// Add new rules here when the schema changes (e.g. key renames, string→object).
func applyExplicitMigrationRules(old, new map[string]interface{}) {
	// Theme: string → { Name, Use256ColorSpace }
	if themeVal, ok := old["Theme"]; ok {
		if themeStr, ok := themeVal.(string); ok {
			new["Theme"] = map[string]interface{}{
				"Name":              themeStr,
				"Use256ColorSpace": false,
			}
		}
	}

	// fieldRenameMap: same-section renames (e.g. Update.AutoCheck → Update.CheckForUpdates)
	for oldPath, newPath := range fieldRenameMap {
		oldParts := strings.Split(oldPath, ".")
		newParts := strings.Split(newPath, ".")
		if len(oldParts) != 2 || len(newParts) != 2 || oldParts[0] != newParts[0] {
			continue
		}
		sectionName := oldParts[0]
		oldField := oldParts[1]
		newField := newParts[1]
		section, ok := old[sectionName].(map[string]interface{})
		if !ok {
			continue
		}
		val, has := section[oldField]
		if !has {
			continue
		}
		newSection, ok := new[sectionName].(map[string]interface{})
		if !ok {
			continue
		}
		if _, exists := newSection[newField]; !exists {
			newSection[newField] = val
		}
	}

	// fieldMoveMap: cross-section moves (e.g. Configuration.EnablePopularitySort → Search.EnablePopularitySort)
	for oldPath, newPath := range fieldMoveMap {
		oldParts := strings.Split(oldPath, ".")
		newParts := strings.Split(newPath, ".")
		if len(oldParts) != 2 || len(newParts) != 2 {
			continue
		}
		oldSection, _ := old[oldParts[0]].(map[string]interface{})
		if oldSection == nil {
			continue
		}
		val, has := oldSection[oldParts[1]]
		if !has {
			continue
		}
		newSection, ok := new[newParts[0]].(map[string]interface{})
		if !ok {
			new[newParts[0]] = map[string]interface{}{newParts[1]: val}
			continue
		}
		if _, exists := newSection[newParts[1]]; !exists {
			newSection[newParts[1]] = val
		}
	}
}

// MigrateToCurrentSchema takes a raw config map (from an older or current schema),
// loads the current schema from the embedded default_config.yaml, copies over matching keys,
// applies explicit migration rules for renames/structural changes, and returns
// an *AppConfig at CurrentConfigVersion.
func MigrateToCurrentSchema(oldRaw map[string]interface{}) (*AppConfig, error) {
	// Normalize legacy version key so copyMatchingKeys can match (schema uses "Version")
	if v, ok := oldRaw["version"].(string); ok && v != "" {
		oldRaw["Version"] = v
	}

	// Start from embedded default schema (same source as DefaultUserPreferences)
	newMap, err := defaultConfigMap()
	if err != nil {
		return nil, fmt.Errorf("migrate: load default config: %w", err)
	}

	// Copy values from old config where key path exists in new schema
	copyMatchingKeys(oldRaw, newMap)

	// Apply explicit rules for renames and structural changes
	applyExplicitMigrationRules(oldRaw, newMap)

	// Normalize version key to current schema (Version)
	newMap["Version"] = CurrentConfigVersion
	delete(newMap, "version")
	delete(newMap, "ConfigVersion")

	// Unmarshal into AppConfig
	finalBytes, err := yaml.Marshal(newMap)
	if err != nil {
		return nil, fmt.Errorf("migrate: marshal migrated map: %w", err)
	}
	var result AppConfig
	if err := yaml.Unmarshal(finalBytes, &result); err != nil {
		return nil, fmt.Errorf("migrate: unmarshal to config: %w", err)
	}
	result.ConfigVersion = CurrentConfigVersion
	return &result, nil
}

// NeedsSchemaMigration returns true if the raw config has an older or missing schema version.
func NeedsSchemaMigration(raw map[string]interface{}) bool {
	v := getConfigVersionFromRaw(raw)
	return v != CurrentConfigVersion
}
