package update

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"fontget/internal/config"
	"fontget/internal/logging"
	"fontget/internal/version"

	"github.com/blang/semver"
)

// executablePath resolves the current binary. Tests override this.
var executablePath = os.Executable

// migrateAfterUpdate is called after a successful binary replace. Tests stub this
// so they do not touch the real user config.
var migrateAfterUpdate = config.MigrateConfigAfterUpdate

// UpdateResult represents the result of checking for updates
type UpdateResult struct {
	Available   bool
	Current     string
	Latest      string
	NeedsUpdate bool
}

// CheckForUpdates checks if updates are available from GitHub Releases
func CheckForUpdates() (*UpdateResult, error) {
	currentVersionStr := version.GetVersion()
	client, err := newReleaseClient()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize updater: %w", err)
	}
	latestVersion, err := client.latestVersion(context.Background())
	if err != nil {
		if errors.Is(err, errReleaseNotFound) {
			return &UpdateResult{
				Available:   false,
				Current:     currentVersionStr,
				Latest:      "",
				NeedsUpdate: false,
			}, nil
		}
		return nil, fmt.Errorf("failed to check for updates: %w", mapLibraryError(err))
	}

	currentVersion, err := parseVersion(currentVersionStr)
	if err != nil {
		needsUpdate := latestVersion.String() != currentVersionStr
		return &UpdateResult{
			Available:   true,
			Current:     currentVersionStr,
			Latest:      latestVersion.String(),
			NeedsUpdate: needsUpdate,
		}, nil
	}

	return &UpdateResult{
		Available:   true,
		Current:     currentVersionStr,
		Latest:      latestVersion.String(),
		NeedsUpdate: latestVersion.GT(currentVersion),
	}, nil
}

// UpdateToLatest updates FontGet to the latest version
func UpdateToLatest() error {
	client, err := newReleaseClient()
	if err != nil {
		return fmt.Errorf("failed to initialize updater: %w", err)
	}
	latestVersion, err := client.latestVersion(context.Background())
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", mapLibraryError(err))
	}
	return applyVersion(client, latestVersion)
}

// UpdateToVersion updates FontGet to a specific version
func UpdateToVersion(targetVersion string) error {
	targetSemver, err := parseVersion(targetVersion)
	if err != nil {
		return fmt.Errorf("invalid version format '%s': %w", targetVersion, err)
	}

	client, err := newReleaseClient()
	if err != nil {
		return fmt.Errorf("failed to initialize updater: %w", err)
	}
	return applyVersion(client, targetSemver)
}

func applyVersion(client *releaseClient, releaseVersion semver.Version) error {
	archiveName := currentArchiveName(releaseVersion.String())
	ctx := context.Background()
	checksumBytes, err := client.checksums(ctx, releaseVersion)
	if err != nil {
		return mapLibraryError(fmt.Errorf("failed to download checksums.txt: %w", err))
	}
	expected, err := checksumForFile(checksumBytes, archiveName)
	if err != nil {
		return err
	}

	archiveBytes, err := client.archive(ctx, releaseVersion, archiveName)
	if err != nil {
		return mapLibraryError(fmt.Errorf("failed to download %s: %w", archiveName, err))
	}
	if err := verifySHA256(archiveBytes, expected); err != nil {
		return mapLibraryError(err)
	}

	binary, err := extractBinary(archiveBytes, archiveName)
	if err != nil {
		return fmt.Errorf("failed to extract binary from %s: %w", archiveName, err)
	}

	cmdPath, err := executablePath()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	if err := replaceExecutable(cmdPath, binary); err != nil {
		return mapLibraryError(err)
	}

	cleanupOldBinary(cmdPath)

	if err := migrateAfterUpdate(); err != nil {
		if logger := logging.GetLogger(); logger != nil {
			logger.Warn("Config migration after update failed: %v (will migrate on next command)", err)
		}
	}

	return nil
}

// cleanupOldBinary removes the .old backup file created during updates.
// On Windows the updater renames the old binary to .old before replacing it.
// Errors during cleanup are ignored so a successful update is not failed.
func cleanupOldBinary(execPath string) {
	oldPath := execPath + ".old"
	if _, err := os.Stat(oldPath); err == nil {
		_ = os.Remove(oldPath)
	}
}

// parseVersion parses a version string to semver.Version.
// Handles "dev" (and other dev-prefixed strings) and versions with or without a "v" prefix.
func parseVersion(versionStr string) (semver.Version, error) {
	trimmed := strings.TrimSpace(versionStr)
	if trimmed == "" {
		return semver.Version{}, fmt.Errorf("empty version")
	}
	if trimmed == "dev" || strings.HasPrefix(trimmed, "dev-") || strings.HasPrefix(trimmed, "dev+") {
		return semver.MustParse("0.0.0"), nil
	}
	trimmed = strings.TrimPrefix(trimmed, "v")
	return semver.Parse(trimmed)
}

// mapLibraryError converts update errors to user-friendly messages
func mapLibraryError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, errReleaseNotFound) {
		return fmt.Errorf("release not found: the specified version may not exist")
	}

	errStr := err.Error()

	switch {
	case strings.Contains(errStr, "permission denied") || strings.Contains(errStr, "Access is denied"):
		return fmt.Errorf("insufficient permissions: try running as administrator/sudo")
	case strings.Contains(errStr, "file is locked") || strings.Contains(errStr, "being used by another process"):
		return fmt.Errorf("FontGet is currently running: please close other instances and try again")
	case strings.Contains(errStr, "checksum verification failed"):
		return fmt.Errorf("download verification failed: the downloaded file may be corrupted")
	case strings.Contains(errStr, "403") || strings.Contains(errStr, "forbidden"):
		return fmt.Errorf("access denied: GitHub releases may be unavailable")
	case strings.Contains(errStr, "network") || strings.Contains(errStr, "connection") || strings.Contains(errStr, "timeout") || strings.Contains(errStr, "DeadlineExceeded"):
		return fmt.Errorf("network error: check your internet connection and try again")
	}

	return fmt.Errorf("update failed: %w", err)
}

// IsUpdateInProgress checks if an update is currently in progress
func IsUpdateInProgress() bool {
	return false
}

// GetBinaryPath returns the path to the current FontGet binary
func GetBinaryPath() (string, error) {
	execPath, err := executablePath()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	return execPath, nil
}
