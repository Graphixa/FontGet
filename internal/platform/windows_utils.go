//go:build windows
// +build windows

package platform

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	advapi32 = syscall.NewLazyDLL("advapi32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")

	isUserAnAdmin = shell32.NewProc("IsUserAnAdmin")
)

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

	// Convert to UTF-16 for Windows API
	exePtr, err := syscall.UTF16PtrFromString(exe)
	if err != nil {
		return fmt.Errorf("failed to convert path to UTF-16: %w", err)
	}

	// Use ShellExecute to trigger UAC prompt
	ret, _, err := shell32.NewProc("ShellExecuteW").Call(
		0, // hwnd
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("runas"))), // lpOperation
		uintptr(unsafe.Pointer(exePtr)),                            // lpFile
		0,                                                          // lpParameters
		0,                                                          // lpDirectory
		syscall.SW_NORMAL,                                          // nShowCmd
	)
	if ret <= 32 {
		return fmt.Errorf("failed to launch elevated process: %w", err)
	}

	return nil
}
