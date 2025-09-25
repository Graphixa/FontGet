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
}

// SortSources sorts sources by type (built-in first) then alphabetically by name
func SortSources(sources []SourceItem) {
	sort.Slice(sources, func(i, j int) bool {
		// Built-in sources come first
		if sources[i].IsBuiltIn != sources[j].IsBuiltIn {
			return sources[i].IsBuiltIn
		}
		// Within same type, sort by name
		return sources[i].Name < sources[j].Name
	})
}

// SortSourcesByEnabled sorts sources by enabled status (enabled first) then by name
func SortSourcesByEnabled(sources []SourceItem) {
	sort.Slice(sources, func(i, j int) bool {
		// Enabled sources come first
		if sources[i].Enabled != sources[j].Enabled {
			return sources[i].Enabled
		}
		// Within same enabled status, sort by name
		return sources[i].Name < sources[j].Name
	})
}

// SortSourcesByName sorts sources alphabetically by name only
func SortSourcesByName(sources []SourceItem) {
	sort.Slice(sources, func(i, j int) bool {
		return sources[i].Name < sources[j].Name
	})
}

// SortSourcesByType sorts sources by type (built-in first) then by enabled status, then by name
func SortSourcesByType(sources []SourceItem) {
	sort.Slice(sources, func(i, j int) bool {
		// Built-in sources come first
		if sources[i].IsBuiltIn != sources[j].IsBuiltIn {
			return sources[i].IsBuiltIn
		}
		// Within same type, enabled sources come first
		if sources[i].Enabled != sources[j].Enabled {
			return sources[i].Enabled
		}
		// Finally, sort by name
		return sources[i].Name < sources[j].Name
	})
}

// ConvertConfigToSourceItems converts config sources to SourceItem slice
func ConvertConfigToSourceItems(sourcesConfig *config.SourcesConfig, isBuiltInSource func(string) bool) []SourceItem {
	var sources []SourceItem
	for name, source := range sourcesConfig.Sources {
		sources = append(sources, SourceItem{
			Name:      name,
			Prefix:    source.Prefix,
			URL:       source.Path,
			Enabled:   source.Enabled,
			IsBuiltIn: isBuiltInSource(name),
		})
	}
	return sources
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
