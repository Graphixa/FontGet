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

	// System font directory
	systemFontDir := "/usr/local/share/fonts"

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

// FlushFontCache runs fc-cache once for the scope.
func (m *linuxFontManager) FlushFontCache(scope InstallationScope) error {
	return m.updateFontCache(scope)
}

// InstallFont installs a font file to the specified font directory
func (m *linuxFontManager) InstallFont(fontPath string, scope InstallationScope, force bool, opts *InstallFontOptions) error {
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

	if err := ensureDir(targetDir); err != nil {
		return fmt.Errorf("failed to ensure font directory exists for scope %s at %q: %w", scope, targetDir, err)
	}

	targetPath := filepath.Join(targetDir, fontName)

	mut, err := placeFontFile(fontPath, targetPath, force, opts)
	if err != nil {
		return err
	}
	mut.FontName = fontName
	mut.Scope = scope
	if opts != nil && opts.Mutation != nil {
		*opts.Mutation = mut
	}
	if opts != nil && opts.FailPoint == InstallFailRegister {
		_ = RollbackMutation(mut)
		return failPointError(InstallFailRegister)
	}

	skipCache := opts != nil && opts.SkipPostInstallCacheRefresh
	if !skipCache {
		if err := m.updateFontCache(scope); err != nil {
			_ = RollbackMutation(mut)
			return fmt.Errorf("failed to update font cache: %w", err)
		}
	}

	return nil
}

// RemoveFont removes a font from the specified font directory
func (m *linuxFontManager) RemoveFont(fontName string, scope InstallationScope, opts *RemoveFontOptions) error {
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

	skipCache := opts != nil && opts.SkipPostRemoveCacheRefresh
	if !skipCache {
		// Update the font cache
		if err := m.updateFontCache(scope); err != nil {
			return fmt.Errorf("failed to update font cache: %w", err)
		}
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
