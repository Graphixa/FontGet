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
	userFontDir   string
	systemFontDir string
}

// NewFontManager creates a new FontManager for Linux
func NewFontManager() (FontManager, error) {
	// Get the user's font directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	userFontDir := filepath.Join(homeDir, ".local", "share", "fonts")
	if err := ensureDir(userFontDir); err != nil {
		return nil, fmt.Errorf("failed to ensure user font directory exists: %w", err)
	}

	// System font directory
	systemFontDir := "/usr/local/share/fonts"
	if err := ensureDir(systemFontDir); err != nil {
		return nil, fmt.Errorf("failed to ensure system font directory exists: %w", err)
	}

	return &linuxFontManager{
		userFontDir:   userFontDir,
		systemFontDir: systemFontDir,
	}, nil
}

// IsElevated checks if the current process is running with root privileges
func (m *linuxFontManager) IsElevated() (bool, error) {
	return os.Geteuid() == 0, nil
}

// GetElevationCommand returns the command to run the current process with sudo
func (m *linuxFontManager) GetElevationCommand() (string, []string, error) {
	// Get the executable path
	exe, err := os.Executable()
	if err != nil {
		return "", nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	// Build the command with sudo
	args := append([]string{exe}, os.Args[1:]...)
	return "sudo", args, nil
}

// InstallFont installs a font file to the specified font directory
func (m *linuxFontManager) InstallFont(fontPath string, scope InstallationScope, force bool) error {
	fontName := getFontName(fontPath)
	var targetDir string

	switch scope {
	case UserScope:
		targetDir = m.userFontDir
	case MachineScope:
		targetDir = m.systemFontDir
	default:
		return fmt.Errorf("invalid installation scope: %s", scope)
	}

	targetPath := filepath.Join(targetDir, fontName)

	// Check if font is already installed
	if _, err := os.Stat(targetPath); err == nil {
		if !force {
			return fmt.Errorf("font already installed: %s", fontName)
		}
		// Remove the existing file if force is true
		if err := os.Remove(targetPath); err != nil {
			return fmt.Errorf("failed to overwrite existing font: %w", err)
		}
	}

	// Copy the font file to the target directory
	if err := copyFile(fontPath, targetPath); err != nil {
		return fmt.Errorf("failed to copy font file: %w", err)
	}

	// Update the font cache
	if err := m.updateFontCache(scope); err != nil {
		// Clean up the file if cache update fails
		os.Remove(targetPath)
		return fmt.Errorf("failed to update font cache: %w", err)
	}

	return nil
}

// RemoveFont removes a font from the specified font directory
func (m *linuxFontManager) RemoveFont(fontName string, scope InstallationScope) error {
	var targetDir string

	switch scope {
	case UserScope:
		targetDir = m.userFontDir
	case MachineScope:
		targetDir = m.systemFontDir
	default:
		return fmt.Errorf("invalid installation scope: %s", scope)
	}

	fontPath := filepath.Join(targetDir, fontName)

	// Delete the font file
	if err := os.Remove(fontPath); err != nil {
		return fmt.Errorf("failed to remove font file: %w", err)
	}

	// Update the font cache
	if err := m.updateFontCache(scope); err != nil {
		return fmt.Errorf("failed to update font cache: %w", err)
	}

	return nil
}

// GetFontDir returns the font directory for the specified scope
func (m *linuxFontManager) GetFontDir(scope InstallationScope) string {
	switch scope {
	case UserScope:
		return m.userFontDir
	case MachineScope:
		return m.systemFontDir
	default:
		return m.userFontDir // Default to user scope
	}
}

// RequiresElevation returns whether the given scope requires elevation
func (m *linuxFontManager) RequiresElevation(scope InstallationScope) bool {
	return scope == MachineScope
}

// updateFontCache runs fc-cache to update the font cache
func (m *linuxFontManager) updateFontCache(scope InstallationScope) error {
	var cmd *exec.Cmd

	switch scope {
	case UserScope:
		// Update user font cache
		cmd = exec.Command("fc-cache", "-f", "-v", m.userFontDir)
	case MachineScope:
		// Update system font cache
		cmd = exec.Command("fc-cache", "-f", "-v", m.systemFontDir)
	default:
		return fmt.Errorf("invalid installation scope: %s", scope)
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("fc-cache failed: %v\nOutput: %s", err, string(output))
	}

	return nil
}

// CreateHiddenDirectory creates a directory and sets it as hidden on Linux
func CreateHiddenDirectory(path string, perm os.FileMode) error {
	// On macOS/Linux, directories starting with . are automatically hidden
	// Just create the directory normally
	return os.MkdirAll(path, perm)
}
