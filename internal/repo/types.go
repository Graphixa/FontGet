package repo

import (
	"os"
	"path/filepath"
	"time"
)

// FontGet-Sources Schema Structures

// SourceData represents the complete FontGet-Sources file structure
type SourceData struct {
	SourceInfo SourceInfo          `json:"source_info"`
	Fonts      map[string]FontData `json:"fonts"`
}

// SourceInfo represents metadata about a font source
type SourceInfo struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	URL         string              `json:"url"`
	APIEndpoint string              `json:"api_endpoint,omitempty"`
	Version     string              `json:"version"`
	LastUpdated time.Time           `json:"last_updated"`
	TotalFonts  int                 `json:"total_fonts"`
	Fonts       map[string]FontInfo `json:"fonts,omitempty"`
}

// FontVariant represents a font variant with detailed information
type FontVariant struct {
	Name    string            `json:"name"`
	Weight  int               `json:"weight"`
	Style   string            `json:"style"`
	Subsets []string          `json:"subsets,omitempty"`
	Files   map[string]string `json:"files"`
}

// FontData represents detailed information about a font from FontGet-Sources
type FontData struct {
	Name          string        `json:"name"`
	Family        string        `json:"family"`
	License       string        `json:"license"`
	LicenseURL    string        `json:"license_url,omitempty"`
	Designer      string        `json:"designer,omitempty"`
	Foundry       string        `json:"foundry,omitempty"`
	Version       string        `json:"version,omitempty"`
	Description   string        `json:"description,omitempty"`
	Categories    []string      `json:"categories,omitempty"`
	Tags          []string      `json:"tags,omitempty"`
	Popularity    int           `json:"popularity,omitempty"`
	LastModified  string        `json:"last_modified,omitempty"`
	MetadataURL   string        `json:"metadata_url,omitempty"`
	SourceURL     string        `json:"source_url,omitempty"`
	Variants      []FontVariant `json:"variants"`
	UnicodeRanges []string      `json:"unicode_ranges,omitempty"`
	Languages     []string      `json:"languages,omitempty"`
	SampleText    string        `json:"sample_text,omitempty"`
}

// FontManifest represents the combined manifest of all fonts
type FontManifest struct {
	Version     string                `json:"version"`
	LastUpdated time.Time             `json:"last_updated"`
	Sources     map[string]SourceInfo `json:"sources"`
}

// Legacy structures (to be deprecated)
// Manifest represents the font manifest
type Manifest struct {
	Sources map[string]Source `json:"sources"`
}

// Source represents a font source
type Source struct {
	Name        string                   `json:"name"`
	URL         string                   `json:"url"`
	Fonts       map[string]BasicFontInfo `json:"fonts"`
	LastUpdated string                   `json:"last_updated"`
}

// FontInfo represents detailed information about a font (legacy compatibility)
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

// BasicFontInfo represents basic font file information
type BasicFontInfo struct {
	Name  string            `json:"name"`
	Files map[string]string `json:"files"`
}

// Cache represents the font cache
type Cache struct {
	Dir string
}

// NewCache creates a new cache in the user's home directory
func NewCache() (*Cache, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	cacheDir := filepath.Join(home, ".fontget", "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, err
	}
	return &Cache{Dir: cacheDir}, nil
}
