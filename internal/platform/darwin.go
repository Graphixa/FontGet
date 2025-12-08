//go:build darwin
// +build darwin

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
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
func (m *darwinFontManager) InstallFont(fontPath string, scope InstallationScope, force bool) error {
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

	// Update the font cache (non-critical on macOS 14+)
	// Fonts in ~/Library/Fonts and /Library/Fonts are auto-detected by macOS
	if err := m.updateFontCache(scope); err != nil {
		// Cache refresh failure is non-critical - font is already installed
		// On macOS 14+, fonts are auto-detected without manual cache refresh
		// Don't remove the file - installation succeeded, cache refresh is optional
		// Return a warning-style error that can be handled gracefully
		return fmt.Errorf("font installed successfully, but cache refresh failed (non-critical): %w", err)
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

	// Update the font cache (non-critical on macOS 14+)
	// Font removal is effective immediately, cache refresh is optional
	if err := m.updateFontCache(scope); err != nil {
		// Cache refresh failure is non-critical - font is already removed
		// On macOS 14+, font removal is effective without manual cache refresh
		// Return a warning-style error that can be handled gracefully
		return fmt.Errorf("font removed successfully, but cache refresh failed (non-critical): %w", err)
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

// updateFontCache refreshes the font cache on macOS
// Uses modern method compatible with macOS 14+ (Sonoma)
// On macOS 14+, atsutil was removed, so we use pkill fontd instead
func (m *darwinFontManager) updateFontCache(scope InstallationScope) error {
	// Modern approach: restart fontd service to refresh cache
	// fontd automatically restarts and picks up new fonts
	// This works on both older macOS versions and macOS 14+

	// Try pkill first (more reliable and available on all macOS versions)
	cmd := exec.Command("pkill", "fontd")
	if output, err := cmd.CombinedOutput(); err != nil {
		// pkill returns non-zero exit code if no process was found
		// This is not necessarily an error - fontd may not be running
		// Check if the error is "no process found" vs actual failure
		errStr := string(output)
		if !contains(errStr, "No matching processes") && !contains(errStr, "no process found") {
			// If pkill fails for other reasons, try killall as fallback
			cmd = exec.Command("killall", "fontd")
			if output, err := cmd.CombinedOutput(); err != nil {
				// Both methods failed, but this is non-critical
				// macOS will auto-detect fonts in ~/Library/Fonts and /Library/Fonts
				// Fonts will be available after app restart or system refresh
				return fmt.Errorf("failed to refresh font cache (non-critical, fonts will still work): %v\nOutput: %s", err, string(output))
			}
		}
	}

	// Small delay to allow fontd to restart
	time.Sleep(500 * time.Millisecond)

	return nil
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// toLower converts a string to lowercase (simple implementation)
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

// CreateHiddenDirectory creates a directory and sets it as hidden on macOS
func CreateHiddenDirectory(path string, perm os.FileMode) error {
	// On macOS/Linux, directories starting with . are automatically hidden
	// Just create the directory normally
	return os.MkdirAll(path, perm)
}
