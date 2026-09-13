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
	ProgressBytes ProgressKind = iota // download
	ProgressCount                     // extract / install / remove / backup file counts
	ProgressFlag                      // precheck / finalize (0 → 1)
)

// ProgressUpdate is one pulse of work-unit progress.
type ProgressUpdate struct {
	Phase  string
	Detail string
	Kind   ProgressKind
	Done   float64
	Total  float64 // 0 = unknown for this pulse
}

// ProgressFunc reports progress. Implementations must be lightweight.
type ProgressFunc func(ProgressUpdate)

// Legacy phase name constants (UI labels + segment keys).
const (
	installStepPrecheck  = "Checking installed"
	installStepDownload  = "Downloading"
	installStepExtract   = "Extracting"
	installStepInstall   = "Installing"
	installStepFinalize  = "Finalizing"
	installStepCompleted = "Installed"

	removeStepScan      = "Scanning"
	removeStepRemove    = "Removing"
	removeStepFinalize  = "Finalizing"
	removeStepCompleted = "Removed"
)

// Reserved bar segments for a single item (monotonic; install/remove owns most of the bar).
const (
	segPrecheckEnd   = 0.02
	segDownloadEnd   = 0.25
	segExtractEnd    = 0.35
	segFilesEnd      = 0.98
	segFinalizeEnd   = 1.00
	segRemoveScanEnd = 0.35 // scan uses the whole prelude band
)

// FormatProgressActivity builds the status line for progress UIs.
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

// phaseFrac returns progress within the active phase in [0,1].
func phaseFrac(u ProgressUpdate) float64 {
	switch u.Kind {
	case ProgressFlag:
		if u.Done >= 1 || (u.Total > 0 && u.Done >= u.Total) {
			return 1
		}
		return shared.Clamp01(u.Done)
	case ProgressBytes, ProgressCount:
		if u.Total <= 0 {
			if u.Done >= 1 {
				return 1
			}
			return 0
		}
		return shared.Clamp01(u.Done / u.Total)
	default:
		return 0
	}
}

// segmentRange returns [start,end) in 0..1 for a phase label.
func segmentRange(phase string) (start, end float64) {
	switch phase {
	case installStepPrecheck:
		return 0, segPrecheckEnd
	case installStepDownload:
		return segPrecheckEnd, segDownloadEnd
	case installStepExtract:
		return segDownloadEnd, segExtractEnd
	case installStepInstall, removeStepRemove:
		return segExtractEnd, segFilesEnd
	case installStepFinalize:
		return segFilesEnd, segFinalizeEnd
	case removeStepScan:
		return 0, segRemoveScanEnd
	case installStepCompleted, removeStepCompleted:
		return 1, 1
	default:
		return segExtractEnd, segFilesEnd
	}
}

// itemFrac maps a ProgressUpdate to 0..1 progress within one catalog item.
func itemFrac(u ProgressUpdate) float64 {
	if u.Phase == installStepCompleted || u.Phase == removeStepCompleted {
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
		itemIndex = itemCount - 1
		return 100
	}
	frac := itemFrac(u)
	overall := (float64(itemIndex) + frac) / float64(itemCount)
	return shared.Clamp01(overall) * 100
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
