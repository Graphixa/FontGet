package components

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"fontget/internal/shared"
	"fontget/internal/ui"
)

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

	// Apply maximum width constraint
	maxWidth := config.MaxWidth
	if maxWidth == 0 {
		maxWidth = DefaultMaxTableWidth // Use default if not specified
	}
	if maxWidth > 0 && tableWidth > maxWidth {
		tableWidth = maxWidth
	}

	// Start with all columns visible
	visibleColumns := make([]int, len(config.Columns))
	for i := range config.Columns {
		visibleColumns[i] = i
	}

	// Type for tracking columns that are candidates for hiding
	type hideCandidate struct {
		index    int
		priority int
	}

	// Helper function to create a visible config from visible column indices
	createVisibleConfig := func(visibleCols []int) TableConfig {
		visibleConfig := TableConfig{
			Columns:  make([]ColumnConfig, len(visibleCols)),
			Rows:     make([][]string, len(config.Rows)),
			Width:    config.Width,
			MaxWidth: config.MaxWidth,
			Mode:     config.Mode,
			Padding:  config.Padding,
		}
		// Copy visible column configs
		for i, colIdx := range visibleCols {
			visibleConfig.Columns[i] = config.Columns[colIdx]
		}
		// Copy rows
		for rowIdx, row := range config.Rows {
			visibleConfig.Rows[rowIdx] = make([]string, len(visibleCols))
			for i, colIdx := range visibleCols {
				if colIdx < len(row) {
					visibleConfig.Rows[rowIdx][i] = row[colIdx]
				}
			}
		}
		return visibleConfig
	}

	// Iteratively hide columns that don't fit, starting with lowest priority
	var columnWidths []int
	for {
		if len(visibleColumns) == 0 {
			break // No columns left
		}

		// Create config with current visible columns
		visibleConfig := createVisibleConfig(visibleColumns)

		// Recalculate widths for current visible columns
		visibleWidths := calculateColumnWidths(visibleConfig, tableWidth)

		// Calculate total width used (including separators)
		totalWidth := 0
		for _, w := range visibleWidths {
			totalWidth += w
		}
		if len(visibleColumns) > 1 {
			totalWidth += len(visibleColumns) - 1 // Separators
		}

		// Check which columns don't fit (header truncated or too narrow)
		columnsThatDontFit := make([]hideCandidate, 0)
		for i, colIdx := range visibleColumns {
			col := config.Columns[colIdx]
			width := visibleWidths[i]

			// Column doesn't fit if:
			// 1. Too narrow (below minimum width), OR
			// 2. Header would be truncated
			// AND it's hideable
			if col.Hideable && (width < minColumnWidthBeforeHide || len(col.Header) > width) {
				columnsThatDontFit = append(columnsThatDontFit, hideCandidate{
					index:    colIdx,
					priority: col.Priority,
				})
			}
		}

		// If we have columns that don't fit OR total width exceeds available space, hide a column
		if len(columnsThatDontFit) > 0 || totalWidth > tableWidth {
			// Collect all hideable columns, sorted by priority (lower = hide first)
			hideableColumns := make([]hideCandidate, 0)
			for _, colIdx := range visibleColumns {
				col := config.Columns[colIdx]
				if col.Hideable {
					hideableColumns = append(hideableColumns, hideCandidate{
						index:    colIdx,
						priority: col.Priority,
					})
				}
			}

			// If we have hideable columns, hide the lowest priority one
			if len(hideableColumns) > 0 {
				// Sort by priority (higher priority number = hide first, lower priority number = hide last)
				// Priority 5 hides before Priority 4, which hides before Priority 3, etc.
				sort.SliceStable(hideableColumns, func(i, j int) bool {
					return hideableColumns[i].priority > hideableColumns[j].priority
				})

				// Hide the highest priority number (lowest priority) column first
				columnToHide := hideableColumns[0].index
				newVisible := make([]int, 0, len(visibleColumns)-1)
				for _, idx := range visibleColumns {
					if idx != columnToHide {
						newVisible = append(newVisible, idx)
					}
				}
				visibleColumns = newVisible
				continue // Recalculate with one less column
			}
		}

		// All columns fit and we're within available width - we're done
		// Map widths back to original column indices
		columnWidths = make([]int, len(config.Columns))
		for i, colIdx := range visibleColumns {
			columnWidths[colIdx] = visibleWidths[i]
		}
		break
	}

	// If no columns are visible, return empty
	if len(visibleColumns) == 0 {
		return ""
	}

	// Get final widths (recalculate one more time to ensure accuracy)
	visibleConfig := createVisibleConfig(visibleColumns)
	finalWidths := calculateColumnWidths(visibleConfig, tableWidth)
	columnWidths = make([]int, len(config.Columns))
	for i, colIdx := range visibleColumns {
		columnWidths[colIdx] = finalWidths[i]
	}

	var output strings.Builder

	// Build header row (only visible columns)
	headerCells := make([]string, 0, len(visibleColumns))
	for _, i := range visibleColumns {
		col := config.Columns[i]
		align := col.Align
		if align == "" {
			align = "left"
		}
		headerText, _ := formatCell(col.Header, columnWidths[i], align, col.Truncatable)
		// Apply style with width constraint to ensure proper formatting
		headerStyle := ui.TableHeader.Copy().Width(columnWidths[i]).MaxWidth(columnWidths[i])
		styledHeader := headerStyle.Render(headerText)
		headerCells = append(headerCells, styledHeader)
	}
	headerRow := strings.Join(headerCells, " ")
	output.WriteString(headerRow)
	output.WriteString("\n")

	// Build separator row (only visible columns)
	separatorCells := make([]string, 0, len(visibleColumns))
	for _, i := range visibleColumns {
		separatorCells = append(separatorCells, strings.Repeat("─", columnWidths[i]))
	}
	separatorRow := strings.Join(separatorCells, " ")
	output.WriteString(separatorRow)
	output.WriteString("\n")

	// Build data rows (only visible columns)
	for _, row := range config.Rows {
		rowCells := make([]string, 0, len(visibleColumns))
		for _, j := range visibleColumns {
			align := config.Columns[j].Align
			if align == "" {
				align = "left"
			}
			var cellValue string
			if j < len(row) {
				cellValue = row[j]
			}
			formattedCell, _ := formatCell(cellValue, columnWidths[j], align, config.Columns[j].Truncatable)
			styledCell := styleCell(formattedCell, columnWidths[j], j == 0)
			rowCells = append(rowCells, styledCell)
		}
		rowText := strings.Join(rowCells, " ")
		output.WriteString(rowText)
		output.WriteString("\n")
	}

	return strings.TrimRight(output.String(), "\n")
}

// styleCell applies styling to a cell based on its position
// First column uses TableSourceName style, others use Text style
func styleCell(formattedCell string, width int, isFirstColumn bool) string {
	var cellStyle lipgloss.Style
	if isFirstColumn {
		cellStyle = ui.TableSourceName.Copy().Width(width).MaxWidth(width)
	} else {
		cellStyle = ui.Text.Copy().Width(width).MaxWidth(width)
	}
	return cellStyle.Render(formattedCell)
}
