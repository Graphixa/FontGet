package repo

import (
	"os"
	"path/filepath"
)

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
