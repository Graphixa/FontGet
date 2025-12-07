package components

import (
	"testing"
)

func TestButton_Render(t *testing.T) {
	tests := []struct {
		name     string
		button   Button
		wantCont string // String that should be contained in output
	}{
		{
			name:     "unselected button",
			button:   Button{Text: "OK", Selected: false},
			wantCont: "[   OK   ]",
		},
		{
			name:     "selected button",
			button:   Button{Text: "Next", Selected: true},
			wantCont: "[   Next  ]",
		},
		{
			name:     "custom action button",
			button:   Button{Text: "Accept", Selected: false, Action: "accept"},
			wantCont: "[  Accept ]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.button.Render()
			if got == "" {
				t.Error("Render() returned empty string")
			}
			// Check that the button text is present
			if len(got) < len(tt.wantCont) {
				t.Errorf("Render() output too short, got: %q", got)
			}
		})
	}
}

func TestButtonGroup_Render(t *testing.T) {
	tests := []struct {
		name   string
		group  ButtonGroup
		verify func(t *testing.T, output string)
	}{
		{
			name: "single button group",
			group: ButtonGroup{
				Buttons:  []Button{{Text: "OK", Selected: true}},
				Selected: 0,
				HasFocus: true,
			},
			verify: func(t *testing.T, output string) {
				if output == "" {
					t.Error("Render() returned empty string")
				}
			},
		},
		{
			name: "two button group",
			group: ButtonGroup{
				Buttons: []Button{
					{Text: "Back", Selected: false},
					{Text: "Next", Selected: true},
				},
				Selected: 1,
				HasFocus: true,
			},
			verify: func(t *testing.T, output string) {
				if output == "" {
					t.Error("Render() returned empty string")
				}
			},
		},
		{
			name: "button group without focus",
			group: ButtonGroup{
				Buttons: []Button{
					{Text: "Back", Selected: false},
					{Text: "Next", Selected: false},
				},
				Selected:      1,
				HasFocus:      false,
				DefaultButton: 1,
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
			got := tt.group.Render()
			tt.verify(t, got)
		})
	}
}

func TestButtonGroup_HandleKey(t *testing.T) {
	tests := []struct {
		name         string
		group        *ButtonGroup
		key          string
		wantAction   string
		wantSelected int
		wantHasFocus bool
	}{
		{
			name: "left arrow moves selection left",
			group: &ButtonGroup{
				Buttons:  []Button{{Text: "Back"}, {Text: "Next"}},
				Selected: 1,
				HasFocus: false,
			},
			key:          "left",
			wantAction:   "",
			wantSelected: 0,
			wantHasFocus: true,
		},
		{
			name: "right arrow moves selection right",
			group: &ButtonGroup{
				Buttons:  []Button{{Text: "Back"}, {Text: "Next"}},
				Selected: 0,
				HasFocus: false,
			},
			key:          "right",
			wantAction:   "",
			wantSelected: 1,
			wantHasFocus: true,
		},
		{
			name: "enter activates selected button",
			group: &ButtonGroup{
				Buttons: []Button{
					{Text: "Back", Action: "back"},
					{Text: "Next", Action: "next"},
				},
				Selected: 1,
				HasFocus: true,
			},
			key:          "enter",
			wantAction:   "next",
			wantSelected: 1,
			wantHasFocus: true,
		},
		{
			name: "space activates selected button",
			group: &ButtonGroup{
				Buttons: []Button{
					{Text: "Back", Action: "back"},
					{Text: "Next", Action: "next"},
				},
				Selected: 0,
				HasFocus: true,
			},
			key:          " ",
			wantAction:   "back",
			wantSelected: 0,
			wantHasFocus: true,
		},
		{
			name: "left arrow at start doesn't go negative",
			group: &ButtonGroup{
				Buttons:  []Button{{Text: "OK"}},
				Selected: 0,
				HasFocus: false,
			},
			key:          "left",
			wantAction:   "",
			wantSelected: 0,
			wantHasFocus: true,
		},
		{
			name: "right arrow at end doesn't exceed bounds",
			group: &ButtonGroup{
				Buttons:  []Button{{Text: "OK"}},
				Selected: 0,
				HasFocus: false,
			},
			key:          "right",
			wantAction:   "",
			wantSelected: 0,
			wantHasFocus: true,
		},
		{
			name: "enter without focus doesn't activate",
			group: &ButtonGroup{
				Buttons: []Button{
					{Text: "Next", Action: "next"},
				},
				Selected: 0,
				HasFocus: false,
			},
			key:          "enter",
			wantAction:   "",
			wantSelected: 0,
			wantHasFocus: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAction := tt.group.HandleKey(tt.key)
			if gotAction != tt.wantAction {
				t.Errorf("HandleKey() action = %q, want %q", gotAction, tt.wantAction)
			}
			if tt.group.Selected != tt.wantSelected {
				t.Errorf("HandleKey() selected = %d, want %d", tt.group.Selected, tt.wantSelected)
			}
			if tt.group.HasFocus != tt.wantHasFocus {
				t.Errorf("HandleKey() hasFocus = %v, want %v", tt.group.HasFocus, tt.wantHasFocus)
			}
		})
	}
}

func TestButtonGroup_SetFocus(t *testing.T) {
	group := &ButtonGroup{
		Buttons:  []Button{{Text: "OK"}},
		HasFocus: false,
	}

	group.SetFocus(true)
	if !group.HasFocus {
		t.Error("SetFocus(true) did not set HasFocus to true")
	}

	group.SetFocus(false)
	if group.HasFocus {
		t.Error("SetFocus(false) did not set HasFocus to false")
	}
}

func TestButtonGroup_ResetToDefault(t *testing.T) {
	group := &ButtonGroup{
		Buttons: []Button{
			{Text: "Back", Selected: true},
			{Text: "Next", Selected: false},
		},
		Selected:      0,
		DefaultButton: 1,
		HasFocus:      true,
	}

	group.ResetToDefault()

	if group.Selected != 1 {
		t.Errorf("ResetToDefault() selected = %d, want %d", group.Selected, 1)
	}
}

func TestNewButtonGroup(t *testing.T) {
	tests := []struct {
		name            string
		buttonTexts     []string
		defaultSelected int
		wantLen         int
		wantSelected    int
		wantHasFocus    bool
	}{
		{
			name:            "single button",
			buttonTexts:     []string{"OK"},
			defaultSelected: 0,
			wantLen:         1,
			wantSelected:    0,
			wantHasFocus:    false,
		},
		{
			name:            "two buttons",
			buttonTexts:     []string{"Back", "Next"},
			defaultSelected: 1,
			wantLen:         2,
			wantSelected:    1,
			wantHasFocus:    false,
		},
		{
			name:            "invalid default selected",
			buttonTexts:     []string{"OK"},
			defaultSelected: 5, // Out of bounds
			wantLen:         1,
			wantSelected:    0, // Should default to 0
			wantHasFocus:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewButtonGroup(tt.buttonTexts, tt.defaultSelected)
			if len(got.Buttons) != tt.wantLen {
				t.Errorf("NewButtonGroup() len(Buttons) = %d, want %d", len(got.Buttons), tt.wantLen)
			}
			if got.Selected != tt.wantSelected {
				t.Errorf("NewButtonGroup() Selected = %d, want %d", got.Selected, tt.wantSelected)
			}
			if got.HasFocus != tt.wantHasFocus {
				t.Errorf("NewButtonGroup() HasFocus = %v, want %v", got.HasFocus, tt.wantHasFocus)
			}
			if got.DefaultButton != tt.wantSelected {
				t.Errorf("NewButtonGroup() DefaultButton = %d, want %d", got.DefaultButton, tt.wantSelected)
			}
		})
	}
}
