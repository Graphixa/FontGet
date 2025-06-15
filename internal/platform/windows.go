//go:build windows
// +build windows

package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
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
func (m *windowsFontManager) InstallFont(fontPath string, scope InstallationScope, force bool) error {
	fmt.Printf("Starting font installation for: %s (scope: %s)\n", fontPath, scope)

	// Validate font file
	fmt.Println("Validating font file...")
	if err := validateFontFile(fontPath); err != nil {
		return fmt.Errorf("font validation failed: %w", err)
	}
	fmt.Println("Font validation successful")

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
	fmt.Printf("Target directory: %s\n", targetDir)

	targetPath := filepath.Join(targetDir, fontName)
	fmt.Printf("Target path: %s\n", targetPath)

	// Check if font is already installed
	fmt.Println("Checking if font is already installed...")
	if _, err := os.Stat(targetPath); err == nil {
		if !force {
			return fmt.Errorf("font already installed: %s", fontName)
		}
		fmt.Println("Font exists, removing due to force flag...")
		// Remove the existing file if force is true
		if err := os.Remove(targetPath); err != nil {
			return fmt.Errorf("failed to overwrite existing font: %w", err)
		}
	}

	// Copy the font file to the target directory
	fmt.Println("Copying font file to target directory...")
	if err := copyFile(fontPath, targetPath); err != nil {
		return fmt.Errorf("failed to copy font file: %w", err)
	}
	fmt.Println("Font file copied successfully")

	// Add the font to the system
	fmt.Println("Adding font resource...")
	if err := m.addFontResource(targetPath); err != nil {
		fmt.Printf("Failed to add font resource: %v\n", err)
		// Clean up on error
		fmt.Println("Cleaning up after failed font resource addition...")
		os.Remove(targetPath)
		return fmt.Errorf("failed to add font resource: %w", err)
	}
	fmt.Println("Font resource added successfully")

	// Add font to registry if machine scope
	if scope == MachineScope {
		fmt.Println("Adding font to registry...")
		if err := m.addFontToRegistry(fontName, targetPath); err != nil {
			fmt.Printf("Failed to add font to registry: %v\n", err)
			// Clean up on error
			fmt.Println("Cleaning up after failed registry addition...")
			m.removeFontResource(targetPath)
			os.Remove(targetPath)
			return fmt.Errorf("failed to add font to registry: %w", err)
		}
		fmt.Println("Font added to registry successfully")
	}

	// Notify other applications about the new font
	fmt.Println("Notifying system about font change...")
	if err := m.notifyFontChange(); err != nil {
		fmt.Printf("Failed to notify font change: %v\n", err)
		// Clean up on error
		fmt.Println("Cleaning up after failed notification...")
		m.removeFontResource(targetPath)
		if scope == MachineScope {
			m.removeFontFromRegistry(fontName)
		}
		os.Remove(targetPath)
		return fmt.Errorf("failed to notify font change: %w", err)
	}
	fmt.Println("Font change notification sent successfully")

	fmt.Println("Font installation completed successfully")
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

	// Check if font exists
	if _, err := os.Stat(fontPath); os.IsNotExist(err) {
		return fmt.Errorf("font not found: %s", fontName)
	}

	// Remove the font resource
	if err := m.removeFontResource(fontPath); err != nil {
		return fmt.Errorf("failed to remove font resource: %w", err)
	}

	// Delete the font file
	if err := os.Remove(fontPath); err != nil {
		// Try to restore the font resource if file deletion fails
		m.addFontResource(fontPath)
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

// IsElevated checks if the current process is running with administrator privileges
func (m *windowsFontManager) IsElevated() (bool, error) {
	return IsElevated()
}

// GetElevationCommand returns the command to run the current process with UAC elevation
func (m *windowsFontManager) GetElevationCommand() (string, []string, error) {
	// Get the executable path
	exe, err := os.Executable()
	if err != nil {
		return "", nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	// For Windows, we'll use the current executable with the same arguments
	// The elevation will be handled by the Windows API
	return exe, os.Args[1:], nil
}

// validateFontFile checks if the file is a valid font file
func validateFontFile(fontPath string) error {
	ext := strings.ToLower(filepath.Ext(fontPath))
	validExts := map[string]bool{
		".ttf":  true,
		".otf":  true,
		".ttc":  true,
		".otc":  true,
		".pfb":  true,
		".pfm":  true,
		".pfa":  true,
		".bdf":  true,
		".pcf":  true,
		".psf":  true,
		".psfu": true,
	}

	if !validExts[ext] {
		return fmt.Errorf("unsupported font file format: %s", ext)
	}

	// Check if file exists and is readable
	file, err := os.Open(fontPath)
	if err != nil {
		return fmt.Errorf("failed to open font file: %w", err)
	}
	defer file.Close()

	// Check file size (minimum 1KB to avoid empty files)
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get font file info: %w", err)
	}
	if stat.Size() < 1024 {
		return fmt.Errorf("font file is too small to be valid")
	}

	return nil
}

func (m *windowsFontManager) addFontResource(fontPath string) error {
	fmt.Printf("Adding font resource for: %s\n", fontPath)
	pathPtr, err := syscall.UTF16PtrFromString(fontPath)
	if err != nil {
		return fmt.Errorf("failed to convert path to UTF-16: %w", err)
	}

	fmt.Println("Calling AddFontResource...")
	ret, _, _ := addFontResource.Call(uintptr(unsafe.Pointer(pathPtr)))
	if ret == 0 {
		// Get the last error code
		errCode := syscall.GetLastError()
		fmt.Printf("AddFontResource failed with error code: %d\n", errCode)
		return fmt.Errorf("AddFontResource failed with error code: %d", errCode)
	}
	fmt.Println("AddFontResource call successful")

	return nil
}

func (m *windowsFontManager) removeFontResource(fontPath string) error {
	pathPtr, err := syscall.UTF16PtrFromString(fontPath)
	if err != nil {
		return fmt.Errorf("failed to convert path to UTF-16: %w", err)
	}

	ret, _, _ := removeFontResource.Call(uintptr(unsafe.Pointer(pathPtr)))
	if ret == 0 {
		// Get the last error code
		errCode := syscall.GetLastError()
		return fmt.Errorf("RemoveFontResource failed with error code: %d", errCode)
	}

	return nil
}

func (m *windowsFontManager) notifyFontChange() error {
	fmt.Println("Sending font change notification...")

	// Create a channel to receive the result
	result := make(chan error, 1)

	// Run PostMessage in a goroutine
	go func() {
		ret, _, _ := postMessage.Call(
			HWND_BROADCAST,
			WM_FONTCHANGE,
			0,
			0,
		)
		if ret == 0 {
			// Get the last error code
			errCode := syscall.GetLastError()
			fmt.Printf("PostMessage failed with error code: %d\n", errCode)
			result <- fmt.Errorf("PostMessage failed with error code: %d", errCode)
			return
		}
		result <- nil
	}()

	// Wait for the result with a timeout
	select {
	case err := <-result:
		if err != nil {
			return err
		}
		fmt.Println("Font change notification sent successfully")
		return nil
	case <-time.After(2 * time.Second):
		fmt.Println("Warning: Font change notification timed out, but continuing...")
		return nil
	}
}

// addFontToRegistry adds a font to the Windows registry
func (m *windowsFontManager) addFontToRegistry(fontName, fontPath string) error {
	// Convert strings to UTF-16
	fontNamePtr, err := syscall.UTF16PtrFromString(fontName)
	if err != nil {
		return fmt.Errorf("failed to convert font name to UTF-16: %w", err)
	}

	fontPathPtr, err := syscall.UTF16PtrFromString(fontPath)
	if err != nil {
		return fmt.Errorf("failed to convert font path to UTF-16: %w", err)
	}

	// Open the registry key
	var hKey syscall.Handle
	keyPath, err := syscall.UTF16PtrFromString("SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\Fonts")
	if err != nil {
		return fmt.Errorf("failed to convert key path to UTF-16: %w", err)
	}

	ret, _, err := regCreateKeyEx.Call(
		uintptr(HKEY_LOCAL_MACHINE),
		uintptr(unsafe.Pointer(keyPath)),
		0,
		0,
		0,
		uintptr(KEY_WRITE),
		0,
		uintptr(unsafe.Pointer(&hKey)),
		0,
	)
	if ret != 0 {
		return fmt.Errorf("failed to open registry key: %w", err)
	}
	defer regCloseKey.Call(uintptr(hKey))

	// Set the font value
	ret, _, err = regSetValueEx.Call(
		uintptr(hKey),
		uintptr(unsafe.Pointer(fontNamePtr)),
		0,
		uintptr(REG_SZ),
		uintptr(unsafe.Pointer(fontPathPtr)),
		uintptr((len(fontPath)+1)*2),
	)
	if ret != 0 {
		return fmt.Errorf("failed to set registry value: %w", err)
	}

	return nil
}

// removeFontFromRegistry removes a font from the Windows registry
func (m *windowsFontManager) removeFontFromRegistry(fontName string) error {
	// Convert string to UTF-16
	fontNamePtr, err := syscall.UTF16PtrFromString(fontName)
	if err != nil {
		return fmt.Errorf("failed to convert font name to UTF-16: %w", err)
	}

	// Open the registry key
	var hKey syscall.Handle
	keyPath, err := syscall.UTF16PtrFromString("SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\Fonts")
	if err != nil {
		return fmt.Errorf("failed to convert key path to UTF-16: %w", err)
	}

	ret, _, err := regCreateKeyEx.Call(
		uintptr(HKEY_LOCAL_MACHINE),
		uintptr(unsafe.Pointer(keyPath)),
		0,
		0,
		0,
		uintptr(KEY_WRITE),
		0,
		uintptr(unsafe.Pointer(&hKey)),
		0,
	)
	if ret != 0 {
		return fmt.Errorf("failed to open registry key: %w", err)
	}
	defer regCloseKey.Call(uintptr(hKey))

	// Delete the font value
	ret, _, err = regSetValueEx.Call(
		uintptr(hKey),
		uintptr(unsafe.Pointer(fontNamePtr)),
		0,
		uintptr(REG_SZ),
		0,
		0,
	)
	if ret != 0 {
		return fmt.Errorf("failed to delete registry value: %w", err)
	}

	return nil
}
