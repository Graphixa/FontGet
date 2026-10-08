package components

import (
	"fmt"
	"strings"
	"testing"
)

func TestFinishedItemsLeaveTheLiveView(t *testing.T) {
	const n = 40
	items := make([]OperationItem, n)
	for i := range items {
		items[i] = OperationItem{Name: fmt.Sprintf("Font %d", i), Status: "pending", SourceName: "FontGet Backup"}
	}
	m := *NewProgressBar("Installing fonts", items, false, false)
	for i := range items {
		next, cmd := m.Update(ItemUpdateMsg{Index: i, Status: "completed", Message: "Installed"})
		m = next.(ProgressBarModel)
		if cmd == nil {
			t.Fatalf("item %d: expected scrollback print", i)
		}
	}
	view := m.View()
	for i := range items {
		if strings.Contains(view, fmt.Sprintf("Font %d", i)) {
			t.Fatalf("Font %d still in live view:\n%s", i, view)
		}
	}
	_, cmd := m.Update(ItemUpdateMsg{Index: 0, Status: "completed", Message: "Installed"})
	if cmd != nil {
		t.Fatal("duplicate print")
	}
}

func TestInProgressStaysInLiveView(t *testing.T) {
	m := *NewProgressBar("Installing fonts", []OperationItem{{
		Name: "Walkway", Status: "pending", SourceName: "FontGet Backup",
	}}, false, false)
	next, cmd := m.Update(ItemUpdateMsg{Index: 0, Status: "in_progress", Message: "Installing..."})
	m = next.(ProgressBarModel)
	if cmd != nil {
		t.Fatal("in-progress row should stay in the live view")
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
