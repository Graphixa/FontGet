package ui

import (
	"fmt"
	"strings"
)

// Table formatting constants for consistent table widths across all commands
const (
	// Font Search/Add/Remove Tables (5 columns, total: 120 chars - uses full 120-char terminals)
	TableColName       = 36 // Font name (wider for longer names)
	TableColID         = 34 // Font ID (wider for longer IDs like "nerd.font-name")
	TableColLicense    = 12 // License (slightly wider)
	TableColCategories = 16 // Categories (wider for multiple categories)
	TableColSource     = 18 // Source (wider for source names)

	// Font List Tables (7 columns, total: 120 chars)
	TableColListName     = 30 // Font family name
	TableColListID       = 28 // Font ID
	TableColListLicense  = 8  // License
	TableColListCategory = 16 // Categories
	TableColType         = 8  // File type
	TableColScope        = 8  // Scope (user/machine)
	TableColListSource   = 16 // Source
	// Total: 30 + 28 + 8 + 16 + 8 + 8 + 16 + 6 spaces = 120 chars (exactly 120)

	// Sources Management Tables (2 columns, total: 120 chars)
	TableColStatus     = 10  // Checkbox/status
	TableColSourceName = 109 // Source name with tags (much wider)

	// Sources Info Table (4 columns, total: 120 chars including 3 spaces)
	// Sum of widths must equal 117 (117 + 3 spaces = 120)
	TableColSrcName    = 36 // Source display name (room for optional [Disabled])
	TableColSrcPrefix  = 12 // Prefix key
	TableColSrcUpdated = 32 // Last updated
	TableColSrcType    = 10 // Type (Built-in/Custom)

	// Total table width (uses full 120-char terminals for maximum space utilization)
	TableTotalWidth = 120
)

// GetSearchTableHeader returns a styled table header for font search/add/remove tables
// Returns a styled string ready to print (includes bold formatting)
func GetSearchTableHeader() string {
	header := fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s",
		TableColName, "Name",
		TableColID, "ID",
		TableColLicense, "License",
		TableColCategories, "Categories",
		TableColSource, "Source")
	return TableHeader.Render(header)
}

// GetListTableHeader returns a styled table header for font list tables
// Returns a styled string ready to print (includes bold formatting)
func GetListTableHeader() string {
	header := fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s %-*s %-*s",
		TableColListName, "Name",
		TableColListID, "Font ID",
		TableColListLicense, "License",
		TableColListCategory, "Categories",
		TableColType, "Type",
		TableColScope, "Scope",
		TableColListSource, "Source")
	return TableHeader.Render(header)
}

// GetSourcesInfoTableHeader returns a styled header for the sources info table
// Returns a styled string ready to print (includes bold formatting)
func GetSourcesInfoTableHeader() string {
	header := fmt.Sprintf("%-*s %-*s %-*s %-*s",
		TableColSrcName, "Source Name",
		TableColSrcPrefix, "Prefix",
		TableColSrcUpdated, "Last Updated",
		TableColSrcType, "Type")
	return TableHeader.Render(header)
}

// GetTableSeparator returns a table separator line with consistent width
func GetTableSeparator() string {
	return strings.Repeat("-", TableTotalWidth)
}
