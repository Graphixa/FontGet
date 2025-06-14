package platform

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// TempDirName is the name of our temporary directory
	TempDirName = "Fontget"
	// TempFontsDir is the name of the fonts subdirectory
	TempFontsDir = "fonts"
)

// GetTempDir returns the platform-specific temp directory path for Fontget
func GetTempDir() (string, error) {
	// Get the system's temp directory
	tempDir := os.TempDir()
	if tempDir == "" {
		return "", fmt.Errorf("failed to get system temp directory")
	}

	// Create our temp directory path
	fontgetTempDir := filepath.Join(tempDir, TempDirName)
	if err := os.MkdirAll(fontgetTempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	return fontgetTempDir, nil
}

// GetTempFontsDir returns the path to the temporary fonts directory
func GetTempFontsDir() (string, error) {
	// Get the base temp directory
	tempDir, err := GetTempDir()
	if err != nil {
		return "", err
	}

	// Create the fonts subdirectory
	fontsDir := filepath.Join(tempDir, TempFontsDir)
	if err := os.MkdirAll(fontsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create fonts directory: %w", err)
	}

	return fontsDir, nil
}

// CleanupTempDir removes all files from the temp directory
func CleanupTempDir() error {
	tempDir, err := GetTempDir()
	if err != nil {
		return err
	}

	// Remove the entire temp directory and its contents
	if err := os.RemoveAll(tempDir); err != nil {
		return fmt.Errorf("failed to cleanup temp directory: %w", err)
	}

	return nil
}

// CleanupTempFontsDir removes all files from the temp fonts directory
func CleanupTempFontsDir() error {
	fontsDir, err := GetTempFontsDir()
	if err != nil {
		return err
	}

	// Remove the fonts directory and its contents
	if err := os.RemoveAll(fontsDir); err != nil {
		return fmt.Errorf("failed to cleanup fonts directory: %w", err)
	}

	return nil
}
