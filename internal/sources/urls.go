package sources

// FontGetSourcesURLs contains all hardcoded URLs to the FontGet-Sources repository
// This is the single source of truth for all FontGet-Sources URLs
const (
	// BaseURL is the base URL for the FontGet-Sources repository
	BaseURL = "https://raw.githubusercontent.com/Graphixa/FontGet-Sources/main/sources"

	// GoogleFontsURL is the URL for Google Fonts source data
	GoogleFontsURL = BaseURL + "/google-fonts.json"

	// NerdFontsURL is the URL for Nerd Fonts source data
	NerdFontsURL = BaseURL + "/nerd-fonts.json"

	// FontSquirrelURL is the URL for Font Squirrel source data
	FontSquirrelURL = BaseURL + "/font-squirrel.json"
)

// DefaultSources returns the default source configuration
// This is the single source of truth for default source settings
func DefaultSources() map[string]SourceInfo {
	return map[string]SourceInfo{
		"Google Fonts": {
			URL:      GoogleFontsURL,
			Prefix:   "google",
			Enabled:  true,
			Filename: "google-fonts.json",
			Priority: 1, // Highest priority - primary source
		},
		"Nerd Fonts": {
			URL:      NerdFontsURL,
			Prefix:   "nerd",
			Enabled:  true,
			Filename: "nerd-fonts.json",
			Priority: 2, // Second priority - secondary source
		},
		"Font Squirrel": {
			URL:      FontSquirrelURL,
			Prefix:   "squirrel",
			Enabled:  true,
			Filename: "font-squirrel.json",
			Priority: 3, // Third priority
		},
	}
}

// SourceInfo represents information about a font source
type SourceInfo struct {
	URL      string `json:"url"`
	Prefix   string `json:"prefix"`
	Enabled  bool   `json:"enabled"`
	Filename string `json:"filename"`
	Priority int    `json:"priority"`
}
