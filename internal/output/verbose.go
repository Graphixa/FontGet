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

// isVerboseEnabled checks if the --verbose flag is set
func isVerboseEnabled() bool {
	if verboseChecker != nil {
		return verboseChecker()
	}
	return false // Default to false if checker not set
}

// Info displays verbose information to users when --verbose flag is enabled
// Shows operational details like file paths, parameters, and progress
func (v *VerboseLogger) Info(format string, args ...interface{}) {
	if isVerboseEnabled() {
		msg := fmt.Sprintf(format, args...)
		fmt.Printf("%s %s\n", ui.FeedbackInfo.Render("[INFO]"), msg)
	}
}

// Warning displays verbose warnings to users when --verbose flag is enabled
// Shows non-critical issues that users should be aware of
func (v *VerboseLogger) Warning(format string, args ...interface{}) {
	if isVerboseEnabled() {
		msg := fmt.Sprintf(format, args...)
		fmt.Printf("%s %s\n", ui.FeedbackWarning.Render("[WARNING]"), msg)
	}
}

// Error displays verbose error details to users when --verbose flag is enabled
// Shows detailed error information without timestamps (user-friendly)
func (v *VerboseLogger) Error(format string, args ...interface{}) {
	if isVerboseEnabled() {
		msg := fmt.Sprintf(format, args...)
		fmt.Printf("%s %s\n", ui.FeedbackError.Render("[ERROR]"), msg)
	}
}

// Success displays verbose success messages to users when --verbose flag is enabled
// Shows detailed success information for completed operations
func (v *VerboseLogger) Success(format string, args ...interface{}) {
	if isVerboseEnabled() {
		msg := fmt.Sprintf(format, args...)
		fmt.Printf("%s %s\n", ui.FeedbackSuccess.Render("[SUCCESS]"), msg)
	}
}

// Detail displays additional detailed information with custom prefix
// Useful for showing file paths, parameters, or step-by-step progress
func (v *VerboseLogger) Detail(prefix, format string, args ...interface{}) {
	if isVerboseEnabled() {
		msg := fmt.Sprintf(format, args...)
		fmt.Printf("    %s %s\n", ui.FeedbackText.Render(prefix+":"), msg)
	}
}
