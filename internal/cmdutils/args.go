// Package cmdutils provides CLI-specific utilities for command implementations.
//
// This file contains CLI argument parsing utilities.
// These functions parse and normalize command-line arguments for CLI commands.

package cmdutils

import (
	"strings"
)

// ParseFontNames parses comma-separated font names from command line arguments.
//
// It splits arguments by comma, trims whitespace, and filters out empty strings.
// This allows users to specify multiple fonts in a single argument: "font1,font2,font3"
//
// Example:
//
//	args := []string{"Roboto", "Open Sans,Noto Sans"}
//	result := ParseFontNames(args)
//	// result: []string{"Roboto", "Open Sans", "Noto Sans"}
func ParseFontNames(args []string) []string {
	var fontNames []string
	for _, arg := range args {
		// Split by comma and trim whitespace
		names := strings.Split(arg, ",")
		for _, name := range names {
			name = strings.TrimSpace(name)
			if name != "" {
				fontNames = append(fontNames, name)
			}
		}
	}
	return fontNames
}
