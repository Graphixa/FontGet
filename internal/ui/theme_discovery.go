package ui

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"fontget/internal/config"
)

// ThemeInfo represents information about an available theme
type ThemeInfo struct {
	Name        string   // "catppuccin"
	Modes       []string // ["dark", "light"]
	IsBuiltIn   bool     // true for embedded themes
	DisplayName string   // "Catppuccin"
}

// ThemeOption represents a specific theme+mode combination for selection
type ThemeOption struct {
	ThemeName   string // "catppuccin"
	Mode        string // "dark" or "light"
	DisplayName string // "Catppuccin Dark"
	IsBuiltIn   bool   // true for embedded themes
	IsSelected  bool   // true if currently active
	FilePath    string // Path to theme file (for loading) - empty for embedded
}

// DiscoverThemes discovers all available themes (both embedded and user themes)
// Returns a list of ThemeInfo grouped by theme name
func DiscoverThemes() ([]ThemeInfo, error) {
	themeMap := make(map[string]*ThemeInfo)

	// 1. Scan embedded themes
	embeddedThemes, err := discoverEmbeddedThemes()
	if err != nil {
		return nil, fmt.Errorf("failed to discover embedded themes: %w", err)
	}
	for _, theme := range embeddedThemes {
		themeMap[theme.Name] = theme
	}

	// 2. Scan user themes directory
	userThemes, err := discoverUserThemes()
	if err != nil {
		// Log error but don't fail - user themes are optional
		// We'll continue with embedded themes only
	}
	for _, theme := range userThemes {
		// If theme already exists (from embedded), merge modes
		if existing, ok := themeMap[theme.Name]; ok {
			// Merge modes, avoiding duplicates
			existing.Modes = mergeModes(existing.Modes, theme.Modes)
		} else {
			themeMap[theme.Name] = theme
		}
	}

	// 3. Convert map to slice
	themes := make([]ThemeInfo, 0, len(themeMap))
	for _, theme := range themeMap {
		themes = append(themes, *theme)
	}

	return themes, nil
}

// discoverEmbeddedThemes scans embedded theme files
func discoverEmbeddedThemes() ([]*ThemeInfo, error) {
	themeMap := make(map[string]*ThemeInfo)

	// Walk embedded themes directory
	err := fs.WalkDir(embeddedThemes, "themes", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Parse filename: {theme-name}-{mode}.yaml
		baseName := filepath.Base(path)
		if !strings.HasSuffix(baseName, ".yaml") {
			return nil // Skip non-YAML files
		}

		themeName, mode, err := parseThemeFileName(baseName)
		if err != nil {
			return nil // Skip invalid filenames
		}

		// Get or create theme info
		theme, ok := themeMap[themeName]
		if !ok {
			theme = &ThemeInfo{
				Name:        themeName,
				Modes:       []string{},
				IsBuiltIn:   true,
				DisplayName: formatThemeDisplayName(themeName),
			}
			themeMap[themeName] = theme
		}

		// Add mode if not already present
		if !contains(theme.Modes, mode) {
			theme.Modes = append(theme.Modes, mode)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Convert map to slice
	themes := make([]*ThemeInfo, 0, len(themeMap))
	for _, theme := range themeMap {
		themes = append(themes, theme)
	}

	return themes, nil
}

// discoverUserThemes scans user themes directory (~/.fontget/themes/)
func discoverUserThemes() ([]*ThemeInfo, error) {
	configDir := config.GetAppConfigDir()
	themesDir := filepath.Join(configDir, "themes")

	// Check if themes directory exists
	if _, err := os.Stat(themesDir); os.IsNotExist(err) {
		return []*ThemeInfo{}, nil // No themes directory, return empty
	}

	themeMap := make(map[string]*ThemeInfo)

	// Walk user themes directory
	err := filepath.Walk(themesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Parse filename: {theme-name}-{mode}.yaml
		baseName := filepath.Base(path)
		if !strings.HasSuffix(baseName, ".yaml") {
			return nil // Skip non-YAML files
		}

		// Skip old format files (no -{mode} suffix)
		if !strings.Contains(baseName, "-dark.yaml") && !strings.Contains(baseName, "-light.yaml") {
			return nil // Skip old format files
		}

		themeName, mode, err := parseThemeFileName(baseName)
		if err != nil {
			return nil // Skip invalid filenames
		}

		// Get or create theme info
		theme, ok := themeMap[themeName]
		if !ok {
			theme = &ThemeInfo{
				Name:        themeName,
				Modes:       []string{},
				IsBuiltIn:   false,
				DisplayName: formatThemeDisplayName(themeName),
			}
			themeMap[themeName] = theme
		}

		// Add mode if not already present
		if !contains(theme.Modes, mode) {
			theme.Modes = append(theme.Modes, mode)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Convert map to slice
	themes := make([]*ThemeInfo, 0, len(themeMap))
	for _, theme := range themeMap {
		themes = append(themes, theme)
	}

	return themes, nil
}

// parseThemeFileName parses a theme filename to extract theme name and mode
// Format: {theme-name}-{mode}.yaml
// Examples: "catppuccin-dark.yaml" -> ("catppuccin", "dark", nil)
//
//	"gruvbox-light.yaml" -> ("gruvbox", "light", nil)
func parseThemeFileName(filename string) (themeName string, mode string, err error) {
	// Remove .yaml extension
	base := strings.TrimSuffix(filename, ".yaml")

	// Check for -dark or -light suffix
	if strings.HasSuffix(base, "-dark") {
		themeName = strings.TrimSuffix(base, "-dark")
		mode = "dark"
		return themeName, mode, nil
	}

	if strings.HasSuffix(base, "-light") {
		themeName = strings.TrimSuffix(base, "-light")
		mode = "light"
		return themeName, mode, nil
	}

	// Invalid format
	return "", "", fmt.Errorf("invalid theme filename format: %s (expected {name}-{dark|light}.yaml)", filename)
}

// formatThemeDisplayName formats a theme name for display
// Examples: "catppuccin" -> "Catppuccin", "my-custom-theme" -> "My Custom Theme"
func formatThemeDisplayName(themeName string) string {
	// Replace hyphens and underscores with spaces
	display := strings.ReplaceAll(themeName, "-", " ")
	display = strings.ReplaceAll(display, "_", " ")

	// Title case
	words := strings.Fields(display)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, " ")
}

// GetThemeOptions converts ThemeInfo list to ThemeOption list for TUI
// Includes current theme/mode to mark as selected
func GetThemeOptions(currentThemeName string, currentMode string) ([]ThemeOption, error) {
	themes, err := DiscoverThemes()
	if err != nil {
		return nil, err
	}

	var options []ThemeOption

	for _, theme := range themes {
		for _, mode := range theme.Modes {
			// Format mode for display (capitalize first letter)
			modeDisplay := mode
			if len(mode) > 0 {
				modeDisplay = strings.ToUpper(mode[:1]) + strings.ToLower(mode[1:])
			}

			option := ThemeOption{
				ThemeName:   theme.Name,
				Mode:        mode,
				DisplayName: fmt.Sprintf("%s %s", theme.DisplayName, modeDisplay),
				IsBuiltIn:   theme.IsBuiltIn,
				IsSelected:  theme.Name == currentThemeName && mode == currentMode,
				FilePath:    "", // Will be determined when loading
			}
			options = append(options, option)
		}
	}

	return options, nil
}

// Helper functions

// mergeModes merges two mode slices, avoiding duplicates
func mergeModes(modes1, modes2 []string) []string {
	modeMap := make(map[string]bool)
	for _, mode := range modes1 {
		modeMap[mode] = true
	}
	for _, mode := range modes2 {
		modeMap[mode] = true
	}

	result := make([]string, 0, len(modeMap))
	// Preserve order: dark first, then light
	if modeMap["dark"] {
		result = append(result, "dark")
	}
	if modeMap["light"] {
		result = append(result, "light")
	}

	return result
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}
