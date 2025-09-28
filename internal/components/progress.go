package components

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"fontget/internal/ui"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	maxWidth = 80
)

// tickMsg is sent to update the progress display
type tickMsg time.Time

// Model represents the progress bar component
type Model struct {
	progress    progress.Model
	header      string
	currentStep *int64 // Use atomic counter for thread-safe access
	totalSteps  int
	completed   bool
	err         error
	callback    func(updateProgress func()) error
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewProgressModel creates a new progress bar model
func NewProgressModel(header string, totalSteps int, callback func(updateProgress func()) error) Model {
	ctx, cancel := context.WithCancel(context.Background())
	currentStep := int64(0)

	// Get gradient colors from the centralized styles
	startColor, endColor := ui.GetProgressBarGradient()
	prog := progress.New(
		progress.WithGradient(startColor, endColor),
	)

	return Model{
		progress:    prog,
		header:      header,
		currentStep: &currentStep,
		totalSteps:  totalSteps,
		callback:    callback,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Init starts the progress bar
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),       // Start the ticker
		m.runCallback(), // Start the background work
	)
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.cancel()
			return m, tea.Quit
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - 8 // Account for container padding and margins
		if m.progress.Width > maxWidth {
			m.progress.Width = maxWidth
		}
		return m, nil

	case tickMsg:
		if m.completed {
			return m, tea.Quit
		}

		// Get current progress atomically
		currentStep := atomic.LoadInt64(m.currentStep)

		// Check if we're done
		if int(currentStep) >= m.totalSteps {
			m.completed = true
			return m, tea.Quit
		}

		// Update progress bar with current value
		percent := float64(currentStep) / float64(m.totalSteps)
		cmd := m.progress.SetPercent(percent)
		return m, tea.Batch(tickCmd(), cmd)

	// FrameMsg is sent when the progress bar wants to animate itself
	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

	default:
		return m, nil
	}
}

// View renders the progress bar
func (m Model) View() string {
	if m.completed && m.err == nil {
		return ""
	}

	// Get current progress atomically for display
	currentStep := atomic.LoadInt64(m.currentStep)
	progressText := fmt.Sprintf("(%d/%d)", currentStep, m.totalSteps)

	// Build the view using the centralized styles
	var result strings.Builder

	// Header with proper styling
	result.WriteString("\n")
	result.WriteString(ui.ProgressBarHeader.Render(m.header))
	result.WriteString("\n")

	// Progress bar with container styling
	progressLine := ui.ProgressBarContainer.Render(
		m.progress.View() + " " + ui.ProgressBarText.Render(progressText),
	)
	result.WriteString(progressLine)
	result.WriteString("\n")

	// Error message if any
	if m.err != nil {
		result.WriteString(ui.RenderError(m.err.Error()))
		result.WriteString("\n")
	}

	return result.String()
}

// tickCmd returns a command that sends a tick message
func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*50, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// runCallback runs the user's callback function in a goroutine
func (m Model) runCallback() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		// Create the update function that the callback can call
		updateProgress := func() {
			// Atomically increment the counter
			newStep := atomic.AddInt64(m.currentStep, 1)

			// Check if we're done
			if int(newStep) >= m.totalSteps {
				m.completed = true
			}
		}

		// Run the callback
		if err := m.callback(updateProgress); err != nil {
			m.err = err
		}

		m.completed = true
		return nil
	})
}

// RunWithProgress runs a progress bar
func RunWithProgress(header string, totalSteps int, callback func(updateProgress func()) error) error {
	model := NewProgressModel(header, totalSteps, callback)

	// Create and run the Bubble Tea program
	program := tea.NewProgram(model)

	finalModel, err := program.Run()
	if err != nil {
		return fmt.Errorf("failed to run progress bar: %w", err)
	}

	// Check if there was an error in the final model
	if m, ok := finalModel.(Model); ok && m.err != nil {
		return m.err
	}

	return nil
}
