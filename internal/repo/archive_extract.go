package repo

import (
	"fmt"
	"io"
	"math"
	"os"
)

// remainingBudget returns bytes still allowed under the total extraction budget.
func remainingBudget(policy ExtractionPolicy, totalWritten int64) int64 {
	rem := policy.MaxTotalBytes - totalWritten
	if rem < 0 {
		return 0
	}
	return rem
}

// checkEntryFitsBudget rejects an entry whose declared uncompressed size exceeds
// the per-file limit or the remaining aggregate budget.
func checkEntryFitsBudget(name string, uncompressed uint64, policy ExtractionPolicy, totalWritten int64) error {
	if sizeExceedsLimit(uncompressed, policy.MaxFileBytes) {
		return fmt.Errorf("%w: %q (%d bytes)", ErrArchiveEntryTooLarge, name, uncompressed)
	}
	rem := remainingBudget(policy, totalWritten)
	if sizeExceedsLimit(uncompressed, rem) {
		return fmt.Errorf("%w: %q needs %d bytes, %d remaining", ErrArchiveTotalLimit, name, uncompressed, rem)
	}
	return nil
}

// sizeExceedsLimit reports whether size is larger than an int64 budget limit.
func sizeExceedsLimit(size uint64, limit int64) bool {
	if limit < 0 {
		return true
	}
	if size > math.MaxInt64 {
		return true
	}
	return int64(size) > limit
}

// streamLimit returns LimitReader size (max allowed bytes + 1) so callers can detect overshoot.
func streamLimit(policy ExtractionPolicy, totalWritten int64) int64 {
	rem := remainingBudget(policy, totalWritten)
	return min(policy.MaxFileBytes, rem) + 1
}

// copyExtractedFile streams src into destPath with per-file and remaining-total limits.
// On failure the destination file is removed. Returns bytes written.
func copyExtractedFile(destPath string, src io.Reader, name string, policy ExtractionPolicy, totalWritten int64) (int64, error) {
	limit := streamLimit(policy, totalWritten)
	if limit <= 1 {
		return 0, fmt.Errorf("%w: %q", ErrArchiveTotalLimit, name)
	}

	destFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return 0, fmt.Errorf("failed to create destination file %s: %w", destPath, err)
	}

	n, copyErr := io.Copy(destFile, io.LimitReader(src, limit))
	closeErr := destFile.Close()
	extractErr := copyErr
	if extractErr == nil && closeErr != nil {
		extractErr = closeErr
	}
	if extractErr != nil {
		_ = os.Remove(destPath)
		return 0, fmt.Errorf("failed to extract file %s: %w", name, extractErr)
	}

	rem := remainingBudget(policy, totalWritten)
	if n > policy.MaxFileBytes {
		_ = os.Remove(destPath)
		return 0, fmt.Errorf("%w: %q", ErrArchiveEntryTooLarge, name)
	}
	if n > rem {
		_ = os.Remove(destPath)
		return 0, fmt.Errorf("%w: %q wrote %d bytes, %d remaining", ErrArchiveTotalLimit, name, n, rem)
	}
	return n, nil
}

// copyExtractedFileWithDeclaredSize checks declared metadata then streams with limits.
func copyExtractedFileWithDeclaredSize(destPath string, src io.Reader, name string, declared uint64, policy ExtractionPolicy, totalWritten int64) (int64, error) {
	if err := checkEntryFitsBudget(name, declared, policy, totalWritten); err != nil {
		return 0, err
	}
	return copyExtractedFile(destPath, src, name, policy, totalWritten)
}
