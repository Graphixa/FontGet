package repo

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestParseExpectedSHA256(t *testing.T) {
	if got, err := ParseExpectedSHA256(""); err != nil || got != "" {
		t.Fatalf("empty: got %q err=%v", got, err)
	}
	ok := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	got, err := ParseExpectedSHA256("SHA256:" + ok)
	if err != nil || got != ok {
		t.Fatalf("prefixed: got %q err=%v", got, err)
	}
	if _, err := ParseExpectedSHA256("not-a-hash"); err == nil || !errors.Is(err, ErrMalformedChecksum) {
		t.Fatalf("malformed: err=%v", err)
	}
	if _, err := ParseExpectedSHA256("xyz"); err == nil || !errors.Is(err, ErrMalformedChecksum) {
		t.Fatalf("short: err=%v", err)
	}
}

func TestVerifyFileSHA256(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.bin")
	data := []byte("font-bytes")
	if err := os.WriteFile(p, data, 0644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	hexSum := hex.EncodeToString(sum[:])
	if err := VerifyFileSHA256(p, hexSum); err != nil {
		t.Fatalf("correct digest: %v", err)
	}
	if err := VerifyFileSHA256(p, ""); err != nil {
		t.Fatalf("no digest: %v", err)
	}
	wrong := hex.EncodeToString(make([]byte, 32))
	if err := VerifyFileSHA256(p, wrong); err == nil || !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("wrong digest: %v", err)
	}
}

func TestChecksumAppliesToCandidate(t *testing.T) {
	sum := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	font := &FontFile{
		SHA:                sum,
		DownloadURL:        "https://cdn.example/a.zip",
		DownloadCandidates: []string{"https://cdn.example/a.zip", "https://cdn.example/a.tar.xz"},
	}
	if err := checksumAppliesToCandidate(font, "https://mirror.example/a.zip"); err != nil {
		t.Fatalf("mirror zip should share digest: %v", err)
	}
	err := checksumAppliesToCandidate(font, "https://cdn.example/a.tar.xz")
	if err == nil || !errors.Is(err, ErrChecksumUnassociated) {
		t.Fatalf("tar.xz must not use zip digest: %v", err)
	}
}
