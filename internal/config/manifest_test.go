package config

import "testing"

func TestIsBuiltInSource(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Google Fonts", "Google Fonts", true},
		{"Nerd Fonts", "Nerd Fonts", true},
		{"Font Squirrel", "Font Squirrel", true},
		{"custom source", "My Custom Source", false},
		{"empty", "", false},
		{"case sensitive - lowercase", "google fonts", false},
		{"partial match", "Google", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsBuiltInSource(tt.input)
			if got != tt.expected {
				t.Errorf("IsBuiltInSource(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestBuiltInSourceNames(t *testing.T) {
	if len(BuiltInSourceNames) != 3 {
		t.Errorf("BuiltInSourceNames should have 3 entries, got %d", len(BuiltInSourceNames))
	}
	for _, n := range BuiltInSourceNames {
		if !IsBuiltInSource(n) {
			t.Errorf("BuiltInSourceNames entry %q should be recognized by IsBuiltInSource", n)
		}
	}
}
