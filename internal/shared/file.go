package shared

import (
	"fmt"
	"strings"
)

// FormatFileSize formats bytes into human-readable format (KB, MB).
// This is a general utility function that can be used anywhere file sizes need to be displayed.
func FormatFileSize(bytes int64) string {
	if bytes == 0 {
		return ""
	}

	const (
		KB = 1024
		MB = KB * 1024
	)

	if bytes >= MB {
		return fmt.Sprintf("%.1fMB", float64(bytes)/float64(MB))
	} else if bytes >= KB {
		return fmt.Sprintf("%.0fKB", float64(bytes)/float64(KB))
	}
	return fmt.Sprintf("%dB", bytes)
}

// SanitizeForZipPath sanitizes a string for use as a path component in a zip archive.
func SanitizeForZipPath(name string) string {
	// Replace invalid characters with underscores
	invalidChars := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
	result := name
	for _, char := range invalidChars {
		result = strings.ReplaceAll(result, char, "_")
	}
	// Remove leading/trailing spaces and dots (Windows restrictions)
	result = strings.Trim(result, " .")
	// Replace multiple consecutive underscores with a single one
	for strings.Contains(result, "__") {
		result = strings.ReplaceAll(result, "__", "_")
	}
	// If empty after sanitization, use a default name
	if result == "" {
		result = "unknown"
	}
	return result
}

// TruncateString truncates a string to the specified length.
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

