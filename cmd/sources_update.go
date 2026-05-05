package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	sources       []string
	currentSource int
	spinner       spinner.Model
	status        map[string]string
	errors        map[string]string
	completed     int
	total         int
	quitting      bool
	startTime     time.Time
	manifest      *repo.FontManifest
	verbose       bool
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
	url    string // configured source URL (for file log / debug)
}

// updateFinishedMsg represents completion of all updates
type updateFinishedMsg struct {
	manifest *repo.FontManifest
	error    error
}

// createHTTPClient creates a properly configured HTTP client with timeouts
func createHTTPClient() *http.Client {
	appConfig := config.GetUserPreferences()
	generalTimeout := config.ParseDuration(appConfig.Network.RequestTimeout, 10*time.Second)

	return &http.Client{
		Timeout: generalTimeout,
		Transport: &http.Transport{
			ResponseHeaderTimeout: 5 * time.Second,  // Keep internal transport timeout
			IdleConnTimeout:       30 * time.Second, // Keep internal transport timeout
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
	spin.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(ui.SpinnerColor)) // Use theme color (accent)

	return &updateModel{
		sources:   enabledSources,
		spinner:   spin,
		status:    make(map[string]string),
		errors:    make(map[string]string),
		total:     len(enabledSources),
		startTime: time.Now(),
		verbose:   verbose,
		// No operation-level timeout - individual source downloads use DownloadTimeout
		// Operation completes when all sources are processed (success or failure)
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

	// No operation-level timeout check - individual downloads use DownloadTimeout
	// Operation completes when all sources are processed

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc", "enter", " ":
			if m.quitting {
				// If already completed, any key quits
				return m, tea.Quit
			} else {
				// If still running, mark as interrupted and quit immediately
				// Don't process any more updateCompleteMsg messages
				m.quitting = true
				return m, tea.Quit
			}
		}

	case updateProgressMsg:
		m.status[msg.source] = msg.status

	case updateCompleteMsg:
		logSourcesUpdateTUIPerSource(msg)
		// Ignore completion messages if we've already quit (interrupted)
		if m.quitting {
			return m, nil
		}

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
			if lg := GetLogger(); lg != nil {
				lg.Error("Sources update: manifest refresh after per-source downloads failed: %v", msg.error)
			}
			output.GetDebug().State("Sources update: GetManifest after downloads failed: %v", msg.error)
		} else {
			m.manifest = msg.manifest
			n := 0
			if msg.manifest != nil {
				for _, s := range msg.manifest.Sources {
					n += len(s.Fonts)
				}
			}
			if lg := GetLogger(); lg != nil {
				lg.Info("Sources update: manifest refresh completed (%d fonts across sources)", n)
			}
			output.GetDebug().State("Sources update: manifest refresh ok, total fonts=%d", n)
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
				url:    "",
			}
		}

		// Get the source URL for validation
		sourceConfig, exists := manifest.Sources[source]
		if !exists {
			return updateCompleteMsg{
				source: source,
				status: "Failed",
				error:  fmt.Errorf("source not found in configuration"),
				url:    "",
			}
		}
		sourceURL := sourceConfig.URL

		// Create HTTP client with proper timeout and context handling
		appConfig := config.GetUserPreferences()
		requestTimeout := config.ParseDuration(appConfig.Network.RequestTimeout, 10*time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		defer cancel()

		client := createHTTPClient()

		// Individual source downloads use DownloadTimeout via stall detector
		// No operation-level timeout check needed

		// Store verbose info for display in TUI
		if m.verbose {
			m.status[source] = fmt.Sprintf("Checking %s...", sourceURL)
		}

		// Create request with context for proper cancellation
		req, err := http.NewRequestWithContext(ctx, "GET", sourceURL, nil)
		if err != nil {
			return updateCompleteMsg{
				source: source,
				status: "Failed",
				error:  fmt.Errorf("failed to create request: %w", err),
				url:    sourceURL,
			}
		}

		// Add proper headers
		req.Header.Set("User-Agent", repo.DownloadUserAgent())
		req.Header.Set("Accept", "application/json")

		// Make the request
		if m.verbose {
			m.status[source] = fmt.Sprintf("Downloading from %s...", sourceURL)
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
				url:    sourceURL,
			}
		}
		defer resp.Body.Close()

		// Check HTTP status code
		if resp.StatusCode >= 400 {
			return updateCompleteMsg{
				source: source,
				status: "Failed",
				error:  fmt.Errorf("source URL returned status %d: %s", resp.StatusCode, resp.Status),
				url:    sourceURL,
			}
		}

		// Store verbose info for display in TUI
		if m.verbose {
			m.status[source] = fmt.Sprintf("Downloaded (%d bytes), validating JSON...", resp.ContentLength)
		}

		// Read the response body with size limit to prevent memory issues
		// Hardcoded 50MB limit for security (prevents DoS attacks from huge source files)
		maxSize := int64(50 * 1024 * 1024) // 50MB hardcoded limit
		body, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
		if err != nil {
			return updateCompleteMsg{
				source: source,
				status: "Failed",
				error:  fmt.Errorf("failed to read source content: %w", err),
				url:    sourceURL,
			}
		}

		// Check if we hit the size limit
		if len(body) == int(maxSize) {
			maxSizeMB := maxSize / (1024 * 1024)
			return updateCompleteMsg{
				source: source,
				status: "Failed",
				error:  fmt.Errorf("source file too large (max %dMB)", maxSizeMB),
				url:    sourceURL,
			}
		}

		// Validate that it's valid JSON
		var jsonData interface{}
		if err := json.Unmarshal(body, &jsonData); err != nil {
			return updateCompleteMsg{
				source: source,
				status: "Failed",
				error:  fmt.Errorf("source content is not valid JSON: %w", err),
				url:    sourceURL,
			}
		}

		// Store verbose info for display in TUI
		if m.verbose {
			m.status[source] = "Successfully downloaded and validated"
		}

		return updateCompleteMsg{
			source: source,
			status: "Completed",
			error:  nil,
			url:    sourceURL,
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
				if uerr := config.UpdateSourcesLastUpdated(); uerr != nil {
					if log := GetLogger(); log != nil {
						log.Warn("UpdateSourcesLastUpdated: %v", uerr)
					}
				}
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

	// Start with a blank line for consistent spacing
	content.WriteString("\n")
	content.WriteString(ui.PageTitle.Render("Updating Sources"))
	content.WriteString("\n\n")

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
			indicator = ui.SuccessText.Render("✓")
		} else if i == m.currentSource {
			// Spinner - strip any extra whitespace (spinner View() includes ANSI color codes)
			indicator = strings.TrimSpace(m.spinner.View())
		} else {
			indicator = "○"
		}

		// Error indicator
		var sourceErr string
		var hasError bool
		if err, ok := m.errors[source]; ok {
			indicator = ui.ErrorText.Render("✗")
			sourceErr = err
			hasError = true
		}

		// Normalize spacing: ensure consistent spacing after all indicators
		// The Dot spinner is 2 columns wide visually, so it doesn't need an extra space
		// Single-column indicators (✓, ○, ✗) need a space to match the spinner's 2-column width
		// Remove any trailing space that might be in the indicator string itself
		indicator = strings.TrimRight(indicator, " \t")

		// Check if this is the spinner (2 columns wide) or a single-column indicator
		indicatorWidth := lipgloss.Width(indicator)
		var indicatorWithSpace string
		if indicatorWidth >= 2 {
			// Spinner is 2+ columns wide - don't add space (it's already wide enough)
			indicatorWithSpace = indicator
		} else {
			// Single-column indicator - add space to match spinner width
			indicatorWithSpace = indicator + " "
		}

		// Render with normalized spacing
		if hasError {
			content.WriteString(fmt.Sprintf("   %s%s (%s)\n", indicatorWithSpace, m.getDisplayName(source), sourceErr))
		} else {
			// Show verbose status if available
			if m.verbose && m.status[source] != "" {
				content.WriteString(fmt.Sprintf("   %s%s - %s\n", indicatorWithSpace, m.getDisplayName(source), m.status[source]))
			} else {
				content.WriteString(fmt.Sprintf("   %s%s\n", indicatorWithSpace, m.getDisplayName(source)))
			}
		}
	}

	// Help (inline) - add spacing and use command label style
	content.WriteString("\n")
	content.WriteString(ui.TextBold.Render("Press 'Q' to Quit"))

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
			content.WriteString(fmt.Sprintf("   %s %s (%s)\n", ui.ErrorText.Render("✗"), m.getDisplayName(source), err))
		} else if m.status[source] == "Completed" {
			// Only show as completed if status is explicitly "Completed"
			// This ensures interrupted or not-started sources don't show as completed
			if m.verbose && m.status[source] != "" {
				content.WriteString(fmt.Sprintf("   %s %s - %s\n", ui.SuccessText.Render("✓"), m.getDisplayName(source), m.status[source]))
			} else {
				content.WriteString(fmt.Sprintf("   %s %s\n", ui.SuccessText.Render("✓"), m.getDisplayName(source)))
			}
		} else {
			// Source was interrupted or not started - show as skipped/interrupted
			if m.status[source] != "" {
				// Was in progress but interrupted
				content.WriteString(fmt.Sprintf("   %s %s (interrupted)\n", ui.WarningText.Render("○"), m.getDisplayName(source)))
			} else {
				// Never started
				content.WriteString(fmt.Sprintf("   %s %s (not started)\n", ui.WarningText.Render("○"), m.getDisplayName(source)))
			}
		}
	}

	// Add font count in darker gray - calculate actual count
	fontCount := m.calculateFontCount()
	content.WriteString(fmt.Sprintf("\n%s\n", ui.Text.Render(fmt.Sprintf("Total fonts available: %d", fontCount))))

	// Status Report at the bottom (only shown in verbose mode)
	if m.verbose {
		content.WriteString("\n")
		content.WriteString(ui.TextBold.Render("Status Report"))
		content.WriteString("\n")
		content.WriteString("---------------------------------------------")
		content.WriteString("\n")

		// Status line with colors like install command
		content.WriteString(fmt.Sprintf("%s: %d  |  %s: %d  |  %s: %d\n\n",
			ui.SuccessText.Render("Updated"), successful,
			ui.WarningText.Render("Skipped"), 0, // No skipped in update
			ui.ErrorText.Render("Failed"), failed))
	} else {
		// Add blank line at end if not verbose
		content.WriteString("\n")
	}

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

func hostForSourcesLog(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Host
}

// logSourcesUpdateCLIStep records the same per-source lines as the TUI path (verbose `sources update -v`).
func logSourcesUpdateCLIStep(sourceName, sourceURL string, err error) {
	if lg := GetLogger(); lg != nil {
		host := hostForSourcesLog(sourceURL)
		if err != nil {
			lg.Error("Sources update: source=%q status=failed host=%s err=%v", sourceName, host, err)
		} else {
			lg.Info("Sources update: source=%q status=ok host=%s", sourceName, host)
		}
	}
	if err != nil {
		output.GetDebug().State("Sources update: source=%q failed url=%q: %v", sourceName, sourceURL, err)
	} else {
		output.GetDebug().State("Sources update: source=%q ok url=%q", sourceName, sourceURL)
	}
}

// logSourcesUpdateTUIPerSource writes one file-log line per source (and debug detail when --debug).
func logSourcesUpdateTUIPerSource(msg updateCompleteMsg) {
	logSourcesUpdateCLIStep(msg.source, msg.url, msg.error)
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
