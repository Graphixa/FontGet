package repo

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strings"

	"fontget/internal/network"
)

var (
	// ErrChecksumMismatch is returned when a downloaded payload does not match its expected digest.
	ErrChecksumMismatch = errors.New("checksum mismatch")
	// ErrMalformedChecksum is returned when a non-empty expected digest is not a valid SHA-256 hex value.
	ErrMalformedChecksum = errors.New("malformed checksum")
	// ErrChecksumUnassociated is returned when an expected digest cannot be applied to the candidate payload.
	ErrChecksumUnassociated = errors.New("checksum does not apply to this payload")
	// ErrCandidatesExhausted is returned when every download candidate failed.
	ErrCandidatesExhausted = errors.New("download candidates exhausted")
)

// ParseExpectedSHA256 validates a supplied expected digest. Empty means no checksum was provided.
// A malformed non-empty value is an error, not equivalent to no checksum.
func ParseExpectedSHA256(expected string) (string, error) {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return "", nil
	}
	if strings.HasPrefix(expected, "sha256:") || strings.HasPrefix(expected, "SHA256:") {
		expected = strings.TrimSpace(expected[7:])
	}
	if len(expected) != 64 {
		return "", fmt.Errorf("%w: expected 64 hex characters", ErrMalformedChecksum)
	}
	for i := 0; i < len(expected); i++ {
		c := expected[i]
		hexDigit := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
		if !hexDigit {
			return "", fmt.Errorf("%w: non-hex character", ErrMalformedChecksum)
		}
	}
	return strings.ToLower(expected), nil
}

// VerifyFileSHA256 compares path bytes against an already-validated expected hex digest.
func VerifyFileSHA256(path, expectedHex string) error {
	expectedHex, err := ParseExpectedSHA256(expectedHex)
	if err != nil {
		return err
	}
	if expectedHex == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("%w: open downloaded file: %v", network.ErrLocalFailure, err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("%w: hash downloaded file: %v", network.ErrLocalFailure, err)
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != expectedHex {
		return fmt.Errorf("%w: expected %s, got %s", ErrChecksumMismatch, expectedHex, got)
	}
	return nil
}

// checksumAppliesToCandidate reports whether font.SHA may be used for candidateURL.
// Same URL or same basename (mirrors) may share a digest. Different representations cannot.
func checksumAppliesToCandidate(font *FontFile, candidateURL string) error {
	if font == nil {
		return nil
	}
	normalized, err := ParseExpectedSHA256(font.SHA)
	if err != nil {
		return err
	}
	if normalized == "" {
		return nil
	}
	font.SHA = normalized
	primary := strings.TrimSpace(font.DownloadURL)
	if primary == "" && len(font.DownloadCandidates) > 0 {
		primary = font.DownloadCandidates[0]
	}
	if sameChecksumPayload(primary, candidateURL) {
		return nil
	}
	return fmt.Errorf("%w: digest is bound to %s, not %s", ErrChecksumUnassociated, primary, candidateURL)
}

func sameChecksumPayload(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" || b == "" {
		return false
	}
	if normalizeDownloadURLKey(a) == normalizeDownloadURLKey(b) {
		return true
	}
	return payloadIdentity(a) == payloadIdentity(b) && payloadIdentity(a) != ""
}

func payloadIdentity(raw string) string {
	u, err := url.Parse(raw)
	name := raw
	if err == nil {
		name = u.Path
	}
	base := strings.ToLower(path.Base(name))
	if i := strings.IndexByte(base, '?'); i >= 0 {
		base = base[:i]
	}
	if base == "" || base == "." || base == "/" {
		return ""
	}
	return base
}
