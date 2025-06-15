package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// Directory structure
	sourcesDir     = "sources"
	manifestFile   = "google-fonts.json"
	updateInterval = 24 * time.Hour

	// Font manifest URL in our repository
	fontManifestURL = "https://raw.githubusercontent.com/graphixa/FontGet/main/sources/google-fonts.json"
	googleFontsCSS  = "https://fonts.googleapis.com/css2?family=%s&display=swap"
)

// GoogleFontsResponse represents the response from Google Fonts API
type GoogleFontsResponse struct {
	Kind  string `json:"kind"`
	Items []struct {
		Family       string            `json:"family"`
		Variants     []string          `json:"variants"`
		Subsets      []string          `json:"subsets"`
		Version      string            `json:"version"`
		LastModified string            `json:"lastModified"`
		Files        map[string]string `json:"files"`
		Category     string            `json:"category"`
		Kind         string            `json:"kind"`
	} `json:"items"`
}

// getFontGetDir returns the path to the user's ~/.fontget directory
func getFontGetDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(home, ".fontget"), nil
}

// getSourcesDir returns the path to the user's ~/.fontget/sources directory
func getSourcesDir() (string, error) {
	fontGetDir, err := getFontGetDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(fontGetDir, sourcesDir), nil
}

// ensureSourcesDir ensures the sources directory exists
func ensureSourcesDir() (string, error) {
	// First ensure .fontget directory exists
	fontGetDir, err := getFontGetDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(fontGetDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create .fontget directory: %w", err)
	}

	// Then ensure sources directory exists
	sourcesDir, err := getSourcesDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(sourcesDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create sources directory: %w", err)
	}

	return sourcesDir, nil
}

// ProgressCallback is a function type for reporting progress
type ProgressCallback func(current, total int, message string)

// FontManifest represents the combined manifest of all fonts
type FontManifest struct {
	Version     string                `json:"version"`
	LastUpdated time.Time             `json:"last_updated"`
	Sources     map[string]SourceInfo `json:"sources"`
}

// SourceInfo represents information about a font source
type SourceInfo struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	URL         string              `json:"url"`
	LastUpdated time.Time           `json:"last_updated"`
	Fonts       map[string]FontInfo `json:"fonts"`
}

// FontInfo represents detailed information about a font
type FontInfo struct {
	Name         string            `json:"name"`
	License      string            `json:"license"`
	Variants     []string          `json:"variants"`
	Subsets      []string          `json:"subsets"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	LastModified time.Time         `json:"last_modified"`
	MetadataURL  string            `json:"metadata_url"`
	SourceURL    string            `json:"source_url"`
	Categories   []string          `json:"categories,omitempty"`
	Tags         []string          `json:"tags,omitempty"`
	Popularity   int               `json:"popularity,omitempty"`
	Files        map[string]string `json:"files,omitempty"`
}

// GetFontInfo retrieves information about a font from the repository
func GetFontInfo(fontID string) (*FontInfo, error) {
	manifest, err := GetManifest(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}

	// Look for font in all sources
	for _, source := range manifest.Sources {
		if font, ok := source.Fonts[fontID]; ok {
			return &font, nil
		}
	}

	return nil, fmt.Errorf("font not found: %s", fontID)
}

// GetFontFiles retrieves the font files for a given font family
func GetFontFiles(fontFamily string) (map[string]string, error) {
	manifest, err := GetManifest(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}

	// Look for font in all sources
	for _, source := range manifest.Sources {
		if font, ok := source.Fonts[fontFamily]; ok {
			return font.Files, nil
		}
	}

	return nil, fmt.Errorf("font not found: %s", fontFamily)
}

// Repository represents a font repository
type Repository struct {
	manifest *FontManifest
}

// GetRepository returns a new Repository instance
func GetRepository() (*Repository, error) {
	manifest, err := GetManifest(nil, nil)
	if err != nil {
		return nil, err
	}
	return &Repository{manifest: manifest}, nil
}

// GetManifest returns the current manifest
func (r *Repository) GetManifest() (*FontManifest, error) {
	return r.manifest, nil
}

// SearchFonts searches for fonts matching the query
func (r *Repository) SearchFonts(query string, category string) ([]SearchResult, error) {
	// Use the better search implementation from search.go
	results, err := SearchFonts(query, false)
	if err != nil {
		return nil, err
	}

	// Filter by category if specified
	if category != "" {
		var filteredResults []SearchResult
		for _, result := range results {
			found := false
			for _, cat := range result.Categories {
				if strings.EqualFold(cat, category) {
					found = true
					break
				}
			}
			if found {
				filteredResults = append(filteredResults, result)
			}
		}
		results = filteredResults
	}

	return results, nil
}
