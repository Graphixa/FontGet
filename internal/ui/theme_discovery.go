package ui

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"fontget/internal/config"

	"gopkg.in/yaml.v3"
)

// ThemeInfo represents information about an available theme
type ThemeInfo struct {
	Name        string // "catppuccin"
	IsBuiltIn   bool   // true for embedded themes
	DisplayName string // "Catppuccin"
	Style       string // "dark" or "light" (defaults to "dark" if not specified)
}

// ThemeOption represents a theme for selection
type ThemeOption struct {
	ThemeName   string // "catppuccin"
	DisplayName string // "Catppuccin"
	IsBuiltIn   bool   // true for embedded themes
	IsSelected  bool   // true if currently active
	FilePath    string // Path to theme file (for loading) - empty for embedded
	Style       string // "dark" or "light" (defaults to "dark" if not specified)
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
		DisplayName: "No Theme",
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
		rawThemeName := strings.TrimSuffix(baseName, ".yaml")
		if rawThemeName == "" {
			return nil // Skip empty filenames
		}

		// Normalize theme name (spaces/underscores -> hyphens) for consistent matching
		themeName := normalizeThemeName(rawThemeName)

		// Try to load the theme to get theme_name and style from YAML
		displayName := formatThemeDisplayName(themeName) // Fallback to filename-based
		style := "dark"                                  // Default to dark
		data, err := embeddedThemes.ReadFile(path)
		if err == nil {
			// Parse theme_name and style fields
			var theme Theme
			if err := yaml.Unmarshal(data, &theme); err == nil {
				if theme.ThemeName != "" {
					displayName = theme.ThemeName
				}
				if theme.Style != "" {
					style = theme.Style
				}
			}
		}

		// Get or create theme info
		if _, ok := themeMap[themeName]; !ok {
			theme := &ThemeInfo{
				Name:        themeName,
				IsBuiltIn:   true,
				DisplayName: displayName,
				Style:       style,
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
		rawThemeName := strings.TrimSuffix(baseName, ".yaml")
		if rawThemeName == "" {
			return nil // Skip empty filenames
		}

		// Normalize theme name (spaces/underscores -> hyphens) for consistent matching
		themeName := normalizeThemeName(rawThemeName)

		// Skip "system" theme (it's built-in)
		if themeName == "system" {
			return nil
		}

		// Try to load the theme to get theme_name and style from YAML
		displayName := formatThemeDisplayName(themeName) // Fallback to filename-based
		style := "dark"                                  // Default to dark
		data, err := os.ReadFile(path)
		if err == nil {
			// Parse theme_name and style fields
			var theme Theme
			if err := yaml.Unmarshal(data, &theme); err == nil {
				if theme.ThemeName != "" {
					displayName = theme.ThemeName
				}
				if theme.Style != "" {
					style = theme.Style
				}
			}
		}

		// Get or create theme info
		if _, ok := themeMap[themeName]; !ok {
			theme := &ThemeInfo{
				Name:        themeName,
				IsBuiltIn:   false,
				DisplayName: displayName,
				Style:       style,
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

	// Normalize current theme name for consistent matching
	normalizedCurrent := normalizeThemeName(currentThemeName)

	var options []ThemeOption
	var systemOption *ThemeOption

	for _, theme := range themes {
		// Special handling for "system" theme - add it directly
		if theme.Name == "system" {
			systemOption = &ThemeOption{
				ThemeName:   "system",
				DisplayName: "No Theme",
				IsBuiltIn:   true,
				IsSelected:  normalizedCurrent == "system",
				FilePath:    "", // System theme doesn't use files
			}
			continue
		}

		// Regular themes - compare normalized names
		style := theme.Style
		if style == "" {
			style = "dark" // Default to dark if not specified
		}
		option := ThemeOption{
			ThemeName:   theme.Name,
			DisplayName: theme.DisplayName,
			IsBuiltIn:   theme.IsBuiltIn,
			IsSelected:  theme.Name == normalizedCurrent,
			FilePath:    "", // Will be determined when loading
			Style:       style,
		}
		options = append(options, option)
	}

	// Sort options: first by style (dark before light), then alphabetically by DisplayName
	for i := 0; i < len(options)-1; i++ {
		for j := i + 1; j < len(options); j++ {
			// Compare by style first
			styleI := options[i].Style
			styleJ := options[j].Style
			if styleI == "" {
				styleI = "dark"
			}
			if styleJ == "" {
				styleJ = "dark"
			}

			// Dark themes come before light themes
			if styleI != styleJ {
				if styleI == "light" && styleJ == "dark" {
					options[i], options[j] = options[j], options[i]
				}
				continue
			}

			// Within same style, sort alphabetically
			if options[i].DisplayName > options[j].DisplayName {
				options[i], options[j] = options[j], options[i]
			}
		}
	}

	// Prepend System option at the beginning
	if systemOption != nil {
		systemOption.Style = "" // System theme has no style
		options = append([]ThemeOption{*systemOption}, options...)
	}

	return options, nil
}
