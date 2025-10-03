package repo

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
