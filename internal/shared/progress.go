package shared

// Clamp01 clamps v to the range [0, 1].
func Clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// WeightedPhaseProgress returns an overall progress in [0,1] given:
// - phaseWeights: relative weights for each phase (need not sum to 1)
// - phaseIndex: which phase is currently active (0-based)
// - phaseProgress: progress within the active phase in [0,1]
//
// If weights are empty or all zero, it falls back to equal weights.
func WeightedPhaseProgress(phaseWeights []float64, phaseIndex int, phaseProgress float64) float64 {
	n := len(phaseWeights)
	if n == 0 {
		return Clamp01(phaseProgress)
	}
	if phaseIndex < 0 {
		phaseIndex = 0
	}
	if phaseIndex >= n {
		phaseIndex = n - 1
		phaseProgress = 1
	}

	// Sum weights, falling back to equal weights if needed.
	sum := 0.0
	for _, w := range phaseWeights {
		if w > 0 {
			sum += w
		}
	}
	useEqual := sum <= 0
	if useEqual {
		sum = float64(n)
	}

	done := 0.0
	for i := 0; i < phaseIndex; i++ {
		if useEqual {
			done += 1
		} else if phaseWeights[i] > 0 {
			done += phaseWeights[i]
		}
	}

	curW := 1.0
	if !useEqual {
		if phaseWeights[phaseIndex] > 0 {
			curW = phaseWeights[phaseIndex]
		}
	}

	p := (done + Clamp01(phaseProgress)*curW) / sum
	return Clamp01(p)
}

