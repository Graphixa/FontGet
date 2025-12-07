package update

import (
	"time"
)

// UpdateConfig represents update configuration settings
type UpdateConfig struct {
	AutoCheck     bool
	AutoUpdate    bool
	CheckInterval int // Hours between checks
	LastChecked   time.Time
	UpdateChannel string // stable/beta/nightly
}

// ShouldCheckForUpdates determines if an update check should be performed
// based on the configuration and last check time
func ShouldCheckForUpdates(config *UpdateConfig) bool {
	if !config.AutoCheck {
		return false
	}

	// If never checked, should check
	if config.LastChecked.IsZero() {
		return true
	}

	// Check if interval has passed
	interval := time.Duration(config.CheckInterval) * time.Hour
	return time.Since(config.LastChecked) >= interval
}

// MarkChecked updates the LastChecked timestamp to now (UTC)
func MarkChecked(config *UpdateConfig) {
	config.LastChecked = time.Now().UTC()
}

// DefaultUpdateConfig returns default update configuration
func DefaultUpdateConfig() *UpdateConfig {
	return &UpdateConfig{
		AutoCheck:     true,
		AutoUpdate:    false,
		CheckInterval: 24, // 24 hours
		LastChecked:   time.Time{},
		UpdateChannel: "stable",
	}
}
