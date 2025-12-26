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
	Long:  "Launch an interactive TUI to select and preview themes with live preview.",
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
	preview       *components.PreviewModel
	width         int
	height        int
	buttons       *components.ButtonGroup
	quitting      bool
}

// NewThemeSelectionModel creates a new theme selection model
func NewThemeSelectionModel() (*themeSelectionModel, error) {
	// Get current theme from config
	appConfig := config.GetUserPreferences()
	currentTheme := appConfig.Theme

	// Default to system if empty
	if currentTheme == "" {
		currentTheme = "system"
	}

	// Get theme options
	options, err := ui.GetThemeOptions(currentTheme)
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
	if err := preview.LoadTheme(options[selectedIndex].ThemeName); err != nil {
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
		preview:       preview,
		buttons:       buttons,
	}, nil
}

// Init initializes the model
func (m themeSelectionModel) Init() tea.Cmd {
	// Initialize preview animations
	return m.preview.Init()
}

// Update handles messages and updates the model
func (m themeSelectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle preview animation updates
	if cmd, shouldRedraw := m.preview.Update(msg); shouldRedraw {
		// Preview requested a redraw, return the command
		return m, cmd
	}

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
				// Hot-reload preview (errors will be shown in preview panel)
				_ = m.preview.LoadTheme(m.themes[m.selectedIndex].ThemeName)
			}
			return m, nil

		case "down", "j":
			if m.selectedIndex < len(m.themes)-1 {
				m.selectedIndex++
				m.buttons.Selected = m.selectedIndex
				// Hot-reload preview (errors will be shown in preview panel)
				_ = m.preview.LoadTheme(m.themes[m.selectedIndex].ThemeName)
			}
			return m, nil

		case "enter":
			// Apply selected theme
			selected := m.themes[m.selectedIndex]
			if err := m.applyTheme(selected.ThemeName); err != nil {
				// Store error in preview for display
				m.preview.SetError(fmt.Sprintf("Failed to apply theme: %v", err))
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

	// Build title (integrated into frame) and footer (commands + current theme)
	titleText := "Theme Selection"
	// Footer: commands on left, current theme on right
	var commands []string
	commands = append(commands, ui.RenderKeyWithDescription("↑/↓", "Navigate"))
	commands = append(commands, ui.RenderKeyWithDescription("Enter", "Select"))
	commands = append(commands, ui.RenderKeyWithDescription("Esc", "Cancel"))
	help := strings.Join(commands, "  ")

	currentInfo := fmt.Sprintf("Current Theme: %s", m.currentTheme)

	// Footer is a single line; height = 1
	headerHeight := 0 // title is inside the frame
	footerHeight := 1

	// Calculate layout using helper function
	layoutConfig := LayoutConfig{
		TerminalWidth:  m.width,
		TerminalHeight: m.height,
		HeaderHeight:   headerHeight,
		FooterHeight:   footerHeight,
		MarginWidth:    2, // 1 char on each side
		SeparatorWidth: 1,
	}

	layout := CalculatePanelLayout(layoutConfig)

	// Build left panel content (theme list)
	leftContent := m.renderLeftPanelContent(layout.LeftWidth, layout.PanelHeight)

	// Build right panel content (preview)
	rightContent := m.renderRightPanelContent(layout.RightWidth, layout.PanelHeight)

	// Render combined panels with shared border and title
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

	// Add margins around the combined panels to prevent border wrapping
	margin := 1 // 1 char margin on each side
	marginedCombined := lipgloss.NewStyle().
		PaddingLeft(margin).
		PaddingRight(margin).
		Render(combined)

	// Footer line: commands left, current info right
	footer := m.renderFooter(help, currentInfo, m.width)

	var content strings.Builder
	content.WriteString(marginedCombined)
	content.WriteString("\n")
	content.WriteString(footer)

	return lipgloss.NewStyle().
		Width(m.width).
		MaxWidth(m.width).
		Render(content.String())
}

// renderLeftPanelContent renders the left panel content (without border)
func (m themeSelectionModel) renderLeftPanelContent(width, _ int) string {
	// Panel structure: │ (border) + content with padding + │ (separator)
	// Content with padding should be: width - 1 (minus left border)
	// Content before padding: (width - 1) - 2 = width - 3
	contentWidth := width - 3
	if contentWidth < 10 {
		contentWidth = 10
	}

	var lines []string
	colors := ui.GetCurrentColors()
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.Secondary)).
		Bold(true)
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Placeholders))

	darkHeaderShown := false
	lightHeaderShown := false

	for i, option := range m.themes {
		// Add "DARK THEMES" header before first dark theme (after System)
		if i > 0 && m.themes[i-1].Style == "" && option.Style == "dark" && !darkHeaderShown {
			lines = append(lines, "") // Blank line
			header := headerStyle.Render("DARK THEMES")
			lines = append(lines, header)
			separator := strings.Repeat("─", contentWidth)
			lines = append(lines, separatorStyle.Render(separator))
			darkHeaderShown = true
		}

		// Add "LIGHT THEMES" header before first light theme (after dark themes)
		if i > 0 && m.themes[i-1].Style == "dark" && option.Style == "light" && !lightHeaderShown {
			lines = append(lines, "") // Blank line
			header := headerStyle.Render("LIGHT THEMES")
			lines = append(lines, header)
			separator := strings.Repeat("─", contentWidth)
			lines = append(lines, separatorStyle.Render(separator))
			lightHeaderShown = true
		}

		buttonText := option.DisplayName
		button := components.Button{
			Text:     buttonText,
			Selected: (i == m.selectedIndex && m.buttons.HasFocus),
		}
		// Use RenderFullWidth to make button expand to fill the content width
		rendered := button.RenderFullWidth(contentWidth)
		lines = append(lines, rendered)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	// Add equal padding on all sides (1 char each)
	return lipgloss.NewStyle().Padding(1, 1).Render(content)
}

// renderRightPanelContent renders the right panel content (without border)
func (m themeSelectionModel) renderRightPanelContent(width, _ int) string {
	// Panel structure: │ (separator) + content with padding + │ (border)
	// Content with padding should be: width - 1 (minus right border)
	// Content before padding: (width - 1) - 2 = width - 3
	contentWidth := width - 3
	if contentWidth < 0 {
		contentWidth = 0
	}

	preview := m.preview.View(contentWidth)
	// Add equal padding on all sides (1 char each)
	return lipgloss.NewStyle().Padding(1, 1).Render(preview)
}

// renderFooter aligns commands on the left and current info on the right
// Aligns with card borders (accounting for 1 char margin on each side)
func (m themeSelectionModel) renderFooter(help, current string, totalWidth int) string {
	// Card has 1 char margin on left, so footer should align with left border
	// Left border position: margin (1) + border char (1) = position 2
	// Right border position: totalWidth - margin (1) - border char (1) = totalWidth - 2
	leftBorderPos := 2
	rightBorderPos := totalWidth - 2

	// Left side: align with left border (position 2)
	left := strings.Repeat(" ", leftBorderPos) + help

	// Right side: align with right border
	rightWidth := lipgloss.Width(current)
	rightStart := rightBorderPos - rightWidth
	if rightStart < 0 {
		rightStart = 0
	}

	// Calculate gap
	leftEnd := leftBorderPos + lipgloss.Width(help)
	gap := rightStart - leftEnd
	if gap < 1 {
		gap = 1
	}

	return left + strings.Repeat(" ", gap) + current
}

// applyTheme applies the selected theme to the config
func (m *themeSelectionModel) applyTheme(themeName string) error {
	// Update config
	appConfig := config.GetUserPreferences()
	appConfig.Theme = themeName

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
