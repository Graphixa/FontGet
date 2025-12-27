// Package cmdutils provides CLI-specific utilities for command implementations.
//
// This file contains CLI-specific file utility functions.
// These are simple wrappers used primarily by CLI commands.

package cmdutils

import (
	"os"
)

// CheckFileExists checks if a file exists at the given path.
// Returns true if the file exists, false if it doesn't exist.
// Returns an error if there was a problem checking the file (other than file not found).
func CheckFileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	// Some other error occurred
	return false, err
}
