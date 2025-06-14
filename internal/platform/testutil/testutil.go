package testutil

import (
	"fmt"
	"os"
	"path/filepath"
)

// TestFont represents a test font file
type TestFont struct {
	Name     string
	Path     string
	Contents []byte
}

// CreateTestFont creates a test font file in the specified directory
func CreateTestFont(dir, name string) (*TestFont, error) {
	// Create a simple TTF file with minimal valid content
	// This is not a real font, but it has the correct extension
	contents := []byte{
		0x00, 0x01, 0x00, 0x00, // TTF signature
		0x00, 0x00, 0x00, 0x01, // Number of tables
		0x00, 0x00, 0x00, 0x00, // Search range
		0x00, 0x00, 0x00, 0x00, // Entry selector
		0x00, 0x00, 0x00, 0x00, // Range shift
	}

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, contents, 0644); err != nil {
		return nil, fmt.Errorf("failed to create test font: %w", err)
	}

	return &TestFont{
		Name:     name,
		Path:     path,
		Contents: contents,
	}, nil
}

// CleanupTestFont removes a test font file
func CleanupTestFont(font *TestFont) error {
	if font == nil {
		return nil
	}
	return os.Remove(font.Path)
}

// CreateTestFontDir creates a temporary directory for test fonts
func CreateTestFontDir() (string, error) {
	dir, err := os.MkdirTemp("", "fontget-test-*")
	if err != nil {
		return "", fmt.Errorf("failed to create test directory: %w", err)
	}
	return dir, nil
}

// CleanupTestFontDir removes a test font directory
func CleanupTestFontDir(dir string) error {
	return os.RemoveAll(dir)
}
