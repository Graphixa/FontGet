package repo

import (
	"testing"
)

func TestFontIDSlug(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"fontshare.satoshi", "satoshi"},
		{"google.roboto", "roboto"},
		{"nerdfonts.fira-code", "fira-code"},
		{"source.family.extra", "family.extra"},
		{"noperiod", "noperiod"},
		{"trailing.", "trailing."},
		{"", ""},
	}
	for _, tt := range tests {
		if got := fontIDSlug(tt.id); got != tt.want {
			t.Errorf("fontIDSlug(%q) = %q, want %q", tt.id, got, tt.want)
		}
	}
}

func TestCalculateMatchScore_IDMatching(t *testing.T) {
	cfg := DefaultSearchConfig()
	var r *Repository
	font := FontInfo{}

	scoreOf := func(matchType string) int {
		switch matchType {
		case "exact":
			return cfg.BaseScore + cfg.MatchBonuses.ExactMatch
		case "prefix":
			return cfg.BaseScore + cfg.MatchBonuses.PrefixMatch
		case "contains":
			return cfg.BaseScore + cfg.MatchBonuses.ContainsMatch
		case "id-prefix":
			return cfg.BaseScore + cfg.MatchBonuses.IDPrefixMatch
		case "id-contains":
			return cfg.BaseScore + cfg.MatchBonuses.IDContainsMatch
		case "no-match":
			return 0
		default:
			t.Fatalf("unknown match type %q", matchType)
			return 0
		}
	}

	tests := []struct {
		name     string
		query    string
		fontName string
		fontID   string
		wantType string
	}{
		{
			name:     "short query does not match source prefix",
			query:    "fon",
			fontName: "satoshi",
			fontID:   "fontshare.satoshi",
			wantType: "no-match",
		},
		{
			name:     "font does not dump fontshare catalog",
			query:    "font",
			fontName: "satoshi",
			fontID:   "fontshare.satoshi",
			wantType: "no-match",
		},
		{
			name:     "google does not dump google catalog",
			query:    "google",
			fontName: "roboto",
			fontID:   "google.roboto",
			wantType: "no-match",
		},
		{
			name:     "fontsource prefix is not searchable",
			query:    "fontsource",
			fontName: "inter",
			fontID:   "fontsource.inter",
			wantType: "no-match",
		},
		{
			name:     "full slug match when name differs",
			query:    "satoshi",
			fontName: "something else",
			fontID:   "fontshare.satoshi",
			wantType: "id-prefix",
		},
		{
			name:     "slug prefix when name does not match",
			query:    "sat",
			fontName: "something else",
			fontID:   "fontshare.satoshi",
			wantType: "id-prefix",
		},
		{
			name:     "dotted query prefixes a font ID",
			query:    "fontshare.sat",
			fontName: "satoshi",
			fontID:   "fontshare.satoshi",
			wantType: "id-prefix",
		},
		{
			name:     "slug contains",
			query:    "toshi",
			fontName: "something else",
			fontID:   "fontshare.satoshi",
			wantType: "id-contains",
		},
		{
			name:     "hyphenated slug matches name-mismatched query",
			query:    "fira-code",
			fontName: "fira code",
			fontID:   "nerdfonts.fira-code",
			wantType: "id-prefix",
		},
		{
			name:     "full font ID exact",
			query:    "google.roboto",
			fontName: "roboto",
			fontID:   "google.roboto",
			wantType: "id-prefix",
		},
		{
			name:     "full font ID prefix",
			query:    "google.rob",
			fontName: "roboto",
			fontID:   "google.roboto",
			wantType: "id-prefix",
		},
		{
			name:     "full font ID does not match a different source",
			query:    "google.roboto",
			fontName: "roboto",
			fontID:   "fontshare.roboto",
			wantType: "no-match",
		},
		{
			name:     "name prefix still wins for fon",
			query:    "fon",
			fontName: "fondamento",
			fontID:   "google.fondamento",
			wantType: "prefix",
		},
		{
			name:     "name takes precedence over slug",
			query:    "roboto",
			fontName: "roboto",
			fontID:   "google.roboto",
			wantType: "exact",
		},
		{
			name:     "unprefixed ID is matched as the slug",
			query:    "custom",
			fontName: "other",
			fontID:   "customfont",
			wantType: "id-prefix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotScore, gotType := r.calculateMatchScoreWithOptions(tt.query, tt.fontName, tt.fontID, font)
			if gotType != tt.wantType {
				t.Fatalf("match type = %q, want %q (score=%d)", gotType, tt.wantType, gotScore)
			}
			wantScore := scoreOf(tt.wantType)
			if gotScore != wantScore {
				t.Fatalf("score = %d, want %d for type %q", gotScore, wantScore, tt.wantType)
			}
		})
	}
}

func TestSearchFonts_DoesNotMatchSourcePrefix(t *testing.T) {
	r := &Repository{
		manifest: &FontManifest{
			Sources: map[string]SourceInfo{
				"Fontshare": {
					Name: "Fontshare",
					Fonts: map[string]FontInfo{
						"fontshare.satoshi":       {Name: "Satoshi"},
						"fontshare.clash-display": {Name: "Clash Display"},
					},
				},
				"Google Fonts": {
					Name: "Google Fonts",
					Fonts: map[string]FontInfo{
						"google.fondamento": {Name: "Fondamento"},
						"google.roboto":     {Name: "Roboto"},
					},
				},
				"Nerd Fonts": {
					Name: "Nerd Fonts",
					Fonts: map[string]FontInfo{
						"nerdfonts.fira-code": {Name: "Fira Code"},
					},
				},
			},
		},
	}

	assertIDs := func(t *testing.T, query string, wantIDs ...string) {
		t.Helper()
		results, err := r.SearchFonts(query, "")
		if err != nil {
			t.Fatalf("SearchFonts(%q): %v", query, err)
		}
		got := make(map[string]bool, len(results))
		var gotIDs []string
		for _, res := range results {
			got[res.ID] = true
			gotIDs = append(gotIDs, res.ID)
		}
		if len(wantIDs) == 0 {
			if len(results) != 0 {
				t.Fatalf("SearchFonts(%q) = %v, want no results", query, gotIDs)
			}
			return
		}
		for _, id := range wantIDs {
			if !got[id] {
				t.Fatalf("SearchFonts(%q) = %v, missing %q", query, gotIDs, id)
			}
		}
		if len(results) != len(wantIDs) {
			t.Fatalf("SearchFonts(%q) = %v, want %v", query, gotIDs, wantIDs)
		}
	}

	assertIDs(t, "fon", "google.fondamento")
	assertIDs(t, "font")
	assertIDs(t, "satoshi", "fontshare.satoshi")
	assertIDs(t, "fira-code", "nerdfonts.fira-code")
	assertIDs(t, "google.roboto", "google.roboto")
	assertIDs(t, "fontshare.sat", "fontshare.satoshi")
}
