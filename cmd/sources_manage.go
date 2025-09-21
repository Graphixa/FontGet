package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"fontget/internal/config"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// SourceItem represents a source in the TUI
type SourceItem struct {
	Name      string
	Prefix    string
	URL       string
	Enabled   bool
	IsBuiltIn bool
}

// sourcesModel represents the main model for the sources management TUI
type sourcesModel struct {
	sources      []SourceItem
	cursor       int
	config       *config.SourcesConfig
	state        string // "list", "add", "edit", "confirm", "save_confirm", "builtin_warning"
	nameInput    textinput.Model
	urlInput     textinput.Model
	prefixInput  textinput.Model
	focusedField int
	err          string
	editingIndex int
}

// Styles for better visual appeal
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170")).
			MarginBottom(1)

	commandStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212"))

	keyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true)
)

// NewSourcesModel creates a new sources management model
func NewSourcesModel() (*sourcesModel, error) {
	sourcesConfig, err := config.LoadSourcesConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load sources config: %w", err)
	}

	sm := &sourcesModel{
		config:       sourcesConfig,
		state:        "list",
		focusedField: 0,
	}

	// Convert config to SourceItem slice
	for name, source := range sourcesConfig.Sources {
		sm.sources = append(sm.sources, SourceItem{
			Name:      name,
			Prefix:    source.Prefix,
			URL:       source.Path,
			Enabled:   source.Enabled,
			IsBuiltIn: isBuiltInSource(name),
		})
	}

	// Sort by name for consistent display
	sort.Slice(sm.sources, func(i, j int) bool {
		return sm.sources[i].Name < sm.sources[j].Name
	})

	// Initialize text inputs
	sm.nameInput = textinput.New()
	sm.nameInput.Placeholder = "Source name"
	sm.nameInput.Focus()

	sm.urlInput = textinput.New()
	sm.urlInput.Placeholder = "https://example.com/fonts.json"

	sm.prefixInput = textinput.New()
	sm.prefixInput.Placeholder = "prefix"

	return sm, nil
}

// isBuiltInSource checks if a source name is a built-in source
func isBuiltInSource(name string) bool {
	return config.IsBuiltInSource(name)
}

// Init initializes the model
func (m sourcesModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages and updates the model
func (m sourcesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.state {
	case "list":
		return m.updateList(msg)
	case "add", "edit":
		return m.updateForm(msg)
	case "confirm":
		return m.updateConfirm(msg)
	case "save_confirm":
		return m.updateSaveConfirm(msg)
	case "builtin_warning":
		return m.updateBuiltinWarning(msg)
	}

	return m, cmd
}

// updateList handles updates in list state
func (m sourcesModel) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.sources)-1 {
				m.cursor++
			}

		case " ", "enter":
			if len(m.sources) > 0 {
				m.sources[m.cursor].Enabled = !m.sources[m.cursor].Enabled
			}

		case "a":
			m.state = "add"
			m.focusedField = 0
			m.nameInput.Focus()
			m.urlInput.Blur()
			m.prefixInput.Blur()
			m.err = ""

		case "e":
			if len(m.sources) > 0 {
				if m.sources[m.cursor].IsBuiltIn {
					m.state = "builtin_warning"
					m.err = "Built-in sources cannot be edited. You can only enable/disable them."
				} else {
					m.state = "edit"
					m.editingIndex = m.cursor
					m.focusedField = 0
					m.nameInput.SetValue(m.sources[m.cursor].Name)
					m.urlInput.SetValue(m.sources[m.cursor].URL)
					m.prefixInput.SetValue(m.sources[m.cursor].Prefix)
					m.nameInput.Focus()
					m.urlInput.Blur()
					m.prefixInput.Blur()
					m.err = ""
				}
			}

		case "f":
			if len(m.sources) > 0 {
				// Open file manager for editing source file
				return m, m.openFileManager()
			}

		case "d":
			if len(m.sources) > 0 {
				if m.sources[m.cursor].IsBuiltIn {
					m.state = "builtin_warning"
					m.err = "Built-in sources cannot be deleted. You can only enable/disable them."
				} else {
					m.state = "confirm"
				}
			}

		case "s":
			m.state = "save_confirm"

		case "esc":
			m.err = ""
		}

	case errorMsg:
		m.err = msg.text
		return m, nil

	case successMsg:
		m.err = ""
		// Could show success message briefly
		return m, nil
	}

	return m, nil
}

// updateForm handles updates in form state
func (m sourcesModel) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.state = "list"
			m.err = ""
			m.resetForm()

		case "tab":
			m.focusedField = (m.focusedField + 1) % 3
			m.updateFocus()

		case "shift+tab":
			m.focusedField = (m.focusedField - 1 + 3) % 3
			m.updateFocus()

		case "enter":
			if m.validateForm() {
				if m.state == "add" {
					m.addSource()
				} else {
					m.updateSource()
				}
				m.state = "list"
				m.resetForm()
			}

		case "ctrl+c":
			return m, tea.Quit
		}
	}

	// Update focused input
	switch m.focusedField {
	case 0:
		m.nameInput, cmd = m.nameInput.Update(msg)
	case 1:
		m.urlInput, cmd = m.urlInput.Update(msg)
	case 2:
		m.prefixInput, cmd = m.prefixInput.Update(msg)
	}

	return m, cmd
}

// updateConfirm handles updates in confirm state
func (m sourcesModel) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			if len(m.sources) > 0 {
				m.sources = append(m.sources[:m.cursor], m.sources[m.cursor+1:]...)
				if m.cursor >= len(m.sources) {
					m.cursor = len(m.sources) - 1
				}
			}
			m.state = "list"

		case "n", "N", "esc":
			m.state = "list"

		case "ctrl+c":
			return m, tea.Quit
		}
	}

	return m, nil
}

// updateSaveConfirm handles updates in save confirmation state
func (m sourcesModel) updateSaveConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			return m, m.saveChanges()
		case "n", "N", "esc":
			m.state = "list"
			m.err = ""
		case "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

// updateBuiltinWarning handles updates in built-in warning state
func (m sourcesModel) updateBuiltinWarning(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "esc", " ":
			m.state = "list"
			m.err = ""
		case "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

// updateFocus updates which input is focused
func (m *sourcesModel) updateFocus() {
	m.nameInput.Blur()
	m.urlInput.Blur()
	m.prefixInput.Blur()

	switch m.focusedField {
	case 0:
		m.nameInput.Focus()
	case 1:
		m.urlInput.Focus()
	case 2:
		m.prefixInput.Focus()
	}
}

// resetForm resets the form inputs
func (m *sourcesModel) resetForm() {
	m.nameInput.SetValue("")
	m.urlInput.SetValue("")
	m.prefixInput.SetValue("")
	m.focusedField = 0
	m.err = ""
}

// validateForm validates the form inputs
func (m *sourcesModel) validateForm() bool {
	name := strings.TrimSpace(m.nameInput.Value())
	url := strings.TrimSpace(m.urlInput.Value())
	prefix := strings.TrimSpace(m.prefixInput.Value())

	if name == "" {
		m.err = "Name is required"
		return false
	}

	if url == "" {
		m.err = "URL is required"
		return false
	}

	// Check for duplicate names (except when editing the same source)
	for i, source := range m.sources {
		if source.Name == name && (m.state != "edit" || i != m.editingIndex) {
			m.err = "Source with this name already exists"
			return false
		}
	}

	if prefix == "" {
		m.prefixInput.SetValue(strings.ToLower(name))
	}

	return true
}

// addSource adds a new source
func (m *sourcesModel) addSource() {
	name := strings.TrimSpace(m.nameInput.Value())
	url := strings.TrimSpace(m.urlInput.Value())
	prefix := strings.TrimSpace(m.prefixInput.Value())

	if prefix == "" {
		prefix = strings.ToLower(name)
	}

	newSource := SourceItem{
		Name:      name,
		Prefix:    prefix,
		URL:       url,
		Enabled:   true,
		IsBuiltIn: false,
	}

	m.sources = append(m.sources, newSource)
	sort.Slice(m.sources, func(i, j int) bool {
		return m.sources[i].Name < m.sources[j].Name
	})

	// Find the new source's position
	for i, source := range m.sources {
		if source.Name == name {
			m.cursor = i
			break
		}
	}
}

// updateSource updates an existing source
func (m *sourcesModel) updateSource() {
	name := strings.TrimSpace(m.nameInput.Value())
	url := strings.TrimSpace(m.urlInput.Value())
	prefix := strings.TrimSpace(m.prefixInput.Value())

	if prefix == "" {
		prefix = strings.ToLower(name)
	}

	m.sources[m.editingIndex].Name = name
	m.sources[m.editingIndex].URL = url
	m.sources[m.editingIndex].Prefix = prefix

	// Re-sort sources
	sort.Slice(m.sources, func(i, j int) bool {
		return m.sources[i].Name < m.sources[j].Name
	})

	// Find the updated source's position
	for i, source := range m.sources {
		if source.Name == name {
			m.cursor = i
			break
		}
	}
}

// saveChanges saves the configuration
func (m sourcesModel) saveChanges() tea.Cmd {
	return func() tea.Msg {
		// Update the config with current sources
		m.config.Sources = make(map[string]config.Source)

		for _, source := range m.sources {
			m.config.Sources[source.Name] = config.Source{
				Path:    source.URL,
				Prefix:  source.Prefix,
				Enabled: source.Enabled,
			}
		}

		// Save the configuration
		if err := config.SaveSourcesConfig(m.config); err != nil {
			return errorMsg{err.Error()}
		}

		return tea.Quit()
	}
}

// openFileManager opens the source file in the default editor
func (m sourcesModel) openFileManager() tea.Cmd {
	return func() tea.Msg {
		if len(m.sources) == 0 {
			return errorMsg{"No sources available"}
		}

		source := m.sources[m.cursor]

		// Get the sources directory
		home, err := os.UserHomeDir()
		if err != nil {
			return errorMsg{fmt.Sprintf("Failed to get home directory: %v", err)}
		}

		sourcesDir := filepath.Join(home, ".fontget", "sources")
		sourceFile := filepath.Join(sourcesDir, source.Prefix+".json")

		// Check if file exists, if not create it
		if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
			// Create the directory if it doesn't exist
			if err := os.MkdirAll(sourcesDir, 0755); err != nil {
				return errorMsg{fmt.Sprintf("Failed to create sources directory: %v", err)}
			}

			// Create a basic source file
			basicContent := fmt.Sprintf(`{
  "source_info": {
    "name": "%s",
    "description": "Font source for %s",
    "url": "%s",
    "version": "1.0.0",
    "last_updated": "2024-01-01T00:00:00Z",
    "total_fonts": 0
  },
  "fonts": {}
}`, source.Name, source.Name, source.URL)

			if err := os.WriteFile(sourceFile, []byte(basicContent), 0644); err != nil {
				return errorMsg{fmt.Sprintf("Failed to create source file: %v", err)}
			}
		}

		// Try to open with default editor
		var cmd *exec.Cmd
		if os.Getenv("EDITOR") != "" {
			cmd = exec.Command(os.Getenv("EDITOR"), sourceFile)
		} else {
			// Try common editors
			editors := []string{"notepad.exe", "code", "vim", "nano", "emacs"}
			for _, editor := range editors {
				if _, err := exec.LookPath(editor); err == nil {
					cmd = exec.Command(editor, sourceFile)
					break
				}
			}
		}

		if cmd == nil {
			return errorMsg{"No editor found. Please set EDITOR environment variable or install a text editor."}
		}

		// Run the editor
		if err := cmd.Run(); err != nil {
			return errorMsg{fmt.Sprintf("Failed to open editor: %v", err)}
		}

		return successMsg{fmt.Sprintf("Opened %s for editing", sourceFile)}
	}
}

// errorMsg represents an error message
type errorMsg struct {
	text string
}

// successMsg represents a success message
type successMsg struct {
	text string
}

// View renders the UI
func (m sourcesModel) View() string {
	switch m.state {
	case "list":
		return m.listView()
	case "add", "edit":
		return m.formView()
	case "confirm":
		return m.confirmView()
	case "save_confirm":
		return m.saveConfirmView()
	case "builtin_warning":
		return m.builtinWarningView()
	}
	return m.listView()
}

// listView renders the main list view
func (m sourcesModel) listView() string {
	if len(m.sources) == 0 {
		return titleStyle.Render("FontGet Sources Manager") + "\n\nNo sources configured\n\n" +
			keyStyle.Render("A") + "dd  " + keyStyle.Render("Q") + "uit"
	}

	out := titleStyle.Render("FontGet Sources Manager") + "\n\n"
	for i, source := range m.sources {
		sel := " "
		if i == m.cursor {
			sel = ">"
		}
		check := " "
		if source.Enabled {
			check = "x"
		}

		builtIn := ""
		if source.IsBuiltIn {
			builtIn = " (built-in)"
		}

		out += fmt.Sprintf("%s [%s] %s%s [%s]\n", sel, check, source.Name, builtIn, source.Prefix)
	}

	out += "\n"
	if m.err != "" {
		out += errorStyle.Render("Error: "+m.err) + "\n\n"
	}

	// Better formatted commands
	commands := []string{
		keyStyle.Render("↑/↓") + " move",
		keyStyle.Render("Space/Enter") + " toggle",
		keyStyle.Render("A") + "dd",
		keyStyle.Render("E") + "dit",
		keyStyle.Render("F") + "ile",
		keyStyle.Render("D") + "elete",
		keyStyle.Render("S") + "ave & quit",
		keyStyle.Render("Q") + "uit",
	}

	out += helpStyle.Render("Commands: " + strings.Join(commands, "  "))

	return out
}

// formView renders the add/edit form
func (m sourcesModel) formView() string {
	title := "Add New Source"
	if m.state == "edit" {
		title = "Edit Source"
	}

	out := titleStyle.Render("~ "+title) + "\n\n"

	fields := []struct {
		label string
		input textinput.Model
		focus bool
	}{
		{"Name:", m.nameInput, m.focusedField == 0},
		{"URL:", m.urlInput, m.focusedField == 1},
		{"Prefix:", m.prefixInput, m.focusedField == 2},
	}

	for i, field := range fields {
		sel := " "
		if field.focus {
			sel = ">"
		}
		out += fmt.Sprintf("%s %s %s\n", sel, field.label, field.input.View())
		if i < len(fields)-1 {
			out += "\n"
		}
	}

	if m.err != "" {
		out += "\n" + errorStyle.Render("Error: "+m.err) + "\n"
	}

	commands := []string{
		keyStyle.Render("Tab/Shift+Tab") + " move",
		keyStyle.Render("Enter") + " submit",
		keyStyle.Render("Esc") + " cancel",
	}

	out += "\n" + helpStyle.Render("Commands: "+strings.Join(commands, "  "))

	return out
}

// confirmView renders the delete confirmation
func (m sourcesModel) confirmView() string {
	if len(m.sources) == 0 {
		return errorStyle.Render("No sources to delete")
	}

	source := m.sources[m.cursor]
	out := titleStyle.Render("~ Confirm Deletion") + "\n\n"
	out += fmt.Sprintf("Are you sure you want to delete '%s'?\nThis action cannot be undone.\n\n", source.Name)

	commands := []string{
		keyStyle.Render("Y") + " confirm",
		keyStyle.Render("N") + " cancel",
	}
	out += helpStyle.Render("Commands: " + strings.Join(commands, "  "))

	return out
}

// saveConfirmView renders the save confirmation
func (m sourcesModel) saveConfirmView() string {
	out := titleStyle.Render("~ Save Changes") + "\n\n"
	out += "Do you want to save your changes before quitting?\n\n"

	commands := []string{
		keyStyle.Render("Y") + " save and quit",
		keyStyle.Render("N") + " quit without saving",
	}
	out += helpStyle.Render("Commands: " + strings.Join(commands, "  "))

	return out
}

// builtinWarningView renders the built-in source warning
func (m sourcesModel) builtinWarningView() string {
	out := titleStyle.Render("~ Warning") + "\n\n"
	out += errorStyle.Render(m.err) + "\n\n"
	out += helpStyle.Render("Press " + keyStyle.Render("Enter") + " to continue")

	return out
}

// sourcesManageCmd handles the Bubble Tea source management
var sourcesManageCmd = &cobra.Command{
	Use:   "manage",
	Short: "Interactive source management with TUI",
	Long: `Launch an interactive TUI for managing font sources.

Navigation:
  ↑/↓ or j/k  - Move cursor
  Space/Enter - Toggle source enabled state
  a           - Add new source
  e           - Edit selected source (non-built-in only)
  f           - Edit source file in external editor
  d           - Delete selected source (non-built-in only)
  s           - Save changes and quit
  q           - Quit without saving

usage: fontget sources manage`,
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		model, err := NewSourcesModel()
		if err != nil {
			return fmt.Errorf("failed to initialize source manager: %w", err)
		}

		p := tea.NewProgram(model, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	sourcesCmd.AddCommand(sourcesManageCmd)
}
