//go:build windows
// +build windows

package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"fontget/internal/logging"
)

// Windows API constants
const (
	WM_FONTCHANGE = 0x001D
)

// Windows API functions
var (
	findWindowEx = syscall.NewLazyDLL("user32.dll").NewProc("FindWindowExW")
	sendMessage  = syscall.NewLazyDLL("user32.dll").NewProc("SendMessageW")
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
	logger := logging.GetLogger()
	logger.Debug("Starting font installation for: %s (scope: %s)", fontPath, scope)

	// Validate font file
	logger.Debug("Validating font file...")
	if err := validateFontFile(fontPath); err != nil {
		logger.Error("Font validation failed for %s: %v", fontPath, err)
		return fmt.Errorf("font validation failed: %w", err)
	}
	logger.Debug("Font validation successful")

	fontName := getFontName(fontPath)
	var targetDir string

	switch scope {
	case UserScope:
		targetDir = m.userFontDir
		logger.Debug("Using user font directory: %s", targetDir)
	case MachineScope:
		targetDir = m.systemFontDir
		logger.Debug("Using system font directory: %s", targetDir)
	default:
		logger.Error("Invalid installation scope: %s", scope)
		return fmt.Errorf("invalid installation scope: %s", scope)
	}

	targetPath := filepath.Join(targetDir, fontName)
	logger.Debug("Target path: %s", targetPath)

	// Check if font is already installed
	logger.Debug("Checking if font is already installed...")
	if _, err := os.Stat(targetPath); err == nil {
		if !force {
			logger.Warn("Font already installed at %s", targetPath)
			return fmt.Errorf("font already installed: %s", fontName)
		}
		logger.Debug("Font exists, removing due to force flag...")
		// Remove the existing file if force is true
		if err := os.Remove(targetPath); err != nil {
			logger.Error("Failed to overwrite existing font at %s: %v", targetPath, err)
			return fmt.Errorf("failed to overwrite existing font: %w", err)
		}
		logger.Debug("Existing font removed successfully")
	}

	// Copy the font file to the target directory
	logger.Debug("Copying font file to target directory...")
	if err := copyFile(fontPath, targetPath); err != nil {
		logger.Error("Failed to copy font file from %s to %s: %v", fontPath, targetPath, err)
		return fmt.Errorf("failed to copy font file: %w", err)
	}
	logger.Debug("Font file copied successfully")

	// Add the font to the system
	logger.Debug("Adding font resource...")
	if err := AddFontResource(targetPath); err != nil {
		logger.Error("Failed to add font resource at %s: %v", targetPath, err)
		// Clean up on error
		logger.Debug("Cleaning up after failed font resource addition...")
		if removeErr := os.Remove(targetPath); removeErr != nil {
			logger.Error("Failed to clean up font file after resource addition failure: %v", removeErr)
		}
		return fmt.Errorf("failed to add font resource: %w", err)
	}
	logger.Debug("Font resource added successfully")

	// Add font to registry if machine scope
	if scope == MachineScope {
		logger.Debug("Adding font to registry...")
		if err := m.addFontToRegistry(fontName, targetPath); err != nil {
			logger.Error("Failed to add font to registry: %v", err)
			// Clean up on error
			logger.Debug("Cleaning up after failed registry addition...")
			RemoveFontResource(targetPath)
			os.Remove(targetPath)
			return fmt.Errorf("failed to add font to registry: %w", err)
		}
		logger.Debug("Font added to registry successfully")
	}

	// Notify other applications about the new font
	logger.Debug("Notifying system about font change...")
	if err := NotifyFontChange(); err != nil {
		logger.Error("Failed to notify font change: %v", err)
		// Clean up on error
		logger.Debug("Cleaning up after failed notification...")
		RemoveFontResource(targetPath)
		if scope == MachineScope {
			m.removeFontFromRegistry(fontName)
		}
		os.Remove(targetPath)
		return fmt.Errorf("failed to notify font change: %w", err)
	}
	logger.Debug("Font change notification sent successfully")

	logger.Info("Font installation completed successfully")
	return nil
}

// RemoveFont removes a font from the specified font directory
func (m *windowsFontManager) RemoveFont(fontName string, scope InstallationScope) error {
	logger := logging.GetLogger()
	logger.Debug("Starting font removal for: %s (scope: %s)", fontName, scope)

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
	logger.Debug("Target path: %s", fontPath)

	// Check if font exists
	if _, err := os.Stat(fontPath); os.IsNotExist(err) {
		logger.Error("Font not found at path: %s", fontPath)
		return fmt.Errorf("font not found: %s", fontName)
	}

	// Remove the font resource
	logger.Debug("Removing font resource...")
	if err := RemoveFontResource(fontPath); err != nil {
		// Check if the error is because the font isn't loaded as a resource
		// This is normal and shouldn't prevent font removal
		if strings.Contains(err.Error(), "error code: 0") || strings.Contains(err.Error(), "The operation completed successfully") {
			logger.Debug("Font resource not loaded, continuing with file removal")
		} else {
			logger.Error("Failed to remove font resource from path %s: %v", fontPath, err)
			return fmt.Errorf("failed to remove font resource: %w", err)
		}
	} else {
		logger.Debug("Font resource removed successfully")
	}

	// Remove from registry if machine scope
	if scope == MachineScope {
		logger.Debug("Removing font from registry...")
		if err := m.removeFontFromRegistry(fontName); err != nil {
			logger.Error("Failed to remove font from registry: %v", err)
			// Continue with file removal even if registry removal fails
		} else {
			logger.Debug("Font removed from registry successfully")
		}
	}

	// Delete the font file
	logger.Debug("Removing font file...")
	if err := os.Remove(fontPath); err != nil {
		logger.Error("Failed to remove font file at path %s: %v", fontPath, err)
		// Try to restore the font resource if file deletion fails
		if restoreErr := AddFontResource(fontPath); restoreErr != nil {
			logger.Error("Failed to restore font resource after file deletion failure: %v", restoreErr)
		}
		return fmt.Errorf("failed to remove font file: %w", err)
	}
	logger.Debug("Font file removed successfully")

	// Notify other applications about the font removal
	// Only send WM_FONTCHANGE to the desktop window to avoid hangs from full window enumeration.
	// Enumerating all windows can hang or be extremely slow on some systems.
	logger.Debug("Notifying system about font change...")
	if err := NotifyFontChange(); err != nil {
		logger.Error("Failed to notify system about font change: %v", err)
		return fmt.Errorf("failed to notify font change: %w", err)
	}
	logger.Debug("Font change notification sent successfully")

	logger.Info("Font removal completed successfully")
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
		return fmt.Errorf("failed to get file info: %w", err)
	}

	if stat.Size() < 1024 {
		return fmt.Errorf("font file is too small (minimum 1KB required)")
	}

	return nil
}

// addFontToRegistry adds a font to the Windows registry
func (m *windowsFontManager) addFontToRegistry(fontName, fontPath string) error {
	logger := logging.GetLogger()
	logger.Debug("Adding font to registry: %s (path: %s)", fontName, fontPath)

	// Open the registry key
	var key syscall.Handle
	ret, _, err := regCreateKeyEx.Call(
		uintptr(HKEY_LOCAL_MACHINE),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\Fonts"))),
		0,
		0,
		0,
		uintptr(KEY_WRITE),
		0,
		uintptr(unsafe.Pointer(&key)),
		0,
	)
	if ret != 0 {
		logger.Error("Failed to open registry key for font %s: %v", fontName, err)
		return fmt.Errorf("failed to open registry key: %w", err)
	}
	defer regCloseKey.Call(uintptr(key))
	logger.Debug("Registry key opened successfully")

	// Set the font value
	valueName := fontName + " (TrueType)"
	valueNamePtr, err := syscall.UTF16PtrFromString(valueName)
	if err != nil {
		logger.Error("Failed to convert value name to UTF16 for font %s: %v", fontName, err)
		return fmt.Errorf("failed to convert value name to UTF16: %w", err)
	}

	fontPathPtr, err := syscall.UTF16PtrFromString(fontPath)
	if err != nil {
		logger.Error("Failed to convert font path to UTF16 for font %s: %v", fontName, err)
		return fmt.Errorf("failed to convert font path to UTF16: %w", err)
	}

	ret, _, err = regSetValueEx.Call(
		uintptr(key),
		uintptr(unsafe.Pointer(valueNamePtr)),
		0,
		uintptr(REG_SZ),
		uintptr(unsafe.Pointer(fontPathPtr)),
		uintptr((len(fontPath)+1)*2),
	)
	if ret != 0 {
		logger.Error("Failed to set registry value for font %s: %v", fontName, err)
		return fmt.Errorf("failed to set registry value: %w", err)
	}

	logger.Debug("Font added to registry successfully: %s", fontName)
	return nil
}

// removeFontFromRegistry removes a font from the Windows registry
func (m *windowsFontManager) removeFontFromRegistry(fontName string) error {
	logger := logging.GetLogger()
	logger.Debug("Removing font from registry: %s", fontName)

	// Open the registry key
	var key syscall.Handle
	ret, _, err := regCreateKeyEx.Call(
		uintptr(HKEY_LOCAL_MACHINE),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\Fonts"))),
		0,
		0,
		0,
		uintptr(KEY_WRITE),
		0,
		uintptr(unsafe.Pointer(&key)),
		0,
	)
	if ret != 0 {
		return fmt.Errorf("failed to open registry key: %w", err)
	}
	defer regCloseKey.Call(uintptr(key))

	// Delete the font value using RegDeleteValueW
	valueName := fontName + " (TrueType)"
	valueNamePtr, err := syscall.UTF16PtrFromString(valueName)
	if err != nil {
		return fmt.Errorf("failed to convert value name to UTF16: %w", err)
	}

	regDeleteValue := syscall.NewLazyDLL("advapi32.dll").NewProc("RegDeleteValueW")
	ret, _, err = regDeleteValue.Call(
		uintptr(key),
		uintptr(unsafe.Pointer(valueNamePtr)),
	)
	if ret != 0 {
		return fmt.Errorf("failed to delete registry value: %w", err)
	}

	logger.Debug("Font removed from registry successfully")
	return nil
}

// FindWindowEx wraps the Windows FindWindowEx function
func FindWindowEx(hwndParent, hwndChildAfter uintptr, lpszClass, lpszWindow *uint16) uintptr {
	ret, _, _ := findWindowEx.Call(
		hwndParent,
		hwndChildAfter,
		uintptr(unsafe.Pointer(lpszClass)),
		uintptr(unsafe.Pointer(lpszWindow)),
	)
	return ret
}

// SendMessage wraps the Windows SendMessage function
func SendMessage(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	ret, _, _ := sendMessage.Call(
		hwnd,
		uintptr(msg),
		wParam,
		lParam,
	)
	return ret
}
