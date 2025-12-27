package cmdutils

import (
	"fmt"

	"fontget/internal/ui"
)

// PrintError prints an error message with standardized formatting.
func PrintError(message string) {
	fmt.Printf("%s\n", ui.ErrorText.Render(message))
}

// PrintErrorf prints a formatted error message with standardized formatting.
func PrintErrorf(format string, args ...interface{}) {
	PrintError(fmt.Sprintf(format, args...))
}

// PrintWarning prints a warning message with standardized formatting.
func PrintWarning(message string) {
	fmt.Printf("%s\n", ui.WarningText.Render(message))
}

// PrintWarningf prints a formatted warning message with standardized formatting.
func PrintWarningf(format string, args ...interface{}) {
	PrintWarning(fmt.Sprintf(format, args...))
}

// PrintInfo prints an info message with standardized formatting.
func PrintInfo(message string) {
	fmt.Printf("%s\n", ui.Text.Render(message))
}

// PrintInfof prints a formatted info message with standardized formatting.
func PrintInfof(format string, args ...interface{}) {
	PrintInfo(fmt.Sprintf(format, args...))
}

