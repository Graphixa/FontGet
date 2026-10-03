package ui

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
)

// CSI ED default (0): erase from cursor through end of display.
// Appended after the frame so a shorter view clears leftover rows from a prior resize
// without padding blank lines below content (lipgloss Height would add those).
const eraseDisplayBelowCursor = "\x1b[J"

// FillTerminalArea constrains the view to the terminal width, truncates if taller than
// height, and clears everything below the drawn frame on the alternate screen.
func FillTerminalArea(view string, width, height int) string {
	if width <= 0 || height <= 0 {
		return view
	}
	out := lipgloss.NewStyle().
		Width(width).
		MaxWidth(width).
		MaxHeight(height).
		Render(view)
	return out + eraseDisplayBelowCursor
}

// RunSpinner runs a bubbletea spinner while the provided function executes
// Always stops with a green check symbol on success
// If doneMsg is empty string, the spinner line will be cleared (hidden) after completion
func RunSpinner(msg, doneMsg string, fn func() error) error {
	// tea.NewProgram hangs when stdout is not a TTY (pipes, redirected output).
	// Skip the spinner in that case; keep it on an interactive terminal.
	if !term.IsTerminal(os.Stdout.Fd()) {
		if fn == nil {
			return nil
		}
		return fn()
	}
	model := NewSpinnerModel(msg, doneMsg, fn)
	program := tea.NewProgram(model)

	// Store program reference so goroutine can send completion message
	model.program = program

	// Run the program
	finalModel, err := program.Run()
	if err != nil {
		return err
	}

	// Extract error from model if operation failed
	if m, ok := finalModel.(*spinnerModel); ok {
		if m.err != nil {
			return m.err
		}
	}

	return nil
}
