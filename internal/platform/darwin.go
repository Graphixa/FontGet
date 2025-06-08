//go:build darwin
// +build darwin

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type darwinFontManager struct {
	fontDir string
}

// NewFontManager creates a new FontManager for macOS
func NewFontManager() (FontManager, error) {
	// Get the user's font directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	fontDir := filepath.Join(homeDir, "Library", "Fonts")
	if err := ensureDir(fontDir); err != nil {
		return nil, fmt.Errorf("failed to ensure font directory exists: %w", err)
	}

	return &darwinFontManager{
		fontDir: fontDir,
	}, nil
}

// InstallFont installs a font file to the macOS font directory
func (m *darwinFontManager) InstallFont(fontPath string) error {
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

// RemoveFont removes a font from the macOS font directory
func (m *darwinFontManager) RemoveFont(fontName string) error {
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

// GetFontDir returns the macOS font directory
func (m *darwinFontManager) GetFontDir() string {
	return m.fontDir
}

// updateFontCache runs atsutil to update the font cache
func (m *darwinFontManager) updateFontCache() error {
	// First, reset the font cache
	cmd := exec.Command("atsutil", "databases", "-remove")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("atsutil remove failed: %v\nOutput: %s", err, string(output))
	}

	// Then, reset the font cache for the current user
	cmd = exec.Command("atsutil", "databases", "-removeUser")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("atsutil removeUser failed: %v\nOutput: %s", err, string(output))
	}

	// Finally, restart the font server
	cmd = exec.Command("atsutil", "server", "-shutdown")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("atsutil shutdown failed: %v\nOutput: %s", err, string(output))
	}

	cmd = exec.Command("atsutil", "server", "-ping")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("atsutil ping failed: %v\nOutput: %s", err, string(output))
	}

	return nil
}
