package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// TempDirName is the name of our temporary directory (shared container only).
	TempDirName        = "Fontget"
	operationDirPrefix = "op-"
)

// GetTempDir returns the platform-specific temp directory path for Fontget.
// This is a shared container for per-operation directories. Callers must not
// treat it as exclusive working space or delete it during normal cleanup.
func GetTempDir() (string, error) {
	tempDir := os.TempDir()
	if tempDir == "" {
		return "", fmt.Errorf("failed to get system temp directory")
	}

	fontgetTempDir := filepath.Join(tempDir, TempDirName)
	if err := os.MkdirAll(fontgetTempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	return fontgetTempDir, nil
}

// CleanupTempDir removes the shared Fontget temp container if it is empty.
// It never deletes unrelated sibling directories. Prefer OperationStaging.Cleanup.
func CleanupTempDir() error {
	tempDir, err := GetTempDir()
	if err != nil {
		return err
	}
	if err := os.Remove(tempDir); err != nil && !os.IsNotExist(err) {
		// Non-empty container is expected while other operations are running.
		if isNotEmptyDirErr(err) {
			return nil
		}
		return fmt.Errorf("failed to cleanup temp directory: %w", err)
	}
	return nil
}

func isNotEmptyDirErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "directory not empty") || strings.Contains(msg, "not empty")
}

// OperationStaging owns a unique temporary directory for one add/install command.
// Cleanup removes only this directory, never the shared Fontget root or siblings.
type OperationStaging struct {
	Root string
}

// NewOperationStaging allocates a unique operation directory under the shared Fontget temp root.
func NewOperationStaging() (*OperationStaging, error) {
	base, err := GetTempDir()
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp(base, operationDirPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to create operation temp directory: %w", err)
	}
	return &OperationStaging{Root: dir}, nil
}

// VariantDir returns a dedicated directory for one variant inside a package.
func (s *OperationStaging) VariantDir(packageID, variant string) (string, error) {
	if s == nil || s.Root == "" {
		return "", fmt.Errorf("nil operation staging")
	}
	pkgName := SanitizePathPart(packageID)
	if pkgName == "" {
		pkgName = "package"
	}
	varName := SanitizePathPart(variant)
	if varName == "" {
		varName = "variant"
	}
	dir := filepath.Join(s.Root, "pkg-"+pkgName, "var-"+varName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create variant staging directory: %w", err)
	}
	return dir, nil
}

// Cleanup removes this operation's disposable staging tree only.
func (s *OperationStaging) Cleanup() error {
	if s == nil || s.Root == "" {
		return nil
	}
	root := s.Root
	s.Root = ""
	if err := os.RemoveAll(root); err != nil {
		return fmt.Errorf("failed to cleanup operation staging %s: %w", root, err)
	}
	return nil
}

// SanitizePathPart returns a filesystem-safe fragment for staging and recovery filenames.
func SanitizePathPart(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := b.String()
	if len(out) > 80 {
		out = out[:80]
	}
	return out
}
