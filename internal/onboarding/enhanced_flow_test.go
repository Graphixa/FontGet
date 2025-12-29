package onboarding

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewEnhancedOnboardingModel(t *testing.T) {
	model := NewEnhancedOnboardingModel()

	if model == nil {
		t.Fatal("NewEnhancedOnboardingModel() returned nil")
	}

	// Check initial state
	if model.currentStep != 0 {
		t.Errorf("NewEnhancedOnboardingModel() currentStep = %d, want 0", model.currentStep)
	}

	if model.width != 80 {
		t.Errorf("NewEnhancedOnboardingModel() width = %d, want 80", model.width)
	}

	if model.height != 24 {
		t.Errorf("NewEnhancedOnboardingModel() height = %d, want 24", model.height)
	}

	if model.quitting {
		t.Error("NewEnhancedOnboardingModel() quitting should be false initially")
	}

	if model.onboardingCompleted {
		t.Error("NewEnhancedOnboardingModel() onboardingCompleted should be false initially")
	}

	// Check that steps are created
	// Expected: Welcome, License Agreement, Wizard Choice, Sources, Settings, Theme Selection, Completion
	expectedStepCount := 7
	if len(model.steps) != expectedStepCount {
		t.Errorf("NewEnhancedOnboardingModel() len(steps) = %d, want %d", len(model.steps), expectedStepCount)
	}

	// Check step names
	expectedSteps := []string{"Welcome", "License Agreement", "Wizard Choice", "Sources", "Settings", "Theme Selection", "Completion"}
	for i, step := range model.steps {
		if i < len(expectedSteps) && step.Name() != expectedSteps[i] {
			t.Errorf("Step %d name = %q, want %q", i, step.Name(), expectedSteps[i])
		}
	}

	// Check source selections are initialized
	if len(model.sourceSelections) == 0 {
		t.Error("NewEnhancedOnboardingModel() sourceSelections should not be empty")
	}

	// Check settings values are initialized
	if len(model.settingsValues) == 0 {
		t.Error("NewEnhancedOnboardingModel() settingsValues should not be empty")
	}

	// Check required settings keys
	requiredKeys := []string{"autoCheck", "checkForUpdates", "usePopularitySort"}
	for _, key := range requiredKeys {
		if _, exists := model.settingsValues[key]; !exists {
			t.Errorf("NewEnhancedOnboardingModel() settingsValues missing key: %q", key)
		}
	}
}

func TestEnhancedOnboardingModel_GoToNextStep(t *testing.T) {
	model := NewEnhancedOnboardingModel()
	initialStep := model.currentStep

	// Move to next step
	model.GoToNextStep()

	if model.currentStep != initialStep+1 {
		t.Errorf("GoToNextStep() currentStep = %d, want %d", model.currentStep, initialStep+1)
	}

	// Move to last step
	for model.currentStep < len(model.steps)-1 {
		model.GoToNextStep()
	}

	// Try to go beyond last step
	lastStep := model.currentStep
	model.GoToNextStep()

	if model.currentStep != lastStep {
		t.Errorf("GoToNextStep() should not exceed last step, currentStep = %d, want %d", model.currentStep, lastStep)
	}
}

func TestEnhancedOnboardingModel_GoToPreviousStep(t *testing.T) {
	model := NewEnhancedOnboardingModel()

	// Move forward a few steps
	model.GoToNextStep()
	model.GoToNextStep()
	currentStep := model.currentStep

	// Move back
	model.GoToPreviousStep()

	if model.currentStep != currentStep-1 {
		t.Errorf("GoToPreviousStep() currentStep = %d, want %d", model.currentStep, currentStep-1)
	}

	// Try to go before first step
	model.currentStep = 0
	model.GoToPreviousStep()

	if model.currentStep != 0 {
		t.Errorf("GoToPreviousStep() should not go below 0, currentStep = %d, want 0", model.currentStep)
	}
}

func TestEnhancedOnboardingModel_View(t *testing.T) {
	model := NewEnhancedOnboardingModel()

	// Test View() when not quitting
	view := model.View()
	if view == "" {
		t.Error("View() returned empty string for valid step")
	}

	// Test View() when quitting
	model.quitting = true
	view = model.View()
	if view != "" {
		t.Error("View() should return empty string when quitting")
	}

	// Test View() with invalid step index
	model.quitting = false
	model.currentStep = 999
	view = model.View()
	if view != "" {
		t.Error("View() should return empty string for invalid step index")
	}
}

func TestEnhancedOnboardingModel_Update(t *testing.T) {
	model := NewEnhancedOnboardingModel()

	// Test WindowSizeMsg
	width, height := 100, 50
	msg := tea.WindowSizeMsg{Width: width, Height: height}
	updatedModel, cmd := model.Update(msg)

	if updatedModel == nil {
		t.Fatal("Update() returned nil model")
	}

	enhancedModel, ok := updatedModel.(*EnhancedOnboardingModel)
	if !ok {
		t.Fatal("Update() returned wrong model type")
	}

	if enhancedModel.width != width {
		t.Errorf("Update(WindowSizeMsg) width = %d, want %d", enhancedModel.width, width)
	}

	if enhancedModel.height != height {
		t.Errorf("Update(WindowSizeMsg) height = %d, want %d", enhancedModel.height, height)
	}

	if cmd != nil {
		t.Error("Update(WindowSizeMsg) should return nil command")
	}

	// Test quit key (ctrl+c)
	model = NewEnhancedOnboardingModel()
	quitMsg := tea.KeyMsg{Type: tea.KeyCtrlC}
	updatedModel, cmd = model.Update(quitMsg)

	enhancedModel, ok = updatedModel.(*EnhancedOnboardingModel)
	if !ok {
		t.Fatal("Update() returned wrong model type")
	}

	if !enhancedModel.quitting {
		t.Error("Update(KeyCtrlC) should set quitting to true")
	}

	if enhancedModel.onboardingCompleted {
		t.Error("Update(KeyCtrlC) should set onboardingCompleted to false")
	}

	// Test quit key (q)
	model = NewEnhancedOnboardingModel()
	quitMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	updatedModel, cmd = model.Update(quitMsg)

	enhancedModel, ok = updatedModel.(*EnhancedOnboardingModel)
	if !ok {
		t.Fatal("Update() returned wrong model type")
	}

	if !enhancedModel.quitting {
		t.Error("Update(KeyQ) should set quitting to true")
	}
}

func TestWelcomeStepEnhanced(t *testing.T) {
	step := NewWelcomeStepEnhanced()
	model := NewEnhancedOnboardingModel()

	if step.Name() != "Welcome" {
		t.Errorf("WelcomeStepEnhanced.Name() = %q, want %q", step.Name(), "Welcome")
	}

	if step.CanGoBack() {
		t.Error("WelcomeStepEnhanced.CanGoBack() should return false")
	}

	if !step.CanGoNext() {
		t.Error("WelcomeStepEnhanced.CanGoNext() should return true")
	}

	view := step.View(model)
	if view == "" {
		t.Error("WelcomeStepEnhanced.View() returned empty string")
	}
}

func TestLicenseStepEnhanced(t *testing.T) {
	step := NewLicenseAgreementStepEnhanced()
	model := NewEnhancedOnboardingModel()

	if step.Name() != "License Agreement" {
		t.Errorf("LicenseAgreementStepEnhanced.Name() = %q, want %q", step.Name(), "License Agreement")
	}

	if !step.CanGoBack() {
		t.Error("LicenseAgreementStepEnhanced.CanGoBack() should return true")
	}

	// CanGoNext should always return true (text-based acceptance, no confirmation needed)
	if !step.CanGoNext() {
		t.Error("LicenseAgreementStepEnhanced.CanGoNext() should return true (text-based acceptance)")
	}

	view := step.View(model)
	if view == "" {
		t.Error("LicenseAgreementStepEnhanced.View() returned empty string")
	}
}

func TestSourcesStepEnhanced(t *testing.T) {
	step := NewSourcesStepEnhanced()
	model := NewEnhancedOnboardingModel()

	if step.Name() != "Sources" {
		t.Errorf("SourcesStepEnhanced.Name() = %q, want %q", step.Name(), "Sources")
	}

	if !step.CanGoBack() {
		t.Error("SourcesStepEnhanced.CanGoBack() should return true")
	}

	if !step.CanGoNext() {
		t.Error("SourcesStepEnhanced.CanGoNext() should return true")
	}

	view := step.View(model)
	if view == "" {
		t.Error("SourcesStepEnhanced.View() returned empty string")
	}
}

func TestSettingsStepEnhanced(t *testing.T) {
	step := NewSettingsStepEnhanced()
	model := NewEnhancedOnboardingModel()

	if step.Name() != "Settings" {
		t.Errorf("SettingsStepEnhanced.Name() = %q, want %q", step.Name(), "Settings")
	}

	if !step.CanGoBack() {
		t.Error("SettingsStepEnhanced.CanGoBack() should return true")
	}

	if !step.CanGoNext() {
		t.Error("SettingsStepEnhanced.CanGoNext() should return true")
	}

	view := step.View(model)
	if view == "" {
		t.Error("SettingsStepEnhanced.View() returned empty string")
	}
}

func TestCompletionStepEnhanced(t *testing.T) {
	step := NewCompletionStepEnhanced()
	model := NewEnhancedOnboardingModel()

	if step.Name() != "Completion" {
		t.Errorf("CompletionStepEnhanced.Name() = %q, want %q", step.Name(), "Completion")
	}

	if !step.CanGoBack() {
		t.Error("CompletionStepEnhanced.CanGoBack() should return true")
	}

	if step.CanGoNext() {
		t.Error("CompletionStepEnhanced.CanGoNext() should return false")
	}

	view := step.View(model)
	if view == "" {
		t.Error("CompletionStepEnhanced.View() returned empty string")
	}
}

func TestEnhancedOnboardingModel_Init(t *testing.T) {
	model := NewEnhancedOnboardingModel()
	cmd := model.Init()

	if cmd != nil {
		t.Error("Init() should return nil command")
	}
}

func TestEnhancedOnboardingModel_resetStepViewFlag(t *testing.T) {
	model := NewEnhancedOnboardingModel()

	// Move to Sources step
	model.GoToNextStep() // Welcome -> License Agreement
	model.GoToNextStep() // License Agreement -> Wizard Choice
	model.GoToNextStep() // Wizard Choice -> Sources (index 3)

	// Get the Sources step
	sourcesStep, ok := model.steps[3].(*SourcesStepEnhanced)
	if !ok {
		t.Fatal("Step at index 3 should be SourcesStepEnhanced")
	}

	// Set hasBeenViewed to true
	sourcesStep.hasBeenViewed = true

	// Reset the flag
	model.resetStepViewFlag()

	// Check that flag was reset
	if sourcesStep.hasBeenViewed {
		t.Error("resetStepViewFlag() should reset hasBeenViewed to false")
	}
}
