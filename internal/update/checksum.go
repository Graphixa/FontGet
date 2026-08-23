package update

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// parseChecksums parses a GoReleaser/sha256sum checksums.txt file.
// Each non-comment line must be: <64-hex>  filename (with an optional
// "*" prefix on the filename). Duplicate filenames are rejected.
func parseChecksums(content []byte) (map[string]string, error) {
	sums := make(map[string]string)
	for lineNumber, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return nil, fmt.Errorf("invalid checksums.txt line %d", lineNumber+1)
		}
		sum := strings.ToLower(fields[0])
		if len(sum) != 64 || !isHex(sum) {
			return nil, fmt.Errorf("invalid SHA256 on checksums.txt line %d", lineNumber+1)
		}
		name := strings.TrimPrefix(fields[1], "*")
		if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
			return nil, fmt.Errorf("invalid filename on checksums.txt line %d", lineNumber+1)
		}
		if _, exists := sums[name]; exists {
			return nil, fmt.Errorf("duplicate checksum entry for %s", name)
		}
		sums[name] = sum
	}
	if len(sums) == 0 {
		return nil, fmt.Errorf("checksums.txt contains no entries")
	}
	return sums, nil
}

func checksumForFile(content []byte, filename string) (string, error) {
	sums, err := parseChecksums(content)
	if err != nil {
		return "", err
	}
	sum, ok := sums[filename]
	if !ok {
		return "", fmt.Errorf("checksums.txt does not contain a SHA256 entry for %s", filename)
	}
	return sum, nil
}

func verifySHA256(data []byte, expectedHex string) error {
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if !strings.EqualFold(got, expectedHex) {
		return fmt.Errorf("checksum verification failed: expected %s, got %s", strings.ToLower(expectedHex), got)
	}
	return nil
}

func isHex(s string) bool {
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
