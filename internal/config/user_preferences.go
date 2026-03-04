package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	"fontget/internal/logging"
	"fontget/internal/platform"

	"gopkg.in/yaml.v3"
)

//go:embed default_config.yaml
var defaultConfigYAML []byte

// AppConfig represents the main application configuration structure
type AppConfig struct {
	ConfigVersion string               `yaml:"version"` // Schema version for migration tracking
	Configuration ConfigurationSection `yaml:"Configuration"`
	Logging       LoggingSection       `yaml:"Logging"`
	Network       NetworkSection       `yaml:"Network"`
	Search        SearchSection        `yaml:"Search"`
	Update        UpdateSection        `yaml:"Update"`
	Theme         ThemeSection         `yaml:"Theme"`
}

// ThemeSection holds theme name and display options (all under the Theme: key in config)
type ThemeSection struct {
	// Name is the theme name (e.g. "catppuccin", "arasaka", "system")
	Name string `yaml:"Name"`
	// Use256ColorSpace downsamples theme hex colors to ANSI 256 for consistent
	// rendering on terminals that don't handle 24-bit well (e.g. Apple Terminal).
	Use256ColorSpace bool `yaml:"Use256ColorSpace"`
}

// ConfigurationSection represents the main configuration settings
type ConfigurationSection struct {
	DefaultEditor string `yaml:"DefaultEditor"`
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

// SearchSection represents search configuration
type SearchSection struct {
	ResultLimit          int  `yaml:"ResultLimit"`          // Maximum number of search results to display (0 = unlimited, default: 0)
	EnablePopularitySort bool `yaml:"EnablePopularitySort"` // Enable/disable popularity scoring in search results
	// Note: When Bubble Tea tables are implemented, this limit may be evaluated differently
	// for interactive browsing vs static output. Leave comments for future evaluation.
}

// UpdateSection represents update configuration
type UpdateSection struct {
	CheckForUpdates     bool   `yaml:"CheckForUpdates"`     // Check for updates on startup
	UpdateCheckInterval int    `yaml:"UpdateCheckInterval"` // Hours between checks
	LastUpdateCheck     string `yaml:"LastUpdateCheck"`     // ISO timestamp
	NextUpdateCheck     string `yaml:"NextUpdateCheck"`     // ISO timestamp - don't prompt until after this time
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

// CurrentConfigVersion is the current config schema version.
//
// When to bump (do not bump for additive-only changes):
//   - Bump when you rename a key, remove a key, or change the structure of a value
//     (e.g. Theme from string to object). Migration will run for users with older configs.
//   - Do not bump when you only add new optional keys with defaults; existing configs
//     stay valid and new keys get defaults from default_config.yaml or DefaultUserPreferences().
//
// Version format "X.Y": increment minor for migrations that need explicit rules
// (see fieldRenameMap, fieldMoveMap, applyExplicitMigrationRules in migrate.go).
//
// When you do bump:
//   - Update this constant only. default_config.yaml does not need editing: the value
//     written for new/initial configs is taken from this constant (see saveDefaultAppConfigWithComments).
//   - Add any new renames/moves to fieldRenameMap/fieldMoveMap and migration rules in migrate.go.
//   - Update struct definitions and validation in validation.go as needed.
const CurrentConfigVersion = "2.0"

// DefaultUserPreferences returns a new default user preferences configuration.
// Loaded from the embedded default_config.yaml (single source of truth).
func DefaultUserPreferences() *AppConfig {
	var cfg AppConfig
	if err := yaml.Unmarshal(defaultConfigYAML, &cfg); err != nil {
		return defaultUserPreferencesFallback()
	}
	cfg.ConfigVersion = CurrentConfigVersion
	return &cfg
}

// defaultUserPreferencesFallback returns a minimal default when embedded default_config.yaml cannot be used.
func defaultUserPreferencesFallback() *AppConfig {
	return &AppConfig{
		ConfigVersion: CurrentConfigVersion,
		Configuration: ConfigurationSection{DefaultEditor: ""},
		Logging:       LoggingSection{LogPath: "$home/.fontget/logs/fontget.log", MaxLogSize: "10MB", MaxLogFiles: 5},
		Network:       NetworkSection{RequestTimeout: "10s", DownloadTimeout: "30s"},
		Search:        SearchSection{ResultLimit: 0, EnablePopularitySort: true},
		Update:        UpdateSection{CheckForUpdates: true, UpdateCheckInterval: 24, LastUpdateCheck: "", NextUpdateCheck: ""},
		Theme:         ThemeSection{Name: "catppuccin", Use256ColorSpace: false},
	}
}

// defaultConfigMap returns the embedded default_config.yaml as a map, for use in schema migration.
func defaultConfigMap() (map[string]interface{}, error) {
	var m map[string]interface{}
	if err := yaml.Unmarshal(defaultConfigYAML, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// mergeConfigValues merges user values from loaded config into default config using reflection
// This automatically preserves all user customizations without manual field copying
func mergeConfigValues(defaultConfig, loadedConfig *AppConfig) {
	// Use reflection to merge all fields from loadedConfig into defaultConfig
	// This preserves user values while keeping defaults for missing fields

	defer func() {
		// Recover from any panic during reflection merge
		if r := recover(); r != nil {
			// If logger is available, log the error
			// Otherwise, just silently continue with defaults
			// This prevents crashes when dealing with malformed or old config files
			_ = r
		}
	}()

	defaultVal := reflect.ValueOf(defaultConfig).Elem()
	loadedVal := reflect.ValueOf(loadedConfig).Elem()

	if !defaultVal.IsValid() || !loadedVal.IsValid() {
		return // Invalid values, skip merge
	}

	// Iterate through all fields in AppConfig
	for i := 0; i < defaultVal.NumField(); i++ {
		defaultField := defaultVal.Field(i)
		loadedField := loadedVal.Field(i)

		if !defaultField.IsValid() || !loadedField.IsValid() {
			continue
		}

		if !defaultField.CanSet() {
			continue
		}

		// Handle different field types
		switch defaultField.Kind() {
		case reflect.String:
			// For strings, only overwrite if loaded value is non-empty
			if loadedField.String() != "" {
				defaultField.SetString(loadedField.String())
			}

		case reflect.Int:
			// For ints, only overwrite if loaded value is non-zero
			if loadedField.Int() != 0 {
				defaultField.SetInt(loadedField.Int())
			}

		case reflect.Bool:
			// For bools, always use loaded value (bool defaults are meaningful)
			defaultField.SetBool(loadedField.Bool())

		case reflect.Struct:
			// For struct fields (sections), merge recursively
			mergeStructFields(defaultField, loadedField)
		}
	}
}

// fieldRenameMap maps old field names to new field names for backward compatibility
// Format: "Section.OldField" -> "Section.NewField"
var fieldRenameMap = map[string]string{
	"Update.AutoCheck": "Update.CheckForUpdates",
	// Add more renames here as needed
}

// fieldMoveMap maps old field locations to new locations for backward compatibility
// Format: "OldSection.OldField" -> "NewSection.NewField"
var fieldMoveMap = map[string]string{
	"Configuration.EnablePopularitySort": "Search.EnablePopularitySort",
	// Add more moves here as needed
}

// mergeStructFields merges fields within a struct section
func mergeStructFields(defaultStruct, loadedStruct reflect.Value) {
	if !defaultStruct.IsValid() || !loadedStruct.IsValid() {
		return
	}

	for i := 0; i < defaultStruct.NumField(); i++ {
		defaultField := defaultStruct.Field(i)
		loadedField := loadedStruct.Field(i)

		if !defaultField.IsValid() || !loadedField.IsValid() {
			continue
		}

		if !defaultField.CanSet() {
			continue
		}

		switch defaultField.Kind() {
		case reflect.String:
			// For strings, only overwrite if loaded value is non-empty
			if loadedField.String() != "" {
				defaultField.SetString(loadedField.String())
			}

		case reflect.Int:
			// For ints, only overwrite if loaded value is non-zero
			if loadedField.Int() != 0 {
				defaultField.SetInt(loadedField.Int())
			}

		case reflect.Bool:
			// For bools, always use loaded value
			defaultField.SetBool(loadedField.Bool())
		}
	}
}

// handleLegacyFieldMapping handles old field names and moves in YAML before unmarshaling.
//
// This function is called automatically by GetUserPreferences() before unmarshaling the config
// into the struct. It:
//   - Parses the YAML into a map
//   - Detects the config version (defaults to "1.0" if not set)
//   - Applies all declarative migration rules (field moves and renames)
//   - Re-marshals the migrated config
//   - Returns the migrated YAML bytes
//
// This ensures that old config files are automatically transformed to the current schema
// before being unmarshaled into the struct, preventing unmarshaling errors from missing
// or misplaced fields.
//
// The migration rules are defined in migrations.go in the migrationRules variable.
func handleLegacyFieldMapping(data []byte) []byte {
	var configMap map[string]interface{}
	if err := yaml.Unmarshal(data, &configMap); err != nil {
		return data // Return original if unmarshaling fails
	}

	// Handle ConfigVersion → version rename
	if version, ok := configMap["ConfigVersion"].(string); ok && version != "" {
		// Migrate to new field name
		configMap["version"] = version
		delete(configMap, "ConfigVersion")
	}

	// Apply field renames (within same section)
	for oldPath, newPath := range fieldRenameMap {
		oldParts := strings.Split(oldPath, ".")
		newParts := strings.Split(newPath, ".")
		if len(oldParts) != 2 || len(newParts) != 2 {
			continue // Invalid format, skip
		}

		oldSection := oldParts[0]
		oldField := oldParts[1]
		newSection := newParts[0]
		newField := newParts[1]

		// Only rename if sections match (field renames stay in same section)
		if oldSection == newSection {
			if section, ok := configMap[oldSection].(map[string]interface{}); ok {
				if value, hasOldField := section[oldField]; hasOldField {
					// Only rename if new field doesn't already exist
					if _, exists := section[newField]; !exists {
						section[newField] = value
						delete(section, oldField)
					}
				}
			}
		}
	}

	// Apply field moves (between sections)
	for oldPath, newPath := range fieldMoveMap {
		oldParts := strings.Split(oldPath, ".")
		newParts := strings.Split(newPath, ".")
		if len(oldParts) != 2 || len(newParts) != 2 {
			continue // Invalid format, skip
		}

		oldSection := oldParts[0]
		oldField := oldParts[1]
		newSection := newParts[0]
		newField := newParts[1]

		// Check if old location exists
		if oldSectionMap, ok := configMap[oldSection].(map[string]interface{}); ok {
			if value, hasField := oldSectionMap[oldField]; hasField {
				// Ensure new section exists
				newSectionMap, ok := configMap[newSection].(map[string]interface{})
				if !ok {
					newSectionMap = make(map[string]interface{})
					configMap[newSection] = newSectionMap
				}

				// Only move if new location doesn't already have the value
				if _, exists := newSectionMap[newField]; !exists {
					newSectionMap[newField] = value
				}

				// Remove from old location
				delete(oldSectionMap, oldField)
			}
		}
	}

	// Theme: convert string (e.g. "arasaka") to object { Name, Use256ColorSpace } for unmarshaling
	if themeVal, ok := configMap["Theme"]; ok {
		if themeStr, ok := themeVal.(string); ok {
			configMap["Theme"] = map[string]interface{}{
				"Name":              themeStr,
				"Use256ColorSpace": false,
			}
		}
	}

	// Re-marshal with migrated fields
	if newData, err := yaml.Marshal(configMap); err == nil {
		return newData
	}

	// Return original if re-marshaling fails
	return data
}

// GetUserPreferences loads user preferences from config file or returns defaults.
// If the file has an older or missing schema version, it is migrated to the current
// schema (matching keys copied over, explicit rules for renames/structure) and the
// updated config is saved. Otherwise legacy field mapping and merge with defaults apply.
func GetUserPreferences() *AppConfig {
	configPath := GetAppConfigPath()
	config := DefaultUserPreferences()

	data, err := os.ReadFile(configPath)
	if err != nil {
		return config
	}

	var rawData map[string]interface{}
	_ = yaml.Unmarshal(data, &rawData) // best-effort: we need raw to check version

	// Schema mismatch: migrate to current schema and save
	if rawData != nil && NeedsSchemaMigration(rawData) {
		migrated, err := MigrateToCurrentSchema(rawData)
		if err == nil {
			_ = SaveUserPreferences(migrated)
			return migrated
		}
		// Migration failed; fall back to legacy path
	}

	// Same-version or unparseable: legacy field mapping then merge with defaults
	data = handleLegacyFieldMapping(data)
	var loadedConfig AppConfig
	if err := yaml.Unmarshal(data, &loadedConfig); err == nil {
		mergeConfigValues(config, &loadedConfig)
		if loadedConfig.ConfigVersion != "" {
			config.ConfigVersion = loadedConfig.ConfigVersion
		} else {
			config.ConfigVersion = CurrentConfigVersion
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

	// If schema version is older or missing, migrate to current schema then load
	if NeedsSchemaMigration(rawData) {
		migrated, err := MigrateToCurrentSchema(rawData)
		if err != nil {
			return nil, fmt.Errorf("config migration failed: %w", err)
		}
		if err := SaveUserPreferences(migrated); err != nil {
			logging.GetLogger().Info("Config migrated but failed to write updated file: %v", err)
		}
		return migrated, nil
	}

	// Same-version path: legacy field mapping then unmarshal
	data = handleLegacyFieldMapping(data)
	var config AppConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse app config file: %w", err)
	}
	if config.ConfigVersion == "" {
		config.ConfigVersion = CurrentConfigVersion
	}
	return &config, nil
}

// GenerateInitialUserPreferences generates a new user preferences file with helpful comments
func GenerateInitialUserPreferences() error {
	configPath := GetAppConfigPath()
	return saveDefaultAppConfigWithComments(configPath)
}

// SaveUserPreferences saves the user preferences to file, preserving comments if the file exists
func SaveUserPreferences(config *AppConfig) error {
	configPath := GetAppConfigPath()

	// Check if file exists - if it does, try to preserve comments
	if _, err := os.Stat(configPath); err == nil {
		// File exists - preserve comments by using yaml.Node
		return saveUserPreferencesWithComments(configPath, config)
	}

	// File doesn't exist - use default template with comments
	return saveDefaultAppConfigWithComments(configPath)
}

// saveUserPreferencesWithComments saves config while preserving existing comments
func saveUserPreferencesWithComments(configPath string, config *AppConfig) error {
	// Read existing file to preserve comments
	data, err := os.ReadFile(configPath)
	if err != nil {
		// If we can't read it, fall back to regular marshal
		return saveUserPreferencesWithoutComments(configPath, config)
	}

	// Parse into yaml.Node to preserve comments
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		// If parsing fails, fall back to regular marshal
		return saveUserPreferencesWithoutComments(configPath, config)
	}

	// Update values in the node tree
	if err := updateNodeWithConfig(&node, config); err != nil {
		// If update fails, fall back to regular marshal
		return saveUserPreferencesWithoutComments(configPath, config)
	}

	// Write to file
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	// Marshal the node tree back to YAML (preserves comments)
	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	defer encoder.Close()

	if err := encoder.Encode(&node); err != nil {
		return fmt.Errorf("failed to encode YAML: %w", err)
	}

	return nil
}

// saveUserPreferencesWithoutComments saves config without preserving comments (fallback)
func saveUserPreferencesWithoutComments(configPath string, config *AppConfig) error {
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

// updateNodeWithConfig updates a yaml.Node tree with values from AppConfig
func updateNodeWithConfig(node *yaml.Node, config *AppConfig) error {
	// Find the root mapping node
	if node.Kind != yaml.DocumentNode || len(node.Content) == 0 {
		return fmt.Errorf("invalid YAML structure")
	}

	root := node.Content[0]
	if root.Kind != yaml.MappingNode {
		return fmt.Errorf("root node is not a mapping")
	}

	// Update each section in the config
	sections := map[string]interface{}{
		"Configuration": config.Configuration,
		"Logging":       config.Logging,
		"Network":       config.Network,
		"Search":        config.Search,
		"Update":        config.Update,
		"Theme":         config.Theme,
	}

	// Iterate through root mapping pairs (key, value, key, value, ...)
	for i := 0; i < len(root.Content); i += 2 {
		if i+1 >= len(root.Content) {
			break
		}

		keyNode := root.Content[i]
		valueNode := root.Content[i+1]

		sectionName := keyNode.Value

		// Theme: ensure value is a mapping (convert scalar to mapping if needed)
		if sectionName == "Theme" {
			if valueNode.Kind != yaml.MappingNode {
				valueNode.Kind = yaml.MappingNode
				valueNode.Content = []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "Name"},
					{Kind: yaml.ScalarNode, Value: config.Theme.Name},
					{Kind: yaml.ScalarNode, Value: "Use256ColorSpace"},
					{Kind: yaml.ScalarNode, Value: strconv.FormatBool(config.Theme.Use256ColorSpace)},
				}
			} else {
				if err := updateSectionNode(valueNode, config.Theme); err != nil {
					return fmt.Errorf("failed to update Theme: %w", err)
				}
			}
			continue
		}

		if sectionData, exists := sections[sectionName]; exists {
			// Update this section's values
			if err := updateSectionNode(valueNode, sectionData); err != nil {
				return fmt.Errorf("failed to update section %s: %w", sectionName, err)
			}
		}
	}

	return nil
}

// updateSectionNode updates a section node with new values
func updateSectionNode(sectionNode *yaml.Node, sectionData interface{}) error {
	if sectionNode.Kind != yaml.MappingNode {
		return fmt.Errorf("section node is not a mapping")
	}

	// Convert section data to map for easier lookup
	sectionMap := make(map[string]interface{})
	v := reflect.ValueOf(sectionData)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		yamlTag := field.Tag.Get("yaml")
		if yamlTag == "" || yamlTag == "-" {
			continue
		}

		// Extract field name from yaml tag (handle ",omitempty" etc.)
		fieldName := strings.Split(yamlTag, ",")[0]
		fieldValue := v.Field(i).Interface()
		sectionMap[fieldName] = fieldValue
	}

	// Update values in the section node
	for i := 0; i < len(sectionNode.Content); i += 2 {
		if i+1 >= len(sectionNode.Content) {
			break
		}

		keyNode := sectionNode.Content[i]
		valueNode := sectionNode.Content[i+1]

		keyName := keyNode.Value
		if newValue, exists := sectionMap[keyName]; exists {
			// Update the value node
			if err := updateValueNode(valueNode, newValue); err != nil {
				return fmt.Errorf("failed to update value for %s: %w", keyName, err)
			}
		}
	}

	return nil
}

// updateValueNode updates a value node with a new value
func updateValueNode(valueNode *yaml.Node, newValue interface{}) error {
	switch v := newValue.(type) {
	case string:
		valueNode.Kind = yaml.ScalarNode
		valueNode.Value = v
		valueNode.Tag = "!!str"
	case bool:
		valueNode.Kind = yaml.ScalarNode
		valueNode.Value = strconv.FormatBool(v)
		valueNode.Tag = "!!bool"
	case int:
		valueNode.Kind = yaml.ScalarNode
		valueNode.Value = strconv.Itoa(v)
		valueNode.Tag = "!!int"
	case int64:
		valueNode.Kind = yaml.ScalarNode
		valueNode.Value = strconv.FormatInt(v, 10)
		valueNode.Tag = "!!int"
	default:
		// For complex types, marshal and update
		data, err := yaml.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}
		var newNode yaml.Node
		if err := yaml.Unmarshal(data, &newNode); err != nil {
			return fmt.Errorf("failed to unmarshal value: %w", err)
		}
		if len(newNode.Content) > 0 {
			*valueNode = *newNode.Content[0]
		}
	}
	return nil
}

// saveDefaultAppConfigWithComments writes the embedded default_config.yaml to the path,
// with the version field set to CurrentConfigVersion. Comments and structure are preserved.
// The version value in the embedded YAML is replaced by CurrentConfigVersion (pattern-based,
// so default_config.yaml does not need editing when the constant is bumped).
func saveDefaultAppConfigWithComments(configPath string) error {
	content := string(defaultConfigYAML)
	// Inject current version: replace the first version: "..." value with CurrentConfigVersion
	if i := strings.Index(content, `version: "`); i >= 0 {
		start := i + len(`version: "`)
		if end := strings.Index(content[start:], `"`); end >= 0 {
			content = content[:start] + CurrentConfigVersion + content[start+end:]
		}
	}
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write app config file: %w", err)
	}
	return nil
}

// ValidateUserPreferences validates the user preferences
func ValidateUserPreferences(config *AppConfig) error {
	// Convert structured config to map for validation
	rawData := map[string]interface{}{
		"version": config.ConfigVersion,
		"Configuration": map[string]interface{}{
			"DefaultEditor": config.Configuration.DefaultEditor,
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
		"Search": map[string]interface{}{
			"ResultLimit":          config.Search.ResultLimit,
			"EnablePopularitySort": config.Search.EnablePopularitySort,
		},
		"Update": map[string]interface{}{
			"CheckForUpdates":     config.Update.CheckForUpdates,
			"UpdateCheckInterval": config.Update.UpdateCheckInterval,
			"LastUpdateCheck":     config.Update.LastUpdateCheck,
		},
		"Theme": map[string]interface{}{
			"Name":              config.Theme.Name,
			"Use256ColorSpace": config.Theme.Use256ColorSpace,
		},
	}

	return ValidateStrictAppConfig(rawData)
}

// MigrateConfigAfterUpdate merges old config with new defaults after an update.
// This preserves all user customizations while applying new defaults.
// Called by the update command after a successful binary update.
func MigrateConfigAfterUpdate() error {
	configPath := GetAppConfigPath()

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// No config file - nothing to migrate, will use defaults
		return nil
	}

	// Load old config (this already handles field renames/moves via handleLegacyFieldMapping)
	oldConfig := GetUserPreferences()
	if oldConfig == nil {
		return fmt.Errorf("failed to load config for migration")
	}

	// Merge: preserve user values, use new defaults for everything else
	// This is already done by GetUserPreferences(), but we want to ensure version is updated
	oldConfig.ConfigVersion = CurrentConfigVersion

	// Save merged config
	if err := SaveUserPreferences(oldConfig); err != nil {
		return fmt.Errorf("failed to save migrated config: %w", err)
	}

	logging.GetLogger().Info("Config migrated after update - user values preserved, new defaults applied")
	return nil
}
