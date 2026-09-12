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

// Classified operation errors. Callers should use errors.Is; do not match message text.
var (
	// ErrOperationCancelled is a sentinel error used to indicate that an operation was cancelled by the user.
	ErrOperationCancelled = errors.New("operation cancelled")

	// ErrRecoveryRequired is returned when installation rollback could not fully restore prior state.
	ErrRecoveryRequired = errors.New("installation recovery required")
	// ErrLocalFailure is returned for local permission or disk errors that network fallback cannot repair.
	ErrLocalFailure = errors.New("local filesystem failure")
)

// DisplayedError wraps an error whose user-facing message has already been printed.
// The process should still exit non-zero, but the entry point must not print it again.
type DisplayedError struct {
	Cause error
}

func (e *DisplayedError) Error() string {
	if e == nil || e.Cause == nil {
		return "error already displayed"
	}
	return e.Cause.Error()
}

func (e *DisplayedError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// AlreadyPrinted wraps err so the executable entry point skips duplicate output.
func AlreadyPrinted(err error) error {
	if err == nil {
		return nil
	}
	var displayed *DisplayedError
	if errors.As(err, &displayed) {
		return err
	}
	return &DisplayedError{Cause: err}
}

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
