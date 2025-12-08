package cmd

import (
	"fmt"
	"strings"

	"fontget/internal/components"
	"fontget/internal/config"
	"fontget/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var themeCmd = &cobra.Command{
	Use:   "theme",
	Short: "Interactive theme selector",
	Long:  "Launch an interactive TUI to select and preview themes. Themes can be switched between dark and light modes with live preview.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Initialize theme manager if not already done
		if err := ui.InitThemeManager(); err != nil {
			return fmt.Errorf("failed to initialize theme manager: %w", err)
		}

		// Initialize styles
		if err := ui.InitStyles(); err != nil {
			return fmt.Errorf("failed to initialize styles: %w", err)
		}

		// Create theme selection model
		model, err := NewThemeSelectionModel()
		if err != nil {
			return fmt.Errorf("failed to create theme selection model: %w", err)
		}

		// Run TUI
		p := tea.NewProgram(model, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("failed to run theme selector: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(themeCmd)
}

// themeSelectionModel represents the theme selection TUI model
type themeSelectionModel struct {
	themes        []ui.ThemeOption
	selectedIndex int
	currentTheme  string
	currentMode   string
	preview       *components.PreviewModel
	width         int
	height        int
	buttons       *components.ButtonGroup
	quitting      bool
	err           string
}

// NewThemeSelectionModel creates a new theme selection model
func NewThemeSelectionModel() (*themeSelectionModel, error) {
	// Get current theme from config
	appConfig := config.GetUserPreferences()
	currentTheme := appConfig.Theme.Name
	currentMode := appConfig.Theme.Mode

	// Default to catppuccin-dark if empty
	if currentTheme == "" {
		currentTheme = "catppuccin"
	}
	if currentMode == "" || (currentMode != "dark" && currentMode != "light") {
		currentMode = "dark"
	}

	// Get theme options
	options, err := ui.GetThemeOptions(currentTheme, currentMode)
	if err != nil {
		return nil, fmt.Errorf("failed to discover themes: %w", err)
	}

	// Find current selection index
	selectedIndex := 0
	for i, option := range options {
		if option.IsSelected {
			selectedIndex = i
			break
		}
	}

	// Create preview model
	preview := components.NewPreviewModel()

	// Load initial theme for preview
	if err := preview.LoadTheme(options[selectedIndex].ThemeName, options[selectedIndex].Mode); err != nil {
		// Log error but continue - preview will show default
	}

	// Create button group for theme options
	buttonTexts := make([]string, len(options))
	for i, option := range options {
		prefix := "  "
		if option.IsSelected {
			prefix = "✔️ "
		}
		buttonTexts[i] = fmt.Sprintf("%s %s", prefix, option.DisplayName)
	}

	buttons := components.NewButtonGroup(buttonTexts, selectedIndex)
	buttons.SetFocus(true)

	return &themeSelectionModel{
		themes:        options,
		selectedIndex: selectedIndex,
		currentTheme:  currentTheme,
		currentMode:   currentMode,
		preview:       preview,
		buttons:       buttons,
	}, nil
}

// Init initializes the model
func (m themeSelectionModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m themeSelectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.selectedIndex > 0 {
				m.selectedIndex--
				m.buttons.Selected = m.selectedIndex
				// Hot-reload preview
				if err := m.preview.LoadTheme(m.themes[m.selectedIndex].ThemeName, m.themes[m.selectedIndex].Mode); err != nil {
					m.err = fmt.Sprintf("Failed to load preview: %v", err)
				} else {
					m.err = ""
				}
			}
			return m, nil

		case "down", "j":
			if m.selectedIndex < len(m.themes)-1 {
				m.selectedIndex++
				m.buttons.Selected = m.selectedIndex
				// Hot-reload preview
				if err := m.preview.LoadTheme(m.themes[m.selectedIndex].ThemeName, m.themes[m.selectedIndex].Mode); err != nil {
					m.err = fmt.Sprintf("Failed to load preview: %v", err)
				} else {
					m.err = ""
				}
			}
			return m, nil

		case "enter":
			// Apply selected theme
			selected := m.themes[m.selectedIndex]
			if err := m.applyTheme(selected.ThemeName, selected.Mode); err != nil {
				m.err = fmt.Sprintf("Failed to apply theme: %v", err)
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit
		}
	}

	return m, nil
}

// View renders the TUI
func (m themeSelectionModel) View() string {
	if m.quitting {
		return ""
	}

	// Calculate layout - 50/50 split
	separatorWidth := 1
	availableWidth := m.width - separatorWidth
	leftWidth := availableWidth / 2
	rightWidth := availableWidth - leftWidth

	// Build left panel (theme list)
	leftPanel := m.renderLeftPanel(leftWidth)

	// Build right panel (preview)
	rightPanel := m.renderRightPanel(rightWidth)

	// Add a padded separator to keep spacing consistent
	separator := lipgloss.NewStyle().
		Padding(0, 1).
		Render("│")

	// Combine panels using lipgloss JoinHorizontal (top aligned)
	combined := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, separator, rightPanel)

	// Add title
	title := ui.PageTitle.Render("Theme Selection")
	currentInfo := ui.PageSubtitle.Render(fmt.Sprintf("Current: %s %s", m.currentTheme, m.currentMode))

	// Add keyboard help using consistent styling
	var commands []string
	commands = append(commands, ui.RenderKeyWithDescription("↑/↓", "Navigate"))
	commands = append(commands, ui.RenderKeyWithDescription("Enter", "Select"))
	commands = append(commands, ui.RenderKeyWithDescription("Esc", "Cancel"))
	help := strings.Join(commands, "  ")

	// Add error if any
	errorText := ""
	if m.err != "" {
		errorText = "\n" + ui.ErrorText.Render(m.err)
	}

	return fmt.Sprintf("%s\n%s\n\n%s\n\n%s%s", title, currentInfo, combined, help, errorText)
}

// renderLeftPanel renders the left panel with theme list
func (m themeSelectionModel) renderLeftPanel(width int) string {
	// Find the longest theme name for fixed width
	maxWidth := 0
	for _, option := range m.themes {
		if len(option.DisplayName) > maxWidth {
			maxWidth = len(option.DisplayName)
		}
	}

	var lines []string

	// Render buttons with fixed width (no checkmark prefix)
	for i, option := range m.themes {
		buttonText := option.DisplayName
		button := components.Button{
			Text:     buttonText,
			Selected: (i == m.selectedIndex && m.buttons.HasFocus),
		}
		rendered := button.Render()
		// Pad to fixed width using lipgloss
		// Calculate target width: max text length + button padding (6 chars for "[  Text  ]")
		targetWidth := maxWidth + 6
		rendered = lipgloss.NewStyle().Width(targetWidth).Render(rendered)
		lines = append(lines, rendered)
	}

	// Add "Add Custom Theme" option (future)
	// lines = append(lines, "\n"+ui.Text.Render("[ Add Custom Theme ]"))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	// Constrain width and add padding for nicer spacing
	panelStyle := lipgloss.NewStyle().
		Width(width).
		Padding(1, 2)

	return ui.CardBorder.
		Width(width).
		Render(panelStyle.Render(content))
}

// renderRightPanel renders the right panel with preview
func (m themeSelectionModel) renderRightPanel(width int) string {
	preview := m.preview.View(width - 4) // leave room for padding

	// Wrap preview in card border (equal frame like left panel)
	panelStyle := lipgloss.NewStyle().
		Width(width).
		Padding(1, 2)

	return ui.CardBorder.
		Width(width).
		Render(panelStyle.Render(preview))
}

// applyTheme applies the selected theme to the config
func (m *themeSelectionModel) applyTheme(themeName string, mode string) error {
	// Update config
	appConfig := config.GetUserPreferences()
	appConfig.Theme.Name = themeName
	appConfig.Theme.Mode = mode

	// Save config
	if err := config.SaveUserPreferences(appConfig); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Reload theme manager
	if err := ui.InitThemeManager(); err != nil {
		return fmt.Errorf("failed to reload theme manager: %w", err)
	}

	// Reinitialize styles
	if err := ui.InitStyles(); err != nil {
		return fmt.Errorf("failed to reinitialize styles: %w", err)
	}

	return nil
}
