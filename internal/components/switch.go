package components

import (
	"fmt"

	"fontget/internal/ui"
)

// Switch represents a switch/toggle component
type Switch struct {
	LeftLabel  string // Default: "Enable"
	RightLabel string // Default: "Disable"
	Value      bool   // true = left (Enable), false = right (Disable)
	Width      int    // Total width of switch
}

// Render renders the switch with appropriate styling
func (s Switch) Render() string {
	// Default labels if not set
	leftLabel := s.LeftLabel
	if leftLabel == "" {
		leftLabel = "Enable"
	}
	rightLabel := s.RightLabel
	if rightLabel == "" {
		rightLabel = "Disable"
	}

	// Add padding to labels (2 spaces on each side)
	leftPadded := fmt.Sprintf("  %s  ", leftLabel)
	rightPadded := fmt.Sprintf("  %s  ", rightLabel)

	// Apply styling based on selection
	var leftStyled, rightStyled string
	if s.Value {
		// Left (Enable) is selected
		leftStyled = ui.SwitchLeftSelected.Render(leftPadded)
		rightStyled = ui.SwitchRightNormal.Render(rightPadded)
	} else {
		// Right (Disable) is selected
		leftStyled = ui.SwitchLeftNormal.Render(leftPadded)
		rightStyled = ui.SwitchRightSelected.Render(rightPadded)
	}

	// Separator
	separator := ui.SwitchSeparator.Render("|")

	// Combine: [  Left  |  Right  ]
	return fmt.Sprintf("[%s%s%s]", leftStyled, separator, rightStyled)
}

// Toggle switches the value
func (s *Switch) Toggle() {
	s.Value = !s.Value
}

// SetValue sets the value explicitly
func (s *Switch) SetValue(value bool) {
	s.Value = value
}

// HandleKey handles keyboard input for switch toggling
// Returns: true if the key was handled, false otherwise
func (s *Switch) HandleKey(key string) bool {
	switch key {
	case "left", "h":
		s.Value = true // Select left
		return true
	case "right", "l":
		s.Value = false // Select right
		return true
	case " ", "enter":
		s.Toggle()
		return true
	default:
		return false
	}
}

// NewSwitch creates a new switch with default labels
func NewSwitch(value bool) *Switch {
	return &Switch{
		LeftLabel:  "Enable",
		RightLabel: "Disable",
		Value:      value,
		Width:      30,
	}
}

// NewSwitchWithLabels creates a new switch with custom labels
func NewSwitchWithLabels(leftLabel, rightLabel string, value bool) *Switch {
	return &Switch{
		LeftLabel:  leftLabel,
		RightLabel: rightLabel,
		Value:      value,
		Width:      30,
	}
}
