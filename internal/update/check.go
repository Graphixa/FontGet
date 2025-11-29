package update

import (
	"fmt"
	"time"
)

// CheckResult represents the result of a startup update check
type CheckResult struct {
	UpdateAvailable bool
	CurrentVersion  string
	LatestVersion   string
	Error           error
}

// ShouldCheckForUpdatesFromConfig determines if an update check should be performed
// based on the configuration and last check time
func ShouldCheckForUpdatesFromConfig(autoCheck bool, checkInterval int, lastChecked string) bool {
	if !autoCheck {
		return false
	}

	// If never checked, should check
	if lastChecked == "" {
		return true
	}

	// Parse last checked time
	lastCheckedTime, err := time.Parse(time.RFC3339, lastChecked)
	if err != nil {
		// If parsing fails, assume we should check
		return true
	}

	// Check if interval has passed
	interval := time.Duration(checkInterval) * time.Hour
	return time.Since(lastCheckedTime) >= interval
}

// PerformStartupCheck performs a non-blocking update check for startup
// This should be called in a goroutine to avoid blocking application startup
func PerformStartupCheck(autoCheck bool, checkInterval int, lastChecked string, callback func(*CheckResult)) {
	// Check if we should perform the check
	if !ShouldCheckForUpdatesFromConfig(autoCheck, checkInterval, lastChecked) {
		return
	}

	// Perform the check
	result, err := CheckForUpdates()
	if err != nil {
		// Don't show errors during startup check - silently fail
		// Errors will be shown when user explicitly runs update command
		return
	}

	checkResult := &CheckResult{
		UpdateAvailable: result.NeedsUpdate,
		CurrentVersion:  result.Current,
		LatestVersion:   result.Latest,
		Error:           nil,
	}

	// Call callback with result
	if callback != nil {
		callback(checkResult)
	}
}

// FormatUpdateNotification formats an update notification message
func FormatUpdateNotification(currentVersion, latestVersion string) string {
	return fmt.Sprintf("FontGet v%s is available (you have v%s).\nRun 'fontget update' to upgrade.", latestVersion, currentVersion)
}

// GetLastCheckedTimestamp returns the current time as an ISO timestamp string
func GetLastCheckedTimestamp() string {
	return time.Now().Format(time.RFC3339)
}
