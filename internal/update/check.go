package update

import (
	"fmt"
	"time"

	"fontget/internal/version"
)

// UpdateDeclinedGracePeriodHours is the grace period in hours before asking again
// after a user declines an update prompt
const UpdateDeclinedGracePeriodHours = 24

// CheckResult represents the result of a startup update check
type CheckResult struct {
	UpdateAvailable bool
	CurrentVersion  string
	LatestVersion   string
	Error           error
}

// ShouldCheckForUpdatesFromConfig determines if an update check should be performed
// based on the configuration and last check time
func ShouldCheckForUpdatesFromConfig(checkForUpdates bool, checkInterval int, lastChecked string) bool {
	if !checkForUpdates {
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
// Always calls callback, even on errors, to ensure timestamp is updated
func PerformStartupCheck(checkForUpdates bool, checkInterval int, lastChecked string, callback func(*CheckResult)) {
	// Check if we should perform the check
	if !ShouldCheckForUpdatesFromConfig(checkForUpdates, checkInterval, lastChecked) {
		return
	}

	// Perform the check
	result, err := CheckForUpdates()

	// Always create result, even on errors
	checkResult := &CheckResult{
		UpdateAvailable: false,
		CurrentVersion:  version.GetVersion(),
		LatestVersion:   "",
		Error:           err,
	}

	if err == nil {
		// Success - populate result with update information
		checkResult.UpdateAvailable = result.NeedsUpdate
		checkResult.CurrentVersion = result.Current
		checkResult.LatestVersion = result.Latest
		checkResult.Error = nil
	}

	// Always call callback, even on errors
	// This ensures LastUpdateCheck timestamp is always updated
	if callback != nil {
		callback(checkResult)
	}
}

// FormatUpdateNotification formats an update notification message
func FormatUpdateNotification(currentVersion, latestVersion string) string {
	return fmt.Sprintf("FontGet v%s is available (you have v%s).\nRun 'fontget update' to upgrade.", latestVersion, currentVersion)
}

// GetLastCheckedTimestamp returns the current time as an ISO timestamp string in UTC
func GetLastCheckedTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// ShouldShowUpdatePrompt checks if we should show the update prompt based on the grace period
// Returns true if we should show the prompt, false if we're still in the grace period
func ShouldShowUpdatePrompt(updateDeclinedUntil string) bool {
	// If no grace period is set, we should show the prompt
	if updateDeclinedUntil == "" {
		return true
	}

	// Parse the grace period timestamp
	declinedUntil, err := time.Parse(time.RFC3339, updateDeclinedUntil)
	if err != nil {
		// If parsing fails, assume we should show the prompt
		return true
	}

	// Check if grace period has passed
	now := time.Now().UTC()
	return now.After(declinedUntil)
}

// GetUpdateDeclinedUntilTimestamp returns a timestamp string for when the grace period expires
// This is calculated as current time + grace period
func GetUpdateDeclinedUntilTimestamp() string {
	gracePeriod := time.Duration(UpdateDeclinedGracePeriodHours) * time.Hour
	return time.Now().UTC().Add(gracePeriod).Format(time.RFC3339)
}
