package cmd

import (
	"fmt"

	"github.com/fatih/color"
)

// Standard color functions for consistent error handling
var (
	ErrorColor   = color.New(color.FgRed, color.Bold)
	WarningColor = color.New(color.FgYellow)
	SuccessColor = color.New(color.FgGreen, color.Bold)
	InfoColor    = color.New(color.FgCyan)
	WhiteColor   = color.New(color.FgWhite)
)

// PrintValidationError prints a validation error with consistent formatting
func PrintValidationError(message string) {
	fmt.Printf("%s %s\n", ErrorColor.Sprint("Error:"), message)
}

// PrintOperationalError prints an operational error without "Error:" prefix
func PrintOperationalError(message string) {
	fmt.Printf("%s\n", ErrorColor.Sprint(message))
}

// PrintWarning prints a warning message
func PrintWarning(message string) {
	fmt.Printf("%s %s\n", WarningColor.Sprint("Warning:"), message)
}

// PrintSuccess prints a success message
func PrintSuccess(message string) {
	fmt.Printf("%s\n", SuccessColor.Sprint(message))
}

// PrintInfo prints an informational message
func PrintInfo(message string) {
	fmt.Printf("%s\n", InfoColor.Sprint(message))
}

// PrintErrorWithHint prints an error with a helpful hint
func PrintErrorWithHint(errorMsg, hint string) {
	fmt.Printf("%s %s\n", ErrorColor.Sprint("Error:"), errorMsg)
	fmt.Printf("   %s\n", WarningColor.Sprint(hint))
}

// PrintSuccessWithDetails prints a success message with additional details
func PrintSuccessWithDetails(successMsg string, details map[string]string) {
	fmt.Printf("%s\n", SuccessColor.Sprint(successMsg))
	for key, value := range details {
		fmt.Printf("   %s %s\n", InfoColor.Sprint(key+":"), value)
	}
}
