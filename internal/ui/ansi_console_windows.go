//go:build windows

package ui

import (
	"os"

	"github.com/charmbracelet/x/term"
	"golang.org/x/sys/windows"
)

// EnableANSIConsole turns on Windows virtual terminal processing for stdout/stderr
// when they are consoles. Without this, lipgloss ANSI (used by list/search tables)
// prints as literal escape text on hosts that do not enable VT by default
// (classic conhost, Windows Sandbox). Bubble Tea enables VT only while it runs and
// restores the prior mode on exit, so a spinner-then-print path would otherwise
// drop VT before the colored table is written.
//
// Intentionally not restored on process exit: leaving VT on for the rest of the
// console session matches Windows Terminal defaults and keeps post-tea CLI output styled.
func EnableANSIConsole() {
	enableVirtualTerminalProcessing(os.Stdout)
	enableVirtualTerminalProcessing(os.Stderr)
}

func enableVirtualTerminalProcessing(f *os.File) {
	if f == nil || !term.IsTerminal(f.Fd()) {
		return
	}
	handle := windows.Handle(f.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return
	}
	if mode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING != 0 {
		return
	}
	_ = windows.SetConsoleMode(handle, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
}
