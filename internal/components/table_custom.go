package components

import (
	"fmt"

	"fontget/internal/shared"
	"fontget/internal/ui"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// CustomTable is a custom table component with full viewport control
type CustomTable struct {
	columns       []ColumnConfig
	rows          [][]string
	cursor        int // Current selected row
	hasFocus      bool
	cellPadding   int
	columnWidths  []int
	headerStyle   lipgloss.Style
	cellStyle     lipgloss.Style
	selectedStyle lipgloss.Style
	tableWidth    int // Total table width
	viewport      viewport.Model
	start         int // First visible row index
	end           int // Last visible row index
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

	// Reserve space for padding and column separators (spaces)
	// Separators: space between columns = (numColumns - 1) chars
	// Padding: cellPadding on each side of each cell = numColumns * cellPadding * 2
	numColumns := len(config.Columns)
	separatorsWidth := numColumns - 1            // Space separators between columns
	paddingWidth := numColumns * cellPadding * 2 // Padding on both sides of each cell
	availableWidthForColumns := tableWidth - separatorsWidth - paddingWidth
	if availableWidthForColumns < numColumns {
		availableWidthForColumns = numColumns // Minimum width
	}

	// Calculate column widths based on available width
	// Note: calculateColumnWidths expects the raw available width for column content
	// Padding is handled separately via lipgloss styles, not in the width calculation
	columnWidths := calculateColumnWidths(config, availableWidthForColumns)

	// Initialize viewport
	// Height will be set later via SetHeight, use a default for now
	viewportHeight := config.Height
	if viewportHeight == 0 {
		viewportHeight = len(config.Rows)
		if viewportHeight > 20 {
			viewportHeight = 20 // Max height for TUI
		}
	}

	// Initialize styles
	headerStyle := ui.TableHeader.Copy().
		Padding(0, cellPadding)

	cellStyle := ui.Text.Copy().
		Padding(0, cellPadding)

	// Selected style - will be applied per cell (no padding for selected rows)
	selectedTextStyle := ui.TableRowSelected.Copy().
		Padding(0, 0)

	// Calculate total table width (including separators and padding)
	// Total width = sum of column widths + separators between columns + padding
	totalColumnWidth := 0
	for _, w := range columnWidths {
		totalColumnWidth += w
	}
	calculatedTableWidth := totalColumnWidth + separatorsWidth + paddingWidth

	// Create viewport with calculated width
	// Width will be updated on WindowSizeMsg, but initialize with calculated width
	vp := viewport.New(calculatedTableWidth, viewportHeight)

	ct := &CustomTable{
		columns:       config.Columns,
		rows:          config.Rows,
		cursor:        0,
		hasFocus:      false,
		cellPadding:   cellPadding,
		columnWidths:  columnWidths,
		headerStyle:   headerStyle,
		cellStyle:     cellStyle,
		selectedStyle: selectedTextStyle,
		tableWidth:    calculatedTableWidth,
		viewport:      vp,
		start:         0,
		end:           viewportHeight,
	}

	// Update viewport content
	ct.UpdateViewport()

	return ct, nil
}

// Cursor returns the current cursor position
func (ct *CustomTable) Cursor() int {
	return ct.cursor
}

// SetCursor sets the cursor position and adjusts viewport if needed
func (ct *CustomTable) SetCursor(index int) {
	ct.cursor = clamp(index, 0, len(ct.rows)-1)
	ct.ensureCursorVisible()
}

// ViewportStart returns the first visible row index
func (ct *CustomTable) ViewportStart() int {
	return ct.start
}

// ViewportEnd returns the last visible row index
func (ct *CustomTable) ViewportEnd() int {
	return ct.end
}

// Height returns the viewport height
func (ct *CustomTable) Height() int {
	return ct.viewport.Height
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
	// If cursor is before start, move start to cursor
	if ct.cursor < ct.start {
		ct.start = ct.cursor
		ct.end = ct.start + ct.viewport.Height
		if ct.end > len(ct.rows) {
			ct.end = len(ct.rows)
		}
	} else if ct.cursor >= ct.end {
		// Cursor is after end - move viewport to show cursor at the start
		ct.start = ct.cursor
		ct.end = ct.start + ct.viewport.Height
		if ct.end > len(ct.rows) {
			ct.end = len(ct.rows)
			// Adjust start to show viewport.Height rows if we hit the end
			ct.start = ct.end - ct.viewport.Height
			if ct.start < 0 {
				ct.start = 0
			}
		}
	}
	// Update viewport content
	ct.UpdateViewport()
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

	ct.cursor = newCursor
	ct.ensureCursorVisible()
}

// renderHeader renders the table header
func (ct *CustomTable) renderHeader() string {
	var cells []string
	for i, col := range ct.columns {
		width := ct.columnWidths[i]
		if width <= 0 {
			continue
		}

		// Use runewidth.Truncate for proper truncation
		headerText := runewidth.Truncate(col.Header, width, "…")

		// Format cell with alignment
		formatted, _ := formatCell(headerText, width, col.Align, true)

		// Apply style with width constraint and inline to prevent wrapping
		style := lipgloss.NewStyle().Width(width).MaxWidth(width).Inline(true)
		renderedCell := style.Render(formatted)
		styledCell := ct.headerStyle.Render(renderedCell)
		cells = append(cells, styledCell)
	}

	// Join cells horizontally (like bubbles table)
	return lipgloss.JoinHorizontal(lipgloss.Top, cells...)
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
		if width <= 0 {
			continue
		}

		var cellValue string
		if i < len(row) {
			cellValue = row[i]
		}

		// Use runewidth.Truncate for proper truncation if truncatable
		var truncated string
		if col.Truncatable {
			truncated = runewidth.Truncate(cellValue, width, "…")
		} else {
			truncated = cellValue
		}

		// Format cell with alignment
		formatted, _ := formatCell(truncated, width, col.Align, col.Truncatable)

		// Apply style with width constraint and inline to prevent wrapping
		style := lipgloss.NewStyle().Width(width).MaxWidth(width).Inline(true)
		renderedCell := style.Render(formatted)

		// Apply the same cell style with padding to all rows (selected or not)
		// This ensures alignment is exactly the same for all rows
		styledCell := ct.cellStyle.Render(renderedCell)
		cells = append(cells, styledCell)
	}

	// Join cells horizontally (like bubbles table)
	rowContent := lipgloss.JoinHorizontal(lipgloss.Top, cells...)

	// Apply selected style to entire row if selected (background color and inverted text only, no additional padding)
	if isSelected {
		// Apply selected style to entire row - background and foreground only
		// The rowContent already has the correct padding from cellStyle, so we just apply colors
		return ct.selectedStyle.Padding(0, 0).Render(rowContent)
	}

	return rowContent
}

// UpdateViewport updates the viewport content based on visible rows
func (ct *CustomTable) UpdateViewport() {
	if len(ct.rows) == 0 {
		ct.viewport.SetContent("")
		return
	}

	// Calculate visible row range - show exactly viewport.Height rows
	// Start from cursor and show viewport.Height rows going forward
	ct.start = ct.cursor
	if ct.start < 0 {
		ct.start = 0
	}

	// End is start + viewport height, but don't exceed total rows
	ct.end = ct.start + ct.viewport.Height
	if ct.end > len(ct.rows) {
		ct.end = len(ct.rows)
		// If we hit the end, adjust start to show viewport.Height rows
		ct.start = ct.end - ct.viewport.Height
		if ct.start < 0 {
			ct.start = 0
		}
	}

	// Render visible rows only
	renderedRows := make([]string, 0, ct.end-ct.start)
	for i := ct.start; i < ct.end; i++ {
		renderedRows = append(renderedRows, ct.renderRow(i, i == ct.cursor))
	}

	// Set viewport content
	ct.viewport.SetContent(
		lipgloss.JoinVertical(lipgloss.Left, renderedRows...),
	)
}

// View renders the table
func (ct *CustomTable) View() string {
	if len(ct.rows) == 0 {
		return ""
	}

	// Render header
	header := ct.renderHeader()

	// Render viewport (rows)
	rowsView := ct.viewport.View()

	// Combine header and rows (like bubbles table)
	return header + "\n" + rowsView
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
		// Account for: cell padding and column separators (spaces)
		// Separators: space between columns = (numColumns - 1) chars
		// Padding: cellPadding on each side of each cell = numColumns * cellPadding * 2
		numColumns := len(ct.columns)
		separatorsWidth := (numColumns - 1)             // Space separators between columns
		paddingWidth := numColumns * ct.cellPadding * 2 // Padding on both sides of each cell
		availableWidthForColumns := msg.Width - separatorsWidth - paddingWidth
		if availableWidthForColumns < numColumns {
			availableWidthForColumns = numColumns // Minimum width
		}
		if availableWidthForColumns < 0 {
			availableWidthForColumns = numColumns // Fallback for very small terminals
		}

		// Recalculate column widths
		config := TableConfig{
			Columns: ct.columns,
			Rows:    ct.rows,
			Width:   msg.Width,
			Padding: ct.cellPadding,
		}
		ct.columnWidths = calculateColumnWidths(config, availableWidthForColumns)

		// Update table width
		totalColumnWidth := 0
		for _, w := range ct.columnWidths {
			totalColumnWidth += w
		}
		ct.tableWidth = totalColumnWidth + separatorsWidth + paddingWidth

		// Update viewport width
		ct.viewport.Width = msg.Width

		// Update viewport content
		ct.UpdateViewport()

		// Ensure cursor is still visible after resize
		ct.ensureCursorVisible()

		// Update viewport (handles scrolling)
		var cmd tea.Cmd
		ct.viewport, cmd = ct.viewport.Update(msg)
		return ct, cmd
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
	// Set viewport height (accounting for header)
	headerHeight := lipgloss.Height(ct.renderHeader())
	ct.viewport.Height = height - headerHeight
	if ct.viewport.Height < 1 {
		ct.viewport.Height = 1
	}
	ct.UpdateViewport()
	ct.ensureCursorVisible()
}

// SetWidth sets the viewport width
func (ct *CustomTable) SetWidth(width int) {
	ct.viewport.Width = width
	ct.UpdateViewport()
}

// SetColumnWidths sets the column widths
func (ct *CustomTable) SetColumnWidths(widths []int) {
	ct.columnWidths = widths
}
