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
	Name        string // "catppuccin"
	IsBuiltIn   bool   // true for embedded themes
	DisplayName string // "Catppuccin"
}

// ThemeOption represents a theme for selection
type ThemeOption struct {
	ThemeName   string // "catppuccin"
	DisplayName string // "Catppuccin"
	IsBuiltIn   bool   // true for embedded themes
	IsSelected  bool   // true if currently active
	FilePath    string // Path to theme file (for loading) - empty for embedded
}

// DiscoverThemes discovers all available themes (both embedded and user themes)
// Returns a list of ThemeInfo grouped by theme name
// Always includes "system" theme as a built-in option
func DiscoverThemes() ([]ThemeInfo, error) {
	themeMap := make(map[string]*ThemeInfo)

	// 0. Add "system" theme as a special built-in (always available)
	themeMap["system"] = &ThemeInfo{
		Name:        "system",
		IsBuiltIn:   true,
		DisplayName: "System",
	}

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
		// If theme already exists (from embedded), keep embedded version
		// User themes with same name override embedded themes
		if _, ok := themeMap[theme.Name]; !ok {
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

		// Parse filename: embedded themes use {theme-name}.yaml
		baseName := filepath.Base(path)
		if !strings.HasSuffix(baseName, ".yaml") {
			return nil // Skip non-YAML files
		}

		// Extract theme name from filename (remove .yaml extension)
		themeName := strings.TrimSuffix(baseName, ".yaml")
		if themeName == "" {
			return nil // Skip empty filenames
		}

		// Get or create theme info
		if _, ok := themeMap[themeName]; !ok {
			theme := &ThemeInfo{
				Name:        themeName,
				IsBuiltIn:   true,
				DisplayName: formatThemeDisplayName(themeName),
			}
			themeMap[themeName] = theme
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

		// Parse filename: {theme-name}.yaml
		baseName := filepath.Base(path)
		if !strings.HasSuffix(baseName, ".yaml") {
			return nil // Skip non-YAML files
		}

		// Extract theme name from filename (remove .yaml extension)
		themeName := strings.TrimSuffix(baseName, ".yaml")
		if themeName == "" {
			return nil // Skip empty filenames
		}

		// Skip "system" theme (it's built-in)
		if themeName == "system" {
			return nil
		}

		// Get or create theme info
		if _, ok := themeMap[themeName]; !ok {
			theme := &ThemeInfo{
				Name:        themeName,
				IsBuiltIn:   false,
				DisplayName: formatThemeDisplayName(themeName),
			}
			themeMap[themeName] = theme
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
// Includes current theme to mark as selected
// Returns options sorted with System first, then alphabetical by display name
func GetThemeOptions(currentThemeName string) ([]ThemeOption, error) {
	themes, err := DiscoverThemes()
	if err != nil {
		return nil, err
	}

	var options []ThemeOption
	var systemOption *ThemeOption

	for _, theme := range themes {
		// Special handling for "system" theme - add it directly
		if theme.Name == "system" {
			systemOption = &ThemeOption{
				ThemeName:   "system",
				DisplayName: "System",
				IsBuiltIn:   true,
				IsSelected:  theme.Name == currentThemeName,
				FilePath:    "", // System theme doesn't use files
			}
			continue
		}

		// Regular themes
		option := ThemeOption{
			ThemeName:   theme.Name,
			DisplayName: theme.DisplayName,
			IsBuiltIn:   theme.IsBuiltIn,
			IsSelected:  theme.Name == currentThemeName,
			FilePath:    "", // Will be determined when loading
		}
		options = append(options, option)
	}

	// Sort options alphabetically by DisplayName (stable sort)
	for i := 0; i < len(options)-1; i++ {
		for j := i + 1; j < len(options); j++ {
			if options[i].DisplayName > options[j].DisplayName {
				options[i], options[j] = options[j], options[i]
			}
		}
	}

	// Prepend System option at the beginning
	if systemOption != nil {
		options = append([]ThemeOption{*systemOption}, options...)
	}

	return options, nil
}
