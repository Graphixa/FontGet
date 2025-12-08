package ui

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fontget/internal/config"

	"gopkg.in/yaml.v3"
)

//go:embed themes/catppuccin-dark.yaml themes/catppuccin-light.yaml themes/gruvbox-dark.yaml themes/gruvbox-light.yaml
var embeddedThemes embed.FS

// Theme represents a color theme loaded from a YAML file
// New format: Single-mode theme (no dark_mode/light_mode wrapper)
type Theme struct {
	FontGetTheme ModeColors `yaml:"fontget_theme"`
}

// ModeColors defines colors for a specific mode (dark/light)
type ModeColors struct {
	Accent              string `yaml:"accent"`
	Accent2             string `yaml:"accent2"`
	Warning             string `yaml:"warning"`
	Error               string `yaml:"error"`
	Success             string `yaml:"success"`
	PageTitle           string `yaml:"page_title"`
	PageSubtitle        string `yaml:"page_subtitle"`
	GreyLight           string `yaml:"grey_light"`
	GreyMid             string `yaml:"grey_mid"`
	GreyDark            string `yaml:"grey_dark"`
	ProgressBarGradient struct {
		ColorStart string `yaml:"color_start"`
		ColorEnd   string `yaml:"color_end"`
	} `yaml:"progress_bar_gradient"`
}

// ThemeManager handles theme loading and management
type ThemeManager struct {
	configDir    string
	currentTheme *Theme
	mode         string // "dark" or "light"
}

// NewThemeManager creates a new theme manager
func NewThemeManager(configDir string) *ThemeManager {
	return &ThemeManager{
		configDir: configDir,
		mode:      "dark", // Default to dark mode
	}
}

// SetMode sets the current mode (dark or light)
// Only "dark" and "light" are supported - no auto-detection
func (tm *ThemeManager) SetMode(mode string) {
	if mode == "light" || mode == "dark" {
		tm.mode = mode
	} else {
		// Default to dark if invalid mode
		tm.mode = "dark"
	}
}

// GetMode returns the current mode
func (tm *ThemeManager) GetMode() string {
	return tm.mode
}

// GetColors returns the colors for the current theme
// Note: Theme now contains only one mode (loaded based on theme+mode combination)
func (tm *ThemeManager) GetColors() *ModeColors {
	if tm.currentTheme == nil {
		return nil
	}
	return &tm.currentTheme.FontGetTheme
}

// GetThemePath returns the path to a theme file
// New format: {themeName}-{mode}.yaml (e.g., "catppuccin-dark.yaml")
func (tm *ThemeManager) GetThemePath(themeName string, mode string) (string, error) {
	themesDir := filepath.Join(tm.configDir, "themes")

	// New format: {themeName}-{mode}.yaml
	themePath := filepath.Join(themesDir, fmt.Sprintf("%s-%s.yaml", themeName, mode))
	if _, err := os.Stat(themePath); err == nil {
		return themePath, nil
	}

	// Fallback: Try old format {themeName}.yaml (for backward compatibility during migration)
	themePath = filepath.Join(themesDir, themeName+".yaml")
	if _, err := os.Stat(themePath); err == nil {
		return themePath, nil
	}

	return "", fmt.Errorf("theme file not found: %s-%s.yaml", themeName, mode)
}

// ValidateTheme validates that a theme has all required color keys
// New format: Theme contains only one mode, so validation is simpler
func ValidateTheme(theme *Theme) error {
	colors := &theme.FontGetTheme

	requiredKeys := []struct {
		name  string
		value string
	}{
		{"accent", colors.Accent},
		{"accent2", colors.Accent2},
		{"warning", colors.Warning},
		{"error", colors.Error},
		{"success", colors.Success},
		{"page_title", colors.PageTitle},
		{"page_subtitle", colors.PageSubtitle},
		{"grey_light", colors.GreyLight},
		{"grey_mid", colors.GreyMid},
		{"grey_dark", colors.GreyDark},
		{"progress_bar_gradient.color_start", colors.ProgressBarGradient.ColorStart},
		{"progress_bar_gradient.color_end", colors.ProgressBarGradient.ColorEnd},
	}

	var missingKeys []string
	for _, key := range requiredKeys {
		if key.value == "" {
			missingKeys = append(missingKeys, key.name)
		}
	}

	if len(missingKeys) > 0 {
		return fmt.Errorf("theme is missing required color keys: %v", missingKeys)
	}

	return nil
}

// LoadTheme loads a theme from a file
// New format: Loads {themeName}-{mode}.yaml (e.g., "catppuccin-dark.yaml")
func (tm *ThemeManager) LoadTheme(themeName string, mode string) (*Theme, error) {
	themePath, err := tm.GetThemePath(themeName, mode)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(themePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read theme file: %w", err)
	}

	var theme Theme
	if err := yaml.Unmarshal(data, &theme); err != nil {
		return nil, fmt.Errorf("failed to parse theme file: %w", err)
	}

	// Validate theme (single-mode validation)
	if err := ValidateTheme(&theme); err != nil {
		return nil, fmt.Errorf("theme validation failed: %w", err)
	}

	return &theme, nil
}

// LoadEmbeddedTheme loads a theme from embedded files
func LoadEmbeddedTheme(themeName string, mode string) (*Theme, error) {
	themePath := fmt.Sprintf("themes/%s-%s.yaml", themeName, mode)
	data, err := embeddedThemes.ReadFile(themePath)
	if err != nil {
		return nil, fmt.Errorf("embedded theme not found: %s", themePath)
	}

	var theme Theme
	if err := yaml.Unmarshal(data, &theme); err != nil {
		return nil, fmt.Errorf("failed to parse embedded theme: %w", err)
	}

	// Validate theme
	if err := ValidateTheme(&theme); err != nil {
		return nil, fmt.Errorf("embedded theme validation failed: %w", err)
	}

	return &theme, nil
}

// GetCurrentTheme returns the currently active theme
func (tm *ThemeManager) GetCurrentTheme() *Theme {
	return tm.currentTheme
}

// SetTheme sets the active theme
func (tm *ThemeManager) SetTheme(theme *Theme) {
	tm.currentTheme = theme
}

// LoadDefaultTheme loads the default Catppuccin theme from embedded files
// Uses catppuccin-dark.yaml as default fallback
func LoadDefaultTheme(mode string) (*Theme, error) {
	// Default to dark if invalid mode
	if mode != "light" && mode != "dark" {
		mode = "dark"
	}
	return LoadEmbeddedTheme("catppuccin", mode)
}

// GetThemeManager returns the global theme manager instance
var globalThemeManager *ThemeManager

// InitThemeManager initializes the global theme manager
// This should be called during application startup
func InitThemeManager() error {
	configDir := config.GetAppConfigDir()

	globalThemeManager = NewThemeManager(configDir)

	// Get theme and mode from config
	appConfig := config.GetUserPreferences()
	themeName := appConfig.Theme.Name
	mode := appConfig.Theme.Mode

	// Check environment variable override (FONTGET_THEME_MODE)
	// Environment variable takes precedence over config file
	if envMode := os.Getenv("FONTGET_THEME_MODE"); envMode != "" {
		envMode = strings.ToLower(envMode)
		if envMode == "dark" || envMode == "light" {
			mode = envMode
		}
		// Ignore invalid env var values, use config instead
	}

	// Validate mode, default to "dark" if invalid
	if mode != "light" && mode != "dark" {
		mode = "dark" // Default fallback
	}

	globalThemeManager.SetMode(mode)

	// If theme name is empty, use embedded default (catppuccin)
	if themeName == "" {
		themeName = "catppuccin"
	}

	// Try to load user's theme from ~/.fontget/themes/ first
	userTheme, err := globalThemeManager.LoadTheme(themeName, mode)
	if err != nil {
		// If user theme fails, try embedded theme
		embeddedTheme, embedErr := LoadEmbeddedTheme(themeName, mode)
		if embedErr != nil {
			// If embedded theme also fails, fallback to catppuccin-dark
			defaultTheme, defaultErr := LoadDefaultTheme("dark")
			if defaultErr != nil {
				return fmt.Errorf("failed to load any theme: user=%v, embedded=%v, default=%v", err, embedErr, defaultErr)
			}
			globalThemeManager.SetTheme(defaultTheme)
			return nil
		}
		globalThemeManager.SetTheme(embeddedTheme)
		return nil
	}

	// User theme loaded successfully
	globalThemeManager.SetTheme(userTheme)
	return nil
}

// GetThemeManager returns the global theme manager
func GetThemeManager() *ThemeManager {
	if globalThemeManager == nil {
		// Initialize with default if not already initialized
		InitThemeManager()
	}
	return globalThemeManager
}

// GetCurrentTheme returns the currently active theme
func GetCurrentTheme() *Theme {
	return GetThemeManager().GetCurrentTheme()
}

// GetCurrentColors returns the colors for the current mode
func GetCurrentColors() *ModeColors {
	return GetThemeManager().GetColors()
}
