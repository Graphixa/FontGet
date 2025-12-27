package shared

import (
	"fmt"
	"path/filepath"
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

// ValidatePathCharacters validates that a path doesn't contain invalid characters.
// This function checks for invalid filename characters, control characters, reserved names,
// and trailing spaces/dots according to Windows file system rules.
// Returns a PathValidationError if validation fails.
func ValidatePathCharacters(path string) error {
	if path == "" {
		return &PathValidationError{
			Path:   path,
			Reason: "path cannot be empty",
		}
	}

	// Get just the filename part (not the directory path)
	// Path separators are valid, we only need to check the filename
	baseName := filepath.Base(path)

	// Windows invalid filename characters: < > : " | ? * and control characters (0x00-0x1F)
	// Note: / and \ are valid path separators, so we only check the filename part
	invalidChars := []rune{'<', '>', ':', '"', '|', '?', '*'}

	// Check for invalid characters in filename
	for _, char := range invalidChars {
		if strings.ContainsRune(baseName, char) {
			return &PathValidationError{
				Path:    path,
				Reason:  "contains invalid character",
				Details: fmt.Sprintf("character '%c' is not allowed in filenames", char),
			}
		}
	}

	// Check for control characters (0x00-0x1F) except tab, newline, carriage return
	for _, r := range path {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' {
			return &PathValidationError{
				Path:    path,
				Reason:  "contains invalid control character",
				Details: fmt.Sprintf("control character 0x%02X is not allowed", r),
			}
		}
	}

	// Check for reserved Windows names (CON, PRN, AUX, NUL, COM1-9, LPT1-9)
	baseNameUpper := strings.ToUpper(strings.TrimSuffix(baseName, filepath.Ext(baseName)))
	reservedNames := []string{"CON", "PRN", "AUX", "NUL"}
	for i := 1; i <= 9; i++ {
		reservedNames = append(reservedNames, fmt.Sprintf("COM%d", i), fmt.Sprintf("LPT%d", i))
	}
	for _, reserved := range reservedNames {
		if baseNameUpper == reserved {
			return &PathValidationError{
				Path:    path,
				Reason:  "uses reserved Windows name",
				Details: fmt.Sprintf("'%s' is a reserved name and cannot be used", reserved),
			}
		}
	}

	// Check for trailing spaces or dots (Windows restriction)
	trimmed := strings.TrimRight(baseName, " .")
	if trimmed != baseName {
		return &PathValidationError{
			Path:    path,
			Reason:  "filename ends with invalid characters",
			Details: "filenames cannot end with spaces or dots on Windows",
		}
	}

	return nil
}
