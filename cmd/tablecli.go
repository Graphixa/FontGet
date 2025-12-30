package cmd

import (
	"fmt"

	"fontget/internal/components"
	"fontget/internal/ui"

	"github.com/spf13/cobra"
)

var tableCliCmd = &cobra.Command{
	Use:   "tablecli",
	Short: "Test static CLI table component",
	Long: `Test command for visually verifying the static CLI table component.
This displays sample tables with various configurations to test:
- Auto-calculated column widths
- Minimum and maximum column widths
- Percentage-based widths
- Text alignment
- Text truncation with ellipsis
- Theme styling`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println()
		fmt.Println(ui.PageTitle.Render("Static CLI Table Test"))
		fmt.Println()

		// Test 1: Simple table with auto-width
		fmt.Println(ui.TextBold.Render("Test 1: Simple Table (Auto-width)"))
		fmt.Println()
		table1 := components.TableConfig{
			Columns: []components.ColumnConfig{
				{Header: "Name", MinWidth: 15},
				{Header: "Value", MinWidth: 10},
				{Header: "Status", MinWidth: 8},
			},
			Rows: [][]string{
				{"Font Name 1", "Value 1", "Active"},
				{"Font Name 2", "Value 2", "Inactive"},
				{"Very Long Font Name That Should Truncate", "Value 3", "Active"},
			},
			Width: 0, // Auto-detect
			Mode:  components.TableModeStatic,
		}
		fmt.Println(components.RenderStaticTable(table1))
		fmt.Println()

		// Test 2: Table with percentage widths
		// DEFAULT CONFIGURATION PATTERN: This configuration should be used as the default
		// for all table invocations in FontGet commands (search, list, sources, etc.)
		//
		// Key features:
		// - Uses percentage-based widths (PercentWidth) for responsive column sizing
		// - Sets minimum widths (MinWidth) to ensure readability
		// - Optionally sets maximum widths (MaxWidth) for columns that shouldn't grow too large
		// - Automatically adapts to terminal width while respecting constraints
		//
		// Recommended pattern:
		// - First column (names/identifiers): 35-40% width, MinWidth: 20
		// - Second column (IDs/codes): 25-30% width, MinWidth: 15
		// - Fixed-size columns (license, type): 10-15% width, MinWidth: 8, MaxWidth: 15
		// - Last column (source/meta): 10-15% width, MinWidth: 12
		fmt.Println(ui.TextBold.Render("Test 2: Table with Percentage Widths (DEFAULT PATTERN)"))
		fmt.Println()
		table2 := components.TableConfig{
			Columns: []components.ColumnConfig{
				{Header: "Font Name", MinWidth: 20, PercentWidth: 40.0},
				{Header: "Font ID", MinWidth: 15, PercentWidth: 30.0},
				{Header: "License", MinWidth: 8, MaxWidth: 15, PercentWidth: 15.0},
				{Header: "Source", MinWidth: 12, PercentWidth: 15.0},
			},
			Rows: [][]string{
				{"Roboto", "google.roboto", "Apache 2.0", "Google Fonts"},
				{"Fira Code", "nerd.fira-code", "SIL OFL", "Nerd Fonts"},
				{"JetBrains Mono", "jetbrains.jetbrains-mono", "SIL OFL", "JetBrains"},
				{"Very Long Font Family Name That Will Truncate", "very.long.font.id.that.should.also.truncate", "MIT", "Custom Source"},
			},
			Width: 0, // Auto-detect
			Mode:  components.TableModeStatic,
		}
		fmt.Println(components.RenderStaticTable(table2))
		fmt.Println()

		// Test 3: Table with alignment
		fmt.Println(ui.TextBold.Render("Test 3: Table with Different Alignments"))
		fmt.Println()
		table3 := components.TableConfig{
			Columns: []components.ColumnConfig{
				{Header: "Left Aligned", MinWidth: 15, Align: "left"},
				{Header: "Center Aligned", MinWidth: 15, Align: "center"},
				{Header: "Right Aligned", MinWidth: 15, Align: "right"},
			},
			Rows: [][]string{
				{"Left", "Center", "Right"},
				{"Text", "Text", "Text"},
				{"Longer Text", "Center Text", "Right Text"},
			},
			Width: 0, // Auto-detect
			Mode:  components.TableModeStatic,
		}
		fmt.Println(components.RenderStaticTable(table3))
		fmt.Println()

		// Test 4: Narrow terminal simulation
		fmt.Println(ui.TextBold.Render("Test 4: Narrow Terminal (60 chars)"))
		fmt.Println()
		table4 := components.TableConfig{
			Columns: []components.ColumnConfig{
				{Header: "Name", MinWidth: 10, PercentWidth: 50.0},
				{Header: "ID", MinWidth: 8, PercentWidth: 30.0},
				{Header: "License", MinWidth: 6, MaxWidth: 10, PercentWidth: 20.0},
			},
			Rows: [][]string{
				{"Roboto", "google.roboto", "Apache"},
				{"Fira Code", "nerd.fira-code", "OFL"},
				{"Very Long Name", "very.long.id", "MIT License"},
			},
			Width: 60, // Fixed narrow width
			Mode:  components.TableModeStatic,
		}
		fmt.Println(components.RenderStaticTable(table4))
		fmt.Println()

		// Test 5: Wide terminal simulation
		fmt.Println(ui.TextBold.Render("Test 5: Wide Terminal (150 chars)"))
		fmt.Println()
		table5 := components.TableConfig{
			Columns: []components.ColumnConfig{
				{Header: "Font Name", MinWidth: 20, PercentWidth: 30.0},
				{Header: "Font ID", MinWidth: 15, PercentWidth: 25.0},
				{Header: "License", MinWidth: 12, MaxWidth: 20, PercentWidth: 15.0},
				{Header: "Categories", MinWidth: 20, PercentWidth: 20.0},
				{Header: "Source", MinWidth: 15, PercentWidth: 10.0},
			},
			Rows: [][]string{
				{"Roboto", "google.roboto", "Apache 2.0", "Sans Serif, Display", "Google Fonts"},
				{"Fira Code", "nerd.fira-code", "SIL OFL 1.1", "Monospace, Programming", "Nerd Fonts"},
			},
			Width: 150, // Fixed wide width
			Mode:  components.TableModeStatic,
		}
		fmt.Println(components.RenderStaticTable(table5))
		fmt.Println()

		fmt.Println(ui.SuccessText.Render("Table CLI test completed successfully!"))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(tableCliCmd)
}
