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
	fontDir string
}

// NewFontManager creates a new FontManager for Windows
func NewFontManager() (FontManager, error) {
	// Get the Windows font directory
	fontDir := filepath.Join(os.Getenv("WINDIR"), "Fonts")
	if err := ensureDir(fontDir); err != nil {
		return nil, fmt.Errorf("failed to ensure font directory exists: %w", err)
	}

	return &windowsFontManager{
		fontDir: fontDir,
	}, nil
}

// InstallFont installs a font file to the Windows font directory
func (m *windowsFontManager) InstallFont(fontPath string) error {
	fontName := getFontName(fontPath)
	targetPath := filepath.Join(m.fontDir, fontName)

	// Copy the font file to the Windows font directory
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

// RemoveFont removes a font from the Windows font directory
func (m *windowsFontManager) RemoveFont(fontName string) error {
	fontPath := filepath.Join(m.fontDir, fontName)

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

// GetFontDir returns the Windows font directory
func (m *windowsFontManager) GetFontDir() string {
	return m.fontDir
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
