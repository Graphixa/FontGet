package network

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestClassifyHTTPStatus(t *testing.T) {
	cases := []struct {
		code int
		bot  bool
		want DownloadAction
	}{
		{200, false, ActionSuccess},
		{404, false, ActionAdvanceCandidate},
		{410, false, ActionAdvanceCandidate},
		{429, false, ActionRateLimit},
		{403, false, ActionAdvanceCandidate},
		{401, false, ActionAdvanceCandidate},
		{500, false, ActionRetrySame},
		{502, false, ActionRetrySame},
		{202, true, ActionExternalFallback},
		{403, true, ActionExternalFallback},
	}
	for _, tc := range cases {
		if got := ClassifyHTTPStatus(tc.code, tc.bot); got != tc.want {
			t.Errorf("ClassifyHTTPStatus(%d, bot=%v)=%v want %v", tc.code, tc.bot, got, tc.want)
		}
	}
}

func TestHTTPStatusErrorUnwrap(t *testing.T) {
	err404 := NewHTTPStatusError(404, "https://example.com/x?token=secret", 0)
	if !errors.Is(err404, ErrCandidateUnavailable) {
		t.Fatalf("404 must unwrap to ErrCandidateUnavailable")
	}
	msg := err404.Error()
	if strings.Contains(msg, "secret") {
		t.Fatalf("url credentials leaked in %q", msg)
	}

	err429 := NewHTTPStatusError(429, "https://example.com/x", time.Second)
	if !errors.Is(err429, ErrRateLimited) {
		t.Fatalf("429 must unwrap to ErrRateLimited")
	}
}

func TestClassifyDownloadError(t *testing.T) {
	if ClassifyDownloadError(context.Canceled) != ActionFailPackage {
		t.Fatal("cancel must fail package")
	}
	if ClassifyDownloadError(&os.PathError{Op: "write", Path: "/tmp/x", Err: os.ErrPermission}) != ActionFailLocal {
		t.Fatal("path error must fail local")
	}
	if ClassifyDownloadError(NewHTTPStatusError(404, "https://example.com/a", 0)) != ActionAdvanceCandidate {
		t.Fatal("404 error must advance")
	}
}

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	h := http.Header{}
	h.Set("Retry-After", "5")
	d, ok := ParseRetryAfter(h, now)
	if !ok || d != 5*time.Second {
		t.Fatalf("delta seconds: got %v ok=%v", d, ok)
	}
	h.Set("Retry-After", now.Add(3*time.Second).Format(http.TimeFormat))
	d, ok = ParseRetryAfter(h, now)
	if !ok || d < 2*time.Second || d > 4*time.Second {
		t.Fatalf("http-date wait: got %v ok=%v", d, ok)
	}
}

func TestRetryAfterWithinBudget(t *testing.T) {
	old := MaxRetryAfterWait
	MaxRetryAfterWait = 2 * time.Second
	t.Cleanup(func() { MaxRetryAfterWait = old })
	if RetryAfterWithinBudget(3 * time.Second) {
		t.Fatal("over-budget wait must be rejected")
	}
	if !RetryAfterWithinBudget(time.Second) {
		t.Fatal("in-budget wait must be allowed")
	}
}

func TestDrainAndCloseBodyBoundsWait(t *testing.T) {
	pr, pw := io.Pipe()
	go func() {
		// Never write — drain must time out and close.
		time.Sleep(5 * time.Second)
		_ = pw.Close()
	}()
	start := time.Now()
	DrainAndCloseBody(pr)
	if time.Since(start) > 4*time.Second {
		t.Fatalf("drain hung for %s", time.Since(start))
	}
}

func TestRedactDownloadURL(t *testing.T) {
	got := RedactDownloadURL("https://user:pass@example.com/f.ttf?token=abc&x=1")
	if strings.Contains(got, "pass") || strings.Contains(got, "abc") {
		t.Fatalf("leaked secret: %q", got)
	}
	if !strings.Contains(got, "REDACTED") {
		t.Fatalf("expected redacted query: %q", got)
	}
}
