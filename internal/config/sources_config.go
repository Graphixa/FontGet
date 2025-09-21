package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SourcesConfig represents the sources configuration structure
type SourcesConfig struct {
	Sources map[string]Source `json:"sources"`
}

// Source represents a font source configuration
type Source struct {
	Path    string `json:"path"`
	Prefix  string `json:"prefix"`
	Enabled bool   `json:"enabled"`
}

// GetSourcesConfigPath returns the path to the sources config file
func GetSourcesConfigPath() (string, error) {
	configDir, err := GetYAMLConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "sources.json"), nil
}

// DefaultSourcesConfig returns a new default sources configuration
func DefaultSourcesConfig() *SourcesConfig {
	return &SourcesConfig{
		Sources: map[string]Source{
			"Google": {
				Path:    "https://raw.githubusercontent.com/Graphixa/FontGet-Sources/main/sources/google-fonts.json",
				Prefix:  "google",
				Enabled: true,
			},
			"NerdFonts": {
				Path:    "https://raw.githubusercontent.com/Graphixa/FontGet-Sources/main/sources/nerd-fonts.json",
				Prefix:  "nerd",
				Enabled: true,
			},
			"FontSquirrel": {
				Path:    "https://raw.githubusercontent.com/Graphixa/FontGet-Sources/main/sources/font-squirrel.json",
				Prefix:  "squirrel",
				Enabled: false,
			},
		},
	}
}

// LoadSourcesConfig loads the sources configuration from file
func LoadSourcesConfig() (*SourcesConfig, error) {
	configPath, err := GetSourcesConfigPath()
	if err != nil {
		return nil, err
	}

	// If config file doesn't exist, return default config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultSourcesConfig(), nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read sources config file: %w", err)
	}

	// First, parse as generic map for strict validation
	var rawData map[string]interface{}
	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse sources config file: %w", err)
	}

	// Perform strict validation
	if err := ValidateStrictSourcesConfig(rawData); err != nil {
		return nil, fmt.Errorf("sources configuration validation failed: %w", err)
	}

	// If validation passes, parse into structured config
	var config SourcesConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse sources config file: %w", err)
	}

	return &config, nil
}

// SaveSourcesConfig saves the sources configuration to file
func SaveSourcesConfig(config *SourcesConfig) error {
	configPath, err := GetSourcesConfigPath()
	if err != nil {
		return err
	}

	// Marshal config to JSON with indentation
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal sources config: %w", err)
	}

	// Write config file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write sources config file: %w", err)
	}

	return nil
}

// ValidateSourcesConfig validates the sources configuration
func ValidateSourcesConfig(config *SourcesConfig) error {
	// Convert structured config to map for validation
	rawData := map[string]interface{}{
		"sources": make(map[string]interface{}),
	}

	for name, source := range config.Sources {
		rawData["sources"].(map[string]interface{})[name] = map[string]interface{}{
			"path":    source.Path,
			"prefix":  source.Prefix,
			"enabled": source.Enabled,
		}
	}

	return ValidateStrictSourcesConfig(rawData)
}

// GetEnabledSources returns a list of enabled sources in order
func GetEnabledSources(config *SourcesConfig) []string {
	var enabled []string
	for name, source := range config.Sources {
		if source.Enabled {
			enabled = append(enabled, name)
		}
	}
	return enabled
}

// GetEnabledSourcesInOrder returns enabled sources in a specific order:
// 1. Built-in sources: Google, NerdFonts, FontSquirrel
// 2. User-added sources in the order they appear in the config file
func GetEnabledSourcesInOrder(config *SourcesConfig) []string {
	var enabled []string

	// Define built-in source order
	builtInOrder := []string{"Google", "NerdFonts", "FontSquirrel"}

	// First, add built-in sources in the specified order
	for _, name := range builtInOrder {
		if source, exists := config.Sources[name]; exists && source.Enabled {
			enabled = append(enabled, name)
		}
	}

	// Then, add user-added sources (not built-in) in the order they appear in the config
	for name, source := range config.Sources {
		if !IsBuiltInSource(name) && source.Enabled {
			enabled = append(enabled, name)
		}
	}

	return enabled
}

// GetSourceByPrefix returns a source by its prefix
func GetSourceByPrefix(config *SourcesConfig, prefix string) (string, *Source, bool) {
	for name, source := range config.Sources {
		if source.Prefix == prefix {
			return name, &source, true
		}
	}
	return "", nil, false
}

// GetSourceByName returns a source by its name
func GetSourceByName(config *SourcesConfig, name string) (*Source, bool) {
	if source, exists := config.Sources[name]; exists {
		return &source, true
	}
	return nil, false
}

// IsBuiltInSource checks if a source name is a built-in source
func IsBuiltInSource(name string) bool {
	builtInSources := []string{"Google", "NerdFonts", "FontSquirrel"}
	for _, builtIn := range builtInSources {
		if name == builtIn {
			return true
		}
	}
	return false
}

// AddSource adds a new source to the configuration
func AddSource(config *SourcesConfig, name string, source Source) error {
	if name == "" {
		return fmt.Errorf("source name cannot be empty")
	}

	if err := ValidateSourcesConfig(config); err != nil {
		return fmt.Errorf("invalid source configuration: %w", err)
	}

	config.Sources[name] = source
	return nil
}

// RemoveSource removes a source from the configuration
func RemoveSource(config *SourcesConfig, name string) error {
	if name == "" {
		return fmt.Errorf("source name cannot be empty")
	}

	if _, exists := config.Sources[name]; !exists {
		return fmt.Errorf("source '%s' does not exist", name)
	}

	delete(config.Sources, name)
	return nil
}

// EnableSource enables a source
func EnableSource(config *SourcesConfig, name string) error {
	if source, exists := config.Sources[name]; exists {
		source.Enabled = true
		config.Sources[name] = source
		return nil
	}
	return fmt.Errorf("source '%s' does not exist", name)
}

// DisableSource disables a source
func DisableSource(config *SourcesConfig, name string) error {
	if source, exists := config.Sources[name]; exists {
		source.Enabled = false
		config.Sources[name] = source
		return nil
	}
	return fmt.Errorf("source '%s' does not exist", name)
}
