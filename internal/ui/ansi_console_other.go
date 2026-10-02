//go:build !windows

package ui

// EnableANSIConsole is a no-op outside Windows. Unix terminals interpret ANSI without a console-mode flag.
func EnableANSIConsole() {}
