package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"fontget/internal/installations"
	"fontget/internal/platform"
	"fontget/internal/testutil"
)

func TestListCmd_FontGetInstalledFlag(t *testing.T) {
	f := listCmd.Flags().Lookup("fontget-installed")
	if f == nil {
		t.Fatal("expected --fontget-installed")
	}
	if f.Shorthand != "" {
		t.Fatalf("--fontget-installed must not have a shorthand, got %q", f.Shorthand)
	}
	if f.Usage != "Show only fonts installed by FontGet" {
		t.Fatalf("unexpected help text: %q", f.Usage)
	}
	if listCmd.Flags().ShorthandLookup("i") != nil {
		t.Fatal("do not use -i for this flag")
	}
	if listCmd.Flags().Lookup("fontget") != nil {
		t.Fatal("do not keep --fontget as an alias")
	}
	if listCmd.Flags().Lookup("managed") != nil {
		t.Fatal("do not keep --managed")
	}
}

func TestFontFileMatchesNameFilter(t *testing.T) {
	if !fontFileMatchesNameFilter("JetBrainsMono-Regular.ttf", "jet") {
		t.Fatal("expected filename substring match")
	}
	if fontFileMatchesNameFilter("Arial.ttf", "jet") {
		t.Fatal("Arial should not match jet")
	}
	if !fontFileMatchesNameFilter("Arial.ttf", "") {
		t.Fatal("empty query matches all")
	}
	if !fontFileMatchesNameFilter(filepath.Join("nested", "JetBrainsMono-Regular.ttf"), "JET") {
		t.Fatal("nested relative path should match on basename")
	}
}

func TestFilterFontRefsByName_DoesNotKeepUnrelatedFiles(t *testing.T) {
	refs := []installedFontRef{
		{fileName: "JetBrainsMono-Regular.ttf"},
		{fileName: "Arial.ttf"},
		{fileName: "JetBrainsMono-Bold.otf"},
	}
	got := filterFontRefsByName(refs, "jet")
	if len(got) != 2 {
		t.Fatalf("got %d refs, want 2 (early filename filter, not the whole OS set)", len(got))
	}
}

func TestFamiliesMatchingNameQuery_EarlyFilterAndFontIDFallback(t *testing.T) {
	families := map[string][]ParsedFont{
		"JetBrains Mono": {{Name: "JetBrainsMono-Regular.ttf", Family: "JetBrains Mono"}},
		"Arial":          {{Name: "arial.ttf", Family: "Arial"}},
		"Roboto":         {{Name: "Roboto-Regular.ttf", Family: "Roboto", FontID: "google.roboto"}},
	}
	named, ok := familiesMatchingNameQuery(families, "jet")
	if !ok {
		t.Fatal("expected name-query hit")
	}
	if len(named) != 1 {
		t.Fatalf("name filter should keep 1 family, got %d", len(named))
	}
	if _, keepArial := named["Arial"]; keepArial {
		t.Fatal("Arial should not be catalog-joined for query jet")
	}

	_, ok = familiesMatchingNameQuery(families, "google.roboto")
	if ok {
		t.Fatal("Font ID-only query must fall back to full catalog join")
	}
}

func TestFilterFontsByFamilyAndID_SubstringCase(t *testing.T) {
	families := map[string][]ParsedFont{
		"Roboto": {{Family: "Roboto", FontID: "google.roboto"}},
		"Arial":  {{Family: "Arial", FontID: "windows.arial"}},
	}
	got := filterFontsByFamilyAndID(families, "ROBO")
	if len(got) != 1 || got["Roboto"] == nil {
		t.Fatalf("family substring: %v", got)
	}
	got = filterFontsByFamilyAndID(families, "google.")
	if len(got) != 1 || got["Roboto"] == nil {
		t.Fatalf("Font ID substring: %v", got)
	}
}

func TestCollectFontGetManagedFonts_RegistryOnlySkipsVanished(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)

	present := filepath.Join(home, "Roboto-Regular.ttf")
	if err := os.WriteFile(present, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(home, "Missing-Regular.ttf")
	unrelated := filepath.Join(home, "Arial.ttf")
	if err := os.WriteFile(unrelated, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := installations.RecordInstallation(installations.RecordParams{
		FontID:             "google.roboto",
		CatalogName:        "Roboto",
		InstallationSource: "Google Fonts",
		Scope:              "user",
		Files: []installations.InstalledFontFile{
			{
				Path: present,
				SFNT: installations.SFNTSnapshot{Family: "Roboto", Style: "Regular"},
			},
			{
				Path: missing,
				SFNT: installations.SFNTSnapshot{Family: "Roboto", Style: "Regular"},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := installations.RecordInstallation(installations.RecordParams{
		FontID:             "nerd.jetbrains-mono",
		CatalogName:        "JetBrains Mono",
		InstallationSource: "Nerd Fonts",
		Scope:              "machine",
		Files: []installations.InstalledFontFile{{
			Path: present,
			SFNT: installations.SFNTSnapshot{Family: "JetBrainsMono Nerd Font", Style: "Regular"},
		}},
	}); err != nil {
		t.Fatal(err)
	}

	got, err := collectFontGetManagedFonts([]platform.InstallationScope{platform.UserScope}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("user-scope FontGet list: got %d rows %+v, want 1 (registry only, skip vanished)", len(got), got)
	}
	if got[0].Family != "Roboto" || got[0].FontID != "google.roboto" {
		t.Fatalf("row = %+v", got[0])
	}
	if got[0].Path != present {
		t.Fatalf("path = %q", got[0].Path)
	}

	all, err := collectFontGetManagedFonts([]platform.InstallationScope{platform.UserScope, platform.MachineScope}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("all-scope FontGet list: got %d, want 2 (machine registry row included)", len(all))
	}

	ttfOnly, err := collectFontGetManagedFonts([]platform.InstallationScope{platform.UserScope}, "TTF")
	if err != nil {
		t.Fatal(err)
	}
	if len(ttfOnly) != 1 {
		t.Fatalf("type filter TTF: got %d", len(ttfOnly))
	}
	otfOnly, err := collectFontGetManagedFonts([]platform.InstallationScope{platform.UserScope}, "OTF")
	if err != nil {
		t.Fatal(err)
	}
	if len(otfOnly) != 0 {
		t.Fatalf("type filter OTF should be empty, got %+v", otfOnly)
	}
}
