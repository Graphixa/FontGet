package platform

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"
)

// detectOSC11 detects background color using OSC 11 query
// Works on xterm-compatible terminals and Windows Terminal
// This is cross-platform and available on all platforms
// On Windows Terminal, consider using detectOSC11Windows for better reliability
func detectOSC11(timeout time.Duration) (TerminalRGB, error) {
	// Check if stdin is a terminal
	if !isTerminal(os.Stdin) {
		return TerminalRGB{}, fmt.Errorf("stdin is not a terminal")
	}

	// Write OSC 11 query: ESC ] 11 ; ? BEL
	// Note: On Windows, this might be sent by detectOSC11Windows, so check if we need to send it
	// For now, always send it (detectOSC11Windows will also send it, but that's okay)
	fmt.Fprint(os.Stdout, "\x1b]11;?\x07")
	os.Stdout.Sync()

	// Small delay to allow terminal to process and send the response
	// Some terminals need a moment to process the query
	time.Sleep(10 * time.Millisecond)

	// Start reading IMMEDIATELY - don't wait, the response might be buffered
	// Start goroutine to read stdin with a timeout
	responseChan := make(chan []byte, 1)
	errChan := make(chan error, 1)

	go func() {
		// Set read deadline to avoid blocking indefinitely
		deadline := time.Now().Add(timeout)
		if err := os.Stdin.SetReadDeadline(deadline); err != nil {
			// If we can't set deadline, continue anyway
		}

		// Read response from stdin
		// Terminal should respond with: ESC ] 11 ; rgb:RR/GG/BB BEL
		// or: ESC ] 11 ; rgb:RRRR/GGGG/BBBB BEL (16-bit)
		reader := bufio.NewReader(os.Stdin)

		// Read until we get a BEL character or timeout
		// Try to read response - use a more aggressive approach for Windows
		// Read in a loop with small delays to catch the response
		var response []byte
		startTime := time.Now()

		// Read aggressively - check buffer frequently
		for time.Since(startTime) < timeout {
			// First, check if data is already buffered
			buffered := reader.Buffered()
			if buffered > 0 {
				// Read all buffered data immediately
				buf := make([]byte, buffered)
				n, err := reader.Read(buf)
				if err == nil && n > 0 {
					response = append(response, buf[:n]...)

					// Check if we got a terminator (BEL or ST)
					belIdx := bytes.IndexByte(response, 0x07)
					stIdx := bytes.Index(response, []byte{0x1b, 0x5c}) // ESC \
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
						response = response[:termEnd]
						responseChan <- response
						return
					}
				}
			}

			// Try reading directly with a very short timeout
			quickDeadline := time.Now().Add(5 * time.Millisecond)
			os.Stdin.SetReadDeadline(quickDeadline)

			buf := make([]byte, 100)
			n, err := reader.Read(buf)
			if err == nil && n > 0 {
				response = append(response, buf[:n]...)

				// Check for terminator (BEL or ST)
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
					response = response[:termEnd]
					responseChan <- response
					return
				}
			}

			// Reset deadline for next iteration
			remaining := timeout - time.Since(startTime)
			if remaining > 0 {
				os.Stdin.SetReadDeadline(time.Now().Add(remaining))
			}

			// Very short delay - we want to catch it quickly
			time.Sleep(2 * time.Millisecond)

			// Safety: limit response size
			if len(response) > 100 {
				errChan <- fmt.Errorf("OSC 11 response too long")
				return
			}
		}

		// Timeout reached
		errChan <- fmt.Errorf("OSC 11 query timeout")
	}()

	// Wait for response with timeout
	select {
	case data := <-responseChan:
		return parseOSC11RGB(data)
	case err := <-errChan:
		return TerminalRGB{}, err
	case <-time.After(timeout):
		return TerminalRGB{}, fmt.Errorf("OSC 11 query timeout")
	}
}

// parseOSC11RGB parses OSC 11 response format
// Expected format: ESC ] 11 ; rgb:RR/GG/BB BEL
// or: ESC ] 11 ; rgb:RRRR/GGGG/BBBB BEL (16-bit)
func parseOSC11RGB(data []byte) (TerminalRGB, error) {
	// Convert to string for easier parsing
	str := string(data)

	// Look for rgb: pattern
	// Format: \x1b]11;rgb:RR/GG/BB\x07 or rgb:RRRR/GGGG/BBBB
	re := regexp.MustCompile(`rgb:([0-9a-fA-F]+)/([0-9a-fA-F]+)/([0-9a-fA-F]+)`)
	matches := re.FindStringSubmatch(str)
	if len(matches) != 4 {
		return TerminalRGB{}, fmt.Errorf("invalid OSC 11 response format: %q", str)
	}

	// Parse hex values
	rHex := matches[1]
	gHex := matches[2]
	bHex := matches[3]

	// Determine if 8-bit (2 hex digits) or 16-bit (4 hex digits)
	var maxVal float64
	if len(rHex) == 2 {
		maxVal = 255.0 // 8-bit
	} else if len(rHex) == 4 {
		maxVal = 65535.0 // 16-bit
	} else {
		return TerminalRGB{}, fmt.Errorf("invalid color value length in OSC 11 response")
	}

	// Parse and normalize to 0..1
	r, err := strconv.ParseUint(rHex, 16, 16)
	if err != nil {
		return TerminalRGB{}, fmt.Errorf("failed to parse red value: %w", err)
	}

	g, err := strconv.ParseUint(gHex, 16, 16)
	if err != nil {
		return TerminalRGB{}, fmt.Errorf("failed to parse green value: %w", err)
	}

	b, err := strconv.ParseUint(bHex, 16, 16)
	if err != nil {
		return TerminalRGB{}, fmt.Errorf("failed to parse blue value: %w", err)
	}

	return TerminalRGB{
		R: float64(r) / maxVal,
		G: float64(g) / maxVal,
		B: float64(b) / maxVal,
	}, nil
}

// isTerminal checks if a file descriptor is a terminal
// This is a simple cross-platform check
func isTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	// Check if it's a character device (typical for terminals)
	mode := stat.Mode()
	return (mode & os.ModeCharDevice) != 0
}
