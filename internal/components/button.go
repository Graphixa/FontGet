package components

import (
	"fmt"
	"strings"

	"fontget/internal/ui"
)

// Button represents a single button
type Button struct {
	Text     string
	Selected bool
	Action   string // "ok", "next", "back", "accept", "cancel", or custom
}

// ButtonGroup represents a group of buttons that can be navigated
type ButtonGroup struct {
	Buttons       []Button
	Selected      int  // Index of selected button
	HasFocus      bool // Whether buttons currently have focus
	DefaultButton int  // Index of default button (shown but not focused)
}

// Render renders a single button with appropriate styling
func (b Button) Render() string {
	padding := 2 // Spaces on each side
	paddedText := fmt.Sprintf("%s%s%s", strings.Repeat(" ", padding), b.Text, strings.Repeat(" ", padding))
	brackets := fmt.Sprintf("[%s]", paddedText)

	if b.Selected {
		return ui.ButtonSelected.Render(brackets)
	}
	return ui.ButtonNormal.Render(brackets)
}

// RenderFullWidth renders a button that expands to fill the specified width
// The text is centered within the button, and the brackets expand to fill the width
func (b Button) RenderFullWidth(width int) string {
	if width < 5 {
		width = 5 // Minimum width for a button
	}

	// Calculate available space for text and padding
	// Format: [  Text  ] - brackets take 2 chars, padding takes 4 chars (2 each side)
	availableTextWidth := width - 6 // 2 for brackets + 4 for padding
	if availableTextWidth < 0 {
		availableTextWidth = 0
	}

	// Truncate text if too long
	text := b.Text
	if len(text) > availableTextWidth {
		text = text[:availableTextWidth]
	}

	// Calculate padding to center text
	textWidth := len(text)
	totalPadding := availableTextWidth - textWidth
	leftPadding := totalPadding / 2
	rightPadding := totalPadding - leftPadding

	// Build the button: [  Text  ]
	paddedText := fmt.Sprintf("%s%s%s",
		strings.Repeat(" ", leftPadding+2), // +2 for inner padding
		text,
		strings.Repeat(" ", rightPadding+2)) // +2 for inner padding
	brackets := fmt.Sprintf("[%s]", paddedText)

	// Apply styling
	if b.Selected {
		return ui.ButtonSelected.Render(brackets)
	}
	return ui.ButtonNormal.Render(brackets)
}

// RenderGroup renders a button group with spacing between buttons
// Only shows selected state when HasFocus is true
func (bg ButtonGroup) Render() string {
	var rendered []string
	for i, button := range bg.Buttons {
		// Only show as selected if buttons have focus AND this is the selected button
		// Otherwise, show default button with normal style
		buttonCopy := button
		if !bg.HasFocus {
			buttonCopy.Selected = false
		} else {
			buttonCopy.Selected = (i == bg.Selected)
		}
		rendered = append(rendered, buttonCopy.Render())
	}
	return strings.Join(rendered, "  ") // Two spaces between buttons
}

// HandleKey handles keyboard input for button navigation
// Returns: action string if button was activated, "" otherwise
func (bg *ButtonGroup) HandleKey(key string) string {
	// When left/right is pressed, give focus to buttons
	if key == "left" || key == "right" || key == "h" || key == "l" {
		bg.HasFocus = true
	}

	switch key {
	case "left", "h":
		if bg.Selected > 0 {
			bg.Selected--
		}
		return ""
	case "right", "l":
		if bg.Selected < len(bg.Buttons)-1 {
			bg.Selected++
		}
		return ""
	case "enter", " ":
		// Allow activation even without focus (for fallback cases)
		// Focus check is mainly for visual indication
		if bg.Selected >= 0 && bg.Selected < len(bg.Buttons) {
			return bg.Buttons[bg.Selected].Action
		}
		return ""
	default:
		return ""
	}
}

// SetFocus sets whether buttons have focus
func (bg *ButtonGroup) SetFocus(hasFocus bool) {
	bg.HasFocus = hasFocus
}

// ResetToDefault resets the selected button to the default button
func (bg *ButtonGroup) ResetToDefault() {
	if bg.DefaultButton >= 0 && bg.DefaultButton < len(bg.Buttons) {
		// Clear all selections
		for i := range bg.Buttons {
			bg.Buttons[i].Selected = false
		}
		// Set default button as selected (only if buttons have focus)
		bg.Selected = bg.DefaultButton
	}
}

// NewButtonGroup creates a new button group with the specified buttons
// defaultSelected is the index of the button that should be selected when buttons have focus
func NewButtonGroup(buttonTexts []string, defaultSelected int) *ButtonGroup {
	if defaultSelected < 0 || defaultSelected >= len(buttonTexts) {
		defaultSelected = 0
	}

	buttons := make([]Button, len(buttonTexts))
	for i, text := range buttonTexts {
		// Normalize to lower for action matching; map is currently identity but kept for clarity
		action := strings.ToLower(text)

		buttons[i] = Button{
			Text:     text,
			Selected: false, // Buttons start unselected (no focus)
			Action:   action,
		}
	}

	return &ButtonGroup{
		Buttons:       buttons,
		Selected:      defaultSelected,
		HasFocus:      false, // Start without focus (list has focus)
		DefaultButton: defaultSelected,
	}
}
