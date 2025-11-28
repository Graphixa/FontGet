package shared

import (
	"strings"
)

// criticalSystemFonts is a list of critical system fonts to not remove or match
// (filenames and families, case-insensitive, no extension)
var criticalSystemFonts = map[string]bool{
	// Windows core fonts
	"arial":                 true,
	"arialbold":             true,
	"arialitalic":           true,
	"arialbolditalic":       true,
	"calibri":               true,
	"calibribold":           true,
	"calibriitalic":         true,
	"calibribolditalic":     true,
	"segoeui":               true,
	"segoeuibold":           true,
	"segoeuiitalic":         true,
	"segoeuibolditalic":     true,
	"times":                 true,
	"timesnewroman":         true,
	"timesnewromanpsmt":     true,
	"courier":               true,
	"tahoma":                true,
	"verdana":               true,
	"symbol":                true,
	"wingdings":             true,
	"consolas":              true,
	"georgia":               true,
	"georgiabold":           true,
	"georgiaitalic":         true,
	"georgiabolditalic":     true,
	"comicsansms":           true,
	"comicsansmsbold":       true,
	"impact":                true,
	"trebuchetms":           true,
	"trebuchetmsbold":       true,
	"trebuchetmsitalic":     true,
	"trebuchetmsbolditalic": true,
	"palatino":              true,
	"palatinolinotype":      true,
	"bookantiqua":           true,
	"centurygothic":         true,
	"franklingothic":        true,
	"gillsans":              true,
	"gillsansmt":            true,

	// macOS core fonts
	"cambria":              true,
	"sfnsdisplay":          true,
	"sfnsrounded":          true,
	"sfnstext":             true,
	"geneva":               true,
	"monaco":               true,
	"lucida grande":        true,
	"menlo":                true,
	"helvetica":            true,
	"helveticaneue":        true,
	"myriad":               true,
	"myriadpro":            true,
	"myriadset":            true,
	"myriadsemibold":       true,
	"myriadsemibolditalic": true,
	"sanfrancisco":         true,
	"sfprodisplay":         true,
	"sfprotext":            true,
	"sfprorounded":         true,
	"athelas":              true,
	"seravek":              true,
	"seraveklight":         true,
	"seravekmedium":        true,
	"seraveksemibold":      true,
	"seravekbold":          true,
	"applegaramond":        true,
	"garamond":             true,
	"garamonditalic":       true,
	"garamondbold":         true,
	"garamondbolditalic":   true,
	"optima":               true,
	"optimabold":           true,
	"optimaitalic":         true,
	"optimabolditalic":     true,
	"futura":               true,
	"futurabold":           true,
	"futuraitalic":         true,
	"futurabolditalic":     true,

	// Linux system fonts
	"ubuntu":              true,
	"ubuntumono":          true,
	"ubuntubold":          true,
	"ubuntuitalic":        true,
	"ubuntubolditalic":    true,
	"cantarell":           true,
	"cantarellbold":       true,
	"cantarellitalic":     true,
	"cantarellbolditalic": true,
}

// IsCriticalSystemFont checks if a font is a critical system font.
// This function normalizes the font name (lowercase, removes spaces/hyphens/underscores) before checking.
// Note: We don't remove file extension here since we're checking family names, not filenames.
func IsCriticalSystemFont(fontName string) bool {
	name := strings.ToLower(fontName)
	name = strings.ReplaceAll(name, " ", "")
	name = strings.ReplaceAll(name, "-", "")
	name = strings.ReplaceAll(name, "_", "")
	return criticalSystemFonts[name]
}
