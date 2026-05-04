package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"fontget/internal/logging"
)

// TestMain ensures logging.GetLogger() is non-nil for tests that do not execute cobra
// PersistentPreRun (e.g. integration tests calling GetLogger() directly).
func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "fontget-cmd-test-*")
	if err != nil {
		panic(err)
	}

	l, err := logging.NewWithPath(logging.DefaultConfig(), filepath.Join(tmp, "fontget.log"))
	if err != nil {
		_ = os.RemoveAll(tmp)
		panic(err)
	}
	logging.SetGlobal(l)
	code := m.Run()
	_ = logging.CloseGlobal()
	_ = os.RemoveAll(tmp)
	os.Exit(code)
}
