package components

import (
	"fmt"
	"strings"

	"fontget/internal/ui"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// FormField represents a single form field
type FormField struct {
	Label       string
	Value       string
	Placeholder string
	Focused     bool
	ReadOnly    bool
	Input       textinput.Model
}

// FormModel represents a form component
type FormModel struct {
	Title        string
	Fields       []FormField
	FocusedField int
	Error        string
	ReadOnly     bool
	Width        int
	Height       int
	OnSubmit     func(values map[string]string) error
	OnCancel     func()
}

// NewFormModel creates a new form model
func NewFormModel(title string, fieldConfigs []FieldConfig) *FormModel {
	fields := make([]FormField, len(fieldConfigs))

	for i, config := range fieldConfigs {
		input := textinput.New()
		input.Placeholder = config.Placeholder
		input.Width = 50 // Default width, will be updated on window resize
		input.TextStyle = ui.FormInput
		input.PlaceholderStyle = ui.FormPlaceholder

		fields[i] = FormField{
			Label:       config.Label,
			Value:       config.Value,
			Placeholder: config.Placeholder,
			Focused:     i == 0, // First field is focused by default
			ReadOnly:    config.ReadOnly,
			Input:       input,
		}
	}

	// Focus the first field
	if len(fields) > 0 {
		fields[0].Input.Focus()
	}

	return &FormModel{
		Title:        title,
		Fields:       fields,
		FocusedField: 0,
		Width:        80,
		Height:       24,
	}
}

// FieldConfig represents configuration for a form field
type FieldConfig struct {
	Label       string
	Value       string
	Placeholder string
	ReadOnly    bool
}

// Init initializes the form model
func (m FormModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages and updates the form
func (m FormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		switch msg.String() {
		case "esc":
			if m.OnCancel != nil {
				m.OnCancel()
			}
			return m, tea.Quit

		case "tab":
			m.FocusedField = (m.FocusedField + 1) % len(m.Fields)
			m.updateFocus()

		case "shift+tab":
			m.FocusedField = (m.FocusedField - 1 + len(m.Fields)) % len(m.Fields)
			m.updateFocus()

		case "enter":
			if m.ReadOnly {
				// In read-only mode, just quit
				return m, tea.Quit
			} else if m.validateForm() {
				values := m.getValues()
				if m.OnSubmit != nil {
					if err := m.OnSubmit(values); err != nil {
						m.Error = err.Error()
						return m, nil
					}
				}
				return m, tea.Quit
			}

		case "ctrl+c":
			return m, tea.Quit
		}
	}

	// Update focused input (only if not in read-only mode)
	if !m.ReadOnly && m.FocusedField < len(m.Fields) {
		m.Fields[m.FocusedField].Input, cmd = m.Fields[m.FocusedField].Input.Update(msg)
	}

	return m, cmd
}

// View renders the form
func (m FormModel) View() string {
	out := ui.PageTitle.Render(m.Title) + "\n\n"

	// Render fields
	for i, field := range m.Fields {
		fieldValue := m.renderFieldValue(field)
		styledLabel := ui.FormLabel.Render(field.Label)
		out += fmt.Sprintf("  %s %s\n", styledLabel, fieldValue)
		if i < len(m.Fields)-1 {
			out += "\n"
		}
	}

	// Render error if any
	if m.Error != "" {
		out += "\n" + ui.RenderError(m.Error) + "\n"
	}

	// Render commands
	commands := m.getCommands()
	helpText := strings.Join(commands, "  ")
	out += "\n" + helpText

	return out
}

// renderFieldValue renders the value for a field
func (m FormModel) renderFieldValue(field FormField) string {
	if field.ReadOnly {
		// In read-only mode, show as static text
		return ui.FormReadOnly.Render(field.Value)
	}

	// In edit mode, show as input field with custom styling
	if field.Focused {
		// For the focused field, use the textinput's View() method to get the blinking cursor
		return field.Input.View()
	} else {
		// For non-focused fields, show the value with custom styling
		inputValue := field.Input.Value()
		if inputValue == "" {
			// Show placeholder with placeholder styling
			return ui.FormPlaceholder.Render(field.Input.Placeholder)
		} else {
			// Show actual input value with form input styling
			return ui.FormInput.Render(inputValue)
		}
	}
}

// getCommands returns the command help text
func (m FormModel) getCommands() []string {
	if m.ReadOnly {
		return []string{
			ui.RenderKeyWithDescription("Tab/Shift+Tab", "Move"),
			ui.RenderKeyWithDescription("Enter/Esc", "Back"),
		}
	}

	return []string{
		ui.RenderKeyWithDescription("Tab/Shift+Tab", "Move"),
		ui.RenderKeyWithDescription("Enter", "Submit"),
		ui.RenderKeyWithDescription("Esc", "Cancel"),
	}
}

// updateInputWidths updates the width of text inputs based on terminal size
func (m *FormModel) updateInputWidths() {
	width := m.calculateInputWidth()
	for i := range m.Fields {
		m.Fields[i].Input.Width = width
	}
}

// calculateInputWidth calculates the appropriate input width based on terminal size
func (m FormModel) calculateInputWidth() int {
	// Use a reasonable default width, but not too wide
	width := m.Width - 20 // Account for margins and labels
	if width < 30 {
		width = 30
	}
	if width > 80 {
		width = 80
	}
	return width
}

// updateFocus updates which input is focused
func (m *FormModel) updateFocus() {
	// Blur all inputs
	for i := range m.Fields {
		m.Fields[i].Input.Blur()
		m.Fields[i].Focused = false
	}

	// Focus the current field
	if m.FocusedField < len(m.Fields) && !m.ReadOnly {
		m.Fields[m.FocusedField].Input.Focus()
		m.Fields[m.FocusedField].Focused = true
	}
}

// validateForm validates the form inputs
func (m *FormModel) validateForm() bool {
	// Basic validation - check that required fields are not empty
	for i, field := range m.Fields {
		if !field.ReadOnly {
			value := strings.TrimSpace(field.Input.Value())
			if value == "" {
				m.Error = fmt.Sprintf("%s is required", field.Label)
				m.FocusedField = i
				m.updateFocus()
				return false
			}
		}
	}

	m.Error = ""
	return true
}

// getValues returns a map of field values
func (m FormModel) getValues() map[string]string {
	values := make(map[string]string)
	for _, field := range m.Fields {
		values[field.Label] = strings.TrimSpace(field.Input.Value())
	}
	return values
}

// SetValues sets the values for the form fields
func (m *FormModel) SetValues(values map[string]string) {
	for i, field := range m.Fields {
		if value, exists := values[field.Label]; exists {
			m.Fields[i].Input.SetValue(value)
		}
	}
}

// SetError sets the error message
func (m *FormModel) SetError(err string) {
	m.Error = err
}

// SetReadOnly sets the read-only mode for the form
func (m *FormModel) SetReadOnly(readOnly bool) {
	m.ReadOnly = readOnly
	for i := range m.Fields {
		m.Fields[i].ReadOnly = readOnly
	}
}

// RunForm runs a form with the given configuration
func RunForm(title string, fieldConfigs []FieldConfig, onSubmit func(values map[string]string) error, onCancel func()) error {
	model := NewFormModel(title, fieldConfigs)
	model.OnSubmit = onSubmit
	model.OnCancel = onCancel

	// Create and run the Bubble Tea program
	program := tea.NewProgram(model, tea.WithAltScreen())

	_, err := program.Run()
	if err != nil {
		return fmt.Errorf("failed to run form: %w", err)
	}

	return nil
}
