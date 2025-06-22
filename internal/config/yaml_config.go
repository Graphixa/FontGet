package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

// YAMLConfig represents the main YAML configuration structure
type YAMLConfig struct {
	Configuration ConfigurationSection `yaml:"Configuration"`
	Logging       LoggingSection       `yaml:"Logging"`
}

// ConfigurationSection represents the main configuration settings
type ConfigurationSection struct {
	DefaultEditor string `yaml:"DefaultEditor"`
}

// LoggingSection represents logging configuration
type LoggingSection struct {
	LogPath  string `yaml:"LogPath"`
	MaxSize  string `yaml:"MaxSize"`
	MaxFiles int    `yaml:"MaxFiles"`
}

// GetYAMLConfigDir returns the platform-specific YAML config directory
func GetYAMLConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	var configDir string
	switch runtime.GOOS {
	case "windows":
		configDir = filepath.Join(homeDir, ".fontget")
	case "darwin":
		configDir = filepath.Join(homeDir, ".fontget")
	default: // Linux and others
		configDir = filepath.Join(homeDir, ".fontget")
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return configDir, nil
}

// GetYAMLConfigPath returns the path to the YAML config file
func GetYAMLConfigPath() (string, error) {
	configDir, err := GetYAMLConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "config.yaml"), nil
}

// DefaultYAMLConfig returns a new default YAML configuration
func DefaultYAMLConfig() *YAMLConfig {
	return &YAMLConfig{
		Configuration: ConfigurationSection{
			DefaultEditor: "", // Use system default editor
		},
		Logging: LoggingSection{
			LogPath:  "$home/.fontget/logs/fontget.log",
			MaxSize:  "10MB",
			MaxFiles: 5,
		},
	}
}

// GetEditorWithFallback returns the configured editor or system default
func GetEditorWithFallback(config *YAMLConfig) string {
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

// LoadYAMLConfig loads the YAML configuration from file
func LoadYAMLConfig() (*YAMLConfig, error) {
	configPath, err := GetYAMLConfigPath()
	if err != nil {
		return nil, err
	}

	// If config file doesn't exist, create it with comments and return default config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := GenerateInitialYAMLConfig(); err != nil {
			return nil, fmt.Errorf("failed to create initial config file: %w", err)
		}
		return DefaultYAMLConfig(), nil
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
	if err := ValidateStrictYAMLConfig(rawData); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// If validation passes, parse into structured config
	var config YAMLConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config file: %w", err)
	}

	return &config, nil
}

// GenerateInitialYAMLConfig generates a new config file with helpful comments
func GenerateInitialYAMLConfig() error {
	configPath, err := GetYAMLConfigPath()
	if err != nil {
		return err
	}
	return saveDefaultYAMLConfigWithComments(configPath)
}

// SaveYAMLConfig saves the YAML configuration to file
func SaveYAMLConfig(config *YAMLConfig) error {
	configPath, err := GetYAMLConfigPath()
	if err != nil {
		return err
	}

	// Marshal config to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML config: %w", err)
	}

	// Write config file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write YAML config file: %w", err)
	}

	return nil
}

// saveDefaultYAMLConfigWithComments saves a default config with a single DefaultEditor line and multiplatform comment
func saveDefaultYAMLConfigWithComments(configPath string) error {
	configContent := `Configuration:
  DefaultEditor: "" # set your own default editor for fontget (e.g. 'code', 'notepad.exe', 'nano', etc.)
Logging:
  LogPath: "$home/.fontget/logs/fontget.log"
  MaxSize: "10MB"
  MaxFiles: 5
`

	// Write config file
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write YAML config file: %w", err)
	}

	return nil
}

// ValidateYAMLConfig validates the YAML configuration
func ValidateYAMLConfig(config *YAMLConfig) error {
	// Convert structured config to map for validation
	rawData := map[string]interface{}{
		"Configuration": map[string]interface{}{
			"DefaultEditor": config.Configuration.DefaultEditor,
		},
		"Logging": map[string]interface{}{
			"LogPath":  config.Logging.LogPath,
			"MaxSize":  config.Logging.MaxSize,
			"MaxFiles": config.Logging.MaxFiles,
		},
	}

	return ValidateStrictYAMLConfig(rawData)
}
