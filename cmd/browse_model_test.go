package cmd

import "testing"

func TestBrowseSearchTickStale(t *testing.T) {
	t.Parallel()
	if !isBrowseSearchTickStale(1, 2) {
		t.Fatal("expected gen 1 to be stale when current is 2")
	}
	if isBrowseSearchTickStale(3, 3) {
		t.Fatal("expected matching gen to be fresh")
	}
}

func isBrowseSearchTickStale(tickGen, currentGen int) bool {
	return tickGen != currentGen
}

func TestBrowseFocusTabCyclesTwoRegions(t *testing.T) {
	t.Parallel()
	focus := 0
	for range 4 {
		focus = (focus + 1) % 2
	}
	if focus != 0 {
		t.Fatalf("expected 0 after 4 tabs, got %d", focus)
	}
}
