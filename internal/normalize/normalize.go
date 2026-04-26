package normalize

import "strings"

// FontKey normalizes a font family/name string for stable comparisons.
// It lowercases and removes common separators (spaces, hyphens, underscores).
func FontKey(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}

// BaseFamilyName removes common suffix patterns from an installed font family name so it can
// be matched to repository IDs (notably Nerd Fonts). If no known patterns apply, it returns the
// original string.
func BaseFamilyName(familyName string) string {
	lower := strings.ToLower(familyName)

	nerdPatterns := []string{
		" nerd font",
		"nerdfont",
		" nerd",
	}

	baseName := familyName
	for _, pattern := range nerdPatterns {
		if idx := strings.Index(lower, pattern); idx > 0 {
			baseName = strings.TrimSpace(familyName[:idx])
			break
		}
	}

	variantSuffixes := []string{
		"NL",           // No Ligatures (e.g., "JetBrainsMonoNL" -> "JetBrainsMono")
		"Propo",        // Proportional variant
		"Proportional", // Proportional variant (full word)
	}

	baseLower := strings.ToLower(baseName)
	for _, suffix := range variantSuffixes {
		suffixLower := strings.ToLower(suffix)
		if strings.HasSuffix(baseLower, suffixLower) {
			baseName = strings.TrimSpace(baseName[:len(baseName)-len(suffix)])
			baseLower = strings.ToLower(baseName)
		}
	}

	return baseName
}
