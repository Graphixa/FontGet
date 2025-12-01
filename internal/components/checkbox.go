package components

import (
	"fmt"
	"strings"

	"fontget/internal/ui"
)

// CheckboxItem represents a single checkbox item
type CheckboxItem struct {
	Label   string
	Checked bool
	Enabled bool // For disabling certain items
}

// CheckboxList represents a list of checkboxes that can be navigated
type CheckboxList struct {
	Items    []CheckboxItem
	Cursor   int
	Width    int
	Height   int
	HasFocus bool // Whether checkbox list currently has focus
}

// Render renders the checkbox list
// Only shows cursor and selection when HasFocus is true
func (cl CheckboxList) Render() string {
	var result strings.Builder

	for i, item := range cl.Items {
		// Determine checkbox symbol
		var checkbox string
		if item.Checked {
			checkbox = ui.CheckboxChecked.Render("[x]")
		} else {
			checkbox = ui.CheckboxUnchecked.Render("[ ]")
		}

		// Build the line
		var line string
		if cl.HasFocus && i == cl.Cursor {
			// Selected row - show cursor and use selected style (only when focused)
			cursor := ui.CheckboxCursor.Render("> ")
			line = fmt.Sprintf("%s%s %s", cursor, checkbox, item.Label)
			line = ui.CheckboxItemSelected.Render(line)
		} else {
			// Normal row
			line = fmt.Sprintf("  %s %s", checkbox, item.Label)
		}

		// Disable styling if item is disabled
		if !item.Enabled {
			line = ui.Text.Render(line)
		}

		result.WriteString(line)
		result.WriteString("\n")
	}

	return result.String()
}

// Toggle toggles the checkbox at the given index
func (cl *CheckboxList) Toggle(index int) {
	if index >= 0 && index < len(cl.Items) && cl.Items[index].Enabled {
		cl.Items[index].Checked = !cl.Items[index].Checked
	}
}

// HandleKey handles keyboard input for checkbox navigation and toggling
// Returns: true if the key was handled, false otherwise
func (cl *CheckboxList) HandleKey(key string) bool {
	// When up/down/space/enter is pressed, give focus to checkbox list
	if key == "up" || key == "down" || key == "k" || key == "j" || key == " " || key == "enter" {
		cl.HasFocus = true
	}

	switch key {
	case "up", "k":
		if cl.Cursor > 0 {
			cl.Cursor--
		}
		return true
	case "down", "j":
		if cl.Cursor < len(cl.Items)-1 {
			cl.Cursor++
		}
		return true
	case " ", "enter":
		cl.Toggle(cl.Cursor)
		return true
	default:
		return false
	}
}

// SetFocus sets whether checkbox list has focus
func (cl *CheckboxList) SetFocus(hasFocus bool) {
	cl.HasFocus = hasFocus
}

// NewCheckboxList creates a new checkbox list
func NewCheckboxList(items []CheckboxItem) *CheckboxList {
	return &CheckboxList{
		Items:    items,
		Cursor:   0,
		Width:    80,
		Height:   24,
		HasFocus: true, // Checkbox list starts with focus by default
	}
}
