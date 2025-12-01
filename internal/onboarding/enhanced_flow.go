package onboarding

import (
	"fmt"
	"strings"

	"fontget/internal/components"
	"fontget/internal/config"
	"fontget/internal/sources"
	"fontget/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
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
		"autoCheck":         defaults.Update.AutoCheck,
		"autoUpdate":        defaults.Update.AutoUpdate,
		"usePopularitySort": defaults.Configuration.UsePopularitySort,
	}

	model := &EnhancedOnboardingModel{
		currentStep:      0,
		sourceSelections: sourceSelections,
		settingsValues:   settingsValues,
		width:            80,
		height:           24,
	}

	// Create steps
	model.steps = []OnboardingStepInterface{
		NewWelcomeStepEnhanced(),
		NewLicenseStepEnhanced(),
		NewSourcesStepEnhanced(),
		NewSettingsStepEnhanced(),
		NewCompletionStepEnhanced(),
	}

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
		case "ctrl+c", "q":
			// User quit early - don't mark as completed
			m.quitting = true
			m.onboardingCompleted = false
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	}

	// Delegate to current step
	if m.currentStep >= 0 && m.currentStep < len(m.steps) {
		return m.steps[m.currentStep].Update(&m, msg)
	}

	return m, nil
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
		case *LicenseStepEnhanced:
			step.hasBeenViewed = false
		case *SettingsStepEnhanced:
			step.hasBeenViewed = false
		}
	}
}

// SaveSelections saves all selections to config files
func (m *EnhancedOnboardingModel) SaveSelections() error {
	// Save source selections
	for name, enabled := range m.sourceSelections {
		// Only update if different from default
		defaultSources := sources.DefaultSources()
		if info, exists := defaultSources[name]; exists && info.Enabled != enabled {
			// Source state changed - we'll handle this in the sources step
			// For now, we just track the selection
		}
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
	if autoCheck, ok := m.settingsValues["autoCheck"].(bool); ok {
		appConfig.Update.AutoCheck = autoCheck
	}
	if autoUpdate, ok := m.settingsValues["autoUpdate"].(bool); ok {
		appConfig.Update.AutoUpdate = autoUpdate
	}
	if usePopularitySort, ok := m.settingsValues["usePopularitySort"].(bool); ok {
		appConfig.Configuration.UsePopularitySort = usePopularitySort
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
	result.WriteString(ui.Text.Render("This is your first time using FontGet. Let's get you set up."))
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

// LicenseStepEnhanced is the enhanced license step with navigation buttons
type LicenseStepEnhanced struct {
	buttonGroup   *components.ButtonGroup
	confirmed     bool
	hasBeenViewed bool // Track if step has been viewed to only reset once
}

func NewLicenseStepEnhanced() *LicenseStepEnhanced {
	group := components.NewButtonGroup([]string{"Back", "Next"}, 1) // Next selected by default
	group.SetFocus(true)                                            // Buttons have focus by default (button-only screen)
	return &LicenseStepEnhanced{
		buttonGroup: group,
	}
}

func (s *LicenseStepEnhanced) Name() string {
	return "License Agreement"
}

func (s *LicenseStepEnhanced) CanGoBack() bool {
	return true
}

func (s *LicenseStepEnhanced) CanGoNext() bool {
	return s.confirmed
}

func (s *LicenseStepEnhanced) View(model *EnhancedOnboardingModel) string {
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
	result.WriteString(ui.WarningText.Render("IMPORTANT: By using FontGet, you acknowledge and agree to the following:"))
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

	// Buttons
	result.WriteString(s.buttonGroup.Render())
	result.WriteString("\n")

	return result.String()
}

func (s *LicenseStepEnhanced) Update(model *EnhancedOnboardingModel, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

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
			case "next", "enter":
				s.confirmed = true
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
		ui.RenderKeyWithDescription("Space/Enter", "Toggle"),
		ui.RenderKeyWithDescription("Tab", "Switch"),
		ui.RenderKeyWithDescription("←/→", "Buttons"),
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
		autoUpdate := model.settingsValues["autoUpdate"].(bool)
		usePopularitySort := model.settingsValues["usePopularitySort"].(bool)

		s.switches = []*components.Switch{
			components.NewSwitch(autoCheck),
			components.NewSwitch(autoUpdate),
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
			name:        "Auto-install updates",
			description: "When updates are available, FontGet will notify you but require manual installation.",
			switchIndex: 1,
		},
		{
			name:        "Sorting method",
			description: "When searching for fonts, results will be sorted by popularity first, then alphabetically.",
			switchIndex: 2,
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

	for i, setting := range settings {
		// Cursor indicator
		if i == s.cursor {
			result.WriteString(ui.CheckboxCursor.Render("> "))
		} else {
			result.WriteString("  ")
		}

		// Setting name (using accent2 color)
		settingName := ui.FormLabel.Render(setting.name)
		result.WriteString(settingName)

		// Calculate padding needed to align switches
		// Account for ANSI escape codes in styled text by using plain length
		nameDisplayLen := len(setting.name)
		paddingNeeded := switchColumnStart - nameDisplayLen
		if paddingNeeded > 0 {
			result.WriteString(strings.Repeat(" ", paddingNeeded))
		}

		// Switch aligned in column
		result.WriteString(s.switches[setting.switchIndex].Render())
		result.WriteString("\n")

		// Add blank line between settings (including after last one)

		result.WriteString("\n")

	}

	// Keyboard help
	commands := []string{
		ui.RenderKeyWithDescription("↑/↓", "Navigate"),
		ui.RenderKeyWithDescription("←/→", "Toggle"),
		ui.RenderKeyWithDescription("Space", "Toggle"),
		ui.RenderKeyWithDescription("Tab", "Switch"),
		ui.RenderKeyWithDescription("i", "Info"),
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
		autoUpdate := model.settingsValues["autoUpdate"].(bool)
		usePopularitySort := model.settingsValues["usePopularitySort"].(bool)

		s.switches = []*components.Switch{
			components.NewSwitch(autoCheck),
			components.NewSwitch(autoUpdate),
			components.NewSwitchWithLabels("Popularity", "Alphabetical", usePopularitySort),
		}
	}

	// Track if we're focused on switches (cursor active) or buttons
	switchesFocused := !s.buttonGroup.HasFocus

	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

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
					model.settingsValues["autoUpdate"] = s.switches[1].Value
					model.settingsValues["usePopularitySort"] = s.switches[2].Value
					model.GoToNextStep()
					return model, nil
				}
			}
			return model, nil
		}
	}

	return model, nil
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
	result.WriteString(ui.SuccessText.Render("Setup complete!"))
	result.WriteString("\n\n")
	result.WriteString(ui.Text.Render("You're all set to start using FontGet."))
	result.WriteString("\n\n")
	result.WriteString(ui.InfoText.Render("Try these commands to get started:"))
	result.WriteString("\n")
	result.WriteString(fmt.Sprintf("  %s  %s\n", ui.CommandExample.Render("fontget search <name>"), ui.Text.Render("Search for fonts")))
	result.WriteString(fmt.Sprintf("  %s  %s\n", ui.CommandExample.Render("fontget list"), ui.Text.Render("List installed fonts")))
	result.WriteString(fmt.Sprintf("  %s  %s\n", ui.CommandExample.Render("fontget add <font>"), ui.Text.Render("Install a font")))
	result.WriteString(fmt.Sprintf("  %s  %s\n", ui.CommandExample.Render("fontget --help"), ui.Text.Render("See all available commands")))
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
