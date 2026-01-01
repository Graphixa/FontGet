package components

import (
	"fmt"

	"fontget/internal/shared"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TableModel wraps CustomTable for dynamic TUI use
type TableModel struct {
	table    *CustomTable
	config   TableConfig
	hasFocus bool
}

// getCellPadding returns the cell padding value (defaults to 1 if not set)
func (tm *TableModel) getCellPadding() int {
	if tm.config.Padding == 0 {
		return 1 // Default padding
	}
	return tm.config.Padding
}

// GetSelectedRow returns the index of the currently selected row
func (tm *TableModel) GetSelectedRow() int {
	return tm.table.Cursor()
}

// SetSelectedRow sets the selected row index
func (tm *TableModel) SetSelectedRow(index int) {
	tm.table.SetCursor(index)
}

// NewTableModel creates a new TableModel for dynamic TUI tables
func NewTableModel(config TableConfig) (*TableModel, error) {
	if len(config.Columns) == 0 {
		return nil, fmt.Errorf("table must have at least one column")
	}

	// Determine table width
	tableWidth := config.Width
	if tableWidth == 0 {
		tableWidth = shared.GetTerminalWidth()
	}

	// Apply maximum width constraint
	maxWidth := config.MaxWidth
	if maxWidth == 0 {
		maxWidth = DefaultMaxTableWidth // Use default if not specified
	}
	if maxWidth > 0 && tableWidth > maxWidth {
		tableWidth = maxWidth
	}

	// Update config with calculated width
	config.Width = tableWidth

	// Create custom table
	customTable, err := NewCustomTable(config)
	if err != nil {
		return nil, err
	}

	return &TableModel{
		table:    customTable,
		config:   config,
		hasFocus: false,
	}, nil
}

// HasFocus returns whether the table currently has focus
func (tm *TableModel) HasFocus() bool {
	return tm.hasFocus
}

// SetFocus sets the focus state of the table
func (tm *TableModel) SetFocus(focused bool) {
	tm.hasFocus = focused
	tm.table.SetFocus(focused)
}

// View returns the rendered table, constrained to terminal width
func (tm *TableModel) View() string {
	tableOutput := tm.table.View()

	// Always constrain table to config.Width (which should be set to terminal width)
	// This ensures proper rendering in small windows
	if tm.config.Width > 0 {
		// Calculate actual rendered width
		tableWidth := lipgloss.Width(tableOutput)

		// If table exceeds terminal width, constrain it
		if tableWidth > tm.config.Width {
			tableOutput = lipgloss.NewStyle().
				Width(tm.config.Width).
				MaxWidth(tm.config.Width).
				Render(tableOutput)
		}
		// Don't center - let table use full width when it fits
	}

	return tableOutput
}

// UpdateWithHeight handles window resize with a specific available height
func (tm *TableModel) UpdateWithHeight(msg tea.WindowSizeMsg, availableHeight int) (*TableModel, tea.Cmd) {
	// Update config width to actual terminal width
	tm.config.Width = msg.Width

	// Apply maximum width constraint if set
	tableWidth := msg.Width
	if tm.config.MaxWidth > 0 && tableWidth > tm.config.MaxWidth {
		tableWidth = tm.config.MaxWidth
	}

	// Recalculate column widths using the same logic as NewCustomTable
	// Account for: cell padding and column separators (spaces)
	// Separators: space between columns = (numColumns - 1) chars
	// Padding: cellPadding on each side of each cell = numColumns * cellPadding * 2
	numColumns := len(tm.config.Columns)
	cellPadding := tm.getCellPadding()
	separatorsWidth := numColumns - 1            // Space separators between columns
	paddingWidth := numColumns * cellPadding * 2 // Padding on both sides of each cell
	availableWidthForColumns := tableWidth - separatorsWidth - paddingWidth
	if availableWidthForColumns < numColumns {
		availableWidthForColumns = numColumns // Minimum width
	}
	if availableWidthForColumns < 0 {
		availableWidthForColumns = numColumns // Fallback for very small terminals
	}

	// Recalculate column widths
	columnWidths := calculateColumnWidths(tm.config, availableWidthForColumns)

	// Update custom table column widths
	tm.table.columnWidths = columnWidths

	// Set viewport height based on available height
	if availableHeight > len(tm.table.rows) {
		availableHeight = len(tm.table.rows)
	}
	if availableHeight < 1 {
		availableHeight = 1
	}
	tm.table.SetHeight(availableHeight)

	// Handle window resize in custom table
	_, cmd := tm.table.Update(msg)

	return tm, cmd
}

// Update handles messages for the table (for tea.Model integration)
func (tm *TableModel) Update(msg interface{}) (*TableModel, tea.Cmd) {
	var cmd tea.Cmd

	// Handle window resize events to recalculate column widths
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Update config width to actual terminal width
		tm.config.Width = msg.Width

		// Apply maximum width constraint if set
		tableWidth := msg.Width
		if tm.config.MaxWidth > 0 && tableWidth > tm.config.MaxWidth {
			tableWidth = tm.config.MaxWidth
		}

		// Recalculate column widths using the same logic as NewCustomTable
		// Account for: cell padding and column separators (spaces)
		// Separators: space between columns = (numColumns - 1) chars
		// Padding: cellPadding on each side of each cell = numColumns * cellPadding * 2
		numColumns := len(tm.config.Columns)
		cellPadding := tm.getCellPadding()
		separatorsWidth := numColumns - 1            // Space separators between columns
		paddingWidth := numColumns * cellPadding * 2 // Padding on both sides of each cell
		availableWidthForColumns := tableWidth - separatorsWidth - paddingWidth
		if availableWidthForColumns < numColumns {
			availableWidthForColumns = numColumns // Minimum width
		}
		if availableWidthForColumns < 0 {
			availableWidthForColumns = numColumns // Fallback for very small terminals
		}

		// Recalculate column widths
		columnWidths := calculateColumnWidths(tm.config, availableWidthForColumns)

		// Update custom table column widths
		tm.table.SetColumnWidths(columnWidths)

		// Handle window resize in custom table
		tm.table, cmd = tm.table.Update(msg)
		return tm, cmd
	}

	// Delegate all other messages to CustomTable - it handles smart scrolling
	if tm.hasFocus {
		tm.table, cmd = tm.table.Update(msg)
	}

	return tm, cmd
}
