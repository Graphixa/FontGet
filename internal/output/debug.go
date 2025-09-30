package output

import (
	"fmt"
	"fontget/internal/ui"
)

// DebugLogger provides developer-focused debug output functionality
type DebugLogger struct{}

var (
	debugInstance *DebugLogger
	debugChecker  func() bool
)

// GetDebug returns a singleton debug logger instance
func GetDebug() *DebugLogger {
	if debugInstance == nil {
		debugInstance = &DebugLogger{}
	}
	return debugInstance
}

// SetDebugChecker allows the cmd package to inject the debug flag checking function
// This avoids circular imports while maintaining clean architecture
func SetDebugChecker(checker func() bool) {
	debugChecker = checker
}

// isDebugEnabled checks if the --debug flag is set
func isDebugEnabled() bool {
	if debugChecker != nil {
		return debugChecker()
	}
	return false // Default to false if checker not set
}

// Message displays debug diagnostic information when --debug flag is enabled
// Shows detailed diagnostic information for developers and troubleshooting
func (d *DebugLogger) Message(format string, args ...interface{}) {
	if isDebugEnabled() {
		msg := fmt.Sprintf(format, args...)
		fmt.Printf("%s %s\n", ui.FeedbackInfo.Render("[DEBUG]"), msg)
	}
}

// Error displays debug error diagnostics when --debug flag is enabled
// Shows critical diagnostic error information for developers
func (d *DebugLogger) Error(format string, args ...interface{}) {
	if isDebugEnabled() {
		msg := fmt.Sprintf(format, args...)
		fmt.Printf("%s %s\n", ui.FeedbackError.Render("[DEBUG ERROR]"), msg)
	}
}

// Warning displays debug warning diagnostics when --debug flag is enabled
// Shows diagnostic warning information for developers
func (d *DebugLogger) Warning(format string, args ...interface{}) {
	if isDebugEnabled() {
		msg := fmt.Sprintf(format, args...)
		fmt.Printf("%s %s\n", ui.FeedbackWarning.Render("[DEBUG WARNING]"), msg)
	}
}

// Performance displays debug performance information when --debug flag is enabled
// Shows timing, memory usage, and other performance metrics
func (d *DebugLogger) Performance(format string, args ...interface{}) {
	if isDebugEnabled() {
		msg := fmt.Sprintf(format, args...)
		fmt.Printf("%s %s\n", ui.FeedbackText.Render("[DEBUG PERF]"), msg)
	}
}

// State displays debug state information when --debug flag is enabled
// Shows variable states, configuration values, and system state
func (d *DebugLogger) State(format string, args ...interface{}) {
	if isDebugEnabled() {
		msg := fmt.Sprintf(format, args...)
		fmt.Printf("%s %s\n", ui.FeedbackText.Render("[DEBUG STATE]"), msg)
	}
}
