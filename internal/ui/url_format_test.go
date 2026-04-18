package ui

import (
	"strings"
	"testing"
)

func TestFormatTerminalURL_osc8HTTPS(t *testing.T) {
	s := FormatTerminalURL("https://example.com/path")
	if !strings.Contains(s, "\x1b]8;;https://example.com/path\x07") {
		t.Fatalf("expected OSC 8 hyperlink, got %q", s)
	}
	if !strings.Contains(s, "\x1b]8;;\x07") { // reset
		t.Fatalf("expected hyperlink reset in output")
	}
}

func TestFormatTerminalURL_nonHTTPPlain(t *testing.T) {
	s := FormatTerminalURL("ftp://example.com")
	if strings.Contains(s, "\x1b]8;") {
		t.Fatalf("did not expect OSC 8 for ftp URL")
	}
}

func TestFormatTerminalURL_empty(t *testing.T) {
	if FormatTerminalURL("") != "" {
		t.Fatal("expected empty")
	}
}
