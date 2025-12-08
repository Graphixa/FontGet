package update

import (
	"fmt"
	"os"
	"strings"

	"fontget/internal/version"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
)

const (
	// GitHub repository information
	githubSlug = "Graphixa/FontGet" // owner/repo format
)

// UpdateResult represents the result of checking for updates
type UpdateResult struct {
	Available   bool
	Current     string
	Latest      string
	Release     *selfupdate.Release
	NeedsUpdate bool
}

// CheckForUpdates checks if updates are available from GitHub Releases
func CheckForUpdates() (*UpdateResult, error) {
	updater, err := selfupdate.NewUpdater(selfupdate.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize updater: %w", err)
	}

	currentVersionStr := version.GetVersion()
	latest, found, err := updater.DetectLatest(githubSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}

	if !found {
		return &UpdateResult{
			Available:   false,
			Current:     currentVersionStr,
			Latest:      "",
			Release:     nil,
			NeedsUpdate: false,
		}, nil
	}

	// Convert current version to semver for comparison
	currentVersion, err := parseVersion(currentVersionStr)
	if err != nil {
		// If we can't parse current version, assume update is needed if versions differ as strings
		needsUpdate := latest.Version.String() != currentVersionStr
		return &UpdateResult{
			Available:   true,
			Current:     currentVersionStr,
			Latest:      latest.Version.String(),
			Release:     latest,
			NeedsUpdate: needsUpdate,
		}, nil
	}

	// Compare versions using semver
	needsUpdate := latest.Version.GT(currentVersion)

	return &UpdateResult{
		Available:   true,
		Current:     currentVersionStr,
		Latest:      latest.Version.String(),
		Release:     latest,
		NeedsUpdate: needsUpdate,
	}, nil
}

// UpdateToLatest updates FontGet to the latest version
func UpdateToLatest() error {
	updater, err := selfupdate.NewUpdater(selfupdate.Config{})
	if err != nil {
		return fmt.Errorf("failed to initialize updater: %w", err)
	}

	currentVersionStr := version.GetVersion()
	currentVersion, err := parseVersion(currentVersionStr)
	if err != nil {
		return fmt.Errorf("failed to parse current version '%s': %w", currentVersionStr, err)
	}

	// Get executable path
	cmdPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Library handles: download, checksum verification, binary replacement, rollback
	_, err = updater.UpdateCommand(cmdPath, currentVersion, githubSlug)
	if err != nil {
		return mapLibraryError(err)
	}

	// Clean up old binary backup file after successful update
	// The library may create a .old file on Windows during the update process
	cleanupOldBinary(cmdPath)

	return nil
}

// UpdateToVersion updates FontGet to a specific version
func UpdateToVersion(targetVersion string) error {
	updater, err := selfupdate.NewUpdater(selfupdate.Config{})
	if err != nil {
		return fmt.Errorf("failed to initialize updater: %w", err)
	}

	// Parse target version
	targetSemver, err := semver.Parse(targetVersion)
	if err != nil {
		return fmt.Errorf("invalid version format '%s': %w", targetVersion, err)
	}

	// Find the release with the target version
	// The library handles both "1.0.4" and "v1.0.4" formats automatically,
	// but we'll try both to be safe
	release, found, err := updater.DetectVersion(githubSlug, targetVersion)
	if err != nil {
		return fmt.Errorf("failed to check for version '%s': %w", targetVersion, err)
	}

	// If not found, try with "v" prefix (common GitHub tag format)
	if !found {
		release, found, err = updater.DetectVersion(githubSlug, "v"+targetVersion)
		if err != nil {
			return fmt.Errorf("failed to check for version 'v%s': %w", targetVersion, err)
		}
	}

	if !found {
		return fmt.Errorf("version %s not found on GitHub. The release may not exist, may be a pre-release (which are ignored), or may not have assets for your platform", targetVersion)
	}

	// Verify the found version matches target
	if !release.Version.EQ(targetSemver) {
		return fmt.Errorf("version mismatch: found %s, expected %s", release.Version.String(), targetVersion)
	}

	// Get executable path
	cmdPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Find the asset URL for the current platform
	assetURL := release.AssetURL
	if assetURL == "" {
		return fmt.Errorf("no asset URL found for version %s", targetVersion)
	}

	// Update to the specific release using UpdateTo
	// This handles: download, checksum verification, binary replacement, rollback
	err = selfupdate.UpdateTo(assetURL, cmdPath)
	if err != nil {
		return mapLibraryError(err)
	}

	// Clean up old binary backup file after successful update
	// The library may create a .old file on Windows during the update process
	cleanupOldBinary(cmdPath)

	return nil
}

// cleanupOldBinary removes the .old backup file created during updates
// On Windows, the self-update library renames the old binary to .old before replacing it
// This function cleans up that backup file after a successful update
// Errors during cleanup are silently ignored to avoid failing the update process
func cleanupOldBinary(execPath string) {
	// Construct the .old backup file path
	oldPath := execPath + ".old"

	// Check if the backup file exists
	if _, err := os.Stat(oldPath); err == nil {
		// File exists, try to remove it
		if err := os.Remove(oldPath); err != nil {
			// Silently ignore cleanup errors - the .old file is just a backup
			// and can be manually removed if needed. We don't want to fail the
			// update process if cleanup fails, as the update itself was successful.
			_ = err
		}
	}
}

// parseVersion parses a version string to semver.Version
// Handles "dev", "dev+{hash}", and version strings with or without "v" prefix
func parseVersion(versionStr string) (semver.Version, error) {
	// Handle "dev" version (with or without commit hash in build metadata)
	if versionStr == "dev" || strings.HasPrefix(versionStr, "dev+") {
		// Return a very old version so updates are always available
		return semver.MustParse("0.0.0"), nil
	}

	// Remove "v" prefix if present
	versionStr = strings.TrimPrefix(versionStr, "v")

	// Parse semver
	return semver.Parse(versionStr)
}

// mapLibraryError converts library errors to user-friendly messages
func mapLibraryError(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// Check for common error patterns
	switch {
	case strings.Contains(errStr, "permission denied") || strings.Contains(errStr, "Access is denied"):
		return fmt.Errorf("insufficient permissions: try running as administrator/sudo")
	case strings.Contains(errStr, "file is locked") || strings.Contains(errStr, "being used by another process"):
		return fmt.Errorf("FontGet is currently running: please close other instances and try again")
	case strings.Contains(errStr, "network") || strings.Contains(errStr, "connection") || strings.Contains(errStr, "timeout"):
		return fmt.Errorf("network error: check your internet connection and try again")
	case strings.Contains(errStr, "checksum") || strings.Contains(errStr, "verification"):
		return fmt.Errorf("download verification failed: the downloaded file may be corrupted")
	case strings.Contains(errStr, "404") || strings.Contains(errStr, "not found"):
		return fmt.Errorf("release not found: the specified version may not exist")
	case strings.Contains(errStr, "403") || strings.Contains(errStr, "forbidden"):
		return fmt.Errorf("access denied: GitHub API may be rate-limited or unavailable")
	}

	// Return original error if no pattern matches
	return fmt.Errorf("update failed: %w", err)
}

// IsUpdateInProgress checks if an update is currently in progress
// This can be used to prevent multiple update attempts
func IsUpdateInProgress() bool {
	// Check if a lock file exists (library may create one during update)
	// For now, we'll rely on the library's internal locking
	return false
}

// GetBinaryPath returns the path to the current FontGet binary
func GetBinaryPath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	return execPath, nil
}
