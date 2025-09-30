//go:build windows
// +build windows

package platform

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"

	"fontget/internal/logging"
)

const (
	HWND_BROADCAST     = 0xFFFF
	HKEY_LOCAL_MACHINE = 0x80000002
	KEY_WRITE          = 0x20006
	REG_SZ             = 1
)

var (
	advapi32 = syscall.NewLazyDLL("advapi32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	user32   = syscall.NewLazyDLL("user32.dll")

	isUserAnAdmin      = shell32.NewProc("IsUserAnAdmin")
	addFontResource    = gdi32.NewProc("AddFontResourceW")
	removeFontResource = gdi32.NewProc("RemoveFontResourceW")
	regCreateKeyEx     = advapi32.NewProc("RegCreateKeyExW")
	regSetValueEx      = advapi32.NewProc("RegSetValueExW")
	regCloseKey        = advapi32.NewProc("RegCloseKey")
	getDesktopWindow   = user32.NewProc("GetDesktopWindow")
)

// AddFontResource adds a font resource to the system
func AddFontResource(fontPath string) error {
	logger := logging.GetLogger()
	logger.Debug("Adding font resource for: %s", fontPath)
	pathPtr, err := syscall.UTF16PtrFromString(fontPath)
	if err != nil {
		return fmt.Errorf("failed to convert path to UTF16: %w", err)
	}

	// Add the font resource
	ret, _, err := addFontResource.Call(uintptr(unsafe.Pointer(pathPtr)))
	if ret == 0 {
		// Get the actual Windows error code
		errCode := syscall.GetLastError()
		return fmt.Errorf("AddFontResource failed (error code: %d): %w", errCode, err)
	}

	return nil
}

// RemoveFontResource removes a font resource from the system
func RemoveFontResource(fontPath string) error {
	logger := logging.GetLogger()
	logger.Debug("Removing font resource for: %s", fontPath)
	pathPtr, err := syscall.UTF16PtrFromString(fontPath)
	if err != nil {
		return fmt.Errorf("failed to convert path to UTF16: %w", err)
	}

	// Remove the font resource
	ret, _, err := removeFontResource.Call(uintptr(unsafe.Pointer(pathPtr)))
	if ret == 0 {
		// Get the actual Windows error code
		errCode := syscall.GetLastError()
		return fmt.Errorf("RemoveFontResource failed (error code: %d): %w", errCode, err)
	}

	return nil
}

// NotifyFontChange notifies the system about font changes
func NotifyFontChange() error {
	logger := logging.GetLogger()
	logger.Debug("Sending font change notification...")

	// Only send WM_FONTCHANGE to the desktop window for safety.
	// Enumerating all windows can hang or be extremely slow on some systems.
	desktopHwnd, _, _ := getDesktopWindow.Call()
	if desktopHwnd != 0 {
		ret := SendMessage(desktopHwnd, WM_FONTCHANGE, 0, 0)
		if ret == 0 {
			logger.Warn("Failed to notify desktop window about font change")
		}
	}

	logger.Debug("Font change notification sent successfully")
	return nil
}

// IsElevated checks if the current process is running with administrator privileges
func IsElevated() (bool, error) {
	ret, _, err := isUserAnAdmin.Call()
	if err != syscall.Errno(0) {
		return false, fmt.Errorf("failed to check elevation status: %w", err)
	}
	return ret != 0, nil
}

// RunAsElevated attempts to relaunch the process with administrator privileges
func RunAsElevated() error {
	// Get the executable path
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Get the current command line arguments
	args := os.Args[1:]

	// Convert to UTF-16 for Windows API
	exePtr, err := syscall.UTF16PtrFromString(exe)
	if err != nil {
		return fmt.Errorf("failed to convert path to UTF-16: %w", err)
	}

	// Build the command line arguments string
	cmdLine := ""
	for i, arg := range args {
		if i > 0 {
			cmdLine += " "
		}
		// Quote arguments that contain spaces
		if strings.Contains(arg, " ") {
			cmdLine += fmt.Sprintf(`"%s"`, arg)
		} else {
			cmdLine += arg
		}
	}

	cmdLinePtr, err := syscall.UTF16PtrFromString(cmdLine)
	if err != nil {
		return fmt.Errorf("failed to convert command line to UTF-16: %w", err)
	}

	// Convert working directory to UTF-16
	cwdPtr, err := syscall.UTF16PtrFromString(cwd)
	if err != nil {
		return fmt.Errorf("failed to convert working directory to UTF-16: %w", err)
	}

	// Use ShellExecute to trigger UAC prompt
	ret, _, err := shell32.NewProc("ShellExecuteW").Call(
		0, // hwnd
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("runas"))), // lpOperation
		uintptr(unsafe.Pointer(exePtr)),                            // lpFile
		uintptr(unsafe.Pointer(cmdLinePtr)),                        // lpParameters
		uintptr(unsafe.Pointer(cwdPtr)),                            // lpDirectory
		syscall.SW_NORMAL,                                          // nShowCmd
	)

	// ShellExecute returns a value greater than 32 on success
	if ret <= 32 {
		// Get the last error code
		errCode := syscall.GetLastError()
		return fmt.Errorf("failed to launch elevated process (error code: %d)", errCode)
	}

	return nil
}

// CreateHiddenDirectory creates a directory and sets it as hidden on Windows
func CreateHiddenDirectory(path string, perm os.FileMode) error {
	// First create the directory normally
	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Set the hidden attribute on Windows
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return fmt.Errorf("failed to convert path to UTF16: %w", err)
	}

	// Use SetFileAttributes to make the directory hidden
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setFileAttributes := kernel32.NewProc("SetFileAttributesW")

	// FILE_ATTRIBUTE_HIDDEN = 0x2
	ret, _, err := setFileAttributes.Call(uintptr(unsafe.Pointer(pathPtr)), 0x2)
	if ret == 0 {
		// Get the actual Windows error code
		errCode := syscall.GetLastError()
		return fmt.Errorf("SetFileAttributes failed (error code: %d): %w", errCode, err)
	}

	return nil
}
