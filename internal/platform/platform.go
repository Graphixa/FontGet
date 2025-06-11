package platform

import (
	"fmt"
	"os"
	"path/filepath"
)

// InstallationScope defines where fonts should be installed
type InstallationScope string

const (
	// UserScope installs fonts for the current user only
	UserScope InstallationScope = "user"
	// MachineScope installs fonts system-wide
	MachineScope InstallationScope = "machine"
)

// FontManager defines the interface for platform-specific font operations
type FontManager interface {
	// InstallFont installs a font file to the system
	InstallFont(fontPath string, scope InstallationScope) error
	// RemoveFont removes a font from the system
	RemoveFont(fontName string, scope InstallationScope) error
	// GetFontDir returns the system's font directory for the given scope
	GetFontDir(scope InstallationScope) string
	// RequiresElevation returns whether the given scope requires elevation
	RequiresElevation(scope InstallationScope) bool
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
