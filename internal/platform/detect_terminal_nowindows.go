//go:build !windows

package platform

import (
	"fmt"
	"time"
)

// detectWin32TerminalRGB is a stub on non-Windows platforms.
// It should never be called because detectTerminalKind() will not
// return TerminalKindWin32 on non-Windows OSes. It exists only to
// satisfy the compiler and linters for cross-platform builds.
func detectWin32TerminalRGB(timeout time.Duration) (TerminalRGB, error) {
	_ = timeout // parameter is unused on non-Windows builds
	return TerminalRGB{}, fmt.Errorf("Win32 terminal detection is only available on Windows")
}
