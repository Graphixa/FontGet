package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func stripEraseSuffix(s string) string {
	return strings.TrimSuffix(s, eraseDisplayBelowCursor)
}

func TestFillTerminalAreaZeroPassthrough(t *testing.T) {
	in := "a\nb"
	if got := FillTerminalArea(in, 0, 24); got != in {
		t.Fatalf("width 0: got %q want unchanged", got)
	}
	if got := FillTerminalArea(in, 80, 0); got != in {
		t.Fatalf("height 0: got %q want unchanged", got)
	}
}

func TestFillTerminalAreaNoVerticalPadAndTruncate(t *testing.T) {
	short := "one line"
	out := FillTerminalArea(short, 30, 20)
	core := stripEraseSuffix(out)
	if h := lipgloss.Height(core); h != 1 {
		t.Fatalf("short view should stay 1 line high, got %d", h)
	}
	if !strings.HasSuffix(out, eraseDisplayBelowCursor) {
		t.Fatal("expected erase-display suffix")
	}

	long := strings.TrimSuffix(strings.Repeat("x\n", 12), "\n")
	out = FillTerminalArea(long, 10, 5)
	core = stripEraseSuffix(out)
	if h := lipgloss.Height(core); h != 5 {
		t.Fatalf("truncate height: got %d want 5", h)
	}
}
