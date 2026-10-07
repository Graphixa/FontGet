package fontkey

import "strings"

// Key normalizes a font family or style string for identity matching:
// trim surrounding whitespace, lowercase, then remove spaces, hyphens and underscores.
func Key(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}
