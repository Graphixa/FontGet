package output

import (
	"fmt"
	"fontget/internal/ui"
)

// VerboseLogger provides user-friendly verbose output functionality
type VerboseLogger struct{}

var (
	verboseInstance *VerboseLogger
	verboseChecker  func() bool
	// debugChecker is defined in debug.go (same package) and accessed here
)

// GetVerbose returns a singleton verbose logger instance
func GetVerbose() *VerboseLogger {
	if verboseInstance == nil {
		verboseInstance = &VerboseLogger{}
	}
	return verboseInstance
}

// SetVerboseChecker allows the cmd package to inject the verbose flag checking function
// This avoids circular imports while maintaining clean architecture
func SetVerboseChecker(checker func() bool) {
	verboseChecker = checker
}

// isVerboseEnabled checks if verbose output should be displayed
// Verbose output should only show when --verbose is set AND --debug is NOT set
// This prevents verbose messages from appearing in debug mode (per logging guidelines)
func isVerboseEnabled() bool {
	if verboseChecker == nil {
		return false // Default to false if checker not set
	}
	// Access debugChecker from debug.go (same package)
	// We need to check if debug is enabled via the debugChecker function
	// Since debugChecker is in the same package, we can access it directly
	// But we need to check if it's been set first
	if debugChecker == nil {
		// If debug checker isn't set yet, just check verbose
		return verboseChecker()
	}
	return verboseChecker() && !debugChecker() // Verbose only when verbose=true AND debug=false
}

// Info displays verbose information to users when --verbose flag is enabled
// Shows operational details like file paths, parameters, and progress
func (v *VerboseLogger) Info(format string, args ...interface{}) {
	if isVerboseEnabled() {
		msg := fmt.Sprintf(format, args...)
		fmt.Printf("%s %s\n", ui.InfoText.Render("[INFO]"), msg)
	}
}

// Warning displays verbose warnings to users when --verbose flag is enabled
// Shows non-critical issues that users should be aware of
func (v *VerboseLogger) Warning(format string, args ...interface{}) {
	if isVerboseEnabled() {
		msg := fmt.Sprintf(format, args...)
		fmt.Printf("%s %s\n", ui.WarningText.Render("[WARNING]"), msg)
	}
}

// Error displays verbose error details to users when --verbose flag is enabled
// Shows detailed error information without timestamps (user-friendly)
func (v *VerboseLogger) Error(format string, args ...interface{}) {
	if isVerboseEnabled() {
		msg := fmt.Sprintf(format, args...)
		fmt.Printf("%s %s\n", ui.ErrorText.Render("[ERROR]"), msg)
	}
}

// Success displays verbose success messages to users when --verbose flag is enabled
// Shows detailed success information for completed operations
func (v *VerboseLogger) Success(format string, args ...interface{}) {
	if isVerboseEnabled() {
		msg := fmt.Sprintf(format, args...)
		fmt.Printf("%s %s\n", ui.SuccessText.Render("[SUCCESS]"), msg)
	}
}

// Detail displays additional detailed information with custom prefix
// Useful for showing file paths, parameters, or step-by-step progress
func (v *VerboseLogger) Detail(prefix, format string, args ...interface{}) {
	if isVerboseEnabled() {
		msg := fmt.Sprintf(format, args...)
		fmt.Printf("    %s %s\n", ui.Text.Render(prefix+":"), msg)
	}
}

// Print displays plain text output without any prefix or styling
// Useful for simple formatted output like variant lists
func (v *VerboseLogger) Print(format string, args ...interface{}) {
	if isVerboseEnabled() {
		fmt.Printf(format, args...)
	}
}

// EndSection adds a blank line after a verbose section, but only if verbose output was enabled
// This follows the spacing framework: "Every section ends with exactly one blank line"
// Use this after a group of verbose Info/Warning/Error calls to ensure proper spacing
func (v *VerboseLogger) EndSection() {
	if isVerboseEnabled() {
		fmt.Println()
	}
}

// DisplayFontOperationDetails displays verbose details for a font installation operation
// This formats and displays variant files, summary, and location information
// downloadSizeFormatted should be the result of cmd.FormatFileSize() - passed as string to avoid circular imports
func (v *VerboseLogger) DisplayFontOperationDetails(
	fontName string,
	sourceName string,
	installedFiles []string,
	skippedFiles []string,
	failedFiles []string,
	downloadSizeFormatted string,
	fontDir string,
	scopeLabel string,
	extractVariantName func(file, fontName string) string,
) {
	if !isVerboseEnabled() {
		return
	}

	// Display individual file operations
	for _, file := range installedFiles {
		variantName := extractVariantName(file, fontName)
		fmt.Printf("      ↳ %s - %s to %s\n", variantName, ui.SuccessText.Render("[Installed]"), scopeLabel)
	}
	for _, file := range skippedFiles {
		variantName := extractVariantName(file, fontName)
		fmt.Printf("      ↳ %s - %s to %s\n", variantName, ui.WarningText.Render("[Skipped] already installed"), scopeLabel)
	}
	for _, file := range failedFiles {
		variantName := extractVariantName(file, fontName)
		fmt.Printf("      ↳ %s - %s to %s\n", variantName, ui.ErrorText.Render("[Failed]"), scopeLabel)
	}

	// Show summary with download size
	totalVariants := len(installedFiles) + len(skippedFiles) + len(failedFiles)
	var summaryText string

	if len(installedFiles) > 0 {
		if downloadSizeFormatted != "" {
			summaryText = fmt.Sprintf("%s (%s) - %d variants installed to %s (%s)", fontName, sourceName, totalVariants, scopeLabel, downloadSizeFormatted)
		} else {
			summaryText = fmt.Sprintf("%s (%s) - %d variants installed to %s", fontName, sourceName, totalVariants, scopeLabel)
		}
	} else if len(skippedFiles) > 0 {
		summaryText = fmt.Sprintf("%s (%s) - %d variants already installed in %s", fontName, sourceName, totalVariants, scopeLabel)
	} else if len(failedFiles) > 0 {
		summaryText = fmt.Sprintf("%s (%s) - %d variants failed to install", fontName, sourceName, totalVariants)
	}

	if summaryText != "" {
		fmt.Printf("\n%s %s\n", ui.InfoText.Render("[INFO]"), ui.Text.Render(summaryText))
	}
	fmt.Printf("    %s\n\n", ui.Text.Render(fmt.Sprintf("Location: %s", fontDir)))
}
