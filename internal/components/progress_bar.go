package components

import (
	"fmt"
	"os"
	"strings"
	"time"

	"fontget/internal/shared"
	"fontget/internal/ui"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
)

// OperationItem represents a single item in the progress display
type OperationItem struct {
	Name          string   // "Roboto"
	SourceName    string   // "Google Fonts", "Font Squirrel", etc.
	Status        string   // "completed", "in_progress", "pending", "failed"
	StatusMessage string   // "Installed", "Downloading from X", etc.
	ErrorMessage  string   // Brief error message for failed items (shown in error color)
	Variants      []string // For verbose mode: ["Roboto-Regular.ttf", ...]
	Scope         string   // For remove command: "user scope"
}

// ProgressBarModel manages the progress display
type ProgressBarModel struct {
	Title         string
	TotalItems    int
	Items         []OperationItem
	VerboseMode   bool // Show operational details and file/variant listings (static display, no spinner)
	DebugMode     bool // Show technical details (static display, no spinner)
	ProgressBar   progress.Model
	Spinner       spinner.Model
	operationFunc func(program *tea.Program) error
	quitting      bool
	cancelled     bool // Track if operation was cancelled
	err           error
	program       *tea.Program
	statusReport  *StatusReportData
	cancelChan    chan struct{} // Channel to signal cancellation
	opDone        chan struct{}
}

// Message types for communication
type ItemUpdateMsg struct {
	Index        int
	Name         string // Optional: update the item name
	Status       string
	Message      string
	ErrorMessage string // Brief error message for failed items
	Variants     []string
	Scope        string
	SourceName   string
}

type ProgressUpdateMsg struct {
	Percent float64
}

// TitleUpdateMsg updates the progress bar title dynamically
type TitleUpdateMsg struct {
	Title string
}

// TotalItemsUpdateMsg updates the total items count dynamically
type TotalItemsUpdateMsg struct {
	TotalItems int
}

type operationCompleteMsg struct {
	err error
}

type quitMsg struct{}

// StatusReportData represents status report data for the progress display
type StatusReportData struct {
	Success      int
	Skipped      int
	Failed       int
	SuccessLabel string
	SkippedLabel string
	FailedLabel  string
}

type StatusReportMsg struct {
	Report StatusReportData
}

// operationTickMsg is sent to update the progress display
type operationTickMsg time.Time

// NewProgressBar creates a new progress bar model
func NewProgressBar(title string, items []OperationItem, verboseMode bool, debugMode bool) *ProgressBarModel {
	// Create progress bar with gradient colors
	startColor, endColor := ui.GetProgressBarGradient()
	prog := progress.New(
		progress.WithGradient(startColor, endColor),
	)
	prog.Width = 30 // Match design width

	// Create spinner (no style - we'll apply it ourselves when rendering)
	spin := spinner.New()
	spin.Spinner = spinner.Dot

	return &ProgressBarModel{
		Title:       title,
		TotalItems:  len(items),
		Items:       items,
		VerboseMode: verboseMode,
		DebugMode:   debugMode,
		ProgressBar: prog,
		Spinner:     spin,
		cancelChan:  make(chan struct{}),
		opDone:      make(chan struct{}),
	}
}

func (m ProgressBarModel) Init() tea.Cmd {
	return tea.Batch(
		operationTickCmd(),
		m.Spinner.Tick,
		m.startOperation(),
	)
}

// startOperation runs the actual work in background
func (m ProgressBarModel) startOperation() tea.Cmd {
	return func() tea.Msg {
		go func() {
			defer func() {
				if m.opDone != nil {
					select {
					case <-m.opDone:
					default:
						close(m.opDone)
					}
				}
			}()
			err := m.operationFunc(m.program)
			select {
			case <-m.cancelChan:
				err = shared.ErrOperationCancelled
			default:
			}
			if m.program != nil {
				m.program.Send(operationCompleteMsg{err: err})
			}
		}()
		return nil
	}
}

func (m ProgressBarModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		// Handle cancellation keys (q, esc, ctrl+c, enter, space) - like sources update
		switch key {
		case "q", "ctrl+c", "esc", "enter", " ":
			if m.quitting {
				// If already completed, any key quits
				return m, tea.Quit
			} else {
				m.quitting = true
				m.cancelled = true
				m.err = shared.ErrOperationCancelled
				if m.cancelChan != nil {
					select {
					case <-m.cancelChan:
					default:
						close(m.cancelChan)
					}
				}
				return m, waitForOperationQuit(m.opDone)
			}
		}
		// If operation is complete, any key press should quit
		if m.quitting {
			return m, tea.Quit
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.ProgressBar.Width = msg.Width - 8
		if m.ProgressBar.Width > 80 {
			m.ProgressBar.Width = 80
		}
		return m, nil

	case ItemUpdateMsg:
		// Update item status
		if msg.Index >= 0 && msg.Index < len(m.Items) {
			if msg.Name != "" {
				m.Items[msg.Index].Name = msg.Name
			}
			m.Items[msg.Index].Status = msg.Status
			m.Items[msg.Index].StatusMessage = msg.Message
			if msg.ErrorMessage != "" {
				m.Items[msg.Index].ErrorMessage = msg.ErrorMessage
			}
			if msg.Variants != nil {
				m.Items[msg.Index].Variants = msg.Variants
			}
			if msg.Scope != "" {
				m.Items[msg.Index].Scope = msg.Scope
			}
			if msg.SourceName != "" {
				m.Items[msg.Index].SourceName = msg.SourceName
			}
		}
		return m, nil

	case ProgressUpdateMsg:
		// Don't process progress updates after operation completes
		if m.quitting {
			return m, nil
		}
		// Update progress bar
		cmd := m.ProgressBar.SetPercent(msg.Percent / 100.0)
		return m, cmd

	case TitleUpdateMsg:
		// Update the title dynamically
		m.Title = msg.Title
		return m, nil

	case TotalItemsUpdateMsg:
		// Update the total items count dynamically
		m.TotalItems = msg.TotalItems
		// If we're adding items, we may need to expand the Items slice
		if msg.TotalItems > len(m.Items) {
			// Expand items slice with pending items
			for len(m.Items) < msg.TotalItems {
				m.Items = append(m.Items, OperationItem{
					Name:   "",
					Status: "pending",
				})
			}
		}
		return m, nil

	case StatusReportMsg:
		m.statusReport = &msg.Report
		return m, nil

	case operationCompleteMsg:
		// Ignore completion messages if we've already quit (interrupted)
		if m.quitting {
			return m, nil
		}
		m.quitting = true
		m.err = msg.err
		// Ensure progress is at 100% when operation completes
		cmd := m.ProgressBar.SetPercent(1.0)
		// For progress bars without items, quit immediately (no delay needed)
		// For progress bars with items, show final state briefly before quitting
		if m.TotalItems == 0 {
			// No items to show - set progress and quit after processing one frame
			// The frame message will update the progress bar, then we'll quit
			return m, cmd
		}
		// Show final state with items, then quit after a brief delay
		// Reduced from 2s to 300ms for better responsiveness
		return m, tea.Batch(
			cmd,
			tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg {
				return quitMsg{}
			}),
		)

	case quitMsg:
		// Handle explicit quit message
		return m, tea.Quit

	case operationTickMsg:
		if m.quitting {
			// Stop tick timer when quitting
			return m, nil
		}
		// Keep the progress bar animating by continuously updating it
		// This is what makes the progress bar animate smoothly
		cmd := m.ProgressBar.SetPercent(m.ProgressBar.Percent())
		// Schedule next tick and process progress bar update
		return m, tea.Batch(operationTickCmd(), cmd)

	case progress.FrameMsg:
		// Handle progress bar animation
		progressModel, cmd := m.ProgressBar.Update(msg)
		m.ProgressBar = progressModel.(progress.Model)
		// If we're quitting and have no items, quit after processing this frame
		if m.quitting && m.TotalItems == 0 {
			return m, tea.Quit
		}
		return m, cmd

	case spinner.TickMsg:
		// Handle spinner animation
		spinnerModel, cmd := m.Spinner.Update(msg)
		m.Spinner = spinnerModel
		return m, cmd

	default:
		return m, nil
	}
}

func (m ProgressBarModel) View() string {
	// Always show the progress bar, don't hide it

	var b strings.Builder

	// Title with count - count all items that are done (completed, failed, or skipped)
	completed := 0
	for _, item := range m.Items {
		if item.Status == "completed" || item.Status == "failed" || item.Status == "skipped" {
			completed++
		}
	}

	// Build inline title with progress bar (skip in verbose/debug mode)
	if !m.VerboseMode && !m.DebugMode {
		// Format: "Title (X of Y) [████████░░] 50%" - plain text except for progress bar gradient
		// Hide count text when TotalItems is 0 (used for simple progress bars without item tracking)
		titleText := m.Title
		progressBar := m.renderInlineProgressBar()

		// Check if any items will be displayed (not pending and not empty name)
		hasDisplayableItems := false
		for _, item := range m.Items {
			if item.Status != "pending" && item.Name != "" {
				hasDisplayableItems = true
				break
			}
		}

		var titleLine string
		// Always show count text to prevent layout jumping when count is updated
		// Show "(0 of 0)" as placeholder when TotalItems is 0
		countText := fmt.Sprintf("(%d of %d)", completed, m.TotalItems)
		if hasDisplayableItems {
			titleLine = fmt.Sprintf("%s %s %s\n\n", titleText, countText, progressBar)
		} else {
			// No items to display - only one newline to avoid extra blank line
			titleLine = fmt.Sprintf("%s %s %s\n", titleText, countText, progressBar)
		}

		// Combine into single line - no styling on text, only progress bar has gradient
		// Title ends with \n\n to create blank line before items (if items will be shown), otherwise just \n
		// No leading \n - commands already start with a blank line per spacing framework
		b.WriteString(titleLine)
	} else {
		// For verbose/debug, don't show title line at all (redundant with verbose output)
		// No newline here - let the command handle spacing to avoid double spacing
	}

	// Items - show only items that have started (not pending) for line-by-line streaming display
	for _, item := range m.Items {
		// Skip pending items - only show items that have started or completed
		if item.Status == "pending" {
			continue
		}
		// Skip items with empty names - these are used for count tracking only
		if item.Name == "" {
			continue
		}

		// Get icon based on status - ensure all icons are the same width for alignment
		var styledIcon string
		switch item.Status {
		case "completed":
			styledIcon = ui.SuccessText.Render("✓")
		case "failed":
			styledIcon = ui.ErrorText.Render("✗")
		case "skipped":
			// Skipped items also use checkmark but green (since font is installed)
			styledIcon = ui.SuccessText.Render("✓")
		case "in_progress":
			// Use spinner for in_progress status (unless in verbose/debug mode which uses static display)
			if m.VerboseMode || m.DebugMode {
				styledIcon = "○" // Static circle for verbose/debug mode
			} else {
				// Get spinner character, trim whitespace, and apply styling using theme color
				// For system theme (empty color), use NoColor to respect terminal defaults
				styledIcon = strings.TrimSpace(m.Spinner.View())
				var spinnerColor lipgloss.TerminalColor
				if ui.SpinnerColor == "" {
					spinnerColor = lipgloss.NoColor{}
				} else {
					spinnerColor = lipgloss.Color(ui.SpinnerColor)
				}
				styledIcon = lipgloss.NewStyle().Foreground(spinnerColor).Render(styledIcon)
			}
		default:
			// Other statuses - use a space to maintain alignment
			styledIcon = " "
		}

		// Format font name (plain text, no styling)
		fontName := item.Name

		// Format source name in brackets with purple color (if available)
		var sourcePart string
		if item.SourceName != "" {
			sourcePart = " " + ui.InfoText.Render(fmt.Sprintf("[%s]", item.SourceName))
		}

		// Format the status message
		var statusText string
		switch item.Status {
		case "completed":
			// For install operations: show "Installed" (no scope info)
			// For remove operations: show "Removed from <scope>" (scope info needed)
			actionWord := item.StatusMessage
			if actionWord == "" {
				actionWord = "Installed" // Default fallback
			}
			// Determine the preposition based on the action
			preposition := "to"
			if strings.EqualFold(actionWord, "Removed") {
				preposition = "from"
				// For remove operations, include scope
				if item.Scope != "" {
					statusText = fmt.Sprintf("%s %s %s", actionWord, preposition, item.Scope)
				} else {
					statusText = actionWord
				}
			} else {
				// For install operations, don't show scope (cleaner output)
				statusText = actionWord
			}
		case "skipped":
			// Show "Skipped... already installed" (no scope info for cleaner output)
			statusText = "Skipped... already installed"
		case "failed":
			// Show full error message in error color (if available), otherwise show generic message
			if item.ErrorMessage != "" {
				statusText = ui.ErrorText.Render(item.ErrorMessage)
			} else {
				// Fallback to generic message
				statusText = "Installation failed"
			}
		case "in_progress":
			// Use status message if available, otherwise default
			if item.StatusMessage != "" {
				statusText = item.StatusMessage
			} else {
				statusText = "Installing..."
			}
		default:
			// Other statuses
			if item.StatusMessage != "" {
				statusText = item.StatusMessage
			} else {
				statusText = "Pending..."
			}
		}

		// Format the font item: "  ✓ Font Name [Source] - Status text"
		// Counter is shown in the progress bar title, not on individual items
		b.WriteString(fmt.Sprintf("  %s %s%s - %s\n", styledIcon, fontName, sourcePart, statusText))

		// Show variants if verbose mode is enabled
		if m.VerboseMode && len(item.Variants) > 0 {
			for _, variant := range item.Variants {
				b.WriteString(fmt.Sprintf("      ↳ %s\n", variant))
			}
		}
	}

	// Progress bar output ends with \n for spacing before next message
	// Check if any items were actually displayed (not just empty placeholder items)
	hasDisplayedItems := false
	for _, item := range m.Items {
		if item.Status != "pending" && item.Name != "" {
			hasDisplayedItems = true
			break
		}
	}

	// Always add a trailing newline to create blank line before next message
	// (either after items if displayed, or after progress bar if no items)
	if hasDisplayedItems {
		b.WriteString("\n")
	} else {
		// No items displayed, but we still want a blank line after the progress bar
		// The titleLine already has one \n, so add another to create blank line
		b.WriteString("\n")
	}

	return b.String()
}

// InlineProgressBarView renders the same gradient block + percent label used by the CLI progress UI.
// percent0to100 is clamped to [0, 100]; barCharWidth is the number of █/░ cells inside the brackets.
func InlineProgressBarView(percent0to100 float64, barCharWidth int) string {
	if barCharWidth < 1 {
		barCharWidth = 1
	}
	p := percent0to100
	if p < 0 {
		p = 0
	}
	if p > 100 {
		p = 100
	}
	filled := int(float64(barCharWidth) * (p / 100.0))
	if filled > barCharWidth {
		filled = barCharWidth
	}
	empty := barCharWidth - filled

	startColor, endColor := ui.GetProgressBarGradient()
	var barBuilder strings.Builder
	for i := 0; i < filled; i++ {
		var ratio float64
		if filled > 1 {
			ratio = float64(i) / float64(filled-1)
		}
		if ratio > 1.0 {
			ratio = 1.0
		}
		if ratio < 0.0 {
			ratio = 0.0
		}
		gradientColor := interpolateHexColor(startColor, endColor, ratio)
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(gradientColor))
		barBuilder.WriteString(style.Render("█"))
	}
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086")) // Overlay 0 - gray
	for i := 0; i < empty; i++ {
		barBuilder.WriteString(emptyStyle.Render("░"))
	}
	barVisual := barBuilder.String()
	percentText := fmt.Sprintf("%.0f%%", p)
	return fmt.Sprintf("[%s] %s", barVisual, percentText)
}

// renderInlineProgressBar creates a compact progress bar for inline display
func (m ProgressBarModel) renderInlineProgressBar() string {
	barWidth := 15
	if m.ProgressBar.Width < 30 {
		barWidth = 10
	}
	return InlineProgressBarView(m.ProgressBar.Percent()*100, barWidth)
}

// interpolateHexColor interpolates between two hex colors
func interpolateHexColor(start, end string, ratio float64) string {
	// Parse hex colors to RGB
	startRGB := hexToRGB(start)
	endRGB := hexToRGB(end)

	// Interpolate each component
	r := int(float64(startRGB[0])*(1-ratio) + float64(endRGB[0])*ratio)
	g := int(float64(startRGB[1])*(1-ratio) + float64(endRGB[1])*ratio)
	b := int(float64(startRGB[2])*(1-ratio) + float64(endRGB[2])*ratio)

	// Convert back to hex
	return rgbToHex(r, g, b)
}

// hexToRGB converts a hex color string to RGB values
func hexToRGB(hex string) [3]int {
	// Remove # if present
	hex = strings.TrimPrefix(hex, "#")

	// Parse hex string
	var r, g, b int
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)

	return [3]int{r, g, b}
}

// rgbToHex converts RGB values to a hex color string
func rgbToHex(r, g, b int) string {
	// Clamp values to valid range
	if r < 0 {
		r = 0
	}
	if r > 255 {
		r = 255
	}
	if g < 0 {
		g = 0
	}
	if g > 255 {
		g = 255
	}
	if b < 0 {
		b = 0
	}
	if b > 255 {
		b = 255
	}

	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// operationTickCmd returns a command that sends a tick message after 1 second
func operationTickCmd() tea.Cmd {
	return tea.Tick(time.Second*1, func(t time.Time) tea.Msg {
		return operationTickMsg(t)
	})
}

func waitForOperationQuit(done <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		if done != nil {
			<-done
		}
		return quitMsg{}
	}
}

// UseInteractiveRenderer is true only when both stdin and stdout are terminals.
func UseInteractiveRenderer() bool {
	return term.IsTerminal(os.Stdin.Fd()) && term.IsTerminal(os.Stdout.Fd())
}

// RunProgressBar runs the progress display with the given operation.
// Interactive Bubble Tea is used only when stdin and stdout are terminals and debug is off.
func RunProgressBar(title string, items []OperationItem, verboseMode bool, debugMode bool, operation func(send func(msg tea.Msg), cancelChan <-chan struct{}) error) error {
	if debugMode || !UseInteractiveRenderer() {
		return runPlainProgressBar(items, operation)
	}
	return runInteractiveProgressBar(title, items, verboseMode, debugMode, operation)
}

func runPlainProgressBar(items []OperationItem, operation func(send func(msg tea.Msg), cancelChan <-chan struct{}) error) error {
	cancelChan := make(chan struct{})
	send := func(msg tea.Msg) {
		update, ok := msg.(ItemUpdateMsg)
		if !ok || update.Index < 0 || update.Index >= len(items) {
			return
		}
		name := items[update.Index].Name
		if update.Name != "" {
			name = update.Name
		}
		switch update.Status {
		case "failed":
			if update.ErrorMessage != "" {
				fmt.Printf("%s: failed: %s\n", name, update.ErrorMessage)
			} else {
				fmt.Printf("%s: failed\n", name)
			}
		case "completed":
			fmt.Printf("%s: installed\n", name)
		case "skipped":
			fmt.Printf("%s: skipped\n", name)
		}
	}
	return operation(send, cancelChan)
}

func runInteractiveProgressBar(title string, items []OperationItem, verboseMode bool, debugMode bool, operation func(send func(msg tea.Msg), cancelChan <-chan struct{}) error) error {
	model := NewProgressBar(title, items, verboseMode, debugMode)
	p := tea.NewProgram(model)
	model.program = p
	model.operationFunc = func(program *tea.Program) error {
		return operation(func(msg tea.Msg) {
			select {
			case <-model.cancelChan:
				return
			default:
			}
			program.Send(msg)
		}, model.cancelChan)
	}

	finalModel, err := p.Run()
	if err != nil {
		select {
		case <-model.cancelChan:
		default:
			close(model.cancelChan)
		}
		if model.opDone != nil {
			<-model.opDone
		}
		return err
	}

	if m, ok := finalModel.(ProgressBarModel); ok {
		if m.cancelled {
			return shared.ErrOperationCancelled
		}
		if m.err != nil {
			return m.err
		}
	}
	return nil
}
