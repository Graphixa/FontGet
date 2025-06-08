package platform

import (
	"fmt"
	"os"
	"path/filepath"
)

// FontManager defines the interface for platform-specific font operations
type FontManager interface {
	// InstallFont installs a font file to the system
	InstallFont(fontPath string) error
	// RemoveFont removes a font from the system
	RemoveFont(fontName string) error
	// GetFontDir returns the system's font directory
	GetFontDir() string
}

// Common helper functions
func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	err = os.WriteFile(dst, input, 0644)
	if err != nil {
		return fmt.Errorf("failed to write destination file: %w", err)
	}

	return nil
}

func getFontName(path string) string {
	return filepath.Base(path)
}
