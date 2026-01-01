package components

import (
	"fmt"
	"strings"

	"fontget/internal/shared"
	"fontget/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CustomTable is a custom table component with full viewport control
type CustomTable struct {
	columns        []ColumnConfig
	rows           [][]string
	cursor         int // Current selected row
	viewportStart  int // First visible row index
	viewportHeight int // Number of visible rows
	hasFocus       bool
	cellPadding    int
	columnWidths   []int
	headerStyle    lipgloss.Style
	cellStyle      lipgloss.Style
	selectedStyle  lipgloss.Style
}

// NewCustomTable creates a new CustomTable instance
func NewCustomTable(config TableConfig) (*CustomTable, error) {
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

	// Get padding value (default to 1 if not set)
	cellPadding := config.Padding
	if cellPadding == 0 {
		cellPadding = 1 // Default padding
	}

	// Reserve space for padding and separators
	numColumns := len(config.Columns)
	paddingAndSeparators := numColumns*cellPadding*2 + (numColumns - 1)
	availableWidthForColumns := tableWidth - paddingAndSeparators
	if availableWidthForColumns < numColumns {
		availableWidthForColumns = numColumns // Minimum width
	}

	// Calculate column widths based on available width
	columnWidths := calculateColumnWidths(config, availableWidthForColumns)

	// Determine viewport height
	viewportHeight := config.Height
	if viewportHeight == 0 {
		viewportHeight = len(config.Rows)
		if viewportHeight > 20 {
			viewportHeight = 20 // Max height for TUI
		}
	}

	// Initialize styles
	headerStyle := ui.TableHeader.Copy().
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderBottom(true).
		BorderLeft(false).
		BorderRight(false).
		Padding(0, cellPadding)

	cellStyle := ui.Text.Copy().
		Padding(0, cellPadding)

	selectedStyle := ui.TableRowSelected.Copy().
		Padding(0, 0) // No padding for selected row to remove indent

	return &CustomTable{
		columns:        config.Columns,
		rows:           config.Rows,
		cursor:         0,
		viewportStart:  0,
		viewportHeight: viewportHeight,
		hasFocus:       false,
		cellPadding:    cellPadding,
		columnWidths:   columnWidths,
		headerStyle:    headerStyle,
		cellStyle:      cellStyle,
		selectedStyle:  selectedStyle,
	}, nil
}

// Cursor returns the current cursor position
func (ct *CustomTable) Cursor() int {
	return ct.cursor
}

// SetCursor sets the cursor position and adjusts viewport if needed
func (ct *CustomTable) SetCursor(index int) {
	if index < 0 {
		index = 0
	}
	if index >= len(ct.rows) {
		index = len(ct.rows) - 1
	}
	ct.cursor = index
	ct.ensureCursorVisible()
}

// ViewportStart returns the first visible row index
func (ct *CustomTable) ViewportStart() int {
	return ct.viewportStart
}

// ViewportEnd returns the last visible row index
func (ct *CustomTable) ViewportEnd() int {
	end := ct.viewportStart + ct.viewportHeight - 1
	if end >= len(ct.rows) {
		end = len(ct.rows) - 1
	}
	return end
}

// Height returns the viewport height
func (ct *CustomTable) Height() int {
	return ct.viewportHeight
}

// Rows returns all table rows
func (ct *CustomTable) Rows() [][]string {
	return ct.rows
}

// Columns returns the column configuration
func (ct *CustomTable) Columns() []ColumnConfig {
	return ct.columns
}

// ensureCursorVisible adjusts the viewport to ensure the cursor is visible
func (ct *CustomTable) ensureCursorVisible() {
	viewportEnd := ct.ViewportEnd()

	if ct.cursor < ct.viewportStart {
		// Cursor above viewport - scroll up
		ct.viewportStart = ct.cursor
	} else if ct.cursor > viewportEnd {
		// Cursor below viewport - scroll down
		ct.viewportStart = ct.cursor - ct.viewportHeight + 1
		if ct.viewportStart < 0 {
			ct.viewportStart = 0
		}
	}
	// If cursor is within viewport, no change needed
}

// handleNavigation handles keyboard navigation with smart scrolling
func (ct *CustomTable) handleNavigation(key string) {
	var newCursor int
	if key == "up" {
		newCursor = ct.cursor - 1
		if newCursor < 0 {
			newCursor = 0
		}
	} else if key == "down" {
		newCursor = ct.cursor + 1
		if newCursor >= len(ct.rows) {
			newCursor = len(ct.rows) - 1
		}
	} else {
		return // Not a navigation key
	}

	viewportEnd := ct.ViewportEnd()

	// Check if cursor would stay in viewport
	if newCursor >= ct.viewportStart && newCursor <= viewportEnd {
		// Cursor stays visible - just update, no scroll
		ct.cursor = newCursor
	} else {
		// Cursor would go off-screen - update and scroll
		ct.cursor = newCursor
		ct.ensureCursorVisible()
	}
}

// renderHeader renders the table header
func (ct *CustomTable) renderHeader() string {
	var cells []string
	for i, col := range ct.columns {
		width := ct.columnWidths[i]
		headerText := col.Header
		if len(headerText) > width {
			headerText = truncateString(headerText, width)
		}
		formatted, _ := formatCell(headerText, width, col.Align, true)
		cells = append(cells, ct.headerStyle.Render(formatted))
	}
	return strings.Join(cells, " ")
}

// renderRow renders a single row
func (ct *CustomTable) renderRow(rowIndex int, isSelected bool) string {
	if rowIndex < 0 || rowIndex >= len(ct.rows) {
		return ""
	}

	row := ct.rows[rowIndex]
	var cells []string
	for i, col := range ct.columns {
		width := ct.columnWidths[i]
		var cellValue string
		if i < len(row) {
			cellValue = row[i]
		}
		formatted, _ := formatCell(cellValue, width, col.Align, col.Truncatable)

		if isSelected {
			cells = append(cells, ct.selectedStyle.Render(formatted))
		} else {
			cells = append(cells, ct.cellStyle.Render(formatted))
		}
	}
	return strings.Join(cells, " ")
}

// View renders the table
func (ct *CustomTable) View() string {
	if len(ct.rows) == 0 {
		return ""
	}

	// Render header
	header := ct.renderHeader()

	// Render visible rows only
	var rows []string
	viewportEnd := ct.ViewportEnd()
	for i := ct.viewportStart; i <= viewportEnd && i < len(ct.rows); i++ {
		row := ct.renderRow(i, i == ct.cursor)
		rows = append(rows, row)
	}

	return strings.Join(append([]string{header}, rows...), "\n")
}

// Update handles messages for the table
func (ct *CustomTable) Update(msg tea.Msg) (*CustomTable, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !ct.hasFocus {
			return ct, nil
		}

		key := msg.String()
		if key == "up" || key == "down" {
			ct.handleNavigation(key)
			return ct, nil
		}

	case tea.WindowSizeMsg:
		// Handle window resize - recalculate column widths
		numColumns := len(ct.columns)
		paddingAndSeparators := numColumns*ct.cellPadding*2 + (numColumns - 1)
		availableWidthForColumns := msg.Width - paddingAndSeparators
		if availableWidthForColumns < numColumns {
			availableWidthForColumns = numColumns
		}

		// Recalculate column widths
		config := TableConfig{
			Columns: ct.columns,
			Rows:    ct.rows,
			Width:   msg.Width,
			Padding: ct.cellPadding,
		}
		ct.columnWidths = calculateColumnWidths(config, availableWidthForColumns)

		// Ensure cursor is still visible after resize
		ct.ensureCursorVisible()
		return ct, nil
	}

	return ct, nil
}

// SetFocus sets the focus state
func (ct *CustomTable) SetFocus(focused bool) {
	ct.hasFocus = focused
}

// SetHeight sets the viewport height and adjusts viewport if needed
func (ct *CustomTable) SetHeight(height int) {
	if height < 1 {
		height = 1
	}
	ct.viewportHeight = height
	ct.ensureCursorVisible()
}

// SetColumnWidths sets the column widths
func (ct *CustomTable) SetColumnWidths(widths []int) {
	ct.columnWidths = widths
}
