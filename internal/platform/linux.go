//go:build linux
// +build linux

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type linuxFontManager struct {
	fontDir string
}

// NewFontManager creates a new FontManager for Linux
func NewFontManager() (FontManager, error) {
	// Get the user's font directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	fontDir := filepath.Join(homeDir, ".local", "share", "fonts")
	if err := ensureDir(fontDir); err != nil {
		return nil, fmt.Errorf("failed to ensure font directory exists: %w", err)
	}

	return &linuxFontManager{
		fontDir: fontDir,
	}, nil
}

// InstallFont installs a font file to the Linux font directory
func (m *linuxFontManager) InstallFont(fontPath string) error {
	fontName := getFontName(fontPath)
	targetPath := filepath.Join(m.fontDir, fontName)

	// Copy the font file to the font directory
	if err := copyFile(fontPath, targetPath); err != nil {
		return fmt.Errorf("failed to copy font file: %w", err)
	}

	// Update the font cache
	if err := m.updateFontCache(); err != nil {
		// Clean up the file if cache update fails
		os.Remove(targetPath)
		return fmt.Errorf("failed to update font cache: %w", err)
	}

	return nil
}

// RemoveFont removes a font from the Linux font directory
func (m *linuxFontManager) RemoveFont(fontName string) error {
	fontPath := filepath.Join(m.fontDir, fontName)

	// Delete the font file
	if err := os.Remove(fontPath); err != nil {
		return fmt.Errorf("failed to remove font file: %w", err)
	}

	// Update the font cache
	if err := m.updateFontCache(); err != nil {
		return fmt.Errorf("failed to update font cache: %w", err)
	}

	return nil
}

// GetFontDir returns the Linux font directory
func (m *linuxFontManager) GetFontDir() string {
	return m.fontDir
}

// updateFontCache runs fc-cache to update the font cache
func (m *linuxFontManager) updateFontCache() error {
	cmd := exec.Command("fc-cache", "-f", "-v")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("fc-cache failed: %v\nOutput: %s", err, string(output))
	}
	return nil
}
