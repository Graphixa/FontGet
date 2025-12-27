// Package cmdutils provides CLI-specific utilities for command implementations.
//
// This file contains CLI-specific path validation for export/backup operations.
// It handles CLI concerns like overwrite confirmation, default filenames, and directory detection.
// Uses shared.ValidatePathCharacters() for character validation.
//
// This is CLI-specific because it handles user-facing concerns (overwrite prompts, default paths).

package cmdutils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fontget/internal/shared"
)

// ValidateOutputPath validates and normalizes an output path for export/backup operations.
// It handles directory vs file detection, extension validation, and file existence checks.
//
// Parameters:
//   - path: The output path provided by the user (can be empty, a directory, or a file)
//   - defaultFilename: The default filename to use if path is empty or a directory (e.g., "fontget-export-2024-01-01.json")
//   - requiredExt: The required file extension (e.g., ".json" or ".zip")
//   - force: Whether to allow overwriting existing files without confirmation
//
// Returns:
//   - normalizedPath: The absolute, normalized path to the output file
//   - needsConfirm: Whether the file exists and needs overwrite confirmation (if force is false)
//   - err: Error if validation fails
func ValidateOutputPath(path string, defaultFilename string, requiredExt string, force bool) (normalizedPath string, needsConfirm bool, err error) {
	// If no path provided, use default filename in current directory
	if path == "" {
		path = defaultFilename
	}

	// Normalize path separators
	path = filepath.Clean(path)

	// Check if it's a directory (ends with separator or exists as directory)
	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		// It's a directory, use default filename in that directory
		path = filepath.Join(path, defaultFilename)
	} else if err == nil && !info.IsDir() {
		// Path exists and is a file - check if it has the required extension
		if !strings.HasSuffix(strings.ToLower(path), strings.ToLower(requiredExt)) {
			return "", false, fmt.Errorf("output path exists and is not a %s file: %s", requiredExt, path)
		}
		// File exists - will check and prompt later after getting absolute path
	} else if os.IsNotExist(err) {
		// Path doesn't exist - check if parent directory exists
		parentDir := filepath.Dir(path)
		if parentDir != "." && parentDir != path {
			parentInfo, err := os.Stat(parentDir)
			if err != nil {
				// Parent doesn't exist - check if we can create it
				// For safety, only allow creating one level deep
				if !strings.HasSuffix(strings.ToLower(path), strings.ToLower(requiredExt)) {
					return "", false, fmt.Errorf("output path must be a %s file: %s", requiredExt, path)
				}
				// Will create parent directory later
			} else if !parentInfo.IsDir() {
				return "", false, fmt.Errorf("parent path exists but is not a directory: %s", parentDir)
			}
		}

		// Ensure it has the required extension
		if !strings.HasSuffix(strings.ToLower(path), strings.ToLower(requiredExt)) {
			path = path + requiredExt
		}
	}

	// Validate path characters
	if err := shared.ValidatePathCharacters(path); err != nil {
		return "", false, err
	}

	// Final validation: ensure it's an absolute or relative path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", false, fmt.Errorf("invalid output path: %w", err)
	}

	// Guard rail: prevent writing to system directories
	systemDirs := []string{
		filepath.Join(os.Getenv("SystemRoot"), "Fonts"),
		"/System/Library/Fonts",
		"/usr/share/fonts",
		"/usr/local/share/fonts",
	}
	for _, sysDir := range systemDirs {
		if sysDir != "" && strings.HasPrefix(strings.ToLower(absPath), strings.ToLower(sysDir)) {
			return "", false, fmt.Errorf("cannot write output to system font directory: %s", absPath)
		}
	}

	// Check if the final file path already exists
	needsConfirm = false
	if _, err := os.Stat(absPath); err == nil {
		if !force {
			needsConfirm = true
		}
	}

	return absPath, needsConfirm, nil
}
