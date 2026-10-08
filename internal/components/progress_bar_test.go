package components

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestProgressBarStaysAboveItems(t *testing.T) {
	const n = 10
	items := make([]OperationItem, n)
	for i := range items {
		items[i] = OperationItem{Name: fmt.Sprintf("Font %d", i), Status: "pending", SourceName: "FontGet Backup"}
	}
	m := *NewProgressBar("Installing fonts", items, false, false)
	m.height = 40
	m.width = 80
	for i := range items {
		next, cmd := m.Update(ItemUpdateMsg{Index: i, Status: "completed", Message: "Installed"})
		m = next.(ProgressBarModel)
		m.height = 40
		m.width = 80
		if cmd != nil {
			t.Fatalf("item %d: unexpected cmd", i)
		}
	}
	view := m.View()
	if !strings.HasPrefix(view, "Installing fonts") {
		t.Fatalf("bar should be line 1:\n%s", view)
	}
	if !strings.Contains(view, "Font 0") || !strings.Contains(view, "Font 9") {
		t.Fatalf("all rows should fit:\n%s", view)
	}
}

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
		m.height = 12
		m.width = 60
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
