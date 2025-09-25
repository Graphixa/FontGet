package repo

import (
	"testing"
	"time"
)

func TestCalculateMatchScore(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		fontName string
		fontID   string
		font     FontInfo
		expected int
	}{
		{
			name:     "exact name match",
			query:    "roboto",
			fontName: "roboto",
			fontID:   "roboto",
			font:     FontInfo{Name: "Roboto"},
			expected: 100,
		},
		{
			name:     "exact id match",
			query:    "roboto",
			fontName: "roboto-sans",
			fontID:   "roboto",
			font:     FontInfo{Name: "Roboto Sans"},
			expected: 100,
		},
		{
			name:     "starts with match",
			query:    "rob",
			fontName: "roboto",
			fontID:   "roboto",
			font:     FontInfo{Name: "Roboto"},
			expected: 50,
		},
		{
			name:     "contains match",
			query:    "oto",
			fontName: "roboto",
			fontID:   "roboto",
			font:     FontInfo{Name: "Roboto"},
			expected: 25,
		},
		{
			name:     "no match",
			query:    "arial",
			fontName: "roboto",
			fontID:   "roboto",
			font:     FontInfo{Name: "Roboto"},
			expected: 0,
		},
		{
			name:     "category match",
			query:    "sans",
			fontName: "roboto",
			fontID:   "roboto",
			font:     FontInfo{Name: "Roboto", Categories: []string{"Sans Serif"}},
			expected: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal repository for testing
			r := &Repository{}
			result := r.calculateMatchScore(tt.query, tt.fontName, tt.fontID, tt.font)
			if result != tt.expected {
				t.Errorf("calculateMatchScore() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func TestSortResultsByScore(t *testing.T) {
	results := []SearchResult{
		{Name: "Arial", Score: 30},
		{Name: "Roboto", Score: 100},
		{Name: "Helvetica", Score: 50},
		{Name: "Times", Score: 30},
	}

	// Create a minimal repository for testing
	r := &Repository{}
	r.sortResultsByScore(results)

	// Check that results are sorted by score (highest first)
	expectedScores := []int{100, 50, 30, 30}
	for i, result := range results {
		if result.Score != expectedScores[i] {
			t.Errorf("sortResultsByScore() [%d].Score = %d, expected %d", i, result.Score, expectedScores[i])
		}
	}

	// Check that equal scores are sorted alphabetically
	if results[2].Name != "Arial" || results[3].Name != "Times" {
		t.Errorf("sortResultsByScore() did not sort equal scores alphabetically")
	}
}

func TestSearchCacheEntry(t *testing.T) {
	// Test that SearchCacheEntry can be created properly
	entry := SearchCacheEntry{
		Query:      "roboto",
		ExactMatch: false,
		Results: []SearchResult{
			{Name: "Roboto", Score: 100},
			{Name: "Roboto Sans", Score: 80},
		},
		Timestamp: time.Now(),
	}

	// Verify struct fields
	if entry.Query != "roboto" {
		t.Errorf("SearchCacheEntry.Query = %q, expected %q", entry.Query, "roboto")
	}
	if entry.ExactMatch != false {
		t.Errorf("SearchCacheEntry.ExactMatch = %v, expected %v", entry.ExactMatch, false)
	}
	if len(entry.Results) != 2 {
		t.Errorf("SearchCacheEntry.Results length = %d, expected %d", len(entry.Results), 2)
	}
}
