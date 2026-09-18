package onboarding

import (
	"fmt"

	"fontget/internal/config"
	"fontget/internal/shared"

	tea "github.com/charmbracelet/bubbletea"
)

// RunFirstRunOnboarding checks if this is the first run and executes onboarding if needed
// This is the main function to call from cmd/root.go
func RunFirstRunOnboarding() error {
	// Check if this is the first run
	isFirstRun, err := config.IsFirstRun()
	if err != nil {
		// User-friendly error message per verbose/debug guidelines
		return fmt.Errorf("unable to check first run status: %w", err)
	}

	if !isFirstRun {
		return nil // Not first run, skip onboarding
	}

	// Use enhanced onboarding flow with interactive TUI (always start at Welcome)
	model := NewEnhancedOnboardingModel()
	program := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := program.Run()
	if err != nil {
		// User-friendly error message per verbose/debug guidelines
		return fmt.Errorf("onboarding failed: %w", err)
	}

	// Check if onboarding was actually completed (not just quit early)
	if m, ok := finalModel.(*EnhancedOnboardingModel); ok {
		if m.quitting && !m.onboardingCompleted {
			// User quit early (Ctrl+C, Q, etc.) - don't mark as completed
			// Return a specific error that indicates cancellation (not failure)
			// This allows the command to exit gracefully, and onboarding will restart on next command
			// since FirstRunCompleted is still false
			return shared.ErrOnboardingCancelled
		}
		if !m.onboardingCompleted {
			// User didn't complete the flow - don't mark as completed
			// Onboarding will restart on next command since FirstRunCompleted is still false
			return shared.ErrOnboardingIncomplete
		}
	} else {
		// Couldn't cast to EnhancedOnboardingModel - assume incomplete
		return shared.ErrOnboardingIncomplete
	}

	// Only mark as completed if user successfully finished the entire flow
	if err := config.MarkFirstRunCompleted(); err != nil {
		// User-friendly error message per verbose/debug guidelines
		return fmt.Errorf("unable to complete setup: %w", err)
	}

	return nil
}

// RunWizard runs the onboarding wizard regardless of first-run status
// This is useful for reconfiguring FontGet or testing
func RunWizard() error {
	// Use enhanced onboarding flow with interactive TUI
	model := NewEnhancedOnboardingModel()
	program := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := program.Run()
	if err != nil {
		return fmt.Errorf("wizard failed: %w", err)
	}

	// Check if wizard was actually completed (not just quit early)
	if m, ok := finalModel.(*EnhancedOnboardingModel); ok {
		if m.quitting && !m.onboardingCompleted {
			// User quit early (Ctrl+C, Q, etc.)
			return shared.ErrOnboardingCancelled
		}
		if !m.onboardingCompleted {
			// User didn't complete the flow
			return shared.ErrOnboardingIncomplete
		}

		// Save all selections (this will update config with new settings)
		if err := m.SaveSelections(); err != nil {
			return fmt.Errorf("failed to save wizard settings: %w", err)
		}
	} else {
		// Couldn't cast to EnhancedOnboardingModel - assume incomplete
		return shared.ErrOnboardingIncomplete
	}

	return nil
}
