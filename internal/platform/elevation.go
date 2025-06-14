package platform

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// ElevationManager defines the interface for platform-specific elevation operations
type ElevationManager interface {
	// IsElevated checks if the current process is running with elevated privileges
	IsElevated() (bool, error)
	// RunAsElevated attempts to relaunch the process with elevated privileges
	RunAsElevated() error
	// GetElevationCommand returns the command to run the current process with elevation
	GetElevationCommand() (string, []string, error)
}

// NewElevationManager creates a new ElevationManager for the current platform
func NewElevationManager() (ElevationManager, error) {
	switch runtime.GOOS {
	case "windows":
		return &windowsElevationManager{}, nil
	case "linux", "darwin":
		return &unixElevationManager{}, nil
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// windowsElevationManager implements ElevationManager for Windows
type windowsElevationManager struct{}

// IsElevated checks if the current process is running with administrator privileges
func (m *windowsElevationManager) IsElevated() (bool, error) {
	return IsElevated()
}

// RunAsElevated attempts to relaunch the process with administrator privileges
func (m *windowsElevationManager) RunAsElevated() error {
	return RunAsElevated()
}

// GetElevationCommand returns the command to run the current process with UAC elevation
func (m *windowsElevationManager) GetElevationCommand() (string, []string, error) {
	// Get the executable path
	exe, err := os.Executable()
	if err != nil {
		return "", nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	// For Windows, we'll use the current executable with the same arguments
	// The elevation will be handled by the Windows API
	return exe, os.Args[1:], nil
}

// unixElevationManager implements ElevationManager for Linux and macOS
type unixElevationManager struct{}

// IsElevated checks if the current process is running with root privileges
func (m *unixElevationManager) IsElevated() (bool, error) {
	return os.Geteuid() == 0, nil
}

// RunAsElevated attempts to relaunch the process with root privileges
func (m *unixElevationManager) RunAsElevated() error {
	cmd, args, err := m.GetElevationCommand()
	if err != nil {
		return err
	}

	// Create the command
	execCmd := exec.Command(cmd, args...)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	// Run the command
	return execCmd.Run()
}

// GetElevationCommand returns the command to run the current process with sudo
func (m *unixElevationManager) GetElevationCommand() (string, []string, error) {
	// Get the executable path
	exe, err := os.Executable()
	if err != nil {
		return "", nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	// Build the command with sudo
	args := append([]string{exe}, os.Args[1:]...)
	return "sudo", args, nil
}
