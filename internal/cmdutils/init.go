package cmdutils

import (
	"fmt"

	"fontget/internal/config"
	"fontget/internal/output"
	"fontget/internal/platform"
)

// Logger is a minimal interface for logging errors.
type Logger interface {
	Error(format string, args ...interface{})
}

// EnsureManifestInitialized ensures the manifest exists with standardized error handling.
// This is used by all commands that require the font repository to be initialized.
func EnsureManifestInitialized(getLogger func() Logger) error {
	if err := config.EnsureManifestExists(); err != nil {
		if logger := getLogger(); logger != nil {
			logger.Error("Failed to ensure manifest exists: %v", err)
		}
		output.GetVerbose().Error("%v", err)
		output.GetDebug().Error("config.EnsureManifestExists() failed: %v", err)
		// Use %w to preserve error chain for error.Is() and errors.Unwrap()
		return fmt.Errorf("unable to load font repository: %w", err)
	}
	return nil
}

// CreateFontManager creates a font manager with standardized error handling.
// This is used by commands that need to interact with system fonts.
func CreateFontManager(getLogger func() Logger) (platform.FontManager, error) {
	fm, err := platform.NewFontManager()
	if err != nil {
		if logger := getLogger(); logger != nil {
			logger.Error("Failed to create font manager: %v", err)
		}
		output.GetVerbose().Error("%v", err)
		output.GetDebug().Error("platform.NewFontManager() failed: %v", err)
		// Use %w to preserve error chain for error.Is() and errors.Unwrap()
		return nil, fmt.Errorf("unable to access system fonts: %w", err)
	}
	return fm, nil
}
