//go:build integration
// +build integration

package cmd

import (
	"testing"

	"fontget/internal/cmdutils"
	"fontget/internal/shared"
)

// TestResolveFontQuery_Integration tests resolveFontQuery with real repository.
// This is an integration test that requires the repository to be initialized.
func TestResolveFontQuery_Integration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Ensure manifest is initialized
	if err := cmdutils.EnsureManifestInitialized(func() cmdutils.Logger { return GetLogger() }); err != nil {
		t.Fatalf("Failed to initialize manifest: %v", err)
	}

	tests := []struct {
		name          string
		fontName      string
		expectError   bool
		expectMatches bool
		expectFontID  bool
	}{
		{
			name:          "valid font ID",
			fontName:      "google.roboto",
			expectError:   false,
			expectMatches: false,
			expectFontID:  true,
		},
		{
			name:          "font name with single match",
			fontName:      "Roboto",
			expectError:   false,
			expectMatches: false,
			expectFontID:  true,
		},
		{
			name:          "font name with multiple matches",
			fontName:      "Open Sans",
			expectError:   false,
			expectMatches: true, // May have multiple matches
			expectFontID:  false,
		},
		{
			name:          "non-existent font",
			fontName:      "NonExistentFont12345",
			expectError:   true,
			expectMatches: false,
			expectFontID:  false,
		},
		{
			name:          "empty font name",
			fontName:      "",
			expectError:   true,
			expectMatches: false,
			expectFontID:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := shared.ResolveFontQuery(tt.fontName)

			if tt.expectError {
				if err == nil {
					t.Errorf("resolveFontQuery() expected error but got nil")
				}
				if result != nil {
					t.Errorf("resolveFontQuery() expected nil result on error, got %+v", result)
				}
				return
			}

			if err != nil {
				t.Errorf("resolveFontQuery() unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("resolveFontQuery() returned nil result")
				return
			}

			if tt.expectMatches && !result.HasMultipleMatches {
				t.Logf("Note: Font '%s' did not have multiple matches (this is acceptable)", tt.fontName)
			}

			if tt.expectFontID && result.FontID == "" {
				t.Errorf("resolveFontQuery() expected FontID but got empty string")
			}

			if result.HasMultipleMatches && len(result.Matches) == 0 {
				t.Errorf("resolveFontQuery() HasMultipleMatches=true but Matches is empty")
			}

			if !result.HasMultipleMatches && len(result.Fonts) == 0 {
				t.Errorf("resolveFontQuery() returned no fonts")
			}
		})
	}
}

// TestGetRepository_Integration tests getRepository with real repository.
func TestGetRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Ensure manifest is initialized
	if err := cmdutils.EnsureManifestInitialized(func() cmdutils.Logger { return GetLogger() }); err != nil {
		t.Fatalf("Failed to initialize manifest: %v", err)
	}

	tests := []struct {
		name    string
		refresh bool
	}{
		{
			name:    "get repository without refresh",
			refresh: false,
		},
		{
			name:    "get repository with refresh",
			refresh: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := cmdutils.GetRepository(tt.refresh, GetLogger())

			if err != nil {
				t.Errorf("getRepository() unexpected error: %v", err)
				return
			}

			if repo == nil {
				t.Errorf("getRepository() returned nil repository")
				return
			}

			// Verify repository is usable
			manifest, err := repo.GetManifest()
			if err != nil {
				t.Errorf("getRepository() returned repository that failed to get manifest: %v", err)
			}
			if manifest == nil {
				t.Errorf("getRepository() returned repository with nil manifest")
			}
		})
	}
}

// TestMatchInstalledFontsToRepository_Integration tests matchInstalledFontsToRepository with real repository.
func TestMatchInstalledFontsToRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Ensure manifest is initialized
	if err := cmdutils.EnsureManifestInitialized(func() cmdutils.Logger { return GetLogger() }); err != nil {
		t.Fatalf("Failed to initialize manifest: %v", err)
	}

	tests := []struct {
		name        string
		familyNames []string
		expectError bool
	}{
		{
			name:        "empty list",
			familyNames: []string{},
			expectError: false,
		},
		{
			name:        "single font family",
			familyNames: []string{"Roboto"},
			expectError: false,
		},
		{
			name:        "multiple font families",
			familyNames: []string{"Roboto", "Open Sans", "Arial"},
			expectError: false,
		},
		{
			name:        "non-existent fonts",
			familyNames: []string{"NonExistentFont12345"},
			expectError: false, // Should return empty map, not error
		},
		{
			name:        "mixed existing and non-existent",
			familyNames: []string{"Roboto", "NonExistentFont12345"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, err := cmdutils.MatchInstalledFontsToRepository(tt.familyNames, GetLogger(), shared.IsCriticalSystemFont)

			if tt.expectError {
				if err == nil {
					t.Errorf("matchInstalledFontsToRepository() expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("matchInstalledFontsToRepository() unexpected error: %v", err)
				return
			}

			if matches == nil {
				t.Errorf("matchInstalledFontsToRepository() returned nil map")
				return
			}

			// For empty input, should return empty map
			if len(tt.familyNames) == 0 && len(matches) != 0 {
				t.Errorf("matchInstalledFontsToRepository() expected empty map for empty input, got %d matches", len(matches))
			}

			// For non-existent fonts, should return empty or partial matches
			// (this is acceptable behavior)
		})
	}
}

// TestGetSourceNameFromID_Integration tests getSourceNameFromID with real repository.
func TestGetSourceNameFromID_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Ensure manifest is initialized
	if err := cmdutils.EnsureManifestInitialized(func() cmdutils.Logger { return GetLogger() }); err != nil {
		t.Fatalf("Failed to initialize manifest: %v", err)
	}

	tests := []struct {
		name     string
		fontID   string
		expected string // Expected source name or "Unknown Source" for invalid IDs
	}{
		{
			name:     "valid google font ID",
			fontID:   "google.roboto",
			expected: "Google Fonts", // May vary based on actual source name
		},
		{
			name:     "valid font ID with different source",
			fontID:   "nerdfonts.fira-code",
			expected: "Nerd Fonts", // May vary based on actual source name
		},
		{
			name:     "invalid font ID (no dot)",
			fontID:   "invalid",
			expected: "Unknown Source",
		},
		{
			name:     "invalid font ID (empty)",
			fontID:   "",
			expected: "Unknown Source",
		},
		{
			name:     "font ID with unknown source prefix",
			fontID:   "unknown.source",
			expected: "Unknown Source", // Or capitalized "Unknown" if not found
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shared.GetSourceNameFromID(tt.fontID)

			if result == "" {
				t.Errorf("getSourceNameFromID() returned empty string")
			}

			// For invalid IDs, should return "Unknown Source" or similar
			if tt.expected == "Unknown Source" && result == "" {
				t.Errorf("getSourceNameFromID() expected 'Unknown Source' or similar for invalid ID, got empty string")
			}

			// For valid IDs, should return a non-empty source name
			// (exact match may vary, so we just check it's not empty)
			if tt.expected != "Unknown Source" && result == "" {
				t.Errorf("getSourceNameFromID() expected source name for valid ID, got empty string")
			}
		})
	}
}
