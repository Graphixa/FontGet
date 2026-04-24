package cmd

import "testing"

func TestOverallInstallPercent_Bounds(t *testing.T) {
	if got := OverallInstallPercent(0, 0, installStepDownload, 0.5); got != 0 {
		t.Fatalf("got %v want 0", got)
	}
	if got := OverallInstallPercent(-1, 2, installStepDownload, -1); got < 0 || got > 100 {
		t.Fatalf("out of bounds: %v", got)
	}
	if got := OverallInstallPercent(99, 2, installStepCompleted, 1); got != 100 {
		t.Fatalf("got %v want 100", got)
	}
}

func TestOverallRemovePercent_Bounds(t *testing.T) {
	if got := OverallRemovePercent(0, 0, removeStepScan, 0.5); got != 0 {
		t.Fatalf("got %v want 0", got)
	}
	if got := OverallRemovePercent(99, 2, removeStepCompleted, 1); got != 100 {
		t.Fatalf("got %v want 100", got)
	}
}

