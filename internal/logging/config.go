package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// DefaultConfig returns the default logging configuration
func DefaultConfig() Config {
	return Config{
		Level:      InfoLevel,
		MaxSize:    10,   // 10MB
		MaxBackups: 5,    // Keep 5 backup files
		MaxAge:     30,   // 30 days
		Compress:   true, // Compress old logs
	}
}

// LoadConfig loads the logging configuration from a file
func LoadConfig(configPath string) (Config, error) {
	// If no config path is provided, use the default location
	if configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return DefaultConfig(), fmt.Errorf("failed to get user home directory: %w", err)
		}

		switch runtime.GOOS {
		case "windows":
			configPath = filepath.Join(os.Getenv("LOCALAPPDATA"), "FontGet", "config", "logging.json")
		case "darwin":
			configPath = filepath.Join(homeDir, "Library", "Application Support", "fontget", "config", "logging.json")
		default: // Linux and others
			configPath = filepath.Join(homeDir, ".config", "fontget", "logging.json")
		}
	}

	// Create config directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return DefaultConfig(), fmt.Errorf("failed to create config directory: %w", err)
	}

	// If config file doesn't exist, create it with defaults
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := DefaultConfig()
		if err := SaveConfig(configPath, config); err != nil {
			return config, fmt.Errorf("failed to create default config: %w", err)
		}
		return config, nil
	}

	// Read and parse the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return DefaultConfig(), fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return DefaultConfig(), fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

// SaveConfig saves the logging configuration to a file
func SaveConfig(configPath string, config Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// UpdateConfig updates specific fields in the logging configuration
func UpdateConfig(configPath string, updates map[string]interface{}) error {
	config, err := LoadConfig(configPath)
	if err != nil {
		return err
	}

	// Update config fields based on the updates map
	if level, ok := updates["level"].(int); ok {
		config.Level = LogLevel(level)
	}
	if maxSize, ok := updates["maxSize"].(int); ok {
		config.MaxSize = maxSize
	}
	if maxBackups, ok := updates["maxBackups"].(int); ok {
		config.MaxBackups = maxBackups
	}
	if maxAge, ok := updates["maxAge"].(int); ok {
		config.MaxAge = maxAge
	}
	if compress, ok := updates["compress"].(bool); ok {
		config.Compress = compress
	}

	return SaveConfig(configPath, config)
}
