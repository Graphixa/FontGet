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
		{"The League of Moveable Type", "The League of Moveable Type", true},
		{"Fontshare", "Fontshare", true},
		{"Fontsource", "Fontsource", true},
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
	if len(BuiltInSourceNames) != 6 {
		t.Errorf("BuiltInSourceNames should have 6 entries, got %d", len(BuiltInSourceNames))
	}
	for _, n := range BuiltInSourceNames {
		if !IsBuiltInSource(n) {
			t.Errorf("BuiltInSourceNames entry %q should be recognized by IsBuiltInSource", n)
		}
	}
}

func TestMergeBuiltInSourcesFromDefaults(t *testing.T) {
	m := &Manifest{
		Sources: map[string]SourceConfig{
			"Google Fonts": {
				URL:      "https://example.com/google.json",
				Prefix:   "google",
				Enabled:  true,
				Filename: "google-fonts.json",
				Priority: 1,
			},
			"Nerd Fonts": {
				URL:      "https://example.com/nerd.json",
				Prefix:   "nerd",
				Enabled:  false,
				Filename: "nerd-fonts.json",
				Priority: 2,
			},
			"Font Squirrel": {
				URL:      "https://example.com/squirrel.json",
				Prefix:   "squirrel",
				Enabled:  true,
				Filename: "font-squirrel.json",
				Priority: 6,
			},
		},
	}

	changed, err := mergeBuiltInSourcesFromDefaults(m)
	if err != nil {
		t.Fatalf("mergeBuiltInSourcesFromDefaults: %v", err)
	}
	if !changed {
		t.Fatal("expected merge to report changes when built-ins are missing")
	}
	if len(m.Sources) != 6 {
		t.Fatalf("len(Sources) = %d, want 6", len(m.Sources))
	}

	for _, name := range []string{"The League of Moveable Type", "Fontshare", "Fontsource"} {
		cfg, ok := m.Sources[name]
		if !ok {
			t.Fatalf("missing merged source %q", name)
		}
		if !cfg.Enabled {
			t.Errorf("merged built-in %q should be enabled by default, got enabled=%v", name, cfg.Enabled)
		}
	}

	// Existing entries must be untouched
	if m.Sources["Google Fonts"].URL != "https://example.com/google.json" {
		t.Errorf("Google Fonts URL was overwritten")
	}
	if m.Sources["Nerd Fonts"].Enabled {
		t.Errorf("Nerd Fonts Enabled should remain false")
	}

	changedAgain, err := mergeBuiltInSourcesFromDefaults(m)
	if err != nil {
		t.Fatalf("second merge: %v", err)
	}
	if changedAgain {
		t.Error("second merge should be a no-op")
	}
}
