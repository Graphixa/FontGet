package onboarding

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"fontget/internal/config"
	"fontget/internal/shared"
	"fontget/internal/sources"
	"fontget/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

// WelcomeStep displays the welcome message to new users
type WelcomeStep struct{}

func NewWelcomeStep() *WelcomeStep {
	return &WelcomeStep{}
}

func (s *WelcomeStep) Name() string {
	return "Welcome"
}

func (s *WelcomeStep) CanSkip() bool {
	return false // Welcome cannot be skipped
}

func (s *WelcomeStep) Execute() (bool, error) {
	// Clear screen for this step
	clearScreen()

	// Display styled welcome message
	// Section starts with blank line per spacing guidelines
	fmt.Println()
	fmt.Println(ui.PageTitle.Render("Welcome to FontGet!"))
	fmt.Println()
	fmt.Println(ui.Text.Render("This is your first time using FontGet. Let's get you set up."))
	fmt.Println()
	fmt.Println(ui.InfoText.Render("FontGet is a powerful command-line font manager that helps you"))
	fmt.Println(ui.InfoText.Render("install and manage fonts from various sources."))
	fmt.Println()

	// Wait for user to continue to next screen
	if err := waitForContinue(); err != nil {
		return false, fmt.Errorf("unable to read input: %w", err)
	}

	return true, nil
}

// LicenseStep handles license acceptance for all default font sources
type LicenseStep struct{}

func NewLicenseStep() *LicenseStep {
	return &LicenseStep{}
}

func (s *LicenseStep) Name() string {
	return "License Agreement"
}

func (s *LicenseStep) CanSkip() bool {
	return false // License acceptance cannot be skipped
}

func (s *LicenseStep) Execute() (bool, error) {
	// Get all default sources
	defaultSources := sources.DefaultSources()

	// Check if all default sources are already accepted
	allAccepted := true
	for sourceName := range defaultSources {
		accepted, err := config.IsSourceAccepted(sourceName)
		if err != nil {
			return false, fmt.Errorf("unable to check source acceptance: %w", err)
		}
		if !accepted {
			allAccepted = false
			break
		}
	}

	if allAccepted {
		return true, nil // Already accepted, continue
	}

	// Use custom confirmation dialog that shows all info in alt-screen
	confirmed, err := runLicenseConfirmation(defaultSources)
	if err != nil {
		return false, fmt.Errorf("unable to show license prompt: %w", err)
	}

	if !confirmed {
		// User declined - show message and end section with blank line
		fmt.Println()
		fmt.Println(ui.WarningText.Render("License acceptance is required to use FontGet."))
		fmt.Println(ui.Text.Render("You can review licenses and accept them later."))
		fmt.Println() // Section ends with blank line per spacing guidelines
		return false, nil
	}

	// Save acceptance for all default sources
	for sourceName := range defaultSources {
		if err := config.AcceptSource(sourceName); err != nil {
			return false, fmt.Errorf("unable to save license acceptance for %s: %w", sourceName, err)
		}
	}

	// Success message - section ends with blank line
	fmt.Println()
	fmt.Println(ui.SuccessText.Render("License agreements accepted."))
	fmt.Println() // Section ends with blank line per spacing guidelines

	return true, nil
}

// SettingsStep displays and confirms default settings
type SettingsStep struct{}

func NewSettingsStep() *SettingsStep {
	return &SettingsStep{}
}

func (s *SettingsStep) Name() string {
	return "Settings Configuration"
}

func (s *SettingsStep) CanSkip() bool {
	return true // Settings can be skipped (will use defaults)
}

func (s *SettingsStep) Execute() (bool, error) {
	// Get default settings
	defaults := config.DefaultUserPreferences()

	// Confirm settings using alt-screen confirmation (all info visible in alt-screen)
	confirmed, err := runSettingsConfirmation(defaults)
	if err != nil {
		// User-friendly error message per verbose/debug guidelines
		return false, fmt.Errorf("unable to show settings confirmation: %w", err)
	}

	if !confirmed {
		// User declined, but we'll continue with defaults anyway
		// Section ends with blank line per spacing guidelines
		fmt.Println()
		fmt.Println(ui.WarningText.Render("Using default settings. You can change them later with 'fontget config edit'."))
		fmt.Println() // Section ends with blank line per spacing guidelines
	}

	// Ensure config file exists with defaults
	// This is safe to call even if file exists - it won't overwrite
	if err := config.GenerateInitialUserPreferences(); err != nil {
		// User-friendly error message per verbose/debug guidelines
		return false, fmt.Errorf("unable to create default configuration: %w", err)
	}

	return true, nil
}

// CompletionStep shows the completion message
type CompletionStep struct{}

func NewCompletionStep() *CompletionStep {
	return &CompletionStep{}
}

func (s *CompletionStep) Name() string {
	return "Completion"
}

func (s *CompletionStep) CanSkip() bool {
	return false // Completion cannot be skipped
}

func (s *CompletionStep) Execute() (bool, error) {
	// Clear screen for this step
	clearScreen()

	// Completion message - section starts with blank line per spacing guidelines
	fmt.Println()
	fmt.Println(ui.SuccessText.Render("Setup complete!"))
	fmt.Println()
	fmt.Println(ui.Text.Render("You're all set to start using FontGet."))
	fmt.Println()
	fmt.Println(ui.InfoText.Render("Try these commands to get started:"))
	fmt.Printf("  %s  %s\n", ui.Text.Render("fontget search <name>"), ui.Text.Render("Search for fonts"))
	fmt.Printf("  %s  %s\n", ui.Text.Render("fontget list"), ui.Text.Render("List installed fonts"))
	fmt.Printf("  %s  %s\n", ui.Text.Render("fontget add <font>"), ui.Text.Render("Install a font"))
	fmt.Printf("  %s  %s\n", ui.Text.Render("fontget --help"), ui.Text.Render("See all available commands"))
	fmt.Println() // Section ends with blank line per spacing guidelines

	// Wait for user to continue (final screen, just to acknowledge)
	if err := waitForContinue(); err != nil {
		return false, fmt.Errorf("unable to read input: %w", err)
	}

	return true, nil
}

// Helper functions

func formatBool(value bool) string {
	if value {
		return ui.SuccessText.Render("Enabled")
	}
	return ui.WarningText.Render("Disabled")
}

func formatSorting(usePopularity bool) string {
	if usePopularity {
		return ui.SuccessText.Render("Popularity-based")
	}
	return ui.Text.Render("Alphabetical")
}

// LicenseConfirmModel represents a license confirmation dialog with all source info
type LicenseConfirmModel struct {
	sources   map[string]sources.SourceInfo
	Confirmed bool
	Quit      bool
	Width     int
	Height    int
}

// NewLicenseConfirmModel creates a new license confirmation model
func NewLicenseConfirmModel(sourcesMap map[string]sources.SourceInfo) *LicenseConfirmModel {
	return &LicenseConfirmModel{
		sources: sourcesMap,
		Width:   80,
		Height:  24,
	}
}

// Init initializes the license confirmation dialog
func (m LicenseConfirmModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the license confirmation dialog
func (m LicenseConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y", "enter":
			m.Confirmed = true
			return m, tea.Quit
		case "n", "N", "esc":
			m.Confirmed = false
			return m, tea.Quit
		case "ctrl+c":
			m.Confirmed = false
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil
	}

	return m, nil
}

// View renders the license confirmation dialog with all source information
func (m LicenseConfirmModel) View() string {
	var result strings.Builder

	// Get actual terminal width for proper text wrapping
	terminalWidth := shared.GetTerminalWidth()

	// Calculate available width (account for margins)
	availableWidth := terminalWidth - 4 // Leave some margin
	if availableWidth < 60 {
		availableWidth = 60 // Minimum readable width
	}

	// Start with blank line
	result.WriteString("\n")

	// Title
	result.WriteString(ui.PageTitle.Render("License Agreement"))
	result.WriteString("\n\n")

	// Legal disclaimer - use warning style only for "IMPORTANT"
	result.WriteString(ui.WarningText.Render("IMPORTANT: By using FontGet, you acknowledge and agree to the following:"))
	result.WriteString("\n\n")

	// Source information - plain text, wrapped
	introText := "FontGet installs fonts from various sources. These fonts are subject to their respective license agreements. You are responsible for ensuring compliance with each font's license terms."
	introLines := shared.WrapText(introText, availableWidth)
	for _, line := range introLines {
		result.WriteString(line)
		result.WriteString("\n")
	}
	result.WriteString("\n")

	// Help text - InfoText (mauve)
	result.WriteString(ui.InfoText.Render("To review a particular font's license, run:"))
	result.WriteString("\n")
	result.WriteString(fmt.Sprintf("  %s\n", ui.Text.Render("fontget info <font-id> --license")))
	result.WriteString("\n")

	// Acceptance statement - plain text, wrapped
	acceptText := "By proceeding, you accept the license agreements for all fonts from the default sources and agree to comply with their terms."
	acceptLines := shared.WrapText(acceptText, availableWidth)
	for _, line := range acceptLines {
		result.WriteString(line)
		result.WriteString("\n")
	}
	result.WriteString("\n")

	// Confirmation prompt is handled separately by promptConfirmSimple
	// No need to include it in View() since we're using regular output, not alt-screen

	return result.String()
}

// renderSourcesInfo renders the sources information screen
func renderSourcesInfo(sourcesMap map[string]sources.SourceInfo, width int) string {
	var result strings.Builder

	// Calculate available width (account for margins)
	availableWidth := width - 4 // Leave some margin
	if availableWidth < 60 {
		availableWidth = 60 // Minimum readable width
	}

	// Start with blank line
	result.WriteString("\n")

	// Title
	result.WriteString(ui.PageTitle.Render("Sources"))
	result.WriteString("\n\n")

	// Introduction text - plain text, wrapped
	introText := "Sources are how FontGet finds and installs fonts. FontGet has the following 3 default sources built-in:"
	introLines := shared.WrapText(introText, availableWidth)
	for _, line := range introLines {
		result.WriteString(line)
		result.WriteString("\n")
	}
	result.WriteString("\n")

	// Source URL mapping to website URLs
	sourceURLs := map[string]string{
		"Google Fonts":  "https://fonts.google.com/",
		"Nerd Fonts":    "https://www.nerdfonts.com/",
		"Font Squirrel": "https://www.fontsquirrel.com/",
	}

	// Display all default sources with website URLs on same line, no space between
	sourceOrder := []string{"Google Fonts", "Nerd Fonts", "Font Squirrel"}
	for _, sourceName := range sourceOrder {
		if _, exists := sourcesMap[sourceName]; exists {
			websiteURL := sourceURLs[sourceName]
			sourceInfo := sourcesMap[sourceName]
			// Source name in pink, URL on same line with dash
			line := fmt.Sprintf("  %s %s - %s", "•", ui.TableSourceName.Render(sourceName), websiteURL)
			// Add disabled note if source is disabled
			if !sourceInfo.Enabled {
				line += fmt.Sprintf(" %s", ui.WarningText.Render("(disabled by default)"))
			}
			result.WriteString(line + "\n")
		}
	}
	result.WriteString("\n")

	// Custom sources section - InfoText header, plain text body
	result.WriteString(ui.InfoText.Render("Custom Sources:"))
	result.WriteString("\n")
	customText := "You can add custom font sources to FontGet. If you add custom font sources, you are solely responsible for ensuring compliance with those sources' license agreements. FontGet does not verify or guarantee license compliance for custom sources."
	customLines := shared.WrapText(customText, availableWidth)
	for _, line := range customLines {
		result.WriteString(line)
		result.WriteString("\n")
	}
	result.WriteString("\n")

	// Source management information - InfoText (mauve)
	result.WriteString(ui.InfoText.Render("Managing Sources:"))
	result.WriteString("\n")
	manageText := "You can manage sources (enable, disable, or add custom sources) using the command:"
	manageLines := shared.WrapText(manageText, availableWidth)
	for _, line := range manageLines {
		result.WriteString(line)
		result.WriteString("\n")
	}
	result.WriteString(fmt.Sprintf("  %s\n", ui.Text.Render("fontget sources manage")))
	result.WriteString("\n")

	return result.String()
}

// runLicenseConfirmation runs the license confirmation dialog with all source info
// Uses regular output (no alt-screen) so users can scroll
func runLicenseConfirmation(sourcesMap map[string]sources.SourceInfo) (bool, error) {
	model := NewLicenseConfirmModel(sourcesMap)

	// Get actual terminal width for proper text wrapping
	terminalWidth := shared.GetTerminalWidth()

	// Clear screen and display the license information screen
	clearScreen()
	fmt.Print(model.View())

	// Wait for user to continue to sources screen
	if err := waitForContinue(); err != nil {
		return false, fmt.Errorf("unable to read input: %w", err)
	}

	// Clear screen and display sources information on separate screen
	clearScreen()
	fmt.Print(renderSourcesInfo(sourcesMap, terminalWidth))

	// Wait for user to continue to confirmation
	if err := waitForContinue(); err != nil {
		return false, fmt.Errorf("unable to read input: %w", err)
	}

	// Use simple confirmation prompt (no alt-screen)
	confirmed, err := promptConfirmSimple("Do you accept the license agreements?")
	if err != nil {
		return false, fmt.Errorf("unable to read response: %w", err)
	}

	return confirmed, nil
}

// SettingsConfirmModel represents a settings confirmation dialog
type SettingsConfirmModel struct {
	defaults  *config.AppConfig
	Confirmed bool
	Quit      bool
	Width     int
	Height    int
}

// NewSettingsConfirmModel creates a new settings confirmation model
func NewSettingsConfirmModel(defaults *config.AppConfig) *SettingsConfirmModel {
	return &SettingsConfirmModel{
		defaults: defaults,
		Width:    80,
		Height:   24,
	}
}

// Init initializes the settings confirmation dialog
func (m SettingsConfirmModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the settings confirmation dialog
func (m SettingsConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y", "enter":
			m.Confirmed = true
			return m, tea.Quit
		case "n", "N", "esc":
			m.Confirmed = false
			return m, tea.Quit
		case "ctrl+c":
			m.Confirmed = false
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil
	}

	return m, nil
}

// View renders the settings confirmation dialog with all settings info
func (m SettingsConfirmModel) View() string {
	var result strings.Builder

	// Calculate available width (account for margins)
	availableWidth := m.Width - 4 // Leave some margin
	if availableWidth < 60 {
		availableWidth = 60 // Minimum readable width
	}

	// Title
	result.WriteString(ui.PageTitle.Render("Default Settings"))
	result.WriteString("\n\n")

	// Introduction - plain text, wrapped
	introText := "FontGet will use the following default settings:"
	introLines := shared.WrapText(introText, availableWidth)
	for _, line := range introLines {
		result.WriteString(line)
		result.WriteString("\n")
	}
	result.WriteString("\n")

	// Display each setting with clear explanation
	settings := []struct {
		name        string
		value       string
		description string
	}{
		{
			name:        "Check for updates",
			value:       formatBool(m.defaults.Update.CheckForUpdates),
			description: "FontGet will automatically check for new versions when you start the application. When an update is available, you'll be prompted to confirm before installing.",
		},
		{
			name:        "Sorting method",
			value:       formatSorting(m.defaults.Search.EnablePopularitySort),
			description: "When searching for fonts, results will be sorted by popularity first (most commonly used fonts appear first), then alphabetically. This helps you find popular fonts more easily.",
		},
	}

	for _, setting := range settings {
		result.WriteString(fmt.Sprintf("  %s %s\n", "•", ui.InfoText.Render(setting.name)))
		result.WriteString(fmt.Sprintf("    Setting: %s\n", setting.value))
		// Wrap description text with indentation
		opts := shared.WrapOptions{
			Width:  availableWidth - 4, // Account for indentation
			Indent: "    ",
		}
		descLines := shared.WrapTextWithOptions(setting.description, opts)
		for _, line := range descLines {
			result.WriteString(line)
			result.WriteString("\n")
		}
		result.WriteString("\n")
	}

	// Footer - plain text
	result.WriteString("These settings can be changed later using 'fontget config edit'.")
	result.WriteString("\n")

	return result.String()
}

// runSettingsConfirmation runs the settings confirmation dialog with all settings info
func runSettingsConfirmation(defaults *config.AppConfig) (bool, error) {
	model := NewSettingsConfirmModel(defaults)

	// Get actual terminal width for proper text wrapping
	terminalWidth := shared.GetTerminalWidth()
	model.Width = terminalWidth

	// Clear screen and display the settings information screen
	clearScreen()
	fmt.Print(model.View())

	// Wait for user to continue to confirmation
	if err := waitForContinue(); err != nil {
		return false, fmt.Errorf("unable to read input: %w", err)
	}

	// Use simple confirmation prompt (no alt-screen)
	confirmed, err := promptConfirmSimple("Accept these default settings?")
	if err != nil {
		return false, fmt.Errorf("unable to read response: %w", err)
	}

	return confirmed, nil
}

// clearScreen clears the terminal screen using ANSI escape codes
func clearScreen() {
	// ANSI escape code to clear screen and move cursor to top-left
	fmt.Print("\033[2J\033[H")
}

// waitForContinue waits for the user to press Enter to continue to the next screen
func waitForContinue() error {
	fmt.Println()
	fmt.Print(ui.InfoText.Render("Press Enter to continue..."))

	reader := bufio.NewReader(os.Stdin)
	_, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("unable to read input: %w", err)
	}

	// Clear screen before showing next step
	clearScreen()

	return nil
}

// promptConfirmSimple provides a simple confirmation prompt without alt-screen
// This is better for onboarding where we want to keep previous content visible
func promptConfirmSimple(message string) (bool, error) {
	// Display the prompt with styled UI
	fmt.Printf("%s\n", ui.Text.Render(message))
	fmt.Println()

	// Show keyboard shortcuts
	commands := []string{
		ui.RenderKeyWithDescription("Y", "Yes"),
		ui.RenderKeyWithDescription("N", "No"),
	}
	helpText := strings.Join(commands, "  ")
	fmt.Println(helpText)
	fmt.Print("\n> ")

	// Read user input
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("unable to read response: %w", err)
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes" || response == "", nil // Empty/Enter defaults to yes
}
