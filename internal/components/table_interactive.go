package components

import (
	"fmt"

	"fontget/internal/shared"
	"fontget/internal/ui"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TableModel wraps bubbles/table.Model for dynamic TUI use
type TableModel struct {
	table    table.Model
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

	// Calculate column widths - the bubbles/table component will handle padding internally
	// We'll account for padding by using a slightly smaller available width
	numColumns := len(config.Columns)
	// Get padding value (default to 1 if not set)
	cellPadding := config.Padding
	if cellPadding == 0 {
		cellPadding = 1 // Default padding
	}
	// Reserve space for padding (cellPadding char left + cellPadding char right per column) and separators
	paddingAndSeparators := numColumns*cellPadding*2 + (numColumns - 1)
	availableWidthForColumns := tableWidth - paddingAndSeparators
	if availableWidthForColumns < numColumns {
		availableWidthForColumns = numColumns // Minimum width
	}

	// Calculate column widths based on available width
	columnWidths := calculateColumnWidths(config, availableWidthForColumns)

	// Build table columns
	tableColumns := make([]table.Column, len(config.Columns))
	for i, col := range config.Columns {
		tableColumns[i] = table.Column{
			Title: col.Header,
			Width: columnWidths[i],
		}
	}

	// Build table rows
	tableRows := make([]table.Row, len(config.Rows))
	for i, row := range config.Rows {
		formattedRow := make([]string, len(config.Columns))
		for j := range formattedRow {
			if j < len(row) {
				formattedRow[j] = row[j]
			} else {
				formattedRow[j] = ""
			}
		}
		tableRows[i] = formattedRow
	}

	// Determine height
	height := config.Height
	if height == 0 {
		height = len(tableRows)
		if height > 20 {
			height = 20 // Max height for TUI
		}
	}

	// Update columns with calculated widths
	for i := range tableColumns {
		tableColumns[i].Width = columnWidths[i]
	}

	// Create table
	t := table.New(
		table.WithColumns(tableColumns),
		table.WithRows(tableRows),
		table.WithFocused(false),
		table.WithHeight(height),
	)

	// Don't set a fixed width - let the table size itself based on columns
	// The View method will center it if needed

	// Apply theme styles
	styles := table.DefaultStyles()
	headerStyle := ui.TableHeader.Copy()
	styles.Header = headerStyle.
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderBottom(true).
		BorderLeft(false).
		BorderRight(false).
		Padding(0, cellPadding)

	cellStyle := ui.Text.Copy()
	styles.Cell = cellStyle.Padding(0, cellPadding)

	// Selected row: use inverted colors (like ButtonSelected) and no padding (no indent)
	styles.Selected = ui.TableRowSelected.Copy().
		Padding(0, 0) // No padding for selected row to remove indent

	t.SetStyles(styles)

	return &TableModel{
		table:    t,
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
	// Recreate table with new focus state
	tm.table = table.New(
		table.WithColumns(tm.table.Columns()),
		table.WithRows(tm.table.Rows()),
		table.WithFocused(focused),
		table.WithHeight(tm.table.Height()),
	)
	// Reapply styles
	cellPadding := tm.getCellPadding()
	styles := table.DefaultStyles()
	headerStyle := ui.TableHeader.Copy()
	styles.Header = headerStyle.
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderBottom(true).
		BorderLeft(false).
		BorderRight(false).
		Padding(0, cellPadding)

	cellStyle := ui.Text.Copy()
	styles.Cell = cellStyle.Padding(0, cellPadding)

	// Selected row: use inverted colors (like ButtonSelected) and no padding (no indent)
	styles.Selected = ui.TableRowSelected.Copy().
		Padding(0, 0) // No padding for selected row to remove indent

	tm.table.SetStyles(styles)
}

// View returns the rendered table, centered if there's extra space
func (tm *TableModel) View() string {
	tableOutput := tm.table.View()

	// Calculate the actual rendered width of the table
	tableWidth := lipgloss.Width(tableOutput)

	// If table is narrower than terminal width, center it
	if tm.config.Width > 0 && tableWidth < tm.config.Width {
		// Center the table horizontally
		padding := (tm.config.Width - tableWidth) / 2
		if padding > 0 {
			tableOutput = lipgloss.NewStyle().
				PaddingLeft(padding).
				Render(tableOutput)
		}
	}

	return tableOutput
}

// UpdateWithHeight handles window resize with a specific available height
func (tm *TableModel) UpdateWithHeight(msg tea.WindowSizeMsg, availableHeight int) (*TableModel, tea.Cmd) {
	// Preserve current state
	selectedRow := tm.table.Cursor()
	rows := tm.table.Rows()

	// Calculate column widths - the bubbles/table component will handle padding internally
	// We'll account for padding by using a slightly smaller available width
	numColumns := len(tm.config.Columns)
	cellPadding := tm.getCellPadding()
	// Reserve space for padding (cellPadding char left + cellPadding char right per column) and separators
	paddingAndSeparators := numColumns*cellPadding*2 + (numColumns - 1)
	availableWidthForColumns := msg.Width - paddingAndSeparators
	if availableWidthForColumns < numColumns {
		availableWidthForColumns = numColumns // Minimum width
	}

	// Recalculate column widths based on available width
	columnWidths := calculateColumnWidths(tm.config, availableWidthForColumns)

	// Update table columns with new widths
	tableColumns := make([]table.Column, len(tm.config.Columns))
	for i, col := range tm.config.Columns {
		tableColumns[i] = table.Column{
			Title: col.Header,
			Width: columnWidths[i],
		}
	}

	// Calculate table height based on available space
	tableHeight := availableHeight
	if tableHeight > len(rows) {
		tableHeight = len(rows)
	}
	if tableHeight < 1 {
		tableHeight = 1
	}

	// Update columns with calculated widths
	for i := range tableColumns {
		tableColumns[i].Width = columnWidths[i]
	}

	// Recreate table with new dimensions
	tm.table = table.New(
		table.WithColumns(tableColumns),
		table.WithRows(rows),
		table.WithFocused(tm.hasFocus),
		table.WithHeight(tableHeight),
	)

	// Don't set a fixed width - let the table size itself based on columns
	// The View method will center it if needed

	// Restore selected row (ensure it's within bounds)
	if selectedRow >= len(rows) {
		selectedRow = len(rows) - 1
	}
	if selectedRow < 0 {
		selectedRow = 0
	}
	tm.table.SetCursor(selectedRow)

	// Reapply styles (cellPadding already declared above)
	styles := table.DefaultStyles()
	headerStyle := ui.TableHeader.Copy()
	styles.Header = headerStyle.
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderBottom(true).
		BorderLeft(false).
		BorderRight(false).
		Padding(0, cellPadding)

	cellStyle := ui.Text.Copy()
	styles.Cell = cellStyle.Padding(0, cellPadding)

	// Selected row: use inverted colors (like ButtonSelected) and no padding (no indent)
	styles.Selected = ui.TableRowSelected.Copy().
		Padding(0, 0) // No padding for selected row to remove indent

	tm.table.SetStyles(styles)

	// Update config width
	tm.config.Width = msg.Width

	return tm, nil
}

// Update handles messages for the table (for tea.Model integration)
func (tm *TableModel) Update(msg interface{}) (*TableModel, tea.Cmd) {
	var cmd tea.Cmd

	// Handle window resize events to recalculate column widths
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Preserve current state
		selectedRow := tm.table.Cursor()
		rows := tm.table.Rows()

		// Calculate column widths - the bubbles/table component will handle padding internally
		// We'll account for padding by using a slightly smaller available width
		numColumns := len(tm.config.Columns)
		cellPadding := tm.getCellPadding()
		// Reserve space for padding (cellPadding char left + cellPadding char right per column) and separators
		paddingAndSeparators := numColumns*cellPadding*2 + (numColumns - 1)
		availableWidthForColumns := msg.Width - paddingAndSeparators
		if availableWidthForColumns < numColumns {
			availableWidthForColumns = numColumns // Minimum width
		}

		// Recalculate column widths based on available width
		columnWidths := calculateColumnWidths(tm.config, availableWidthForColumns)

		// Update table columns with new widths
		tableColumns := make([]table.Column, len(tm.config.Columns))
		for i, col := range tm.config.Columns {
			tableColumns[i] = table.Column{
				Title: col.Header,
				Width: columnWidths[i],
			}
		}

		// Calculate table height - use config height if set, otherwise use current height
		tableHeight := tm.config.Height
		if tableHeight == 0 {
			// Keep current height or use number of rows
			tableHeight = tm.table.Height()
			if tableHeight == 0 {
				tableHeight = len(rows)
				if tableHeight > 20 {
					tableHeight = 20
				}
			}
		}

		// Recreate table with new column widths
		tm.table = table.New(
			table.WithColumns(tableColumns),
			table.WithRows(rows),
			table.WithFocused(tm.hasFocus),
			table.WithHeight(tableHeight),
		)

		// Don't set a fixed width - let the table size itself based on columns
		// The View method will center it if needed

		// Restore selected row (ensure it's within bounds)
		if selectedRow >= len(rows) {
			selectedRow = len(rows) - 1
		}
		if selectedRow < 0 {
			selectedRow = 0
		}
		tm.table.SetCursor(selectedRow)

		// Reapply styles (cellPadding already declared above)
		styles := table.DefaultStyles()
		headerStyle := ui.TableHeader.Copy()
		styles.Header = headerStyle.
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderBottom(true).
			BorderLeft(false).
			BorderRight(false).
			Padding(0, cellPadding)

		cellStyle := ui.Text.Copy()
		styles.Cell = cellStyle.Padding(0, cellPadding)

		// Selected row: use inverted colors (like ButtonSelected) and no padding (no indent)
		styles.Selected = ui.TableRowSelected.Copy().
			Padding(0, 0) // No padding for selected row to remove indent

		tm.table.SetStyles(styles)

		// Update config width
		tm.config.Width = msg.Width

		return tm, nil
	default:
		if tm.hasFocus {
			tm.table, cmd = tm.table.Update(msg)
		}
		return tm, cmd
	}
}
