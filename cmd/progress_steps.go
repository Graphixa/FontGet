package cmd

import "fontget/internal/shared"

// StepProgressFunc reports progress for a named step. stepPct must be in [0,1].
// Implementations should be lightweight and may be called frequently.
type StepProgressFunc func(step string, stepPct float64)

const (
	installStepPrecheck  = "Checking installed"
	installStepDownload  = "Downloading"
	installStepExtract   = "Extracting"
	installStepInstall   = "Installing"
	installStepFinalize  = "Finalizing"
	installStepCompleted = "Installed"
)

// Weights are tuned to keep progress moving during the longest phases without byte streaming.
// Download/install dominate. Finalize is small but non-zero so post-install cache flush isn’t a “jump”.
var installStepOrder = []string{
	installStepPrecheck,
	installStepDownload,
	installStepExtract,
	installStepInstall,
	installStepFinalize,
}

var installStepWeights = []float64{
	0.10, // precheck
	0.50, // download (byte-streaming when available)
	0.05, // extract (file/entry streaming when available)
	0.30, // install (file-by-file)
	0.05, // finalize (cache flush / cleanup)
}

const (
	removeStepScan      = "Scanning"
	removeStepRemove    = "Removing"
	removeStepFinalize  = "Finalizing"
	removeStepCompleted = "Removed"
)

var removeStepOrder = []string{
	removeStepScan,
	removeStepRemove,
	removeStepFinalize,
}

var removeStepWeights = []float64{
	0.55, // scan/find (can be expensive on some platforms)
	0.40, // remove files
	0.05, // cache flush / finalize
}

func removeStepIndex(step string) int {
	for i, s := range removeStepOrder {
		if s == step {
			return i
		}
	}
	return 0
}

func OverallRemovePercent(fontIndex int, totalFonts int, step string, stepPct float64) float64 {
	if totalFonts <= 0 {
		return 0
	}
	if fontIndex < 0 {
		fontIndex = 0
	}
	if fontIndex >= totalFonts {
		fontIndex = totalFonts - 1
		stepPct = 1
	}

	if step == removeStepCompleted {
		return (float64(fontIndex+1) / float64(totalFonts)) * 100.0
	}

	stepIdx := removeStepIndex(step)
	perFont := shared.WeightedPhaseProgress(removeStepWeights, stepIdx, stepPct)
	overall := (float64(fontIndex) + perFont) / float64(totalFonts)
	if overall < 0 {
		overall = 0
	}
	if overall > 1 {
		overall = 1
	}
	return overall * 100.0
}

func installStepIndex(step string) int {
	for i, s := range installStepOrder {
		if s == step {
			return i
		}
	}
	// Unknown steps map to the current/first phase to avoid breaking callers.
	return 0
}

// OverallInstallPercent maps per-font step progress to global 0..100 percent across N fonts.
// - fontIndex is 0-based, totalFonts must be > 0.
func OverallInstallPercent(fontIndex int, totalFonts int, step string, stepPct float64) float64 {
	if totalFonts <= 0 {
		return 0
	}
	if fontIndex < 0 {
		fontIndex = 0
	}
	if fontIndex >= totalFonts {
		fontIndex = totalFonts - 1
		stepPct = 1
	}

	if step == installStepCompleted {
		return (float64(fontIndex+1) / float64(totalFonts)) * 100.0
	}

	stepIdx := installStepIndex(step)
	perFont := shared.WeightedPhaseProgress(installStepWeights, stepIdx, stepPct) // 0..1 within this font

	overall := (float64(fontIndex) + perFont) / float64(totalFonts) // 0..1 overall
	if overall < 0 {
		overall = 0
	}
	if overall > 1 {
		overall = 1
	}
	return overall * 100.0
}

