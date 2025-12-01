package onboarding

import (
	"fmt"

	"fontget/internal/config"
)

// OnboardingStep represents a single step in the onboarding flow
// This interface allows for easy extension - just implement it to add new steps
type OnboardingStep interface {
	// Name returns a human-readable name for this step (for logging/debugging)
	Name() string

	// Execute runs the step and returns:
	// - shouldContinue: true if onboarding should continue to next step, false to abort
	// - error: any error that occurred during execution
	Execute() (shouldContinue bool, err error)

	// CanSkip returns true if this step can be skipped
	// If true, the step may offer a skip option to the user
	CanSkip() bool
}

// OnboardingFlow manages the execution of onboarding steps
type OnboardingFlow struct {
	steps []OnboardingStep
}

// NewOnboardingFlow creates a new onboarding flow
func NewOnboardingFlow() *OnboardingFlow {
	return &OnboardingFlow{
		steps: make([]OnboardingStep, 0),
	}
}

// AddStep adds a step to the onboarding flow
// Steps are executed in the order they are added
func (f *OnboardingFlow) AddStep(step OnboardingStep) {
	f.steps = append(f.steps, step)
}

// Run executes all steps in the flow sequentially
// Stops if any step returns shouldContinue=false or an error
func (f *OnboardingFlow) Run() error {
	for _, step := range f.steps {
		shouldContinue, err := step.Execute()
		if err != nil {
			// User-friendly error message per verbose/debug guidelines
			// The step's error message is already user-friendly, so we preserve it
			return err
		}
		if !shouldContinue {
			// User declined or aborted - this is expected behavior, not an error
			// Return a user-friendly message
			return fmt.Errorf("setup was cancelled")
		}
	}
	return nil
}

// RunStep executes a single step (useful for testing or conditional execution)
func (f *OnboardingFlow) RunStep(step OnboardingStep) (bool, error) {
	return step.Execute()
}

// NewDefaultOnboardingFlow creates the default onboarding flow with all standard steps
// This is the main entry point for first-run onboarding
func NewDefaultOnboardingFlow() *OnboardingFlow {
	flow := NewOnboardingFlow()

	// Add steps in order: Welcome -> License -> Settings -> Completion
	flow.AddStep(NewWelcomeStep())
	flow.AddStep(NewLicenseStep())
	flow.AddStep(NewSettingsStep())
	flow.AddStep(NewCompletionStep())

	return flow
}

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

	// Create and run the default onboarding flow
	flow := NewDefaultOnboardingFlow()
	if err := flow.Run(); err != nil {
		// Error messages from flow.Run() are already user-friendly
		return err
	}

	// Mark first run as completed
	if err := config.MarkFirstRunCompleted(); err != nil {
		// User-friendly error message per verbose/debug guidelines
		return fmt.Errorf("unable to complete setup: %w", err)
	}

	return nil
}
