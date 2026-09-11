package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"fontget/internal/cmdutils"
	"fontget/internal/components"
	"fontget/internal/installations"
	"fontget/internal/output"
	"fontget/internal/platform"
	"fontget/internal/repo"
	"fontget/internal/ui"

	"github.com/spf13/cobra"
)

// ParsedFont represents a parsed font file with metadata
type ParsedFont struct {
	Name   string
	Family string
	Style  string
	Type   string
	Scope  string
	// Path is the absolute path to the font file on disk (used for installation registry merge; not shown in tables).
	Path string
	// Repository match fields
	FontID     string
	License    string
	Categories []string
	Source     string
}

var listCmd = &cobra.Command{
	Use:          "list [query]",
	Short:        "List installed fonts",
	SilenceUsage: true,
	Long: `List fonts installed on your system.

By default, shows fonts from both user and system-wide installations.
Results can be filtered by font family name, Font ID, type, or scope.
Use --fontget-installed to show only fonts installed by FontGet (from the installation registry).

The query parameter can match either font family names (e.g., "Roboto") or Font IDs (e.g., "google.roboto").

Name queries (the common case, e.g. "jet") are matched against SFNT family names and file names before catalog join so the whole OS font set is not parsed or matched. If that produces no rows, list falls back to a full collect + catalog join and then filters by Font ID (e.g. "google.roboto", or a source prefix like "google" when no family/file name contains the query).`,
	Example: `  fontget list
  fontget list "jet"
  fontget list roboto -t ttf
  fontget list "google.roboto"
  fontget list "fira" -x
  fontget list -s user
  fontget list --fontget-installed`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Query is optional - no validation needed
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		GetLogger().Info("Starting font list operation")

		if err := cmdutils.EnsureManifestInitialized(func() cmdutils.Logger { return GetLogger() }); err != nil {
			return err
		}

		fm, err := cmdutils.CreateFontManager(func() cmdutils.Logger { return GetLogger() })
		if err != nil {
			return err
		}

		scope, _ := cmd.Flags().GetString("scope")
		typeFilter, _ := cmd.Flags().GetString("type")
		showVariants, _ := cmd.Flags().GetBool("expand")
		fontgetOnly, _ := cmd.Flags().GetBool("fontget-installed")

		// Get query from positional argument
		var familyFilter string
		if len(args) > 0 {
			familyFilter = args[0]
		}

		// Log parameters (always log to file)
		scopeDisplay := "all"
		if scope != "" {
			scopeDisplay = scope
		}
		GetLogger().Info("List parameters - Scope: %s, Type filter: %s, Family filter: %s, Show variants: %v, FontGet-managed: %v", scopeDisplay, typeFilter, familyFilter, showVariants, fontgetOnly)

		// Verbose output for parameters
		output.GetVerbose().Info("Scope: %s", scopeDisplay)
		if typeFilter != "" {
			output.GetVerbose().Info("Type filter: %s", typeFilter)
		}
		if familyFilter != "" {
			output.GetVerbose().Info("Family filter: %s", familyFilter)
		}
		if fontgetOnly {
			output.GetVerbose().Info("Showing only fonts installed by FontGet")
		}
		// Verbose section ends with blank line per spacing framework (only if verbose was shown)
		if output.IsVerboseOutputEnabled() {
			fmt.Println()
		}

		var scopes []platform.InstallationScope
		// Default to "all" (both scopes) if no scope specified
		if scope == "" {
			scopes = []platform.InstallationScope{platform.UserScope, platform.MachineScope}
		} else {
			// Validate scope - only "user" or "machine" are valid
			installScope := platform.InstallationScope(scope)
			if installScope != platform.UserScope && installScope != platform.MachineScope {
				err := fmt.Errorf("invalid scope '%s'. Valid options are: user, machine", scope)
				GetLogger().Error("Invalid scope provided: %s", scope)
				output.GetVerbose().Error("%v", err)
				output.GetDebug().Error("Invalid scope provided: '%s'", scope)
				return err
			}
			// machine scope requires elevation
			if installScope == platform.MachineScope {
				if err := cmdutils.CheckElevation(cmd, fm, installScope); err != nil {
					if errors.Is(err, cmdutils.ErrElevationRequired) {
						return nil
					}
					GetLogger().Error("Failed to check elevation: %v", err)
					output.GetVerbose().Error("%v", err)
					output.GetDebug().Error("checkElevation() failed: %v", err)
					return fmt.Errorf("unable to verify system permissions: %v", err)
				}
			}
			scopes = []platform.InstallationScope{installScope}
		}
		// Debug: initial parameter dump removed to reduce noise

		// Variables to capture from spinner operation
		var fonts []ParsedFont
		var families map[string][]ParsedFont
		var matches map[string]*repo.InstalledFontMatch
		var names []string
		var filteredFamilies map[string][]ParsedFont

		runWork := func() error {
			var workErr error
			t0 := time.Now()

			tCollect := time.Now()
			if fontgetOnly {
				fonts, workErr = collectFontGetManagedFonts(scopes, typeFilter)
			} else {
				fonts, workErr = collectFontsForQuery(scopes, fm, typeFilter, familyFilter)
			}
			if workErr != nil {
				GetLogger().Error("Failed to collect fonts: %v", workErr)
				output.GetVerbose().Error("%v", workErr)
				output.GetDebug().Error("collectFonts() failed: %v", workErr)
				return fmt.Errorf("unable to read installed fonts: %w", workErr)
			}
			output.GetDebug().State("Collected %d font files before filtering", len(fonts))
			output.GetDebug().State("Timing: collectFonts=%s", time.Since(tCollect))

			tGroup := time.Now()
			families = groupByFamily(fonts)
			if !fontgetOnly {
				// Name queries (e.g. "jet") join only matching families. If nothing matches
				// by SFNT family / file name, collectFontsForQuery already fell back to the
				// full file set so Font ID substrings can still match after catalog join.
				if named, ok := familiesMatchingNameQuery(families, familyFilter); ok {
					families = named
				}
			}
			allFamilyNames := make([]string, 0, len(families))
			for k := range families {
				allFamilyNames = append(allFamilyNames, k)
			}
			output.GetDebug().State("Grouped %d font files into %d unique families", len(fonts), len(allFamilyNames))
			output.GetDebug().State("Timing: groupByFamily=%s", time.Since(tGroup))

			output.GetVerbose().Info("Matching installed fonts to repository...")
			output.GetDebug().State("Matching %d font families against repository", len(allFamilyNames))
			tMatch := time.Now()
			if fontgetOnly {
				applyCatalogMatchesByFontID(families)
				matches = make(map[string]*repo.InstalledFontMatch)
			} else {
				matches, workErr = repo.MatchAllInstalledFonts(allFamilyNames, nil)
				if workErr != nil {
					output.GetVerbose().Error("%v", workErr)
					output.GetDebug().Error("repo.MatchAllInstalledFonts() failed: %v", workErr)
					matches = make(map[string]*repo.InstalledFontMatch)
				} else {
					matchCount := 0
					for _, match := range matches {
						if match != nil {
							matchCount++
						}
					}
					output.GetVerbose().Info("Found %d matches out of %d installed fonts", matchCount, len(allFamilyNames))
					if output.IsVerboseOutputEnabled() {
						fmt.Println()
					}
				}
				applyRepositoryMatches(families, matches)
			}
			output.GetDebug().State("Timing: MatchAllInstalledFonts=%s", time.Since(tMatch))

			if !fontgetOnly {
				tReg := time.Now()
				if reg, regErr := installations.Load(); regErr != nil {
					output.GetDebug().Error("installation registry: %v", regErr)
				} else {
					mergeInstallationRegistryIntoFamilies(families, reg)
				}
				output.GetDebug().State("Timing: mergeInstallationRegistry=%s", time.Since(tReg))
			}
			// Apply filters (now that Font IDs are populated)
			// Filter by family name and Font ID (type filter already applied during collection)
			// Apply filter using helper function
			tFilter := time.Now()
			filteredFamilies = filterFontsByFamilyAndID(families, familyFilter)

			// Case-insensitive sort: precompute one ToLower per family (avoids O(n log n) repeated work).
			type nameSortKey struct {
				name, key string
			}
			keys := make([]nameSortKey, 0, len(filteredFamilies))
			for k := range filteredFamilies {
				keys = append(keys, nameSortKey{name: k, key: strings.ToLower(k)})
			}
			sort.SliceStable(keys, func(i, j int) bool { return keys[i].key < keys[j].key })
			names = make([]string, len(keys))
			for i := range keys {
				names[i] = keys[i].name
			}
			output.GetDebug().State("Timing: filter+sort=%s", time.Since(tFilter))
			output.GetDebug().State("After filtering: %d font families remaining", len(names))

			output.GetDebug().State("Timing: total list work=%s", time.Since(t0))
			return nil
		}

		// Wrap the main work (collecting, grouping, matching, filtering) in a spinner.
		// In --debug, disable the spinner so debug lines don't get mangled by \r spinner rendering.
		if output.IsDebugOutputEnabled() {
			err = runWork()
		} else {
			err = ui.RunSpinner("Loading...", "", runWork)
		}

		if err != nil {
			// Spinner already showed the error message
			return err
		}

		if len(names) == 0 {

			if familyFilter != "" || typeFilter != "" || fontgetOnly {
				filterInfo := fmt.Sprintf("Found 0 font families installed matching '%s'", ui.QueryText.Render(familyFilter))
				if familyFilter == "" && typeFilter == "" {
					filterInfo = "Found 0 font families installed"
				}
				if typeFilter != "" {
					filterInfo += fmt.Sprintf(" | Filtered by type: '%s'", ui.QueryText.Render(typeFilter))
				}
				if fontgetOnly {
					filterInfo += " | FontGet-managed"
				}
				fmt.Printf("%s\n", filterInfo)
				fmt.Println()
			} else {
				fmt.Printf("%s\n", ui.Text.Render("Found 0 font families installed"))
				fmt.Println()
			}
			return nil
		}

		// Log completion
		GetLogger().Info("List operation complete - Found %d font families", len(names))

		if familyFilter != "" || typeFilter != "" || fontgetOnly {
			filterInfo := fmt.Sprintf("Found %d font families installed matching '%s'", len(names), ui.QueryText.Render(familyFilter))
			if familyFilter == "" && typeFilter == "" {
				filterInfo = fmt.Sprintf("Found %d font families installed", len(names))
			}
			if typeFilter != "" {
				filterInfo += fmt.Sprintf(" | Filtered by type: '%s'", ui.QueryText.Render(typeFilter))
			}
			if fontgetOnly {
				filterInfo += " | FontGet-managed"
			}
			fmt.Printf("%s\n", filterInfo)
			fmt.Println()
		} else {
			info := fmt.Sprintf("Found %d font families installed", len(names))
			fmt.Printf("%s\n", ui.Text.Render(info))
			fmt.Println()
		}

		// Build table rows with priority: Font ID > Font Name > Categories > License > Type > Scope > Source
		var tableRows [][]string
		for _, fam := range names {
			group := filteredFamilies[fam]
			sort.Slice(group, func(i, j int) bool { return group[i].Style < group[j].Style })
			rep := group[0]

			// Format Font ID (empty string if not available)
			fontID := rep.FontID

			// Format License (empty string if not available)
			license := rep.License

			// Format Categories (first category only, like search command)
			categories := ""
			if len(rep.Categories) > 0 {
				categories = rep.Categories[0]
			}

			// Format Source (empty string if not available)
			source := rep.Source

			// Build row: Font Name, Font ID, Categories, License, Type, Scope, Source
			row := []string{
				fam,
				fontID,
				categories,
				license,
				rep.Type,
				rep.Scope,
				source,
			}
			tableRows = append(tableRows, row)

			// Add variant rows if requested
			if showVariants {
				uniq := map[string]bool{}
				var styles []string
				for _, f := range group {
					if !uniq[f.Style] {
						uniq[f.Style] = true
						styles = append(styles, f.Style)
					}
				}
				sort.Strings(styles)
				for _, s := range styles {
					variantRow := []string{
						fmt.Sprintf("  ↳ %s", s),
						"", // Font ID
						"", // Categories
						"", // License
						"", // Type
						"", // Scope
						"", // Source
					}
					tableRows = append(tableRows, variantRow)
				}
			}
		}

		// Render table with priority configuration
		tableConfig := components.TableConfig{
			Columns: []components.ColumnConfig{
				{Header: "Font Name", Truncatable: true, Hideable: false, MinWidth: 18, Priority: 2, PercentWidth: 20.0},
				{Header: "Font ID", Truncatable: false, Hideable: false, Priority: 1, PercentWidth: 28.0}, // Highest priority, don't trim
				{Header: "Categories", Truncatable: true, MaxWidth: 14, Hideable: true, Priority: 3, PercentWidth: 12.0},
				{Header: "License", Truncatable: true, MaxWidth: 8, Hideable: true, Priority: 4, PercentWidth: 8.0},
				{Header: "Type", Truncatable: true, Hideable: true, Priority: 6, PercentWidth: 6.0},
				{Header: "Scope", Truncatable: true, Hideable: true, Priority: 5, PercentWidth: 10.0},
				{Header: "Source", Truncatable: true, MaxWidth: 14, MinWidth: 12, Hideable: true, Priority: 7, PercentWidth: 16.0}, // Lowest priority
			},
			Rows:     tableRows,
			Width:    0,   // Auto-detect terminal width
			MaxWidth: 120, // Maximum width
			Mode:     components.TableModeStatic,
			Padding:  1, // Default padding
		}

		fmt.Println(components.RenderStaticTable(tableConfig))

		fmt.Println()
		return nil
	},
}

type installedFontRef struct {
	fontPath string
	fileName string
	scope    platform.InstallationScope
}

// collectFonts collects font files from the specified scopes
func collectFonts(scopes []platform.InstallationScope, fm platform.FontManager, typeFilter string, suppressVerbose ...bool) ([]ParsedFont, error) {
	shouldSuppressVerbose := false
	if len(suppressVerbose) > 0 {
		shouldSuppressVerbose = suppressVerbose[0]
	}
	refs, err := enumerateInstalledFonts(scopes, fm, typeFilter, shouldSuppressVerbose)
	if err != nil {
		return nil, err
	}
	parsed := parseInstalledFontRefs(refs)
	if !shouldSuppressVerbose {
		output.GetVerbose().Info("Scan complete: parsed %d files across %d scope(s)", len(parsed), len(scopes))
		if output.IsVerboseOutputEnabled() {
			fmt.Println()
		}
	}
	return parsed, nil
}

// collectFontsForQuery enumerates OS font dirs once. For name queries it parses only
// files whose names contain the query; if no SFNT family then matches, it falls back
// to parsing every file so Font ID substring matching can still run.
func collectFontsForQuery(scopes []platform.InstallationScope, fm platform.FontManager, typeFilter, familyFilter string) ([]ParsedFont, error) {
	refs, err := enumerateInstalledFonts(scopes, fm, typeFilter, false)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(familyFilter) == "" {
		parsed := parseInstalledFontRefs(refs)
		output.GetVerbose().Info("Scan complete: parsed %d files across %d scope(s)", len(parsed), len(scopes))
		if output.IsVerboseOutputEnabled() {
			fmt.Println()
		}
		return parsed, nil
	}
	nameHits := filterFontRefsByName(refs, familyFilter)
	parsed := parseInstalledFontRefs(nameHits)
	if _, ok := familiesMatchingNameQuery(groupByFamily(parsed), familyFilter); ok {
		output.GetDebug().State("List name filter: parsed %d of %d files for query %q", len(nameHits), len(refs), familyFilter)
		output.GetVerbose().Info("Scan complete: parsed %d files across %d scope(s)", len(parsed), len(scopes))
		if output.IsVerboseOutputEnabled() {
			fmt.Println()
		}
		return parsed, nil
	}
	output.GetDebug().State("List name filter missed family names for %q; falling back to full parse for Font ID matching", familyFilter)
	parsed = parseInstalledFontRefs(refs)
	output.GetVerbose().Info("Scan complete: parsed %d files across %d scope(s)", len(parsed), len(scopes))
	if output.IsVerboseOutputEnabled() {
		fmt.Println()
	}
	return parsed, nil
}

func filterFontRefsByName(refs []installedFontRef, query string) []installedFontRef {
	if strings.TrimSpace(query) == "" {
		return refs
	}
	out := make([]installedFontRef, 0, len(refs))
	for _, r := range refs {
		if fontFileMatchesNameFilter(r.fileName, query) {
			out = append(out, r)
		}
	}
	return out
}

func fontFileMatchesNameFilter(fileName, query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return true
	}
	return strings.Contains(strings.ToLower(filepath.Base(fileName)), q)
}

func enumerateInstalledFonts(scopes []platform.InstallationScope, fm platform.FontManager, typeFilter string, shouldSuppressVerbose bool) ([]installedFontRef, error) {
	typeFilterUpper := ""
	if typeFilter != "" {
		typeFilterUpper = strings.ToUpper(typeFilter)
	}
	var refs []installedFontRef
	for _, scope := range scopes {
		fontDir := fm.GetFontDir(scope)
		if !shouldSuppressVerbose {
			output.GetVerbose().Info("Scanning %s scope: %s", scope, fontDir)
		}
		output.GetDebug().State("Checking font directory: %s (scope: %s)", fontDir, scope)

		if _, err := os.Stat(fontDir); os.IsNotExist(err) {
			output.GetVerbose().Warning("Font directory does not exist: %s", fontDir)
			output.GetDebug().Warning("Directory %s does not exist, skipping", fontDir)
			continue
		}

		if f, err := os.Open(fontDir); err != nil {
			if os.IsPermission(err) {
				output.GetVerbose().Warning("No read permission for font directory: %s", fontDir)
				output.GetDebug().Error("Permission denied accessing %s: %v", fontDir, err)
				if scope == platform.MachineScope {
					return nil, fmt.Errorf("insufficient permissions to read %s. Try running with sudo or use --scope user", fontDir)
				}
				output.GetDebug().Warning("Permission denied for user scope directory (unusual), continuing...")
				continue
			}
			output.GetDebug().Error("Unable to access font directory %s: %v", fontDir, err)
			return nil, fmt.Errorf("unable to access font directory %s: %w", fontDir, err)
		} else {
			_ = f.Close()
		}

		names, err := platform.ListInstalledFonts(fontDir)
		if err != nil {
			output.GetDebug().Error("platform.ListInstalledFonts() failed for %s: %v", fontDir, err)
			return nil, fmt.Errorf("failed to list fonts in %s: %w", fontDir, err)
		}
		if !shouldSuppressVerbose {
			output.GetVerbose().Info("Found %d files in %s", len(names), fontDir)
		}
		for _, name := range names {
			fileExt := strings.ToUpper(strings.TrimPrefix(filepath.Ext(name), "."))
			if typeFilterUpper != "" && fileExt != typeFilterUpper {
				continue
			}
			refs = append(refs, installedFontRef{
				fontPath: filepath.Join(fontDir, name),
				fileName: name,
				scope:    scope,
			})
		}
	}
	return refs, nil
}

func parseInstalledFontRefs(refs []installedFontRef) []ParsedFont {
	workerCount := runtime.GOMAXPROCS(0)
	if workerCount < 2 {
		workerCount = 2
	}
	if workerCount > 8 {
		workerCount = 8
	}

	jobs := make(chan installedFontRef, workerCount*4)
	var outMu sync.Mutex
	parsed := make([]ParsedFont, 0, len(refs))

	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				parsedFont := buildParsedFont(job.fontPath, job.fileName, job.scope)
				if parsedFont != nil {
					outMu.Lock()
					parsed = append(parsed, *parsedFont)
					outMu.Unlock()
				}
			}
		}()
	}
	for _, ref := range refs {
		jobs <- ref
	}
	close(jobs)
	wg.Wait()
	return parsed
}

// collectFontGetManagedFonts builds list rows from the FontGet installation registry
// instead of scanning OS font directories. Missing files are skipped.
func collectFontGetManagedFonts(scopes []platform.InstallationScope, typeFilter string) ([]ParsedFont, error) {
	reg, err := installations.Load()
	if err != nil {
		return nil, fmt.Errorf("unable to read FontGet installation registry: %w", err)
	}
	if reg == nil || len(reg.Installations) == 0 {
		return nil, nil
	}

	typeFilterUpper := ""
	if typeFilter != "" {
		typeFilterUpper = strings.ToUpper(typeFilter)
	}
	allowed := make(map[string]bool, len(scopes))
	for _, s := range scopes {
		allowed[string(s)] = true
	}

	output.GetVerbose().Info("Listing FontGet-managed installs from the installation registry (%d entries)", len(reg.Installations))

	var parsed []ParsedFont
	for _, inst := range reg.Installations {
		if inst == nil {
			continue
		}
		instScope := strings.TrimSpace(inst.Scope)
		if instScope != "" && !allowed[instScope] {
			continue
		}
		for _, face := range inst.FlatFiles() {
			p := strings.TrimSpace(face.Path)
			if p == "" {
				continue
			}
			if _, err := os.Stat(p); err != nil {
				output.GetDebug().Warning("Skipping vanished FontGet install file: %s (%v)", p, err)
				continue
			}
			fileName := filepath.Base(p)
			fileExt := strings.ToUpper(strings.TrimPrefix(filepath.Ext(fileName), "."))
			if typeFilterUpper != "" && fileExt != typeFilterUpper {
				continue
			}
			family := strings.TrimSpace(face.SFNT.Family)
			style := strings.TrimSpace(face.SFNT.Style)
			scope := instScope
			if family == "" || style == "" {
				if pf := buildParsedFont(p, fileName, platform.InstallationScope(scope)); pf != nil {
					if family == "" {
						family = pf.Family
					}
					if style == "" {
						style = pf.Style
					}
					if scope == "" {
						scope = pf.Scope
					}
					if fileExt == "" {
						fileExt = pf.Type
					}
				}
			}
			if family == "" {
				family = strings.TrimSuffix(fileName, filepath.Ext(fileName))
			}
			if style == "" {
				style = "Regular"
			}
			if scope == "" {
				scope = string(platform.UserScope)
			}
			parsed = append(parsed, ParsedFont{
				Name:   fileName,
				Family: family,
				Style:  style,
				Type:   fileExt,
				Scope:  scope,
				Path:   p,
				FontID: strings.TrimSpace(inst.FontID),
				Source: strings.TrimSpace(inst.InstallationSource),
			})
		}
	}
	return parsed, nil
}

func applyRepositoryMatches(families map[string][]ParsedFont, matches map[string]*repo.InstalledFontMatch) {
	for familyName, fontGroup := range families {
		if match, exists := matches[familyName]; exists && match != nil {
			for i := range fontGroup {
				fontGroup[i].FontID = match.FontID
				fontGroup[i].License = match.License
				fontGroup[i].Categories = match.Categories
				fontGroup[i].Source = match.Source
			}
			families[familyName] = fontGroup
		}
	}
}

func applyCatalogMatchesByFontID(families map[string][]ParsedFont) {
	seen := make(map[string]*repo.InstalledFontMatch)
	for familyName, group := range families {
		for i := range group {
			id := strings.TrimSpace(group[i].FontID)
			if id == "" {
				continue
			}
			key := strings.ToLower(id)
			match, cached := seen[key]
			if !cached {
				var err error
				match, err = repo.MatchRepositoryFontByID(id)
				if err != nil {
					output.GetDebug().Error("MatchRepositoryFontByID(%s): %v", id, err)
					match = nil
				}
				seen[key] = match
			}
			if match == nil {
				continue
			}
			group[i].FontID = match.FontID
			group[i].License = match.License
			group[i].Categories = match.Categories
			group[i].Source = match.Source
		}
		families[familyName] = group
	}
}

// buildParsedFont extracts font metadata from a file path and builds a ParsedFont struct
// Returns nil if the font file is invalid and should be skipped
func buildParsedFont(fontPath, fileName string, scope platform.InstallationScope) *ParsedFont {
	// Extract file extension for type
	fileExt := strings.ToUpper(strings.TrimPrefix(filepath.Ext(fileName), "."))

	// Try to extract metadata from the font file
	md, err := platform.ExtractFontMetadata(fontPath)
	family := ""
	style := ""

	if err == nil {
		// Prefer typographic names for display when present
		if md.TypographicFamily != "" {
			family = md.TypographicFamily
		} else {
			family = md.FamilyName
		}
		if md.TypographicStyle != "" {
			style = md.TypographicStyle
		} else {
			style = md.StyleName
		}
	} else {
		// Check if this is an invalid font file error - if so, skip it
		if errors.Is(err, platform.ErrInvalidFontFile) {
			// Log debug info but don't include in list
			output.GetDebug().Warning("Skipping invalid font file: %s (%v)", fontPath, err)
			return nil
		}
		// For other errors (e.g., parsing issues but file might still be valid), use filename fallback
		// This handles edge cases where metadata extraction fails but file structure is OK
		base := strings.TrimSuffix(fileName, filepath.Ext(fileName))
		family = base
		style = "Regular"
	}

	return &ParsedFont{
		Name:   fileName,
		Family: family,
		Style:  style,
		Type:   fileExt,
		Scope:  string(scope),
		Path:   fontPath,
	}
}

// parsedFontMergeAdapter bridges ParsedFont to installations.RegistryMergeMutableRow without copying merge fields.
type parsedFontMergeAdapter struct {
	p *ParsedFont
}

func (a *parsedFontMergeAdapter) BlankFontID() bool {
	return strings.TrimSpace(a.p.FontID) == ""
}
func (a *parsedFontMergeAdapter) PathForMerge() string {
	return strings.TrimSpace(a.p.Path)
}
func (a *parsedFontMergeAdapter) FamilyForMerge() string {
	return strings.TrimSpace(a.p.Family)
}
func (a *parsedFontMergeAdapter) ApplyRepoCatalog(fontID, license string, categories []string, source string) {
	a.p.FontID = fontID
	a.p.License = license
	a.p.Categories = append([]string(nil), categories...)
	a.p.Source = source
}

// mergeInstallationRegistryIntoFamilies fills Font ID / source / license / categories from the
// installation registry when repository matching left them empty (e.g. Nerd patched family names).
func mergeInstallationRegistryIntoFamilies(families map[string][]ParsedFont, reg *installations.Registry) {
	if reg == nil {
		return
	}
	wrapped := make(map[string][]installations.RegistryMergeMutableRow, len(families))
	for k, g := range families {
		slots := make([]installations.RegistryMergeMutableRow, len(g))
		for i := range g {
			slots[i] = &parsedFontMergeAdapter{p: &families[k][i]}
		}
		wrapped[k] = slots
	}
	installations.MergeInstallationRegistryIntoFamilyGroups(wrapped, reg, func(instFontID string) (string, string, []string, string, bool) {
		match, err := repo.MatchRepositoryFontByID(instFontID)
		if err != nil || match == nil {
			return "", "", nil, "", false
		}
		return match.FontID, match.License, append([]string(nil), match.Categories...), match.Source, true
	})
}

// groupByFamily groups fonts by family name
func groupByFamily(fonts []ParsedFont) map[string][]ParsedFont {
	res := make(map[string][]ParsedFont)
	for _, f := range fonts {
		res[f.Family] = append(res[f.Family], f)
	}
	return res
}

// familiesMatchingNameQuery returns families whose SFNT family name contains query
// (case-insensitive substring, same as list <query> name matching). If nothing matches
// by family name, it returns the original map and false so callers can fall back to a
// full catalog join and Font ID filtering.
func familiesMatchingNameQuery(families map[string][]ParsedFont, query string) (map[string][]ParsedFont, bool) {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return families, false
	}
	out := make(map[string][]ParsedFont)
	for name, group := range families {
		if strings.Contains(strings.ToLower(name), q) {
			out[name] = group
		}
	}
	if len(out) == 0 {
		return families, false
	}
	return out, true
}

// filterFontsByFamilyAndID filters font families by family name or Font ID
func filterFontsByFamilyAndID(families map[string][]ParsedFont, familyFilter string) map[string][]ParsedFont {
	// Optimization: Cache lowercased strings to avoid repeated ToLower() calls
	filteredFamilies := make(map[string][]ParsedFont)
	familyFilterLower := ""
	if familyFilter != "" {
		familyFilterLower = strings.ToLower(familyFilter)
	}

	for familyName, fontGroup := range families {
		// Get Font ID for this family (from first font in group, all have same Font ID)
		fontID := ""
		if len(fontGroup) > 0 {
			fontID = fontGroup[0].FontID
		}

		// Check if family name or Font ID matches the filter
		matchesFilter := true
		if familyFilterLower != "" {
			// Cache lowercased strings (optimization)
			familyLower := strings.ToLower(familyName)
			fontIDLower := ""
			if fontID != "" {
				fontIDLower = strings.ToLower(fontID)
			}

			// Check if filter matches family name OR Font ID
			matchesFilter = strings.Contains(familyLower, familyFilterLower) ||
				(fontIDLower != "" && strings.Contains(fontIDLower, familyFilterLower))
		}

		if !matchesFilter {
			continue
		}

		filteredFamilies[familyName] = fontGroup
	}

	return filteredFamilies
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringP("scope", "s", "", "Filter by installation scope (user or machine). Default: show all scopes")
	listCmd.Flags().StringP("type", "t", "", "Filter by font type (TTF, OTF, etc.)")
	listCmd.Flags().BoolP("expand", "x", false, "Show font styles in hierarchical view")
	listCmd.Flags().Bool("fontget-installed", false, "Show only fonts installed by FontGet")
}
