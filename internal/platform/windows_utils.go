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
	postMessage        = user32.NewProc("PostMessageW")
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
		return fmt.Errorf("AddFontResource failed: %w", err)
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
		return fmt.Errorf("RemoveFontResource failed: %w", err)
	}

	return nil
}

// NotifyFontChange notifies the system about font changes
func NotifyFontChange() error {
	logger := logging.GetLogger()
	logger.Debug("Sending font change notification...")

	// Send WM_FONTCHANGE to all top-level windows
	hwnd := uintptr(0)
	for {
		hwnd = FindWindowEx(0, hwnd, nil, nil)
		if hwnd == 0 {
			break
		}
		ret := SendMessage(hwnd, WM_FONTCHANGE, 0, 0)
		if ret == 0 {
			// Log the error but continue with other windows
			logger.Warn("Failed to notify window %v about font change", hwnd)
		}
	}

	// Also send to the desktop window
	desktopHwnd, _, _ := getDesktopWindow.Call()
	if desktopHwnd != 0 {
		ret := SendMessage(desktopHwnd, WM_FONTCHANGE, 0, 0)
		if ret == 0 {
			logger.Warn("Failed to notify desktop window about font change")
		}
	}

	// Force a refresh of the font cache
	if err := refreshFontCache(); err != nil {
		logger.Warn("Failed to refresh font cache: %v", err)
	}

	logger.Debug("Font change notification sent successfully")
	return nil
}

// refreshFontCache forces a refresh of the Windows font cache
func refreshFontCache() error {
	// Send WM_FONTCHANGE to the desktop window
	desktopHwnd, _, _ := getDesktopWindow.Call()
	if desktopHwnd != 0 {
		ret := SendMessage(desktopHwnd, WM_FONTCHANGE, 0, 0)
		if ret == 0 {
			return fmt.Errorf("failed to refresh font cache")
		}
	}
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

// IsProcessElevated checks if a specific process is running with administrator privileges
func IsProcessElevated(pid int) (bool, error) {
	// Open the process
	handle, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return false, fmt.Errorf("failed to open process: %w", err)
	}
	defer syscall.CloseHandle(handle)

	// Get the process token
	var token syscall.Token
	err = syscall.OpenProcessToken(handle, syscall.TOKEN_QUERY, &token)
	if err != nil {
		return false, fmt.Errorf("failed to open process token: %w", err)
	}
	defer token.Close()

	// Get the elevation information
	var elevation uint32
	var size uint32
	err = syscall.GetTokenInformation(token, syscall.TokenElevation, (*byte)(unsafe.Pointer(&elevation)), uint32(unsafe.Sizeof(elevation)), &size)
	if err != nil {
		return false, fmt.Errorf("failed to get token elevation information: %w", err)
	}

	return elevation != 0, nil
}
