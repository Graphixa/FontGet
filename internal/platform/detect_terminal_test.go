package platform

import (
	"os"
	"testing"
	"time"
)

func TestClassifyTerminalTheme(t *testing.T) {
	tests := []struct {
		name     string
		color    TerminalRGB
		expected TerminalTheme
	}{
		{
			name:     "black is dark",
			color:    TerminalRGB{R: 0, G: 0, B: 0},
			expected: TerminalThemeDark,
		},
		{
			name:     "white is light",
			color:    TerminalRGB{R: 1, G: 1, B: 1},
			expected: TerminalThemeLight,
		},
		{
			name:     "mid grey treated as dark by threshold",
			color:    TerminalRGB{R: 0.4, G: 0.4, B: 0.4},
			expected: TerminalThemeDark,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyTerminalTheme(tt.color)
			if got != tt.expected {
				t.Fatalf("classifyTerminalTheme(%v) = %v, want %v", tt.color, got, tt.expected)
			}
		})
	}
}

func TestAnsi16ToTerminalRGBBounds(t *testing.T) {
	tests := []int{-1, 0, 7, 8, 15, 16}
	for _, idx := range tests {
		t.Run(os.Getenv("GOOS"), func(t *testing.T) {
			_ = ansi16ToTerminalRGB(idx) // just ensure it does not panic and returns a value
		})
	}
}

func TestAnsi256ToTerminalRGBRanges(t *testing.T) {
	// Check key ranges do not panic and stay within 0..1
	indices := []int{-1, 0, 15, 16, 100, 231, 232, 255, 256}
	for _, idx := range indices {
		t.Run(os.Getenv("GOOS"), func(t *testing.T) {
			c := ansi256ToTerminalRGB(idx)
			if c.R < 0 || c.R > 1 || c.G < 0 || c.G > 1 || c.B < 0 || c.B > 1 {
				t.Fatalf("ansi256ToTerminalRGB(%d) produced out of range value: %+v", idx, c)
			}
		})
	}
}

func TestDetectFromColorFGBG(t *testing.T) {
	orig := os.Getenv("COLORFGBG")
	defer os.Setenv("COLORFGBG", orig)

	tests := []struct {
		name      string
		envValue  string
		wantError bool
	}{
		{"empty env", "", true},
		{"invalid format", "abc", true},
		{"single background index", "0", false},
		{"foreground and background", "7;0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.Setenv("COLORFGBG", tt.envValue); err != nil {
				t.Fatalf("failed to set COLORFGBG: %v", err)
			}
			_, err := detectFromColorFGBG()
			if tt.wantError && err == nil {
				t.Fatalf("expected error but got nil for value %q", tt.envValue)
			}
			if !tt.wantError && err != nil {
				t.Fatalf("did not expect error but got %v for value %q", err, tt.envValue)
			}
		})
	}
}

func TestTerminalThemeFromEnvOrDetectOverride(t *testing.T) {
	const envVar = "FONTGET_THEME_MODE_TEST"

	tests := []struct {
		name     string
		value    string
		expected TerminalTheme
	}{
		{"dark override", "dark", TerminalThemeDark},
		{"light override", "light", TerminalThemeLight},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := os.Getenv(envVar)
			defer func() {
				if orig != "" {
					os.Setenv(envVar, orig)
				} else {
					os.Unsetenv(envVar)
				}
			}()

			os.Setenv(envVar, tt.value)

			// Use a very short timeout to prevent hanging in CI environments
			// Since we're setting the override, detection should never be called,
			// but this ensures the test completes quickly if something goes wrong
			res, err := TerminalThemeFromEnvOrDetect(envVar, 10*time.Millisecond)
			if err != nil {
				t.Fatalf("TerminalThemeFromEnvOrDetect returned error with override %q: %v", tt.value, err)
			}
			if res.Theme != tt.expected {
				t.Fatalf("expected theme %v, got %v for override %q", tt.expected, res.Theme, tt.value)
			}
		})
	}
}
