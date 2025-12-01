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
func (m ConfirmModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the confirmation dialog
func (m ConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y", "enter":
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
func (m ConfirmModel) View() string {
	var result strings.Builder

	// Title
	if m.Title != "" {
		result.WriteString(ui.PageTitle.Render(m.Title))
		result.WriteString("\n\n")
	}

	// Message
	result.WriteString(ui.Text.Render(m.Message))
	result.WriteString("\n\n")

	// Confirmation prompt - match sources_manage.go styling
	commands := []string{
		ui.RenderKeyWithDescription("Y", m.ConfirmText),
		ui.RenderKeyWithDescription("N", m.CancelText),
	}
	helpText := strings.Join(commands, "  ")
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
	if m, ok := finalModel.(ConfirmModel); ok {
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
	if m, ok := finalModel.(ConfirmModel); ok {
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
