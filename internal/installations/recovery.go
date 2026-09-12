package installations

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"fontget/internal/config"
	"fontget/internal/platform"
	"fontget/internal/shared"
)

const recoveryDirName = "recovery"

// RecoveryRecord describes incomplete rollback so a human can restore fonts manually.
// It must not contain secrets or signed download URLs.
type RecoveryRecord struct {
	OperationID      string    `json:"operation_id"`
	PackageID        string    `json:"package_id"`
	Scope            string    `json:"scope"`
	AffectedPaths    []string  `json:"affected_paths"`
	BackupPaths      []string  `json:"backup_paths"`
	CompletedSteps   []string  `json:"completed_steps"`
	OutstandingSteps []string  `json:"outstanding_steps"`
	Errors           []string  `json:"errors"`
	CreatedAt        time.Time `json:"created_at"`
}

func recoveryDir() string {
	return filepath.Join(config.GetAppConfigDir(), recoveryDirName)
}

// SaveRecoveryRecord persists recovery information. Backups listed in the record must be left in place.
func SaveRecoveryRecord(rec RecoveryRecord) (string, error) {
	if rec.OperationID == "" {
		rec.OperationID = fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	}
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now().UTC()
	}
	dir := recoveryDir()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("%w: create recovery dir: %v", shared.ErrRecoveryRequired, err)
	}
	name := rec.OperationID
	if rec.PackageID != "" {
		part := platform.SanitizePathPart(rec.PackageID)
		if part == "" {
			part = "package"
		}
		name = rec.OperationID + "-" + part
	}
	path := filepath.Join(dir, name+".json")
	payload, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return "", fmt.Errorf("%w: marshal recovery: %v", shared.ErrRecoveryRequired, err)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		return "", fmt.Errorf("%w: write recovery: %v", shared.ErrRecoveryRequired, err)
	}
	return path, nil
}
