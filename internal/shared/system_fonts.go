package shared

import (
	"runtime"
	"strings"
)

// windowsSystemFonts is a list of critical Windows system fonts
// (filenames and families, case-insensitive, no extension)
var windowsSystemFonts = map[string]bool{
	// Core Windows UI (absolute)
	"segoeui":            true,
	"segoeuibold":        true,
	"segoeuiitalic":      true,
	"segoeuibolditalic":  true,
	"segoeuivariable":    true, // Segoe UI Variable
	"microsoftsansserif": true, // Microsoft Sans Serif
	"tahoma":             true,
	"mssansserif":        true, // MS Sans Serif

	// Icon, glyph & UI glue (critical)
	"marlett":          true,
	"segoefluenticons": true, // Segoe Fluent Icons
	"segoemdl2assets":  true, // Segoe MDL2 Assets
	"segoeuisymbol":    true, // Segoe UI Symbol
	"wingdings":        true,
	"wingdings2":       true,
	"wingdings3":       true,
	"webdings":         true,
	"symbol":           true,

	// Text & legacy compatibility (system-referenced)
	"arial":                 true,
	"arialbold":             true,
	"arialitalic":           true,
	"arialbolditalic":       true,
	"arialblack":            true,
	"times":                 true,
	"timesnewroman":         true, // Shared with macOS
	"timesnewromanpsmt":     true,
	"courier":               true, // Shared with macOS
	"couriernew":            true, // Shared with macOS
	"verdana":               true,
	"trebuchetms":           true,
	"trebuchetmsbold":       true,
	"trebuchetmsitalic":     true,
	"trebuchetmsbolditalic": true,
	"georgia":               true,
	"georgiabold":           true,
	"georgiaitalic":         true,
	"georgiabolditalic":     true,

	// Modern Office-era fonts (still OS-linked)
	"calibri":           true,
	"calibribold":       true,
	"calibriitalic":     true,
	"calibribolditalic": true,
	"cambria":           true,
	"candara":           true,
	"consolas":          true,
	"constantia":        true,
	"corbel":            true,

	// Console & developer critical
	"lucidaconsole": true, // Lucida Console

	// Emoji
	"segoeuiemoji": true, // Segoe UI Emoji

	// CJK & regional system fonts
	"meiryo":       true,
	"yugothic":     true, // Yu Gothic
	"msgothic":     true, // MS Gothic (Japanese)
	"msmincho":     true, // MS Mincho
	"simsun":       true, // SimSun (Simplified Chinese)
	"simhei":       true, // SimHei
	"mingliub":     true, // MingLiU (Traditional Chinese)
	"pmingliu":     true, // PMingLiU
	"malgungothic": true, // Malgun Gothic
	"gulim":        true,
	"batang":       true,
	"msjh":         true, // Microsoft JhengHei
	"msjhbd":       true, // Microsoft JhengHei Bold
	"msjhl":        true, // Microsoft JhengHei Light
	"msyh":         true, // Microsoft YaHei
	"msyhbd":       true, // Microsoft YaHei Bold
	"msyhl":        true, // Microsoft YaHei Light

	// Math & special
	"cambriamath": true, // Cambria Math

	// Core fallbacks
	"framd":      true, // Franklin Gothic Medium (often referenced)
	"msgothicui": true, // UI alias used by older Japanese systems
	"msuigothic": true, // MS UI Gothic

	// Very system-linked fonts (filename variants)
	"seguisb": true, // Segoe UI Semibold
	"seguili": true, // Segoe UI Light
	"seguisl": true, // Segoe UI Light (variant)

	// Math / symbol safety nets
	"arialunicode": true, // Arial Unicode MS (still referenced by apps)

	// Legacy fonts (still present in some systems)
	"comicsansms":        true,
	"comicsansmsbold":    true,
	"impact":             true,
	"palatino":           true,
	"palatinolinotype":   true,
	"bookantiqua":        true,
	"centurygothic":      true,
	"franklingothic":     true,
	"gillsans":           true,
	"gillsansmt":         true,
	"garamond":           true, // Shared with macOS
	"garamonditalic":     true,
	"garamondbold":       true,
	"garamondbolditalic": true,
}

// macOSSystemFonts is a list of critical macOS system fonts
// (filenames and families, case-insensitive, no extension)
var macOSSystemFonts = map[string]bool{
	// Core system UI (absolute)
	"sfpro":        true, // SF Pro (family umbrella)
	"sfprodisplay": true, // SF Pro Display
	"sfprotext":    true, // SF Pro Text
	"sfprorounded": true, // SF Pro Rounded
	"sfcompact":    true, // SF Compact
	"sfmono":       true, // SF Mono
	"sanfrancisco": true, // San Francisco (family umbrella)
	"sfnsdisplay":  true,
	"sfnsrounded":  true,
	"sfnstext":     true,
	"systemfont":   true, // generic alias used internally
	"sfarabic":     true,
	"sfarmenian":   true,
	"sfhebrew":     true,
	"sfsymbols":    true, // SF Symbols (not the app)

	// UI fallbacks & legacy
	"helvetica":     true, // Shared with Windows (sometimes)
	"helveticaneue": true,
	"lucidagrande":  true, // "Lucida Grande" normalized
	"geneva":        true,
	"monaco":        true,
	"menlo":         true,
	"chicago":       true,

	// Apple system classics
	"arial":              true, // Shared with Windows
	"arialblack":         true,
	"times":              true, // Shared with Windows
	"timesnewroman":      true, // Shared with Windows
	"courier":            true, // Shared with Windows
	"couriernew":         true, // Shared with Windows
	"palatino":           true, // Shared with Windows
	"baskerville":        true,
	"optima":             true,
	"optimabold":         true,
	"optimaitalic":       true,
	"optimabolditalic":   true,
	"didot":              true,
	"americantypewriter": true, // American Typewriter
	"hoeflertext":        true, // Hoefler Text

	// Emoji & symbol
	"applecoloremoji": true, // Apple Color Emoji
	"applesymbols":    true, // Apple Symbols

	// CJK & international (system-critical per locale)
	"hiraginosans":     true, // Hiragino Sans
	"hiraginomincho":   true, // Hiragino Mincho
	"pingfangsc":       true, // PingFang SC
	"pingfangtc":       true, // PingFang TC
	"heitisc":          true, // Heiti SC
	"heititc":          true, // Heiti TC
	"songtisc":         true, // Songti SC
	"songtitc":         true, // Songti TC
	"applesdgothicneo": true, // Apple SD Gothic Neo
	"osaka":            true,

	// Math & technical
	"stixgeneral":      true, // STIXGeneral
	"stixsizeonesym":   true, // STIXSizeOneSym
	"stixsizetwosym":   true, // STIXSizeTwoSym
	"stixsizethreesym": true, // STIXSizeThreeSym
	"stixsizefoursym":  true, // STIXSizeFourSym

	// Legacy-but-used fonts
	"applebraille": true,
	"lastresort":   true, // Critical fallback font

	// Legacy fonts (still present in some systems)
	"cambria":              true,
	"bookantiqua":          true, // Shared with Windows
	"centurygothic":        true, // Shared with Windows
	"trebuchetms":          true, // Shared with Windows
	"verdana":              true, // Shared with Windows
	"georgia":              true, // Shared with Windows
	"comicsansms":          true, // Shared with Windows
	"impact":               true, // Shared with Windows
	"tahoma":               true, // Shared with Windows
	"myriad":               true,
	"myriadpro":            true,
	"myriadset":            true,
	"myriadsemibold":       true,
	"myriadsemibolditalic": true,
	"athelas":              true,
	"seravek":              true,
	"seraveklight":         true,
	"seravekmedium":        true,
	"seraveksemibold":      true,
	"seravekbold":          true,
	"applegaramond":        true,
	"garamond":             true, // Shared with Windows
	"garamonditalic":       true,
	"garamondbold":         true,
	"garamondbolditalic":   true,
	"futura":               true,
	"futurabold":           true,
	"futuraitalic":         true,
	"futurabolditalic":     true,
}

// linuxSystemFonts is a list of critical Linux system fonts
// (filenames and families, case-insensitive, no extension)
var linuxSystemFonts = map[string]bool{
	"ubuntu":              true,
	"ubuntumono":          true,
	"ubuntubold":          true,
	"ubuntuitalic":        true,
	"ubuntubolditalic":    true,
	"dejavusans":          true, // DejaVu Sans
	"dejavusansmono":      true, // DejaVu Sans Mono
	"dejavuserif":         true, // DejaVu Serif
	"cantarell":           true,
	"cantarellbold":       true,
	"cantarellitalic":     true,
	"cantarellbolditalic": true,
	"symbola":             true,

	// Liberation fonts (common fallbacks)
	"liberationsans":  true,
	"liberationserif": true,
	"liberationmono":  true,

	// Noto core (modern distros rely on this)
	"notosans":       true,
	"notoserif":      true,
	"notosansmono":   true,
	"notocoloremoji": true,

	// Terminal & fallback safety
	"terminus": true,
	"hack":     true,
	"firacode": true,

	// KDE Plasma safety
	"firafonts":     true,
	"firamonospace": true,
}

// normalizeFontNameForCheck normalizes a font name for comparison
// (lowercase, removes spaces/hyphens/underscores)
func normalizeFontNameForCheck(fontName string) string {
	name := strings.ToLower(fontName)
	name = strings.ReplaceAll(name, " ", "")
	name = strings.ReplaceAll(name, "-", "")
	name = strings.ReplaceAll(name, "_", "")
	return name
}

// IsCriticalSystemFont checks if a font is a critical system font for any platform.
// This function normalizes the font name (lowercase, removes spaces/hyphens/underscores) before checking.
// Note: We don't remove file extension here since we're checking family names, not filenames.
// This is used for repository matching to avoid matching system fonts.
func IsCriticalSystemFont(fontName string) bool {
	name := normalizeFontNameForCheck(fontName)
	return windowsSystemFonts[name] || macOSSystemFonts[name] || linuxSystemFonts[name]
}

// IsPlatformSystemFont checks if a font is a critical system font for the current platform.
// This is used for removal protection - only block fonts that are system fonts on the current OS.
func IsPlatformSystemFont(fontName string) bool {
	name := normalizeFontNameForCheck(fontName)
	switch runtime.GOOS {
	case "windows":
		return windowsSystemFonts[name]
	case "darwin":
		return macOSSystemFonts[name]
	case "linux":
		return linuxSystemFonts[name]
	default:
		return false
	}
}
