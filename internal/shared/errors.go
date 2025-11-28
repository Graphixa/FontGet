package shared

import (
	"fmt"
	"strings"
)

// FontNotFoundError represents when a font is not found
type FontNotFoundError struct {
	FontName    string
	Suggestions []string
}

func (e *FontNotFoundError) Error() string {
	if len(e.Suggestions) > 0 {
		return fmt.Sprintf("font '%s' not found. Did you mean: %s?", e.FontName, strings.Join(e.Suggestions, ", "))
	}
	return fmt.Sprintf("font '%s' not found", e.FontName)
}

// FontInstallationError represents font installation failures
type FontInstallationError struct {
	FailedCount int
	TotalCount  int
	Details     []string
}

func (e *FontInstallationError) Error() string {
	return fmt.Sprintf("failed to install %d out of %d fonts", e.FailedCount, e.TotalCount)
}

// FontRemovalError represents font removal failures
type FontRemovalError struct {
	FailedCount int
	TotalCount  int
	Details     []string
}

func (e *FontRemovalError) Error() string {
	return fmt.Sprintf("failed to remove %d out of %d fonts", e.FailedCount, e.TotalCount)
}

// ConfigurationError represents configuration-related errors
type ConfigurationError struct {
	Field string
	Value string
	Hint  string
}

func (e *ConfigurationError) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("configuration error in field '%s' with value '%s': %s", e.Field, e.Value, e.Hint)
	}
	return fmt.Sprintf("configuration error in field '%s' with value '%s'", e.Field, e.Value)
}

// ElevationError represents elevation-related errors
type ElevationError struct {
	Operation string
	Platform  string
}

func (e *ElevationError) Error() string {
	return fmt.Sprintf("elevation required for operation '%s' on platform '%s'", e.Operation, e.Platform)
}
