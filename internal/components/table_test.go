package components

import (
	"strings"
	"testing"
)

func TestCalculateColumnWidths(t *testing.T) {
	tests := []struct {
		name           string
		config         TableConfig
		availableWidth int
		verify         func(t *testing.T, got []int)
	}{
		{
			name: "simple equal distribution",
			config: TableConfig{
				Columns: []ColumnConfig{
					{Header: "Col1", MinWidth: 10},
					{Header: "Col2", MinWidth: 10},
				},
			},
			availableWidth: 30,
			verify: func(t *testing.T, got []int) {
				if len(got) != 2 {
					t.Fatalf("expected 2 columns, got %d", len(got))
				}
				// Total should be 29 (30 - 1 separator)
				total := got[0] + got[1]
				if total != 29 {
					t.Errorf("total width = %d, want 29", total)
				}
				// Both should be at least min width
				if got[0] < 10 || got[1] < 10 {
					t.Errorf("columns should be at least 10, got %d, %d", got[0], got[1])
				}
			},
		},
		{
			name: "minimum widths only",
			config: TableConfig{
				Columns: []ColumnConfig{
					{Header: "Col1", MinWidth: 5},
					{Header: "Col2", MinWidth: 5},
				},
			},
			availableWidth: 15,
			verify: func(t *testing.T, got []int) {
				if len(got) != 2 {
					t.Fatalf("expected 2 columns, got %d", len(got))
				}
				// Both should be at least min width
				if got[0] < 5 || got[1] < 5 {
					t.Errorf("columns should be at least 5, got %d, %d", got[0], got[1])
				}
			},
		},
		{
			name: "percentage widths",
			config: TableConfig{
				Columns: []ColumnConfig{
					{Header: "Col1", MinWidth: 5, PercentWidth: 50.0},
					{Header: "Col2", MinWidth: 5, PercentWidth: 50.0},
				},
			},
			availableWidth: 30,
			verify: func(t *testing.T, got []int) {
				if len(got) != 2 {
					t.Fatalf("expected 2 columns, got %d", len(got))
				}
				// Total should be approximately 29 (30 - 1 separator), allow for rounding
				total := got[0] + got[1]
				if total < 28 || total > 29 {
					t.Errorf("total width = %d, want 28-29", total)
				}
				// Both should be at least min width
				if got[0] < 5 || got[1] < 5 {
					t.Errorf("columns should be at least 5, got %d, %d", got[0], got[1])
				}
			},
		},
		{
			name: "fixed widths",
			config: TableConfig{
				Columns: []ColumnConfig{
					{Header: "Col1", Width: 10},
					{Header: "Col2", Width: 15},
				},
			},
			availableWidth: 30,
			verify: func(t *testing.T, got []int) {
				if len(got) != 2 {
					t.Fatalf("expected 2 columns, got %d", len(got))
				}
				if got[0] != 10 || got[1] != 15 {
					t.Errorf("got [%d, %d], want [10, 15]", got[0], got[1])
				}
			},
		},
		{
			name: "max width constraint",
			config: TableConfig{
				Columns: []ColumnConfig{
					{Header: "Col1", MinWidth: 5, MaxWidth: 10},
					{Header: "Col2", MinWidth: 5},
				},
			},
			availableWidth: 30,
			verify: func(t *testing.T, got []int) {
				if len(got) != 2 {
					t.Fatalf("expected 2 columns, got %d", len(got))
				}
				if got[0] > 10 {
					t.Errorf("column[0] should be <= 10 (max), got %d", got[0])
				}
				if got[1] < 5 {
					t.Errorf("column[1] should be >= 5 (min), got %d", got[1])
				}
			},
		},
		{
			name: "very narrow terminal",
			config: TableConfig{
				Columns: []ColumnConfig{
					{Header: "Col1", MinWidth: 5},
					{Header: "Col2", MinWidth: 5},
				},
			},
			availableWidth: 5,
			verify: func(t *testing.T, got []int) {
				if len(got) != 2 {
					t.Fatalf("expected 2 columns, got %d", len(got))
				}
				// Should respect minimum widths even if it exceeds available
				if got[0] < 5 || got[1] < 5 {
					t.Errorf("columns should respect min width 5, got %d, %d", got[0], got[1])
				}
			},
		},
		{
			name: "mixed fixed and percentage",
			config: TableConfig{
				Columns: []ColumnConfig{
					{Header: "Col1", Width: 10},
					{Header: "Col2", MinWidth: 5, PercentWidth: 50.0},
					{Header: "Col3", MinWidth: 5, PercentWidth: 50.0},
				},
			},
			availableWidth: 40,
			verify: func(t *testing.T, got []int) {
				if len(got) != 3 {
					t.Fatalf("expected 3 columns, got %d", len(got))
				}
				if got[0] != 10 {
					t.Errorf("column[0] (fixed) = %d, want 10", got[0])
				}
				if got[1] < 5 || got[2] < 5 {
					t.Errorf("columns should be at least 5, got %d, %d", got[1], got[2])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateColumnWidths(tt.config, tt.availableWidth)
			tt.verify(t, got)
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		width int
		want  string
	}{
		{
			name:  "normal truncation",
			input: "This is a long string",
			width: 10,
			want:  "This is...",
		},
		{
			name:  "no truncation needed",
			input: "Short",
			width: 10,
			want:  "Short",
		},
		{
			name:  "exact width",
			input: "Exactly10",
			width: 8,
			want:  "Exact...",
		},
		{
			name:  "very narrow",
			input: "Test",
			width: 3,
			want:  "...",
		},
		{
			name:  "empty string",
			input: "",
			width: 10,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateString(tt.input, tt.width)
			if got != tt.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
			}
		})
	}
}

func TestFormatCell(t *testing.T) {
	tests := []struct {
		name  string
		value string
		width int
		align string
		want  string
	}{
		{
			name:  "left align",
			value: "Test",
			width: 10,
			align: "left",
			want:  "Test      ",
		},
		{
			name:  "right align",
			value: "Test",
			width: 10,
			align: "right",
			want:  "      Test",
		},
		{
			name:  "center align",
			value: "Test",
			width: 10,
			align: "center",
			want:  "   Test   ",
		},
		{
			name:  "truncation with left align",
			value: "This is a very long string",
			width: 10,
			align: "left",
			want:  "This is...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCell(tt.value, tt.width, tt.align)
			if got != tt.want {
				t.Errorf("formatCell(%q, %d, %q) = %q, want %q", tt.value, tt.width, tt.align, got, tt.want)
			}
		})
	}
}

func TestRenderStaticTable(t *testing.T) {
	tests := []struct {
		name   string
		config TableConfig
		verify func(t *testing.T, output string)
	}{
		{
			name: "empty table",
			config: TableConfig{
				Columns: []ColumnConfig{},
				Rows:    [][]string{},
			},
			verify: func(t *testing.T, output string) {
				if output != "" {
					t.Errorf("RenderStaticTable() should return empty string for empty table, got: %q", output)
				}
			},
		},
		{
			name: "simple table",
			config: TableConfig{
				Columns: []ColumnConfig{
					{Header: "Name", MinWidth: 10},
					{Header: "Value", MinWidth: 10},
				},
				Rows: [][]string{
					{"Test1", "Value1"},
					{"Test2", "Value2"},
				},
				Width: 30,
			},
			verify: func(t *testing.T, output string) {
				if output == "" {
					t.Error("RenderStaticTable() returned empty string")
					return
				}
				// Check that headers are present
				if !strings.Contains(output, "Name") {
					t.Error("RenderStaticTable() output should contain 'Name' header")
				}
				if !strings.Contains(output, "Value") {
					t.Error("RenderStaticTable() output should contain 'Value' header")
				}
				// Check that data rows are present
				if !strings.Contains(output, "Test1") {
					t.Error("RenderStaticTable() output should contain 'Test1'")
				}
				if !strings.Contains(output, "Test2") {
					t.Error("RenderStaticTable() output should contain 'Test2'")
				}
			},
		},
		{
			name: "table with alignment",
			config: TableConfig{
				Columns: []ColumnConfig{
					{Header: "Left", MinWidth: 10, Align: "left"},
					{Header: "Right", MinWidth: 10, Align: "right"},
					{Header: "Center", MinWidth: 10, Align: "center"},
				},
				Rows: [][]string{
					{"L", "R", "C"},
				},
				Width: 40,
			},
			verify: func(t *testing.T, output string) {
				if output == "" {
					t.Error("RenderStaticTable() returned empty string")
				}
			},
		},
		{
			name: "table with percentage widths",
			config: TableConfig{
				Columns: []ColumnConfig{
					{Header: "Col1", MinWidth: 5, PercentWidth: 50.0},
					{Header: "Col2", MinWidth: 5, PercentWidth: 50.0},
				},
				Rows: [][]string{
					{"Data1", "Data2"},
				},
				Width: 30,
			},
			verify: func(t *testing.T, output string) {
				if output == "" {
					t.Error("RenderStaticTable() returned empty string")
				}
				if !strings.Contains(output, "Col1") {
					t.Error("RenderStaticTable() output should contain 'Col1' header")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderStaticTable(tt.config)
			tt.verify(t, got)
		})
	}
}

func TestNewTableModel(t *testing.T) {
	tests := []struct {
		name    string
		config  TableConfig
		wantErr bool
	}{
		{
			name: "valid table model",
			config: TableConfig{
				Columns: []ColumnConfig{
					{Header: "Name", MinWidth: 10},
					{Header: "Value", MinWidth: 10},
				},
				Rows: [][]string{
					{"Test1", "Value1"},
				},
				Width: 30,
				Mode:  TableModeDynamic,
			},
			wantErr: false,
		},
		{
			name: "empty columns",
			config: TableConfig{
				Columns: []ColumnConfig{},
				Rows:    [][]string{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTableModel(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTableModel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("NewTableModel() returned nil model")
			}
		})
	}
}

func TestTableModel_Focus(t *testing.T) {
	config := TableConfig{
		Columns: []ColumnConfig{
			{Header: "Name", MinWidth: 10},
		},
		Rows: [][]string{
			{"Test1"},
		},
		Width: 30,
		Mode:  TableModeDynamic,
	}

	model, err := NewTableModel(config)
	if err != nil {
		t.Fatalf("NewTableModel() error = %v", err)
	}

	if model.HasFocus() {
		t.Error("HasFocus() should return false initially")
	}

	model.SetFocus(true)
	if !model.HasFocus() {
		t.Error("HasFocus() should return true after SetFocus(true)")
	}

	model.SetFocus(false)
	if model.HasFocus() {
		t.Error("HasFocus() should return false after SetFocus(false)")
	}
}
