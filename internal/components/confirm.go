package components

import (
	"fmt"
	"strings"

	"fontget/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

// ConfirmModel represents a confirmation dialog
type ConfirmModel struct {
	Title       string
	Message     string
	ConfirmText string
	CancelText  string
	Confirmed   bool
	Quit        bool
	Width       int
	Height      int
	buttons     *ButtonGroup
}

// NewConfirmModel creates a new confirmation dialog
func NewConfirmModel(title, message string) *ConfirmModel {
	return &ConfirmModel{
		Title:       title,
		Message:     message,
		ConfirmText: "Yes",
		CancelText:  "No",
		Width:       80,
		Height:      24,
	}
}

// Init initializes the confirmation dialog
func (m *ConfirmModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the confirmation dialog
func (m *ConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Initialize buttons if needed
	if m.buttons == nil {
		m.buttons = NewButtonGroup([]string{m.ConfirmText, m.CancelText}, 0)
		m.buttons.SetFocus(true)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		// Handle button navigation
		action := m.buttons.HandleKey(key)
		if action != "" {
			switch strings.ToLower(action) {
			case strings.ToLower(m.ConfirmText), "yes", "save", "accept":
				m.Confirmed = true
				return m, tea.Quit
			case strings.ToLower(m.CancelText), "no", "discard", "cancel":
				m.Confirmed = false
				return m, tea.Quit
			}
		}

		// Fallback for direct key presses (backward compatibility)
		switch key {
		case "y", "Y":
			m.Confirmed = true
			return m, tea.Quit
		case "n", "N", "esc":
			m.Confirmed = false
			return m, tea.Quit
		case "ctrl+c":
			m.Confirmed = false
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil
	}

	return m, nil
}

// View renders the confirmation dialog
func (m *ConfirmModel) View() string {
	var result strings.Builder

	// Initialize buttons if needed
	if m.buttons == nil {
		m.buttons = NewButtonGroup([]string{m.ConfirmText, m.CancelText}, 0)
		m.buttons.SetFocus(true)
	}

	// Title
	if m.Title != "" {
		result.WriteString(ui.PageTitle.Render(m.Title))
		result.WriteString("\n\n")
	}

	// Message
	result.WriteString(ui.Text.Render(m.Message))
	result.WriteString("\n\n")

	// Render button group
	if m.buttons != nil {
		result.WriteString(m.buttons.Render())
		result.WriteString("\n")
	}

	// Keyboard help
	commands := []string{
		ui.RenderKeyWithDescription("←/→", "Navigate"),
		ui.RenderKeyWithDescription("Enter", "Select"),
	}
	helpText := strings.Join(commands, "  ")
	result.WriteString("\n")
	result.WriteString(helpText)

	return result.String()
}

// RunConfirm runs a confirmation dialog
func RunConfirm(title, message string) (bool, error) {
	model := NewConfirmModel(title, message)

	// Create and run the Bubble Tea program
	program := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := program.Run()
	if err != nil {
		return false, fmt.Errorf("failed to run confirmation dialog: %w", err)
	}

	// Check if the user confirmed
	if m, ok := finalModel.(*ConfirmModel); ok {
		return m.Confirmed, nil
	}

	return false, nil
}

// RunConfirmWithOptions runs a confirmation dialog with custom options
func RunConfirmWithOptions(title, message, confirmText, cancelText string) (bool, error) {
	model := NewConfirmModel(title, message)
	model.ConfirmText = confirmText
	model.CancelText = cancelText

	// Create and run the Bubble Tea program
	program := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := program.Run()
	if err != nil {
		return false, fmt.Errorf("failed to run confirmation dialog: %w", err)
	}

	// Check if the user confirmed
	if m, ok := finalModel.(*ConfirmModel); ok {
		return m.Confirmed, nil
	}

	return false, nil
}

// DeleteConfirm runs a delete confirmation dialog
func DeleteConfirm(itemName string) (bool, error) {
	title := "Confirm Deletion"
	message := fmt.Sprintf("Are you sure you want to delete '%s'?", ui.TableSourceName.Render(itemName))
	return RunConfirm(title, message)
}

// SaveConfirm runs a save confirmation dialog
func SaveConfirm() (bool, error) {
	title := "Save Changes"
	message := "You have unsaved changes. Do you want to save before exiting?"
	return RunConfirmWithOptions(title, message, "Save", "Discard")
}

// WarningConfirm runs a warning confirmation dialog
func WarningConfirm(title, message string) (bool, error) {
	return RunConfirm(title, message)
}
