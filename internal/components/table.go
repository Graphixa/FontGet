package components

import (
	"fmt"
	"strings"

	"fontget/internal/shared"
	"fontget/internal/ui"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TableMode determines how the table is rendered
type TableMode int

const (
	TableModeStatic  TableMode = iota // Static CLI rendering
	TableModeDynamic                  // Dynamic TUI (for future)
)

// ColumnConfig defines column properties
//
// DEFAULT CONFIGURATION PATTERN (recommended for all FontGet tables):
// Use percentage-based widths (PercentWidth) combined with MinWidth constraints:
//   - First column (names/identifiers): PercentWidth: 35-40%, MinWidth: 20
//   - Second column (IDs/codes): PercentWidth: 25-30%, MinWidth: 15
//   - Fixed-size columns (license, type): PercentWidth: 10-15%, MinWidth: 8, MaxWidth: 15
//   - Last column (source/meta): PercentWidth: 10-15%, MinWidth: 12
//
// Example:
//
//	Columns: []ColumnConfig{
//	    {Header: "Font Name", MinWidth: 20, PercentWidth: 40.0},
//	    {Header: "Font ID", MinWidth: 15, PercentWidth: 30.0},
//	    {Header: "License", MinWidth: 8, MaxWidth: 15, PercentWidth: 15.0},
//	    {Header: "Source", MinWidth: 12, PercentWidth: 15.0},
//	}
type ColumnConfig struct {
	Header       string  // Column header text
	Width        int     // Fixed width (0 = auto-calculate)
	MinWidth     int     // Minimum column width
	MaxWidth     int     // Maximum column width (0 = no limit)
	PercentWidth float64 // Percentage of available width (0 = not used)
	Wrap         bool    // Enable word wrapping (for future)
	Align        string  // "left", "right", "center" (default: "left")
}

// TableConfig holds table configuration
type TableConfig struct {
	Columns []ColumnConfig
	Rows    [][]string
	Width   int       // Table width (0 = auto-detect terminal width)
	Height  int       // Table height (0 = auto-size to content, for TUI)
	Mode    TableMode // Static or dynamic mode
	Padding int       // Cell padding (0 = no padding, 1 = default, 2 = more padding)
}

// calculateColumnWidths calculates actual column widths based on configuration
// Returns a slice of calculated widths for each column
func calculateColumnWidths(config TableConfig, availableWidth int) []int {
	if len(config.Columns) == 0 {
		return []int{}
	}

	numColumns := len(config.Columns)
	columnWidths := make([]int, numColumns)

	// Reserve space for column separators (1 space between columns)
	separatorSpace := numColumns - 1
	availableWidth -= separatorSpace

	// Ensure we have at least some space
	if availableWidth < numColumns {
		// Very narrow terminal - give each column at least 1 char
		for i := range columnWidths {
			columnWidths[i] = 1
		}
		return columnWidths
	}

	// Step 1: Calculate minimum total width
	totalMinWidth := 0
	for _, col := range config.Columns {
		if col.MinWidth > 0 {
			totalMinWidth += col.MinWidth
		} else {
			// Default minimum of 1 if not specified
			totalMinWidth += 1
		}
	}

	// Step 2: If minimum exceeds available width, use minimums
	if totalMinWidth >= availableWidth {
		for i, col := range config.Columns {
			if col.MinWidth > 0 {
				columnWidths[i] = col.MinWidth
			} else {
				columnWidths[i] = 1
			}
		}
		return columnWidths
	}

	// Step 3: Start with minimum widths
	for i, col := range config.Columns {
		if col.MinWidth > 0 {
			columnWidths[i] = col.MinWidth
		} else {
			columnWidths[i] = 1
		}
	}

	remainingWidth := availableWidth - totalMinWidth

	// Step 4: Apply fixed widths first
	for i, col := range config.Columns {
		if col.Width > 0 {
			// Fixed width takes priority
			oldWidth := columnWidths[i]
			columnWidths[i] = col.Width
			remainingWidth -= (col.Width - oldWidth)
		}
	}

	// Step 5: Distribute remaining width based on percentage widths
	totalPercent := 0.0
	percentColumns := make([]int, 0)
	for i, col := range config.Columns {
		if col.Width == 0 && col.PercentWidth > 0 {
			totalPercent += col.PercentWidth
			percentColumns = append(percentColumns, i)
		}
	}

	if totalPercent > 0 && remainingWidth > 0 {
		// Distribute remaining width proportionally based on percentages
		for _, idx := range percentColumns {
			col := config.Columns[idx]
			proportionalWidth := int(float64(remainingWidth) * col.PercentWidth / totalPercent)
			columnWidths[idx] += proportionalWidth
		}
		// Recalculate remaining width after percentage distribution
		usedWidth := 0
		for _, idx := range percentColumns {
			usedWidth += (columnWidths[idx] - config.Columns[idx].MinWidth)
		}
		remainingWidth -= usedWidth
	}

	// Step 6: Distribute remaining width equally to columns without fixed/percent widths
	equalColumns := make([]int, 0)
	for i, col := range config.Columns {
		if col.Width == 0 && col.PercentWidth == 0 {
			equalColumns = append(equalColumns, i)
		}
	}

	if len(equalColumns) > 0 && remainingWidth > 0 {
		extraPerColumn := remainingWidth / len(equalColumns)
		extraRemainder := remainingWidth % len(equalColumns)
		for i, idx := range equalColumns {
			columnWidths[idx] += extraPerColumn
			if i < extraRemainder {
				columnWidths[idx]++
			}
		}
	}

	// Step 7: Apply MaxWidth constraints
	for i, col := range config.Columns {
		if col.MaxWidth > 0 && columnWidths[i] > col.MaxWidth {
			excess := columnWidths[i] - col.MaxWidth
			columnWidths[i] = col.MaxWidth
			// Redistribute excess to other columns that haven't hit their max
			for j := range columnWidths {
				if j != i && (config.Columns[j].MaxWidth == 0 || columnWidths[j] < config.Columns[j].MaxWidth) {
					columnWidths[j] += excess / (len(columnWidths) - 1)
					excess = excess % (len(columnWidths) - 1)
					if excess == 0 {
						break
					}
				}
			}
		}
	}

	// Step 8: Ensure total doesn't exceed available width
	totalWidth := 0
	for _, w := range columnWidths {
		totalWidth += w
	}
	if totalWidth > availableWidth {
		// Scale down proportionally
		scale := float64(availableWidth) / float64(totalWidth)
		for i := range columnWidths {
			columnWidths[i] = int(float64(columnWidths[i]) * scale)
			if columnWidths[i] < 1 {
				columnWidths[i] = 1
			}
		}
	}

	return columnWidths
}

// truncateString truncates a string to the specified width with ellipsis
func truncateString(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if width <= 3 {
		return strings.Repeat(".", width)
	}
	if len(s) <= width {
		return s
	}
	return s[:width-3] + "..."
}

// formatCell formats a cell value according to alignment and width
// Returns the formatted cell and its actual visual width (accounting for ANSI codes)
func formatCell(value string, width int, align string) (string, int) {
	truncated := truncateString(value, width)
	var formatted string
	switch align {
	case "right":
		formatted = fmt.Sprintf("%*s", width, truncated)
	case "center":
		padding := width - len(truncated)
		leftPad := padding / 2
		rightPad := padding - leftPad
		formatted = strings.Repeat(" ", leftPad) + truncated + strings.Repeat(" ", rightPad)
	default: // "left"
		formatted = fmt.Sprintf("%-*s", width, truncated)
	}
	// Get visual width (strips ANSI codes)
	visualWidth := lipgloss.Width(formatted)
	return formatted, visualWidth
}

// RenderStaticTable renders a table as a string for CLI output
func RenderStaticTable(config TableConfig) string {
	if len(config.Columns) == 0 {
		return ""
	}

	// Determine table width
	tableWidth := config.Width
	if tableWidth == 0 {
		tableWidth = shared.GetTerminalWidth()
	}

	// Calculate column widths
	columnWidths := calculateColumnWidths(config, tableWidth)

	var output strings.Builder

	// Build header row
	headerCells := make([]string, len(config.Columns))
	for i, col := range config.Columns {
		align := col.Align
		if align == "" {
			align = "left"
		}
		headerText, _ := formatCell(col.Header, columnWidths[i], align)
		// Apply style with width constraint to ensure proper formatting
		headerStyle := ui.TableHeader.Copy().Width(columnWidths[i]).MaxWidth(columnWidths[i])
		styledHeader := headerStyle.Render(headerText)
		headerCells[i] = styledHeader
	}
	headerRow := strings.Join(headerCells, " ")
	output.WriteString(headerRow)
	output.WriteString("\n")

	// Build separator row
	separatorCells := make([]string, len(config.Columns))
	for i, width := range columnWidths {
		separatorCells[i] = strings.Repeat("─", width)
	}
	separatorRow := strings.Join(separatorCells, " ")
	output.WriteString(separatorRow)
	output.WriteString("\n")

	// Build data rows
	for _, row := range config.Rows {
		rowCells := make([]string, len(config.Columns))
		for j := range config.Columns {
			align := config.Columns[j].Align
			if align == "" {
				align = "left"
			}
			var cellValue string
			if j < len(row) {
				cellValue = row[j]
			}
			formattedCell, _ := formatCell(cellValue, columnWidths[j], align)

			// Apply style with width constraint to ensure proper formatting
			var styledCell string
			if j == 0 {
				// First column uses TableSourceName style
				cellStyle := ui.TableSourceName.Copy().Width(columnWidths[j]).MaxWidth(columnWidths[j])
				styledCell = cellStyle.Render(formattedCell)
			} else {
				// Other columns use Text style
				cellStyle := ui.Text.Copy().Width(columnWidths[j]).MaxWidth(columnWidths[j])
				styledCell = cellStyle.Render(formattedCell)
			}
			rowCells[j] = styledCell
		}
		rowText := strings.Join(rowCells, " ")
		output.WriteString(rowText)
		output.WriteString("\n")
	}

	return strings.TrimRight(output.String(), "\n")
}

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
