package ui

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"fontget/internal/config"

	"gopkg.in/yaml.v3"
)

//go:embed themes/*
var embeddedThemes embed.FS

// Theme represents a color theme loaded from a YAML file
type Theme struct {
	ThemeName string     `yaml:"theme_name"` // Display name for the theme (optional, falls back to filename)
	Colors    ModeColors `yaml:"colors"`     // Color definitions
}

// ModeColors defines colors for a theme
type ModeColors struct {
	Accent              string `yaml:"accent"`
	Accent2             string `yaml:"accent2"`
	Warning             string `yaml:"warning"`
	Error               string `yaml:"error"`
	Success             string `yaml:"success"`
	PageTitle           string `yaml:"page_title"`
	PageSubtitle        string `yaml:"page_subtitle"`
	CheckboxChecked     string `yaml:"checkbox_checked"`   // Checked checkbox color (falls back to accent2 if not set)
	CheckboxUnchecked   string `yaml:"checkbox_unchecked"` // Unchecked checkbox color (empty = no color/terminal default)
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
}

// NewThemeManager creates a new theme manager
func NewThemeManager(configDir string) *ThemeManager {
	return &ThemeManager{
		configDir: configDir,
	}
}

// GetColors returns the colors for the current theme
func (tm *ThemeManager) GetColors() *ModeColors {
	if tm.currentTheme == nil {
		return nil
	}
	return &tm.currentTheme.Colors
}

// GetThemePath returns the path to a theme file
// User themes: {themeName}.yaml (e.g., "catppuccin.yaml")
func (tm *ThemeManager) GetThemePath(themeName string) (string, error) {
	themesDir := filepath.Join(tm.configDir, "themes")

	// Format: {themeName}.yaml
	themePath := filepath.Join(themesDir, themeName+".yaml")
	if _, err := os.Stat(themePath); err == nil {
		return themePath, nil
	}

	return "", fmt.Errorf("theme file not found: %s.yaml", themeName)
}

// ValidateTheme validates that a theme has all required color keys
func ValidateTheme(theme *Theme) error {
	colors := &theme.Colors

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
// User themes: Loads {themeName}.yaml (e.g., "catppuccin.yaml")
// Special case: "system" theme returns LoadSystemTheme() without file loading
func (tm *ThemeManager) LoadTheme(themeName string) (*Theme, error) {
	// Special handling for "system" theme - no file needed
	if themeName == "system" {
		return LoadSystemTheme(), nil
	}

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

	// Validate theme
	// System theme is exempt from validation
	if themeName != "system" {
		if err := ValidateTheme(&theme); err != nil {
			return nil, fmt.Errorf("theme validation failed: %w", err)
		}
	}

	return &theme, nil
}

// LoadEmbeddedTheme loads a theme from embedded files
// Special case: "system" theme returns LoadSystemTheme() without file loading
func LoadEmbeddedTheme(themeName string) (*Theme, error) {
	// Special handling for "system" theme - no file needed
	if themeName == "system" {
		return LoadSystemTheme(), nil
	}

	// Embedded themes use format: themes/{themeName}.yaml
	themePath := fmt.Sprintf("themes/%s.yaml", themeName)
	data, err := embeddedThemes.ReadFile(themePath)
	if err != nil {
		return nil, fmt.Errorf("embedded theme not found: %s", themePath)
	}

	var theme Theme
	if err := yaml.Unmarshal(data, &theme); err != nil {
		return nil, fmt.Errorf("failed to parse embedded theme: %w", err)
	}

	// Validate theme (system theme is exempt)
	if themeName != "system" {
		if err := ValidateTheme(&theme); err != nil {
			return nil, fmt.Errorf("embedded theme validation failed: %w", err)
		}
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

// LoadSystemTheme creates a "system" theme that uses terminal default colors
// This theme has empty color strings, which InitStyles() will interpret as "no color"
func LoadSystemTheme() *Theme {
	return &Theme{
		Colors: ModeColors{
			// All colors are empty - InitStyles() will use lipgloss.NoColor{} for these
		},
	}
}

// LoadDefaultTheme loads the default Catppuccin theme from embedded files
// Uses catppuccin.yaml as default fallback
func LoadDefaultTheme() (*Theme, error) {
	return LoadEmbeddedTheme("catppuccin")
}

// GetThemeManager returns the global theme manager instance
var globalThemeManager *ThemeManager

// InitThemeManager initializes the global theme manager
// This should be called during application startup
func InitThemeManager() error {
	configDir := config.GetAppConfigDir()

	globalThemeManager = NewThemeManager(configDir)

	// Get theme from config
	appConfig := config.GetUserPreferences()
	themeName := appConfig.Theme.Name

	// If theme name is empty, use embedded default (catppuccin)
	if themeName == "" {
		themeName = "catppuccin"
	}

	// Special handling for "system" theme - no file loading needed
	if themeName == "system" {
		globalThemeManager.SetTheme(LoadSystemTheme())
		return nil
	}

	// Try to load user's theme from ~/.fontget/themes/ first
	userTheme, err := globalThemeManager.LoadTheme(themeName)
	if err != nil {
		// If user theme fails, try embedded theme
		embeddedTheme, embedErr := LoadEmbeddedTheme(themeName)
		if embedErr != nil {
			// If embedded theme also fails, fallback to catppuccin
			defaultTheme, defaultErr := LoadDefaultTheme()
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

// GetCurrentColors returns the colors for the current theme
func GetCurrentColors() *ModeColors {
	return GetThemeManager().GetColors()
}
