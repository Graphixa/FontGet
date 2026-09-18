package cmd

import (
	"testing"

	"fontget/internal/shared"
)

func TestDownloadFromSourceMessage(t *testing.T) {
	if got := DownloadFromSourceMessage("Google Fonts"); got != "Downloading from Google Fonts..." {
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

func TestProgressActivityLabel(t *testing.T) {
	cases := []struct {
		u    ProgressUpdate
		src  string
		want string
	}{
		{ProgressUpdate{Phase: installStepDownload}, "Google Fonts", "Downloading from Google Fonts..."},
		{ProgressUpdate{Phase: installStepExtract}, "", progressLabelExtract},
		{ProgressUpdate{Phase: installStepInstall, Done: 0, Total: 10}, "", "Installing variant (1 of 10)..."},
		{ProgressUpdate{Phase: installStepInstall, Done: 9, Total: 10}, "", "Installing variant (10 of 10)..."},
		{ProgressUpdate{Phase: installStepInstall, Done: 10, Total: 10}, "", "Installing variant (10 of 10)..."},
		{ProgressUpdate{Phase: installStepForceRemove}, "", progressLabelRemove},
		{ProgressUpdate{Phase: removeStepRemove}, "", progressLabelRemove},
		{ProgressUpdate{Phase: removeStepScan}, "", ""},
		{ProgressUpdate{Phase: installStepPrecheck}, "", ""},
		{ProgressUpdate{Phase: exportStepPrep}, "", progressLabelExport},
		{ProgressUpdate{Phase: backupStepFiles}, "", progressLabelBackup},
	}
	for _, tc := range cases {
		if got := ProgressActivityLabel(tc.u, tc.src); got != tc.want {
			t.Fatalf("phase %q: got %q want %q", tc.u.Phase, got, tc.want)
		}
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
	if midDL < 2 || midDL > 28 {
		t.Fatalf("mid download %% = %v, want in (2,28)", midDL)
	}

	startInst := OverallWorkPercent(0, 1, ProgressUpdate{
		Phase: installStepInstall, Kind: ProgressCount, Done: 0, Total: 400,
	})
	if startInst < 37 || startInst > 39 {
		t.Fatalf("install start %% = %v, want ~38", startInst)
	}

	midInst := OverallWorkPercent(0, 1, ProgressUpdate{
		Phase: installStepInstall, Kind: ProgressCount, Done: 200, Total: 400,
	})
	wantMid := segForceRemoveEnd*100 + (segFilesEnd-segForceRemoveEnd)*100*0.5
	if midInst < wantMid-2 || midInst > wantMid+2 {
		t.Fatalf("install mid %% = %v, want ~%v", midInst, wantMid)
	}

	done := OverallWorkPercent(0, 1, ProgressUpdate{Phase: installStepCompleted})
	if done != 100 {
		t.Fatalf("completed %% = %v want 100", done)
	}
}

func TestOverallWorkPercent_ForceRemoveThenInstall(t *testing.T) {
	endPrep := OverallWorkPercent(0, 1, ProgressUpdate{
		Phase: installStepDownload, Kind: ProgressCount, Done: 1, Total: 1,
	})
	midForce := OverallWorkPercent(0, 1, ProgressUpdate{
		Phase: installStepForceRemove, Kind: ProgressCount, Done: 0.5, Total: 1,
	})
	if midForce <= endPrep {
		t.Fatalf("force remove must continue after prep: prep=%v force=%v", endPrep, midForce)
	}
	startInst := OverallWorkPercent(0, 1, ProgressUpdate{
		Phase: installStepInstall, Kind: ProgressCount, Done: 0, Total: 10,
	})
	if startInst < midForce {
		t.Fatalf("install must not reset below force remove: force=%v install=%v", midForce, startInst)
	}
}

func TestOverallWorkPercent_MultiItem(t *testing.T) {
	got := OverallWorkPercent(1, 2, ProgressUpdate{
		Phase: installStepInstall, Kind: ProgressCount, Done: 0, Total: 10,
	})
	// package 2 starts at 50% + force-remove band (38%) → ~69%
	if got < 68 || got > 70 {
		t.Fatalf("multi-item %% = %v, want ~69", got)
	}
	pkg1Done := OverallWorkPercent(0, 2, ProgressUpdate{Phase: installStepCompleted})
	if pkg1Done != 50 {
		t.Fatalf("package 1 complete = %v want 50", pkg1Done)
	}
	if got < pkg1Done {
		t.Fatalf("package 2 must continue from package 1 endpoint")
	}
}

func TestForceRemoveFinalizeDoesNotCompleteBar(t *testing.T) {
	// Reproduce google.lekton --force: after prep, removal finalize must not hit 100%
	// before install, or the bar looks like it completes then restarts.
	endPrep := OverallWorkPercent(0, 1, ProgressUpdate{
		Phase: installStepDownload, Kind: ProgressCount, Done: 1, Total: 1,
	})
	midForce := OverallWorkPercent(0, 1, remapForceInstallProgress(ProgressUpdate{
		Phase: removeStepRemove, Kind: ProgressCount, Done: 1, Total: 2,
	}))
	endForceFiles := OverallWorkPercent(0, 1, remapForceInstallProgress(ProgressUpdate{
		Phase: removeStepRemove, Kind: ProgressCount, Done: 2, Total: 2,
	}))
	// Bug: raw removeStepFinalize jumps to ~100%.
	rawFinalize := OverallWorkPercent(0, 1, ProgressUpdate{
		Phase: removeStepFinalize, Kind: ProgressFlag, Done: 1, Total: 1,
	})
	if rawFinalize < 98 {
		t.Fatalf("sanity: raw remove finalize should be ~100, got %v", rawFinalize)
	}
	mappedFinalize := OverallWorkPercent(0, 1, remapForceInstallProgress(ProgressUpdate{
		Phase: removeStepFinalize, Kind: ProgressFlag, Done: 1, Total: 1,
	}))
	startInstall := OverallWorkPercent(0, 1, ProgressUpdate{
		Phase: installStepInstall, Kind: ProgressCount, Done: 0, Total: 2,
	})

	seq := []float64{endPrep, midForce, endForceFiles, mappedFinalize, startInstall}
	for i := 1; i < len(seq); i++ {
		if seq[i]+0.01 < seq[i-1] {
			t.Fatalf("progress reset at step %d: %v → %v (seq=%v)", i, seq[i-1], seq[i], seq)
		}
	}
	if mappedFinalize >= 90 {
		t.Fatalf("mapped force finalize must stay in force band, got %v (raw was %v)", mappedFinalize, rawFinalize)
	}
	if startInstall+0.01 < mappedFinalize {
		t.Fatalf("install must not restart below force end: finalize=%v install=%v", mappedFinalize, startInstall)
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

func TestOverallExportPercent_NoPrematureComplete(t *testing.T) {
	prep := OverallExportPercent(ProgressUpdate{Phase: exportStepPrep, Kind: ProgressFlag, Done: 1, Total: 1})
	if prep < 39 || prep > 41 {
		t.Fatalf("prep end = %v want ~40", prep)
	}
	writeMid := OverallExportPercent(ProgressUpdate{Phase: exportStepWrite, Kind: ProgressCount, Done: 1, Total: 2})
	if writeMid <= prep {
		t.Fatalf("write must advance past prep: prep=%v write=%v", prep, writeMid)
	}
	if writeMid >= 100 {
		t.Fatalf("write mid must not be 100: %v", writeMid)
	}
	done := OverallExportPercent(ProgressUpdate{Phase: exportStepDone})
	if done != 100 {
		t.Fatalf("done = %v want 100", done)
	}
}

func TestOverallBackupPercent_FinalizeOnlyAfterClose(t *testing.T) {
	mid := OverallBackupPercent(5, 10, false)
	if mid < 48 || mid > 50 {
		t.Fatalf("mid backup = %v want ~49", mid)
	}
	allFiles := OverallBackupPercent(10, 10, false)
	if allFiles >= 100 {
		t.Fatalf("all files archived must leave headroom for finalize: %v", allFiles)
	}
	done := OverallBackupPercent(10, 10, true)
	if done != 100 {
		t.Fatalf("finalized = %v want 100", done)
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

func TestSingleVariantDoesNotFillPrepBand(t *testing.T) {
	// One of four download units at full unit progress should sit at 25% of prep band, not 100%.
	got := OverallWorkPercent(0, 1, ProgressUpdate{
		Phase: installStepDownload, Kind: ProgressCount, Done: 1, Total: 4,
	})
	prepSpan := (segPrepEnd - segPrecheckEnd) * 100
	want := segPrecheckEnd*100 + prepSpan*0.25
	if got < want-1 || got > want+1 {
		t.Fatalf("unit 1/4 complete = %v want ~%v (must not fill entire prep)", got, want)
	}
}
