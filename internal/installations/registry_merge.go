package installations

import "strings"

// ResolveInstallationForCatalogMerge finds a registry installation for path (preferred)
// or a unique SFNT family match. Returns nil when ambiguous or not found.
// Prefer ResolveInstallationForCatalogMergeWithIndexes when resolving many rows.
func ResolveInstallationForCatalogMerge(reg *Registry, path, family string) *Installation {
	if reg == nil {
		return nil
	}
	return ResolveInstallationForCatalogMergeWithIndexes(reg.PathIndex(), reg.FamilyInstallationsIndex(), path, family)
}

// ResolveInstallationForCatalogMergeWithIndexes applies path-first then unambiguous-family rules
// using prebuilt indexes.
func ResolveInstallationForCatalogMergeWithIndexes(
	byPath map[string]*Installation,
	byFamily map[string][]*Installation,
	path, family string,
) *Installation {
	if p := strings.TrimSpace(path); p != "" && byPath != nil {
		if hit := byPath[NormalizePathKey(p)]; hit != nil {
			return hit
		}
	}
	if fam := strings.TrimSpace(family); fam != "" && byFamily != nil {
		cands := byFamily[strings.ToLower(fam)]
		if len(cands) == 1 {
			return cands[0]
		}
	}
	return nil
}
