package cmd

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"fontget/internal/cmdutils"
	"fontget/internal/components"
	"fontget/internal/output"
	"fontget/internal/platform"
	"fontget/internal/repo"
	"fontget/internal/shared"
	"fontget/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

const (
	// DefaultSourceName is used when a font's source cannot be determined
	DefaultSourceName = "Other"
)

// backupResult tracks the result of a backup operation
type backupResult struct {
	familyCount int
	fileCount   int
}

var backupCmd = &cobra.Command{
	Use:           "backup [output-path]",
	Short:         "Backup installed font files to a zip archive",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `Backup installed font files to a zip archive organized by source and family name.

Automatically detects accessible scopes (user-only for regular users, both scopes for administrators).
Fonts are deduplicated across scopes. System fonts are always excluded.

Use --scope to specify which scopes to backup when providing an output path.`,
	Example: `  fontget backup
  fontget backup fonts-backup.zip
  fontget backup D:\Backups\my-fonts.zip --force
  fontget backup ./backups/ --scope user
  fontget backup --scope machine --force`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Output path is optional - will use default name if not provided
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		GetLogger().Info("Starting font backup operation")

		// Always start with a blank line for consistent spacing
		fmt.Println()

		// Debug output for operation start
		output.GetDebug().State("Starting font backup operation")

		// Get flags
		force, _ := cmd.Flags().GetBool("force")
		scopeFlag, _ := cmd.Flags().GetString("scope")

		// Ensure manifest system is initialized
		if err := cmdutils.EnsureManifestInitialized(func() cmdutils.Logger { return GetLogger() }); err != nil {
			return err
		}

		fontManager, err := cmdutils.CreateFontManager(func() cmdutils.Logger { return GetLogger() })
		if err != nil {
			return err
		}

		// Get output path from args or use default
		var outputPath string
		if len(args) > 0 {
			outputPath = args[0]
		}

		// Validate output path using shared utility
		defaultFilename := generateDefaultBackupFilename()
		zipPath, needsOverwriteConfirm, err := cmdutils.ValidateOutputPath(outputPath, defaultFilename, ".zip", force)
		if err != nil {
			// Check if this is a path validation error
			var pathErr *shared.PathValidationError
			if errors.As(err, &pathErr) {
				cmdutils.PrintError(pathErr.Error())
				fmt.Println()
				return nil
			}
			return err
		}

		// If force flag is set, skip overwrite confirmation
		if force {
			needsOverwriteConfirm = false
		}

		// Auto-detect accessible scopes
		availableScopes, err := cmdutils.DetectAccessibleScopes(fontManager)
		if err != nil {
			GetLogger().Error("Failed to detect accessible scopes: %v", err)
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("cmdutils.DetectAccessibleScopes() failed: %v", err)
			return fmt.Errorf("unable to detect accessible font scopes: %w", err)
		}

		// Determine selected scopes based on flags
		var selectedScopes []platform.InstallationScope

		// Parse scope flag if provided
		if scopeFlag != "" {
			// Check if machine scope is requested (requires elevation)
			scopeFlagLower := strings.ToLower(strings.TrimSpace(scopeFlag))
			if scopeFlagLower == "machine" || scopeFlagLower == "both" || scopeFlagLower == "all" {
				// Check elevation for machine scope before parsing
				if err := cmdutils.CheckElevation(cmd, fontManager, platform.MachineScope); err != nil {
					if errors.Is(err, cmdutils.ErrElevationRequired) {
						return nil // Error already printed
					}
					return err
				}
			}

			selectedScopes, err = parseScopeFlag(scopeFlag, availableScopes)
			if err != nil {
				return err
			}
		} else {
			// No scope flag = use all accessible scopes
			selectedScopes = availableScopes
		}

		// Validate that scopes were selected
		if len(selectedScopes) == 0 {
			cmdutils.PrintWarning("No scopes selected. Backup cancelled.")
			fmt.Println()
			return nil
		}

		// Use selected scopes
		scopes := selectedScopes

		// Handle overwrite - if file exists and no force flag, show error and return
		if needsOverwriteConfirm && !force {
			cmdutils.PrintErrorf("File already exists: %s", zipPath)
			cmdutils.PrintInfo("Use --force to overwrite")
			fmt.Println()
			return nil
		}

		// Log backup parameters (always log to file)
		GetLogger().Info("Backup parameters - Output: %s, Scopes: %v", zipPath, scopes)

		// Verbose output
		output.GetVerbose().Info("Backing up font files")
		output.GetVerbose().Info("Output: %s", zipPath)
		output.GetVerbose().Info("Scopes: %v", scopes)
		output.GetVerbose().Info("System fonts are excluded")
		// Verbose section ends with blank line per spacing framework (only if verbose was shown)
		if IsVerbose() {
			fmt.Println()
		}

		// For debug mode, do everything without spinner
		if IsDebug() {
			return performBackup(fontManager, scopes, zipPath)
		}

		// Use progress bar for backup operation
		return runBackupWithProgressBar(fontManager, scopes, zipPath)
	},
}

// generateDefaultBackupFilename generates a date-based backup filename
func generateDefaultBackupFilename() string {
	now := time.Now()
	dateStr := now.Format("2006-01-02")
	return fmt.Sprintf("fontget-backup-%s.zip", dateStr)
}

// performBackup performs the backup operation (for debug mode)
func performBackup(fontManager platform.FontManager, scopes []platform.InstallationScope, zipPath string) error {
	output.GetDebug().State("Calling performBackup(scopes=%v, zipPath=%s)", scopes, zipPath)

	fonts, err := collectFonts(scopes, fontManager, "", true) // Suppress verbose for debug mode
	if err != nil {
		output.GetDebug().State("Error collecting fonts for backup: %v", err)
		return err
	}
	output.GetDebug().State("Total fonts to backup: %d", len(fonts))
	output.GetDebug().State("Calling performBackupWithCollectedFonts(scopes=%v, zipPath=%s, fontCount=%d)", scopes, zipPath, len(fonts))
	result, err := performBackupWithCollectedFonts(fontManager, scopes, zipPath, fonts)
	if err != nil {
		return err
	}

	GetLogger().Info("Backup operation complete - Backed up %d font families, %d files to %s", result.familyCount, result.fileCount, zipPath)
	output.GetDebug().State("Backup operation complete - Families: %d, Files: %d", result.familyCount, result.fileCount)
	fmt.Printf("  %s %s\n", ui.SuccessText.Render("✓"), ui.Text.Render(fmt.Sprintf("Font files backed up to: %s", ui.InfoText.Render(fmt.Sprintf("'%s'", zipPath)))))
	fmt.Println()
	return nil
}

// runBackupWithProgressBar runs the backup operation with a progress bar
func runBackupWithProgressBar(fontManager platform.FontManager, scopes []platform.InstallationScope, zipPath string) error {
	// First, collect fonts to determine how many families we'll be backing up
	output.GetVerbose().Info("Scanning fonts to determine backup scope...")
	fonts, err := collectFonts(scopes, fontManager, "", true) // Suppress verbose - we have our own high-level message
	if err != nil {
		return fmt.Errorf("unable to collect fonts: %w", err)
	}
	output.GetDebug().State("Total fonts to backup: %d", len(fonts))

	// Match fonts to repository to get source information and organize by family
	var names []string
	fontMap := make(map[string][]ParsedFont)
	for _, font := range fonts {
		if shared.IsCriticalSystemFont(font.Family) {
			continue
		}
		names = append(names, font.Family)
		fontMap[font.Family] = append(fontMap[font.Family], font)
	}
	sort.Strings(names)

	matches, err := repo.MatchAllInstalledFonts(names, shared.IsCriticalSystemFont)
	if err != nil {
		output.GetVerbose().Warning("Some fonts could not be matched to repository: %v", err)
		if matches == nil {
			matches = make(map[string]*repo.InstalledFontMatch)
		}
	}

	// Count total families to backup
	totalFamilies := len(fontMap)
	if totalFamilies == 0 {
		cmdutils.PrintWarning("No fonts found to backup.")
		fmt.Println()
		return nil
	}

	// Create progress bar title - will show count automatically
	progressTitle := "Backing Up Fonts"

	// Create operation items for count tracking (x of y) but keep them as "pending" so they don't display
	// We'll track the count manually via progress updates
	sortedFamilyNames := make([]string, 0, totalFamilies)
	for familyName := range fontMap {
		sortedFamilyNames = append(sortedFamilyNames, familyName)
	}
	sort.Strings(sortedFamilyNames)

	// Create items with a special status that won't display but allows count tracking
	// Actually, we need items to be "completed" for count, but we don't want them to display
	// Solution: Create items but don't send ItemUpdateMsg - instead track count manually
	operationItems := make([]components.OperationItem, totalFamilies)
	for i := range operationItems {
		// Use empty name and keep as pending - we'll track count via progress percentage
		operationItems[i] = components.OperationItem{
			Name:   "", // Empty name so it won't display
			Status: "pending",
		}
	}

	// Run progress bar with no items (just progress bar, no count, no item list)
	verbose := IsVerbose()
	debug := IsDebug()
	var backupResult *backupResult
	progressErr := components.RunProgressBar(
		progressTitle,
		operationItems,
		verbose, // Verbose mode: show operational details and file/variant listings
		debug,   // Debug mode: show technical details
		func(send func(msg tea.Msg), cancelChan <-chan struct{}) error {
			// Perform the actual backup operation
			var err error
			backupResult, err = performBackupWithProgress(fontManager, zipPath, fontMap, matches, sortedFamilyNames, send, cancelChan)
			return err
		},
	)

	if progressErr != nil {
		// Check if it was a cancellation
		if errors.Is(progressErr, shared.ErrOperationCancelled) {
			// Clean up any temp backup files that may have been created
			cleanupTempBackupFiles(zipPath)
			cmdutils.PrintWarning("Backup cancelled.")
			fmt.Println()
			return nil // Don't return error for cancellation
		}
		// Print error with proper styling (Cobra won't print it since SilenceErrors is true)
		cmdutils.PrintErrorf("%v", progressErr)
		fmt.Println()
		return progressErr
	}

	// Show success message after progress bar completes
	if backupResult != nil {
		// Show destination path with checkmark, matching remove command style
		fmt.Printf("  %s %s\n", ui.SuccessText.Render("✓"), ui.Text.Render(fmt.Sprintf("Font files backed up to: %s", ui.InfoText.Render(fmt.Sprintf("'%s'", zipPath)))))
		fmt.Println()
	}

	return nil
}

// performBackupWithProgress performs the backup operation with progress updates
// fontFileInfo represents a font file with its path and scope
type fontFileInfo struct {
	filePath string
	scope    string
}

// organizeFontsBySourceAndFamily organizes fonts by source and family name for zip structure.
// Returns a map: sourceName -> familyName -> []fontFileInfo
// organizeFontsBySourceAndFamily organizes fonts by source and family name for backup.
//
// It creates a nested map structure: source -> family name -> font files, which is used
// to organize fonts in the backup zip archive. Fonts are matched to repository entries
// to determine their source.
func organizeFontsBySourceAndFamily(fontManager platform.FontManager, fontMap map[string][]ParsedFont, matches map[string]*repo.InstalledFontMatch) map[string]map[string][]fontFileInfo {
	sourceFamilyMap := make(map[string]map[string][]fontFileInfo)
	dedupeMap := make(map[string]bool)

	// Process fonts and organize by source
	for familyName, fontGroup := range fontMap {
		sourceName := DefaultSourceName
		if match, exists := matches[familyName]; exists && match != nil {
			sourceName = match.Source
			if sourceName == "" {
				sourceName = DefaultSourceName
			}
		}

		if sourceFamilyMap[sourceName] == nil {
			sourceFamilyMap[sourceName] = make(map[string][]fontFileInfo)
		}

		for _, font := range fontGroup {
			// Check if we've already added this filename (deduplication)
			if dedupeMap[font.Name] {
				output.GetDebug().State("Skipping duplicate font file: %s (already added)", font.Name)
				continue
			}

			fontDir := fontManager.GetFontDir(platform.InstallationScope(font.Scope))
			filePath := filepath.Join(fontDir, font.Name)

			// Verify file exists
			if _, err := os.Stat(filePath); err != nil {
				output.GetVerbose().Warning("Font file not found: %s", filePath)
				output.GetDebug().Error("Font file not found: %s", filePath)
				continue
			}

			sourceFamilyMap[sourceName][familyName] = append(sourceFamilyMap[sourceName][familyName], fontFileInfo{
				filePath: filePath,
				scope:    font.Scope,
			})

			dedupeMap[font.Name] = true
		}
	}

	return sourceFamilyMap
}

// createBackupZipArchive creates a zip archive from organized font structure.
// If send is not nil, it will be called with progress updates.
// createBackupZipArchive creates a zip archive from organized font files.
//
// It creates a zip archive with fonts organized by source and family name.
// The archive structure is: source/family-name/font-file.ttf
// Progress updates are sent via the send function for TUI display.
// The archive is created in a temp location first, then moved to the final location.
//
// Parameters:
//   - sourceFamilyMap: Nested map of source -> family -> font files
//   - zipPath: Path to the final output zip file
//   - familyIndexMap: Map from family name to operation item index for progress tracking
//   - send: Function to send progress updates (for TUI)
//   - cancelChan: Channel to check for cancellation requests
//
// Returns:
//   - *backupResult: Contains family and file counts
//   - error: Error if archive creation fails (including cancellation)
func createBackupZipArchive(sourceFamilyMap map[string]map[string][]fontFileInfo, zipPath string, familyIndexMap map[string]int, send func(msg tea.Msg), cancelChan <-chan struct{}) (*backupResult, error) {
	// Ensure parent directory exists
	if dir := filepath.Dir(zipPath); dir != "." && dir != zipPath {
		if err := os.MkdirAll(dir, 0755); err != nil {
			GetLogger().Error("Failed to create backup directory: %v", err)
			// Don't print error here - let Cobra handle it when we return
			return nil, fmt.Errorf("unable to create directory for backup archive: %w", err)
		}
	}

	// Create temp file in the same directory as final path (same filesystem = fast move)
	tempDir := filepath.Dir(zipPath)
	if tempDir == "." || tempDir == zipPath {
		// Fallback to system temp if we can't determine directory
		tempDir = os.TempDir()
	}
	tempFile, err := os.CreateTemp(tempDir, "fontget-backup-*.zip.tmp")
	if err != nil {
		GetLogger().Error("Failed to create temp backup zip file: %v", err)
		return nil, fmt.Errorf("unable to create temp backup archive: %w", err)
	}
	tempPath := tempFile.Name()
	tempFile.Close() // Close so we can delete it if needed
	// Defer will always try to clean up temp file and ignore "file not found" errors
	// (which means the file was successfully moved to final location)

	// Create zip file in temp location
	zipFile, err := os.Create(tempPath)
	if err != nil {
		GetLogger().Error("Failed to create backup zip file: %v", err)
		os.Remove(tempPath) // Clean up temp file
		return nil, fmt.Errorf("unable to create backup archive: %w", err)
	}

	// Declare zipWriter here so it's accessible to the defer function
	var zipWriter *zip.Writer

	defer func() {
		// Close resources in proper order: writer first (flushes), then file (releases handle)
		if zipWriter != nil {
			zipWriter.Close()
		}
		if zipFile != nil {
			zipFile.Sync() // Ensure data is flushed
			zipFile.Close()
		}

		// Attempt to clean up temp file with retry logic
		// Small delay to allow OS to release file handle (Windows file locking)
		// This helps prevent "file in use" errors on Windows, especially on cancellation
		if tempPath != "" {
			removeTempFileWithRetry(tempPath)
		}
	}()

	zipWriter = zip.NewWriter(zipFile)

	// Sort sources and families for consistent processing
	sourceNames := make([]string, 0, len(sourceFamilyMap))
	for sourceName := range sourceFamilyMap {
		sourceNames = append(sourceNames, sourceName)
	}
	sort.Strings(sourceNames)

	// Count total files first for accurate progress tracking (if progress callback provided)
	totalFiles := 0
	if send != nil {
		for _, sourceName := range sourceNames {
			familyMap := sourceFamilyMap[sourceName]
			for _, fontFiles := range familyMap {
				totalFiles += len(fontFiles)
			}
		}
	}

	familyCount := 0
	fileCount := 0
	processedFiles := 0

	// Count total files for debug output (even if send is nil)
	totalFilesForDebug := 0
	for _, sourceName := range sourceNames {
		familyMap := sourceFamilyMap[sourceName]
		for _, fontFiles := range familyMap {
			totalFilesForDebug += len(fontFiles)
		}
	}

	// Process each family and add to zip
	for _, sourceName := range sourceNames {
		familyMap := sourceFamilyMap[sourceName]
		familyNames := make([]string, 0, len(familyMap))
		for familyName := range familyMap {
			familyNames = append(familyNames, familyName)
		}
		sort.Strings(familyNames)

		for _, familyName := range familyNames {
			fontFiles := familyMap[familyName]
			familyCount++

			// Debug output: Starting to archive a family
			if IsDebug() {
				output.GetDebug().State("Archiving family '%s' to '%s' (%d files)", familyName, sourceName, len(fontFiles))
			}

			// Sanitize source and family names for zip paths
			sanitizedSource := shared.SanitizeForZipPath(sourceName)
			sanitizedFamily := shared.SanitizeForZipPath(familyName)

			// Add each font file to the zip
			for _, fontInfo := range fontFiles {
				// Check for cancellation before processing each file
				if cancelChan != nil {
					select {
					case <-cancelChan:
						// Cancellation requested - return error so defer can clean up temp file
						return nil, shared.ErrOperationCancelled
					default:
						// Continue processing
					}
				}

				// Create path in zip: {Source}/{FamilyName}/{filename}
				zipEntryPath := filepath.Join(sanitizedSource, sanitizedFamily, filepath.Base(fontInfo.filePath))
				// Normalize path separators for zip (use forward slashes)
				zipEntryPath = strings.ReplaceAll(zipEntryPath, "\\", "/")

				// Open source file
				srcFile, err := os.Open(fontInfo.filePath)
				if err != nil {
					output.GetVerbose().Warning("Failed to open %s: %v", fontInfo.filePath, err)
					output.GetDebug().Error("os.Open() failed for %s: %v", fontInfo.filePath, err)
					continue
				}

				// Create file in zip
				zipEntry, err := zipWriter.Create(zipEntryPath)
				if err != nil {
					srcFile.Close()
					output.GetVerbose().Warning("Failed to create zip entry for %s: %v", zipEntryPath, err)
					output.GetDebug().Error("zipWriter.Create() failed for %s: %v", zipEntryPath, err)
					continue
				}

				// Copy file contents to zip
				_, err = io.Copy(zipEntry, srcFile)
				srcFile.Close()
				if err != nil {
					output.GetVerbose().Warning("Failed to write %s to zip: %v", fontInfo.filePath, err)
					output.GetDebug().Error("io.Copy() failed for %s: %v", fontInfo.filePath, err)
					continue
				}

				fileCount++
				processedFiles++

				// Debug output: Show progress every 10 files or at milestones
				if IsDebug() {
					if processedFiles%10 == 0 || processedFiles == totalFilesForDebug {
						output.GetDebug().State("Archived %d of %d files (%.1f%%)", processedFiles, totalFilesForDebug, float64(processedFiles)/float64(totalFilesForDebug)*100)
					}
				}

				// Update progress after each file for smooth progress bar (if callback provided)
				if send != nil && totalFiles > 0 {
					percent := float64(processedFiles) / float64(totalFiles) * 100
					send(components.ProgressUpdateMsg{Percent: percent})
				}
			}

			// Debug output: Completed archiving a family
			if IsDebug() {
				output.GetDebug().State("Completed archiving family '%s' (%d files)", familyName, len(fontFiles))
			}

			// Mark family as completed after processing all its files
			// Use empty name so item doesn't display, but status update allows count to increment
			if send != nil && familyIndexMap != nil {
				if index, exists := familyIndexMap[familyName]; exists {
					send(components.ItemUpdateMsg{
						Index:  index,
						Name:   "", // Keep empty so it doesn't display
						Status: "completed",
					})
				}
			}
		}
	}

	// Close zip writer to finalize the archive
	// Note: We close it here for normal completion, but the defer will also close it
	// if we return early (cancellation/error), so we need to track if it's already closed
	if err := zipWriter.Close(); err != nil {
		GetLogger().Error("Failed to close zip writer: %v", err)
		// Don't print error here - let Cobra handle it when we return
		return nil, fmt.Errorf("unable to finalize backup archive: %w", err)
	}
	zipWriter = nil // Mark as closed so defer doesn't try to close it again

	// Debug output: Starting to move temp file to final location
	if IsDebug() {
		output.GetDebug().State("Moving archive from temp location to final location: %s", zipPath)
	}

	// Move temp file to final location
	if err := os.Rename(tempPath, zipPath); err != nil {
		GetLogger().Error("Failed to move temp backup to final location: %v", err)
		// Try to copy if rename fails (different filesystem)
		if IsDebug() {
			output.GetDebug().State("Rename failed, attempting copy instead (different filesystem)")
		}
		if copyErr := copyFile(tempPath, zipPath); copyErr != nil {
			// Both rename and copy failed - temp file will be cleaned up by defer
			return nil, fmt.Errorf("unable to move backup archive to final location: %w (copy also failed: %w)", err, copyErr)
		}
		// Copy succeeded - temp file will be cleaned up by defer
		if IsDebug() {
			output.GetDebug().State("Archive copied successfully to: %s", zipPath)
		}
	} else {
		// Rename succeeded - file should be moved, but defer will clean up if it still exists
		// (This handles edge cases on Windows where rename reports success but file isn't moved)
		if IsDebug() {
			output.GetDebug().State("Archive moved successfully to: %s", zipPath)
		}
	}

	output.GetVerbose().Info("Backup archive created: %d font families, %d files", familyCount, fileCount)
	output.GetDebug().State("Backup operation complete - Families: %d, Files: %d", familyCount, fileCount)

	return &backupResult{
		familyCount: familyCount,
		fileCount:   fileCount,
	}, nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func performBackupWithProgress(fontManager platform.FontManager, zipPath string, fontMap map[string][]ParsedFont, matches map[string]*repo.InstalledFontMatch, sortedFamilyNames []string, send func(msg tea.Msg), cancelChan <-chan struct{}) (*backupResult, error) {
	// Organize fonts by source -> family name
	sourceFamilyMap := organizeFontsBySourceAndFamily(fontManager, fontMap, matches)

	// Create a map from family name to index for progress tracking
	familyIndexMap := make(map[string]int)
	for i, familyName := range sortedFamilyNames {
		familyIndexMap[familyName] = i
	}

	// Create zip archive with progress tracking and cancellation support
	return createBackupZipArchive(sourceFamilyMap, zipPath, familyIndexMap, send, cancelChan)
}

// removeTempFileWithRetry attempts to remove a temp file with retry logic
// This handles Windows file locking issues where files may remain locked briefly after closing
func removeTempFileWithRetry(tempPath string) {
	const maxRetries = 5
	const initialDelay = 100 * time.Millisecond

	for i := 0; i < maxRetries; i++ {
		// Delay before attempting removal (longer delay on first attempt for active writes)
		if i > 0 {
			time.Sleep(initialDelay * time.Duration(i))
		} else {
			time.Sleep(initialDelay)
		}

		err := os.Remove(tempPath)
		if err == nil {
			GetLogger().Debug("Cleaned up temp backup file: %s", tempPath)
			return
		}
		if os.IsNotExist(err) {
			// File already gone - success
			return
		}
		// File still locked - will retry
		if i < maxRetries-1 {
			GetLogger().Debug("Temp file still locked, retrying cleanup: %s (attempt %d/%d)", tempPath, i+1, maxRetries)
		}
	}

	// All retries failed - log but don't fail (temp files will be cleaned up eventually)
	GetLogger().Debug("Could not remove temp backup file after %d retries (may be cleaned up later): %s", maxRetries, tempPath)
}

// cleanupTempBackupFiles removes any temporary backup files that may have been created
// This is called explicitly on cancellation to ensure cleanup happens
func cleanupTempBackupFiles(zipPath string) {
	// Determine the temp directory where backup files would be created
	tempDir := filepath.Dir(zipPath)
	if tempDir == "." || tempDir == zipPath {
		tempDir = os.TempDir()
	}

	// Find all temp backup files matching our pattern
	pattern := filepath.Join(tempDir, "fontget-backup-*.zip.tmp")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		GetLogger().Debug("Could not search for temp backup files: %v", err)
		return
	}

	// Remove each temp file found with retry logic
	for _, match := range matches {
		removeTempFileWithRetry(match)
	}
}

// performBackupWithCollectedFonts performs the backup operation with pre-collected fonts (for debug mode)
func performBackupWithCollectedFonts(fontManager platform.FontManager, _ []platform.InstallationScope, zipPath string, fonts []ParsedFont) (*backupResult, error) {
	output.GetDebug().State("Calling performBackupWithCollectedFonts(zipPath=%s, fontCount=%d)", zipPath, len(fonts))

	output.GetVerbose().Info("Found %d font files", len(fonts))
	output.GetDebug().State("Processing %d font files", len(fonts))

	// Match fonts to repository to get source information
	var names []string
	fontMap := make(map[string][]ParsedFont)
	for _, font := range fonts {
		// Skip system fonts
		if shared.IsCriticalSystemFont(font.Family) {
			continue
		}
		names = append(names, font.Family)
		fontMap[font.Family] = append(fontMap[font.Family], font)
	}
	sort.Strings(names)

	output.GetVerbose().Info("Matching fonts to repository...")
	output.GetDebug().State("Calling repo.MatchAllInstalledFonts(familyCount=%d)", len(names))
	matches, err := repo.MatchAllInstalledFonts(names, shared.IsCriticalSystemFont)
	if err != nil {
		output.GetVerbose().Warning("Some fonts could not be matched to repository: %v", err)
		output.GetDebug().Error("repo.MatchAllInstalledFonts() failed: %v", err)
		// Continue with partial matches
		if matches == nil {
			matches = make(map[string]*repo.InstalledFontMatch)
		}
	}

	// Organize fonts by source -> family name
	sourceFamilyMap := organizeFontsBySourceAndFamily(fontManager, fontMap, matches)

	output.GetVerbose().Info("Creating zip archive...")
	output.GetDebug().State("Organized fonts into %d sources", len(sourceFamilyMap))

	// Create zip archive without progress tracking (debug mode)
	return createBackupZipArchive(sourceFamilyMap, zipPath, make(map[string]int), nil, nil)
}

// parseScopeFlag parses the --scope flag value and returns the corresponding scopes
func parseScopeFlag(scopeFlag string, availableScopes []platform.InstallationScope) ([]platform.InstallationScope, error) {
	scopeFlag = strings.ToLower(strings.TrimSpace(scopeFlag))

	var requestedScopes []platform.InstallationScope
	switch scopeFlag {
	case "user":
		requestedScopes = []platform.InstallationScope{platform.UserScope}
	case "machine":
		requestedScopes = []platform.InstallationScope{platform.MachineScope}
	case "both", "all":
		requestedScopes = []platform.InstallationScope{platform.UserScope, platform.MachineScope}
	default:
		return nil, fmt.Errorf("invalid scope value: %s (must be 'user', 'machine', or 'both')", scopeFlag)
	}

	// Filter to only include accessible scopes
	var selectedScopes []platform.InstallationScope
	for _, requested := range requestedScopes {
		for _, available := range availableScopes {
			if requested == available {
				selectedScopes = append(selectedScopes, requested)
				break
			}
		}
	}

	if len(selectedScopes) == 0 {
		return nil, fmt.Errorf("none of the requested scopes are accessible")
	}

	return selectedScopes, nil
}

func init() {
	backupCmd.Flags().BoolP("force", "f", false, "Force overwrite existing archive without confirmation")
	backupCmd.Flags().StringP("scope", "s", "", "Scope to backup: 'user', 'machine', or 'both' (default: all accessible)")
	rootCmd.AddCommand(backupCmd)
}
