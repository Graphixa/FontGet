package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"fontget/internal/platform"

	"gopkg.in/yaml.v3"
)

// AppConfig represents the main application configuration structure
type AppConfig struct {
	Configuration ConfigurationSection `yaml:"Configuration"`
	Logging       LoggingSection       `yaml:"Logging"`
	Network       NetworkSection       `yaml:"Network"`
	Limits        LimitsSection        `yaml:"Limits"`
	Update        UpdateSection        `yaml:"Update"`
	Theme         ThemeSection         `yaml:"Theme"`
}

// ConfigurationSection represents the main configuration settings
type ConfigurationSection struct {
	DefaultEditor        string `yaml:"DefaultEditor"`
	EnablePopularitySort bool   `yaml:"EnablePopularitySort"` // Enable/disable popularity scoring
}

// LoggingSection represents logging configuration
type LoggingSection struct {
	LogPath     string `yaml:"LogPath"`
	MaxLogSize  string `yaml:"MaxLogSize"`
	MaxLogFiles int    `yaml:"MaxLogFiles"`
}

// NetworkSection represents network configuration
type NetworkSection struct {
	RequestTimeout  string `yaml:"RequestTimeout"`  // Quick HTTP requests and checks (e.g., "10s")
	DownloadTimeout string `yaml:"DownloadTimeout"` // Download timeout: max time without data transfer (stall detection) (e.g., "30s")
}

// LimitsSection represents size limits configuration
type LimitsSection struct {
	MaxSourceFileSize  string `yaml:"MaxSourceFileSize"`  // Maximum size for source JSON files (e.g., "50MB")
	FileCopyBufferSize string `yaml:"FileCopyBufferSize"` // Buffer size for file operations (e.g., "32KB")
}

// UpdateSection represents update configuration
type UpdateSection struct {
	AutoCheck           bool   `yaml:"AutoCheck"`           // Check on startup
	AutoUpdate          bool   `yaml:"AutoUpdate"`          // Auto-install (default: false)
	UpdateCheckInterval int    `yaml:"UpdateCheckInterval"` // Hours between checks
	LastUpdateCheck     string `yaml:"LastUpdateCheck"`     // ISO timestamp
	UpdateChannel       string `yaml:"UpdateChannel"`       // stable/beta/nightly
}

// ThemeSection represents theme configuration
type ThemeSection struct {
	Name string `yaml:"Name"` // Theme name (e.g., "catppuccin", "gruvbox") - empty string uses default
	Mode string `yaml:"Mode"` // "auto" (detect from terminal), "dark", or "light" - defaults to "auto"
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
			DefaultEditor:        "",   // Use system default editor
			EnablePopularitySort: true, // Default to popularity-based sorting
		},
		Logging: LoggingSection{
			LogPath:     "$home/.fontget/logs/fontget.log",
			MaxLogSize:  "10MB",
			MaxLogFiles: 5,
		},
		Network: NetworkSection{
			RequestTimeout:  "10s", // Quick HTTP requests and checks
			DownloadTimeout: "30s", // Download timeout: cancel if no data transferred for this duration (stall detection)
		},
		Limits: LimitsSection{
			MaxSourceFileSize:  "50MB",
			FileCopyBufferSize: "32KB",
		},
		Update: UpdateSection{
			AutoCheck:           true,  // Check by default
			AutoUpdate:          false, // Manual install by default
			UpdateCheckInterval: 24,    // Check daily
			LastUpdateCheck:     "",    // Never checked
			UpdateChannel:       "stable",
		},
		Theme: ThemeSection{
			Name: "",     // Empty string uses embedded default theme (catppuccin)
			Mode: "auto", // "auto" detects terminal theme, "dark" or "light" for manual override
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
			config.Configuration.EnablePopularitySort = loadedConfig.Configuration.EnablePopularitySort

			// Merge Update section if it exists (optional for backward compatibility)
			if loadedConfig.Update.UpdateCheckInterval > 0 || loadedConfig.Update.LastUpdateCheck != "" {
				config.Update = loadedConfig.Update
			} else {
				// If Update section doesn't exist, merge individual fields if present
				// This handles partial Update sections in existing configs
				if loadedConfig.Update.AutoCheck || !loadedConfig.Update.AutoCheck {
					config.Update.AutoCheck = loadedConfig.Update.AutoCheck
				}
				if loadedConfig.Update.AutoUpdate || !loadedConfig.Update.AutoUpdate {
					config.Update.AutoUpdate = loadedConfig.Update.AutoUpdate
				}
				if loadedConfig.Update.UpdateCheckInterval > 0 {
					config.Update.UpdateCheckInterval = loadedConfig.Update.UpdateCheckInterval
				}
				if loadedConfig.Update.LastUpdateCheck != "" {
					config.Update.LastUpdateCheck = loadedConfig.Update.LastUpdateCheck
				}
				if loadedConfig.Update.UpdateChannel != "" {
					config.Update.UpdateChannel = loadedConfig.Update.UpdateChannel
				}
			}

			// Merge Logging section if it exists (optional for backward compatibility)
			if loadedConfig.Logging.LogPath != "" {
				config.Logging.LogPath = loadedConfig.Logging.LogPath
			}
			if loadedConfig.Logging.MaxLogSize != "" {
				config.Logging.MaxLogSize = loadedConfig.Logging.MaxLogSize
			}
			if loadedConfig.Logging.MaxLogFiles > 0 {
				config.Logging.MaxLogFiles = loadedConfig.Logging.MaxLogFiles
			}

			// Merge Network section if it exists (optional for backward compatibility)
			if loadedConfig.Network.RequestTimeout != "" {
				config.Network.RequestTimeout = loadedConfig.Network.RequestTimeout
			}
			if loadedConfig.Network.DownloadTimeout != "" {
				config.Network.DownloadTimeout = loadedConfig.Network.DownloadTimeout
			}

			// Backward compatibility: migrate old timeout names to new ones
			// Note: We can't directly access old fields that don't exist in the struct anymore,
			// but YAML unmarshaling will populate them in the raw map if present.
			// For now, we rely on users to manually update their config files.
			// The old timeout names are ignored and defaults are used.

			// Merge Limits section if it exists (optional for backward compatibility)
			if loadedConfig.Limits.MaxSourceFileSize != "" {
				config.Limits.MaxSourceFileSize = loadedConfig.Limits.MaxSourceFileSize
			}
			if loadedConfig.Limits.FileCopyBufferSize != "" {
				config.Limits.FileCopyBufferSize = loadedConfig.Limits.FileCopyBufferSize
			}

			// Merge Theme section if it exists (optional for backward compatibility)
			if loadedConfig.Theme.Name != "" || loadedConfig.Theme.Mode != "" {
				// Start from existing defaults and overlay loaded values
				if loadedConfig.Theme.Name != "" {
					config.Theme.Name = loadedConfig.Theme.Name
				}
				if loadedConfig.Theme.Mode != "" {
					config.Theme.Mode = loadedConfig.Theme.Mode
				}

				// Ensure mode is valid: allow "auto", "dark", "light"
				switch config.Theme.Mode {
				case "", "auto", "dark", "light":
					// Treat empty as "auto"
					if config.Theme.Mode == "" {
						config.Theme.Mode = "auto"
					}
				default:
					// Unknown value - fall back to auto-detection
					config.Theme.Mode = "auto"
				}
			}
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

// ExpandLogPath expands the LogPath from config, replacing $home with actual home directory
func ExpandLogPath(logPath string) (string, error) {
	if logPath == "" {
		return "", fmt.Errorf("log path is empty")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Replace $home with actual home directory (case-insensitive)
	expanded := strings.ReplaceAll(logPath, "$home", homeDir)
	expanded = strings.ReplaceAll(expanded, "$HOME", homeDir)
	expanded = strings.ReplaceAll(expanded, "${home}", homeDir)
	expanded = strings.ReplaceAll(expanded, "${HOME}", homeDir)

	// Also expand ~ if present
	if strings.HasPrefix(expanded, "~") {
		expanded = strings.Replace(expanded, "~", homeDir, 1)
	}

	return expanded, nil
}

// ParseMaxSize parses MaxSize string (e.g., "10MB", "5MB") to megabytes (int)
func ParseMaxSize(maxSizeStr string) (int, error) {
	if maxSizeStr == "" {
		return 10, nil // Default to 10MB
	}

	maxSizeStr = strings.TrimSpace(strings.ToUpper(maxSizeStr))

	// Remove "MB" suffix if present
	maxSizeStr = strings.TrimSuffix(maxSizeStr, "MB")
	maxSizeStr = strings.TrimSpace(maxSizeStr)

	// Parse as integer
	size, err := strconv.Atoi(maxSizeStr)
	if err != nil {
		return 10, fmt.Errorf("invalid MaxSize format: %s (expected format: '10MB')", maxSizeStr)
	}

	if size <= 0 {
		return 10, fmt.Errorf("MaxSize must be greater than 0")
	}

	return size, nil
}

// ParseDuration parses a duration string (e.g., "10s", "5m", "1h") to time.Duration
// Falls back to defaultDuration if parsing fails
func ParseDuration(durationStr string, defaultDuration time.Duration) time.Duration {
	if durationStr == "" {
		return defaultDuration
	}

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return defaultDuration
	}

	return duration
}

// ParseSize parses a size string (e.g., "50MB", "1KB", "2GB") to bytes (int64)
// Falls back to defaultSize if parsing fails
func ParseSize(sizeStr string, defaultSize int64) int64 {
	if sizeStr == "" {
		return defaultSize
	}

	sizeStr = strings.TrimSpace(strings.ToUpper(sizeStr))

	// Extract numeric part and unit
	var numericPart string
	var unit string

	// Find where the number ends
	for i, r := range sizeStr {
		if r < '0' || r > '9' {
			numericPart = sizeStr[:i]
			unit = sizeStr[i:]
			break
		}
	}

	if numericPart == "" {
		return defaultSize
	}

	// Parse numeric part
	size, err := strconv.ParseInt(numericPart, 10, 64)
	if err != nil {
		return defaultSize
	}

	// Convert based on unit
	switch unit {
	case "KB", "K":
		return size * 1024
	case "MB", "M":
		return size * 1024 * 1024
	case "GB", "G":
		return size * 1024 * 1024 * 1024
	case "B", "":
		return size
	default:
		return defaultSize
	}
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
  EnablePopularitySort: true # Fonts will be returned by their match and popularity first, then by alphabetical order. This may mean that popular fonts appear higher in search results.
Logging:
  LogPath: "$home/.fontget/logs/fontget.log"
  MaxLogSize: "10MB"
  MaxLogFiles: 5
Network:
  RequestTimeout: "10s"  # Quick HTTP requests and checks
  DownloadTimeout: "30s" # Download timeout: cancel if no data transferred for this duration (stall detection)
Limits:
  MaxSourceFileSize: "50MB" # Maximum size for source JSON files
  FileCopyBufferSize: "32KB" # Buffer size for file operations
Update:
  AutoCheck: true # Check for updates on startup
  AutoUpdate: false # Automatically install updates (manual by default for security)
  UpdateCheckInterval: 24 # Hours between update checks
  LastUpdateCheck: "" # ISO timestamp of last check (automatically updated)
  UpdateChannel: "stable" # Update channel: stable, beta, or nightly
Theme:
  Name: "" # Theme name (e.g., "catppuccin", "gruvbox") - empty string uses embedded default
  Mode: auto # Theme mode: auto (detect from terminal), dark, or light
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
			"DefaultEditor":        config.Configuration.DefaultEditor,
			"EnablePopularitySort": config.Configuration.EnablePopularitySort,
		},
		"Logging": map[string]interface{}{
			"LogPath":     config.Logging.LogPath,
			"MaxLogSize":  config.Logging.MaxLogSize,
			"MaxLogFiles": config.Logging.MaxLogFiles,
		},
		"Network": map[string]interface{}{
			"RequestTimeout":  config.Network.RequestTimeout,
			"DownloadTimeout": config.Network.DownloadTimeout,
		},
		"Limits": map[string]interface{}{
			"MaxSourceFileSize":  config.Limits.MaxSourceFileSize,
			"FileCopyBufferSize": config.Limits.FileCopyBufferSize,
		},
		"Update": map[string]interface{}{
			"AutoCheck":           config.Update.AutoCheck,
			"AutoUpdate":          config.Update.AutoUpdate,
			"UpdateCheckInterval": config.Update.UpdateCheckInterval,
			"LastUpdateCheck":     config.Update.LastUpdateCheck,
			"UpdateChannel":       config.Update.UpdateChannel,
		},
		"Theme": map[string]interface{}{
			"Name": config.Theme.Name,
			"Mode": config.Theme.Mode,
		},
	}

	return ValidateStrictAppConfig(rawData)
}
