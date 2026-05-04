package installations

import "strings"

// RegistryMergeMutableRow is one list row that can receive catalog enrichment from the installation registry.
type RegistryMergeMutableRow interface {
	BlankFontID() bool
	PathForMerge() string
	FamilyForMerge() string
	ApplyRepoCatalog(fontID, license string, categories []string, source string)
}

// RegistryCatalogLookup resolves manifest metadata for a Font ID (e.g. repo.MatchRepositoryFontByID).
type RegistryCatalogLookup func(instFontID string) (canonicalID string, license string, categories []string, source string, ok bool)

// MergeInstallationRegistryIntoFamilyGroups fills Font ID / license / categories / source when rows have blank Font ID,
// using PathIndex first, then FamilyInstallationsIndex only when exactly one installation owns that SFNT family.
func MergeInstallationRegistryIntoFamilyGroups(groups map[string][]RegistryMergeMutableRow, reg *Registry, lookup RegistryCatalogLookup) {
	if reg == nil || lookup == nil || len(groups) == 0 {
		return
	}
	pathIdx := reg.PathIndex()
	famIdx := reg.FamilyInstallationsIndex()

	for familyName := range groups {
		group := groups[familyName]
		for i := range group {
			row := group[i]
			if row == nil || !row.BlankFontID() {
				continue
			}
			var inst *Installation
			if p := row.PathForMerge(); p != "" {
				if hit := pathIdx[NormalizePathKey(p)]; hit != nil {
					inst = hit
				}
			}
			if inst == nil {
				if fam := row.FamilyForMerge(); fam != "" {
					cands := famIdx[strings.ToLower(fam)]
					if len(cands) == 1 {
						inst = cands[0]
					}
				}
			}
			if inst == nil {
				continue
			}
			matchID, lic, cats, src, ok := lookup(strings.TrimSpace(inst.FontID))
			if !ok {
				continue
			}
			row.ApplyRepoCatalog(matchID, lic, cats, src)
		}
		groups[familyName] = group
	}
}
