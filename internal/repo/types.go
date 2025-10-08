package repo

import (
	"os"
	"path/filepath"
	"time"
)

// FontGet-Sources Schema Structures

// SourceData represents the complete FontGet-Sources file structure
type SourceData struct {
	SourceInfo SourceInfo      `json:"source_info"`
	Fonts      map[string]Font `json:"fonts"`
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

// Font represents a consolidated font structure that replaces FontData, FontInfo, Font, and FontFileInfo
type Font struct {
	// Basic identification
	Name   string `json:"name"`
	Family string `json:"family"`
	ID     string `json:"id,omitempty"`
	Source string `json:"source,omitempty"`

	// Licensing and attribution
	License    string `json:"license"`
	LicenseURL string `json:"license_url,omitempty"`
	Designer   string `json:"designer,omitempty"`
	Foundry    string `json:"foundry,omitempty"`

	// Version and metadata
	Version      string `json:"version,omitempty"`
	Description  string `json:"description,omitempty"`
	LastModified string `json:"last_modified,omitempty"`

	// Classification
	Categories []string `json:"categories,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Popularity int      `json:"popularity,omitempty"`

	// Font variants and files
	Variants []FontVariant     `json:"variants"`
	Files    map[string]string `json:"files,omitempty"`
	Subsets  []string          `json:"subsets,omitempty"`

	// URLs
	MetadataURL string `json:"metadata_url,omitempty"`
	SourceURL   string `json:"source_url,omitempty"`

	// Typography
	UnicodeRanges []string `json:"unicode_ranges,omitempty"`
	Languages     []string `json:"languages,omitempty"`
	SampleText    string   `json:"sample_text,omitempty"`

	// Installation-specific fields
	Path        string `json:"path,omitempty"`
	SHA         string `json:"sha,omitempty"`
	DownloadURL string `json:"download_url,omitempty"`
}

// Conversion functions to help migrate from old structures

// FromFontInfo converts FontInfo to Font
func (f *Font) FromFontInfo(info FontInfo) {
	f.Name = info.Name
	f.License = info.License
	f.Version = info.Version
	f.Description = info.Description
	f.LastModified = info.LastModified.Format("2006-01-02T15:04:05Z")
	f.MetadataURL = info.MetadataURL
	f.SourceURL = info.SourceURL
	f.Categories = info.Categories
	f.Subsets = info.Subsets
	// Convert variants from []string to []FontVariant
	f.Variants = make([]FontVariant, len(info.Variants))
	for i, variant := range info.Variants {
		f.Variants[i] = FontVariant{Name: variant}
	}
}

// FromFontFileInfo converts FontFileInfo to Font
func (f *Font) FromFontFileInfo(info FontFileInfo) {
	f.Name = info.Name
	f.ID = info.ID
	f.Source = info.Source
	f.License = info.License
	f.Version = info.Version
	f.Description = info.Description
	f.Files = info.Files
	f.Subsets = info.Subsets
	// Convert variants from []string to []FontVariant
	f.Variants = make([]FontVariant, len(info.Variants))
	for i, variant := range info.Variants {
		f.Variants[i] = FontVariant{Name: variant}
	}
	// Set category from single string to slice
	if info.Category != "" {
		f.Categories = []string{info.Category}
	}
}

// FontManifest represents the combined manifest of all fonts
type FontManifest struct {
	Version     string                `json:"version"`
	LastUpdated time.Time             `json:"last_updated"`
	Sources     map[string]SourceInfo `json:"sources"`
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
	Name         string                       `json:"name"`
	License      string                       `json:"license"`
	LicenseURL   string                       `json:"license_url,omitempty"`
	Variants     []string                     `json:"variants"`
	Subsets      []string                     `json:"subsets"`
	Version      string                       `json:"version"`
	Description  string                       `json:"description"`
	LastModified time.Time                    `json:"last_modified"`
	MetadataURL  string                       `json:"metadata_url"`
	SourceURL    string                       `json:"source_url"`
	Categories   []string                     `json:"categories,omitempty"`
	Tags         []string                     `json:"tags,omitempty"`
	Popularity   int                          `json:"popularity,omitempty"`
	Files        map[string]string            `json:"files,omitempty"`
	VariantFiles map[string]map[string]string `json:"variant_files,omitempty"` // variant_name -> file_type -> url
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
