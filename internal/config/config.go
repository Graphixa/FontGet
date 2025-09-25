package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// UserConfig represents the user's configuration
type UserConfig struct {
	FirstRunCompleted  bool                    `json:"first_run_completed"`
	AcceptedSources    map[string]SourceAccept `json:"accepted_sources"`
	SourcesLastUpdated time.Time               `json:"sources_last_updated,omitempty"`
}

// SourceAccept represents acceptance of a font source
type SourceAccept struct {
	Accepted     bool      `json:"accepted"`
	AcceptedDate time.Time `json:"accepted_date"`
}

// DefaultConfig returns a new default user configuration
func DefaultConfig() *UserConfig {
	return &UserConfig{
		FirstRunCompleted: false,
		AcceptedSources:   make(map[string]SourceAccept),
	}
}

// GetConfigDir returns the platform-specific config directory
func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	var configDir string
	switch runtime.GOOS {
	case "windows":
		configDir = filepath.Join(homeDir, "AppData", "Local", "FontGet")
	case "darwin":
		configDir = filepath.Join(homeDir, "Library", "Application Support", "FontGet")
	default: // Linux and others
		configDir = filepath.Join(homeDir, ".config", "fontget")
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return configDir, nil
}

// GetConfigPath returns the path to the user config file
func GetConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "config.json"), nil
}

// LoadConfig loads the user configuration from file
func LoadConfig() (*UserConfig, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// If config file doesn't exist, return default config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config UserConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Ensure AcceptedSources is initialized
	if config.AcceptedSources == nil {
		config.AcceptedSources = make(map[string]SourceAccept)
	}

	return &config, nil
}

// SaveConfig saves the user configuration to file
func SaveConfig(config *UserConfig) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Marshal config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write config file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// IsFirstRun checks if this is the user's first time running FontGet
func IsFirstRun() (bool, error) {
	config, err := LoadConfig()
	if err != nil {
		return false, err
	}
	return !config.FirstRunCompleted, nil
}

// MarkFirstRunCompleted marks the first run as completed
func MarkFirstRunCompleted() error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	config.FirstRunCompleted = true
	return SaveConfig(config)
}

// AcceptSource marks a source as accepted
func AcceptSource(sourceName string) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	config.AcceptedSources[sourceName] = SourceAccept{
		Accepted:     true,
		AcceptedDate: time.Now(),
	}

	return SaveConfig(config)
}

// IsSourceAccepted checks if a source has been accepted
func IsSourceAccepted(sourceName string) (bool, error) {
	config, err := LoadConfig()
	if err != nil {
		return false, err
	}

	if source, exists := config.AcceptedSources[sourceName]; exists {
		return source.Accepted, nil
	}

	return false, nil
}

// UpdateSourcesLastUpdated updates the timestamp when sources were last updated
func UpdateSourcesLastUpdated() error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	config.SourcesLastUpdated = time.Now()
	return SaveConfig(config)
}

// GetSourcesLastUpdated returns when sources were last updated
func GetSourcesLastUpdated() (time.Time, error) {
	config, err := LoadConfig()
	if err != nil {
		return time.Time{}, err
	}

	return config.SourcesLastUpdated, nil
}

// ShouldRefreshSources checks if sources should be refreshed (>24 hours old)
func ShouldRefreshSources() (bool, error) {
	lastUpdated, err := GetSourcesLastUpdated()
	if err != nil {
		return true, err // If we can't determine, assume we should refresh
	}

	// If never updated, we should refresh
	if lastUpdated.IsZero() {
		return true, nil
	}

	// Check if more than 24 hours have passed
	return time.Since(lastUpdated) > 24*time.Hour, nil
}
