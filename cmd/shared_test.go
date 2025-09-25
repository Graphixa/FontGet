package cmd

import (
	"testing"
)

func TestParseFontNames(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "single font name",
			args:     []string{"Roboto"},
			expected: []string{"Roboto"},
		},
		{
			name:     "multiple font names",
			args:     []string{"Roboto", "Open Sans"},
			expected: []string{"Roboto", "Open Sans"},
		},
		{
			name:     "comma-separated font names",
			args:     []string{"Roboto,Open Sans,Arial"},
			expected: []string{"Roboto", "Open Sans", "Arial"},
		},
		{
			name:     "mixed single and comma-separated",
			args:     []string{"Roboto", "Open Sans,Arial", "Helvetica"},
			expected: []string{"Roboto", "Open Sans", "Arial", "Helvetica"},
		},
		{
			name:     "empty strings and whitespace",
			args:     []string{"Roboto", "", "  ", "Open Sans"},
			expected: []string{"Roboto", "Open Sans"},
		},
		{
			name:     "whitespace around commas",
			args:     []string{"Roboto , Open Sans , Arial"},
			expected: []string{"Roboto", "Open Sans", "Arial"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseFontNames(tt.args)
			if len(result) != len(tt.expected) {
				t.Errorf("ParseFontNames() returned %d items, expected %d", len(result), len(tt.expected))
				return
			}
			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("ParseFontNames() [%d] = %q, expected %q", i, result[i], expected)
				}
			}
		})
	}
}

func TestStatusReport(t *testing.T) {
	tests := []struct {
		name     string
		report   StatusReport
		expected string
	}{
		{
			name: "all zero values",
			report: StatusReport{
				Success:      0,
				Skipped:      0,
				Failed:       0,
				SuccessLabel: "Installed",
				SkippedLabel: "Skipped",
				FailedLabel:  "Failed",
			},
			expected: "", // Should not print anything
		},
		{
			name: "with successful operations",
			report: StatusReport{
				Success:      5,
				Skipped:      2,
				Failed:       1,
				SuccessLabel: "Installed",
				SkippedLabel: "Skipped",
				FailedLabel:  "Failed",
			},
			expected: "Status Report", // Should contain this text
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This is a bit tricky to test since it prints to stdout
			// In a real test, you'd capture stdout or use a different approach
			// For now, we'll just ensure it doesn't panic
			PrintStatusReport(tt.report)
		})
	}
}

func TestFontNotFoundError(t *testing.T) {
	tests := []struct {
		name        string
		fontName    string
		suggestions []string
		expected    string
	}{
		{
			name:        "no suggestions",
			fontName:    "NonExistentFont",
			suggestions: []string{},
			expected:    "font 'NonExistentFont' not found",
		},
		{
			name:        "with suggestions",
			fontName:    "Roboto",
			suggestions: []string{"Roboto Sans", "Roboto Mono"},
			expected:    "font 'Roboto' not found. Did you mean: Roboto Sans, Roboto Mono?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &FontNotFoundError{
				FontName:    tt.fontName,
				Suggestions: tt.suggestions,
			}
			if err.Error() != tt.expected {
				t.Errorf("FontNotFoundError.Error() = %q, expected %q", err.Error(), tt.expected)
			}
		})
	}
}

func TestFontInstallationError(t *testing.T) {
	err := &FontInstallationError{
		FailedCount: 2,
		TotalCount:  10,
		Details:     []string{"Font A failed", "Font B failed"},
	}
	expected := "failed to install 2 out of 10 fonts"
	if err.Error() != expected {
		t.Errorf("FontInstallationError.Error() = %q, expected %q", err.Error(), expected)
	}
}

func TestFontRemovalError(t *testing.T) {
	err := &FontRemovalError{
		FailedCount: 1,
		TotalCount:  5,
		Details:     []string{"Font A failed"},
	}
	expected := "failed to remove 1 out of 5 fonts"
	if err.Error() != expected {
		t.Errorf("FontRemovalError.Error() = %q, expected %q", err.Error(), expected)
	}
}

func TestConfigurationError(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		value    string
		hint     string
		expected string
	}{
		{
			name:     "without hint",
			field:    "scope",
			value:    "invalid",
			hint:     "",
			expected: "configuration error in field 'scope' with value 'invalid'",
		},
		{
			name:     "with hint",
			field:    "scope",
			value:    "invalid",
			hint:     "must be 'user' or 'machine'",
			expected: "configuration error in field 'scope' with value 'invalid': must be 'user' or 'machine'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &ConfigurationError{
				Field: tt.field,
				Value: tt.value,
				Hint:  tt.hint,
			}
			if err.Error() != tt.expected {
				t.Errorf("ConfigurationError.Error() = %q, expected %q", err.Error(), tt.expected)
			}
		})
	}
}

func TestElevationError(t *testing.T) {
	err := &ElevationError{
		Operation: "install",
		Platform:  "windows",
	}
	expected := "elevation required for operation 'install' on platform 'windows'"
	if err.Error() != expected {
		t.Errorf("ElevationError.Error() = %q, expected %q", err.Error(), expected)
	}
}
