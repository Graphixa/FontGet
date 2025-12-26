package ui

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fontget/internal/config"

	"gopkg.in/yaml.v3"
)

//go:embed themes
var embeddedThemes embed.FS

// normalizeThemeName normalizes a theme name by replacing spaces with hyphens
// This ensures consistent handling of theme names regardless of how they're specified
// Examples: "awesome theme" -> "awesome-theme", "my_theme" -> "my-theme"
func normalizeThemeName(themeName string) string {
	// Replace spaces and underscores with hyphens
	normalized := strings.ReplaceAll(themeName, " ", "-")
	normalized = strings.ReplaceAll(normalized, "_", "-")
	// Remove any duplicate hyphens
	for strings.Contains(normalized, "--") {
		normalized = strings.ReplaceAll(normalized, "--", "-")
	}
	// Trim hyphens from start and end
	normalized = strings.Trim(normalized, "-")
	return normalized
}

// Theme represents a color theme loaded from a YAML file
type Theme struct {
	ThemeName string     `yaml:"theme_name"` // Display name for the theme (optional, falls back to filename)
	Style     string     `yaml:"style"`      // Theme style: "dark" or "light" (optional, defaults to "dark")
	Colors    ModeColors `yaml:"colors"`     // Color definitions
}

// ModeColors defines colors for a theme
type ModeColors struct {
	// Base colors (required)
	Primary      string `yaml:"primary"`      // Main accent color
	Secondary    string `yaml:"secondary"`    // Secondary accent color
	Components   string `yaml:"components"`   // Interactive elements (buttons, switches)
	Placeholders string `yaml:"placeholders"` // Muted elements (borders, placeholders)
	Base         string `yaml:"base"`         // Dark backgrounds and inverted text

	// Status colors (required)
	Warning string `yaml:"warning"`
	Error   string `yaml:"error"`
	Success string `yaml:"success"`

	// Component overrides (optional)
	Overrides ComponentOverrides `yaml:"overrides"`
}

// ComponentOverrides defines optional component-specific color overrides
type ComponentOverrides struct {
	PageTitle struct {
		Text       string `yaml:"text"`
		Background string `yaml:"background"`
	} `yaml:"page_title"`

	Button struct {
		Foreground string `yaml:"foreground"`
		Background string `yaml:"background"`
	} `yaml:"button"`

	Switch struct {
		Foreground string `yaml:"foreground"`
		Background string `yaml:"background"`
	} `yaml:"switch"`

	Checkbox struct {
		Unchecked string `yaml:"unchecked"`
		Checked   string `yaml:"checked"`
	} `yaml:"checkbox"`

	Card struct {
		TitleText       string `yaml:"title_text"`
		TitleBackground string `yaml:"title_background"`
		Label           string `yaml:"label"`
		Border          string `yaml:"border"`
	} `yaml:"card"`

	CommandKeys struct {
		Text       string `yaml:"text"`
		Background string `yaml:"background"`
	} `yaml:"command_keys"`

	Table struct {
		Header   string `yaml:"header"`
		Row      string `yaml:"row"`
		Selected string `yaml:"selected"`
	} `yaml:"table"`

	Spinner struct {
		Normal string `yaml:"normal"`
		Done   string `yaml:"done"`
	} `yaml:"spinner"`

	ProgressBar struct {
		Start  string `yaml:"start"`
		Finish string `yaml:"finish"`
	} `yaml:"progress_bar"`
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
// Theme names are normalized (spaces/underscores -> hyphens) for consistent file matching
func (tm *ThemeManager) GetThemePath(themeName string) (string, error) {
	themesDir := filepath.Join(tm.configDir, "themes")

	// Normalize theme name for consistent file matching
	normalized := normalizeThemeName(themeName)

	// Try normalized name first (e.g., "broken-theme.yaml")
	themePath := filepath.Join(themesDir, normalized+".yaml")
	if _, err := os.Stat(themePath); err == nil {
		return themePath, nil
	}

	// Fallback: try original name (for backward compatibility with existing files)
	if normalized != themeName {
		themePath = filepath.Join(themesDir, themeName+".yaml")
		if _, err := os.Stat(themePath); err == nil {
			return themePath, nil
		}
	}

	return "", fmt.Errorf("theme file not found: %s.yaml (tried: %s.yaml)", themeName, normalized)
}

// ValidateTheme validates that a theme has all required color keys
func ValidateTheme(theme *Theme) error {
	colors := &theme.Colors

	requiredKeys := []struct {
		name  string
		value string
	}{
		{"primary", colors.Primary},
		{"secondary", colors.Secondary},
		{"components", colors.Components},
		{"placeholders", colors.Placeholders},
		{"base", colors.Base},
		{"warning", colors.Warning},
		{"error", colors.Error},
		{"success", colors.Success},
	}

	var missingKeys []string
	for _, key := range requiredKeys {
		if key.value == "" {
			missingKeys = append(missingKeys, key.name)
		}
	}

	if len(missingKeys) > 0 {
		// Format error message with bulleted list
		var msg strings.Builder
		msg.WriteString("This theme is missing the following required color keys:\n")
		for _, key := range missingKeys {
			msg.WriteString(fmt.Sprintf("- %s\n", key))
		}
		// Remove trailing newline
		errorMsg := strings.TrimRight(msg.String(), "\n")
		return errors.New(errorMsg)
	}

	return nil
}

// LoadTheme loads a theme from a file
// User themes: Loads {themeName}.yaml (e.g., "catppuccin.yaml")
// Special case: "system" theme returns LoadSystemTheme() without file loading
// Theme names are normalized (spaces/underscores -> hyphens) for consistent matching
func (tm *ThemeManager) LoadTheme(themeName string) (*Theme, error) {
	// Special handling for "system" theme - no file needed
	if themeName == "system" {
		return LoadSystemTheme(), nil
	}

	// Normalize theme name for consistent file matching
	normalized := normalizeThemeName(themeName)
	themePath, err := tm.GetThemePath(normalized)
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
	// Use normalized name for validation check
	if normalized != "system" {
		if err := ValidateTheme(&theme); err != nil {
			return nil, err // Return validation error directly (already formatted)
		}
	}

	return &theme, nil
}

// LoadEmbeddedTheme loads a theme from embedded files
// Special case: "system" theme returns LoadSystemTheme() without file loading
// Theme names are normalized (spaces/underscores -> hyphens) for consistent matching
func LoadEmbeddedTheme(themeName string) (*Theme, error) {
	// Special handling for "system" theme - no file needed
	if themeName == "system" {
		return LoadSystemTheme(), nil
	}

	// Normalize theme name for consistent file matching
	normalized := normalizeThemeName(themeName)

	// Embedded themes use format: themes/{themeName}.yaml (relative to internal/themes/)
	// Try normalized name first
	themePath := fmt.Sprintf("themes/%s.yaml", normalized)
	data, err := embeddedThemes.ReadFile(themePath)
	if err != nil {
		// Fallback: try original name (for backward compatibility)
		if normalized != themeName {
			themePath = fmt.Sprintf("themes/%s.yaml", themeName)
			var fallbackErr error
			data, fallbackErr = embeddedThemes.ReadFile(themePath)
			if fallbackErr == nil {
				err = nil // Success with fallback
			}
		}
		if err != nil {
			return nil, fmt.Errorf("embedded theme not found: %s (tried: themes/%s.yaml)", themeName, normalized)
		}
	}

	var theme Theme
	if err := yaml.Unmarshal(data, &theme); err != nil {
		return nil, fmt.Errorf("failed to parse embedded theme: %w", err)
	}

	// Validate theme (system theme is exempt)
	// Use normalized name for validation check
	if normalized != "system" {
		if err := ValidateTheme(&theme); err != nil {
			return nil, err // Return validation error directly (already formatted)
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
	themeName := appConfig.Theme

	// If theme name is empty, use catppuccin theme (default)
	if themeName == "" {
		themeName = "catppuccin"
	}

	// Normalize theme name for consistent matching
	normalized := normalizeThemeName(themeName)

	// Special handling for "system" theme - no file loading needed
	if normalized == "system" {
		globalThemeManager.SetTheme(LoadSystemTheme())
		return nil
	}

	// Try to load user's theme from ~/.fontget/themes/ first
	// LoadTheme will normalize the name internally, but we normalize here for consistency
	userTheme, err := globalThemeManager.LoadTheme(normalized)
	if err != nil {
		// If user theme fails, try embedded theme
		embeddedTheme, embedErr := LoadEmbeddedTheme(normalized)
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
