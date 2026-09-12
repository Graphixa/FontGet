package installations

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fontget/internal/testutil"
)

func TestSaveRecoveryRecord(t *testing.T) {
	home := t.TempDir()
	testutil.SetHome(t, home)

	path, err := SaveRecoveryRecord(RecoveryRecord{
		OperationID:      "op1",
		PackageID:        "google.roboto",
		Scope:            "user",
		AffectedPaths:    []string{"/fonts/Roboto.ttf"},
		BackupPaths:      []string{"/fonts/Roboto.ttf.fontget-bak"},
		CompletedSteps:   []string{"restore Roboto.ttf"},
		OutstandingSteps: []string{"re-register Roboto.ttf"},
		Errors:           []string{"register failed"},
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !strings.Contains(body, "google.roboto") || !strings.Contains(body, "Roboto.ttf.fontget-bak") {
		t.Fatalf("missing fields: %s", body)
	}
	if filepath.Dir(path) != recoveryDir() {
		t.Fatalf("unexpected dir %s", path)
	}
}
