package components

// FormNavigation handles navigation between a list component and buttons
// It provides a consistent navigation pattern: Tab switches focus, Up/Down navigates list,
// Left/Right navigates buttons, and automatically switches focus at boundaries.
type FormNavigation struct {
	// List state
	ListFocused   bool
	ListCursor    int
	ListLength    int
	ListHasFocus  func() bool
	ListSetFocus  func(bool)
	ListNavigate  func(direction string) bool // Returns true if navigation was handled
	ListGetCursor func() int
	ListSetCursor func(int)

	// Button state
	ButtonGroup *ButtonGroup
}

// NewFormNavigation creates a new FormNavigation instance
func NewFormNavigation(listLength int, buttonGroup *ButtonGroup) *FormNavigation {
	return &FormNavigation{
		ListFocused: true,
		ListCursor:  0,
		ListLength:  listLength,
		ButtonGroup: buttonGroup,
	}
}

// HandleKey processes a key press and returns:
// - handled: whether the key was handled
// - action: any action that should be taken (e.g., button action)
// - listAction: any list-specific action (e.g., toggle)
func (fn *FormNavigation) HandleKey(key string) (handled bool, action string, listAction string) {
	// Tab switches focus between list and buttons
	if key == "tab" {
		fn.ListFocused = !fn.ListFocused
		if fn.ListSetFocus != nil {
			fn.ListSetFocus(fn.ListFocused)
		}
		if fn.ButtonGroup != nil {
			fn.ButtonGroup.SetFocus(!fn.ListFocused)
		}
		return true, "", ""
	}

	// Handle list navigation when list has focus
	if fn.ListFocused {
		return fn.handleListNavigation(key)
	}

	// Handle button navigation when buttons have focus
	if fn.ButtonGroup != nil && fn.ButtonGroup.HasFocus {
		return fn.handleButtonNavigation(key)
	}

	return false, "", ""
}

// handleListNavigation handles keys when the list has focus
func (fn *FormNavigation) handleListNavigation(key string) (handled bool, action string, listAction string) {
	switch key {
	case "up", "k":
		if fn.ListCursor > 0 {
			fn.ListCursor--
			if fn.ListSetCursor != nil {
				fn.ListSetCursor(fn.ListCursor)
			}
			return true, "", ""
		}
		return true, "", "" // At top, stay there

	case "down", "j":
		if fn.ListCursor < fn.ListLength-1 {
			fn.ListCursor++
			if fn.ListSetCursor != nil {
				fn.ListSetCursor(fn.ListCursor)
			}
			return true, "", ""
		} else {
			// At bottom, move focus to buttons
			fn.ListFocused = false
			if fn.ListSetFocus != nil {
				fn.ListSetFocus(false)
			}
			if fn.ButtonGroup != nil {
				fn.ButtonGroup.SetFocus(true)
			}
			return true, "", ""
		}

	case " ", "enter":
		// List-specific action (e.g., toggle checkbox)
		return true, "", "toggle"

	case "left", "right", "h", "l":
		// Switch focus to buttons when left/right is pressed
		fn.ListFocused = false
		if fn.ListSetFocus != nil {
			fn.ListSetFocus(false)
		}
		if fn.ButtonGroup != nil {
			fn.ButtonGroup.SetFocus(true)
		}
		// Then handle the key for button navigation
		return fn.handleButtonNavigation(key)
	}

	// Let list handle other keys (e.g., custom navigation)
	if fn.ListNavigate != nil {
		if fn.ListNavigate(key) {
			return true, "", ""
		}
	}

	return false, "", ""
}

// handleButtonNavigation handles keys when buttons have focus
func (fn *FormNavigation) handleButtonNavigation(key string) (handled bool, action string, listAction string) {
	switch key {
	case "up", "k":
		// Move focus back to list
		fn.ListFocused = true
		if fn.ListSetFocus != nil {
			fn.ListSetFocus(true)
		}
		if fn.ButtonGroup != nil {
			fn.ButtonGroup.SetFocus(false)
		}
		return true, "", ""

	case "left", "right", "h", "l", "tab":
		// Navigate buttons
		if fn.ButtonGroup != nil {
			buttonAction := fn.ButtonGroup.HandleKey(key)
			if buttonAction != "" {
				return true, buttonAction, ""
			}
			return true, "", ""
		}

	case "enter":
		// Activate selected button
		if fn.ButtonGroup != nil {
			buttonAction := fn.ButtonGroup.HandleKey(key)
			if buttonAction != "" {
				return true, buttonAction, ""
			}
			return true, "", ""
		}
	}

	return false, "", ""
}

// SetListFocus sets whether the list has focus
func (fn *FormNavigation) SetListFocus(focused bool) {
	fn.ListFocused = focused
	if fn.ListSetFocus != nil {
		fn.ListSetFocus(focused)
	}
	if fn.ButtonGroup != nil {
		fn.ButtonGroup.SetFocus(!focused)
	}
}

// SetListCursor sets the list cursor position
func (fn *FormNavigation) SetListCursor(cursor int) {
	if cursor >= 0 && cursor < fn.ListLength {
		fn.ListCursor = cursor
		if fn.ListSetCursor != nil {
			fn.ListSetCursor(cursor)
		}
	}
}

// GetListCursor returns the current list cursor position
func (fn *FormNavigation) GetListCursor() int {
	if fn.ListGetCursor != nil {
		return fn.ListGetCursor()
	}
	return fn.ListCursor
}
