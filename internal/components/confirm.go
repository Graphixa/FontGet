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
	// Initialize buttons in Init to avoid duplicate initialization
	if m.buttons == nil {
		m.buttons = NewButtonGroup([]string{m.ConfirmText, m.CancelText}, 0)
		m.buttons.SetFocus(true)
	}
	return nil
}

// Update handles messages and updates the confirmation dialog
func (m *ConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Ensure buttons are initialized (defensive check)
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
				m.Quit = true
				return m, nil // Don't quit here - let parent modal handle it
			case strings.ToLower(m.CancelText), "no", "discard", "cancel":
				m.Confirmed = false
				m.Quit = true
				return m, nil // Don't quit here - let parent modal handle it
			}
		}

		// Fallback for direct key presses (backward compatibility)
		switch key {
		case "y", "Y":
			m.Confirmed = true
			m.Quit = true
			return m, nil // Don't quit here - let parent modal handle it
		case "n", "N", "esc":
			m.Confirmed = false
			m.Quit = true
			return m, nil // Don't quit here - let parent modal handle it
		case "ctrl+c":
			m.Confirmed = false
			m.Quit = true
			return m, nil // Don't quit here - let parent modal handle it
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

// GetResult returns the modal result
func (m *ConfirmModel) GetResult() ModalResult {
	return ModalResult{
		Confirmed: m.Confirmed,
		Cancelled: !m.Confirmed,
		Data:      nil,
	}
}

// RunConfirm runs a confirmation dialog
func RunConfirm(title, message string) (bool, error) {
	model := NewConfirmModel(title, message)
	model.ConfirmText = "Yes"
	model.CancelText = "No"

	overlay := NewOverlayWithOptions(model, &BlankBackgroundModel{}, Center, Center, 0, 0, OverlayOptions{
		ShowBorder:  true,
		BorderWidth: 0,
	})

	finalModel, err := tea.NewProgram(overlay, tea.WithAltScreen()).Run()
	if err != nil {
		return false, fmt.Errorf("failed to run confirmation dialog: %w", err)
	}

	if overlayModel, ok := finalModel.(*OverlayModel); ok {
		if confirmModel, ok := overlayModel.Foreground.(*ConfirmModel); ok {
			return confirmModel.Confirmed, nil
		}
	}

	return false, nil
}
