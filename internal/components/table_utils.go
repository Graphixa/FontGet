package components

import (
	"sort"
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
	Priority     int     // Priority for hiding/truncation (higher = hide/truncate last, 0 = default/lowest priority)
	Hideable     bool    // Whether column can be hidden (default: true)
	Truncatable  bool    // Whether content can be truncated (default: true). If false, column sizes to fit content
	Wrap         bool    // Enable word wrapping (for future)
	Align        string  // "left", "right", "center" (default: "left")
}

const (
	// DefaultMaxTableWidth is the default maximum width for tables
	// This prevents tables from becoming too wide on ultrawide screens
	// Set to 0 to disable maximum width constraint
	DefaultMaxTableWidth = 120

	// Minimum column width before hiding (in characters)
	minColumnWidthBeforeHide = 4

	// Priority weight calculation constants
	priorityWeightStep     = 0.1 // Weight decreases by this amount per priority level
	minPriorityWeight      = 0.1 // Minimum weight to avoid zero/negative weights
	maxReductionAdjustment = 0.5 // Maximum 50% adjustment for priority-based reduction

	// Distribution iteration safety limit
	maxDistributionIterations = 100
)

// TableConfig holds table configuration
type TableConfig struct {
	Columns  []ColumnConfig
	Rows     [][]string
	Width    int       // Table width (0 = auto-detect terminal width)
	MaxWidth int       // Maximum table width (0 = use DefaultMaxTableWidth, -1 = no limit)
	Height   int       // Table height (0 = auto-size to content, for TUI)
	Mode     TableMode // Static or dynamic mode
	Padding  int       // Cell padding (0 = no padding, 1 = default, 2 = more padding)
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

	// Step 0: Calculate percentage-based target widths for ALL columns first
	// This ensures fair distribution based on PercentWidth
	totalPercent := 0.0
	percentColumns := make([]int, 0)
	for i, col := range config.Columns {
		if col.Width == 0 && col.PercentWidth > 0 {
			totalPercent += col.PercentWidth
			percentColumns = append(percentColumns, i)
		}
	}

	// Calculate target widths based on percentages of TOTAL available width
	targetWidths := make([]int, numColumns)
	for i, col := range config.Columns {
		if col.Width > 0 {
			// Fixed width takes priority
			targetWidths[i] = col.Width
		} else if col.PercentWidth > 0 && totalPercent > 0 {
			// Calculate target based on percentage of total available width
			targetWidths[i] = int(float64(availableWidth) * col.PercentWidth / totalPercent)
			// Ensure target is at least MinWidth
			if col.MinWidth > 0 && targetWidths[i] < col.MinWidth {
				targetWidths[i] = col.MinWidth
			}
			// For truncatable columns, respect MaxWidth
			// For non-truncatable columns, MaxWidth is ignored (content takes priority)
			if col.Truncatable && col.MaxWidth > 0 && targetWidths[i] > col.MaxWidth {
				targetWidths[i] = col.MaxWidth
			}
		} else {
			// No percentage or fixed width - use MinWidth or default
			if col.MinWidth > 0 {
				targetWidths[i] = col.MinWidth
			} else {
				targetWidths[i] = 1
			}
		}
	}

	// Step 0.5: For non-truncatable columns, check if content is wider than target
	// If so, use content width; otherwise use target width (to respect percentage distribution)
	nonTruncatableWidth := 0
	for i, col := range config.Columns {
		if !col.Truncatable {
			contentWidth := calculateColumnContentWidth(config, i)
			// Use the wider of: content width, target width (from percentage), or MinWidth
			finalWidth := targetWidths[i] // Start with percentage-based target
			if contentWidth > finalWidth {
				finalWidth = contentWidth // Content is wider, use content width
			}
			if col.MinWidth > 0 && col.MinWidth > finalWidth {
				finalWidth = col.MinWidth // MinWidth takes priority
			}
			// Don't respect MaxWidth for non-truncatable - content takes priority
			columnWidths[i] = finalWidth
			nonTruncatableWidth += finalWidth
		}
	}

	// Reserve space for non-truncatable columns
	// Separators are already accounted for in the initial availableWidth calculation
	availableForTruncatable := availableWidth - nonTruncatableWidth
	if availableForTruncatable < 0 {
		availableForTruncatable = 0
	}

	// Step 1: Calculate minimum total width for truncatable columns only
	totalMinWidth := 0
	truncatableColumns := make([]int, 0)
	for i, col := range config.Columns {
		if !col.Truncatable {
			continue // Skip non-truncatable columns
		}
		truncatableColumns = append(truncatableColumns, i)
		if col.MinWidth > 0 {
			totalMinWidth += col.MinWidth
		} else {
			totalMinWidth += 1
		}
	}

	// Step 2: If minimum exceeds available width for truncatable columns, use minimums
	if totalMinWidth >= availableForTruncatable {
		for _, i := range truncatableColumns {
			col := config.Columns[i]
			if col.MinWidth > 0 {
				columnWidths[i] = col.MinWidth
			} else {
				columnWidths[i] = 1
			}
		}
		return columnWidths
	}

	// Step 3: Apply target widths to truncatable columns
	// Non-truncatable columns already have their widths set in Step 0.5
	// (they use the larger of percentage target or content width)
	// For truncatable columns, use the percentage-based target calculated in Step 0
	// But ensure MinWidth is respected
	for _, i := range truncatableColumns {
		col := config.Columns[i]
		if col.Width > 0 {
			// Fixed width takes priority
			columnWidths[i] = col.Width
		} else {
			// Use the target width calculated in Step 0 (based on percentage of total width)
			columnWidths[i] = targetWidths[i]
		}
		// Ensure MinWidth is respected for truncatable columns too
		if col.MinWidth > 0 && columnWidths[i] < col.MinWidth {
			columnWidths[i] = col.MinWidth
		}
	}

	// Step 4: Calculate remaining width after all columns have initial widths
	// This includes both truncatable and non-truncatable columns
	totalWidth := 0
	for _, w := range columnWidths {
		totalWidth += w
	}
	remainingWidth := availableWidth - totalWidth

	// Step 5: If total width exceeds available width, scale down truncatable columns proportionally
	// Lower priority truncatable columns are truncated more aggressively (get less width)
	// Higher priority columns are scaled down less (preserved more)
	// Non-truncatable columns are already at their minimum (content width), so we only scale truncatable
	if remainingWidth < 0 {
		// Identify truncation candidates: all truncatable columns
		// Lower priority columns will be truncated more (receive less width)
		type truncateCandidate struct {
			index    int
			priority int
		}
		truncateCandidates := make([]truncateCandidate, 0, len(truncatableColumns))
		for _, i := range truncatableColumns {
			truncateCandidates = append(truncateCandidates, truncateCandidate{
				index:    i,
				priority: config.Columns[i].Priority,
			})
		}

		// Sort by priority (higher priority number = truncate more aggressively first)
		// Priority 5 truncates before Priority 4, which truncates before Priority 3, etc.
		sort.SliceStable(truncateCandidates, func(i, j int) bool {
			return truncateCandidates[i].priority > truncateCandidates[j].priority
		})

		// Calculate total width of only truncatable columns
		truncatableCurrentWidth := 0
		for _, i := range truncatableColumns {
			truncatableCurrentWidth += columnWidths[i]
		}

		// Calculate how much we need to reduce truncatable columns by
		reductionNeeded := -remainingWidth

		if truncatableCurrentWidth > 0 {
			// Calculate priority weights: higher priority number = lower weight (more reduction/truncation)
			// Priority 1 = weight 1.0, Priority 2 = weight 0.9, Priority 3 = weight 0.8, etc.
			// Lower priority numbers (higher importance) get more weight (less truncation)
			totalPriorityWeight := 0.0
			priorityWeights := make(map[int]float64)
			for _, candidate := range truncateCandidates {
				col := config.Columns[candidate.index]
				// Lower priority number (higher importance) = higher weight (less truncation)
				// Higher priority number (lower importance) = lower weight (more truncation)
				// Invert: weight decreases as priority number increases
				weight := calculatePriorityWeight(col.Priority)
				priorityWeights[candidate.index] = weight
				totalPriorityWeight += weight
			}

			// Calculate base scale (what we'd use without priority)
			baseScale := float64(truncatableCurrentWidth-reductionNeeded) / float64(truncatableCurrentWidth)

			// Distribute reduction based on priority: lower priority columns get more reduction (more truncation)
			for _, candidate := range truncateCandidates {
				i := candidate.index
				col := config.Columns[i]
				weight := priorityWeights[i]
				// Higher priority = less reduction (closer to baseScale) = less truncation
				// Lower priority = more reduction (further from baseScale) = more truncation
				// Calculate reduction factor: lower priority columns contribute more to reduction
				reductionFactor := 1.0 - (weight/totalPriorityWeight)*maxReductionAdjustment
				priorityScale := baseScale - (1.0-baseScale)*reductionFactor

				newWidth := int(float64(columnWidths[i]) * priorityScale)
				if col.MinWidth > 0 && newWidth < col.MinWidth {
					newWidth = col.MinWidth
				}
				columnWidths[i] = newWidth
			}
		}
		// Recalculate remaining width
		totalWidth = 0
		for _, w := range columnWidths {
			totalWidth += w
		}
		remainingWidth = availableWidth - totalWidth
	}

	// Step 6: If we have remaining width, distribute it proportionally to expandable columns
	// Prioritize columns that are currently truncated (content > width)
	if remainingWidth > 0 {
		// First, identify columns that are truncated (content width > column width)
		truncatedColumns := make([]int, 0)
		for i, col := range config.Columns {
			if col.Truncatable {
				contentWidth := calculateColumnContentWidth(config, i)
				// If content is wider than current width, column is truncated
				if contentWidth > columnWidths[i] {
					truncatedColumns = append(truncatedColumns, i)
				}
			}
		}

		// Sort truncated columns by priority (lower priority number = higher importance = expand first)
		sort.SliceStable(truncatedColumns, func(i, j int) bool {
			return config.Columns[truncatedColumns[i]].Priority < config.Columns[truncatedColumns[j]].Priority
		})

		// First pass: Give width to truncated columns to reduce/eliminate truncation
		// Prioritize higher priority (lower number) truncated columns
		for _, idx := range truncatedColumns {
			if remainingWidth <= 0 {
				break
			}
			col := config.Columns[idx]

			// Calculate how much width this column needs to show all content
			contentWidth := calculateColumnContentWidth(config, idx)
			neededWidth := contentWidth - columnWidths[idx]
			if neededWidth > 0 {
				// Respect MaxWidth if set
				if col.MaxWidth > 0 {
					maxAdditional := col.MaxWidth - columnWidths[idx]
					if neededWidth > maxAdditional {
						neededWidth = maxAdditional
					}
				}

				// Give as much as needed, but don't exceed remaining width
				toGive := neededWidth
				if toGive > remainingWidth {
					toGive = remainingWidth
				}

				if toGive > 0 {
					columnWidths[idx] += toGive
					remainingWidth -= toGive
				}
			}
		}

		// Second pass: Distribute any remaining width proportionally to expandable columns
		distributed := 0
		iterations := 0

		for remainingWidth > 0 && iterations < maxDistributionIterations {
			iterations++

			// Recalculate which columns can still receive width
			activePercentColumns := make([]int, 0)
			activeTotalPercent := 0.0
			for _, idx := range percentColumns {
				col := config.Columns[idx]
				// Column can receive width if it has no MaxWidth or hasn't reached it
				if col.MaxWidth == 0 || columnWidths[idx] < col.MaxWidth {
					activePercentColumns = append(activePercentColumns, idx)
					activeTotalPercent += col.PercentWidth
				}
			}

			if len(activePercentColumns) == 0 {
				break // No columns can receive more width
			}

			// Distribute proportionally to active columns, weighted by priority
			// Lower priority numbers (higher importance) get more space
			iterationDistributed := 0

			// Calculate priority weights for distribution (inverted: lower number = higher weight)
			totalPriorityWeight := 0.0
			priorityWeights := make(map[int]float64)
			for _, idx := range activePercentColumns {
				col := config.Columns[idx]
				// Lower priority number = higher weight (more space)
				// Priority 1 = weight 1.0, Priority 2 = weight 0.9, etc.
				weight := calculatePriorityWeight(col.Priority)
				priorityWeights[idx] = weight
				// Weight by both priority and percentage
				totalPriorityWeight += weight * col.PercentWidth
			}

			for _, idx := range activePercentColumns {
				col := config.Columns[idx]
				priorityWeight := priorityWeights[idx]
				// Calculate proportional width: base percentage * priority weight
				proportionalWidth := int(float64(remainingWidth) * (priorityWeight * col.PercentWidth) / totalPriorityWeight)

				// Respect MaxWidth constraint
				if col.MaxWidth > 0 && columnWidths[idx]+proportionalWidth > col.MaxWidth {
					proportionalWidth = col.MaxWidth - columnWidths[idx]
				}

				if proportionalWidth > 0 {
					columnWidths[idx] += proportionalWidth
					iterationDistributed += proportionalWidth
				}
			}

			distributed += iterationDistributed
			remainingWidth -= iterationDistributed

			// If we didn't distribute anything this iteration, break
			if iterationDistributed == 0 {
				break
			}
		}

		// Handle any remaining width from rounding - distribute to higher priority columns first
		if remainingWidth > 0 {
			expandableColumns := make([]int, 0)
			for _, idx := range percentColumns {
				col := config.Columns[idx]
				if col.MaxWidth == 0 || columnWidths[idx] < col.MaxWidth {
					expandableColumns = append(expandableColumns, idx)
				}
			}
			if len(expandableColumns) > 0 {
				// Sort by priority (higher priority first)
				sort.SliceStable(expandableColumns, func(i, j int) bool {
					return config.Columns[expandableColumns[i]].Priority > config.Columns[expandableColumns[j]].Priority
				})
				for _, idx := range expandableColumns {
					if remainingWidth > 0 {
						col := config.Columns[idx]
						// Respect MaxWidth
						if col.MaxWidth == 0 || columnWidths[idx] < col.MaxWidth {
							columnWidths[idx]++
							remainingWidth--
						}
					} else {
						break
					}
				}
			}
		}
	}

	// Step 6.5: Distribute remaining width equally to columns without fixed/percent widths
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

	// Step 7: Apply MaxWidth constraints and redistribute excess proportionally
	for i, col := range config.Columns {
		if col.MaxWidth > 0 && columnWidths[i] > col.MaxWidth {
			excess := columnWidths[i] - col.MaxWidth
			columnWidths[i] = col.MaxWidth

			// Find columns that can receive excess (haven't hit their max and have percentages)
			redistributeColumns := make([]int, 0)
			totalRedistPercent := 0.0
			for j := range columnWidths {
				if j != i && (config.Columns[j].MaxWidth == 0 || columnWidths[j] < config.Columns[j].MaxWidth) {
					if config.Columns[j].PercentWidth > 0 {
						redistributeColumns = append(redistributeColumns, j)
						totalRedistPercent += config.Columns[j].PercentWidth
					}
				}
			}

			// Redistribute proportionally based on percentages
			if totalRedistPercent > 0 && len(redistributeColumns) > 0 {
				for _, idx := range redistributeColumns {
					proportionalExcess := int(float64(excess) * config.Columns[idx].PercentWidth / totalRedistPercent)
					columnWidths[idx] += proportionalExcess
					excess -= proportionalExcess
				}
				// Distribute any remaining excess equally (rounding errors)
				if excess > 0 && len(redistributeColumns) > 0 {
					perColumn := excess / len(redistributeColumns)
					remainder := excess % len(redistributeColumns)
					for k, idx := range redistributeColumns {
						columnWidths[idx] += perColumn
						if k < remainder {
							columnWidths[idx]++
						}
					}
				}
			} else if len(redistributeColumns) > 0 {
				// No percentages available, distribute equally as fallback
				perColumn := excess / len(redistributeColumns)
				remainder := excess % len(redistributeColumns)
				for k, idx := range redistributeColumns {
					columnWidths[idx] += perColumn
					if k < remainder {
						columnWidths[idx]++
					}
				}
			}
		}
	}

	// Step 8: Ensure total doesn't exceed available width
	totalWidth = 0
	for _, w := range columnWidths {
		totalWidth += w
	}
	if totalWidth > availableWidth {
		// Scale down proportionally, but protect non-truncatable columns
		// Calculate width of non-truncatable columns
		nonTruncatableWidth := 0
		truncatableWidth := 0
		for i, w := range columnWidths {
			if !config.Columns[i].Truncatable {
				nonTruncatableWidth += w
			} else {
				truncatableWidth += w
			}
		}

		// If non-truncatable columns alone exceed available width, we have a problem
		// But we must respect them - scale down truncatable columns only
		if nonTruncatableWidth >= availableWidth {
			// Can't fit - non-truncatable columns take priority
			// Scale down truncatable columns to minimum
			for i, col := range config.Columns {
				if !col.Truncatable {
					continue // Keep non-truncatable width
				}
				if col.MinWidth > 0 {
					columnWidths[i] = col.MinWidth
				} else {
					columnWidths[i] = 1
				}
			}
		} else {
			// Scale down only truncatable columns
			availableForTruncatable := availableWidth - nonTruncatableWidth
			if truncatableWidth > 0 && availableForTruncatable > 0 {
				scale := float64(availableForTruncatable) / float64(truncatableWidth)
				for i := range columnWidths {
					if !config.Columns[i].Truncatable {
						continue // Don't scale non-truncatable
					}
					columnWidths[i] = int(float64(columnWidths[i]) * scale)
					if columnWidths[i] < 1 {
						columnWidths[i] = 1
					}
					// Respect MinWidth
					if config.Columns[i].MinWidth > 0 && columnWidths[i] < config.Columns[i].MinWidth {
						columnWidths[i] = config.Columns[i].MinWidth
					}
				}
			}
		}
		// Recalculate total after scaling
		totalWidth = 0
		for _, w := range columnWidths {
			totalWidth += w
		}
	}

	// Step 9: Distribute any remaining width to columns without MaxWidth constraints
	// This ensures we fully utilize the terminal width when some columns have MaxWidth limits
	remainingWidth = availableWidth - totalWidth
	for remainingWidth > 0 {
		// Find columns that can receive additional width (no MaxWidth or haven't hit it)
		expandableColumns := make([]int, 0)
		totalExpandPercent := 0.0
		for i, col := range config.Columns {
			// Column is expandable if it has no MaxWidth or hasn't reached it
			if col.MaxWidth == 0 || columnWidths[i] < col.MaxWidth {
				expandableColumns = append(expandableColumns, i)
				if col.PercentWidth > 0 {
					totalExpandPercent += col.PercentWidth
				}
			}
		}

		// If no columns can expand, we're done
		if len(expandableColumns) == 0 {
			break
		}

		// Distribute remaining width
		if totalExpandPercent > 0 {
			// Distribute proportionally based on PercentWidth
			distributed := 0
			for _, idx := range expandableColumns {
				col := config.Columns[idx]
				if col.PercentWidth > 0 {
					proportionalWidth := int(float64(remainingWidth) * col.PercentWidth / totalExpandPercent)
					// Don't exceed MaxWidth if one is set
					if col.MaxWidth > 0 && columnWidths[idx]+proportionalWidth > col.MaxWidth {
						proportionalWidth = col.MaxWidth - columnWidths[idx]
					}
					if proportionalWidth > 0 {
						columnWidths[idx] += proportionalWidth
						distributed += proportionalWidth
					}
				}
			}
			remainingWidth -= distributed

			// If still have remaining width, distribute equally to all expandable columns
			if remainingWidth > 0 {
				// Re-check which columns can still expand
				stillExpandable := make([]int, 0)
				for _, idx := range expandableColumns {
					col := config.Columns[idx]
					if col.MaxWidth == 0 || columnWidths[idx] < col.MaxWidth {
						stillExpandable = append(stillExpandable, idx)
					}
				}
				if len(stillExpandable) > 0 {
					extraPerColumn := remainingWidth / len(stillExpandable)
					extraRemainder := remainingWidth % len(stillExpandable)
					distributed := 0
					for i, idx := range stillExpandable {
						col := config.Columns[idx]
						extra := extraPerColumn
						if i < extraRemainder {
							extra++
						}
						// Respect MaxWidth if set
						if col.MaxWidth > 0 && columnWidths[idx]+extra > col.MaxWidth {
							extra = col.MaxWidth - columnWidths[idx]
						}
						if extra > 0 {
							columnWidths[idx] += extra
							distributed += extra
						}
					}
					remainingWidth -= distributed
				}
			}
		} else {
			// No PercentWidth specified, distribute equally to expandable columns
			extraPerColumn := remainingWidth / len(expandableColumns)
			extraRemainder := remainingWidth % len(expandableColumns)
			distributed := 0
			for i, idx := range expandableColumns {
				col := config.Columns[idx]
				extra := extraPerColumn
				if i < extraRemainder {
					extra++
				}
				// Respect MaxWidth if set
				if col.MaxWidth > 0 && columnWidths[idx]+extra > col.MaxWidth {
					extra = col.MaxWidth - columnWidths[idx]
				}
				if extra > 0 {
					columnWidths[idx] += extra
					distributed += extra
				}
			}
			remainingWidth -= distributed
		}

		// If we didn't distribute anything, break to avoid infinite loop
		if remainingWidth == availableWidth-totalWidth {
			break
		}
	}

	return columnWidths
}

// calculateColumnContentWidth calculates the widest content in a column (header + all cells)
// Uses visual width (strips ANSI codes) for accurate measurement
func calculateColumnContentWidth(config TableConfig, columnIndex int) int {
	if columnIndex >= len(config.Columns) {
		return 0
	}
	contentWidth := lipgloss.Width(config.Columns[columnIndex].Header)
	for _, row := range config.Rows {
		if columnIndex < len(row) {
			cellWidth := lipgloss.Width(row[columnIndex])
			if cellWidth > contentWidth {
				contentWidth = cellWidth
			}
		}
	}
	return contentWidth
}

// calculatePriorityWeight calculates the weight for a given priority level
// Lower priority numbers (higher importance) get higher weights
// Priority 1 = weight 1.0, Priority 2 = weight 0.9, Priority 3 = weight 0.8, etc.
func calculatePriorityWeight(priority int) float64 {
	weight := 1.0 - float64(priority-1)*priorityWeightStep
	if weight < minPriorityWeight {
		weight = minPriorityWeight
	}
	return weight
}

// truncateString truncates a string to the specified width with ellipsis
// Uses visual width (strips ANSI codes) for accurate truncation
func truncateString(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if width <= 3 {
		return strings.Repeat(".", width)
	}
	// Use visual width (strips ANSI codes) for comparison
	visualWidth := lipgloss.Width(s)
	if visualWidth <= width {
		return s
	}
	// Use lipgloss to truncate, which handles ANSI codes correctly
	// Lipgloss will preserve ANSI codes and truncate at the visual width
	truncatedWidth := width - 3 // Reserve space for "..."
	truncated := lipgloss.NewStyle().Width(truncatedWidth).MaxWidth(truncatedWidth).Render(s)
	// Check if lipgloss actually truncated it
	if lipgloss.Width(truncated) < visualWidth {
		// Lipgloss truncated it, add our ellipsis
		// We need to append "..." but preserve any trailing ANSI reset codes
		return truncated + "..."
	}
	// Fallback: if lipgloss didn't truncate (shouldn't happen), use simple approach
	// This is a fallback for edge cases
	return s[:width-3] + "..."
}

// formatCell formats a cell value according to alignment and width
// truncatable: if false, content is never truncated (width is ignored for truncation)
//
//	if true (default), content is truncated if it exceeds width
//
// Returns the formatted cell and its actual visual width (accounting for ANSI codes)
func formatCell(value string, width int, align string, truncatable bool) (string, int) {
	// Get visual width (strips ANSI codes) for accurate measurement
	visualValueWidth := lipgloss.Width(value)

	var truncated string
	// IMPORTANT: truncatable parameter:
	// - true: truncate content if it exceeds width
	// - false: never truncate, use content as-is (for non-truncatable columns)
	// Default behavior should be to truncate, but since zero value is false,
	// we need to handle this at the call site
	if truncatable {
		// Truncatable: truncate if needed (use visual width for comparison)
		if visualValueWidth > width {
			truncated = truncateString(value, width)
		} else {
			truncated = value
		}
	} else {
		// Non-truncatable: use content as-is, but pad to width
		truncated = value
	}

	// Get visual width of truncated value for padding calculation
	visualTruncatedWidth := lipgloss.Width(truncated)

	var formatted string
	switch align {
	case "right":
		// Use visual width for padding calculation
		padding := width - visualTruncatedWidth
		if padding < 0 {
			padding = 0
		}
		formatted = strings.Repeat(" ", padding) + truncated
	case "center":
		padding := width - visualTruncatedWidth
		if padding < 0 {
			padding = 0
		}
		leftPad := padding / 2
		rightPad := padding - leftPad
		formatted = strings.Repeat(" ", leftPad) + truncated + strings.Repeat(" ", rightPad)
	default: // "left"
		// Use visual width for padding calculation
		padding := width - visualTruncatedWidth
		if padding < 0 {
			padding = 0
		}
		formatted = truncated + strings.Repeat(" ", padding)
	}
	// Get visual width (strips ANSI codes)
	visualWidth := lipgloss.Width(formatted)
	return formatted, visualWidth
}
