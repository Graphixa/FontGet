package functions

import (
	"fontget/internal/config"
	"sort"
)

// SourceItem represents a source for sorting purposes
type SourceItem struct {
	Name      string
	Prefix    string
	URL       string
	Enabled   bool
	IsBuiltIn bool
	Priority  int
}

// SortSources sorts sources by type (built-in first) then by priority order
func SortSources(sources []SourceItem) {
	sort.Slice(sources, func(i, j int) bool {
		// Built-in sources come first
		if sources[i].IsBuiltIn != sources[j].IsBuiltIn {
			return sources[i].IsBuiltIn
		}
		// Within same type, sort by priority (lower number = higher priority)
		if sources[i].Priority != sources[j].Priority {
			return sources[i].Priority < sources[j].Priority
		}
		// If priorities are equal, sort by name
		return sources[i].Name < sources[j].Name
	})
}

// GetEnabledSourcesInOrder returns enabled sources in priority order from config manifest
func GetEnabledSourcesInOrder(manifest *config.Manifest) []string {
	var sources []SourceItem

	for name, source := range manifest.Sources {
		if source.Enabled {
			sources = append(sources, SourceItem{
				Name:     name,
				Priority: source.Priority,
			})
		}
	}

	// Sort by priority
	SortSources(sources)

	var result []string
	for _, source := range sources {
		result = append(result, source.Name)
	}
	return result
}

// FindSourceIndex finds the index of a source by name
func FindSourceIndex(sources []SourceItem, name string) int {
	for i, source := range sources {
		if source.Name == name {
			return i
		}
	}
	return -1
}
