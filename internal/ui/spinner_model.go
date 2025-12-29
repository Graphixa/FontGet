package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// spinnerModel is a minimal bubbletea spinner model for blocking operations
type spinnerModel struct {
	spinner      spinner.Model
	message      string
	doneMsg      string
	err          error
	quitting     bool
	operation    func() error
	program      *tea.Program
	startTime    time.Time
	minDisplayMs int // Minimum display time in milliseconds
}

// operationCompleteMsg signals that the operation has completed
type operationCompleteMsg struct {
	err error
}

// quitAfterDelayMsg signals that minimum display time has passed
type quitAfterDelayMsg struct{}

// NewSpinnerModel creates a new spinner model
func NewSpinnerModel(msg, doneMsg string, fn func() error) *spinnerModel {
	spin := spinner.New()
	spin.Spinner = spinner.Dot

	// Apply theme color to spinner
	var spinnerColor lipgloss.TerminalColor
	if SpinnerColor == "" {
		spinnerColor = lipgloss.NoColor{}
	} else {
		spinnerColor = lipgloss.Color(SpinnerColor)
	}
	spin.Style = lipgloss.NewStyle().Foreground(spinnerColor)

	return &spinnerModel{
		spinner:      spin,
		message:      msg,
		doneMsg:      doneMsg,
		operation:    fn,
		minDisplayMs: 2500, // Minimum 2.5s display time so spinner is visible even for fast operations
	}
}

// Init starts the spinner
func (m *spinnerModel) Init() tea.Cmd {
	m.startTime = time.Now()
	return tea.Batch(
		m.spinner.Tick,
		m.startOperation(),
	)
}

// startOperation runs the actual work in background
func (m *spinnerModel) startOperation() tea.Cmd {
	return func() tea.Msg {
		// Run the operation in a goroutine to avoid blocking
		go func() {
			err := m.operation()
			// Send completion message to program
			// Note: program reference should be set by the time this executes,
			// but add a small safety check with retry
			if m.program != nil {
				m.program.Send(operationCompleteMsg{err: err})
			} else {
				// Program not set yet (shouldn't happen, but safety check)
				// Wait a tiny bit and try again
				time.Sleep(10 * time.Millisecond)
				if m.program != nil {
					m.program.Send(operationCompleteMsg{err: err})
				}
			}
		}()
		return nil
	}
}

// Update handles messages
func (m *spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Allow cancellation with Ctrl+C
		if msg.Type == tea.KeyCtrlC {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		// If operation is done, quit
		if m.quitting {
			return m, tea.Quit
		}
		return m, cmd

	case operationCompleteMsg:
		// Operation completed - ensure minimum display time
		m.err = msg.err
		elapsed := time.Since(m.startTime)
		minDisplay := time.Duration(m.minDisplayMs) * time.Millisecond

		if elapsed < minDisplay {
			// Wait for minimum display time before quitting
			remaining := minDisplay - elapsed
			return m, tea.Tick(remaining, func(time.Time) tea.Msg {
				return quitAfterDelayMsg{}
			})
		}

		// Already displayed long enough, quit immediately
		m.quitting = true
		return m, tea.Quit

	case quitAfterDelayMsg:
		// Minimum display time has passed, now quit
		m.quitting = true
		return m, tea.Quit

	default:
		return m, nil
	}
}

// View renders the spinner
func (m *spinnerModel) View() string {
	if m.quitting {
		// Operation complete - show result
		if m.err != nil {
			// Show error with red X
			return fmt.Sprintf("\r%s %s\n", ErrorText.Render("✗"), ErrorText.Render(m.err.Error()))
		}

		// Success - show done message or clear line
		if m.doneMsg == "" {
			// Clear the line (same behavior as pin package)
			return "\r\033[2K\n"
		}

		// Show success message with green checkmark
		// Apply done color if available
		var doneColor lipgloss.TerminalColor
		if SpinnerDoneColor == "" {
			doneColor = lipgloss.NoColor{}
		} else {
			doneColor = lipgloss.Color(SpinnerDoneColor)
		}
		styledCheckmark := lipgloss.NewStyle().Foreground(doneColor).Render("✓")
		return fmt.Sprintf("\r%s %s\n", styledCheckmark, m.doneMsg)
	}

	// Show spinner with message
	spinnerView := m.spinner.View()
	return fmt.Sprintf("\r%s %s", spinnerView, m.message)
}
