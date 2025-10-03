package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"fontget/internal/platform"
	"fontget/internal/sources"
	"fontget/internal/version"
)

// Manifest represents the new sources configuration system
type Manifest struct {
	Version        string                  `json:"version"`
	Created        time.Time               `json:"created"`
	LastUpdated    time.Time               `json:"last_updated"`
	FontGetVersion string                  `json:"fontget_version"`
	Sources        map[string]SourceConfig `json:"sources"`
	CachePolicy    CachePolicy             `json:"cache_policy"`
}

// SourceConfig represents configuration for a single font source
type SourceConfig struct {
	URL        string     `json:"url"`
	Prefix     string     `json:"prefix"`
	Enabled    bool       `json:"enabled"`
	Filename   string     `json:"filename"`
	Priority   int        `json:"priority"`
	LastSynced *time.Time `json:"last_synced"`
	FontCount  int        `json:"font_count"`
	Version    string     `json:"version"`
}

// CachePolicy defines cache management behavior
type CachePolicy struct {
	AutoUpdateDays int  `json:"auto_update_days"`
	CheckOnStartup bool `json:"check_on_startup"`
}

// SourceData represents the structure of downloaded source files
type SourceData struct {
	SourceInfo struct {
		Name        string    `json:"name"`
		Description string    `json:"description"`
		URL         string    `json:"url"`
		Version     string    `json:"version"`
		LastUpdated time.Time `json:"last_updated"`
		TotalFonts  int       `json:"total_fonts"`
	} `json:"source_info"`
	Fonts map[string]interface{} `json:"fonts"`
}

// EnsureManifestExists creates and bootstraps the manifest system if it doesn't exist
func EnsureManifestExists() error {
	manifestPath := getManifestPath()

	if !fileExists(manifestPath) {
		// Phase 1: Create default manifest
		manifest, err := createDefaultManifest()
		if err != nil {
			return fmt.Errorf("failed to create default manifest: %v", err)
		}

		// Phase 2: Download and verify each enabled source
		for sourceName, source := range manifest.Sources {
			if source.Enabled {
				sourceData, err := downloadAndParseSource(source.URL, source.Filename)
				if err != nil {
					// Continue with other sources if one fails
					fmt.Printf("Warning: Failed to bootstrap source %s: %v\n", sourceName, err)
					continue
				}

				// Phase 3: Read actual display name from source_info.name
				actualName := sourceData.SourceInfo.Name

				// Phase 4: Update manifest with verified data
				sourceConfig := manifest.Sources[sourceName]
				now := time.Now()
				sourceConfig.LastSynced = &now
				sourceConfig.FontCount = len(sourceData.Fonts)
				sourceConfig.Version = sourceData.SourceInfo.Version

				// Phase 5: Handle name changes
				if actualName != sourceName {
					manifest.Sources[actualName] = sourceConfig
					delete(manifest.Sources, sourceName)
				} else {
					manifest.Sources[sourceName] = sourceConfig
				}
			}
		}

		// Phase 6: Save verified manifest
		manifest.LastUpdated = time.Now()
		return saveManifest(manifest)
	}

	return validateExistingManifest(manifestPath)
}

// downloadAndParseSource downloads a source file and parses its content
func downloadAndParseSource(url, filename string) (*SourceData, error) {
	// Download to ~/.fontget/sources/{filename}
	sourcePath := filepath.Join(getSourcesDir(), filename)

	// Ensure sources directory exists (as hidden directory)
	if err := platform.CreateHiddenDirectory(getSourcesDir(), 0755); err != nil {
		return nil, fmt.Errorf("failed to create sources directory: %v", err)
	}

	// Download source file
	if err := downloadFile(url, sourcePath); err != nil {
		return nil, fmt.Errorf("failed to download source: %v", err)
	}

	// Parse and validate source data
	return parseSourceFile(sourcePath)
}

// LoadManifest loads the manifest from disk
func LoadManifest() (*Manifest, error) {
	// Ensure manifest exists (auto-bootstrap)
	if err := EnsureManifestExists(); err != nil {
		return nil, fmt.Errorf("failed to ensure manifest exists: %v", err)
	}

	manifestPath := getManifestPath()
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %v", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %v", err)
	}

	// Note: Source files are downloaded on-demand by the repo system
	// No need for self-healing here as it causes unwanted warnings

	return &manifest, nil
}

// SaveManifest saves the manifest to disk
func SaveManifest(manifest *Manifest) error {
	return saveManifest(manifest)
}

// GetEnabledSourcesFromManifest returns enabled sources in proper order
func GetEnabledSourcesFromManifest() (map[string]SourceConfig, error) {
	manifest, err := LoadManifest()
	if err != nil {
		return nil, err
	}

	enabled := make(map[string]SourceConfig)
	for name, source := range manifest.Sources {
		if source.Enabled {
			enabled[name] = source
		}
	}

	return enabled, nil
}

// Helper functions

func getManifestPath() string {
	configDir := GetAppConfigDir()
	return filepath.Join(configDir, "manifest.json")
}

func getSourcesDir() string {
	configDir := GetAppConfigDir()
	return filepath.Join(configDir, "sources")
}

// Note: GetAppConfigDir is now defined in app_config.go to avoid duplication

func createDefaultManifest() (*Manifest, error) {
	now := time.Now()

	// Get default sources from centralized location
	defaultSources := sources.DefaultSources()

	// Convert to SourceConfig format
	manifestSources := make(map[string]SourceConfig)
	for name, info := range defaultSources {
		manifestSources[name] = SourceConfig{
			URL:      info.URL,
			Prefix:   info.Prefix,
			Enabled:  info.Enabled,
			Filename: info.Filename,
			Priority: info.Priority,
		}
	}

	return &Manifest{
		Version:        version.GetManifestVersion(),
		Created:        now,
		LastUpdated:    now,
		FontGetVersion: version.GetVersion(),
		Sources:        manifestSources,
		CachePolicy: CachePolicy{
			AutoUpdateDays: 7,
			CheckOnStartup: false,
		},
	}, nil
}

func saveManifest(manifest *Manifest) error {
	manifestPath := getManifestPath()

	// Ensure config directory exists (as hidden directory)
	if err := platform.CreateHiddenDirectory(filepath.Dir(manifestPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %v", err)
	}

	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %v", err)
	}

	return nil
}

func downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func parseSourceFile(filepath string) (*SourceData, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var sourceData SourceData
	if err := json.Unmarshal(data, &sourceData); err != nil {
		return nil, fmt.Errorf("failed to parse source file: %v", err)
	}

	return &sourceData, nil
}

func fileExists(filepath string) bool {
	_, err := os.Stat(filepath)
	return err == nil
}

func validateExistingManifest(manifestPath string) error {
	// Basic validation - check if file exists and is readable
	if _, err := os.Stat(manifestPath); err != nil {
		return fmt.Errorf("manifest file not found: %w", err)
	}
	return nil
}

// GetSourceByName returns a source by name from the manifest
func GetSourceByName(manifest *Manifest, name string) (*SourceConfig, bool) {
	source, exists := manifest.Sources[name]
	if !exists {
		return nil, false
	}
	return &source, true
}

// GetDefaultManifest returns a default manifest with built-in sources
func GetDefaultManifest() (*Manifest, error) {
	return createDefaultManifest()
}
