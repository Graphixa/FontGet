package components

import (
	"fmt"
	"strings"

	"fontget/internal/ui"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FormComponentType represents different types of form components
type FormComponentType int

const (
	ComponentTextInput FormComponentType = iota
	ComponentCheckboxList
	ComponentButtonGroup
	ComponentCustom // For future extensibility
)

// FormComponent represents a single component in a form
type FormComponent struct {
	Type  FormComponentType
	ID    string // Unique identifier for the component
	Label string // Optional label for text inputs

	// Component instances (only one will be set based on Type)
	TextInput    *textinput.Model
	CheckboxList *CheckboxList
	ButtonGroup  *ButtonGroup

	// Navigation
	CanReceiveFocus bool
	FocusOrder      int // Order in tab sequence

	// Validation
	Required  bool
	Validator func(value interface{}) error

	// Custom renderer (optional)
	CustomRenderer func() string
}

// UnifiedFormModel manages a complete form with mixed components
type UnifiedFormModel struct {
	Title      string
	Components []FormComponent
	FocusedIdx int // Index of currently focused component
	Error      string
	Width      int
	Height     int

	// Callbacks
	OnSubmit   func(values map[string]interface{}) error
	OnCancel   func()
	OnValidate func() error // Custom validation

	// Navigation settings
	WrapNavigation bool // Whether Tab wraps around
}

// NewUnifiedFormModel creates a new unified form model
func NewUnifiedFormModel(title string) *UnifiedFormModel {
	return &UnifiedFormModel{
		Title:          title,
		Components:     []FormComponent{},
		FocusedIdx:     0,
		Width:          80,
		Height:         24,
		WrapNavigation: true,
	}
}

// AddTextInput adds a text input component to the form
func (m *UnifiedFormModel) AddTextInput(id, label, placeholder string, required bool) {
	input := textinput.New()
	input.Placeholder = placeholder
	input.Width = 50 // Default width, will be updated on window resize
	input.TextStyle = ui.FormInput
	input.PlaceholderStyle = ui.FormPlaceholder

	comp := FormComponent{
		Type:            ComponentTextInput,
		ID:              id,
		Label:           label,
		TextInput:       &input,
		CanReceiveFocus: true,
		FocusOrder:      len(m.Components),
		Required:        required,
	}

	m.Components = append(m.Components, comp)

	// Focus first component
	if len(m.Components) == 1 {
		input.Focus()
	}
}

// AddCheckboxList adds a checkbox list component to the form
func (m *UnifiedFormModel) AddCheckboxList(id string, items []CheckboxItem) {
	checkboxList := NewCheckboxList(items)

	comp := FormComponent{
		Type:            ComponentCheckboxList,
		ID:              id,
		CheckboxList:    checkboxList,
		CanReceiveFocus: true,
		FocusOrder:      len(m.Components),
		Required:        false,
	}

	m.Components = append(m.Components, comp)
}

// AddButtonGroup adds a button group component to the form
func (m *UnifiedFormModel) AddButtonGroup(id string, buttons []string, defaultIdx int) {
	buttonGroup := NewButtonGroup(buttons, defaultIdx)

	comp := FormComponent{
		Type:            ComponentButtonGroup,
		ID:              id,
		ButtonGroup:     buttonGroup,
		CanReceiveFocus: true,
		FocusOrder:      len(m.Components),
		Required:        false,
	}

	m.Components = append(m.Components, comp)
}

// Init initializes the form model
func (m UnifiedFormModel) Init() tea.Cmd {
	// Return blink command if first component is a text input
	if len(m.Components) > 0 && m.Components[0].Type == ComponentTextInput {
		return textinput.Blink
	}
	return nil
}

// Update handles messages and updates the form
func (m *UnifiedFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle window resize
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.Width = msg.Width
		m.Height = msg.Height
		m.updateInputWidths()
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		// Handle Tab keys
		if msg.Type == tea.KeyTab {
			m.navigateForward()
			if m.FocusedIdx < len(m.Components) && m.Components[m.FocusedIdx].Type == ComponentTextInput {
				return m, textinput.Blink
			}
			return m, nil
		}

		switch key {
		case "esc":
			if m.OnCancel != nil {
				m.OnCancel()
			}
			return m, tea.Quit

		case "shift+tab":
			m.navigateBackward()
			if m.FocusedIdx < len(m.Components) && m.Components[m.FocusedIdx].Type == ComponentTextInput {
				return m, textinput.Blink
			}
			return m, nil

		case "enter":
			// Handle enter based on focused component
			if m.FocusedIdx < len(m.Components) {
				comp := &m.Components[m.FocusedIdx]
				if comp.Type == ComponentButtonGroup {
					// Button group handles enter internally
					action := comp.ButtonGroup.HandleKey(key)
					if action != "" {
						// Button was activated - could trigger submit
						if m.OnSubmit != nil {
							values := m.GetValues()
							if err := m.OnSubmit(values); err != nil {
								m.Error = err.Error()
								return m, nil
							}
							return m, tea.Quit
						}
					}
				} else if comp.Type == ComponentCheckboxList {
					// Toggle checkbox
					if comp.CheckboxList.Cursor >= 0 && comp.CheckboxList.Cursor < len(comp.CheckboxList.Items) {
						comp.CheckboxList.Items[comp.CheckboxList.Cursor].Checked = !comp.CheckboxList.Items[comp.CheckboxList.Cursor].Checked
					}
				} else {
					// Text input - validate and submit
					if m.validateForm() {
						if m.OnSubmit != nil {
							values := m.GetValues()
							if err := m.OnSubmit(values); err != nil {
								m.Error = err.Error()
								return m, nil
							}
							return m, tea.Quit
						}
					}
				}
			}

		case "ctrl+c":
			return m, tea.Quit
		}
	}

	// Update focused component
	if m.FocusedIdx < len(m.Components) {
		comp := &m.Components[m.FocusedIdx]
		switch comp.Type {
		case ComponentTextInput:
			if comp.TextInput != nil {
				var cmd tea.Cmd
				*comp.TextInput, cmd = comp.TextInput.Update(msg)
				return m, cmd
			}
		case ComponentCheckboxList:
			if comp.CheckboxList != nil {
				handled := comp.CheckboxList.HandleKey(msg.(tea.KeyMsg).String())
				if handled {
					return m, nil
				}
			}
		case ComponentButtonGroup:
			if comp.ButtonGroup != nil {
				action := comp.ButtonGroup.HandleKey(msg.(tea.KeyMsg).String())
				if action != "" {
					// Button action - could trigger submit
					if m.OnSubmit != nil {
						values := m.GetValues()
						if err := m.OnSubmit(values); err != nil {
							m.Error = err.Error()
							return m, nil
						}
						return m, tea.Quit
					}
				}
				return m, nil
			}
		}
	}

	return m, cmd
}

// navigateForward moves focus to the next component
func (m *UnifiedFormModel) navigateForward() {
	if len(m.Components) == 0 {
		return
	}
	if m.WrapNavigation {
		m.FocusedIdx = (m.FocusedIdx + 1) % len(m.Components)
	} else {
		if m.FocusedIdx < len(m.Components)-1 {
			m.FocusedIdx++
		}
	}
	m.updateFocus()
}

// navigateBackward moves focus to the previous component
func (m *UnifiedFormModel) navigateBackward() {
	if len(m.Components) == 0 {
		return
	}
	if m.WrapNavigation {
		m.FocusedIdx = (m.FocusedIdx - 1 + len(m.Components)) % len(m.Components)
	} else {
		if m.FocusedIdx > 0 {
			m.FocusedIdx--
		}
	}
	m.updateFocus()
}

// updateFocus updates which component is focused
func (m *UnifiedFormModel) updateFocus() {
	// Blur all components
	for i := range m.Components {
		m.blurComponent(i)
	}

	// Focus current component
	if m.FocusedIdx < len(m.Components) {
		m.focusComponent(m.FocusedIdx)
	}
}

// focusComponent focuses a specific component
func (m *UnifiedFormModel) focusComponent(idx int) {
	if idx >= len(m.Components) {
		return
	}

	comp := &m.Components[idx]
	if !comp.CanReceiveFocus {
		return
	}

	switch comp.Type {
	case ComponentTextInput:
		if comp.TextInput != nil {
			comp.TextInput.Focus()
		}
	case ComponentCheckboxList:
		if comp.CheckboxList != nil {
			comp.CheckboxList.HasFocus = true
		}
	case ComponentButtonGroup:
		if comp.ButtonGroup != nil {
			comp.ButtonGroup.SetFocus(true)
		}
	}
}

// blurComponent blurs a specific component
func (m *UnifiedFormModel) blurComponent(idx int) {
	if idx >= len(m.Components) {
		return
	}

	comp := &m.Components[idx]

	switch comp.Type {
	case ComponentTextInput:
		if comp.TextInput != nil {
			comp.TextInput.Blur()
		}
	case ComponentCheckboxList:
		if comp.CheckboxList != nil {
			comp.CheckboxList.HasFocus = false
		}
	case ComponentButtonGroup:
		if comp.ButtonGroup != nil {
			comp.ButtonGroup.SetFocus(false)
		}
	}
}

// View renders the form
func (m UnifiedFormModel) View() string {
	var result strings.Builder

	// Title
	if m.Title != "" {
		result.WriteString(ui.PageTitle.Render(m.Title))
		result.WriteString("\n\n")
	}

	// Render components
	for i, comp := range m.Components {
		result.WriteString(m.renderComponent(comp))
		if i < len(m.Components)-1 {
			result.WriteString("\n")
		}
	}

	// Render error if any
	if m.Error != "" {
		result.WriteString("\n")
		result.WriteString(ui.RenderError(m.Error))
		result.WriteString("\n")
	}

	// Render commands
	commands := m.getCommands()
	helpText := strings.Join(commands, "  ")
	result.WriteString("\n")
	result.WriteString(helpText)

	return result.String()
}

// renderComponent renders a single component
func (m UnifiedFormModel) renderComponent(comp FormComponent) string {
	switch comp.Type {
	case ComponentTextInput:
		return m.renderTextInput(comp)
	case ComponentCheckboxList:
		if comp.CheckboxList != nil {
			return comp.CheckboxList.Render()
		}
	case ComponentButtonGroup:
		if comp.ButtonGroup != nil {
			return comp.ButtonGroup.Render()
		}
	case ComponentCustom:
		if comp.CustomRenderer != nil {
			return comp.CustomRenderer()
		}
	}
	return ""
}

// renderTextInput renders a text input component
func (m UnifiedFormModel) renderTextInput(comp FormComponent) string {
	if comp.TextInput == nil {
		return ""
	}

	fieldValue := comp.TextInput.View()

	// Apply background styling if needed (similar to old TextInput component)
	colors := ui.GetCurrentColors()
	if colors != nil {
		width := comp.TextInput.Width
		if width == 0 {
			width = 50
		}
		contentWidth := width - 2 // Account for padding

		// Ensure the input view is at least the content width
		actualInputWidth := lipgloss.Width(fieldValue)
		if actualInputWidth < contentWidth {
			paddingNeeded := contentWidth - actualInputWidth
			padding := strings.Repeat(" ", paddingNeeded)
			paddingStyle := lipgloss.NewStyle().
				Background(lipgloss.Color(colors.Base))
			paddedInputView := fieldValue + paddingStyle.Render(padding)
			fieldValue = paddedInputView
		}

		// Create background style
		bgStyle := lipgloss.NewStyle().
			Background(lipgloss.Color(colors.Base)).
			Width(width).
			Padding(0, 1)

		fieldValue = bgStyle.Render(fieldValue)
	}

	if comp.Label != "" {
		styledLabel := ui.FormLabel.Render(comp.Label)
		return fmt.Sprintf("  %s %s %s", styledLabel, " ", fieldValue)
	}

	return fieldValue
}

// getCommands returns the command help text
func (m UnifiedFormModel) getCommands() []string {
	return []string{
		ui.RenderKeyWithDescription("Tab/Shift+Tab", "Move"),
		ui.RenderKeyWithDescription("Enter", "Submit/Select"),
		ui.RenderKeyWithDescription("Esc", "Cancel"),
	}
}

// updateInputWidths updates the width of text inputs based on terminal size
func (m *UnifiedFormModel) updateInputWidths() {
	width := m.calculateInputWidth()
	for i := range m.Components {
		if m.Components[i].Type == ComponentTextInput && m.Components[i].TextInput != nil {
			m.Components[i].TextInput.Width = width
		}
	}
}

// calculateInputWidth calculates the appropriate input width based on terminal size
func (m UnifiedFormModel) calculateInputWidth() int {
	width := m.Width - 20 // Account for margins and labels
	if width < 30 {
		width = 30
	}
	if width > 80 {
		width = 80
	}
	return width
}

// validateForm validates the form inputs
func (m *UnifiedFormModel) validateForm() bool {
	// Custom validation
	if m.OnValidate != nil {
		if err := m.OnValidate(); err != nil {
			m.Error = err.Error()
			return false
		}
	}

	// Component-level validation
	for i, comp := range m.Components {
		if comp.Required {
			var value interface{}
			switch comp.Type {
			case ComponentTextInput:
				if comp.TextInput != nil {
					value = strings.TrimSpace(comp.TextInput.Value())
				}
			case ComponentCheckboxList:
				if comp.CheckboxList != nil {
					// Check if at least one is selected
					hasSelected := false
					for _, item := range comp.CheckboxList.Items {
						if item.Checked {
							hasSelected = true
							break
						}
					}
					value = hasSelected
				}
			}

			// Check if required field is empty
			if value == nil || value == "" || value == false {
				m.Error = fmt.Sprintf("%s is required", comp.Label)
				m.FocusedIdx = i
				m.updateFocus()
				return false
			}
		}

		// Custom validator
		if comp.Validator != nil {
			var value interface{}
			switch comp.Type {
			case ComponentTextInput:
				if comp.TextInput != nil {
					value = comp.TextInput.Value()
				}
			case ComponentCheckboxList:
				if comp.CheckboxList != nil {
					value = comp.CheckboxList
				}
			case ComponentButtonGroup:
				if comp.ButtonGroup != nil {
					value = comp.ButtonGroup
				}
			}

			if err := comp.Validator(value); err != nil {
				m.Error = err.Error()
				m.FocusedIdx = i
				m.updateFocus()
				return false
			}
		}
	}

	m.Error = ""
	return true
}

// GetValues returns a map of component values
func (m UnifiedFormModel) GetValues() map[string]interface{} {
	values := make(map[string]interface{})

	for _, comp := range m.Components {
		switch comp.Type {
		case ComponentTextInput:
			if comp.TextInput != nil {
				values[comp.ID] = strings.TrimSpace(comp.TextInput.Value())
			}
		case ComponentCheckboxList:
			if comp.CheckboxList != nil {
				// Return selected items
				selected := []int{}
				for i, item := range comp.CheckboxList.Items {
					if item.Checked {
						selected = append(selected, i)
					}
				}
				values[comp.ID] = selected
			}
		case ComponentButtonGroup:
			if comp.ButtonGroup != nil {
				values[comp.ID] = comp.ButtonGroup.Selected
			}
		}
	}

	return values
}

// SetValue sets a value for a component by ID
func (m *UnifiedFormModel) SetValue(id string, value interface{}) {
	for i := range m.Components {
		if m.Components[i].ID == id {
			switch m.Components[i].Type {
			case ComponentTextInput:
				if m.Components[i].TextInput != nil {
					if str, ok := value.(string); ok {
						m.Components[i].TextInput.SetValue(str)
					}
				}
			}
			break
		}
	}
}

// GetTextInputValue gets a text input value by ID
func (m UnifiedFormModel) GetTextInputValue(id string) string {
	for _, comp := range m.Components {
		if comp.ID == id && comp.Type == ComponentTextInput && comp.TextInput != nil {
			return comp.TextInput.Value()
		}
	}
	return ""
}

// GetCheckboxListSelected gets selected indices from a checkbox list by ID
func (m UnifiedFormModel) GetCheckboxListSelected(id string) []int {
	for _, comp := range m.Components {
		if comp.ID == id && comp.Type == ComponentCheckboxList && comp.CheckboxList != nil {
			selected := []int{}
			for i, item := range comp.CheckboxList.Items {
				if item.Checked {
					selected = append(selected, i)
				}
			}
			return selected
		}
	}
	return []int{}
}

// SetError sets the error message
func (m *UnifiedFormModel) SetError(err string) {
	m.Error = err
}

// Helper functions for common component creation

// NewTextInputComponent creates a FormComponent for a text input
func NewTextInputComponent(id, label, placeholder string, required bool) FormComponent {
	input := textinput.New()
	input.Placeholder = placeholder
	input.Width = 50
	input.TextStyle = ui.FormInput
	input.PlaceholderStyle = ui.FormPlaceholder

	return FormComponent{
		Type:            ComponentTextInput,
		ID:              id,
		Label:           label,
		TextInput:       &input,
		CanReceiveFocus: true,
		Required:        required,
	}
}

// NewCheckboxListComponent creates a FormComponent for a checkbox list
func NewCheckboxListComponent(id string, items []CheckboxItem) FormComponent {
	checkboxList := NewCheckboxList(items)

	return FormComponent{
		Type:            ComponentCheckboxList,
		ID:              id,
		CheckboxList:    checkboxList,
		CanReceiveFocus: true,
		Required:        false,
	}
}

// NewButtonGroupComponent creates a FormComponent for a button group
func NewButtonGroupComponent(id string, buttons []string, defaultIdx int) FormComponent {
	buttonGroup := NewButtonGroup(buttons, defaultIdx)

	return FormComponent{
		Type:            ComponentButtonGroup,
		ID:              id,
		ButtonGroup:     buttonGroup,
		CanReceiveFocus: true,
		Required:        false,
	}
}
