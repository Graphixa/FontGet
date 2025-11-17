package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"fontget/internal/config"
	"fontget/internal/functions"
	"fontget/internal/output"
	"fontget/internal/repo"
	"fontget/internal/ui"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	timeout          time.Duration
}

// updateProgressMsg represents progress update
type updateProgressMsg struct {
	source string
	status string
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

// Styles are now centralized in internal/ui/styles.go

// createHTTPClient creates a properly configured HTTP client with timeouts
func createHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			ResponseHeaderTimeout: 5 * time.Second,
			IdleConnTimeout:       30 * time.Second,
			MaxIdleConns:          10,
			MaxIdleConnsPerHost:   2,
		},
	}
}

// NewUpdateModel creates a new update progress model
func NewUpdateModel(verbose bool) (*updateModel, error) {
	manifest, err := config.LoadManifest()
	if err != nil {
		// Note: This is called from TUI, so verbose/debug output may not be appropriate here
		// The error will be returned and handled by the calling command
		return nil, fmt.Errorf("unable to load font repository: %v", err)
	}

	// Get enabled sources
	enabledSources := functions.GetEnabledSourcesInOrder(manifest)
	if len(enabledSources) == 0 {
		return nil, fmt.Errorf("no sources are enabled")
	}

	// Create spinner
	spin := spinner.New()
	spin.Spinner = spinner.Dot
	spin.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#cba6f7")) // Mauve color

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
		timeout:          60 * time.Second, // 1 minute total timeout
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

	// Check for timeout
	if time.Since(m.startTime) > m.timeout {
		m.errors["system"] = "update timeout exceeded"
		m.quitting = true
		return m, tea.Quit
	}

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
		// Load the current manifest to get the source URL
		manifest, err := config.LoadManifest()
		if err != nil {
			return updateCompleteMsg{
				source: source,
				status: "Failed",
				error:  err,
			}
		}

		// Get the source URL for validation
		sourceConfig, exists := manifest.Sources[source]
		if !exists {
			return updateCompleteMsg{
				source: source,
				status: "Failed",
				error:  fmt.Errorf("source not found in configuration"),
			}
		}

		// Create HTTP client with proper timeout and context handling
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		client := createHTTPClient()

		// Check if we should abort due to timeout
		if time.Since(m.startTime) > m.timeout {
			cancel()
			return updateCompleteMsg{
				source: source,
				status: "Failed",
				error:  fmt.Errorf("update timeout exceeded"),
			}
		}

		// Store verbose info for display in TUI
		if m.verbose {
			m.status[source] = fmt.Sprintf("Checking %s...", sourceConfig.URL)
		}

		// Create request with context for proper cancellation
		req, err := http.NewRequestWithContext(ctx, "GET", sourceConfig.URL, nil)
		if err != nil {
			return updateCompleteMsg{
				source: source,
				status: "Failed",
				error:  fmt.Errorf("failed to create request: %w", err),
			}
		}

		// Add proper headers
		req.Header.Set("User-Agent", "FontGet/1.0")
		req.Header.Set("Accept", "application/json")

		// Make the request
		if m.verbose {
			m.status[source] = fmt.Sprintf("Downloading from %s...", sourceConfig.URL)
		}

		resp, err := client.Do(req)
		if err != nil {
			// Provide more specific error messages
			var errorMsg string
			if ctx.Err() == context.DeadlineExceeded {
				errorMsg = "request timeout after 10 seconds"
			} else if strings.Contains(err.Error(), "no such host") {
				errorMsg = "host not found"
			} else if strings.Contains(err.Error(), "connection refused") {
				errorMsg = "connection refused"
			} else if strings.Contains(err.Error(), "timeout") {
				errorMsg = "connection timeout"
			} else {
				errorMsg = fmt.Sprintf("network error: %v", err)
			}

			return updateCompleteMsg{
				source: source,
				status: "Failed",
				error:  fmt.Errorf("%s", errorMsg),
			}
		}
		defer resp.Body.Close()

		// Check HTTP status code
		if resp.StatusCode >= 400 {
			return updateCompleteMsg{
				source: source,
				status: "Failed",
				error:  fmt.Errorf("source URL returned status %d: %s", resp.StatusCode, resp.Status),
			}
		}

		// Store verbose info for display in TUI
		if m.verbose {
			m.status[source] = fmt.Sprintf("Downloaded (%d bytes), validating JSON...", resp.ContentLength)
		}

		// Read the response body with size limit to prevent memory issues
		const maxSize = 50 * 1024 * 1024 // 50MB limit
		body, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
		if err != nil {
			return updateCompleteMsg{
				source: source,
				status: "Failed",
				error:  fmt.Errorf("failed to read source content: %w", err),
			}
		}

		// Check if we hit the size limit
		if len(body) == maxSize {
			return updateCompleteMsg{
				source: source,
				status: "Failed",
				error:  fmt.Errorf("source file too large (max 50MB)"),
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
		// Use a timeout context to prevent hanging
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Load manifest to get actual font count with timeout
		done := make(chan struct{})
		var manifest *repo.FontManifest
		var err error

		// Use a goroutine with proper cleanup
		go func() {
			defer func() {
				// Ensure the done channel is always closed
				select {
				case <-done:
					// Already closed
				default:
					close(done)
				}
			}()

			// Check if context is already cancelled
			select {
			case <-ctx.Done():
				return
			default:
			}

			manifest, err = repo.GetManifest(nil, nil)
		}()

		select {
		case <-done:
			// Manifest loaded successfully or with error
			if err == nil {
				// Update the sources last updated timestamp
				config.UpdateSourcesLastUpdated()
			}
		case <-ctx.Done():
			// Timeout occurred - ensure goroutine cleanup
			cancel()
			err = fmt.Errorf("manifest loading timeout after 5 seconds")
		}

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
	content.WriteString("\n") // Add space above the message
	content.WriteString(ui.PageTitle.Render("Updating Sources"))
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
			indicator = ui.FeedbackSuccess.Render("✓")
		} else if i == m.currentSource {
			indicator = m.spinner.View()
		} else {
			indicator = "○"
		}

		// Error indicator
		if err, hasError := m.errors[source]; hasError {
			indicator = ui.FeedbackError.Render("✗")
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

	// Help (inline) - add spacing and use command label style
	content.WriteString("\n")
	content.WriteString(ui.CommandLabel.Render("Press 'Q' to Quit"))

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
	for _, source := range m.sources {
		if err, hasError := m.errors[source]; hasError {
			content.WriteString(fmt.Sprintf("   %s %s (%s)\n", ui.FeedbackError.Render("✗"), m.getDisplayName(source), err))
		} else {
			// Show verbose status if available
			if m.verbose && m.status[source] != "" {
				content.WriteString(fmt.Sprintf("   %s %s - %s\n", ui.FeedbackSuccess.Render("✓"), m.getDisplayName(source), m.status[source]))
			} else {
				content.WriteString(fmt.Sprintf("   %s %s\n", ui.FeedbackSuccess.Render("✓"), m.getDisplayName(source)))
			}
		}
	}

	// Status Report at the bottom like install command
	content.WriteString("\n")
	content.WriteString(ui.ReportTitle.Render("Status Report"))
	content.WriteString("\n")
	content.WriteString("---------------------------------------------")
	content.WriteString("\n")

	// Status line with colors like install command
	content.WriteString(fmt.Sprintf("%s: %d  |  %s: %d  |  %s: %d\n",
		ui.FeedbackSuccess.Render("Updated"), successful,
		ui.FeedbackWarning.Render("Skipped"), 0, // No skipped in update
		ui.FeedbackError.Render("Failed"), failed))

	// Add font count in darker gray - calculate actual count
	fontCount := m.calculateFontCount()
	content.WriteString(fmt.Sprintf("\n%s\n\n", ui.FeedbackText.Render(fmt.Sprintf("Total fonts available: %d", fontCount))))

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
		output.GetVerbose().Error("%v", err)
		output.GetDebug().Error("NewUpdateModel() failed: %v", err)
		return fmt.Errorf("unable to initialize update model: %v", err)
	}

	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}
