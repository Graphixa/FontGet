package cmd

import (
	"fmt"
	"testing"

	"fontget/internal/cmdutils"
	"fontget/internal/output"
	"fontget/internal/platform"
	"fontget/internal/shared"
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
			result := cmdutils.ParseFontNames(tt.args)
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
		report   output.StatusReport
		expected string
	}{
		{
			name: "all zero values",
			report: output.StatusReport{
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
			report: output.StatusReport{
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
			output.PrintStatusReport(tt.report, IsVerbose())
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
			err := &shared.FontNotFoundError{
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
	err := &shared.FontInstallationError{
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
	err := &shared.FontRemovalError{
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
			err := &shared.ConfigurationError{
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
	err := &shared.ElevationError{
		Operation: "install",
		Platform:  "windows",
	}
	expected := "elevation required for operation 'install' on platform 'windows'"
	if err.Error() != expected {
		t.Errorf("ElevationError.Error() = %q, expected %q", err.Error(), expected)
	}
}

func TestGetFontFamilyNameFromFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{
			name:     "simple font name",
			filename: "Roboto-Regular.ttf",
			expected: "Roboto",
		},
		{
			name:     "font with variant",
			filename: "ABeeZee-Italic.ttf",
			expected: "ABeeZee",
		},
		{
			name:     "font with multiple hyphens",
			filename: "RobotoMono-Bold-Italic.ttf",
			expected: "RobotoMono",
		},
		{
			name:     "font without variant",
			filename: "Arial.ttf",
			expected: "Arial",
		},
		{
			name:     "font with path",
			filename: "/path/to/fonts/OpenSans-Regular.ttf",
			expected: "OpenSans",
		},
		{
			name:     "font with underscore",
			filename: "Font_Name-Regular.ttf",
			expected: "Font_Name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shared.GetFontFamilyNameFromFilename(tt.filename)
			if result != tt.expected {
				t.Errorf("GetFontFamilyNameFromFilename(%q) = %q, expected %q", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestGetDisplayNameFromFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{
			name:     "simple font name",
			filename: "Roboto-Regular.ttf",
			expected: "Roboto Regular",
		},
		{
			name:     "font with variant",
			filename: "ABeeZee-Italic.ttf",
			expected: "ABeeZee Italic",
		},
		{
			name:     "font with multiple hyphens",
			filename: "RobotoMono-Bold-Italic.ttf",
			expected: "Roboto Mono Bold Italic",
		},
		{
			name:     "font without variant",
			filename: "Arial.ttf",
			expected: "Arial",
		},
		{
			name:     "font with path (camelCase converted)",
			filename: "/path/to/fonts/OpenSans-Regular.ttf",
			expected: "Open Sans Regular", // camelCase is converted to spaced format
		},
		{
			name:     "font with underscore",
			filename: "Font_Name-Regular.ttf",
			expected: "Font_Name Regular",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shared.GetDisplayNameFromFilename(tt.filename)
			if result != tt.expected {
				t.Errorf("GetDisplayNameFromFilename(%q) = %q, expected %q", tt.filename, result, tt.expected)
			}
		})
	}
}

// mockFontManager is a mock implementation of platform.FontManager for testing
type mockFontManager struct {
	isElevatedValue bool
	isElevatedError error
}

func (m *mockFontManager) InstallFont(fontPath string, scope platform.InstallationScope, force bool) error {
	return nil
}

func (m *mockFontManager) RemoveFont(fontName string, scope platform.InstallationScope) error {
	return nil
}

func (m *mockFontManager) GetFontDir(scope platform.InstallationScope) string {
	return "/test/fonts"
}

func (m *mockFontManager) RequiresElevation(scope platform.InstallationScope) bool {
	return scope == platform.MachineScope
}

func (m *mockFontManager) IsElevated() (bool, error) {
	return m.isElevatedValue, m.isElevatedError
}

func (m *mockFontManager) GetElevationCommand() (string, []string, error) {
	return "", nil, nil
}

func TestAutoDetectScope(t *testing.T) {
	// Logger is now optional in autoDetectScope, so no initialization needed

	tests := []struct {
		name          string
		isElevated    bool
		isElevatedErr error
		defaultScope  string
		elevatedScope string
		expected      string
		expectError   bool
	}{
		{
			name:          "elevated returns elevated scope",
			isElevated:    true,
			isElevatedErr: nil,
			defaultScope:  "user",
			elevatedScope: "machine",
			expected:      "machine",
			expectError:   false,
		},
		{
			name:          "not elevated returns user",
			isElevated:    false,
			isElevatedErr: nil,
			defaultScope:  "user",
			elevatedScope: "machine",
			expected:      "user",
			expectError:   false,
		},
		{
			name:          "elevation check error returns default scope",
			isElevated:    false,
			isElevatedErr: fmt.Errorf("elevation check failed"),
			defaultScope:  "user",
			elevatedScope: "machine",
			expected:      "user",
			expectError:   false,
		},
		{
			name:          "custom default scope on error",
			isElevated:    false,
			isElevatedErr: fmt.Errorf("elevation check failed"),
			defaultScope:  "custom",
			elevatedScope: "machine",
			expected:      "custom",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFM := &mockFontManager{
				isElevatedValue: tt.isElevated,
				isElevatedError: tt.isElevatedErr,
			}

			result, err := platform.AutoDetectScope(mockFM, tt.defaultScope, tt.elevatedScope, nil)

			if tt.expectError && err == nil {
				t.Errorf("autoDetectScope() expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("autoDetectScope() unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("autoDetectScope() = %q, expected %q", result, tt.expected)
			}
		})
	}
}
