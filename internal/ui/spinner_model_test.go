package ui

import (
	"errors"
	"strings"
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

func TestSpinner_CtrlCCancelsNotSuccess(t *testing.T) {
	m := NewSpinnerModel("Updating Sources...", "Sources Updated", func() error {
		time.Sleep(time.Hour)
		return nil
	})
	m.startTime = time.Now()

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	sm, ok := updated.(*spinnerModel)
	if !ok {
		t.Fatalf("expected *spinnerModel, got %T", updated)
	}
	if !sm.quitting {
		t.Fatal("expected quitting")
	}
	if !errors.Is(sm.err, ErrCancelled) {
		t.Fatalf("want ErrCancelled, got %v", sm.err)
	}
	view := sm.View()
	if strings.Contains(view, "Sources Updated") || strings.Contains(view, "✓") {
		t.Fatalf("cancel must not look like success: %q", view)
	}
	if !strings.Contains(view, "Cancelled") {
		t.Fatalf("want cancelled message, got %q", view)
	}
	if cmd == nil {
		t.Fatal("expected quit cmd")
	}
}

