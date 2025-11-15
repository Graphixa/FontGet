package components

import (
	"fmt"
	"strings"
	"time"

	"fontget/internal/ui"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	ListMode      bool // Show file/variant listings (add/remove commands)
	VerboseMode   bool // Show operational details (static display, no spinner)
	DebugMode     bool // Show technical details (static display, no spinner)
	ProgressBar   progress.Model
	Spinner       spinner.Model
	operationFunc func(program *tea.Program) error
	quitting      bool
	err           error
	program       *tea.Program
	statusReport  *StatusReportData
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

type operationCompleteMsg struct {
	err error
}

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
func NewProgressBar(title string, items []OperationItem, listMode bool, verboseMode bool, debugMode bool) *ProgressBarModel {
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
		ListMode:    listMode,
		VerboseMode: verboseMode,
		DebugMode:   debugMode,
		ProgressBar: prog,
		Spinner:     spin,
	}
}

// Init implements tea.Model
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
		// Run the operation in a goroutine to avoid blocking
		go func() {
			err := m.operationFunc(m.program)
			m.program.Send(operationCompleteMsg{err: err})
		}()
		return nil
	}
}

// Update implements tea.Model
func (m ProgressBarModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
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
		// Update progress bar
		cmd := m.ProgressBar.SetPercent(msg.Percent / 100.0)
		return m, cmd

	case StatusReportMsg:
		m.statusReport = &msg.Report
		return m, nil

	case operationCompleteMsg:
		m.quitting = true
		m.err = msg.err
		// Show final state with variants, then quit after a delay
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return tea.Quit
		})

	case operationTickMsg:
		if m.quitting {
			return m, tea.Quit
		}
		// Keep the progress bar animating by continuously updating it
		// This is what makes the progress bar animate smoothly
		cmd := m.ProgressBar.SetPercent(m.ProgressBar.Percent())
		// Only send a new tick if not quitting
		if !m.quitting {
			return m, tea.Batch(operationTickCmd(), cmd)
		}
		return m, cmd

	case progress.FrameMsg:
		// Handle progress bar animation
		progressModel, cmd := m.ProgressBar.Update(msg)
		m.ProgressBar = progressModel.(progress.Model)
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

// View implements tea.Model
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
		titleText := m.Title
		countText := fmt.Sprintf("(%d of %d)", completed, m.TotalItems)
		progressBar := m.renderInlineProgressBar()

		// Combine into single line - no styling on text, only progress bar has gradient
		// Title ends with \n\n to create blank line before items, and last item ends with \n
		titleLine := fmt.Sprintf("\n%s %s %s\n\n", titleText, countText, progressBar)
		b.WriteString(titleLine)
	} else {
		// For verbose/debug, don't show title line at all (redundant with verbose output)
		// No newline here - let the command handle spacing to avoid double spacing
	}

	// Items - show ALL items (pending, in_progress, completed, failed)
	for _, item := range m.Items {
		// Get icon based on status - ensure all icons are the same width for alignment
		var styledIcon string
		switch item.Status {
		case "completed":
			styledIcon = ui.FeedbackSuccess.Render("✓")
		case "failed":
			styledIcon = ui.FeedbackError.Render("✗")
		case "skipped":
			// Skipped items also use checkmark but green (since font is installed)
			styledIcon = ui.FeedbackSuccess.Render("✓")
		case "in_progress":
			// Use spinner for in_progress status (unless in verbose/debug mode which uses static display)
			if m.VerboseMode || m.DebugMode {
				styledIcon = "○" // Static circle for verbose/debug mode
			} else {
				// Get spinner character, trim whitespace, and apply styling
				styledIcon = strings.TrimSpace(m.Spinner.View())
				styledIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("#cba6f7")).Render(styledIcon)
			}
		default:
			// Pending items - use a space to maintain alignment
			styledIcon = " "
		}

		// Format font name (plain text, no styling)
		fontName := item.Name

		// Format source name in brackets with purple color (if available)
		var sourcePart string
		if item.SourceName != "" {
			sourcePart = " " + ui.FeedbackInfo.Render(fmt.Sprintf("[%s]", item.SourceName))
		}

		// Format the status message
		var statusText string
		switch item.Status {
		case "completed":
			// Use the actual status message (e.g., "Installed" or "Removed")
			actionWord := item.StatusMessage
			if actionWord == "" {
				actionWord = "Installed" // Default fallback
			}
			// Determine the preposition based on the action
			preposition := "to"
			if strings.EqualFold(actionWord, "Removed") {
				preposition = "from"
			}
			if item.Scope != "" {
				statusText = fmt.Sprintf("%s %s %s", actionWord, preposition, item.Scope)
			} else {
				statusText = actionWord
			}
		case "skipped":
			if item.Scope != "" {
				statusText = fmt.Sprintf("already installed to %s", item.Scope)
			} else {
				statusText = "already installed"
			}
		case "failed":
			// Show brief error message in error color (if available), otherwise show generic message
			if item.ErrorMessage != "" {
				// Extract brief error - take first sentence or reasonable length
				errorMsg := item.ErrorMessage
				// Remove filename prefix if present (e.g., "Roboto-Regular.ttf could not..." -> "could not...")
				// Look for common patterns like ".ttf could" or ".otf could"
				if idx := strings.Index(errorMsg, " could"); idx > 0 {
					errorMsg = errorMsg[idx+1:] // Keep the space before "could"
				}
				// Try to get first sentence (up to period), or take up to 120 chars if no period
				if idx := strings.Index(errorMsg, "."); idx > 0 {
					errorMsg = errorMsg[:idx]
				} else if len(errorMsg) > 120 {
					// Only truncate if extremely long (120+ chars), and try to break at word boundary
					errorMsg = errorMsg[:120]
					if lastSpace := strings.LastIndex(errorMsg, " "); lastSpace > 100 {
						errorMsg = errorMsg[:lastSpace]
					}
				}
				// Capitalize first letter to match other status messages
				if len(errorMsg) > 0 {
					errorMsg = strings.ToUpper(string(errorMsg[0])) + errorMsg[1:]
				}
				// Apply error color styling (no bold) - FeedbackError already has the right color
				statusText = ui.FeedbackError.Render(errorMsg)
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
			// Pending or other status
			if item.StatusMessage != "" {
				statusText = item.StatusMessage
			} else {
				statusText = "Pending"
			}
		}

		// Format the font item: "  ✓ Font Name [Source] - Status text"
		b.WriteString(fmt.Sprintf("  %s %s%s - %s\n", styledIcon, fontName, sourcePart, statusText))

		// Show variants if list mode is enabled
		if m.ListMode && len(item.Variants) > 0 {
			for _, variant := range item.Variants {
				b.WriteString(fmt.Sprintf("      ↳ %s\n", variant))
			}
		}
	}

	// Progress bar output always ends with exactly one \n (from last item)
	// Commands are responsible for adding spacing after the progress bar as needed

	return b.String()
}

// renderInlineProgressBar creates a compact progress bar for inline display
func (m ProgressBarModel) renderInlineProgressBar() string {
	percent := m.ProgressBar.Percent()

	// Smaller width for inline display (15-20 chars)
	barWidth := 15
	if m.ProgressBar.Width < 30 {
		// If terminal is narrow, make bar even smaller
		barWidth = 10
	}

	// Calculate filled and empty portions
	filled := int(float64(barWidth) * percent)
	empty := barWidth - filled

	// Get gradient colors
	startColor, endColor := ui.GetProgressBarGradient()

	// Build the progress bar manually with gradient colors
	var barBuilder strings.Builder

	// Filled portion with gradient
	// Calculate gradient across the filled portion only (not the entire bar width)
	for i := 0; i < filled; i++ {
		// Calculate color interpolation for gradient effect
		// Ratio should be based on position within the filled portion
		var ratio float64
		if filled > 1 {
			ratio = float64(i) / float64(filled-1)
		} else {
			ratio = 0.0
		}
		// Clamp ratio to [0, 1]
		if ratio > 1.0 {
			ratio = 1.0
		}
		if ratio < 0.0 {
			ratio = 0.0
		}

		// Use lipgloss to create gradient color
		gradientColor := interpolateHexColor(startColor, endColor, ratio)
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(gradientColor))
		barBuilder.WriteString(style.Render("█"))
	}

	// Empty portion
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086")) // Overlay 0 - gray
	for i := 0; i < empty; i++ {
		barBuilder.WriteString(emptyStyle.Render("░"))
	}

	barVisual := barBuilder.String()
	percentText := fmt.Sprintf("%.0f%%", percent*100)

	return fmt.Sprintf("[%s] %s", barVisual, percentText)
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

// RunProgressBar runs the progress display with the given operation
func RunProgressBar(title string, items []OperationItem, listMode bool, verboseMode bool, debugMode bool, operation func(send func(msg tea.Msg)) error) error {
	// Initialize the model
	model := NewProgressBar(title, items, listMode, verboseMode, debugMode)

	// Create the Bubble Tea program
	p := tea.NewProgram(model)

	// Store the program reference so operation can send messages
	model.program = p

	// Wrap the operation to work with the program
	model.operationFunc = func(program *tea.Program) error {
		// Call the operation with a send function that uses program.Send
		return operation(func(msg tea.Msg) {
			program.Send(msg)
		})
	}

	// Run the program
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	// Check if there was an operation error
	if m, ok := finalModel.(ProgressBarModel); ok && m.err != nil {
		return m.err
	}

	return nil
}
