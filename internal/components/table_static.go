package components

import (
	"strings"

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
