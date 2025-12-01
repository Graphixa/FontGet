package platform

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// TerminalTheme represents the detected terminal theme
type TerminalTheme int

const (
	TerminalThemeUnknown TerminalTheme = iota
	TerminalThemeDark
	TerminalThemeLight
)

// TerminalRGB represents a color in normalized RGB space (0.0 to 1.0)
type TerminalRGB struct {
	R, G, B float64
}

// TerminalKind represents the terminal kind detected
type TerminalKind int

const (
	TerminalKindUnknown TerminalKind = iota
	TerminalKindWin32
	TerminalKindXterm
	TerminalKindOther
)

// TerminalThemeResult contains the detection result
type TerminalThemeResult struct {
	Kind  TerminalKind
	RGB   TerminalRGB
	Theme TerminalTheme
}

// DetectTerminalTheme attempts to determine the terminal background colour using the
// best available mechanism for the platform (Win32 API, OSC 11, etc.).
func DetectTerminalTheme(timeout time.Duration) (TerminalThemeResult, error) {
	kind := detectTerminalKind()

	switch kind {
	case TerminalKindXterm:
		// Try OSC 11 first (works on Windows Terminal and xterm-compatible terminals)
		// On Windows, use Windows-specific implementation for better reliability
		var rgb TerminalRGB
		var err error
		if runtime.GOOS == "windows" && isWindowsTerminal() {
			rgb, err = detectOSC11Windows(timeout)
		} else {
			rgb, err = detectOSC11(timeout)
		}
		if err == nil {
			return buildTerminalResult(TerminalKindXterm, rgb), nil
		}
		// If OSC 11 fails, fall through to COLORFGBG
		// Note: We do NOT use Win32 API for Windows Terminal as it returns incorrect values (0,0,0)
		// For legacy Windows console, Win32 API is tried in the TerminalKindWin32 case below
		if runtime.GOOS == "windows" && !isWindowsTerminal() {
			// Legacy Windows console - Win32 API should work
			if rgb, err := detectWin32TerminalRGB(timeout); err == nil {
				return buildTerminalResult(TerminalKindWin32, rgb), nil
			}
		}

	case TerminalKindWin32:
		// Legacy Windows console - use Win32 API
		if rgb, err := detectWin32TerminalRGB(timeout); err == nil {
			return buildTerminalResult(TerminalKindWin32, rgb), nil
		}
	}

	// Fallback: COLORFGBG env var
	if rgb, err := detectFromColorFGBG(); err == nil {
		return buildTerminalResult(TerminalKindOther, rgb), nil
	}

	return TerminalThemeResult{
		Kind:  TerminalKindUnknown,
		RGB:   TerminalRGB{},
		Theme: TerminalThemeUnknown,
	}, fmt.Errorf("theme detection unavailable")
}

// TerminalThemeFromEnvOrDetect allows env-based override for the theme,
// e.g. FONTGET_THEME_MODE=dark|light
func TerminalThemeFromEnvOrDetect(envVar string, timeout time.Duration) (TerminalThemeResult, error) {
	override := strings.ToLower(os.Getenv(envVar))
	switch override {
	case "dark":
		return TerminalThemeResult{Kind: TerminalKindOther, Theme: TerminalThemeDark}, nil
	case "light":
		return TerminalThemeResult{Kind: TerminalKindOther, Theme: TerminalThemeLight}, nil
	case "auto":
		// "auto" means use detection
		return DetectTerminalTheme(timeout)
	}
	// Empty or invalid value - use detection
	return DetectTerminalTheme(timeout)
}

// detectTerminalKind determines the terminal "kind" with heuristics
func detectTerminalKind() TerminalKind {
	if runtime.GOOS == "windows" {
		// Windows Terminal supports OSC 11, so treat it as TerminalKindXterm
		if isWindowsTerminal() {
			return TerminalKindXterm // Use OSC 11 detection
		}
		return TerminalKindWin32 // Legacy console, use Win32 API
	}

	// Unix-like systems
	term := os.Getenv("TERM")
	if strings.Contains(term, "xterm") ||
		strings.Contains(term, "screen") ||
		strings.Contains(term, "tmux") ||
		strings.Contains(term, "gnome") {
		return TerminalKindXterm
	}
	return TerminalKindOther
}

// isWindowsTerminal detects if we're running in Windows Terminal
func isWindowsTerminal() bool {
	// Windows Terminal sets WT_SESSION environment variable
	if os.Getenv("WT_SESSION") != "" {
		return true
	}
	// Windows Terminal also sets TERM_PROGRAM to "WindowsTerminal"
	if os.Getenv("TERM_PROGRAM") == "WindowsTerminal" {
		return true
	}
	// VS Code terminal also supports OSC 11
	if os.Getenv("TERM_PROGRAM") == "vscode" {
		return true
	}
	// Additional heuristics can be added here
	return false
}

// classifyTerminalTheme classifies an RGB color as dark or light using luminance
func classifyTerminalTheme(c TerminalRGB) TerminalTheme {
	// Luminance formula: Y = 0.299*R + 0.587*G + 0.114*B
	y := 0.299*c.R + 0.587*c.G + 0.114*c.B
	if y > 0.5 {
		return TerminalThemeLight
	}
	return TerminalThemeDark
}

// buildTerminalResult creates a TerminalThemeResult from RGB and kind
func buildTerminalResult(kind TerminalKind, rgb TerminalRGB) TerminalThemeResult {
	return TerminalThemeResult{
		Kind:  kind,
		RGB:   rgb,
		Theme: classifyTerminalTheme(rgb),
	}
}

// detectFromColorFGBG parses COLORFGBG env var as fallback
func detectFromColorFGBG() (TerminalRGB, error) {
	v := os.Getenv("COLORFGBG")
	if v == "" {
		return TerminalRGB{}, fmt.Errorf("COLORFGBG not set")
	}

	parts := strings.Split(v, ";")
	if len(parts) == 0 {
		return TerminalRGB{}, fmt.Errorf("invalid COLORFGBG format")
	}

	// Background is the last part
	bgStr := parts[len(parts)-1]
	idx, err := strconv.Atoi(bgStr)
	if err != nil {
		return TerminalRGB{}, fmt.Errorf("invalid background index in COLORFGBG: %w", err)
	}

	return ansi256ToTerminalRGB(idx), nil
}

// ansi256ToTerminalRGB converts an ANSI 256-color index to RGB
func ansi256ToTerminalRGB(idx int) TerminalRGB {
	if idx < 0 || idx > 255 {
		return TerminalRGB{R: 0.5, G: 0.5, B: 0.5} // Default to grey
	}

	// 0-15: 16 basic ANSI colors
	if idx < 16 {
		return ansi16ToTerminalRGB(idx)
	}

	// 16-231: 6×6×6 color cube
	if idx < 232 {
		idx -= 16
		r := idx / 36
		g := (idx / 6) % 6
		b := idx % 6
		return TerminalRGB{
			R: float64(r) / 5.0,
			G: float64(g) / 5.0,
			B: float64(b) / 5.0,
		}
	}

	// 232-255: Greyscale ramp
	grey := float64(idx-232) / 23.0
	return TerminalRGB{R: grey, G: grey, B: grey}
}

// ansi16ToTerminalRGB converts ANSI 16-color index to RGB
// Maps standard ANSI colors (0-15) to RGB values
func ansi16ToTerminalRGB(idx int) TerminalRGB {
	// Standard ANSI 16-color palette
	// 0-7: standard colors, 8-15: bright colors
	colors := []TerminalRGB{
		// 0-7: Standard colors
		{0.0, 0.0, 0.0}, // 0: Black
		{0.8, 0.0, 0.0}, // 1: Red
		{0.0, 0.8, 0.0}, // 2: Green
		{0.8, 0.8, 0.0}, // 3: Yellow
		{0.0, 0.0, 0.8}, // 4: Blue
		{0.8, 0.0, 0.8}, // 5: Magenta
		{0.0, 0.8, 0.8}, // 6: Cyan
		{0.8, 0.8, 0.8}, // 7: White (light grey)
		// 8-15: Bright colors
		{0.4, 0.4, 0.4}, // 8: Bright Black (dark grey)
		{1.0, 0.4, 0.4}, // 9: Bright Red
		{0.4, 1.0, 0.4}, // 10: Bright Green
		{1.0, 1.0, 0.4}, // 11: Bright Yellow
		{0.4, 0.4, 1.0}, // 12: Bright Blue
		{1.0, 0.4, 1.0}, // 13: Bright Magenta
		{0.4, 1.0, 1.0}, // 14: Bright Cyan
		{1.0, 1.0, 1.0}, // 15: Bright White
	}

	if idx >= 0 && idx < len(colors) {
		return colors[idx]
	}
	return TerminalRGB{R: 0.5, G: 0.5, B: 0.5} // Default to grey
}
