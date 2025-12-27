package shared

import (
	"errors"
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

// ErrOperationCancelled is a sentinel error used to indicate that an operation was cancelled by the user.
var ErrOperationCancelled = errors.New("operation cancelled")

// ErrExportCancelled is a sentinel error used to indicate that an export operation was cancelled by the user.
var ErrExportCancelled = errors.New("export cancelled")

// ErrOnboardingCancelled is a sentinel error used to indicate that onboarding was cancelled by the user.
var ErrOnboardingCancelled = errors.New("onboarding cancelled - please complete setup to continue")

// ErrOnboardingIncomplete is a sentinel error used to indicate that onboarding was not completed.
var ErrOnboardingIncomplete = errors.New("onboarding incomplete - please complete setup to continue")

// PathValidationError represents path validation errors
type PathValidationError struct {
	Path    string
	Reason  string
	Details string
}

func (e *PathValidationError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("invalid path '%s': %s (%s)", e.Path, e.Reason, e.Details)
	}
	return fmt.Sprintf("invalid path '%s': %s", e.Path, e.Reason)
}

// Unwrap returns nil as this is a terminal error
func (e *PathValidationError) Unwrap() error {
	return nil
}
