package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"fontget/internal/platform"

	"gopkg.in/yaml.v3"
)

// AppConfig represents the main application configuration structure
type AppConfig struct {
	Configuration ConfigurationSection `yaml:"Configuration"`
	Logging       LoggingSection       `yaml:"Logging"`
}

// ConfigurationSection represents the main configuration settings
type ConfigurationSection struct {
	DefaultEditor     string `yaml:"DefaultEditor"`
	UsePopularitySort bool   `yaml:"UsePopularitySort"` // Enable/disable popularity scoring
}

// LoggingSection represents logging configuration
type LoggingSection struct {
	LogPath  string `yaml:"LogPath"`
	MaxSize  string `yaml:"MaxSize"`
	MaxFiles int    `yaml:"MaxFiles"`
}

// GetAppConfigDir returns the app-specific config directory (~/.fontget)
// This is now consolidated with the manifest system
func GetAppConfigDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if we can't get home directory
		// This is a graceful degradation - system continues to work
		// Note: In production, this should be very rare (only in containerized/restricted environments)
		return ".fontget"
	}

	configDir := filepath.Join(homeDir, ".fontget")

	// Create config directory if it doesn't exist
	// We ignore the error here because:
	// 1. If directory creation fails, subsequent file operations will catch it
	// 2. This keeps the API simple and allows graceful degradation
	// 3. Create as hidden directory for better UX
	platform.CreateHiddenDirectory(configDir, 0755)

	return configDir
}

// GetAppConfigPath returns the path to the app config file
func GetAppConfigPath() string {
	configDir := GetAppConfigDir()
	return filepath.Join(configDir, "config.yaml")
}

// DefaultUserPreferences returns a new default user preferences configuration
func DefaultUserPreferences() *AppConfig {
	return &AppConfig{
		Configuration: ConfigurationSection{
			DefaultEditor:     "",   // Use system default editor
			UsePopularitySort: true, // Default to popularity-based sorting
		},
		Logging: LoggingSection{
			LogPath:  "$home/.fontget/logs/fontget.log",
			MaxSize:  "10MB",
			MaxFiles: 5,
		},
	}
}

// GetUserPreferences loads user preferences from config file or returns defaults
func GetUserPreferences() *AppConfig {
	// Use GetAppConfigPath() to get config.yaml, not GetConfigPath() which returns config.json
	configPath := GetAppConfigPath()

	// Start with defaults
	config := DefaultUserPreferences()

	// Try to load existing config and merge with defaults
	if data, err := os.ReadFile(configPath); err == nil {
		var loadedConfig AppConfig
		if err := yaml.Unmarshal(data, &loadedConfig); err == nil {
			// Merge loaded config with defaults
			config.Configuration.DefaultEditor = loadedConfig.Configuration.DefaultEditor

			// For bool, we need to check if the field was explicitly set in YAML
			// Since Go's zero value for bool is false, we can't distinguish between
			// "not set" and "explicitly set to false". For now, we'll assume if the
			// file exists and is valid, use the loaded values.
			config.Configuration.UsePopularitySort = loadedConfig.Configuration.UsePopularitySort
		}
	}

	return config
}

// MarshalYAML marshals the AppConfig to YAML format
func (a *AppConfig) MarshalYAML() ([]byte, error) {
	return yaml.Marshal(a)
}

// GetEditorWithFallback returns the configured editor or system default
func GetEditorWithFallback(config *AppConfig) string {
	if config.Configuration.DefaultEditor != "" {
		return config.Configuration.DefaultEditor
	}
	return getDefaultEditor()
}

// getDefaultEditor returns the platform-specific default editor
func getDefaultEditor() string {
	switch runtime.GOOS {
	case "windows":
		return "notepad.exe"
	case "darwin":
		return "open -e"
	default: // Linux and others
		return "nano"
	}
}

// LoadUserPreferences loads the user preferences from file
func LoadUserPreferences() (*AppConfig, error) {
	configPath := GetAppConfigPath()

	// If config file doesn't exist, create it with comments and return default config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := GenerateInitialUserPreferences(); err != nil {
			return nil, fmt.Errorf("failed to create initial config file: %w", err)
		}
		return DefaultUserPreferences(), nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// First, parse as generic map for strict validation
	var rawData map[string]interface{}
	if err := yaml.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config file: %w", err)
	}

	// Perform strict validation
	if err := ValidateStrictAppConfig(rawData); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// If validation passes, parse into structured config
	var config AppConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse app config file: %w", err)
	}

	return &config, nil
}

// GenerateInitialUserPreferences generates a new user preferences file with helpful comments
func GenerateInitialUserPreferences() error {
	configPath := GetAppConfigPath()
	return saveDefaultAppConfigWithComments(configPath)
}

// SaveUserPreferences saves the user preferences to file
func SaveUserPreferences(config *AppConfig) error {
	configPath := GetAppConfigPath()

	// Marshal config to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML config: %w", err)
	}

	// Write config file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write app config file: %w", err)
	}

	return nil
}

// saveDefaultAppConfigWithComments saves a default config with a single DefaultEditor line and multiplatform comment
func saveDefaultAppConfigWithComments(configPath string) error {
	configContent := `Configuration:
  DefaultEditor: "" # set your own default editor for fontget (e.g. 'code', 'notepad.exe', 'nano', etc.)
Logging:
  LogPath: "$home/.fontget/logs/fontget.log"
  MaxSize: "10MB"
  MaxFiles: 5
`

	// Write config file
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write app config file: %w", err)
	}

	return nil
}

// ValidateUserPreferences validates the user preferences
func ValidateUserPreferences(config *AppConfig) error {
	// Convert structured config to map for validation
	rawData := map[string]interface{}{
		"Configuration": map[string]interface{}{
			"DefaultEditor":     config.Configuration.DefaultEditor,
			"UsePopularitySort": config.Configuration.UsePopularitySort,
		},
		"Logging": map[string]interface{}{
			"LogPath":  config.Logging.LogPath,
			"MaxSize":  config.Logging.MaxSize,
			"MaxFiles": config.Logging.MaxFiles,
		},
	}

	return ValidateStrictAppConfig(rawData)
}
