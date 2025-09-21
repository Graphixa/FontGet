package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"fontget/internal/config"
	"fontget/internal/repo"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
)

// updateModel represents the update progress TUI
type updateModel struct {
	sources          []string
	currentSource    int
	spinner          spinner.Model
	status           map[string]string
	errors           map[string]string
	completed        int
	total            int
	quitting         bool
	startTime        time.Time
	manifest         *repo.FontManifest
	verbose          bool
	initialFontCount int
}

// updateProgressMsg represents progress update
type updateProgressMsg struct {
	source   string
	progress float64
	status   string
}

// updateCompleteMsg represents completion of a source
type updateCompleteMsg struct {
	source string
	status string
	error  error
}

// updateFinishedMsg represents completion of all updates
type updateFinishedMsg struct {
	manifest *repo.FontManifest
	error    error
}

// Styles for the update TUI
var (
	updateTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("170")).
				MarginBottom(1)

	updateSourceStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("212"))

	updateStatusStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("82")).
				Bold(true)

	updateErrorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Bold(true)

	updateWarningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214")).
				Bold(true)

	updateSummaryStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				MarginTop(1)
)

// NewUpdateModel creates a new update progress model
func NewUpdateModel(verbose bool) (*updateModel, error) {
	sourcesConfig, err := config.LoadSourcesConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load sources config: %w", err)
	}

	// Get enabled sources
	enabledSources := config.GetEnabledSourcesInOrder(sourcesConfig)
	if len(enabledSources) == 0 {
		return nil, fmt.Errorf("no sources are enabled")
	}

	// Create spinner
	spin := spinner.New()
	spin.Spinner = spinner.Dot
	spin.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Skip initial font count calculation for faster startup
	initialFontCount := 0

	return &updateModel{
		sources:          enabledSources,
		spinner:          spin,
		status:           make(map[string]string),
		errors:           make(map[string]string),
		total:            len(enabledSources),
		startTime:        time.Now(),
		verbose:          verbose,
		initialFontCount: initialFontCount,
	}, nil
}

// Init initializes the model
func (m updateModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.updateNextSource(),
	)
}

// Update handles messages and updates the model
func (m updateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc", "enter", " ":
			if m.quitting {
				// If already completed, any key quits
				return m, tea.Quit
			} else {
				// If still running, quit immediately
				m.quitting = true
				return m, tea.Quit
			}
		}

	case updateProgressMsg:
		m.status[msg.source] = msg.status

	case updateCompleteMsg:
		m.status[msg.source] = msg.status
		if msg.error != nil {
			m.errors[msg.source] = msg.error.Error()
		}
		m.completed++

		if m.completed < m.total {
			m.currentSource++
			return m, m.updateNextSource()
		} else {
			return m, m.finishUpdate()
		}

	case updateFinishedMsg:
		if msg.error != nil {
			m.errors["system"] = msg.error.Error()
		} else {
			m.manifest = msg.manifest
		}
		m.quitting = true
		// Quit immediately - no delays
		return m, tea.Quit

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, cmd
}

// updateNextSource starts updating the next source
func (m updateModel) updateNextSource() tea.Cmd {
	if m.currentSource >= len(m.sources) {
		return nil
	}

	source := m.sources[m.currentSource]
	m.status[source] = "Starting..."

	return func() tea.Msg {
		// Load the current config to get the source URL
		sourcesConfig, err := config.LoadSourcesConfig()
		if err != nil {
			return updateCompleteMsg{
				source: source,
				status: "Failed",
				error:  err,
			}
		}

		// Get the source URL for validation
		sourceConfig, exists := sourcesConfig.Sources[source]
		if !exists {
			return updateCompleteMsg{
				source: source,
				status: "Failed",
				error:  fmt.Errorf("source not found in configuration"),
			}
		}

		// Create HTTP client with shorter timeout for faster error detection
		client := &http.Client{
			Timeout: 5 * time.Second,
		}

		// Store verbose info for display in TUI
		if m.verbose {
			m.status[source] = fmt.Sprintf("Checking %s...", sourceConfig.Path)
		}

		// First, check if source is reachable with HEAD request (fast validation)
		headResp, err := client.Head(sourceConfig.Path)
		if err != nil {
			// Provide more specific error messages
			var errorMsg string
			if strings.Contains(err.Error(), "timeout") {
				errorMsg = fmt.Sprintf("request timeout after 5 seconds")
			} else if strings.Contains(err.Error(), "no such host") {
				errorMsg = fmt.Sprintf("host not found")
			} else if strings.Contains(err.Error(), "connection refused") {
				errorMsg = fmt.Sprintf("connection refused")
			} else {
				errorMsg = fmt.Sprintf("network error: %v", err)
			}

			return updateCompleteMsg{
				source: source,
				status: "Failed",
				error:  fmt.Errorf(errorMsg),
			}
		}
		headResp.Body.Close()

		// Check HTTP status code immediately
		if headResp.StatusCode >= 400 {
			return updateCompleteMsg{
				source: source,
				status: "Failed",
				error:  fmt.Errorf("source URL returned status %d", headResp.StatusCode),
			}
		}

		// Source is reachable, now download the full content
		if m.verbose {
			m.status[source] = fmt.Sprintf("Downloading from %s...", sourceConfig.Path)
		}

		resp, err := client.Get(sourceConfig.Path)
		if err != nil {
			return updateCompleteMsg{
				source: source,
				status: "Failed",
				error:  fmt.Errorf("failed to download source: %w", err),
			}
		}

		// Store verbose info for display in TUI
		if m.verbose {
			m.status[source] = fmt.Sprintf("Downloaded (%d bytes), validating JSON...", resp.ContentLength)
		}

		// Read the response body to actually download the content
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close() // Close immediately after reading
		if err != nil {
			return updateCompleteMsg{
				source: source,
				status: "Failed",
				error:  fmt.Errorf("failed to read source content: %w", err),
			}
		}

		// Validate that it's valid JSON
		var jsonData interface{}
		if err := json.Unmarshal(body, &jsonData); err != nil {
			return updateCompleteMsg{
				source: source,
				status: "Failed",
				error:  fmt.Errorf("source content is not valid JSON: %w", err),
			}
		}

		// Store verbose info for display in TUI
		if m.verbose {
			m.status[source] = "Successfully downloaded and validated"
		}

		// Simulate a small delay for the update process
		time.Sleep(200 * time.Millisecond)

		return updateCompleteMsg{
			source: source,
			status: "Completed",
			error:  nil,
		}
	}
}

// finishUpdate completes the update process
func (m updateModel) finishUpdate() tea.Cmd {
	return func() tea.Msg {
		// Load manifest to get actual font count
		manifest, err := repo.GetManifest(nil, nil)
		return updateFinishedMsg{
			manifest: manifest,
			error:    err,
		}
	}
}

// View renders the update progress UI
func (m updateModel) View() string {
	var content strings.Builder

	// Always show the updating message at the top
	cyan := color.New(color.FgCyan).SprintFunc()
	content.WriteString("\n") // Add space above the message
	content.WriteString(cyan("Updating FontGet Sources"))
	content.WriteString("\n\n") // Add space between message and source list

	if m.quitting {
		// Show completed results
		content.WriteString(m.renderSummary())
		return content.String()
	}

	// Show all sources with their status
	for i, source := range m.sources {
		// Status indicator - clean text-based
		var indicator string
		if i < m.currentSource {
			indicator = updateStatusStyle.Render("✓")
		} else if i == m.currentSource {
			indicator = m.spinner.View()
		} else {
			indicator = "○"
		}

		// Error indicator
		if err, hasError := m.errors[source]; hasError {
			indicator = updateErrorStyle.Render("✗")
			content.WriteString(fmt.Sprintf("   %s %s (%s)\n", indicator, m.getDisplayName(source), err))
		} else {
			// Show verbose status if available
			if m.verbose && m.status[source] != "" {
				content.WriteString(fmt.Sprintf("   %s %s - %s\n", indicator, m.getDisplayName(source), m.status[source]))
			} else {
				content.WriteString(fmt.Sprintf("   %s %s\n", indicator, m.getDisplayName(source)))
			}
		}
	}

	// Help (inline)
	content.WriteString("   ")
	content.WriteString(updateSummaryStyle.Render("Press 'q' to quit"))

	return content.String()
}

// getDisplayName returns the proper display name for a source
func (m updateModel) getDisplayName(sourceName string) string {
	switch sourceName {
	case "Google":
		return "Google Fonts"
	case "NerdFonts":
		return "Nerd Fonts"
	case "FontSquirrel":
		return "Font Squirrel"
	default:
		return sourceName
	}
}

// renderSummary renders the final summary
func (m updateModel) renderSummary() string {
	var content strings.Builder

	// Count successful and failed sources
	successful := 0
	failed := 0
	for _, source := range m.sources {
		if _, hasError := m.errors[source]; hasError {
			failed++
		} else {
			successful++
		}
	}

	// Individual source results first
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	for _, source := range m.sources {
		if err, hasError := m.errors[source]; hasError {
			content.WriteString(fmt.Sprintf("   %s %s (%s)\n", red("✗"), m.getDisplayName(source), err))
		} else {
			// Show verbose status if available
			if m.verbose && m.status[source] != "" {
				content.WriteString(fmt.Sprintf("   %s %s - %s\n", green("✓"), m.getDisplayName(source), m.status[source]))
			} else {
				content.WriteString(fmt.Sprintf("   %s %s\n", green("✓"), m.getDisplayName(source)))
			}
		}
	}

	// Status Report at the bottom like install command
	content.WriteString("\n")
	content.WriteString("Status Report")
	content.WriteString("\n")
	content.WriteString("---------------------------------------------")
	content.WriteString("\n")

	// Status line with colors like install command
	yellow := color.New(color.FgYellow).SprintFunc()
	content.WriteString(fmt.Sprintf("%s: %d  |  %s: %d  |  %s: %d\n",
		green("Updated"), successful,
		yellow("Skipped"), 0, // No skipped in update
		red("Failed"), failed))

	// Add font count in gray - calculate actual count
	gray := color.New(color.FgHiBlack).SprintFunc()
	fontCount := m.calculateFontCount()
	content.WriteString(fmt.Sprintf("\n%s\n\n", gray(fmt.Sprintf("Total fonts available: %d", fontCount))))

	return content.String()
}

// calculateFontCount calculates the total number of fonts available
func (m updateModel) calculateFontCount() int {
	if m.manifest == nil {
		return 0
	}

	totalFonts := 0
	for _, sourceInfo := range m.manifest.Sources {
		totalFonts += len(sourceInfo.Fonts)
	}
	return totalFonts
}

// RunSourcesUpdateTUI runs the sources update TUI
func RunSourcesUpdateTUI(verbose bool) error {
	model, err := NewUpdateModel(verbose)
	if err != nil {
		return fmt.Errorf("failed to initialize update model: %w", err)
	}

	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}
