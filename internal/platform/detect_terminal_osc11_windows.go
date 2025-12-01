//go:build windows
// +build windows

package platform

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"

	"fontget/internal/logging"

	"golang.org/x/sys/windows"
)

// debugLog writes debug output directly to stderr for immediate visibility
// This bypasses the logging system to ensure we see what's happening
func debugLog(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[OSC11-DEBUG] "+format+"\n", args...)
	// Also try to log via logger if available
	if logger := logging.GetLogger(); logger != nil {
		logger.Debug("[OSC11-Windows] "+format, args...)
	}
}

var (
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	getStdHandle        = kernel32.NewProc("GetStdHandle")
	getConsoleMode      = kernel32.NewProc("GetConsoleMode")
	setConsoleMode      = kernel32.NewProc("SetConsoleMode")
	readConsoleInput    = kernel32.NewProc("ReadConsoleInputW")
	peekConsoleInput    = kernel32.NewProc("PeekConsoleInputW")
	waitForSingleObject = kernel32.NewProc("WaitForSingleObject")
	readFile            = kernel32.NewProc("ReadFile")
	readConsoleA        = kernel32.NewProc("ReadConsoleA")
)

const (
	STD_INPUT_HANDLE              = uintptr(0xFFFFFFF6)
	ENABLE_VIRTUAL_TERMINAL_INPUT = 0x0200
	ENABLE_PROCESSED_INPUT        = 0x0001
	ENABLE_LINE_INPUT             = 0x0002
	ENABLE_ECHO_INPUT             = 0x0004
	ENABLE_INSERT_MODE            = 0x0020
	WAIT_OBJECT_0                 = 0x00000000
	WAIT_TIMEOUT                  = 0x00000102
	INFINITE                      = 0xFFFFFFFF
	KEY_EVENT                     = 0x0001
)

// INPUT_RECORD structure for Windows Console API
type inputRecord struct {
	EventType uint16
	_         [2]byte  // padding
	Event     [20]byte // KEY_EVENT_RECORD is larger than 16 bytes
}

// KEY_EVENT_RECORD structure
type keyEventRecord struct {
	KeyDown         int32
	RepeatCount     uint16
	VirtualKeyCode  uint16
	VirtualScanCode uint16
	UnicodeChar     uint16
	ControlKeyState uint32
}

// detectOSC11Windows is a Windows-specific implementation that uses console APIs
// to read OSC 11 responses more reliably by setting console mode before reading
func detectOSC11Windows(timeout time.Duration) (TerminalRGB, error) {
	debugLog("Starting OSC 11 detection with timeout: %v", timeout)

	// Get stdin handle
	stdinHandle := windows.Handle(os.Stdin.Fd())
	debugLog("Got stdin handle: %d", stdinHandle)

	// Get current console mode
	var mode uint32
	ret, _, _ := getConsoleMode.Call(uintptr(stdinHandle), uintptr(unsafe.Pointer(&mode)))
	if ret == 0 {
		errCode := syscall.GetLastError()
		debugLog("Failed to get console mode (error code: %d), falling back to standard method", errCode)
		// Fall back to standard method if we can't get console mode
		return detectOSC11(timeout)
	}
	debugLog("Current console mode: 0x%08x", mode)

	// Enable virtual terminal input processing
	// This helps with reading escape sequences on Windows Terminal
	newMode := mode | ENABLE_VIRTUAL_TERMINAL_INPUT
	// Disable line input to get raw bytes (needed for escape sequences)
	newMode &^= ENABLE_LINE_INPUT
	// Disable echo input - prevents OSC response from being echoed to terminal
	// This allows us to read it from stdin instead
	newMode &^= ENABLE_ECHO_INPUT
	// Disable processed input - we want raw escape sequences, not processed
	newMode &^= ENABLE_PROCESSED_INPUT

	debugLog("Setting new console mode: 0x%08x", newMode)
	debugLog("Mode flags: VIRTUAL_TERMINAL_INPUT=%v, LINE_INPUT=%v, ECHO_INPUT=%v, PROCESSED_INPUT=%v",
		(newMode&ENABLE_VIRTUAL_TERMINAL_INPUT) != 0,
		(newMode&ENABLE_LINE_INPUT) != 0,
		(newMode&ENABLE_ECHO_INPUT) != 0,
		(newMode&ENABLE_PROCESSED_INPUT) != 0)

	ret, _, _ = setConsoleMode.Call(uintptr(stdinHandle), uintptr(newMode))
	if ret == 0 {
		errCode := syscall.GetLastError()
		debugLog("Failed to set console mode (error code: %d), falling back to standard method", errCode)
		// If we can't set the mode, fall back to standard method
		return detectOSC11(timeout)
	}
	debugLog("Console mode set successfully")

	// Restore original mode when done
	defer func() {
		debugLog("Restoring original console mode: 0x%08x", mode)
		setConsoleMode.Call(uintptr(stdinHandle), uintptr(mode))
	}()

	// Send OSC 11 query
	query := "\x1b]11;?\x07"
	debugLog("Sending OSC 11 query: %q (hex: %s)", query, hex.EncodeToString([]byte(query)))
	fmt.Fprint(os.Stdout, query)
	os.Stdout.Sync()
	debugLog("Query sent and stdout synced")

	// Small delay to allow Windows Terminal to process and send the response
	delay := 20 * time.Millisecond
	debugLog("Waiting %v for terminal to process query...", delay)
	time.Sleep(delay)

	// Try multiple approaches to read the OSC 11 response
	var response []byte
	startTime := time.Now()
	debugLog("Starting to read response (timeout: %v)", timeout)

	// Approach 1: Try reading raw bytes using ReadConsoleA on the console handle
	// Use WaitForSingleObject to wait for input to be available
	debugLog("Approach 1: Using ReadConsoleA with WaitForSingleObject")
	iteration := 0
	for time.Since(startTime) < timeout {
		iteration++
		// Wait for input to be available (with timeout)
		remaining := timeout - time.Since(startTime)
		if remaining <= 0 {
			debugLog("Timeout reached, breaking from Approach 1")
			break
		}
		waitMs := uint32(remaining.Milliseconds())
		if waitMs > 100 {
			waitMs = 100 // Max 100ms wait per iteration
		}

		// Only log first few iterations and then every 10th to reduce noise
		if iteration <= 3 || iteration%10 == 0 {
			debugLog("Approach 1 iteration %d, waiting %dms for input (remaining: %v)", iteration, waitMs, remaining)
		}

		ret, _, _ := waitForSingleObject.Call(
			uintptr(stdinHandle),
			uintptr(waitMs),
		)

		if ret == WAIT_OBJECT_0 {
			debugLog("Input available, attempting ReadConsoleA...")
			// Input is available, try to read it using ReadConsoleA
			// ReadConsoleA is specifically designed for console input and handles
			// escape sequences better than ReadFile
			buf := make([]byte, 200)
			var bytesRead uint32

			readRet, _, err := readConsoleA.Call(
				uintptr(stdinHandle),
				uintptr(unsafe.Pointer(&buf[0])),
				uintptr(len(buf)),
				uintptr(unsafe.Pointer(&bytesRead)),
				0, // lpReserved
			)

			if readRet != 0 && bytesRead > 0 {
				readData := buf[:bytesRead]
				debugLog("ReadConsoleA succeeded: read %d bytes", bytesRead)
				debugLog("Read data (hex): %s", hex.EncodeToString(readData))
				debugLog("Read data (repr): %q", string(readData))
				response = append(response, readData...)

				// Check for terminator: BEL (0x07) or ST (ESC \)
				belIdx := bytes.IndexByte(response, 0x07)
				stIdx := bytes.Index(response, []byte{0x1b, 0x5c})
				termEnd := -1
				if belIdx >= 0 {
					termEnd = belIdx + 1
				}
				if stIdx >= 0 {
					if termEnd == -1 || stIdx+2 < termEnd {
						termEnd = stIdx + 2
					}
				}
				if termEnd > 0 {
					debugLog("Found terminator at index %d, complete response received!", termEnd-1)
					debugLog("Complete response (hex): %s", hex.EncodeToString(response[:termEnd]))
					debugLog("Complete response (repr): %q", string(response[:termEnd]))
					response = response[:termEnd]
					return parseOSC11RGB(response)
				}
				debugLog("No terminator yet, total response so far: %d bytes", len(response))
			} else {
				errCode := syscall.GetLastError()
				debugLog("ReadConsoleA failed: ret=%d, bytesRead=%d, error=%v, errCode=%d", readRet, bytesRead, err, errCode)
			}
		} else if ret == WAIT_TIMEOUT {
			// Timeout - continue to next iteration or try stdin fallback
			if iteration%10 == 0 || iteration <= 5 {
				debugLog("WaitForSingleObject timeout (iteration %d)", iteration)
			}
			continue
		} else {
			debugLog("WaitForSingleObject returned unexpected value: %d", ret)
		}

		// Safety check: limit response size
		if len(response) > 200 {
			debugLog("Response size limit reached (%d bytes), breaking", len(response))
			break
		}
	}
	debugLog("Approach 1 completed after %d iterations, total response: %d bytes", iteration, len(response))

	// Approach 2: If ReadConsoleA didn't produce a complete response, try standard stdin reading
	// This might work if the response got buffered
	debugLog("Approach 2: Trying standard stdin reading with bufio")
	reader := bufio.NewReader(os.Stdin)
	deadline := time.Now().Add(200 * time.Millisecond)
	if err := os.Stdin.SetReadDeadline(deadline); err == nil {
		stdinIteration := 0
		for time.Since(startTime) < timeout {
			stdinIteration++
			buffered := reader.Buffered()
			if buffered > 0 {
				debugLog("Approach 2: Found %d bytes buffered", buffered)
				buf := make([]byte, buffered)
				n, err := reader.Read(buf)
				if err == nil && n > 0 {
					readData := buf[:n]
					debugLog("Approach 2: Read %d bytes from buffer (hex): %s", n, hex.EncodeToString(readData))
					debugLog("Approach 2: Read data (repr): %q", string(readData))
					response = append(response, readData...)

					// Check for terminator: BEL or ST
					belIdx := bytes.IndexByte(response, 0x07)
					stIdx := bytes.Index(response, []byte{0x1b, 0x5c})
					termEnd := -1
					if belIdx >= 0 {
						termEnd = belIdx + 1
					}
					if stIdx >= 0 {
						if termEnd == -1 || stIdx+2 < termEnd {
							termEnd = stIdx + 2
						}
					}
					if termEnd > 0 {
						debugLog("Approach 2: Found terminator at index %d!", termEnd-1)
						response = response[:termEnd]
						return parseOSC11RGB(response)
					}
				} else if err != nil {
					debugLog("Approach 2: Error reading buffered data: %v", err)
				}
			}

			// Try reading with a short timeout
			quickDeadline := time.Now().Add(5 * time.Millisecond)
			os.Stdin.SetReadDeadline(quickDeadline)
			buf := make([]byte, 100)
			n, err := reader.Read(buf)
			if err == nil && n > 0 {
				readData := buf[:n]
				debugLog("Approach 2: Read %d bytes directly (hex): %s", n, hex.EncodeToString(readData))
				debugLog("Approach 2: Read data (repr): %q", string(readData))
				response = append(response, readData...)

				// Check for terminator: BEL or ST
				belIdx := bytes.IndexByte(response, 0x07)
				stIdx := bytes.Index(response, []byte{0x1b, 0x5c})
				termEnd := -1
				if belIdx >= 0 {
					termEnd = belIdx + 1
				}
				if stIdx >= 0 {
					if termEnd == -1 || stIdx+2 < termEnd {
						termEnd = stIdx + 2
					}
				}
				if termEnd > 0 {
					debugLog("Approach 2: Found terminator at index %d!", termEnd-1)
					response = response[:termEnd]
					return parseOSC11RGB(response)
				}
			} else if err != nil && (stdinIteration%20 == 0 || stdinIteration <= 5) {
				debugLog("Approach 2: Read attempt %d failed: %v", stdinIteration, err)
			}

			time.Sleep(2 * time.Millisecond)
			if len(response) > 200 {
				debugLog("Approach 2: Response size limit reached")
				break
			}
		}
		debugLog("Approach 2 completed after %d iterations", stdinIteration)
	} else {
		debugLog("Approach 2: Failed to set read deadline: %v", err)
	}

	// Final fallback to standard cross-platform method
	remaining := timeout - time.Since(startTime)
	debugLog("All approaches failed, falling back to standard detectOSC11 (remaining time: %v)", remaining)
	debugLog("Total response collected: %d bytes (hex): %s", len(response), hex.EncodeToString(response))
	return detectOSC11(remaining)
}
