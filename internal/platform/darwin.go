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
	userFontDir   string
	systemFontDir string
}

// NewFontManager creates a new FontManager for macOS
func NewFontManager() (FontManager, error) {
	// Get the user's font directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	userFontDir := filepath.Join(homeDir, "Library", "Fonts")
	if err := ensureDir(userFontDir); err != nil {
		return nil, fmt.Errorf("failed to ensure user font directory exists: %w", err)
	}

	// System font directory
	systemFontDir := "/Library/Fonts"
	if err := ensureDir(systemFontDir); err != nil {
		return nil, fmt.Errorf("failed to ensure system font directory exists: %w", err)
	}

	return &darwinFontManager{
		userFontDir:   userFontDir,
		systemFontDir: systemFontDir,
	}, nil
}

// IsElevated checks if the current process is running with root privileges
func (m *darwinFontManager) IsElevated() (bool, error) {
	return os.Geteuid() == 0, nil
}

// GetElevationCommand returns the command to run the current process with sudo
func (m *darwinFontManager) GetElevationCommand() (string, []string, error) {
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
func (m *darwinFontManager) InstallFont(fontPath string, scope InstallationScope) error {
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
func (m *darwinFontManager) RemoveFont(fontName string, scope InstallationScope) error {
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
func (m *darwinFontManager) GetFontDir(scope InstallationScope) string {
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
func (m *darwinFontManager) RequiresElevation(scope InstallationScope) bool {
	return scope == MachineScope
}

// updateFontCache runs atsutil to update the font cache
func (m *darwinFontManager) updateFontCache(scope InstallationScope) error {
	var cmds []*exec.Cmd

	switch scope {
	case UserScope:
		// Reset user font cache
		cmds = []*exec.Cmd{
			exec.Command("atsutil", "databases", "-removeUser"),
			exec.Command("atsutil", "server", "-shutdown"),
			exec.Command("atsutil", "server", "-ping"),
		}
	case MachineScope:
		// Reset system font cache
		cmds = []*exec.Cmd{
			exec.Command("atsutil", "databases", "-remove"),
			exec.Command("atsutil", "databases", "-removeUser"),
			exec.Command("atsutil", "server", "-shutdown"),
			exec.Command("atsutil", "server", "-ping"),
		}
	default:
		return fmt.Errorf("invalid installation scope: %s", scope)
	}

	// Execute each command in sequence
	for _, cmd := range cmds {
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("atsutil command failed: %v\nOutput: %s", err, string(output))
		}
	}

	return nil
}
