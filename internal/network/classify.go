package network

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// maxErrorBodyDrain bounds how much of a failed HTTP body we discard before closing.
	maxErrorBodyDrain = 64 << 10
	// defaultMaxRetryAfterWait is the allowed wait budget for honoring Retry-After.
	defaultMaxRetryAfterWait = 60 * time.Second
)

// Download classification sentinels. Callers should use errors.Is.
var (
	ErrCandidateUnavailable = errors.New("download candidate unavailable")
	ErrRateLimited          = errors.New("download rate limited")
	ErrLocalFailure         = errors.New("local filesystem failure")
)

// MaxRetryAfterWait is the allowed wait budget for Retry-After. Tests may lower it.
var MaxRetryAfterWait = defaultMaxRetryAfterWait

// DownloadAction is the centralized decision after a transport result.
type DownloadAction int

const (
	// ActionSuccess means the payload is ready for completion/validation.
	ActionSuccess DownloadAction = iota
	// ActionAdvanceCandidate means this URL is gone; try the next candidate, no external tools.
	ActionAdvanceCandidate
	// ActionRetrySame means retry the same URL with the same transport (bounded).
	ActionRetrySame
	// ActionRateLimit means wait (if within budget) or fail as rate-limited; do not change tools.
	ActionRateLimit
	// ActionExternalFallback means a recognised challenge and external tools are allowed.
	ActionExternalFallback
	// ActionFailCandidate means a permanent candidate failure without blind same-URL tool retries.
	ActionFailCandidate
	// ActionFailPackage means stop the package (checksum, cancel, unassociated digest).
	ActionFailPackage
	// ActionFailLocal means a local I/O/permission error; switching transports cannot repair it.
	ActionFailLocal
)

// HTTPStatusError carries a classified HTTP failure without requiring string matching.
type HTTPStatusError struct {
	StatusCode int
	URL        string
	RetryAfter time.Duration
}

func (e *HTTPStatusError) Error() string {
	if e == nil {
		return "http status error"
	}
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, RedactDownloadURL(e.URL))
}

func (e *HTTPStatusError) Unwrap() error {
	if e == nil {
		return nil
	}
	switch e.StatusCode {
	case http.StatusNotFound, http.StatusGone:
		return ErrCandidateUnavailable
	case http.StatusTooManyRequests:
		return ErrRateLimited
	default:
		return nil
	}
}

// ClassifyHTTPStatus decides the next download action for a native or external HTTP status.
func ClassifyHTTPStatus(statusCode int, botChallenge bool) DownloadAction {
	if botChallenge {
		return ActionExternalFallback
	}
	switch statusCode {
	case http.StatusOK:
		return ActionSuccess
	case http.StatusNotFound, http.StatusGone:
		return ActionAdvanceCandidate
	case http.StatusTooManyRequests:
		return ActionRateLimit
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return ActionRetrySame
	default:
		if statusCode >= 400 && statusCode < 500 {
			return ActionFailCandidate
		}
		if statusCode >= 500 {
			return ActionRetrySame
		}
		return ActionFailCandidate
	}
}

// ClassifyDownloadError maps a transport error to an action when no HTTP status is available.
func ClassifyDownloadError(err error) DownloadAction {
	if err == nil {
		return ActionSuccess
	}
	if isCancelErr(err) {
		return ActionFailPackage
	}
	if errors.Is(err, ErrCandidateUnavailable) {
		return ActionAdvanceCandidate
	}
	if errors.Is(err, ErrRateLimited) {
		return ActionRateLimit
	}
	if isLocalIOError(err) {
		return ActionFailLocal
	}
	var httpErr *HTTPStatusError
	if errors.As(err, &httpErr) && httpErr != nil {
		return ClassifyHTTPStatus(httpErr.StatusCode, false)
	}
	return ActionExternalFallback
}

func isCancelErr(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func isLocalIOError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrLocalFailure) {
		return true
	}
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		return true
	}
	var linkErr *os.LinkError
	if errors.As(err, &linkErr) {
		return true
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "permission denied") || strings.Contains(msg, "access is denied") {
		return true
	}
	if strings.Contains(msg, "no space left") || strings.Contains(msg, "disk full") {
		return true
	}
	if strings.Contains(msg, "read-only file system") {
		return true
	}
	return false
}

// DrainAndCloseBody closes the body promptly. A short, bounded drain avoids holding a host
// slot on a stalled error body; on drain timeout the body is closed immediately.
func DrainAndCloseBody(body io.ReadCloser) {
	if body == nil {
		return
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = io.Copy(io.Discard, io.LimitReader(body, maxErrorBodyDrain))
		_ = body.Close()
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		_ = body.Close()
		<-done
	}
}

// ParseRetryAfter returns the wait duration from a Retry-After header.
func ParseRetryAfter(h http.Header, now time.Time) (time.Duration, bool) {
	if h == nil {
		return 0, false
	}
	raw := strings.TrimSpace(h.Get("Retry-After"))
	if raw == "" {
		return 0, false
	}
	if secs, err := strconv.Atoi(raw); err == nil {
		if secs < 0 {
			return 0, false
		}
		return time.Duration(secs) * time.Second, true
	}
	if when, err := http.ParseTime(raw); err == nil {
		d := when.Sub(now)
		if d < 0 {
			return 0, false
		}
		return d, true
	}
	return 0, false
}

// RetryAfterWithinBudget reports whether wait is allowed. If wait exceeds the budget, callers
// must return a rate-limit error rather than retrying early.
func RetryAfterWithinBudget(wait time.Duration) bool {
	if wait <= 0 {
		return true
	}
	return wait <= MaxRetryAfterWait
}

// RedactDownloadURL strips credentials and sensitive query values from user-facing messages.
func RedactDownloadURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" {
		return raw
	}
	u.User = nil
	q := u.Query()
	if len(q) > 0 {
		for key := range q {
			lk := strings.ToLower(key)
			switch lk {
			case "token", "key", "signature", "sig", "access_token", "auth", "password", "secret":
				q.Set(key, "REDACTED")
			}
		}
		u.RawQuery = q.Encode()
	}
	return u.String()
}

// NewHTTPStatusError builds a classified HTTP error with the original URL (redacted in Error()).
func NewHTTPStatusError(status int, rawURL string, retryAfter time.Duration) *HTTPStatusError {
	return &HTTPStatusError{StatusCode: status, URL: rawURL, RetryAfter: retryAfter}
}
