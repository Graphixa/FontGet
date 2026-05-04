package installations

import (
	"strings"

	"fontget/internal/platform"
)

// ManifestFontIDProbe reports whether the argument is an exact Font ID in the manifest (caller typically passes repo.IsFontIDInManifest with one loaded manifest, or repo.IsFontIDInCachedManifest).
type ManifestFontIDProbe func(fontID string) bool

// ShouldConsultRegistryForRemoval is true when removal should consider installation_registry.json:
// dotted IDs, an install record for an ID without dots, or an exact manifest Font ID.
func ShouldConsultRegistryForRemoval(fontName string, reg *Registry, manifestHasFontID ManifestFontIDProbe) bool {
	fn := strings.TrimSpace(fontName)
	if fn == "" {
		return false
	}
	if strings.Contains(fn, ".") {
		return true
	}
	if reg != nil && reg.FindByFontID(fn) != nil {
		return true
	}
	return manifestHasFontID != nil && manifestHasFontID(fn)
}

// InstallationHasBasenamesUnderDir reports whether reg lists at least one face path under fontDir for fontID.
func InstallationHasBasenamesUnderDir(reg *Registry, fontID, fontDir string) bool {
	if reg == nil {
		return false
	}
	inst := reg.FindByFontID(fontID)
	if inst == nil {
		return false
	}
	return len(inst.BasenamesForDir(fontDir)) > 0
}

// InstallationRegistryResolvable reports whether reg lists files under any requested scope directory and returns a display label (catalog name or fontName).
func InstallationRegistryResolvable(fontName string, reg *Registry, scopes []platform.InstallationScope, fontDir func(platform.InstallationScope) string, manifestHasFontID ManifestFontIDProbe) (ok bool, catalogDisplay string) {
	if !ShouldConsultRegistryForRemoval(fontName, reg, manifestHasFontID) || reg == nil {
		return false, ""
	}
	inst := reg.FindByFontID(fontName)
	if inst == nil || !inst.HasFaces() {
		return false, ""
	}
	display := strings.TrimSpace(inst.CatalogName)
	if display == "" {
		display = fontName
	}
	for _, sc := range scopes {
		dir := fontDir(sc)
		if InstallationHasBasenamesUnderDir(reg, fontName, dir) {
			return true, display
		}
	}
	return false, ""
}
