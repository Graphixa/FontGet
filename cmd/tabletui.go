package cmd

import (
	"fmt"
	"strings"

	"fontget/internal/components"
	"fontget/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// tableTuiModel represents the model for the TUI table test
type tableTuiModel struct {
	tableModel *components.TableModel
	width      int
	height     int
	help       string
	quitting   bool
}

// Init initializes the TUI model
func (m tableTuiModel) Init() tea.Cmd {
	return nil
}

// calculateAvailableHeight calculates the available height for the table
// Accounts for title, instructions, help text, blank line between table and controls, and margins
// Returns the exact height the table should use (including table header and border lines)
func (m tableTuiModel) calculateAvailableHeight() int {
	// Count actual header lines
	header := ui.PageTitle.Render("Dynamic TUI Table Test") + "\n\n" +
		ui.Text.Render("This is a dynamic TUI table that resizes with the terminal window.") + "\n" +
		ui.Text.Render("Try resizing your terminal to see the table adjust automatically.") + "\n\n"
	headerLines := strings.Count(header, "\n")

	// Help text is 5 lines (blank line + "Controls:" + 3 instruction lines)
	// Plus 1 blank line between table and controls
	helpLines := 5 + 1 // 5 for help text + 1 blank line

	// Table has 3 border lines: top, below header, and bottom
	// These are accounted for in SetHeight, so we don't need to subtract them here
	// The table component will handle the border lines internally

	// Available height = total height - header - help
	availableHeight := m.height - headerLines - helpLines

	// Ensure minimum height (at least 3 rows visible + border lines + header)
	if availableHeight < 6 {
		availableHeight = 6 // Minimum: 3 border lines + 1 header + 2 rows
	}
	return availableHeight
}

// Update handles messages
func (m tableTuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate available height for table
		availableHeight := m.calculateAvailableHeight()

		// Update table with new size and available height
		// Pass the actual terminal width to ensure proper sizing
		if m.tableModel != nil {
			// Update table with new size and available height
			// The UpdateWithHeight method will update the config width internally
			updated, cmd := m.tableModel.UpdateWithHeight(msg, availableHeight)
			if updated != nil {
				m.tableModel = updated
			}
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}

	if m.tableModel != nil {
		updated, cmd := m.tableModel.Update(msg)
		if updated != nil {
			m.tableModel = updated
		}
		return m, cmd
	}

	return m, nil
}

// View renders the TUI
func (m tableTuiModel) View() string {
	if m.quitting {
		return ""
	}

	// If width/height not set yet, return empty (will be set by WindowSizeMsg)
	if m.width == 0 || m.height == 0 {
		return ""
	}

	// Build title and instructions (fixed at top)
	header := ui.PageTitle.Render("Dynamic TUI Table Test") + "\n\n" +
		ui.Text.Render("This is a dynamic TUI table that resizes with the terminal window.") + "\n" +
		ui.Text.Render("Try resizing your terminal to see the table adjust automatically.") + "\n\n"

	// Table - the table component handles its own width constraints
	var tableOutput string
	if m.tableModel != nil {
		tableOutput = m.tableModel.View()
	}

	// Build help text (at bottom, but only if there's space)
	var help string
	headerLines := strings.Count(header, "\n")
	tableLines := strings.Count(tableOutput, "\n")
	helpHeight := 4
	remainingHeight := m.height - headerLines - tableLines
	if remainingHeight >= helpHeight {
		help = "\n" +
			ui.TextBold.Render("Controls:") + "\n" +
			ui.Text.Render("  ↑/↓ or j/k - Navigate table rows") + "\n" +
			ui.Text.Render("  q, Esc, or Ctrl+C - Quit") + "\n" +
			ui.Text.Render("  Resize terminal - Table adjusts automatically")
	}

	// Combine: header (top) + table + blank line + help (bottom if space)
	// Build result string directly to avoid lipgloss wrapping issues
	var result strings.Builder
	result.WriteString(header)
	result.WriteString(tableOutput)
	if help != "" {
		result.WriteString("\n") // Always add a blank line between table and controls
		result.WriteString(help)
	}

	// Constrain final output to terminal width to prevent wrapping
	output := result.String()
	if m.width > 0 {
		output = lipgloss.NewStyle().
			Width(m.width).
			MaxWidth(m.width).
			Render(output)
	}

	return output
}

var tableTuiCmd = &cobra.Command{
	Use:   "tabletui",
	Short: "Test dynamic TUI table component",
	Long: `Test command for visually verifying the dynamic TUI table component.
This displays an interactive table that:
- Resizes dynamically with terminal window
- Supports keyboard navigation
- Applies theme styling
- Demonstrates percentage-based and min/max column widths`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create sample table data
		tableConfig := components.TableConfig{
			Columns: []components.ColumnConfig{
				{Header: "Font Name", MinWidth: 20, PercentWidth: 35.0},
				{Header: "Font ID", MinWidth: 15, PercentWidth: 30.0},
				{Header: "License", MinWidth: 8, MaxWidth: 15, PercentWidth: 12.0},
				{Header: "Categories", MinWidth: 12, PercentWidth: 15.0},
				{Header: "Source", MinWidth: 12, PercentWidth: 8.0},
			},
			Padding: 0, // Reduced padding for tighter layout
			Rows: [][]string{
				{"Roboto", "google.roboto", "Apache 2.0", "Sans Serif", "Google Fonts"},
				{"Fira Code", "nerd.fira-code", "SIL OFL", "Monospace", "Nerd Fonts"},
				{"JetBrains Mono", "jetbrains.jetbrains-mono", "SIL OFL", "Monospace", "JetBrains"},
				{"Inter", "rsms.inter", "SIL OFL", "Sans Serif", "Rasmus Andersson"},
				{"Source Code Pro", "adobe.source-code-pro", "SIL OFL", "Monospace", "Adobe"},
				{"Open Sans", "google.open-sans", "Apache 2.0", "Sans Serif", "Google Fonts"},
				{"Lato", "google.lato", "SIL OFL", "Sans Serif", "Google Fonts"},
				{"Montserrat", "google.montserrat", "SIL OFL", "Sans Serif", "Google Fonts"},
				{"Raleway", "google.raleway", "SIL OFL", "Sans Serif", "Google Fonts"},
				{"Poppins", "google.poppins", "SIL OFL", "Sans Serif", "Google Fonts"},
				{"Ubuntu", "canonical.ubuntu", "Ubuntu Font License", "Sans Serif", "Canonical"},
				{"Noto Sans", "google.noto-sans", "SIL OFL", "Sans Serif", "Google Fonts"},
				{"Playfair Display", "google.playfair-display", "SIL OFL", "Serif", "Google Fonts"},
				{"Merriweather", "google.merriweather", "SIL OFL", "Serif", "Google Fonts"},
				{"Oswald", "google.oswald", "SIL OFL", "Sans Serif", "Google Fonts"},
			},
			Width:  0,  // Auto-detect
			Height: 10, // Show 10 rows at a time
			Mode:   components.TableModeDynamic,
		}

		// Create table model
		tableModel, err := components.NewTableModel(tableConfig)
		if err != nil {
			return fmt.Errorf("failed to create table model: %w", err)
		}

		// Set focus
		tableModel.SetFocus(true)

		// Create TUI model
		// Initial width/height will be set by first WindowSizeMsg
		model := tableTuiModel{
			tableModel: tableModel,
			width:      0, // Will be set by WindowSizeMsg
			height:     0, // Will be set by WindowSizeMsg
			help:       "Use arrow keys to navigate, q to quit",
		}

		// Run TUI
		p := tea.NewProgram(model, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("failed to run TUI: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(tableTuiCmd)
}
