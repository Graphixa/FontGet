package repo

import (
	"os"
	"path/filepath"
	"testing"

	"fontget/internal/logging"
)

// TestMain registers a file logger so production code paths that call logging.GetLogger()
// do not see nil outside cmd.PersistentPreRun.
func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "fontget-repo-test-*")
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
