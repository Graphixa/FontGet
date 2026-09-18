package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"fontget/internal/shared"
)

// ProgressKind selects how Done/Total are interpreted for overall %.
type ProgressKind int

const (
	ProgressBytes ProgressKind = iota // single-stream bytes (known size only for bar math)
	ProgressCount                     // work units / file counts
	ProgressFlag                      // precheck / finalize (0 → 1)
)

// ProgressUpdate is one pulse of work-unit progress.
type ProgressUpdate struct {
	Phase  string
	Detail string
	Kind   ProgressKind
	Done   float64
	Total  float64 // 0 = unknown; never treat unknown bytes as complete
}

// ProgressFunc reports progress. Implementations must be lightweight.
type ProgressFunc func(ProgressUpdate)

// Internal phase keys (not user-facing labels).
const (
	installStepPrecheck    = "Checking installed"
	installStepDownload    = "Downloading"
	installStepExtract     = "Extracting"
	installStepForceRemove = "ForceRemoving"
	installStepInstall     = "Installing"
	installStepFinalize    = "InstallFinalizing"
	installStepCompleted   = "Installed"

	removeStepScan      = "Scanning"
	removeStepRemove    = "Removing"
	removeStepFinalize  = "RemoveFinalizing"
	removeStepCompleted = "Removed"

	exportStepPrep = "ExportPrep"
	exportStepWrite = "ExportWrite"
	exportStepDone  = "ExportDone"

	backupStepFiles = "BackupFiles"
	backupStepDone  = "BackupDone"
)

// Exact user-facing activity labels (checklist).
const (
	progressLabelExtract = "Extracting..."
	progressLabelRemove  = "Removing fonts..."
	progressLabelExport  = "Exporting font list..."
	progressLabelBackup  = "Backing up fonts..."
	progressLabelCancel  = "Cancelling..."
)

// Reserved bar segments for a single package/item (monotonic).
// Prep aggregates downloads+extracts; force-remove band is reserved even when unused.
const (
	segPrecheckEnd      = 0.02
	segPrepEnd          = 0.28 // download+extract preparation
	segForceRemoveEnd   = 0.38 // force reinstall removal (idle when not force)
	segFilesEnd         = 0.98
	segFinalizeEnd      = 1.00
	segRemoveScanEnd    = 0.35

	prepDownloadWeight = 0.75
	prepExtractWeight  = 0.25

	// Export: prep (scan/group/match/filter) then write, then finalize after file flush.
	segExportPrepEnd  = 0.40
	segExportWriteEnd = 0.98

	// Backup: file copy band, finalize after archive close.
	segBackupFilesEnd = 0.98

	// Aliases: prep band replaced the old split download/extract segments.
	segDownloadEnd = segPrepEnd
	segExtractEnd  = segPrepEnd
)

// DownloadFromSourceMessage is the user-facing download label.
func DownloadFromSourceMessage(sourceName string) string {
	sourceName = strings.TrimSpace(sourceName)
	if sourceName == "" {
		return "Downloading..."
	}
	return "Downloading from " + sourceName + "..."
}

// InstallingVariantMessage is the user-facing install label (1-based current).
func InstallingVariantMessage(current, total int) string {
	if total < 1 {
		total = 1
	}
	if current < 1 {
		current = 1
	}
	if current > total {
		current = total
	}
	return fmt.Sprintf("Installing variant (%d of %d)...", current, total)
}

// ProgressActivityLabel returns the exact checklist label for a progress pulse.
// Empty means keep the previous activity text (brief internal steps).
func ProgressActivityLabel(u ProgressUpdate, sourceName string) string {
	switch u.Phase {
	case installStepDownload:
		return DownloadFromSourceMessage(sourceName)
	case installStepExtract:
		return progressLabelExtract
	case installStepInstall:
		total := int(u.Total)
		if total <= 0 {
			return ""
		}
		// Done is completed count; current file is Done+1 while work is in flight.
		cur := int(u.Done) + 1
		if u.Done >= u.Total {
			cur = total
		}
		return InstallingVariantMessage(cur, total)
	case installStepForceRemove, removeStepRemove:
		return progressLabelRemove
	case removeStepScan:
		// Brief scan: keep prior label / avoid chatter.
		return ""
	case installStepPrecheck, installStepFinalize, removeStepFinalize:
		return ""
	case exportStepPrep, exportStepWrite, exportStepDone:
		return progressLabelExport
	case backupStepFiles, backupStepDone:
		return progressLabelBackup
	case installStepCompleted, removeStepCompleted:
		return ""
	default:
		return ""
	}
}

// isInstallPrepPhase reports download or extract (shared prep band).
func isInstallPrepPhase(phase string) bool {
	return phase == installStepDownload || phase == installStepExtract
}

// FormatProgressActivity builds a generic status line (legacy helpers / tests).
func FormatProgressActivity(phase, detail string) string {
	phase = strings.TrimSpace(phase)
	detail = strings.TrimSpace(detail)
	if phase == "" {
		if detail == "" {
			return ""
		}
		return detail
	}
	if detail == "" {
		return phase + "..."
	}
	return phase + " " + detail
}

// CountDetail formats "i/n name" for count-based phases.
func CountDetail(i, n int, name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Sprintf("%d/%d", i, n)
	}
	return fmt.Sprintf("%d/%d %s", i, n, filepath.Base(name))
}

// prepDownloadUnitFrac is progress within one download/extract unit during download [0, prepDownloadWeight].
// Unknown Content-Length returns 0 (activity via label only); never invents completion.
func prepDownloadUnitFrac(doneBytes, totalBytes int64) float64 {
	if totalBytes <= 0 {
		return 0
	}
	return prepDownloadWeight * shared.Clamp01(float64(doneBytes)/float64(totalBytes))
}

// prepExtractUnitFrac is progress within one unit during extraction [prepDownloadWeight, 1].
// Unknown totals hold at prepDownloadWeight until the unit succeeds.
func prepExtractUnitFrac(done, total int) float64 {
	if total <= 0 {
		return prepDownloadWeight
	}
	return prepDownloadWeight + prepExtractWeight*shared.Clamp01(float64(done)/float64(total))
}

// phaseFrac returns progress within the active phase in [0,1].
func phaseFrac(u ProgressUpdate) float64 {
	switch u.Kind {
	case ProgressFlag:
		if u.Total > 0 {
			return shared.Clamp01(u.Done / u.Total)
		}
		if u.Done >= 1 {
			return 1
		}
		return shared.Clamp01(u.Done)
	case ProgressBytes, ProgressCount:
		if u.Total <= 0 {
			// Unknown size: never treat bytes received as phase-complete.
			return 0
		}
		return shared.Clamp01(u.Done / u.Total)
	default:
		return 0
	}
}

// segmentRange returns [start,end) in 0..1 for a phase key.
func segmentRange(phase string) (start, end float64) {
	switch phase {
	case installStepPrecheck:
		return 0, segPrecheckEnd
	case installStepDownload, installStepExtract:
		return segPrecheckEnd, segPrepEnd
	case installStepForceRemove:
		return segPrepEnd, segForceRemoveEnd
	case installStepInstall:
		return segForceRemoveEnd, segFilesEnd
	case installStepFinalize:
		return segFilesEnd, segFinalizeEnd
	case removeStepScan:
		return 0, segRemoveScanEnd
	case removeStepRemove:
		return segRemoveScanEnd, segFilesEnd
	case removeStepFinalize:
		return segFilesEnd, segFinalizeEnd
	case exportStepPrep:
		return 0, segExportPrepEnd
	case exportStepWrite:
		return segExportPrepEnd, segExportWriteEnd
	case exportStepDone, backupStepDone, installStepCompleted, removeStepCompleted:
		return 1, 1
	case backupStepFiles:
		return 0, segBackupFilesEnd
	default:
		return segForceRemoveEnd, segFilesEnd
	}
}

// itemFrac maps a ProgressUpdate to 0..1 progress within one catalog item.
func itemFrac(u ProgressUpdate) float64 {
	if u.Phase == installStepCompleted || u.Phase == removeStepCompleted ||
		u.Phase == exportStepDone || u.Phase == backupStepDone {
		return 1
	}
	start, end := segmentRange(u.Phase)
	if end <= start {
		return 1
	}
	f := phaseFrac(u)
	return start + (end-start)*f
}

// OverallWorkPercent maps work-unit progress to global 0..100 across itemCount items.
func OverallWorkPercent(itemIndex, itemCount int, u ProgressUpdate) float64 {
	if itemCount <= 0 {
		return 0
	}
	if itemIndex < 0 {
		itemIndex = 0
	}
	if itemIndex >= itemCount {
		return 100
	}
	frac := itemFrac(u)
	overall := (float64(itemIndex) + frac) / float64(itemCount)
	return shared.Clamp01(overall) * 100
}

// remapForceInstallProgress keeps force-reinstall removal inside the install force-remove
// band. Without this, removeFontFiles' removeStepFinalize maps to 98–100% and the bar
// appears to complete, then install starts again near 38%.
func remapForceInstallProgress(u ProgressUpdate) ProgressUpdate {
	switch u.Phase {
	case removeStepRemove:
		u.Phase = installStepForceRemove
		return u
	case removeStepFinalize, removeStepScan, removeStepCompleted:
		return ProgressUpdate{Phase: installStepForceRemove, Kind: ProgressCount, Done: 1, Total: 1}
	default:
		return u
	}
}

// OverallInstallPercent maps a simple phase fraction for install (used by thin callers).
func OverallInstallPercent(itemIndex, itemCount int, phase string, phasePct float64) float64 {
	return OverallWorkPercent(itemIndex, itemCount, ProgressUpdate{
		Phase: phase,
		Kind:  ProgressFlag,
		Done:  shared.Clamp01(phasePct),
		Total: 1,
	})
}

// OverallRemovePercent maps a simple phase fraction for remove (used by thin callers).
func OverallRemovePercent(itemIndex, itemCount int, phase string, phasePct float64) float64 {
	return OverallWorkPercent(itemIndex, itemCount, ProgressUpdate{
		Phase: phase,
		Kind:  ProgressFlag,
		Done:  shared.Clamp01(phasePct),
		Total: 1,
	})
}

// OverallExportPercent maps export work to 0..100 for a single-command bar.
func OverallExportPercent(u ProgressUpdate) float64 {
	return OverallWorkPercent(0, 1, u)
}

// OverallBackupPercent maps backup file work to 0..100; finalized only after archive close.
func OverallBackupPercent(doneFiles, totalFiles int, finalized bool) float64 {
	if finalized {
		return 100
	}
	return OverallWorkPercent(0, 1, ProgressUpdate{
		Phase: backupStepFiles,
		Kind:  ProgressCount,
		Done:  float64(doneFiles),
		Total: float64(max(totalFiles, 1)),
	})
}

// progressThrottle reduces UI spam for byte pulses while always forwarding phase/detail changes.
type progressThrottle struct {
	lastPhase  string
	lastDetail string
	lastBucket int
}

func (t *progressThrottle) ShouldSend(u ProgressUpdate, overallPct float64) bool {
	bucket := int(overallPct) // 1% steps
	if u.Phase != t.lastPhase || u.Detail != t.lastDetail || bucket != t.lastBucket {
		t.lastPhase = u.Phase
		t.lastDetail = u.Detail
		t.lastBucket = bucket
		return true
	}
	return false
}
