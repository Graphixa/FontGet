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

// Theme UI text constants
const (
	ThemeSelectionTitle = "Theme Selection"
	DarkThemesLabel     = "DARK THEMES"
	LightThemesLabel    = "LIGHT THEMES"
)

var themeCmd = &cobra.Command{
	Use:     "theme",
	Short:   "Interactive theme selector",
	Long:    `Launch an interactive TUI to select and preview themes with live preview.`,
	Example: `  fontget theme`,
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
	scrollOffset  int // First visible line index (line-based scrolling)
	currentTheme  string
	preview       *components.PreviewModel
	width         int
	height        int
	buttons       *components.ButtonGroup
	quitting      bool
}

// MenuLine represents a single line in the menu (header line or theme button)
type MenuLine struct {
	Type       string // "header_blank", "header_text", "header_separator", or "theme"
	Content    string // The actual content to render
	ThemeIndex int    // For theme lines, the index in m.themes array (-1 for header lines)
	IsSelected bool   // Whether this line represents the selected theme
}

// NewThemeSelectionModel creates a new theme selection model
func NewThemeSelectionModel() (*themeSelectionModel, error) {
	// Get current theme from config
	appConfig := config.GetUserPreferences()
	currentTheme := appConfig.Theme.Name

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

	// Create a temporary model to calculate line positions
	tempModel := &themeSelectionModel{
		themes:        options,
		selectedIndex: selectedIndex,
		buttons:       buttons,
	}

	// Build all lines to find the line index of the selected theme
	// Use a reasonable contentWidth for calculation (will be recalculated on render)
	allLines := tempModel.buildAllMenuLines(30)
	selectedLineIndex := tempModel.findThemeLineIndex(selectedIndex, allLines)
	if selectedLineIndex < 0 {
		selectedLineIndex = 0
	}

	// Initialize scrollOffset: always show from top, but scroll just enough to keep selected theme visible
	// We'll estimate available height (will be recalculated when window size is known)
	estimatedAvailableHeight := 20

	var initialScrollOffset int
	// If selected theme is within the first viewport, show from top
	if selectedLineIndex < estimatedAvailableHeight {
		initialScrollOffset = 0
	} else {
		// Selected theme is beyond first viewport - scroll just enough to show it
		// Position it near the top (with a small buffer) but still visible
		initialScrollOffset = selectedLineIndex - estimatedAvailableHeight + 1
		if initialScrollOffset < 0 {
			initialScrollOffset = 0
		}
	}

	return &themeSelectionModel{
		themes:        options,
		selectedIndex: selectedIndex,
		scrollOffset:  initialScrollOffset, // Initialize: center if in middle, otherwise top/bottom
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
				// Adjust scroll to keep selected item visible (line-based)
				m = m.adjustScrollForSelection()
			}
			return m, nil

		case "down", "j":
			if m.selectedIndex < len(m.themes)-1 {
				m.selectedIndex++
				m.buttons.Selected = m.selectedIndex
				// Hot-reload preview (errors will be shown in preview panel)
				_ = m.preview.LoadTheme(m.themes[m.selectedIndex].ThemeName)
				// Adjust scroll to keep selected item visible (line-based)
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

	// Style "Current Theme: " with the primary color from the current theme
	colors := ui.GetCurrentColors()
	primaryColor := colors.Primary
	// Handle empty color (e.g., system theme) by using NoColor
	var color lipgloss.TerminalColor
	if primaryColor == "" {
		color = lipgloss.NoColor{}
	} else {
		color = lipgloss.Color(primaryColor)
	}
	themeLabelStyle := lipgloss.NewStyle().Foreground(color)
	themeLabel := themeLabelStyle.Render("Current Theme: ")
	currentInfo := themeLabel + m.currentTheme

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

	out := content.String()
	if m.width > 0 && m.height > 0 {
		return ui.FillTerminalArea(out, m.width, m.height)
	}
	return lipgloss.NewStyle().
		Width(m.width).
		MaxWidth(m.width).
		Render(out)
}

// buildAllMenuLines builds a complete array of all menu lines (headers + themes)
// This is the single source of truth for all lines in the menu
func (m themeSelectionModel) buildAllMenuLines(contentWidth int) []MenuLine {
	var allLines []MenuLine
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
			allLines = append(allLines, MenuLine{
				Type:       "header_blank",
				Content:    "",
				ThemeIndex: -1,
			})
			allLines = append(allLines, MenuLine{
				Type:       "header_text",
				Content:    headerStyle.Render(DarkThemesLabel),
				ThemeIndex: -1,
			})
			allLines = append(allLines, MenuLine{
				Type:       "header_separator",
				Content:    separatorStyle.Render(strings.Repeat("─", contentWidth)),
				ThemeIndex: -1,
			})
			darkHeaderShown = true
		}

		// Add "LIGHT THEMES" header before first light theme (after dark themes)
		if i > 0 && m.themes[i-1].Style == "dark" && option.Style == "light" && !lightHeaderShown {
			allLines = append(allLines, MenuLine{
				Type:       "header_blank",
				Content:    "",
				ThemeIndex: -1,
			})
			allLines = append(allLines, MenuLine{
				Type:       "header_text",
				Content:    headerStyle.Render(LightThemesLabel),
				ThemeIndex: -1,
			})
			allLines = append(allLines, MenuLine{
				Type:       "header_separator",
				Content:    separatorStyle.Render(strings.Repeat("─", contentWidth)),
				ThemeIndex: -1,
			})
			lightHeaderShown = true
		}

		// Add theme button line
		buttonText := option.DisplayName
		button := components.Button{
			Text:     buttonText,
			Selected: (i == m.selectedIndex && m.buttons.HasFocus),
		}
		rendered := button.RenderFullWidth(contentWidth)
		allLines = append(allLines, MenuLine{
			Type:       "theme",
			Content:    rendered,
			ThemeIndex: i,
			IsSelected: (i == m.selectedIndex && m.buttons.HasFocus),
		})
	}

	return allLines
}

// findThemeLineIndex finds the line index for a given theme index
func (m themeSelectionModel) findThemeLineIndex(themeIndex int, allLines []MenuLine) int {
	for i, line := range allLines {
		if line.ThemeIndex == themeIndex {
			return i
		}
	}
	return -1
}

// adjustScrollForSelection adjusts scrollOffset (line-based) to keep selectedIndex visible
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
	availableHeight := layout.PanelHeight - 2 - 2
	if availableHeight < 1 {
		availableHeight = 1
	}

	// Build all lines to find the selected theme's line index
	// Use a reasonable contentWidth for calculation
	contentWidth := 30
	if m.width > 0 {
		contentWidth = m.width - 3
		if contentWidth < 10 {
			contentWidth = 10
		}
	}
	allLines := m.buildAllMenuLines(contentWidth)
	selectedLineIndex := m.findThemeLineIndex(m.selectedIndex, allLines)
	if selectedLineIndex < 0 {
		selectedLineIndex = 0
	}

	// Check if selected line is currently visible
	isCurrentlyVisible := selectedLineIndex >= m.scrollOffset && selectedLineIndex < m.scrollOffset+availableHeight

	if !isCurrentlyVisible {
		// Selected line is not visible - scroll just enough to show it, keeping it near the top
		if selectedLineIndex < availableHeight {
			// Selected theme is within first viewport - show from top
			m.scrollOffset = 0
		} else {
			// Selected theme is beyond first viewport - scroll just enough to show it
			// Position it near the top (with a small buffer) but still visible
			m.scrollOffset = selectedLineIndex - availableHeight + 1
			if m.scrollOffset < 0 {
				m.scrollOffset = 0
			}
		}
	} else {
		// Selected line is visible - only adjust if it's about to go out of view
		// Use a small buffer to prevent it from going out of view
		buffer := 2
		if availableHeight <= 4 {
			buffer = 1
		}

		// Check if we're at or near the end of the list
		totalThemes := len(m.themes)
		isNearEnd := m.selectedIndex >= totalThemes-2 // Within last 2 items

		if isNearEnd {
			// At the bottom: don't adjust - prevents climbing bug
		} else {
			// Not at the bottom: only adjust if item is about to go out of view
			// Check if too close to bottom edge (about to scroll out of view)
			if selectedLineIndex >= m.scrollOffset+availableHeight-buffer {
				// Scroll down just enough to keep it visible, keeping it near the top
				m.scrollOffset = selectedLineIndex - availableHeight + buffer + 1
				if m.scrollOffset < 0 {
					m.scrollOffset = 0
				}
			}

			// Check if too close to top edge (about to scroll out of view)
			if selectedLineIndex < m.scrollOffset+buffer {
				// Scroll up just enough to keep it visible, keeping it near the top
				m.scrollOffset = selectedLineIndex - buffer
				if m.scrollOffset < 0 {
					m.scrollOffset = 0
				}
			}
		}
	}

	// Ensure scrollOffset doesn't exceed bounds
	maxScrollOffset := len(allLines) - 1
	if m.scrollOffset > maxScrollOffset {
		m.scrollOffset = maxScrollOffset
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}

	return m
}

// renderLeftPanelContent renders the left panel content (without border)
// Uses line-based scrolling: builds all lines first, then renders visible slice
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

	// Build all menu lines (headers + themes) - single source of truth
	allLines := m.buildAllMenuLines(contentWidth)

	// Ensure scrollOffset is within bounds
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
	if m.scrollOffset >= len(allLines) {
		m.scrollOffset = len(allLines) - 1
		if m.scrollOffset < 0 {
			m.scrollOffset = 0
		}
	}

	// Calculate visible range: slice the array based on scrollOffset
	startLine := m.scrollOffset
	endLine := startLine + availableHeight
	if endLine > len(allLines) {
		endLine = len(allLines)
	}

	// Extract visible lines
	var visibleLines []string
	for i := startLine; i < endLine; i++ {
		visibleLines = append(visibleLines, allLines[i].Content)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, visibleLines...)
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
	appConfig.Theme.Name = themeName

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
