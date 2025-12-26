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
	scrollOffset  int // First visible theme index in the menu
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
		scrollOffset:  selectedIndex, // Initialize to show selected theme
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
		// Adjust scroll to keep selected item visible after window resize
		m = m.adjustScrollForSelection()
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
				// Adjust scroll to keep selected item visible
				m = m.adjustScrollForSelection()
			}
			return m, nil

		case "down", "j":
			if m.selectedIndex < len(m.themes)-1 {
				m.selectedIndex++
				m.buttons.Selected = m.selectedIndex
				// Hot-reload preview (errors will be shown in preview panel)
				_ = m.preview.LoadTheme(m.themes[m.selectedIndex].ThemeName)
				// Adjust scroll to keep selected item visible
				m = m.adjustScrollForSelection()
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

// calculateThemeLinePositions calculates the line positions for each theme,
// accounting for headers. Returns a map of themeIndex -> lineNumber and
// the total number of lines needed to render all themes.
func (m themeSelectionModel) calculateThemeLinePositions() (map[int]int, int) {
	positions := make(map[int]int)
	currentLine := 0
	darkHeaderShown := false
	lightHeaderShown := false

	for i := range m.themes {
		// Add "DARK THEMES" header before first dark theme (after System)
		if i > 0 && m.themes[i-1].Style == "" && m.themes[i].Style == "dark" && !darkHeaderShown {
			currentLine += 3 // Blank line + header + separator
			darkHeaderShown = true
		}

		// Add "LIGHT THEMES" header before first light theme (after dark themes)
		if i > 0 && m.themes[i-1].Style == "dark" && m.themes[i].Style == "light" && !lightHeaderShown {
			currentLine += 3 // Blank line + header + separator
			lightHeaderShown = true
		}

		// Map theme index to line position
		positions[i] = currentLine
		currentLine++ // Each theme button takes 1 line
	}

	return positions, currentLine
}

// adjustScrollForSelection adjusts scrollOffset to keep selectedIndex visible
// Returns the modified model with updated scrollOffset
func (m themeSelectionModel) adjustScrollForSelection() themeSelectionModel {
	if len(m.themes) == 0 || m.height == 0 {
		return m
	}

	// Calculate layout to get panel height (same as in View method)
	layoutConfig := LayoutConfig{
		TerminalWidth:  m.width,
		TerminalHeight: m.height,
		HeaderHeight:   0, // title is inside the frame
		FooterHeight:   1, // footer is a single line
		MarginWidth:    2, // 1 char on each side
		SeparatorWidth: 1,
	}
	layout := CalculatePanelLayout(layoutConfig)

	// Available height = panel height - 2 (borders) - 2 (padding)
	// This matches the calculation in renderLeftPanelContent
	availableHeight := layout.PanelHeight - 2 - 2
	if availableHeight < 1 {
		availableHeight = 1
	}

	// Calculate visible item count from current scrollOffset
	visibleCount := m.calculateVisibleItemCount(availableHeight)

	// If selected item is beyond visible range, adjust scrollOffset
	if m.selectedIndex >= m.scrollOffset+visibleCount {
		// Selected item is below visible range - scroll down
		// Set scrollOffset so selected item is at the bottom of visible range
		m.scrollOffset = m.selectedIndex - visibleCount + 1
		if m.scrollOffset < 0 {
			m.scrollOffset = 0
		}
	} else if m.selectedIndex < m.scrollOffset {
		// Selected item is above visible range - scroll up
		// Set scrollOffset to selected item
		m.scrollOffset = m.selectedIndex
	}

	// Ensure scrollOffset doesn't exceed bounds
	maxScrollOffset := len(m.themes) - 1
	if m.scrollOffset > maxScrollOffset {
		m.scrollOffset = maxScrollOffset
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}

	return m
}

// calculateVisibleItemCount calculates how many theme items can fit in the available height
// Note: availableHeight already accounts for borders and padding
func (m themeSelectionModel) calculateVisibleItemCount(availableHeight int) int {
	if len(m.themes) == 0 {
		return 0
	}

	// availableHeight already accounts for padding, so use it directly
	visibleLines := availableHeight
	if visibleLines < 1 {
		visibleLines = 1
	}

	// Count lines starting from scrollOffset
	linesUsed := 0
	darkHeaderShown := false
	lightHeaderShown := false
	itemCount := 0

	// Check if we need to show headers before scrollOffset
	if m.scrollOffset > 0 {
		if m.themes[m.scrollOffset-1].Style == "" && m.themes[m.scrollOffset].Style == "dark" {
			linesUsed += 3 // DARK THEMES header
			darkHeaderShown = true
		}
		if m.scrollOffset > 0 && m.themes[m.scrollOffset-1].Style == "dark" && m.themes[m.scrollOffset].Style == "light" {
			linesUsed += 3 // LIGHT THEMES header
			lightHeaderShown = true
		}
	}

	// Count lines for themes starting from scrollOffset
	for i := m.scrollOffset; i < len(m.themes); i++ {
		// Check if we need to add header before this theme
		if i > 0 && m.themes[i-1].Style == "" && m.themes[i].Style == "dark" && !darkHeaderShown {
			if linesUsed+3 > visibleLines {
				break
			}
			linesUsed += 3
			darkHeaderShown = true
		}
		if i > 0 && m.themes[i-1].Style == "dark" && m.themes[i].Style == "light" && !lightHeaderShown {
			if linesUsed+3 > visibleLines {
				break
			}
			linesUsed += 3
			lightHeaderShown = true
		}

		// Add line for theme button
		if linesUsed+1 > visibleLines {
			break
		}
		linesUsed++
		itemCount++
	}

	return itemCount
}

// getVisibleThemeRange calculates which theme indices should be visible
// given the scrollOffset and available height. Returns start and end indices (inclusive).
func (m themeSelectionModel) getVisibleThemeRange(availableHeight int) (start, end int) {
	if len(m.themes) == 0 {
		return 0, 0
	}

	positions, totalLines := m.calculateThemeLinePositions()

	// Calculate how many lines we can show
	// availableHeight already accounts for padding, so use it directly
	visibleLines := availableHeight
	if visibleLines < 1 {
		visibleLines = 1
	}

	// Find the first theme that should be visible based on scrollOffset
	// scrollOffset represents the first visible theme index
	start = m.scrollOffset
	if start < 0 {
		start = 0
	}
	if start >= len(m.themes) {
		start = len(m.themes) - 1
	}

	// Find the last visible theme by counting lines
	linesUsed := 0
	darkHeaderShown := false
	lightHeaderShown := false

	// Check if we need to show headers before start index
	if start > 0 {
		if m.themes[start-1].Style == "" && m.themes[start].Style == "dark" {
			linesUsed += 3 // DARK THEMES header
			darkHeaderShown = true
		}
		if start > 0 && m.themes[start-1].Style == "dark" && m.themes[start].Style == "light" {
			linesUsed += 3 // LIGHT THEMES header
			lightHeaderShown = true
		}
	}

	// Count lines for visible themes
	end = start
	for i := start; i < len(m.themes); i++ {
		// Check if we need to add header before this theme
		if i > 0 && m.themes[i-1].Style == "" && m.themes[i].Style == "dark" && !darkHeaderShown {
			if linesUsed+3 > visibleLines {
				break
			}
			linesUsed += 3
			darkHeaderShown = true
		}
		if i > 0 && m.themes[i-1].Style == "dark" && m.themes[i].Style == "light" && !lightHeaderShown {
			if linesUsed+3 > visibleLines {
				break
			}
			linesUsed += 3
			lightHeaderShown = true
		}

		// Add line for theme button
		if linesUsed+1 > visibleLines {
			break
		}
		linesUsed++
		end = i
	}

	// Ensure end doesn't exceed total themes
	if end >= len(m.themes) {
		end = len(m.themes) - 1
	}

	_ = positions  // Keep for potential future use
	_ = totalLines // Keep for potential future use

	return start, end
}

// renderLeftPanelContent renders the left panel content (without border)
func (m themeSelectionModel) renderLeftPanelContent(width, panelHeight int) string {
	// Panel structure: │ (border) + content with padding + │ (separator)
	// Content with padding should be: width - 1 (minus left border)
	// Content before padding: (width - 1) - 2 = width - 3
	contentWidth := width - 3
	if contentWidth < 10 {
		contentWidth = 10
	}

	// Calculate available height: panelHeight - 2 (borders) - 2 (padding) = availableLines
	availableHeight := panelHeight - 2 - 2
	if availableHeight < 1 {
		availableHeight = 1
	}

	// Get visible theme range based on scrollOffset
	startIndex, endIndex := m.getVisibleThemeRange(availableHeight)

	var lines []string
	colors := ui.GetCurrentColors()
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors.Secondary)).
		Bold(true)
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Placeholders))

	darkHeaderShown := false
	lightHeaderShown := false

	// Track if we've already shown headers in the visible range
	// Check if we need to show DARK THEMES header before startIndex
	if startIndex > 0 && len(m.themes) > 0 {
		if m.themes[startIndex-1].Style == "" && m.themes[startIndex].Style == "dark" {
			lines = append(lines, "") // Blank line
			header := headerStyle.Render("Dark Themes")
			lines = append(lines, header)
			separator := strings.Repeat("─", contentWidth)
			lines = append(lines, separatorStyle.Render(separator))
			darkHeaderShown = true
		}
		// Check if we need to show LIGHT THEMES header before startIndex
		if startIndex > 0 && m.themes[startIndex-1].Style == "dark" && m.themes[startIndex].Style == "light" {
			lines = append(lines, "") // Blank line
			header := headerStyle.Render("Light Themes")
			lines = append(lines, header)
			separator := strings.Repeat("─", contentWidth)
			lines = append(lines, separatorStyle.Render(separator))
			lightHeaderShown = true
		}
	}

	// Render only visible themes
	for i := startIndex; i <= endIndex && i < len(m.themes); i++ {
		option := m.themes[i]

		// Add "DARK THEMES" header before first dark theme (after System)
		if i > 0 && m.themes[i-1].Style == "" && option.Style == "dark" && !darkHeaderShown {
			lines = append(lines, "") // Blank line
			header := headerStyle.Render("Dark Themes")
			lines = append(lines, header)
			separator := strings.Repeat("─", contentWidth)
			lines = append(lines, separatorStyle.Render(separator))
			darkHeaderShown = true
		}

		// Add "LIGHT THEMES" header before first light theme (after dark themes)
		if i > 0 && m.themes[i-1].Style == "dark" && option.Style == "light" && !lightHeaderShown {
			lines = append(lines, "") // Blank line
			header := headerStyle.Render("Light Themes")
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
