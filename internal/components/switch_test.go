package components

import (
	"testing"
)

func TestSwitch_Render(t *testing.T) {
	tests := []struct {
		name     string
		switch_  Switch
		wantCont string // String that should be contained in output
	}{
		{
			name: "default labels enabled",
			switch_: Switch{
				LeftLabel:  "",
				RightLabel: "",
				Value:      true,
			},
			wantCont: "Enable",
		},
		{
			name: "default labels disabled",
			switch_: Switch{
				LeftLabel:  "",
				RightLabel: "",
				Value:      false,
			},
			wantCont: "Disable",
		},
		{
			name: "custom labels left selected",
			switch_: Switch{
				LeftLabel:  "On",
				RightLabel: "Off",
				Value:      true,
			},
			wantCont: "On",
		},
		{
			name: "custom labels right selected",
			switch_: Switch{
				LeftLabel:  "On",
				RightLabel: "Off",
				Value:      false,
			},
			wantCont: "Off",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.switch_.Render()
			if got == "" {
				t.Error("Render() returned empty string")
			}
			// Check that the expected label is present
			if len(got) < len(tt.wantCont) {
				t.Errorf("Render() output too short, got: %q", got)
			}
		})
	}
}

func TestSwitch_Toggle(t *testing.T) {
	tests := []struct {
		name      string
		switch_   *Switch
		wantValue bool
	}{
		{
			name: "toggle from true to false",
			switch_: &Switch{
				Value: true,
			},
			wantValue: false,
		},
		{
			name: "toggle from false to true",
			switch_: &Switch{
				Value: false,
			},
			wantValue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.switch_.Toggle()
			if tt.switch_.Value != tt.wantValue {
				t.Errorf("Toggle() Value = %v, want %v", tt.switch_.Value, tt.wantValue)
			}
		})
	}
}

func TestSwitch_SetValue(t *testing.T) {
	switch_ := &Switch{
		Value: false,
	}

	switch_.SetValue(true)
	if !switch_.Value {
		t.Error("SetValue(true) did not set Value to true")
	}

	switch_.SetValue(false)
	if switch_.Value {
		t.Error("SetValue(false) did not set Value to false")
	}
}

func TestSwitch_HandleKey(t *testing.T) {
	tests := []struct {
		name        string
		switch_     *Switch
		key         string
		wantValue   bool
		wantHandled bool
	}{
		{
			name: "left arrow selects left (true)",
			switch_: &Switch{
				Value: false,
			},
			key:         "left",
			wantValue:   true,
			wantHandled: true,
		},
		{
			name: "right arrow selects right (false)",
			switch_: &Switch{
				Value: true,
			},
			key:         "right",
			wantValue:   false,
			wantHandled: true,
		},
		{
			name: "space toggles",
			switch_: &Switch{
				Value: false,
			},
			key:         " ",
			wantValue:   true,
			wantHandled: true,
		},
		{
			name: "enter toggles",
			switch_: &Switch{
				Value: true,
			},
			key:         "enter",
			wantValue:   false,
			wantHandled: true,
		},
		{
			name: "h key selects left (true)",
			switch_: &Switch{
				Value: false,
			},
			key:         "h",
			wantValue:   true,
			wantHandled: true,
		},
		{
			name: "l key selects right (false)",
			switch_: &Switch{
				Value: true,
			},
			key:         "l",
			wantValue:   false,
			wantHandled: true,
		},
		{
			name: "unknown key not handled",
			switch_: &Switch{
				Value: false,
			},
			key:         "x",
			wantValue:   false,
			wantHandled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHandled := tt.switch_.HandleKey(tt.key)
			if gotHandled != tt.wantHandled {
				t.Errorf("HandleKey() handled = %v, want %v", gotHandled, tt.wantHandled)
			}
			if tt.switch_.Value != tt.wantValue {
				t.Errorf("HandleKey() Value = %v, want %v", tt.switch_.Value, tt.wantValue)
			}
		})
	}
}

func TestNewSwitch(t *testing.T) {
	tests := []struct {
		name      string
		value     bool
		wantValue bool
		wantLeft  string
		wantRight string
	}{
		{
			name:      "new switch true",
			value:     true,
			wantValue: true,
			wantLeft:  "Enable",
			wantRight: "Disable",
		},
		{
			name:      "new switch false",
			value:     false,
			wantValue: false,
			wantLeft:  "Enable",
			wantRight: "Disable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSwitch(tt.value)
			if got.Value != tt.wantValue {
				t.Errorf("NewSwitch() Value = %v, want %v", got.Value, tt.wantValue)
			}
			if got.LeftLabel != tt.wantLeft {
				t.Errorf("NewSwitch() LeftLabel = %q, want %q", got.LeftLabel, tt.wantLeft)
			}
			if got.RightLabel != tt.wantRight {
				t.Errorf("NewSwitch() RightLabel = %q, want %q", got.RightLabel, tt.wantRight)
			}
		})
	}
}

func TestNewSwitchWithLabels(t *testing.T) {
	got := NewSwitchWithLabels("On", "Off", true)

	if got.Value != true {
		t.Errorf("NewSwitchWithLabels() Value = %v, want true", got.Value)
	}
	if got.LeftLabel != "On" {
		t.Errorf("NewSwitchWithLabels() LeftLabel = %q, want %q", got.LeftLabel, "On")
	}
	if got.RightLabel != "Off" {
		t.Errorf("NewSwitchWithLabels() RightLabel = %q, want %q", got.RightLabel, "Off")
	}
}
