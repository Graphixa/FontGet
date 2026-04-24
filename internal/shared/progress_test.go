package shared

import "testing"

func TestClamp01(t *testing.T) {
	if got := Clamp01(-1); got != 0 {
		t.Fatalf("Clamp01(-1)=%v want 0", got)
	}
	if got := Clamp01(0.25); got != 0.25 {
		t.Fatalf("Clamp01(0.25)=%v want 0.25", got)
	}
	if got := Clamp01(2); got != 1 {
		t.Fatalf("Clamp01(2)=%v want 1", got)
	}
}

func TestWeightedPhaseProgress_EmptyWeights(t *testing.T) {
	if got := WeightedPhaseProgress(nil, 0, 0.3); got != 0.3 {
		t.Fatalf("got %v want 0.3", got)
	}
}

func TestWeightedPhaseProgress_EqualWeightsFallback(t *testing.T) {
	got := WeightedPhaseProgress([]float64{0, 0, 0}, 1, 0.5)
	// With 3 equal phases, phase 1 halfway: (1 + 0.5)/3 = 0.5
	if got != 0.5 {
		t.Fatalf("got %v want 0.5", got)
	}
}

func TestWeightedPhaseProgress_ClampsAndBounds(t *testing.T) {
	weights := []float64{1, 1}
	if got := WeightedPhaseProgress(weights, -2, -1); got != 0 {
		t.Fatalf("got %v want 0", got)
	}
	if got := WeightedPhaseProgress(weights, 99, 0.2); got != 1 {
		t.Fatalf("got %v want 1", got)
	}
}

