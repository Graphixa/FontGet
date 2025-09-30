package cmd

import (
	"fmt"
	"strings"
	"time"

	"fontget/internal/config"
	"fontget/internal/functions"
	"fontget/internal/ui"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// SourceItem represents a source in the TUI
// This type is now defined in internal/functions/sort.go for consistency

// sourcesModel represents the main model for the sources management TUI
type sourcesModel struct {
	sources      []functions.SourceItem
	cursor       int
	manifest     *config.Manifest
	state        string // "list", "add", "edit", "confirm", "save_confirm", "builtin_warning"
	nameInput    textinput.Model
	urlInput     textinput.Model
	prefixInput  textinput.Model
	focusedField int
	err          string
	editingIndex int
	readOnly     bool // true when viewing built-in source details
	width        int  // terminal width
	height       int  // terminal height
}

// Styles are now centralized in internal/ui/styles.go

// NewSourcesModel creates a new sources management model
func NewSourcesModel() (*sourcesModel, error) {
	manifest, err := config.LoadManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
	}

	sm := &sourcesModel{
		manifest:     manifest,
		state:        "list",
		focusedField: 0,
	}

	// Convert manifest to SourceItem slice
	sm.sources = convertManifestToSourceItems(manifest)

	// Sort sources using the centralized sorting function
	functions.SortSources(sm.sources)

	// Initialize text inputs with default width (will be updated on window resize)
	sm.nameInput = textinput.New()
	sm.nameInput.Placeholder = "Source name"
	sm.nameInput.Width = 50
	sm.nameInput.TextStyle = ui.FormInput
	sm.nameInput.PlaceholderStyle = ui.FormPlaceholder
	sm.nameInput.Focus()

	sm.urlInput = textinput.New()
	sm.urlInput.Placeholder = "https://example.com/fonts.json"
	sm.urlInput.Width = 50
	sm.urlInput.TextStyle = ui.FormInput
	sm.urlInput.PlaceholderStyle = ui.FormPlaceholder

	sm.prefixInput = textinput.New()
	sm.prefixInput.Placeholder = "prefix"
	sm.prefixInput.Width = 50
	sm.prefixInput.TextStyle = ui.FormInput
	sm.prefixInput.PlaceholderStyle = ui.FormPlaceholder

	return sm, nil
}

// isBuiltInSource checks if a source name is a built-in source
func isBuiltInSource(name string) bool {
	switch name {
	case "Google Fonts", "Nerd Fonts", "Font Squirrel":
		return true
	default:
		return false
	}
}

// updateInputWidths updates the width of text inputs based on terminal size
func (m *sourcesModel) updateInputWidths() {
	width := functions.CalculateInputWidth(m.width)
	m.nameInput.Width = width
	m.urlInput.Width = width
	m.prefixInput.Width = width
}

// Init initializes the model
func (m sourcesModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages and updates the model
func (m sourcesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle window resize
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height
		m.updateInputWidths()
		return m, nil
	}

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
		case "esc", "ctrl+c":
			// Check if there are any changes made
			if m.hasChanges() {
				m.state = "save_confirm"
			} else {
				return m, tea.Quit
			}

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

				// Set read-only mode for built-in sources
				m.readOnly = m.sources[m.cursor].IsBuiltIn
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

		}

	case errorMsg:
		m.err = msg.text
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
			if m.readOnly {
				// In read-only mode, just go back to list
				m.state = "list"
				m.resetForm()
			} else if m.validateForm() {
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

	// Update focused input (only if not in read-only mode)
	if !m.readOnly {
		switch m.focusedField {
		case 0:
			m.nameInput, cmd = m.nameInput.Update(msg)
		case 1:
			m.prefixInput, cmd = m.prefixInput.Update(msg)
		case 2:
			m.urlInput, cmd = m.urlInput.Update(msg)
		}
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
			return m, tea.Quit
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

	if !m.readOnly {
		switch m.focusedField {
		case 0:
			m.nameInput.Focus()
		case 1:
			m.prefixInput.Focus()
		case 2:
			m.urlInput.Focus()
		}
	}
	// In read-only mode, we don't need to focus inputs since they're not editable
}

// resetForm resets the form inputs
func (m *sourcesModel) resetForm() {
	m.nameInput.SetValue("")
	m.urlInput.SetValue("")
	m.prefixInput.SetValue("")
	m.focusedField = 0
	m.err = ""
	m.readOnly = false
}

// convertManifestToSourceItems converts manifest sources to SourceItem slice
func convertManifestToSourceItems(manifest *config.Manifest) []functions.SourceItem {
	var sources []functions.SourceItem
	for name, source := range manifest.Sources {
		sources = append(sources, functions.SourceItem{
			Name:      name,
			Prefix:    source.Prefix,
			URL:       source.URL,
			Enabled:   source.Enabled,
			IsBuiltIn: isBuiltInSource(name),
			Priority:  source.Priority,
		})
	}
	return sources
}

// hasChanges checks if any changes have been made to the sources
func (m *sourcesModel) hasChanges() bool {
	// Load original manifest to compare
	originalManifest, err := config.LoadManifest()
	if err != nil {
		return false
	}

	// Check if number of sources changed
	if len(m.sources) != len(originalManifest.Sources) {
		return true
	}

	// Check if any source properties changed
	for _, source := range m.sources {
		if originalSource, exists := originalManifest.Sources[source.Name]; exists {
			if originalSource.Enabled != source.Enabled ||
				originalSource.Prefix != source.Prefix ||
				originalSource.URL != source.URL ||
				originalSource.Priority != source.Priority {
				return true
			}
		} else {
			// New source added
			return true
		}
	}

	// Check if any original sources were removed
	for name := range originalManifest.Sources {
		found := false
		for _, source := range m.sources {
			if source.Name == name {
				found = true
				break
			}
		}
		if !found {
			return true
		}
	}

	return false
}

// validateForm validates the form inputs using the centralized validation
func (m *sourcesModel) validateForm() bool {
	name := strings.TrimSpace(m.nameInput.Value())
	url := strings.TrimSpace(m.urlInput.Value())
	prefix := strings.TrimSpace(m.prefixInput.Value())

	// Convert prefix to lowercase for consistency
	prefix = strings.ToLower(prefix)
	m.prefixInput.SetValue(prefix)

	// Convert editingIndex to -1 if not editing
	editingIndex := -1
	if m.state == "edit" {
		editingIndex = m.editingIndex
	}

	// Use centralized validation
	result := functions.ValidateSourceForm(name, url, prefix, m.sources, editingIndex)

	if !result.IsValid {
		m.err = result.GetFirstError()
		return false
	}

	// Auto-generate prefix if empty
	if prefix == "" {
		generatedPrefix := functions.AutoGeneratePrefix(name)
		m.prefixInput.SetValue(generatedPrefix)
	}

	return true
}

// addSource adds a new source
func (m *sourcesModel) addSource() {
	name := strings.TrimSpace(m.nameInput.Value())
	url := strings.TrimSpace(m.urlInput.Value())
	prefix := strings.TrimSpace(m.prefixInput.Value())

	// Ensure prefix is always lowercase
	prefix = strings.ToLower(prefix)

	if prefix == "" {
		prefix = strings.ToLower(name)
	}

	// Assign priority to custom sources (100+ to ensure they come after built-in sources)
	newSource := functions.SourceItem{
		Name:      name,
		Prefix:    prefix,
		URL:       url,
		Enabled:   true,
		IsBuiltIn: false,
		Priority:  m.getNextCustomSourcePriority(),
	}

	m.sources = append(m.sources, newSource)
	functions.SortSources(m.sources)

	// Find the new source's position using the utility function
	m.cursor = functions.FindSourceIndex(m.sources, name)
}

// updateSource updates an existing source
func (m *sourcesModel) updateSource() {
	name := strings.TrimSpace(m.nameInput.Value())
	url := strings.TrimSpace(m.urlInput.Value())
	prefix := strings.TrimSpace(m.prefixInput.Value())

	// Ensure prefix is always lowercase
	prefix = strings.ToLower(prefix)

	if prefix == "" {
		prefix = strings.ToLower(name)
	}

	m.sources[m.editingIndex].Name = name
	m.sources[m.editingIndex].URL = url
	m.sources[m.editingIndex].Prefix = prefix

	// Re-sort sources using the centralized sorting function
	functions.SortSources(m.sources)

	// Find the updated source's position using the utility function
	m.cursor = functions.FindSourceIndex(m.sources, name)
}

// saveChanges saves the configuration to the new manifest system
func (m sourcesModel) saveChanges() tea.Cmd {
	return func() tea.Msg {
		// Load current manifest
		manifest, err := config.LoadManifest()
		if err != nil {
			return errorMsg{fmt.Sprintf("Failed to load manifest: %v", err)}
		}

		// Update manifest with current sources (including priority)
		manifest.Sources = make(map[string]config.SourceConfig)
		for _, source := range m.sources {
			manifest.Sources[source.Name] = config.SourceConfig{
				URL:      source.URL,
				Prefix:   source.Prefix,
				Enabled:  source.Enabled,
				Filename: generateFilename(source.Name),
				Priority: source.Priority,
				// Keep existing metadata
				LastSynced: getExistingLastSynced(manifest.Sources, source.Name),
				FontCount:  getExistingFontCount(manifest.Sources, source.Name),
				Version:    getExistingVersion(manifest.Sources, source.Name),
			}
		}

		// Update last modified time
		manifest.LastUpdated = time.Now()

		// Save the manifest
		if err := config.SaveManifest(manifest); err != nil {
			return errorMsg{fmt.Sprintf("Failed to save manifest: %v", err)}
		}

		return tea.Quit()
	}
}

// getNextCustomSourcePriority returns the next priority for custom sources
func (m *sourcesModel) getNextCustomSourcePriority() int {
	// Find the highest priority among custom sources
	maxPriority := 99 // Start custom sources at 100
	for _, source := range m.sources {
		if !source.IsBuiltIn && source.Priority > maxPriority {
			maxPriority = source.Priority
		}
	}
	return maxPriority + 1
}

// Helper functions to preserve existing metadata
func getExistingLastSynced(sources map[string]config.SourceConfig, name string) *time.Time {
	if existing, exists := sources[name]; exists {
		return existing.LastSynced
	}
	return nil
}

func getExistingFontCount(sources map[string]config.SourceConfig, name string) int {
	if existing, exists := sources[name]; exists {
		return existing.FontCount
	}
	return 0
}

func getExistingVersion(sources map[string]config.SourceConfig, name string) string {
	if existing, exists := sources[name]; exists {
		return existing.Version
	}
	return ""
}

// generateFilename creates a clean filename from source name
func generateFilename(sourceName string) string {
	// Convert to lowercase and replace spaces with hyphens
	filename := strings.ToLower(sourceName)
	filename = strings.ReplaceAll(filename, " ", "-")
	filename = strings.ReplaceAll(filename, "_", "-")
	return filename + ".json"
}

// errorMsg represents an error message
type errorMsg struct {
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
		return ui.PageTitle.Render("FontGet Sources Manager") + "\n\nNo sources configured\n\n" +
			ui.CommandKey.Render("A") + "dd  " + ui.CommandKey.Render("Q") + "uit"
	}

	out := ui.PageTitle.Render("FontGet Sources Manager") + "\n\n"

	// Define column widths - simplified structure
	statusWidth := 6 // For "[x] " or "[ ] "
	nameWidth := 35  // Source name column (wider to accommodate type)

	// Table rows
	for i, source := range m.sources {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		checkbox := "[ ]"
		if source.Enabled {
			checkbox = "[x]"
		}

		// Build styled source name with styled tag using the utility function
		sourceNameWithType := ui.RenderSourceNameWithTag(source.Name, source.IsBuiltIn)

		// Format the row with proper spacing - align with header
		row := fmt.Sprintf("%s%-*s %-*s",
			cursor,
			statusWidth, checkbox,
			nameWidth, sourceNameWithType)

		// Highlight selected row
		if i == m.cursor {
			row = ui.TableSelectedRow.Render(row)
		}

		out += row + "\n"
	}

	out += "\n"
	if m.err != "" {
		out += ui.RenderError(m.err) + "\n\n"
	}

	// Better formatted commands using the utility function
	commands := []string{
		ui.RenderKeyWithDescription("↑/↓", "Move"),
		ui.RenderKeyWithDescription("Space/Enter", "Enable/Disable"),
		ui.RenderKeyWithDescription("A", "Add"),
		ui.RenderKeyWithDescription("E", "Edit"),
		ui.RenderKeyWithDescription("D", "Delete"),
		ui.RenderKeyWithDescription("Esc", "Exit"),
	}

	helpText := strings.Join(commands, "  ")
	out += helpText

	return out
}

// formView renders the add/edit form
func (m sourcesModel) formView() string {
	title := "Add New Source"
	if m.state == "edit" {
		if m.readOnly {
			title = "Edit Source (Read-Only)"
		} else {
			title = "Edit Source"
		}
	}

	out := ui.PageTitle.Render(title) + "\n\n"

	// Create fields based on mode
	var fields []struct {
		label string
		value string
		focus bool
	}

	if m.readOnly && len(m.sources) > m.editingIndex {
		// Read-only mode: show all fields as static text
		source := m.sources[m.editingIndex]
		fields = []struct {
			label string
			value string
			focus bool
		}{
			{"Source Name:", source.Name, m.focusedField == 0},
			{"URL:", source.URL, m.focusedField == 1},
			{"Prefix:", source.Prefix, m.focusedField == 2},
		}
	} else {
		// Edit mode: show input fields
		fields = []struct {
			label string
			value string
			focus bool
		}{
			{"Name:", m.nameInput.Value(), m.focusedField == 0},
			{"Prefix:", m.prefixInput.Value(), m.focusedField == 1},
			{"URL:", m.urlInput.Value(), m.focusedField == 2},
		}
	}

	for i, field := range fields {
		// Format the field value
		var fieldValue string
		if m.readOnly {
			// In read-only mode, show as static text
			fieldValue = ui.FormReadOnly.Render(field.value)
		} else {
			// In edit mode, show as input field with custom styling
			if field.focus {
				// For the focused field, use the textinput's View() method to get the blinking cursor
				if i == 0 {
					fieldValue = m.nameInput.View()
				} else if i == 1 {
					fieldValue = m.prefixInput.View()
				} else if i == 2 {
					fieldValue = m.urlInput.View()
				}
			} else {
				// For non-focused fields, show the value with custom styling
				var inputValue string
				if i == 0 {
					inputValue = m.nameInput.Value()
				} else if i == 1 {
					inputValue = m.prefixInput.Value()
				} else if i == 2 {
					inputValue = m.urlInput.Value()
				}

				// Apply custom styling to the input value
				if inputValue == "" {
					// Show placeholder with placeholder styling
					var placeholder string
					if i == 0 {
						placeholder = m.nameInput.Placeholder
					} else if i == 1 {
						placeholder = m.prefixInput.Placeholder
					} else if i == 2 {
						placeholder = m.urlInput.Placeholder
					}
					fieldValue = ui.FormPlaceholder.Render(placeholder)
				} else {
					// Show actual input value with form input styling
					fieldValue = ui.FormInput.Render(inputValue)
				}
			}
		}

		// For focused fields, don't add manual cursor since textinput.View() handles it
		// For non-focused fields, add a space for alignment
		sel := " "
		if !field.focus && !m.readOnly {
			sel = " "
		}

		styledLabel := ui.FormLabel.Render(field.label)
		out += fmt.Sprintf("  %s %s %s\n", styledLabel, sel, fieldValue)
		if i < len(fields)-1 {
			out += "\n"
		}
	}

	if m.err != "" {
		out += "\n" + ui.RenderError(m.err) + "\n"
	}

	commands := []string{
		ui.RenderKeyWithDescription("Tab/Shift+Tab", "Move"),
		ui.RenderKeyWithDescription("Enter", "Submit"),
		ui.RenderKeyWithDescription("Esc", "Cancel"),
	}

	if m.readOnly {
		commands = []string{
			ui.RenderKeyWithDescription("Tab/Shift+Tab", "Move"),
			ui.RenderKeyWithDescription("Enter/Esc", "Back"),
		}
	}

	helpText := strings.Join(commands, "  ")
	out += "\n" + helpText

	return out
}

// confirmView renders the delete confirmation
func (m sourcesModel) confirmView() string {
	if len(m.sources) == 0 {
		return ui.RenderError("No sources to delete")
	}

	source := m.sources[m.cursor]
	out := ui.PageTitle.Render("Confirm Deletion") + "\n\n"
	styledName := ui.TableSourceName.Render(source.Name)
	out += fmt.Sprintf("Are you sure you want to delete '%s'?\nThis cannot be undone.\n\n", styledName)

	commands := []string{
		ui.RenderKeyWithDescription("Y", "Yes"),
		ui.RenderKeyWithDescription("N", "No"),
	}
	helpText := strings.Join(commands, "  ")
	out += helpText

	return out
}

// saveConfirmView renders the save confirmation
func (m sourcesModel) saveConfirmView() string {
	out := ui.PageTitle.Render("Save Changes") + "\n\n"
	out += "You have unsaved changes. Do you want to save your changes?\n\n"

	commands := []string{
		ui.RenderKeyWithDescription("Y", "Yes"),
		ui.RenderKeyWithDescription("N", "No"),
	}
	helpText := strings.Join(commands, "  ")
	out += helpText

	return out
}

// builtinWarningView renders the built-in source warning
func (m sourcesModel) builtinWarningView() string {
	out := ui.PageTitle.Render("Warning") + "\n\n"
	out += ui.RenderError(m.err) + "\n\n"
	out += ui.FeedbackText.Render("Press " + ui.CommandKey.Render("Enter") + " to continue")

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
  e           - Edit selected source (view built-in details)
  d           - Delete selected source (non-built-in only)
  esc         - Quit (prompts to save if changes made)

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
