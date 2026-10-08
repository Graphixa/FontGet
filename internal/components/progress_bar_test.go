package components

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestClipsOldestItemsUnderTheBar(t *testing.T) {
	const n = 40
	items := make([]OperationItem, n)
	for i := range items {
		items[i] = OperationItem{Name: fmt.Sprintf("Font %d", i), Status: "pending"}
	}
	m := *NewProgressBar("Installing fonts", items, false, false)
	m.height = 12
	m.width = 60
	for i := range items {
		next, cmd := m.Update(ItemUpdateMsg{Index: i, Status: "completed", Message: "Installed"})
		m = next.(ProgressBarModel)
		if cmd != nil {
			t.Fatalf("item %d: no scrollback Println", i)
		}
	}
	view := m.View()
	if !strings.HasPrefix(view, "Installing fonts") {
		t.Fatalf("bar should be line 1:\n%s", view)
	}
	if strings.Contains(view, "Font 0") {
		t.Fatalf("oldest should be clipped:\n%s", view)
	}
	if !strings.Contains(view, "Font 39") {
		t.Fatalf("newest missing:\n%s", view)
	}
	max := m.height - viewportMargin
	if got := len(strings.Split(view, "\n")); got > max {
		t.Fatalf("view is %d lines, max %d", got, max)
	}
}

func TestClipStaysFastWithManyItems(t *testing.T) {
	const n = 400
	items := make([]OperationItem, n)
	for i := range items {
		items[i] = OperationItem{
			Name: fmt.Sprintf("Font %d", i), Status: "completed", StatusMessage: "Installed",
		}
	}
	m := *NewProgressBar("Installing fonts", items, false, false)
	m.height = 15
	m.width = 80
	// One View must not re-scan all 400 items on every drop (old O(n²) path).
	view := m.View()
	if !strings.HasPrefix(view, "Installing fonts") {
		t.Fatalf("bar missing:\n%s", view)
	}
	if strings.Contains(view, "Font 0") || !strings.Contains(view, "Font 399") {
		t.Fatalf("should keep newest only:\n%s", view)
	}
}

func TestClipsWithUnknownHeight(t *testing.T) {
	const n = 40
	items := make([]OperationItem, n)
	for i := range items {
		items[i] = OperationItem{
			Name: fmt.Sprintf("Font %d", i), Status: "completed", StatusMessage: "Installed",
		}
	}
	m := *NewProgressBar("Installing fonts", items, false, false)
	// height left at 0 — must still clip via defaultViewportHeight
	view := m.View()
	if strings.Contains(view, "Font 0") {
		t.Fatalf("fallback height should clip oldest:\n%s", view)
	}
	if !strings.Contains(view, "Font 39") {
		t.Fatalf("newest missing:\n%s", view)
	}
	max := defaultViewportHeight - viewportMargin
	if got := len(strings.Split(view, "\n")); got > max {
		t.Fatalf("view is %d lines, max %d", got, max)
	}
}

func TestZeroWindowSizeDoesNotUncap(t *testing.T) {
	const n = 40
	items := make([]OperationItem, n)
	for i := range items {
		items[i] = OperationItem{
			Name: fmt.Sprintf("Font %d", i), Status: "completed", StatusMessage: "Installed",
		}
	}
	m := *NewProgressBar("Installing fonts", items, false, false)
	m.height = 12
	m.width = 60
	next, _ := m.Update(tea.WindowSizeMsg{Width: 0, Height: 0})
	m = next.(ProgressBarModel)
	if m.height != 12 || m.width != 60 {
		t.Fatalf("zero resize wiped size: height=%d width=%d", m.height, m.width)
	}
	view := m.View()
	if strings.Contains(view, "Font 0") {
		t.Fatalf("still must clip after zero resize:\n%s", view)
	}
}

func TestLiveLinesDoNotWrap(t *testing.T) {
	m := *NewProgressBar("Installing fonts", []OperationItem{{
		Name: "Walkway", Status: "completed", StatusMessage: "Installed",
		SourceName: "FontGet Backup",
	}}, false, false)
	m.height = 20
	m.width = 40
	for _, line := range strings.Split(m.View(), "\n") {
		if ansi.StringWidth(line) >= m.width {
			t.Fatalf("line wraps: %q", line)
		}
	}
}

func TestInProgressStaysInLiveView(t *testing.T) {
	m := *NewProgressBar("Installing fonts", []OperationItem{{
		Name: "Walkway", Status: "pending", SourceName: "FontGet Backup",
	}}, false, false)
	m.height = 20
	m.width = 80
	next, cmd := m.Update(ItemUpdateMsg{Index: 0, Status: "in_progress", Message: "Installing..."})
	m = next.(ProgressBarModel)
	if cmd != nil {
		t.Fatal("unexpected cmd")
	}
	if !strings.Contains(m.View(), "Walkway") {
		t.Fatalf("view:\n%s", m.View())
	}
}

func TestItemBlockIncludesVerboseVariants(t *testing.T) {
	m := NewProgressBar("Installing fonts", nil, true, false)
	block := m.itemBlock(OperationItem{
		Name: "Walkway", Status: "completed", StatusMessage: "Installed",
		Variants: []string{"Regular"},
	})
	if !strings.Contains(block, "Walkway") || !strings.Contains(block, "↳ Regular") {
		t.Fatalf("block:\n%s", block)
	}
}
