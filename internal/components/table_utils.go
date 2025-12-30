package components

import (
	"fmt"
	"strings"

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
