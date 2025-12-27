package components

import (
	"fmt"
	"strings"

	"fontget/internal/ui"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// TextPromptModel is a simple inline text input prompt
type TextPromptModel struct {
	promptText string
	textInput  textinput.Model
	value      string
	err        error
	quitting   bool
	confirmed  bool
}

// NewTextPrompt creates a new inline text prompt
func NewTextPrompt(promptText, placeholder string, width int) *TextPromptModel {
	if width <= 0 {
		width = 60
	}

	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Focus()
	ti.Width = width
	ti.TextStyle = ui.FormInput
	ti.PlaceholderStyle = ui.FormPlaceholder

	return &TextPromptModel{
		promptText: promptText,
		textInput:  ti,
		value:      "",
	}
}

func (m TextPromptModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *TextPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			m.value = strings.TrimSpace(m.textInput.Value())
			if m.value == "" {
				// Use placeholder if empty
				m.value = m.textInput.Placeholder
			}
			m.confirmed = true
			m.quitting = true
			return m, tea.Quit
		case tea.KeyEsc, tea.KeyCtrlC:
			m.confirmed = false
			m.quitting = true
			return m, tea.Quit
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m TextPromptModel) View() string {
	if m.quitting {
		return ""
	}
	prompt := m.promptText
	if prompt == "" {
		prompt = "Enter value:"
	}
	return fmt.Sprintf(
		"%s\n%s\n%s",
		ui.Text.Render(prompt),
		m.textInput.View(),
		ui.Text.Render("(Enter to confirm, Esc to cancel)"),
	)
}

// Value returns the entered value
func (m *TextPromptModel) Value() string {
	return m.value
}

// Confirmed returns whether the prompt was confirmed (Enter pressed)
func (m *TextPromptModel) Confirmed() bool {
	return m.confirmed
}

// RunTextPrompt runs a simple inline text prompt and returns the value
func RunTextPrompt(promptText, placeholder string, width int) (string, bool, error) {
	model := NewTextPrompt(promptText, placeholder, width)
	program := tea.NewProgram(model)

	finalModel, err := program.Run()
	if err != nil {
		return "", false, fmt.Errorf("failed to run text prompt: %w", err)
	}

	if promptModel, ok := finalModel.(*TextPromptModel); ok {
		if promptModel.Confirmed() {
			return promptModel.Value(), true, nil
		}
		return "", false, nil
	}

	return "", false, nil
}

// CheckboxPromptModel is a simple inline checkbox selection prompt
type CheckboxPromptModel struct {
	title        string
	items        []CheckboxItem
	checkboxList *CheckboxList
	quitting     bool
	confirmed    bool
}

// NewCheckboxPrompt creates a new inline checkbox prompt
func NewCheckboxPrompt(title string, items []CheckboxItem) *CheckboxPromptModel {
	checkboxList := NewCheckboxList(items)
	checkboxList.SetFocus(true)

	return &CheckboxPromptModel{
		title:        title,
		items:        items,
		checkboxList: checkboxList,
		quitting:     false,
		confirmed:    false,
	}
}

func (m CheckboxPromptModel) Init() tea.Cmd {
	return nil
}

func (m *CheckboxPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		switch key {
		case "enter":
			// Confirm selection
			m.confirmed = true
			m.quitting = true
			return m, tea.Quit
		case "esc", "ctrl+c":
			// Cancel
			m.confirmed = false
			m.quitting = true
			return m, tea.Quit
		}

		// Handle checkbox navigation
		if m.checkboxList != nil {
			m.checkboxList.HandleKey(key)
		}
	}

	return m, nil
}

func (m CheckboxPromptModel) View() string {
	if m.quitting {
		return ""
	}

	var result strings.Builder
	if m.title != "" {
		result.WriteString(ui.Text.Render(m.title))
		result.WriteString("\n")
	}
	result.WriteString("\n")

	if m.checkboxList != nil {
		result.WriteString(m.checkboxList.Render())
	}

	result.WriteString("\n")
	commands := []string{
		ui.RenderKeyWithDescription("↑/↓", "Navigate"),
		ui.RenderKeyWithDescription("Space", "Toggle"),
		ui.RenderKeyWithDescription("Enter", "Confirm"),
		ui.RenderKeyWithDescription("Esc", "Cancel"),
	}
	helpText := strings.Join(commands, "  ")
	result.WriteString(helpText)

	return result.String()
}

// GetSelectedIndices returns the indices of selected items
func (m *CheckboxPromptModel) GetSelectedIndices() []int {
	selected := []int{}
	if m.checkboxList != nil {
		for i, item := range m.checkboxList.Items {
			if item.Checked {
				selected = append(selected, i)
			}
		}
	}
	return selected
}

// Confirmed returns whether the prompt was confirmed
func (m *CheckboxPromptModel) Confirmed() bool {
	return m.confirmed
}

// RunCheckboxPrompt runs a simple inline checkbox prompt and returns selected indices
func RunCheckboxPrompt(title string, items []CheckboxItem) ([]int, bool, error) {
	model := NewCheckboxPrompt(title, items)
	program := tea.NewProgram(model)

	finalModel, err := program.Run()
	if err != nil {
		return nil, false, fmt.Errorf("failed to run checkbox prompt: %w", err)
	}

	if promptModel, ok := finalModel.(*CheckboxPromptModel); ok {
		if promptModel.Confirmed() {
			selected := promptModel.GetSelectedIndices()
			if len(selected) > 0 {
				return selected, true, nil
			}
		}
		return nil, false, nil
	}

	return nil, false, nil
}
