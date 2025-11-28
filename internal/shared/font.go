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

// FormatFontNameWithVariant formats a font name with its variant for display.
func FormatFontNameWithVariant(fontName, variant string) string {
	if variant == "" || strings.EqualFold(variant, "regular") {
		return fontName
	}
	cleanVariant := strings.ReplaceAll(variant, " ", "")
	cleanVariant = strings.ReplaceAll(cleanVariant, "-", "")
	cleanVariant = strings.ReplaceAll(cleanVariant, "_", "")

	cleanFontName := strings.ReplaceAll(fontName, " ", "")
	cleanFontName = strings.ReplaceAll(cleanFontName, "-", "")
	cleanFontName = strings.ReplaceAll(cleanFontName, "_", "")

	lv := strings.ToLower(cleanVariant)
	lfn := strings.ToLower(cleanFontName)
	if strings.HasPrefix(lv, lfn) {
		cleanVariant = cleanVariant[len(cleanFontName):]
	} else if strings.HasPrefix(strings.ToLower(cleanVariant), strings.ToLower(fontName)) {
		cleanVariant = cleanVariant[len(fontName):]
	}

	if len(cleanVariant) > 0 {
		cleanVariant = strings.ToUpper(cleanVariant[:1]) + cleanVariant[1:]
	}

	if cleanVariant != "" && !strings.EqualFold(cleanVariant, "Regular") {
		return fontName + " " + cleanVariant
	}
	return fontName
}

// GetFontDisplayName returns a human-friendly display name, handling Nerd Fonts and variants.
func GetFontDisplayName(fontPath, fontName, variant string) string {
	baseName := filepath.Base(fontPath)
	ext := filepath.Ext(baseName)
	fileName := strings.TrimSuffix(baseName, ext)

	if strings.Contains(fileName, "NerdFont") || strings.Contains(fileName, "Nerd") {
		displayName := strings.ReplaceAll(fileName, "NerdFont", " Nerd Font ")
		displayName = strings.ReplaceAll(displayName, "-", " ")
		for strings.Contains(displayName, "  ") {
			displayName = strings.ReplaceAll(displayName, "  ", " ")
		}
		return strings.TrimSpace(displayName)
	}
	return FormatFontNameWithVariant(fontName, variant)
}

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

// GetFontFamilyNameFromFilename extracts just the font family name (without variant) from a filename.
// This is useful for getting the font name from installed font files.
// e.g., "ABeeZee-Italic.ttf" -> "ABeeZee", "RobotoMono-Bold.ttf" -> "RobotoMono"
func GetFontFamilyNameFromFilename(filename string) string {
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// Split on hyphen to separate family from variant
	if strings.Contains(name, "-") {
		parts := strings.Split(name, "-")
		if len(parts) >= 2 {
			// Return the base font name (first part) without spacing conversion
			// This matches the repository format (e.g., "ABeeZee" not "A Bee Zee")
			return parts[0]
		}
	}
	return name
}
