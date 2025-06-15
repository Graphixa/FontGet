package platform

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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
	InstallFont(fontPath string, scope InstallationScope, force bool) error
	// RemoveFont removes a font from the system
	RemoveFont(fontName string, scope InstallationScope) error
	// GetFontDir returns the system's font directory for the given scope
	GetFontDir(scope InstallationScope) string
	// RequiresElevation returns whether the given scope requires elevation
	RequiresElevation(scope InstallationScope) bool
	// IsElevated checks if the current process is running with elevated privileges
	IsElevated() (bool, error)
	// GetElevationCommand returns the command to run the current process with elevation
	GetElevationCommand() (string, []string, error)
}

// Common helper functions
func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

func copyFile(src, dst string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Get source file info for permissions
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get source file info: %w", err)
	}

	// Create destination file with same permissions
	dstFile, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// Copy file contents in chunks
	buf := make([]byte, 32*1024) // 32KB buffer
	for {
		n, err := srcFile.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read source file: %w", err)
		}
		if n == 0 {
			break
		}
		if _, err := dstFile.Write(buf[:n]); err != nil {
			return fmt.Errorf("failed to write destination file: %w", err)
		}
	}

	// Ensure all data is written to disk
	if err := dstFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync destination file: %w", err)
	}

	return nil
}

// getFontName extracts the font filename from the path
func getFontName(fontPath string) string {
	return filepath.Base(fontPath)
}

// ListInstalledFonts returns a list of font files in the specified directory
func ListInstalledFonts(dir string) ([]string, error) {
	var fonts []string

	// Walk through the directory
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if the file is a font file
		ext := strings.ToLower(filepath.Ext(path))
		if isFontFile(ext) {
			fonts = append(fonts, filepath.Base(path))
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list fonts: %w", err)
	}

	return fonts, nil
}

// isFontFile checks if the file extension indicates a font file
func isFontFile(ext string) bool {
	switch ext {
	case ".ttf", ".otf", ".ttc", ".otc", ".pfb", ".pfm", ".pfa", ".bdf", ".pcf", ".psf", ".psfu":
		return true
	default:
		return false
	}
}
