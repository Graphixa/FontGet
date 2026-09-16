package cmd

import (
	"testing"

	"fontget/internal/shared"
)

func TestDownloadFromSourceMessage(t *testing.T) {
	if got := DownloadFromSourceMessage("Google Fonts"); got != "Downloading from Google Fonts" {
		t.Fatalf("got %q", got)
	}
	if got := DownloadFromSourceMessage("  "); got != "Downloading..." {
		t.Fatalf("empty source got %q", got)
	}
	if !isInstallPrepPhase(installStepDownload) || !isInstallPrepPhase(installStepExtract) {
		t.Fatal("prep phases")
	}
	if isInstallPrepPhase(installStepInstall) {
		t.Fatal("install is not prep")
	}
}

func TestFormatProgressActivity(t *testing.T) {
	if got := FormatProgressActivity("Installing", ""); got != "Installing..." {
		t.Fatalf("got %q", got)
	}
	if got := FormatProgressActivity("Installing", "variants (12 of 399)"); got != "Installing variants (12 of 399)" {
		t.Fatalf("got %q", got)
	}
}

func TestCountDetail(t *testing.T) {
	if got := CountDetail(2, 10, `/tmp/dir/Foo.ttf`); got != "2/10 Foo.ttf" {
		t.Fatalf("got %q", got)
	}
}

func TestOverallWorkPercent_DownloadThenInstall(t *testing.T) {
	midDL := OverallWorkPercent(0, 1, ProgressUpdate{
		Phase: installStepDownload, Kind: ProgressCount, Done: 0.5, Total: 1,
	})
	if midDL < 2 || midDL > 35 {
		t.Fatalf("mid download %% = %v, want in (2,35)", midDL)
	}

	startInst := OverallWorkPercent(0, 1, ProgressUpdate{
		Phase: installStepInstall, Kind: ProgressCount, Done: 0, Total: 400,
	})
	if startInst < 34 || startInst > 36 {
		t.Fatalf("install start %% = %v, want ~35", startInst)
	}

	midInst := OverallWorkPercent(0, 1, ProgressUpdate{
		Phase: installStepInstall, Kind: ProgressCount, Done: 200, Total: 400,
	})
	wantMid := 35 + (98-35)*0.5
	if midInst < wantMid-2 || midInst > wantMid+2 {
		t.Fatalf("install mid %% = %v, want ~%v", midInst, wantMid)
	}

	done := OverallWorkPercent(0, 1, ProgressUpdate{Phase: installStepCompleted})
	if done != 100 {
		t.Fatalf("completed %% = %v want 100", done)
	}
}

func TestOverallWorkPercent_MultiItem(t *testing.T) {
	got := OverallWorkPercent(1, 2, ProgressUpdate{
		Phase: installStepInstall, Kind: ProgressCount, Done: 0, Total: 10,
	})
	if got < 65 || got > 70 {
		t.Fatalf("multi-item %% = %v, want ~67.5", got)
	}
}

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

func TestProgressThrottle(t *testing.T) {
	var th progressThrottle
	u := ProgressUpdate{Phase: installStepDownload, Detail: "", Kind: ProgressBytes, Done: 1, Total: 100}
	if !th.ShouldSend(u, 3) {
		t.Fatal("first send")
	}
	if th.ShouldSend(u, 3.4) {
		t.Fatal("same 1% bucket should skip")
	}
	if !th.ShouldSend(u, 4) {
		t.Fatal("next percent should send")
	}
	u.Detail = "1/2 a.ttf"
	if !th.ShouldSend(u, 4) {
		t.Fatal("detail change should send")
	}
}

func TestPhaseFrac_UnknownBytesNeverComplete(t *testing.T) {
	if got := phaseFrac(ProgressUpdate{Kind: ProgressBytes, Done: 1, Total: 0}); got != 0 {
		t.Fatalf("1 byte unknown size must not complete phase, got %v", got)
	}
	if got := phaseFrac(ProgressUpdate{Kind: ProgressBytes, Done: 1e9, Total: -1}); got != 0 {
		t.Fatalf("unknown total must stay 0, got %v", got)
	}
	if got := phaseFrac(ProgressUpdate{Kind: ProgressBytes, Done: 50, Total: 100}); shared.Clamp01(got) != 0.5 {
		t.Fatalf("known size mid = %v", got)
	}
}

func TestPrepUnitFracs(t *testing.T) {
	if got := prepDownloadUnitFrac(50, 100); got < 0.37 || got > 0.38 {
		t.Fatalf("half download = %v want ~0.375", got)
	}
	if got := prepDownloadUnitFrac(100, -1); got != 0 {
		t.Fatalf("unknown download = %v want 0", got)
	}
	if got := prepExtractUnitFrac(0, 0); got != prepDownloadWeight {
		t.Fatalf("unknown extract hold = %v", got)
	}
	if got := prepExtractUnitFrac(1, 2); got < 0.87 || got > 0.88 {
		t.Fatalf("half extract = %v want ~0.875", got)
	}
}

func TestMultiArchivePrepIsMonotonic(t *testing.T) {
	// Two downloads: end of first unit must be below mid of second; no reset to prep start.
	endFirst := OverallWorkPercent(0, 1, ProgressUpdate{
		Phase: installStepExtract, Kind: ProgressCount, Done: 1, Total: 2,
	})
	midSecond := OverallWorkPercent(0, 1, ProgressUpdate{
		Phase: installStepDownload, Kind: ProgressCount, Done: 1.5, Total: 2,
	})
	if midSecond <= endFirst {
		t.Fatalf("second download must advance past first unit: endFirst=%v midSecond=%v", endFirst, midSecond)
	}
	startSecond := OverallWorkPercent(0, 1, ProgressUpdate{
		Phase: installStepDownload, Kind: ProgressCount, Done: 1, Total: 2,
	})
	if startSecond < endFirst-0.01 {
		t.Fatalf("starting second unit must not jump backwards: endFirst=%v startSecond=%v", endFirst, startSecond)
	}
}

func TestDownloadExtractSharePrepBand(t *testing.T) {
	ds, de := segmentRange(installStepDownload)
	es, ee := segmentRange(installStepExtract)
	if ds != es || de != ee || de != segPrepEnd {
		t.Fatalf("download/extract must share prep band, got dl=(%v,%v) ex=(%v,%v)", ds, de, es, ee)
	}
}
