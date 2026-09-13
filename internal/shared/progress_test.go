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
