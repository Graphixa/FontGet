package shared

import (
	"fmt"
	"os"

	"github.com/charmbracelet/x/term"
)

// GetTerminalWidth returns the current terminal width, or a default if unavailable.
func GetTerminalWidth() int {
	width, _, err := term.GetSize(os.Stdout.Fd())
	if err == nil && width > 0 {
		return width
	}

	if cols := os.Getenv("COLUMNS"); cols != "" {
		var w int
		if _, err := fmt.Sscanf(cols, "%d", &w); err == nil && w > 0 {
			return w
		}
	}

	return 80
}
