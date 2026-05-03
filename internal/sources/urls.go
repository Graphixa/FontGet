package sources

import "sort"

// FontGetSourcesURLs contains all hardcoded URLs to the FontGet-Sources repository
// This is the single source of truth for all FontGet-Sources URLs
const (
	// BaseURL is the base URL for the FontGet-Sources repository
	BaseURL = "https://raw.githubusercontent.com/Graphixa/FontGet-Sources/main/sources"

	// GoogleFontsURL is the URL for Google Fonts source data
	GoogleFontsURL = BaseURL + "/google-fonts.json"

	// NerdFontsURL is the URL for Nerd Fonts source data
	NerdFontsURL = BaseURL + "/nerd-fonts.json"

	// LeagueOfMoveableTypeURL is the URL for The League of Moveable Type source data
	LeagueOfMoveableTypeURL = BaseURL + "/league-of-moveable-type.json"

	// FontshareURL is the URL for Fontshare source data
	FontshareURL = BaseURL + "/fontshare.json"

	// FontsourceURL is the URL for Fontsource source data
	FontsourceURL = BaseURL + "/fontsource.json"

	// FontSquirrelURL is the URL for Font Squirrel source data
	FontSquirrelURL = BaseURL + "/font-squirrel.json"
)

// DefaultSources returns the default source configuration.
// This is the single source of truth for default source settings.
// When adding a built-in source here, also update config.BuiltInSourceNames,
// cmd/import BuiltInSources, internal/repo sourcePriority, and related tests.
func DefaultSources() map[string]SourceInfo {
	return map[string]SourceInfo{
		"Google Fonts": {
			URL:      GoogleFontsURL,
			Prefix:   "google",
			Enabled:  true,
			Filename: "google-fonts.json",
			Priority: 1,
		},
		"Nerd Fonts": {
			URL:      NerdFontsURL,
			Prefix:   "nerd",
			Enabled:  true,
			Filename: "nerd-fonts.json",
			Priority: 2,
		},
		"The League of Moveable Type": {
			URL:      LeagueOfMoveableTypeURL,
			Prefix:   "league",
			Enabled:  true,
			Filename: "league-of-moveable-type.json",
			Priority: 3,
		},
		"Fontshare": {
			URL:      FontshareURL,
			Prefix:   "fontshare",
			Enabled:  true,
			Filename: "fontshare.json",
			Priority: 4,
		},
		"Fontsource": {
			URL:      FontsourceURL,
			Prefix:   "fontsource",
			Enabled:  true,
			Filename: "fontsource.json",
			Priority: 5,
		},
		"Font Squirrel": {
			URL:      FontSquirrelURL,
			Prefix:   "squirrel",
			Enabled:  true,
			Filename: "font-squirrel.json",
			Priority: 6,
		},
	}
}

// DefaultSourceNamesInPriorityOrder returns built-in manifest source names sorted by Priority (lower first).
func DefaultSourceNamesInPriorityOrder() []string {
	def := DefaultSources()
	type pair struct {
		name     string
		priority int
	}
	list := make([]pair, 0, len(def))
	for name, info := range def {
		list = append(list, pair{name: name, priority: info.Priority})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].priority != list[j].priority {
			return list[i].priority < list[j].priority
		}
		return list[i].name < list[j].name
	})
	out := make([]string, len(list))
	for i := range list {
		out[i] = list[i].name
	}
	return out
}

// SourceInfo represents information about a font source
type SourceInfo struct {
	URL      string `json:"url"`
	Prefix   string `json:"prefix"`
	Enabled  bool   `json:"enabled"`
	Filename string `json:"filename"`
	Priority int    `json:"priority"`
}
