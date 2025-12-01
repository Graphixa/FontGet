package ui

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"fontget/internal/config"
	"fontget/internal/platform"

	"gopkg.in/yaml.v3"
)

//go:embed themes/catppuccin.yaml
var defaultThemeYAML []byte

// Theme represents a color theme loaded from a YAML file
type Theme struct {
	FontGetTheme struct {
		DarkMode  ModeColors `yaml:"dark_mode"`
		LightMode ModeColors `yaml:"light_mode"`
	} `yaml:"fontget_theme"`
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

// SetMode sets the current mode (dark, light, or auto)
// If "auto" is passed, it will detect the terminal theme
func (tm *ThemeManager) SetMode(mode string) {
	if mode == "auto" {
		mode = DetectTerminalTheme()
	}
	if mode == "light" || mode == "dark" {
		tm.mode = mode
	}
}

// GetMode returns the current mode
func (tm *ThemeManager) GetMode() string {
	return tm.mode
}

// GetColors returns the colors for the current mode
func (tm *ThemeManager) GetColors() *ModeColors {
	if tm.currentTheme == nil {
		return nil
	}
	if tm.mode == "light" {
		return &tm.currentTheme.FontGetTheme.LightMode
	}
	return &tm.currentTheme.FontGetTheme.DarkMode
}

// GetThemePath returns the path to a theme file
func (tm *ThemeManager) GetThemePath(themeName string) (string, error) {
	themesDir := filepath.Join(tm.configDir, "themes")

	// Try .yaml extension first
	themePath := filepath.Join(themesDir, themeName+".yaml")
	if _, err := os.Stat(themePath); err == nil {
		return themePath, nil
	}

	// Try -theme.yaml extension
	themePath = filepath.Join(themesDir, themeName+"-theme.yaml")
	if _, err := os.Stat(themePath); err == nil {
		return themePath, nil
	}

	return "", fmt.Errorf("theme file not found: %s", themeName)
}

// ValidateTheme validates that a theme has all required color keys
func ValidateTheme(theme *Theme, mode string) error {
	var colors *ModeColors
	if mode == "light" {
		colors = &theme.FontGetTheme.LightMode
	} else {
		colors = &theme.FontGetTheme.DarkMode
	}

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
		return fmt.Errorf("theme is missing required color keys in %s mode: %v", mode, missingKeys)
	}

	return nil
}

// LoadTheme loads a theme from a file
func (tm *ThemeManager) LoadTheme(themeName string) (*Theme, error) {
	themePath, err := tm.GetThemePath(themeName)
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

	// Validate theme for both modes
	if err := ValidateTheme(&theme, "dark"); err != nil {
		return nil, fmt.Errorf("theme validation failed: %w", err)
	}
	if err := ValidateTheme(&theme, "light"); err != nil {
		return nil, fmt.Errorf("theme validation failed: %w", err)
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

// LoadDefaultTheme loads the default Catppuccin theme from the embedded YAML file
// This uses the semantic color system defined in the YAML structure
// The theme contains both dark_mode (Mocha) and light_mode (Latte) colors
func LoadDefaultTheme() (*Theme, error) {
	var theme Theme
	if err := yaml.Unmarshal(defaultThemeYAML, &theme); err != nil {
		return nil, fmt.Errorf("failed to parse embedded default theme: %w", err)
	}
	return &theme, nil
}

// GetThemeManager returns the global theme manager instance
var globalThemeManager *ThemeManager

// DetectTerminalTheme detects whether the terminal has a dark or light background
// Uses the platform package for robust cross-platform detection
func DetectTerminalTheme() string {
	// Use platform package with 2 second timeout
	result, err := platform.TerminalThemeFromEnvOrDetect("FONTGET_THEME_MODE", 2*time.Second)
	if err != nil || result.Theme == platform.TerminalThemeUnknown {
		// Default fallback to dark mode
		return "dark"
	}
	if result.Theme == platform.TerminalThemeDark {
		return "dark"
	}
	return "light"
}

// InitThemeManager initializes the global theme manager
// This should be called during application startup
func InitThemeManager() error {
	configDir := config.GetAppConfigDir()

	globalThemeManager = NewThemeManager(configDir)

	// Load default theme from embedded file as fallback
	defaultTheme, err := LoadDefaultTheme()
	if err != nil {
		return fmt.Errorf("failed to load embedded default theme: %w", err)
	}

	// Try to load user's selected theme from config
	appConfig := config.GetUserPreferences()
	themeName := appConfig.Theme.Name
	mode := appConfig.Theme.Mode

	// Handle "auto" mode - detect terminal theme
	if mode == "auto" {
		mode = DetectTerminalTheme()
	}

	// Set mode (defaults to "dark" if invalid)
	if mode == "light" || mode == "dark" {
		globalThemeManager.SetMode(mode)
	} else {
		// Invalid mode, try auto-detection as fallback
		mode = DetectTerminalTheme()
		globalThemeManager.SetMode(mode)
	}

	// If theme name is empty, use embedded default
	if themeName == "" {
		globalThemeManager.SetTheme(defaultTheme)
		return nil
	}

	// Try to load user's theme from ~/.fontget/themes/
	userTheme, err := globalThemeManager.LoadTheme(themeName)
	if err != nil {
		// If user theme fails to load or validate, fallback to embedded default
		// Note: Error is logged but not returned to allow graceful fallback
		globalThemeManager.SetTheme(defaultTheme)
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
