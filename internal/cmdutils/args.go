package cmdutils

import (
	"strings"
)

// ParseFontNames parses comma-separated font names from command line arguments
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
