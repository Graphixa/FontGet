// Package cmdutils provides CLI-specific utilities for command implementations.
//
// This file contains CLI wrappers for repository operations.
// These functions wrap internal/repo functions with CLI-specific logging, error formatting,
// and verbose/debug output integration.

package cmdutils

import (
	"fmt"

	"fontget/internal/output"
	"fontget/internal/repo"
)

// GetRepository gets the font repository with optional refresh.
// If refresh is true, forces a refresh of the font manifest.
// Returns standardized error handling for repository initialization.
//
// logger can be nil (for testing or when logging is not needed).
//
// NOTE: This function is tested via integration tests (see cmd/integration_test.go)
// because it depends on package-level repo functions that are difficult to mock.
func GetRepository(refresh bool, logger Logger) (*repo.Repository, error) {
	var r *repo.Repository
	var err error

	if refresh {
		// Force refresh of font manifest before use
		output.GetVerbose().Info("Forcing refresh of font manifest")
		output.GetDebug().State("Using GetRepositoryWithRefresh() to force source updates")
		r, err = repo.GetRepositoryWithRefresh()
	} else {
		output.GetVerbose().Info("Using cached font manifest")
		output.GetDebug().State("Using GetRepository() with cached sources")
		r, err = repo.GetRepository()
	}

	if err != nil {
		if logger != nil {
			logger.Error("Failed to get repository: %v", err)
		}
		output.GetVerbose().Error("%v", err)
		output.GetDebug().Error("repo.GetRepository() failed: %v", err)
		return nil, fmt.Errorf("unable to load font repository: %w", err)
	}

	return r, nil
}

// MatchInstalledFontsToRepository matches installed fonts to repository entries.
// This is reusable for list and export commands.
//
// logger can be nil (for testing or when logging is not needed).
// isCriticalSystemFont is a function that checks if a font is a critical system font.
//
// NOTE: This function is tested via integration tests (see cmd/integration_test.go)
// because it depends on package-level repo functions that are difficult to mock.
func MatchInstalledFontsToRepository(familyNames []string, logger Logger, isCriticalSystemFont func(string) bool) (map[string]*repo.InstalledFontMatch, error) {
	output.GetVerbose().Info("Matching installed fonts to repository...")
	output.GetDebug().State("Total installed font families to match: %d", len(familyNames))
	output.GetDebug().State("Calling repo.MatchAllInstalledFonts(familyCount=%d)", len(familyNames))
	matches, err := repo.MatchAllInstalledFonts(familyNames, isCriticalSystemFont)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to match fonts to repository: %v", err)
		}
		output.GetVerbose().Error("%v", err)
		output.GetDebug().Error("repo.MatchAllInstalledFonts() failed: %v", err)
		return nil, err
	}
	output.GetVerbose().Info("Matched %d font families to repository entries", len(matches))
	return matches, nil
}
