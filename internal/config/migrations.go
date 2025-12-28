package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// MigrationFunc represents a function that migrates config from one version to another
type MigrationFunc func(*AppConfig) error

// Migration represents a single migration step
type Migration struct {
	FromVersion string
	ToVersion   string
	Migrate     MigrationFunc
	Description string
}

// migrations is the registry of all available migrations
var migrations = []Migration{
	{
		FromVersion: "1.0",
		ToVersion:   "2.0",
		Migrate:     migrateV1ToV2,
		Description: "Remove Limits section, add Search section",
	},
}

// CurrentConfigVersion is the current config schema version
const CurrentConfigVersion = "2.0"

// migrateV1ToV2 migrates config from version 1.0 to 2.0
// CRITICAL: Preserves ALL user custom values
func migrateV1ToV2(config *AppConfig) error {
	// The config already has all user values loaded from the old config file
	// We just need to:
	// 1. Ensure Search section exists (add with defaults if missing - v1.0 didn't have Search)
	// 2. Set ConfigVersion to 2.0
	// 3. Limits section is already ignored (not in struct), so it's effectively removed

	// If Search.ResultLimit is still at default (0) and we're migrating from v1.0,
	// ensure Search section is properly initialized (though it should already be from defaults)
	// This is mainly for clarity - the Search section should already exist from DefaultUserPreferences
	if config.Search.ResultLimit == 0 {
		// Ensure Search section is set (default: unlimited)
		config.Search = SearchSection{
			ResultLimit: 0, // Default: unlimited
		}
	}

	// Set version to 2.0
	config.ConfigVersion = "2.0"

	return nil
}

// createBackup creates a backup of the config file before migration
func createBackup(configPath string) (string, error) {
	// Read the original config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read config file for backup: %w", err)
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	backupPath := fmt.Sprintf("%s.backup.%s", configPath, timestamp)

	// Write backup file
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write backup file: %w", err)
	}

	return backupPath, nil
}

// rotateBackups keeps only the last N backups, removing older ones
func rotateBackups(configPath string, keepCount int) error {
	configDir := filepath.Dir(configPath)
	configBase := filepath.Base(configPath)

	// Find all backup files
	entries, err := os.ReadDir(configDir)
	if err != nil {
		return fmt.Errorf("failed to read config directory: %w", err)
	}

	var backups []string
	backupPrefix := configBase + ".backup."
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), backupPrefix) {
			backupPath := filepath.Join(configDir, entry.Name())
			// Get file info for sorting by modification time
			info, err := entry.Info()
			if err != nil {
				continue
			}
			backups = append(backups, backupPath)
			_ = info // We'll sort by path which includes timestamp
		}
	}

	// Sort backups by name (timestamp is in filename, so this sorts by time)
	sort.Sort(sort.Reverse(sort.StringSlice(backups)))

	// Remove backups beyond keepCount
	if len(backups) > keepCount {
		for i := keepCount; i < len(backups); i++ {
			if err := os.Remove(backups[i]); err != nil {
				// Log but don't fail - backup cleanup is best effort
				continue
			}
		}
	}

	return nil
}

// getConfigVersion returns the config version, or "1.0" if not set (for backward compatibility)
func getConfigVersion(config *AppConfig) string {
	if config.ConfigVersion == "" {
		return "1.0" // Assume v1.0 for old configs without version
	}
	return config.ConfigVersion
}

// needsMigration checks if the config needs migration
func needsMigration(config *AppConfig) bool {
	version := getConfigVersion(config)
	return version != CurrentConfigVersion
}

// RunMigrations runs all applicable migrations for the config
// Returns the migrated config and any error
func RunMigrations(config *AppConfig, configPath string) (*AppConfig, error) {
	currentVersion := getConfigVersion(config)

	// If already at current version, no migration needed
	if currentVersion == CurrentConfigVersion {
		return config, nil
	}

	// Create backup before migration
	backupPath, err := createBackup(configPath)
	if err != nil {
		// Log warning but continue - backup failure shouldn't block migration
		// In production, you might want to use a logger here
		_ = backupPath
	} else {
		// Rotate backups (keep last 3)
		_ = rotateBackups(configPath, 3)
	}

	// Find and run applicable migrations
	for _, migration := range migrations {
		if migration.FromVersion == currentVersion {
			// Run this migration
			if err := migration.Migrate(config); err != nil {
				// Migration failed - try to restore from backup
				if backupPath != "" {
					if restoreErr := restoreFromBackup(configPath, backupPath); restoreErr != nil {
						return nil, fmt.Errorf("migration failed and backup restore failed: migration error: %w, restore error: %v", err, restoreErr)
					}
				}
				return nil, fmt.Errorf("migration from %s to %s failed: %w", migration.FromVersion, migration.ToVersion, err)
			}

			// Update current version for next migration in chain
			currentVersion = migration.ToVersion

			// If we've reached the target version, stop
			if currentVersion == CurrentConfigVersion {
				break
			}
		}
	}

	// If we didn't reach the target version, there might be missing migrations
	if currentVersion != CurrentConfigVersion {
		return nil, fmt.Errorf("no migration path found from version %s to %s", getConfigVersion(config), CurrentConfigVersion)
	}

	return config, nil
}

// restoreFromBackup restores the config file from a backup
func restoreFromBackup(configPath, backupPath string) error {
	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	if err := os.WriteFile(configPath, backupData, 0644); err != nil {
		return fmt.Errorf("failed to restore config from backup: %w", err)
	}

	return nil
}
