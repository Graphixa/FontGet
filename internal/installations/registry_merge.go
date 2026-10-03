package installations

import "strings"

// ResolveInstallationForCatalogMerge finds a registry installation for path (preferred)
// or a unique SFNT family match. Returns nil when ambiguous or not found.
func ResolveInstallationForCatalogMerge(reg *Registry, path, family string) *Installation {
	if reg == nil {
		return nil
	}
	if p := strings.TrimSpace(path); p != "" {
		if hit := reg.PathIndex()[NormalizePathKey(p)]; hit != nil {
			return hit
		}
	}
	if fam := strings.TrimSpace(family); fam != "" {
		cands := reg.FamilyInstallationsIndex()[strings.ToLower(fam)]
		if len(cands) == 1 {
			return cands[0]
		}
	}
	return nil
}
