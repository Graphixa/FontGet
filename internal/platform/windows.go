//go:build windows
// +build windows

package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

type windowsFontManager struct {
	systemFontDir string
	userFontDir   string
}

// NewFontManager creates a new FontManager for Windows
func NewFontManager() (FontManager, error) {
	// Get the Windows system font directory
	systemFontDir := filepath.Join(os.Getenv("WINDIR"), "Fonts")
	if err := ensureDir(systemFontDir); err != nil {
		return nil, fmt.Errorf("failed to ensure system font directory exists: %w", err)
	}

	// Get the user font directory
	userFontDir := filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Windows", "Fonts")
	if err := ensureDir(userFontDir); err != nil {
		return nil, fmt.Errorf("failed to ensure user font directory exists: %w", err)
	}

	return &windowsFontManager{
		systemFontDir: systemFontDir,
		userFontDir:   userFontDir,
	}, nil
}

// InstallFont installs a font file to the specified font directory
func (m *windowsFontManager) InstallFont(fontPath string, scope InstallationScope) error {
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

	// Add the font to the system using AddFontResource
	if err := m.addFontResource(targetPath); err != nil {
		// Clean up the file if installation fails
		os.Remove(targetPath)
		return fmt.Errorf("failed to add font resource: %w", err)
	}

	// Notify other applications about the new font
	if err := m.notifyFontChange(); err != nil {
		return fmt.Errorf("failed to notify font change: %w", err)
	}

	return nil
}

// RemoveFont removes a font from the specified font directory
func (m *windowsFontManager) RemoveFont(fontName string, scope InstallationScope) error {
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

	// Remove the font resource
	if err := m.removeFontResource(fontPath); err != nil {
		return fmt.Errorf("failed to remove font resource: %w", err)
	}

	// Delete the font file
	if err := os.Remove(fontPath); err != nil {
		return fmt.Errorf("failed to remove font file: %w", err)
	}

	// Notify other applications about the font removal
	if err := m.notifyFontChange(); err != nil {
		return fmt.Errorf("failed to notify font change: %w", err)
	}

	return nil
}

// GetFontDir returns the font directory for the specified scope
func (m *windowsFontManager) GetFontDir(scope InstallationScope) string {
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
func (m *windowsFontManager) RequiresElevation(scope InstallationScope) bool {
	return scope == MachineScope
}

// Windows API functions
var (
	gdi32  = syscall.NewLazyDLL("gdi32.dll")
	user32 = syscall.NewLazyDLL("user32.dll")

	addFontResource    = gdi32.NewProc("AddFontResourceW")
	removeFontResource = gdi32.NewProc("RemoveFontResourceW")
	sendMessage        = user32.NewProc("SendMessageW")
)

const (
	HWND_BROADCAST = 0xFFFF
	WM_FONTCHANGE  = 0x001D
)

func (m *windowsFontManager) addFontResource(fontPath string) error {
	pathPtr, err := syscall.UTF16PtrFromString(fontPath)
	if err != nil {
		return err
	}

	ret, _, _ := addFontResource.Call(uintptr(unsafe.Pointer(pathPtr)))
	if ret == 0 {
		return fmt.Errorf("AddFontResource failed")
	}

	return nil
}

func (m *windowsFontManager) removeFontResource(fontPath string) error {
	pathPtr, err := syscall.UTF16PtrFromString(fontPath)
	if err != nil {
		return err
	}

	ret, _, _ := removeFontResource.Call(uintptr(unsafe.Pointer(pathPtr)))
	if ret == 0 {
		return fmt.Errorf("RemoveFontResource failed")
	}

	return nil
}

func (m *windowsFontManager) notifyFontChange() error {
	ret, _, _ := sendMessage.Call(
		HWND_BROADCAST,
		WM_FONTCHANGE,
		0,
		0,
	)
	if ret == 0 {
		return fmt.Errorf("SendMessage failed")
	}

	return nil
}
