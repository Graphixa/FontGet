package components

import (
	"testing"
)

func TestCheckboxList_Render(t *testing.T) {
	tests := []struct {
		name   string
		list   CheckboxList
		verify func(t *testing.T, output string)
	}{
		{
			name: "empty list",
			list: CheckboxList{
				Items:    []CheckboxItem{},
				Cursor:   0,
				HasFocus: true,
			},
			verify: func(t *testing.T, output string) {
				if output != "" {
					t.Errorf("Render() should return empty string for empty list, got: %q", output)
				}
			},
		},
		{
			name: "single checked item",
			list: CheckboxList{
				Items: []CheckboxItem{
					{Label: "Item 1", Checked: true, Enabled: true},
				},
				Cursor:   0,
				HasFocus: true,
			},
			verify: func(t *testing.T, output string) {
				if output == "" {
					t.Error("Render() returned empty string")
				}
			},
		},
		{
			name: "multiple items with cursor",
			list: CheckboxList{
				Items: []CheckboxItem{
					{Label: "Item 1", Checked: false, Enabled: true},
					{Label: "Item 2", Checked: true, Enabled: true},
					{Label: "Item 3", Checked: false, Enabled: true},
				},
				Cursor:   1,
				HasFocus: true,
			},
			verify: func(t *testing.T, output string) {
				if output == "" {
					t.Error("Render() returned empty string")
				}
			},
		},
		{
			name: "list without focus",
			list: CheckboxList{
				Items: []CheckboxItem{
					{Label: "Item 1", Checked: true, Enabled: true},
				},
				Cursor:   0,
				HasFocus: false,
			},
			verify: func(t *testing.T, output string) {
				if output == "" {
					t.Error("Render() returned empty string")
				}
			},
		},
		{
			name: "disabled item",
			list: CheckboxList{
				Items: []CheckboxItem{
					{Label: "Disabled Item", Checked: false, Enabled: false},
				},
				Cursor:   0,
				HasFocus: true,
			},
			verify: func(t *testing.T, output string) {
				if output == "" {
					t.Error("Render() returned empty string")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.list.Render()
			tt.verify(t, got)
		})
	}
}

func TestCheckboxList_Toggle(t *testing.T) {
	tests := []struct {
		name      string
		list      *CheckboxList
		index     int
		wantState bool
	}{
		{
			name: "toggle unchecked to checked",
			list: &CheckboxList{
				Items: []CheckboxItem{
					{Label: "Item 1", Checked: false, Enabled: true},
				},
			},
			index:     0,
			wantState: true,
		},
		{
			name: "toggle checked to unchecked",
			list: &CheckboxList{
				Items: []CheckboxItem{
					{Label: "Item 1", Checked: true, Enabled: true},
				},
			},
			index:     0,
			wantState: false,
		},
		{
			name: "toggle disabled item (should not change)",
			list: &CheckboxList{
				Items: []CheckboxItem{
					{Label: "Item 1", Checked: false, Enabled: false},
				},
			},
			index:     0,
			wantState: false, // Should remain unchanged
		},
		{
			name: "toggle out of bounds (should not panic)",
			list: &CheckboxList{
				Items: []CheckboxItem{
					{Label: "Item 1", Checked: false, Enabled: true},
				},
			},
			index:     10, // Out of bounds
			wantState: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initialState := tt.list.Items[0].Checked
			tt.list.Toggle(tt.index)
			if tt.index < len(tt.list.Items) && tt.list.Items[tt.index].Enabled {
				if tt.list.Items[tt.index].Checked != tt.wantState {
					t.Errorf("Toggle() state = %v, want %v", tt.list.Items[tt.index].Checked, tt.wantState)
				}
			} else if tt.index < len(tt.list.Items) && !tt.list.Items[tt.index].Enabled {
				// Disabled items should not change
				if tt.list.Items[tt.index].Checked != initialState {
					t.Errorf("Toggle() disabled item changed state from %v", initialState)
				}
			}
		})
	}
}

func TestCheckboxList_HandleKey(t *testing.T) {
	tests := []struct {
		name         string
		list         *CheckboxList
		key          string
		wantHandled  bool
		wantCursor   int
		wantHasFocus bool
		wantToggled  bool // Whether the item at cursor should be toggled
	}{
		{
			name: "up arrow moves cursor up",
			list: &CheckboxList{
				Items: []CheckboxItem{
					{Label: "Item 1", Checked: false, Enabled: true},
					{Label: "Item 2", Checked: false, Enabled: true},
				},
				Cursor:   1,
				HasFocus: false,
			},
			key:          "up",
			wantHandled:  true,
			wantCursor:   0,
			wantHasFocus: true,
			wantToggled:  false,
		},
		{
			name: "down arrow moves cursor down",
			list: &CheckboxList{
				Items: []CheckboxItem{
					{Label: "Item 1", Checked: false, Enabled: true},
					{Label: "Item 2", Checked: false, Enabled: true},
				},
				Cursor:   0,
				HasFocus: false,
			},
			key:          "down",
			wantHandled:  true,
			wantCursor:   1,
			wantHasFocus: true,
			wantToggled:  false,
		},
		{
			name: "space toggles checkbox",
			list: &CheckboxList{
				Items: []CheckboxItem{
					{Label: "Item 1", Checked: false, Enabled: true},
				},
				Cursor:   0,
				HasFocus: false,
			},
			key:          " ",
			wantHandled:  true,
			wantCursor:   0,
			wantHasFocus: true,
			wantToggled:  true,
		},
		{
			name: "enter toggles checkbox",
			list: &CheckboxList{
				Items: []CheckboxItem{
					{Label: "Item 1", Checked: true, Enabled: true},
				},
				Cursor:   0,
				HasFocus: false,
			},
			key:          "enter",
			wantHandled:  true,
			wantCursor:   0,
			wantHasFocus: true,
			wantToggled:  true,
		},
		{
			name: "up arrow at start doesn't go negative",
			list: &CheckboxList{
				Items: []CheckboxItem{
					{Label: "Item 1", Checked: false, Enabled: true},
				},
				Cursor:   0,
				HasFocus: false,
			},
			key:          "up",
			wantHandled:  true,
			wantCursor:   0,
			wantHasFocus: true,
			wantToggled:  false,
		},
		{
			name: "down arrow at end doesn't exceed bounds",
			list: &CheckboxList{
				Items: []CheckboxItem{
					{Label: "Item 1", Checked: false, Enabled: true},
				},
				Cursor:   0,
				HasFocus: false,
			},
			key:          "down",
			wantHandled:  true,
			wantCursor:   0,
			wantHasFocus: true,
			wantToggled:  false,
		},
		{
			name: "unknown key not handled",
			list: &CheckboxList{
				Items: []CheckboxItem{
					{Label: "Item 1", Checked: false, Enabled: true},
				},
				Cursor:   0,
				HasFocus: false,
			},
			key:          "x",
			wantHandled:  false,
			wantCursor:   0,
			wantHasFocus: false,
			wantToggled:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initialChecked := false
			if len(tt.list.Items) > 0 && tt.list.Cursor < len(tt.list.Items) {
				initialChecked = tt.list.Items[tt.list.Cursor].Checked
			}

			gotHandled := tt.list.HandleKey(tt.key)

			if gotHandled != tt.wantHandled {
				t.Errorf("HandleKey() handled = %v, want %v", gotHandled, tt.wantHandled)
			}
			if tt.list.Cursor != tt.wantCursor {
				t.Errorf("HandleKey() cursor = %d, want %d", tt.list.Cursor, tt.wantCursor)
			}
			if tt.list.HasFocus != tt.wantHasFocus {
				t.Errorf("HandleKey() hasFocus = %v, want %v", tt.list.HasFocus, tt.wantHasFocus)
			}

			if tt.wantToggled && len(tt.list.Items) > 0 && tt.list.Cursor < len(tt.list.Items) {
				if tt.list.Items[tt.list.Cursor].Checked == initialChecked {
					t.Errorf("HandleKey() should have toggled checkbox, but state unchanged")
				}
			}
		})
	}
}

func TestCheckboxList_SetFocus(t *testing.T) {
	list := &CheckboxList{
		Items:    []CheckboxItem{{Label: "Item 1", Checked: false, Enabled: true}},
		HasFocus: false,
	}

	list.SetFocus(true)
	if !list.HasFocus {
		t.Error("SetFocus(true) did not set HasFocus to true")
	}

	list.SetFocus(false)
	if list.HasFocus {
		t.Error("SetFocus(false) did not set HasFocus to false")
	}
}

func TestNewCheckboxList(t *testing.T) {
	items := []CheckboxItem{
		{Label: "Item 1", Checked: false, Enabled: true},
		{Label: "Item 2", Checked: true, Enabled: true},
	}

	list := NewCheckboxList(items)

	if len(list.Items) != len(items) {
		t.Errorf("NewCheckboxList() len(Items) = %d, want %d", len(list.Items), len(items))
	}
	if list.Cursor != 0 {
		t.Errorf("NewCheckboxList() Cursor = %d, want 0", list.Cursor)
	}
	if !list.HasFocus {
		t.Errorf("NewCheckboxList() HasFocus = %v, want true", list.HasFocus)
	}
}
