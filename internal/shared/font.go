package shared

import (
	"fmt"
	"path/filepath"
	"strings"
)

// camelCaseConversionThreshold determines when to convert camelCase to spaced format.
// Only converts if the capital letter is within the first 60% of the word.
// This catches "OpenSans" (capital at pos 4 of 9) but not "ABeeZee" (multiple capitals).
const camelCaseConversionThreshold = 0.6

// convertCamelCaseToSpaced converts camelCase to spaced format (e.g., RobotoMono -> Roboto Mono)
func convertCamelCaseToSpaced(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, ' ')
		}
		result = append(result, r)
	}
	return string(result)
}

// GetDisplayNameFromFilename builds a display name purely from filename (no metadata required).
//
// It replaces hyphens with spaces and converts camelCase to spaced format for simple cases
// (e.g., "OpenSans" -> "Open Sans"). Complex names like "ABeeZee" are preserved as-is.
// This function is used as a fallback when font metadata extraction fails.
//
// The function uses a threshold-based approach to determine if camelCase conversion should
// be applied, only converting when the capital letter is in the first 60% of the word.
func GetDisplayNameFromFilename(filename string) string {
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// Split on hyphen to separate base font name from variant
	if strings.Contains(name, "-") {
		parts := strings.Split(name, "-")
		if len(parts) >= 2 {
			baseFontName := parts[0]
			// Only convert camelCase for simple two-word cases (e.g., "OpenSans" -> "Open Sans")
			// But preserve complex names like "ABeeZee" and "RobotoMono"
			if shouldConvertCamelCase(baseFontName) {
				baseFontName = convertCamelCaseToSpaced(baseFontName)
			}
			variantPart := strings.Join(parts[1:], " ")
			return fmt.Sprintf("%s %s", baseFontName, variantPart)
		}
	}

	// No hyphen, check if we should convert camelCase
	if shouldConvertCamelCase(name) {
		return convertCamelCaseToSpaced(name)
	}
	return name
}

// shouldConvertCamelCase determines if a font name should have camelCase conversion.
// Only converts simple two-word cases like "OpenSans", not complex names like "ABeeZee" or "RobotoMono".
// Also preserves names with underscores.
func shouldConvertCamelCase(name string) bool {
	// Don't convert if name contains underscores (preserve them)
	if strings.Contains(name, "_") {
		return false
	}
	// Count capital letters (excluding the first character) and find position of first one
	// We skip the first character since it's expected to be capitalized in camelCase
	capCount := 0
	firstCapPos := -1
	for i := 1; i < len(name); i++ {
		if name[i] >= 'A' && name[i] <= 'Z' {
			capCount++
			// Track the position of the first capital letter (after the first character)
			if firstCapPos == -1 {
				firstCapPos = i
			}
		}
	}
	// Only convert if there's exactly one capital and it's in the first 60% of the word
	// This catches "OpenSans" (1 capital at pos 4 of 9) and "RobotoMono" (1 capital at pos 6 of 10)
	// but not "ABeeZee" (2 capitals: B, Z)
	// The threshold prevents converting names where the capital is too far into the word
	if capCount == 1 && firstCapPos > 0 {
		threshold := int(float64(len(name)) * camelCaseConversionThreshold)
		return firstCapPos <= threshold
	}
	return false
}
