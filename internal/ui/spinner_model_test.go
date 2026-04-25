package ui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSpinner_NoMinDelayWhenDoneMsgEmpty(t *testing.T) {
	m := NewSpinnerModel("Loading...", "", func() error { return nil })
	// Simulate a fast operation so we're within the default min display window.
	m.startTime = time.Now()
	m.minDisplayMs = 2500

	_, cmd := m.Update(operationCompleteMsg{err: nil})
	if cmd == nil {
		t.Fatalf("expected quit command, got nil")
	}

	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %T", msg)
	}
}

