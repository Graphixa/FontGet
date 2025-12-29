package onboarding

import (
	"fmt"
	"strings"

	"fontget/internal/components"
	"fontget/internal/config"
	"fontget/internal/sources"
	"fontget/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// EnhancedOnboardingModel represents the enhanced onboarding flow with navigation
type EnhancedOnboardingModel struct {
	currentStep         int
	steps               []OnboardingStepInterface
	sourceSelections    map[string]bool        // Source name -> enabled
	settingsValues      map[string]interface{} // Setting name -> value
	width               int
	height              int
	quitting            bool
	onboardingCompleted bool // True only when user successfully completes the entire flow
	customizeChoice     bool // true = customize, false = let it ride
}

// OnboardingStepInterface represents a step in the enhanced onboarding flow
type OnboardingStepInterface interface {
	Name() string
	View(model *EnhancedOnboardingModel) string
	Update(model *EnhancedOnboardingModel, msg tea.Msg) (tea.Model, tea.Cmd)
	CanGoBack() bool
	CanGoNext() bool
}

// NewEnhancedOnboardingModel creates a new enhanced onboarding model
func NewEnhancedOnboardingModel() *EnhancedOnboardingModel {
	// Initialize source selections from default sources
	defaultSources := sources.DefaultSources()
	sourceSelections := make(map[string]bool)
	for name, info := range defaultSources {
		sourceSelections[name] = info.Enabled
	}

	// Initialize settings values from defaults
	defaults := config.DefaultUserPreferences()
	settingsValues := map[string]interface{}{
		"checkForUpdates":   defaults.Update.CheckForUpdates,
		"usePopularitySort": defaults.Search.EnablePopularitySort,
		// Legacy key for backward compatibility (used by settings step)
		"autoCheck": defaults.Update.CheckForUpdates,
	}

	model := &EnhancedOnboardingModel{
		currentStep:      0,
		sourceSelections: sourceSelections,
		settingsValues:   settingsValues,
		width:            80,
		height:           24,
		customizeChoice:  true, // Default to customize (will be set by wizard choice step)
	}

	// Create steps
	themeStep, err := NewThemeSelectionStepEnhanced()
	if err != nil {
		// If theme step creation fails, we'll skip it (shouldn't happen, but handle gracefully)
		themeStep = nil
	}

	steps := []OnboardingStepInterface{
		NewWelcomeStepEnhanced(),
		NewLicenseAgreementStepEnhanced(),
		NewWizardChoiceStepEnhanced(),
		NewSourcesStepEnhanced(),
		NewSettingsStepEnhanced(),
	}
	if themeStep != nil {
		steps = append(steps, themeStep)
	}
	steps = append(steps, NewCompletionStepEnhanced())

	model.steps = steps

	return model
}

// Init initializes the enhanced onboarding model
func (m EnhancedOnboardingModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m EnhancedOnboardingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			// User quit early - don't mark as completed
			m.quitting = true
			m.onboardingCompleted = false
			return &m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return &m, nil
	}

	// Delegate to current step
	if m.currentStep >= 0 && m.currentStep < len(m.steps) {
		return m.steps[m.currentStep].Update(&m, msg)
	}

	return &m, nil
}

// View renders the current step
func (m EnhancedOnboardingModel) View() string {
	if m.quitting {
		return ""
	}

	if m.currentStep >= 0 && m.currentStep < len(m.steps) {
		return m.steps[m.currentStep].View(&m)
	}

	return ""
}

// GoToNextStep moves to the next step
func (m *EnhancedOnboardingModel) GoToNextStep() {
	if m.currentStep < len(m.steps)-1 {
		m.currentStep++
		// Reset focus to list when entering new step
		m.resetFocus()
		// Reset hasBeenViewed flag for the new step
		m.resetStepViewFlag()
	}
}

// GoToPreviousStep moves to the previous step
func (m *EnhancedOnboardingModel) GoToPreviousStep() {
	if m.currentStep > 0 {
		m.currentStep--
		// Reset focus to list when entering new step
		m.resetFocus()
		// Reset hasBeenViewed flag for the new step
		m.resetStepViewFlag()
	}
}

// resetFocus resets focus to the list (checkboxes/switches) when entering a step
func (m *EnhancedOnboardingModel) resetFocus() {
	// This will be handled by each step's initialization
	// Steps should reset their focus state when entered
}

// resetStepViewFlag resets the hasBeenViewed flag for the current step
func (m *EnhancedOnboardingModel) resetStepViewFlag() {
	if m.currentStep >= 0 && m.currentStep < len(m.steps) {
		switch step := m.steps[m.currentStep].(type) {
		case *SourcesStepEnhanced:
			step.hasBeenViewed = false
		case *LicenseAgreementStepEnhanced:
			step.hasBeenViewed = false
		case *SettingsStepEnhanced:
			step.hasBeenViewed = false
		case *WizardChoiceStepEnhanced:
			step.hasBeenViewed = false
		}
	}
}

// SaveSelections saves all selections to config files
func (m *EnhancedOnboardingModel) SaveSelections() error {
	// Save source selections to manifest
	manifest, err := config.LoadManifest()
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	// Update source enabled states in manifest
	for name, enabled := range m.sourceSelections {
		if sourceConfig, exists := manifest.Sources[name]; exists {
			sourceConfig.Enabled = enabled
			manifest.Sources[name] = sourceConfig
		}
	}

	// Save manifest with updated source states
	if err := config.SaveManifest(manifest); err != nil {
		return fmt.Errorf("failed to save source selections: %w", err)
	}

	// Save settings
	appConfig := config.GetUserPreferences()
	if appConfig == nil {
		// Create default config first
		if err := config.GenerateInitialUserPreferences(); err != nil {
			return fmt.Errorf("failed to create config: %w", err)
		}
		appConfig = config.GetUserPreferences()
	}

	// Update settings
	// Check for checkForUpdates first, fallback to autoCheck for backward compatibility
	var checkForUpdates bool
	if val, ok := m.settingsValues["checkForUpdates"].(bool); ok {
		checkForUpdates = val
	} else if val, ok := m.settingsValues["autoCheck"].(bool); ok {
		checkForUpdates = val
	} else {
		// Use default if neither exists
		defaults := config.DefaultUserPreferences()
		checkForUpdates = defaults.Update.CheckForUpdates
	}
	appConfig.Update.CheckForUpdates = checkForUpdates
	if usePopularitySort, ok := m.settingsValues["usePopularitySort"].(bool); ok {
		appConfig.Search.EnablePopularitySort = usePopularitySort
	}
	// Save theme selection
	if theme, ok := m.settingsValues["theme"].(string); ok && theme != "" {
		appConfig.Theme = theme
	} else {
		// Default to "catppuccin" if not set
		appConfig.Theme = "catppuccin"
	}

	// Save config
	if err := config.SaveUserPreferences(appConfig); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	return nil
}

// WelcomeStepEnhanced is the enhanced welcome step with button
type WelcomeStepEnhanced struct {
	buttonGroup *components.ButtonGroup
}

func NewWelcomeStepEnhanced() *WelcomeStepEnhanced {
	group := components.NewButtonGroup([]string{"Next"}, 0)
	group.SetFocus(true) // Button has focus by default (only component)
	return &WelcomeStepEnhanced{
		buttonGroup: group,
	}
}

func (s *WelcomeStepEnhanced) Name() string {
	return "Welcome"
}

func (s *WelcomeStepEnhanced) CanGoBack() bool {
	return false
}

func (s *WelcomeStepEnhanced) CanGoNext() bool {
	return true
}

func (s *WelcomeStepEnhanced) View(model *EnhancedOnboardingModel) string {
	var result strings.Builder
	// Calculate available width
	availableWidth := model.width - 4
	if availableWidth < 60 {
		availableWidth = 60
	}

	result.WriteString("\n")
	result.WriteString(ui.PageTitle.Render("Welcome to FontGet!"))
	result.WriteString("\n\n")
	result.WriteString(ui.Text.Render("Welcome! It looks like this is your first time using FontGet."))
	result.WriteString("\n\n")
	aboutText := "FontGet is a powerful command-line font manager that helps you install and manage fonts from various sources."
	// Wrap plain text first, then apply styling to each line
	aboutLines := wrapText(aboutText, availableWidth)
	for _, line := range aboutLines {
		result.WriteString(ui.SecondaryText.Render(line))
		result.WriteString("\n")
	}
	result.WriteString("\n")

	// Button at bottom
	result.WriteString(s.buttonGroup.Render())
	result.WriteString("\n")

	return result.String()
}

func (s *WelcomeStepEnhanced) Update(model *EnhancedOnboardingModel, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Welcome step only has button, so button always has focus
		s.buttonGroup.SetFocus(true)

		action := s.buttonGroup.HandleKey(msg.String())
		if action == "next" || action == "enter" || msg.String() == "enter" || msg.String() == " " {
			model.GoToNextStep()
			return model, nil
		}
	}
	return model, nil
}

// LicenseAgreementStepEnhanced is the enhanced license step with text-based acceptance
type LicenseAgreementStepEnhanced struct {
	buttonGroup   *components.ButtonGroup
	hasBeenViewed bool // Track if step has been viewed to only reset once
}

func NewLicenseAgreementStepEnhanced() *LicenseAgreementStepEnhanced {
	group := components.NewButtonGroup([]string{"Back", "Continue"}, 1) // Continue selected by default
	group.SetFocus(true)                                                // Buttons have focus by default (button-only screen)
	return &LicenseAgreementStepEnhanced{
		buttonGroup: group,
	}
}

func (s *LicenseAgreementStepEnhanced) Name() string {
	return "License Agreement"
}

func (s *LicenseAgreementStepEnhanced) CanGoBack() bool {
	return true
}

func (s *LicenseAgreementStepEnhanced) CanGoNext() bool {
	return true // No validation needed - text-based acceptance
}

func (s *LicenseAgreementStepEnhanced) View(model *EnhancedOnboardingModel) string {
	// Reset button to default selection only on first view of this step
	if !s.hasBeenViewed {
		s.buttonGroup.ResetToDefault()
		s.hasBeenViewed = true
	}

	var result strings.Builder

	// Get default sources for license info
	defaultSources := sources.DefaultSources()

	// Calculate available width
	availableWidth := model.width - 4
	if availableWidth < 60 {
		availableWidth = 60
	}

	result.WriteString("\n")
	result.WriteString(ui.PageTitle.Render("License Agreement"))
	result.WriteString("\n\n")

	// License text
	introText := "FontGet installs fonts from various sources. These fonts are subject to their respective license agreements. You are responsible for ensuring compliance with each font's license terms."
	introLines := wrapText(introText, availableWidth)
	for _, line := range introLines {
		result.WriteString(line)
		result.WriteString("\n")
	}
	result.WriteString("\n")

	// Source list
	sourceOrder := []string{"Google Fonts", "Nerd Fonts", "Font Squirrel"}
	for _, sourceName := range sourceOrder {
		if _, exists := defaultSources[sourceName]; exists {
			result.WriteString(fmt.Sprintf("  %s %s\n", "•", ui.TableSourceName.Render(sourceName)))
		}
	}
	result.WriteString("\n")

	// Text-based acceptance message
	acceptanceText := "By continuing to use FontGet, you agree to all license agreements."
	acceptanceLines := wrapText(acceptanceText, availableWidth)
	result.WriteString(ui.InfoText.Render(acceptanceLines[0]))
	if len(acceptanceLines) > 1 {
		for i := 1; i < len(acceptanceLines); i++ {
			result.WriteString("\n")
			result.WriteString(ui.InfoText.Render(acceptanceLines[i]))
		}
	}
	result.WriteString("\n\n")

	// Buttons
	result.WriteString(s.buttonGroup.Render())
	result.WriteString("\n")

	return result.String()
}

func (s *LicenseAgreementStepEnhanced) Update(model *EnhancedOnboardingModel, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		// Handle backspace for back navigation
		if key == "backspace" && s.CanGoBack() {
			model.GoToPreviousStep()
			return model, nil
		}

		// License step only has buttons, so buttons always have focus
		s.buttonGroup.SetFocus(true)

		// Handle button navigation (left/right to switch between buttons)
		action := s.buttonGroup.HandleKey(key)
		if action != "" {
			// Button was activated
			switch action {
			case "back":
				model.GoToPreviousStep()
				return model, nil
			case "continue", "enter":
				// Accept all default sources
				for sourceName := range sources.DefaultSources() {
					if err := config.AcceptSource(sourceName); err != nil {
						// Log error but continue
					}
				}
				model.GoToNextStep()
				return model, nil
			}
		}
		// Return even if no action, to allow button navigation (left/right)
		return model, nil
	}
	return model, nil
}

// WizardChoiceStepEnhanced is the wizard choice step where user chooses to customize or accept defaults
type WizardChoiceStepEnhanced struct {
	buttonGroup   *components.ButtonGroup // Buttons: Back, Accept Defaults, Customize
	hasBeenViewed bool                    // Track if step has been viewed to only reset once
}

func NewWizardChoiceStepEnhanced() *WizardChoiceStepEnhanced {
	// Buttons: Back (0), Accept Defaults (1), Customize (2) - Customize selected by default
	group := components.NewButtonGroup([]string{"Back", "Accept Defaults", "Customize"}, 2)
	group.SetFocus(true) // Buttons have focus by default
	return &WizardChoiceStepEnhanced{
		buttonGroup: group,
	}
}

func (s *WizardChoiceStepEnhanced) Name() string {
	return "Wizard Choice"
}

func (s *WizardChoiceStepEnhanced) CanGoBack() bool {
	return true
}

func (s *WizardChoiceStepEnhanced) CanGoNext() bool {
	return true // Selection triggers navigation, no separate Next button
}

func (s *WizardChoiceStepEnhanced) View(model *EnhancedOnboardingModel) string {
	// Reset button to default selection only on first view of this step
	if !s.hasBeenViewed {
		s.buttonGroup.ResetToDefault()
		s.hasBeenViewed = true
	}

	var result strings.Builder

	// Calculate available width
	availableWidth := model.width - 4
	if availableWidth < 60 {
		availableWidth = 60
	}

	result.WriteString("\n")
	result.WriteString(ui.PageTitle.Render("Customize FontGet"))
	result.WriteString("\n\n")

	descriptionText := "Would you like to customize FontGet settings, or accept the defaults?"
	descriptionLines := wrapText(descriptionText, availableWidth)
	for _, line := range descriptionLines {
		result.WriteString(ui.Text.Render(line))
		result.WriteString("\n")
	}
	result.WriteString("\n")

	// Buttons
	result.WriteString(s.buttonGroup.Render())
	result.WriteString("\n")

	return result.String()
}

func (s *WizardChoiceStepEnhanced) Update(model *EnhancedOnboardingModel, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		// Handle backspace for back navigation
		if key == "backspace" && s.CanGoBack() {
			model.GoToPreviousStep()
			return model, nil
		}

		// Wizard choice step only has buttons, so buttons always have focus
		s.buttonGroup.SetFocus(true)

		// Handle button navigation (left/right to switch between buttons)
		action := s.buttonGroup.HandleKey(key)
		if action != "" {
			// Button was activated
			switch action {
			case "back":
				model.GoToPreviousStep()
				return model, nil
			case "customize":
				// User chose to customize
				model.customizeChoice = true
				model.GoToNextStep()
				return model, nil
			case "accept defaults":
				// User chose to accept defaults - skip to completion
				model.customizeChoice = false
				// Set default theme to "catppuccin"
				model.settingsValues["theme"] = "catppuccin"
				// Jump to completion step (last step)
				model.currentStep = len(model.steps) - 1
				return model, nil
			}
		}
		// Return even if no action, to allow button navigation (left/right)
		return model, nil
	}
	return model, nil
}

// wrapText wraps plain text to the specified width, breaking on word boundaries.
// Returns a slice of strings, one per line. If width <= 0, uses a default of 60.
func wrapText(text string, width int) []string {
	// Validate width
	if width <= 0 {
		width = 60
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	currentLine := ""

	for _, word := range words {
		testLine := currentLine
		if testLine != "" {
			testLine += " "
		}
		testLine += word

		if len(testLine) > width {
			if currentLine != "" {
				// Current line is full, save it and start new line with this word
				lines = append(lines, currentLine)
				currentLine = word
			} else {
				// Single word is longer than width - add it anyway
				lines = append(lines, word)
				currentLine = ""
			}
		} else {
			currentLine = testLine
		}
	}

	// Add the last line if it's not empty
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

// SourcesStepEnhanced is the enhanced sources step with interactive checkboxes
type SourcesStepEnhanced struct {
	checkboxList  *components.CheckboxList
	buttonGroup   *components.ButtonGroup
	hasBeenViewed bool // Track if step has been viewed to only reset once
}

func NewSourcesStepEnhanced() *SourcesStepEnhanced {
	return &SourcesStepEnhanced{
		buttonGroup: components.NewButtonGroup([]string{"Back", "Continue"}, 1),
	}
}

func (s *SourcesStepEnhanced) Name() string {
	return "Sources"
}

func (s *SourcesStepEnhanced) CanGoBack() bool {
	return true
}

func (s *SourcesStepEnhanced) CanGoNext() bool {
	return true
}

func (s *SourcesStepEnhanced) View(model *EnhancedOnboardingModel) string {
	// Reset button to default selection only on first view of this step
	if !s.hasBeenViewed {
		s.buttonGroup.ResetToDefault()
		s.hasBeenViewed = true
	}

	var result strings.Builder

	// Initialize checkbox list if not already done
	if s.checkboxList == nil {
		defaultSources := sources.DefaultSources()
		sourceOrder := []string{"Google Fonts", "Nerd Fonts", "Font Squirrel"}
		items := make([]components.CheckboxItem, 0, len(sourceOrder))

		for _, sourceName := range sourceOrder {
			if _, exists := defaultSources[sourceName]; exists {
				items = append(items, components.CheckboxItem{
					Label:   sourceName,
					Checked: model.sourceSelections[sourceName],
					Enabled: true,
				})
			}
		}

		s.checkboxList = components.NewCheckboxList(items)
		// Ensure checkbox list has focus, buttons don't
		s.checkboxList.SetFocus(true)
		s.buttonGroup.SetFocus(false)
	}

	result.WriteString("\n")
	result.WriteString(ui.PageTitle.Render("Font Sources"))
	result.WriteString("\n\n")
	result.WriteString(ui.Text.Render("Select which font sources you want to enable:"))
	result.WriteString("\n\n")

	// Render checkbox list
	result.WriteString(s.checkboxList.Render())
	result.WriteString("\n")

	// Show error if no sources selected
	checkedCount := 0
	for _, item := range s.checkboxList.Items {
		if item.Checked {
			checkedCount++
		}
	}
	if checkedCount == 0 {
		result.WriteString("\n")
		result.WriteString(ui.ErrorText.Render("⚠ At least one font source must be enabled"))
		result.WriteString("\n")
	}

	// Keyboard help
	commands := []string{
		ui.RenderKeyWithDescription("↑/↓", "Navigate"),
		ui.RenderKeyWithDescription("Space", "Toggle"),
		ui.RenderKeyWithDescription("Tab", "Switch"),
		ui.RenderKeyWithDescription("Enter", "Select"),
	}
	helpText := strings.Join(commands, "  ")
	result.WriteString(helpText)
	result.WriteString("\n\n")

	// Buttons
	result.WriteString(s.buttonGroup.Render())
	result.WriteString("\n")

	return result.String()
}

func (s *SourcesStepEnhanced) Update(model *EnhancedOnboardingModel, msg tea.Msg) (tea.Model, tea.Cmd) {
	if s.checkboxList == nil {
		// Initialize if needed
		defaultSources := sources.DefaultSources()
		sourceOrder := []string{"Google Fonts", "Nerd Fonts", "Font Squirrel"}
		items := make([]components.CheckboxItem, 0, len(sourceOrder))

		for _, sourceName := range sourceOrder {
			if _, exists := defaultSources[sourceName]; exists {
				items = append(items, components.CheckboxItem{
					Label:   sourceName,
					Checked: model.sourceSelections[sourceName],
					Enabled: true,
				})
			}
		}

		s.checkboxList = components.NewCheckboxList(items)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		// Handle backspace for back navigation
		if key == "backspace" && s.CanGoBack() {
			model.GoToPreviousStep()
			return model, nil
		}

		// Tab key switches focus between list and buttons
		if key == "tab" {
			if s.checkboxList.HasFocus {
				s.checkboxList.SetFocus(false)
				s.buttonGroup.SetFocus(true)
			} else {
				s.checkboxList.SetFocus(true)
				s.buttonGroup.SetFocus(false)
			}
			return model, nil
		}

		// Handle checkbox navigation/toggling (when checkbox list has focus)
		if s.checkboxList.HasFocus {
			// Check cursor position before handling key
			wasAtBottom := s.checkboxList.Cursor >= len(s.checkboxList.Items)-1

			// Check if this is a toggle action and validate
			if key == " " || key == "enter" {
				// Count currently checked items
				checkedCount := 0
				for _, item := range s.checkboxList.Items {
					if item.Checked {
						checkedCount++
					}
				}
				// Prevent unchecking if this is the last checked item
				if checkedCount <= 1 && s.checkboxList.Items[s.checkboxList.Cursor].Checked {
					// Can't uncheck the last item - show error or just ignore
					// For now, just ignore the toggle
					return model, nil
				}
			}

			handled := s.checkboxList.HandleKey(key)
			if handled {
				// Only move to buttons if we were already at the bottom and pressed down
				// (meaning HandleKey didn't move the cursor because it was already at the end)
				if (key == "down" || key == "j") && wasAtBottom {
					s.checkboxList.SetFocus(false)
					s.buttonGroup.SetFocus(true)
					return model, nil
				}
				return model, nil
			}
		}

		// If buttons have focus and user presses up, move focus back to checkbox list
		if s.buttonGroup.HasFocus && (key == "up" || key == "k") {
			s.buttonGroup.SetFocus(false)
			s.checkboxList.SetFocus(true)
			return model, nil
		}

		// Handle button navigation (when buttons have focus or left/right/tab pressed)
		if s.buttonGroup.HasFocus || key == "left" || key == "right" || key == "h" || key == "l" {
			// Switch focus to buttons when left/right is pressed
			if key == "left" || key == "right" || key == "h" || key == "l" {
				s.buttonGroup.SetFocus(true)
				s.checkboxList.SetFocus(false)
			}

			// Handle button navigation (left/right to switch between buttons)
			action := s.buttonGroup.HandleKey(key)
			if action != "" {
				// Button was activated
				switch action {
				case "back":
					model.GoToPreviousStep()
					return model, nil
				case "continue", "enter":
					// Validate at least one source is selected
					checkedCount := 0
					for _, item := range s.checkboxList.Items {
						if item.Checked {
							checkedCount++
						}
					}
					if checkedCount == 0 {
						// Can't continue without at least one source
						return model, nil
					}
					// Save source selections
					for _, item := range s.checkboxList.Items {
						model.sourceSelections[item.Label] = item.Checked
					}
					model.GoToNextStep()
					return model, nil
				}
			}
			// Return even if no action, to allow button navigation
			return model, nil
		}
	}

	return model, nil
}

// SettingsStepEnhanced is the enhanced settings step with interactive switches
type SettingsStepEnhanced struct {
	switches      []*components.Switch
	cursor        int
	buttonGroup   *components.ButtonGroup
	hasBeenViewed bool // Track if step has been viewed to only reset once
}

func NewSettingsStepEnhanced() *SettingsStepEnhanced {
	return &SettingsStepEnhanced{
		cursor:      0,
		buttonGroup: components.NewButtonGroup([]string{"Back", "Continue"}, 1), // Continue selected by default
	}
}

func (s *SettingsStepEnhanced) Name() string {
	return "Settings"
}

func (s *SettingsStepEnhanced) CanGoBack() bool {
	return true
}

func (s *SettingsStepEnhanced) CanGoNext() bool {
	return true
}

func (s *SettingsStepEnhanced) View(model *EnhancedOnboardingModel) string {
	// Reset button to default selection only on first view of this step
	if !s.hasBeenViewed {
		s.buttonGroup.ResetToDefault()
		s.hasBeenViewed = true
	}

	var result strings.Builder

	// Initialize switches if not already done
	if s.switches == nil {
		autoCheck := model.settingsValues["autoCheck"].(bool)
		usePopularitySort := model.settingsValues["usePopularitySort"].(bool)

		s.switches = []*components.Switch{
			components.NewSwitch(autoCheck),
			components.NewSwitchWithLabels("Popularity", "Alphabetical", usePopularitySort),
		}
	}

	result.WriteString("\n")
	result.WriteString(ui.PageTitle.Render("Default Settings"))
	result.WriteString("\n\n")
	result.WriteString(ui.Text.Render("Configure your default settings:\n"))
	result.WriteString("\n")

	// Settings list
	settings := []struct {
		name        string
		description string
		switchIndex int
	}{
		{
			name:        "Auto-check for updates",
			description: "FontGet will automatically check for new versions when you start the application.",
			switchIndex: 0,
		},
		{
			name:        "Sorting method",
			description: "When searching for fonts, results will be sorted by popularity first, then alphabetically.",
			switchIndex: 1,
		},
	}

	// Calculate maximum setting name width for alignment
	maxNameWidth := 0
	for _, setting := range settings {
		nameLen := len(setting.name)
		if nameLen > maxNameWidth {
			maxNameWidth = nameLen
		}
	}
	// Add some padding (4 spaces) between name and switch
	switchColumnStart := maxNameWidth + 4

	// Check if switches have focus (cursor should only show when switches are focused)
	switchesFocused := !s.buttonGroup.HasFocus

	for i, setting := range settings {
		// Build the line content
		var lineContent strings.Builder

		// Cursor indicator - only show when switches have focus
		if switchesFocused && i == s.cursor {
			lineContent.WriteString(ui.Cursor.Render("> "))
		} else {
			lineContent.WriteString("  ")
		}

		// Setting name (using accent2 color)
		settingName := ui.FormLabel.Render(setting.name)
		lineContent.WriteString(settingName)

		// Calculate padding needed to align switches
		// Account for ANSI escape codes in styled text by using plain length
		nameDisplayLen := len(setting.name)
		paddingNeeded := switchColumnStart - nameDisplayLen
		if paddingNeeded > 0 {
			lineContent.WriteString(strings.Repeat(" ", paddingNeeded))
		}

		// Switch aligned in column
		lineContent.WriteString(s.switches[setting.switchIndex].Render())

		// Apply background highlighting if this is the selected item
		line := lineContent.String()
		if switchesFocused && i == s.cursor {
			line = ui.CheckboxItemSelected.Render(line)
		}

		result.WriteString(line)
		result.WriteString("\n")

		// Add blank line between settings (including after last one)
		result.WriteString("\n")

	}

	// Keyboard help
	commands := []string{
		ui.RenderKeyWithDescription("↑/↓", "Navigate"),
		ui.RenderKeyWithDescription("Space", "Toggle"),
		ui.RenderKeyWithDescription("Tab", "Switch"),
		ui.RenderKeyWithDescription("Enter", "Select"),
	}
	helpText := strings.Join(commands, "  ")
	result.WriteString(helpText)
	result.WriteString("\n\n")

	// Buttons
	result.WriteString(s.buttonGroup.Render())
	result.WriteString("\n")

	return result.String()
}

func (s *SettingsStepEnhanced) Update(model *EnhancedOnboardingModel, msg tea.Msg) (tea.Model, tea.Cmd) {
	// Initialize switches if needed
	if s.switches == nil {
		autoCheck := model.settingsValues["autoCheck"].(bool)
		usePopularitySort := model.settingsValues["usePopularitySort"].(bool)

		s.switches = []*components.Switch{
			components.NewSwitch(autoCheck),
			components.NewSwitchWithLabels("Popularity", "Alphabetical", usePopularitySort),
		}
	}

	// Track if we're focused on switches (cursor active) or buttons
	switchesFocused := !s.buttonGroup.HasFocus

	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		// Handle backspace for back navigation
		if key == "backspace" && s.CanGoBack() {
			model.GoToPreviousStep()
			return model, nil
		}

		// Tab key switches focus between switches and buttons
		if key == "tab" {
			if switchesFocused {
				s.buttonGroup.SetFocus(true)
			} else {
				s.buttonGroup.SetFocus(false)
			}
			return model, nil
		}

		// Handle switch navigation/toggling (when switches have focus)
		if switchesFocused {
			switch key {
			case "up", "k":
				if s.cursor > 0 {
					s.cursor--
				}
				return model, nil
			case "down", "j":
				if s.cursor < len(s.switches)-1 {
					s.cursor++
				} else {
					// Already at bottom, pressing down moves to buttons
					s.buttonGroup.SetFocus(true)
				}
				return model, nil
			case "left", "right", "h", "l", " ", "enter":
				if s.cursor >= 0 && s.cursor < len(s.switches) {
					// Toggle the switch
					s.switches[s.cursor].HandleKey(key)
					return model, nil
				}
			}
		}

		// If buttons have focus and user presses up, move focus back to switches
		if s.buttonGroup.HasFocus && (key == "up" || key == "k") {
			s.buttonGroup.SetFocus(false)
			// Focus returns to switches (cursor stays where it was)
			return model, nil
		}

		// Handle button navigation (when buttons have focus or left/right/tab pressed)
		if s.buttonGroup.HasFocus || key == "left" || key == "right" || key == "h" || key == "l" {
			// Switch focus to buttons when left/right is pressed
			if key == "left" || key == "right" || key == "h" || key == "l" {
				s.buttonGroup.SetFocus(true)
			}

			action := s.buttonGroup.HandleKey(key)
			if action != "" {
				switch action {
				case "back":
					model.GoToPreviousStep()
					return model, nil
				case "continue", "enter":
					// Save settings values
					model.settingsValues["autoCheck"] = s.switches[0].Value
					model.settingsValues["checkForUpdates"] = s.switches[0].Value // Also save to checkForUpdates key
					model.settingsValues["usePopularitySort"] = s.switches[1].Value
					model.GoToNextStep()
					return model, nil
				}
			}
			return model, nil
		}
	}

	return model, nil
}

// LayoutConfig holds configuration for layout calculations (copied from cmd/theme_layout.go)
type LayoutConfig struct {
	TerminalWidth  int
	TerminalHeight int
	HeaderHeight   int
	FooterHeight   int
	MarginWidth    int
	SeparatorWidth int
}

// PanelLayout holds calculated panel dimensions (copied from cmd/theme_layout.go)
type PanelLayout struct {
	LeftWidth       int
	RightWidth      int
	PanelHeight     int
	AvailableWidth  int
	AvailableHeight int
}

// CalculatePanelLayout calculates panel dimensions based on terminal size and layout config (copied from cmd/theme_layout.go)
func CalculatePanelLayout(config LayoutConfig) PanelLayout {
	// Calculate available space
	marginWidth := config.MarginWidth
	if marginWidth == 0 {
		marginWidth = 2 // Default: 1 char on each side
	}

	separatorWidth := config.SeparatorWidth
	if separatorWidth == 0 {
		separatorWidth = 1 // Default separator width
	}

	availableWidth := config.TerminalWidth - marginWidth
	availableHeight := config.TerminalHeight - config.HeaderHeight - config.FooterHeight

	// Ensure minimum dimensions
	if availableWidth < 40 {
		availableWidth = 40
	}
	if availableHeight < 10 {
		availableHeight = 10
	}

	// Calculate panel widths (30/70 split accounting for separator)
	panelAreaWidth := availableWidth - separatorWidth
	if panelAreaWidth < 0 {
		panelAreaWidth = 0
	}

	// 30% for left, 70% for right
	leftWidth := int(float64(panelAreaWidth) * 0.3)
	rightWidth := panelAreaWidth - leftWidth

	// Ensure minimum panel widths
	if leftWidth < 20 {
		leftWidth = 20
	}
	if rightWidth < 20 {
		rightWidth = 20
	}

	// Safety check: ensure total doesn't exceed terminal width
	maxTotalWidth := config.TerminalWidth
	if leftWidth+rightWidth+separatorWidth > maxTotalWidth {
		panelAreaWidth = maxTotalWidth - separatorWidth
		if panelAreaWidth < 0 {
			panelAreaWidth = 0
		}
		leftWidth = panelAreaWidth / 2
		rightWidth = panelAreaWidth - leftWidth
	}

	return PanelLayout{
		LeftWidth:       leftWidth,
		RightWidth:      rightWidth,
		PanelHeight:     availableHeight,
		AvailableWidth:  availableWidth,
		AvailableHeight: availableHeight,
	}
}

// trimContent removes trailing newlines and whitespace from content (copied from cmd/theme_layout.go)
func trimContent(content string) string {
	content = strings.TrimRight(content, "\n")
	lines := strings.Split(content, "\n")
	trimmed := make([]string, len(lines))
	for i, line := range lines {
		trimmed[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(trimmed, "\n")
}

// renderCombinedPanels renders two panels side-by-side with a shared border (copied from cmd/theme_layout.go)
func renderCombinedPanels(title string, leftWidth, rightWidth, height int, leftContent, rightContent string, _ lipgloss.Style, separatorColor, borderColor lipgloss.Color, titleStyle lipgloss.Style) string {
	// Guard minimums
	if leftWidth < 4 {
		leftWidth = 4
	}
	if rightWidth < 4 {
		rightWidth = 4
	}
	if height < 3 {
		height = 3
	}

	leftContentWidth := leftWidth - 1
	if leftContentWidth < 0 {
		leftContentWidth = 0
	}
	rightContentWidth := rightWidth - 1
	if rightContentWidth < 0 {
		rightContentWidth = 0
	}
	contentHeight := height - 2
	if contentHeight < 1 {
		contentHeight = 1
	}

	leftTrimmed := trimContent(leftContent)
	rightTrimmed := trimContent(rightContent)

	leftConstrained := lipgloss.NewStyle().
		Height(contentHeight).
		Render(leftTrimmed)

	rightConstrained := lipgloss.NewStyle().
		Height(contentHeight).
		Render(rightTrimmed)

	topLeft := "╭"
	topRight := "╮"
	bottomLeft := "╰"
	bottomRight := "╯"
	topTee := "┬"
	bottomTee := "┴"
	vertical := "│"
	horizontal := "─"

	borderCharStyle := lipgloss.NewStyle().Foreground(borderColor)
	separatorStyle := lipgloss.NewStyle().Foreground(separatorColor)

	topLeftChar := borderCharStyle.Render(topLeft)
	topRightChar := borderCharStyle.Render(topRight)
	bottomLeftChar := borderCharStyle.Render(bottomLeft)
	bottomRightChar := borderCharStyle.Render(bottomRight)
	topTeeChar := borderCharStyle.Render(topTee)
	bottomTeeChar := borderCharStyle.Render(bottomTee)
	leftBorderChar := borderCharStyle.Render(vertical)
	rightBorderChar := borderCharStyle.Render(vertical)
	separatorChar := separatorStyle.Render(vertical)
	horizontalChar := borderCharStyle.Render(horizontal)

	titleRendered := titleStyle.Render(title)
	titleWidth := lipgloss.Width(titleRendered)

	totalBorderWidth := leftWidth + 1 + rightWidth

	leftInner := leftWidth - 1
	if leftInner < 0 {
		leftInner = 0
	}
	rightInner := rightWidth - 1
	if rightInner < 0 {
		rightInner = 0
	}

	titleSectionWidth := 1 + 1 + titleWidth + 1 + 1
	remainingLeft := leftInner - titleSectionWidth
	if remainingLeft < 0 {
		remainingLeft = 0
	}

	topBorderLeft := topLeftChar + horizontalChar + " " + titleRendered + " " + horizontalChar + strings.Repeat(horizontalChar, remainingLeft)

	leftSegmentWidth := lipgloss.Width(topBorderLeft)
	if leftSegmentWidth != leftWidth {
		actualRemaining := leftWidth - lipgloss.Width(topLeftChar+horizontalChar+" "+titleRendered+" "+horizontalChar)
		if actualRemaining < 0 {
			actualRemaining = 0
		}
		topBorderLeft = topLeftChar + horizontalChar + " " + titleRendered + " " + horizontalChar + strings.Repeat(horizontalChar, actualRemaining)
	}

	topBorderRight := strings.Repeat(horizontalChar, rightInner) + topRightChar
	topBorder := topBorderLeft + topTeeChar + topBorderRight

	actualWidth := lipgloss.Width(topBorder)
	if actualWidth != totalBorderWidth {
		adjust := totalBorderWidth - actualWidth
		newRightInner := rightInner + adjust
		if newRightInner < 0 {
			newRightInner = 0
		}
		topBorderRight = strings.Repeat(horizontalChar, newRightInner) + topRightChar
		topBorder = topBorderLeft + topTeeChar + topBorderRight
	}

	bottomBorder := bottomLeftChar + strings.Repeat(horizontalChar, leftWidth-1) + bottomTeeChar + strings.Repeat(horizontalChar, rightWidth-1) + bottomRightChar

	leftLines := strings.Split(strings.TrimRight(leftConstrained, "\n"), "\n")
	rightLines := strings.Split(strings.TrimRight(rightConstrained, "\n"), "\n")

	maxLines := contentHeight
	if len(leftLines) < maxLines {
		leftLines = append(leftLines, make([]string, maxLines-len(leftLines))...)
	}
	if len(rightLines) < maxLines {
		rightLines = append(rightLines, make([]string, maxLines-len(rightLines))...)
	}
	if len(leftLines) > maxLines {
		leftLines = leftLines[:maxLines]
	}
	if len(rightLines) > maxLines {
		rightLines = rightLines[:maxLines]
	}

	var middleLines []string
	for i := 0; i < maxLines; i++ {
		leftLine := leftLines[i]
		rightLine := rightLines[i]

		leftLineWidth := lipgloss.Width(leftLine)
		rightLineWidth := lipgloss.Width(rightLine)

		var leftPadded string
		if leftLineWidth < leftContentWidth {
			leftPadded = leftLine + strings.Repeat(" ", leftContentWidth-leftLineWidth)
		} else if leftLineWidth > leftContentWidth {
			leftPadded = lipgloss.NewStyle().Width(leftContentWidth).MaxWidth(leftContentWidth).Render(leftLine)
		} else {
			leftPadded = leftLine
		}

		var rightPadded string
		if rightLineWidth < rightContentWidth {
			rightPadded = rightLine + strings.Repeat(" ", rightContentWidth-rightLineWidth)
		} else if rightLineWidth > rightContentWidth {
			rightPadded = lipgloss.NewStyle().Width(rightContentWidth).MaxWidth(rightContentWidth).Render(rightLine)
		} else {
			rightPadded = rightLine
		}

		middleLine := leftBorderChar + leftPadded + separatorChar + rightPadded + rightBorderChar
		middleLines = append(middleLines, middleLine)
	}

	var result strings.Builder
	result.WriteString(topBorder)
	result.WriteString("\n")
	result.WriteString(strings.Join(middleLines, "\n"))
	result.WriteString("\n")
	result.WriteString(bottomBorder)

	return result.String()
}

// MenuLine represents a single line in the theme menu (copied from cmd/theme.go)
type MenuLine struct {
	Type       string // "header_blank", "header_text", "header_separator", or "theme"
	Content    string
	ThemeIndex int
	IsSelected bool
}

// ThemeSelectionStepEnhanced is the theme selection step using the full theme picker TUI
type ThemeSelectionStepEnhanced struct {
	themes        []ui.ThemeOption
	selectedIndex int
	scrollOffset  int
	preview       *components.PreviewModel
	buttonGroup   *components.ButtonGroup
	navButtons    *components.ButtonGroup // Back/Continue buttons
	hasBeenViewed bool
	width         int
	height        int
}

func NewThemeSelectionStepEnhanced() (*ThemeSelectionStepEnhanced, error) {
	// Default to "catppuccin" instead of "system"
	defaultTheme := "catppuccin"

	// Get theme options
	options, err := ui.GetThemeOptions(defaultTheme)
	if err != nil {
		return nil, fmt.Errorf("failed to discover themes: %w", err)
	}

	// Find catppuccin selection index (or first theme if not found)
	selectedIndex := 0
	for i, option := range options {
		if option.ThemeName == "catppuccin" {
			selectedIndex = i
			break
		}
		if option.IsSelected {
			selectedIndex = i
		}
	}

	// Create preview model
	preview := components.NewPreviewModel()

	// Load initial theme for preview
	if err := preview.LoadTheme(options[selectedIndex].ThemeName); err != nil {
		// Log error but continue
	}

	// Create button group for theme options
	buttonTexts := make([]string, len(options))
	for i, option := range options {
		prefix := "  "
		if i == selectedIndex {
			prefix = "✔️ "
		}
		buttonTexts[i] = fmt.Sprintf("%s %s", prefix, option.DisplayName)
	}

	buttons := components.NewButtonGroup(buttonTexts, selectedIndex)
	buttons.SetFocus(true)

	// Create navigation buttons (Back/Continue)
	navButtons := components.NewButtonGroup([]string{"Back", "Continue"}, 1)
	navButtons.SetFocus(false)

	return &ThemeSelectionStepEnhanced{
		themes:        options,
		selectedIndex: selectedIndex,
		scrollOffset:  0,
		preview:       preview,
		buttonGroup:   buttons,
		navButtons:    navButtons,
		width:         80,
		height:        24,
	}, nil
}

func (s *ThemeSelectionStepEnhanced) Name() string {
	return "Theme Selection"
}

func (s *ThemeSelectionStepEnhanced) CanGoBack() bool {
	return true
}

func (s *ThemeSelectionStepEnhanced) CanGoNext() bool {
	return true
}

func (s *ThemeSelectionStepEnhanced) View(model *EnhancedOnboardingModel) string {
	if !s.hasBeenViewed {
		s.hasBeenViewed = true
	}

	// Update width/height from model
	s.width = model.width
	s.height = model.height

	// Build title and footer
	titleText := "Theme Selection"
	var commands []string
	commands = append(commands, ui.RenderKeyWithDescription("↑/↓", "Navigate"))
	commands = append(commands, ui.RenderKeyWithDescription("Tab", "Switch"))
	commands = append(commands, ui.RenderKeyWithDescription("Enter", "Select"))
	help := strings.Join(commands, "  ")

	headerHeight := 0
	footerHeight := 3 // Help text (1) + blank line (1) + navigation buttons (1)

	// Calculate layout
	layoutConfig := LayoutConfig{
		TerminalWidth:  model.width,
		TerminalHeight: model.height,
		HeaderHeight:   headerHeight,
		FooterHeight:   footerHeight,
		MarginWidth:    2,
		SeparatorWidth: 1,
	}

	layout := CalculatePanelLayout(layoutConfig)

	// Build left panel content (theme list)
	leftContent := s.renderLeftPanelContent(layout.LeftWidth, layout.PanelHeight)

	// Build right panel content (preview)
	rightContent := s.renderRightPanelContent(layout.RightWidth, layout.PanelHeight)

	// Render combined panels
	colors := ui.GetCurrentColors()
	separatorColor := lipgloss.Color(colors.Placeholders)
	borderColor := lipgloss.Color(colors.Placeholders)

	combined := renderCombinedPanels(
		titleText,
		layout.LeftWidth,
		layout.RightWidth,
		layout.PanelHeight,
		leftContent,
		rightContent,
		ui.CardBorder,
		separatorColor,
		borderColor,
		ui.PageTitle,
	)

	// Add margins
	margin := 1
	marginedCombined := lipgloss.NewStyle().
		PaddingLeft(margin).
		PaddingRight(margin).
		Render(combined)

	// Footer: help text + navigation buttons
	var footer strings.Builder
	footer.WriteString(help)
	footer.WriteString("\n\n")
	footer.WriteString(s.navButtons.Render())

	var content strings.Builder
	content.WriteString(marginedCombined)
	content.WriteString("\n")
	content.WriteString(footer.String())

	return lipgloss.NewStyle().
		Width(model.width).
		MaxWidth(model.width).
		Render(content.String())
}

func (s *ThemeSelectionStepEnhanced) Update(model *EnhancedOnboardingModel, msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle preview animation updates
	if cmd, shouldRedraw := s.preview.Update(msg); shouldRedraw {
		return model, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		model.width = msg.Width
		model.height = msg.Height
		s.width = msg.Width
		s.height = msg.Height
		s.adjustScrollForSelection()
		return model, nil

	case tea.KeyMsg:
		key := msg.String()

		// Handle backspace for back navigation
		if key == "backspace" && s.CanGoBack() {
			model.GoToPreviousStep()
			return model, nil
		}

		// Tab switches between theme list and navigation buttons
		if key == "tab" {
			if s.buttonGroup.HasFocus {
				s.buttonGroup.SetFocus(false)
				s.navButtons.SetFocus(true)
			} else {
				s.buttonGroup.SetFocus(true)
				s.navButtons.SetFocus(false)
			}
			return model, nil
		}

		// If navigation buttons have focus and user presses up, move focus back to theme list
		if s.navButtons.HasFocus && (key == "up" || key == "k") {
			s.navButtons.SetFocus(false)
			s.buttonGroup.SetFocus(true)
			return model, nil
		}

		// Handle navigation buttons
		if s.navButtons.HasFocus {
			action := s.navButtons.HandleKey(key)
			if action != "" {
				switch action {
				case "back":
					model.GoToPreviousStep()
					return model, nil
				case "continue", "enter":
					// Store selected theme and apply it immediately
					selectedTheme := s.themes[s.selectedIndex].ThemeName
					model.settingsValues["theme"] = selectedTheme

					// Apply the theme immediately so subsequent steps use it
					// Save to config temporarily so InitThemeManager() will load it
					appConfig := config.GetUserPreferences()
					if appConfig != nil {
						appConfig.Theme = selectedTheme
						// Save to config so theme manager can reload it
						if err := config.SaveUserPreferences(appConfig); err == nil {
							// Reload theme manager to pick up the new theme from config
							if err := ui.InitThemeManager(); err == nil {
								// Re-initialize styles with the new theme
								// This updates all global style variables (SuccessText, InfoText, etc.)
								if err := ui.InitStyles(); err != nil {
									// Log error but continue - theme will be applied on next app start
								}
							}
						}
					}

					model.GoToNextStep()
					return model, nil
				}
			}
			return model, nil
		}

		// Handle theme navigation (when theme list has focus)
		if s.buttonGroup.HasFocus {
			switch key {
			case "up", "k":
				if s.selectedIndex > 0 {
					s.selectedIndex--
					s.buttonGroup.Selected = s.selectedIndex
					_ = s.preview.LoadTheme(s.themes[s.selectedIndex].ThemeName)
					s.adjustScrollForSelection()
				}
				return model, nil

			case "down", "j":
				if s.selectedIndex < len(s.themes)-1 {
					s.selectedIndex++
					s.buttonGroup.Selected = s.selectedIndex
					_ = s.preview.LoadTheme(s.themes[s.selectedIndex].ThemeName)
					s.adjustScrollForSelection()
				} else {
					// Already at bottom, pressing down moves to navigation buttons
					s.buttonGroup.SetFocus(false)
					s.navButtons.SetFocus(true)
				}
				return model, nil

			case "enter":
				// Store selected theme and apply it immediately
				selectedTheme := s.themes[s.selectedIndex].ThemeName
				model.settingsValues["theme"] = selectedTheme

				// Apply the theme immediately so subsequent steps use it
				// Save to config temporarily so InitThemeManager() will load it
				appConfig := config.GetUserPreferences()
				if appConfig != nil {
					appConfig.Theme = selectedTheme
					// Save to config so theme manager can reload it
					if err := config.SaveUserPreferences(appConfig); err == nil {
						// Reload theme manager to pick up the new theme from config
						if err := ui.InitThemeManager(); err == nil {
							// Re-initialize styles with the new theme
							// This updates all global style variables (SuccessText, InfoText, etc.)
							if err := ui.InitStyles(); err != nil {
								// Log error but continue - theme will be applied on next app start
							}
						}
					}
				}

				model.GoToNextStep()
				return model, nil
			}
		}
	}

	return model, nil
}

// buildAllMenuLines builds all menu lines (headers + themes) - copied from cmd/theme.go
func (s *ThemeSelectionStepEnhanced) buildAllMenuLines(contentWidth int) []MenuLine {
	var allLines []MenuLine
	colors := ui.GetCurrentColors()
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.Secondary)).
		Bold(true)
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Placeholders))

	darkHeaderShown := false
	lightHeaderShown := false

	for i, option := range s.themes {
		// Add "DARK THEMES" header before first dark theme
		if i > 0 && s.themes[i-1].Style == "" && option.Style == "dark" && !darkHeaderShown {
			allLines = append(allLines, MenuLine{Type: "header_blank", Content: "", ThemeIndex: -1})
			allLines = append(allLines, MenuLine{Type: "header_text", Content: headerStyle.Render("DARK THEMES"), ThemeIndex: -1})
			allLines = append(allLines, MenuLine{Type: "header_separator", Content: separatorStyle.Render(strings.Repeat("─", contentWidth)), ThemeIndex: -1})
			darkHeaderShown = true
		}

		// Add "LIGHT THEMES" header before first light theme
		if i > 0 && s.themes[i-1].Style == "dark" && option.Style == "light" && !lightHeaderShown {
			allLines = append(allLines, MenuLine{Type: "header_blank", Content: "", ThemeIndex: -1})
			allLines = append(allLines, MenuLine{Type: "header_text", Content: headerStyle.Render("LIGHT THEMES"), ThemeIndex: -1})
			allLines = append(allLines, MenuLine{Type: "header_separator", Content: separatorStyle.Render(strings.Repeat("─", contentWidth)), ThemeIndex: -1})
			lightHeaderShown = true
		}

		// Add theme button line
		buttonText := option.DisplayName
		button := components.Button{
			Text:     buttonText,
			Selected: (i == s.selectedIndex && s.buttonGroup.HasFocus),
		}
		rendered := button.RenderFullWidth(contentWidth)
		allLines = append(allLines, MenuLine{
			Type:       "theme",
			Content:    rendered,
			ThemeIndex: i,
			IsSelected: (i == s.selectedIndex && s.buttonGroup.HasFocus),
		})
	}

	return allLines
}

// findThemeLineIndex finds the line index for a given theme index
func (s *ThemeSelectionStepEnhanced) findThemeLineIndex(themeIndex int, allLines []MenuLine) int {
	for i, line := range allLines {
		if line.ThemeIndex == themeIndex {
			return i
		}
	}
	return -1
}

// adjustScrollForSelection adjusts scrollOffset to keep selectedIndex visible
func (s *ThemeSelectionStepEnhanced) adjustScrollForSelection() *ThemeSelectionStepEnhanced {
	if len(s.themes) == 0 || s.height == 0 {
		return s
	}

	layoutConfig := LayoutConfig{
		TerminalWidth:  s.width,
		TerminalHeight: s.height,
		HeaderHeight:   0,
		FooterHeight:   3,
		MarginWidth:    2,
		SeparatorWidth: 1,
	}
	layout := CalculatePanelLayout(layoutConfig)

	availableHeight := layout.PanelHeight - 2 - 2
	if availableHeight < 1 {
		availableHeight = 1
	}

	contentWidth := 30
	if s.width > 0 {
		contentWidth = s.width - 3
		if contentWidth < 10 {
			contentWidth = 10
		}
	}
	allLines := s.buildAllMenuLines(contentWidth)
	selectedLineIndex := s.findThemeLineIndex(s.selectedIndex, allLines)
	if selectedLineIndex < 0 {
		selectedLineIndex = 0
	}

	isCurrentlyVisible := selectedLineIndex >= s.scrollOffset && selectedLineIndex < s.scrollOffset+availableHeight

	if !isCurrentlyVisible {
		if selectedLineIndex < availableHeight {
			s.scrollOffset = 0
		} else {
			s.scrollOffset = selectedLineIndex - availableHeight + 1
			if s.scrollOffset < 0 {
				s.scrollOffset = 0
			}
		}
	}

	maxScrollOffset := len(allLines) - 1
	if s.scrollOffset > maxScrollOffset {
		s.scrollOffset = maxScrollOffset
	}
	if s.scrollOffset < 0 {
		s.scrollOffset = 0
	}

	return s
}

// renderLeftPanelContent renders the left panel content
func (s *ThemeSelectionStepEnhanced) renderLeftPanelContent(width, panelHeight int) string {
	contentWidth := width - 3
	if contentWidth < 10 {
		contentWidth = 10
	}

	availableHeight := panelHeight - 2 - 2
	if availableHeight < 1 {
		availableHeight = 1
	}

	allLines := s.buildAllMenuLines(contentWidth)

	if s.scrollOffset < 0 {
		s.scrollOffset = 0
	}
	if s.scrollOffset >= len(allLines) {
		s.scrollOffset = len(allLines) - 1
		if s.scrollOffset < 0 {
			s.scrollOffset = 0
		}
	}

	startLine := s.scrollOffset
	endLine := startLine + availableHeight
	if endLine > len(allLines) {
		endLine = len(allLines)
	}

	var visibleLines []string
	for i := startLine; i < endLine; i++ {
		visibleLines = append(visibleLines, allLines[i].Content)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, visibleLines...)
	return lipgloss.NewStyle().Padding(1, 1).Render(content)
}

// renderRightPanelContent renders the right panel content
func (s *ThemeSelectionStepEnhanced) renderRightPanelContent(width, _ int) string {
	contentWidth := width - 3
	if contentWidth < 0 {
		contentWidth = 0
	}

	preview := s.preview.View(contentWidth)
	return lipgloss.NewStyle().Padding(1, 1).Render(preview)
}

// CompletionStepEnhanced is the enhanced completion step with OK button
type CompletionStepEnhanced struct {
	buttonGroup *components.ButtonGroup
}

func NewCompletionStepEnhanced() *CompletionStepEnhanced {
	group := components.NewButtonGroup([]string{"OK"}, 0)
	group.SetFocus(true) // OK button has focus by default
	return &CompletionStepEnhanced{
		buttonGroup: group,
	}
}

func (s *CompletionStepEnhanced) Name() string {
	return "Completion"
}

func (s *CompletionStepEnhanced) CanGoBack() bool {
	return true
}

func (s *CompletionStepEnhanced) CanGoNext() bool {
	return false
}

func (s *CompletionStepEnhanced) View(model *EnhancedOnboardingModel) string {
	// Reset button to default selection when viewing this step
	s.buttonGroup.ResetToDefault()
	// Ensure button has focus so it shows as selected
	s.buttonGroup.SetFocus(true)

	var result strings.Builder

	result.WriteString("\n")
	result.WriteString(ui.PageTitle.Render("Setup Complete!"))
	result.WriteString("\n\n")
	if !model.customizeChoice {
		result.WriteString(ui.Text.Render("You're all set. FontGet is now configured with default settings."))
	} else {
		result.WriteString(ui.Text.Render("You're all set to start using FontGet."))
	}
	result.WriteString("\n\n")
	result.WriteString(ui.FormLabel.Render("Try these commands to get started:"))
	result.WriteString("\n")
	result.WriteString(fmt.Sprintf("  %s  %s\n", ui.Text.Render("- fontget search <font-name>"), ui.InfoText.Render("// Searches for fonts")))
	result.WriteString(fmt.Sprintf("  %s  %s\n", ui.Text.Render("- fontget list"), ui.InfoText.Render("// Lists installed fonts")))
	result.WriteString(fmt.Sprintf("  %s  %s\n", ui.Text.Render("- fontget add <font-id>"), ui.InfoText.Render("// Installs a font")))
	result.WriteString(fmt.Sprintf("  %s  %s\n", ui.Text.Render("- fontget --help"), ui.InfoText.Render("// Shows all available commands")))
	result.WriteString("\n")
	result.WriteString("To run this wizard again, run 'fontget --wizard'\n")
	result.WriteString("\n")

	// Button
	result.WriteString(s.buttonGroup.Render())
	result.WriteString("\n")

	return result.String()
}

func (s *CompletionStepEnhanced) Update(model *EnhancedOnboardingModel, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Completion step only has OK button, so buttons always have focus
		s.buttonGroup.SetFocus(true)

		action := s.buttonGroup.HandleKey(msg.String())
		if action == "ok" || action == "enter" || msg.String() == "enter" {
			// Save all selections
			if err := model.SaveSelections(); err != nil {
				// Log error but continue
			}
			// Mark onboarding as successfully completed
			model.onboardingCompleted = true
			model.quitting = true
			return model, tea.Quit
		}
	}
	return model, nil
}
