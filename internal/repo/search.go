package repo

// SearchResult represents a font search result
type SearchResult struct {
	Name       string   `json:"name"`
	ID         string   `json:"id"`
	Source     string   `json:"source"`
	SourceName string   `json:"source_name"`
	License    string   `json:"license"`
	Categories []string `json:"categories,omitempty"`
	Popularity int      `json:"popularity,omitempty"`
	Score      int      `json:"-"` // Internal score for sorting
	MatchType  string   `json:"-"` // Internal match type for debugging
}
