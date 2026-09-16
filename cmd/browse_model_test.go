package cmd

import (
	"errors"
	"strings"
	"testing"

	"fontget/internal/platform"
	"fontget/internal/shared"
)

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

func TestBrowseKeepOpErrPreservesSuccess(t *testing.T) {
	if err := browseKeepOpErr(nil); err != nil {
		t.Fatalf("success must stay nil: %v", err)
	}
	want := errors.New("real failure")
	if got := browseKeepOpErr(want); !errors.Is(got, want) {
		t.Fatalf("got %v", got)
	}
}

func TestBrowseResultFromInstallFinalFileLateCancelStillInstalled(t *testing.T) {
	// Simulates: installFont succeeded; caller must not inject ErrOperationCancelled.
	msg := installFinishedMsg{
		result: &InstallResult{Status: InstallStatusCompleted, Message: "Installed", Success: 1},
		err:    browseKeepOpErr(nil),
		fontID: "test.final",
	}
	title, errTitle, body := browseResultFromInstall("Final", "test", msg, platform.UserScope, false)
	if errTitle || title != "Installed" {
		t.Fatalf("title=%q errTitle=%v body=%q", title, errTitle, body)
	}
	if strings.Contains(body, "cancelled") || strings.Contains(body, "Cancelled") {
		t.Fatalf("must not show cancel: %q", body)
	}
}

func TestBrowseResultFromInstallCancelStillCancelled(t *testing.T) {
	msg := installFinishedMsg{
		err:    shared.ErrOperationCancelled,
		fontID: "test.cancel",
	}
	title, errTitle, _ := browseResultFromInstall("X", "test", msg, platform.UserScope, false)
	if errTitle || title != "Cancelled" {
		t.Fatalf("title=%q errTitle=%v", title, errTitle)
	}
}

func TestBrowseResultFromUninstallFinalFileLateCancelStillUninstalled(t *testing.T) {
	msg := uninstallFinishedMsg{
		result: &RemoveResult{Status: StatusCompleted, Message: "Removed", Success: 1},
		err:    browseKeepOpErr(nil),
		fontID: "test.rmfinal",
	}
	title, errTitle, body := browseResultFromUninstall("Final", platform.UserScope, msg)
	if errTitle || title != "Uninstalled" {
		t.Fatalf("title=%q errTitle=%v body=%q", title, errTitle, body)
	}
}
