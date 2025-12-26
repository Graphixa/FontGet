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

	return &themeSelectionModel{
		themes:        options,
		selectedIndex: selectedIndex,
		scrollOffset:  selectedLineIndex, // Initialize to show selected theme (line-based)
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
				Content:    headerStyle.Render("DARK THEMES"),
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
				Content:    headerStyle.Render("LIGHT THEMES"),
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

// calculateLinePosition calculates the line position (0-based) for a given theme index
// accounting for headers that appear before it
// Returns the line number where this theme's button would appear
func (m themeSelectionModel) calculateLinePosition(themeIndex int) int {
	if themeIndex < 0 || themeIndex >= len(m.themes) {
		return 0
	}

	linePos := 0
	darkHeaderShown := false
	lightHeaderShown := false

	// Count all lines before themeIndex (headers + theme buttons)
	for i := 0; i < themeIndex; i++ {
		// Add header before first dark theme (after System)
		if i > 0 && m.themes[i-1].Style == "" && m.themes[i].Style == "dark" && !darkHeaderShown {
			linePos += 3 // Blank line + header + separator
			darkHeaderShown = true
		}

		// Add header before first light theme (after dark themes)
		if i > 0 && m.themes[i-1].Style == "dark" && m.themes[i].Style == "light" && !lightHeaderShown {
			linePos += 3 // Blank line + header + separator
			lightHeaderShown = true
		}

		// Add line for theme button
		linePos++
	}

	// Check if we need to add a header before themeIndex itself
	if themeIndex > 0 {
		if m.themes[themeIndex-1].Style == "" && m.themes[themeIndex].Style == "dark" && !darkHeaderShown {
			linePos += 3 // Blank line + header + separator
		}
		if themeIndex > 0 && m.themes[themeIndex-1].Style == "dark" && m.themes[themeIndex].Style == "light" && !lightHeaderShown {
			linePos += 3 // Blank line + header + separator
		}
	}

	// linePos now represents the line where themeIndex's button appears
	return linePos
}

// findScrollOffsetForLinePosition finds the theme index (scrollOffset) that would
// position the selected item at the target line position (accounting for headers)
func (m themeSelectionModel) findScrollOffsetForLinePosition(selectedIndex, targetLinePos int) int {
	if selectedIndex < 0 || selectedIndex >= len(m.themes) {
		return 0
	}

	// Calculate the absolute line position of the selected item
	selectedLinePos := m.calculateLinePosition(selectedIndex)

	// We want: selectedLinePos - scrollOffsetLinePos = targetLinePos
	// So: scrollOffsetLinePos = selectedLinePos - targetLinePos
	targetScrollLinePos := selectedLinePos - targetLinePos
	if targetScrollLinePos < 0 {
		targetScrollLinePos = 0
	}

	// Find the theme index whose line position is closest to targetScrollLinePos
	bestOffset := 0
	bestDiff := 999999

	// Search from 0 to selectedIndex
	for offset := 0; offset <= selectedIndex && offset < len(m.themes); offset++ {
		offsetLinePos := m.calculateLinePosition(offset)
		diff := offsetLinePos - targetScrollLinePos
		if diff < 0 {
			diff = -diff
		}

		if diff < bestDiff {
			bestDiff = diff
			bestOffset = offset
		}

		// If we've passed the target, we can stop (line positions are monotonic)
		if offsetLinePos > targetScrollLinePos {
			break
		}
	}

	return bestOffset
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
		// Selected line is not visible - adjust scrollOffset to show it
		if selectedLineIndex >= m.scrollOffset+availableHeight {
			// Selected line is below visible range - scroll down
			// Position it with a small buffer from top (2 lines)
			buffer := 2
			m.scrollOffset = selectedLineIndex - buffer
			if m.scrollOffset < 0 {
				m.scrollOffset = 0
			}
		} else if selectedLineIndex < m.scrollOffset {
			// Selected line is above visible range - scroll up
			// Position it with a small buffer from top
			buffer := 2
			m.scrollOffset = selectedLineIndex - buffer
			if m.scrollOffset < 0 {
				m.scrollOffset = 0
			}
		}
	} else {
		// Selected line is visible - only adjust if it's too close to edges
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
			// Not at the bottom: normal edge detection
			// Check if too close to bottom edge
			if selectedLineIndex >= m.scrollOffset+availableHeight-buffer {
				m.scrollOffset = selectedLineIndex - availableHeight + buffer + 1
				if m.scrollOffset < 0 {
					m.scrollOffset = 0
				}
			}

			// Check if too close to top edge
			if selectedLineIndex < m.scrollOffset+buffer {
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

// calculateVisibleItemCount is deprecated - no longer used with line-based scrolling
// Kept for reference but not called
func (m themeSelectionModel) calculateVisibleItemCount(availableHeight int) int {
	if len(m.themes) == 0 {
		return 0
	}

	// availableHeight already accounts for padding, so use it directly
	visibleLines := availableHeight
	if visibleLines < 1 {
		visibleLines = 1
	}

	// Count lines starting from scrollOffset, checking each line individually
	linesUsed := 0
	darkHeaderBlankShown := false
	darkHeaderTextShown := false
	darkHeaderSeparatorShown := false
	lightHeaderBlankShown := false
	lightHeaderTextShown := false
	lightHeaderSeparatorShown := false
	itemCount := 0

	// Check if we need to show header lines before scrollOffset
	if m.scrollOffset > 0 {
		if m.themes[m.scrollOffset-1].Style == "" && m.themes[m.scrollOffset].Style == "dark" {
			// Check each header line individually
			if linesUsed+1 <= visibleLines {
				linesUsed++ // Blank line
				darkHeaderBlankShown = true
			}
			if linesUsed+1 <= visibleLines {
				linesUsed++ // Header text
				darkHeaderTextShown = true
			}
			if linesUsed+1 <= visibleLines {
				linesUsed++ // Separator
				darkHeaderSeparatorShown = true
			}
		}
		if m.scrollOffset > 0 && m.themes[m.scrollOffset-1].Style == "dark" && m.themes[m.scrollOffset].Style == "light" {
			// Check each header line individually
			if linesUsed+1 <= visibleLines {
				linesUsed++ // Blank line
				lightHeaderBlankShown = true
			}
			if linesUsed+1 <= visibleLines {
				linesUsed++ // Header text
				lightHeaderTextShown = true
			}
			if linesUsed+1 <= visibleLines {
				linesUsed++ // Separator
				lightHeaderSeparatorShown = true
			}
		}
	}

	// Count lines for themes starting from scrollOffset, checking each line individually
	// Continue even if some header lines don't fit - include theme if button can fit
	for i := m.scrollOffset; i < len(m.themes); i++ {
		// Check if we need to add header lines before this theme (line by line)
		// Try each line, but don't break if it doesn't fit - continue to check theme button
		if i > 0 && m.themes[i-1].Style == "" && m.themes[i].Style == "dark" {
			if !darkHeaderBlankShown {
				if linesUsed+1 <= visibleLines {
					linesUsed++ // Blank line
					darkHeaderBlankShown = true
				}
				// Continue even if this line didn't fit
			}
			if !darkHeaderTextShown {
				if linesUsed+1 <= visibleLines {
					linesUsed++ // Header text
					darkHeaderTextShown = true
				}
				// Continue even if this line didn't fit
			}
			if !darkHeaderSeparatorShown {
				if linesUsed+1 <= visibleLines {
					linesUsed++ // Separator
					darkHeaderSeparatorShown = true
				}
				// Continue even if this line didn't fit
			}
		}
		if i > 0 && m.themes[i-1].Style == "dark" && m.themes[i].Style == "light" {
			if !lightHeaderBlankShown {
				if linesUsed+1 <= visibleLines {
					linesUsed++ // Blank line
					lightHeaderBlankShown = true
				}
				// Continue even if this line didn't fit
			}
			if !lightHeaderTextShown {
				if linesUsed+1 <= visibleLines {
					linesUsed++ // Header text
					lightHeaderTextShown = true
				}
				// Continue even if this line didn't fit
			}
			if !lightHeaderSeparatorShown {
				if linesUsed+1 <= visibleLines {
					linesUsed++ // Separator
					lightHeaderSeparatorShown = true
				}
				// Continue even if this line didn't fit
			}
		}

		// Add line for theme button - only break if button doesn't fit
		if linesUsed+1 > visibleLines {
			break
		}
		linesUsed++
		itemCount++
	}

	return itemCount
}

// getVisibleThemeRange is deprecated - no longer used with line-based scrolling
// Kept for reference but not called
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

	// Find the last visible theme by counting lines individually
	linesUsed := 0
	darkHeaderBlankShown := false
	darkHeaderTextShown := false
	darkHeaderSeparatorShown := false
	lightHeaderBlankShown := false
	lightHeaderTextShown := false
	lightHeaderSeparatorShown := false

	// Check if we need to show header lines before start index (line by line)
	if start > 0 {
		if m.themes[start-1].Style == "" && m.themes[start].Style == "dark" {
			if linesUsed+1 <= visibleLines {
				linesUsed++ // Blank line
				darkHeaderBlankShown = true
			}
			if linesUsed+1 <= visibleLines {
				linesUsed++ // Header text
				darkHeaderTextShown = true
			}
			if linesUsed+1 <= visibleLines {
				linesUsed++ // Separator
				darkHeaderSeparatorShown = true
			}
		}
		if start > 0 && m.themes[start-1].Style == "dark" && m.themes[start].Style == "light" {
			if linesUsed+1 <= visibleLines {
				linesUsed++ // Blank line
				lightHeaderBlankShown = true
			}
			if linesUsed+1 <= visibleLines {
				linesUsed++ // Header text
				lightHeaderTextShown = true
			}
			if linesUsed+1 <= visibleLines {
				linesUsed++ // Separator
				lightHeaderSeparatorShown = true
			}
		}
	}

	// Count lines for visible themes, checking each header line individually
	// Continue even if some header lines don't fit - include theme if button can fit
	end = start
	for i := start; i < len(m.themes); i++ {
		// Check if we need to add header lines before this theme (line by line)
		// Try each line, but don't break if it doesn't fit - continue to check theme button
		if i > 0 && m.themes[i-1].Style == "" && m.themes[i].Style == "dark" {
			if !darkHeaderBlankShown {
				if linesUsed+1 <= visibleLines {
					linesUsed++ // Blank line
					darkHeaderBlankShown = true
				}
				// Continue even if this line didn't fit
			}
			if !darkHeaderTextShown {
				if linesUsed+1 <= visibleLines {
					linesUsed++ // Header text
					darkHeaderTextShown = true
				}
				// Continue even if this line didn't fit
			}
			if !darkHeaderSeparatorShown {
				if linesUsed+1 <= visibleLines {
					linesUsed++ // Separator
					darkHeaderSeparatorShown = true
				}
				// Continue even if this line didn't fit
			}
		}
		if i > 0 && m.themes[i-1].Style == "dark" && m.themes[i].Style == "light" {
			if !lightHeaderBlankShown {
				if linesUsed+1 <= visibleLines {
					linesUsed++ // Blank line
					lightHeaderBlankShown = true
				}
				// Continue even if this line didn't fit
			}
			if !lightHeaderTextShown {
				if linesUsed+1 <= visibleLines {
					linesUsed++ // Header text
					lightHeaderTextShown = true
				}
				// Continue even if this line didn't fit
			}
			if !lightHeaderSeparatorShown {
				if linesUsed+1 <= visibleLines {
					linesUsed++ // Separator
					lightHeaderSeparatorShown = true
				}
				// Continue even if this line didn't fit
			}
		}

		// Add line for theme button - only break if button doesn't fit
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
