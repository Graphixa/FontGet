package repo

import (
	"time"
)

// SearchResult represents a font search result
type SearchResult struct {
	Name       string   `json:"name"`
	ID         string   `json:"id"`
	Source     string   `json:"source"`
	SourceName string   `json:"source_name"`
	License    string   `json:"license"`
	Categories []string `json:"categories,omitempty"`
	Score      int      `json:"-"` // Internal score for sorting
}

// SearchCacheEntry represents a cached search result
type SearchCacheEntry struct {
	Query      string         `json:"query"`
	ExactMatch bool           `json:"exact_match"`
	Results    []SearchResult `json:"results"`
	Timestamp  time.Time      `json:"timestamp"`
}
