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

This command creates a backup zip archive containing all font files installed on your system.
Fonts are organized by source (e.g., Google Fonts, Nerd Fonts) and then by family name.

The command automatically detects which scopes are accessible:
  - If running as regular user: backs up user-scope fonts only
  - If running as administrator/sudo: backs up both user and machine-scope fonts

Fonts are deduplicated across scopes - if the same font exists in both scopes,
only one copy is included in the backup.

System fonts are always excluded from backups.

When an output path is provided, the TUI is bypassed and all accessible scopes are used.
Use --scope to specify which scopes to backup when providing a path.`,
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

		fm, err := cmdutils.CreateFontManager(func() cmdutils.Logger { return GetLogger() })
		if err != nil {
			return err
		}

		// Get output path from args or use default
		var outputPath string
		if len(args) > 0 {
			outputPath = args[0]
		}

		// Validate output path (without prompting for overwrite yet)
		zipPath, needsOverwriteConfirm, err := validateOutputPathForBackup(outputPath)
		if err != nil {
			return err
		}

		// If force flag is set, skip overwrite confirmation
		if force {
			needsOverwriteConfirm = false
		}

		// Auto-detect accessible scopes
		availableScopes, err := detectAccessibleScopes(fm)
		if err != nil {
			GetLogger().Error("Failed to detect accessible scopes: %v", err)
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("detectAccessibleScopes() failed: %v", err)
			return fmt.Errorf("unable to detect accessible font scopes: %v", err)
		}

		// Determine selected scopes based on flags and path
		var selectedScopes []platform.InstallationScope
		var confirmed bool
		pathProvided := outputPath != ""

		// Parse scope flag if provided
		if scopeFlag != "" {
			// Check if machine scope is requested (requires elevation)
			scopeFlagLower := strings.ToLower(strings.TrimSpace(scopeFlag))
			if scopeFlagLower == "machine" || scopeFlagLower == "both" || scopeFlagLower == "all" {
				// Check elevation for machine scope before parsing
				if err := cmdutils.CheckElevation(cmd, fm, platform.MachineScope); err != nil {
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
			confirmed = true // Scope flag implies confirmation
		} else if pathProvided {
			// Path provided (with or without force) = use all accessible scopes, skip TUI
			selectedScopes = availableScopes
			confirmed = true
		} else {
			// No path provided = show TUI modal
			title := "Select Backup Scope"
			message := "Choose which font scopes to backup:"
			selectedScopes, confirmed, err = runBackupModal(title, message, availableScopes, zipPath, needsOverwriteConfirm)
			if err != nil {
				return fmt.Errorf("unable to show backup modal: %v", err)
			}
		}

		if !confirmed {
			fmt.Printf("%s\n", ui.WarningText.Render("Backup cancelled."))
			fmt.Println()
			return nil
		}

		// Validate that scopes were selected
		if len(selectedScopes) == 0 {
			fmt.Printf("%s\n", ui.WarningText.Render("No scopes selected. Backup cancelled."))
			fmt.Println()
			return nil
		}

		// Handle overwrite confirmation if needed (and not using TUI)
		if pathProvided && needsOverwriteConfirm && !force {
			confirmed, err := components.RunConfirm(
				"File Already Exists",
				fmt.Sprintf("File already exists. Overwrite '%s'?", ui.SecondaryText.Render(filepath.Base(zipPath))),
			)
			if err != nil {
				return fmt.Errorf("unable to show confirmation dialog: %v", err)
			}
			if !confirmed {
				fmt.Printf("%s\n", ui.WarningText.Render("Backup cancelled."))
				fmt.Println()
				return nil
			}
		}

		// Log backup parameters (always log to file)
		GetLogger().Info("Backup parameters - Output: %s, Scopes: %v", zipPath, selectedScopes)

		// Verbose output
		output.GetVerbose().Info("Backing up font files")
		output.GetVerbose().Info("Output: %s", zipPath)
		output.GetVerbose().Info("Scopes: %v", selectedScopes)
		output.GetVerbose().Info("System fonts are excluded")
		// Verbose section ends with blank line per spacing framework (only if verbose was shown)
		if IsVerbose() {
			fmt.Println()
		}

		// For debug mode, do everything without spinner
		if IsDebug() {
			return performBackup(fm, selectedScopes, zipPath)
		}

		// Use progress bar for backup operation
		return runBackupWithProgressBar(fm, selectedScopes, zipPath)
	},
}

// generateDefaultBackupFilename generates a date-based backup filename
func generateDefaultBackupFilename() string {
	now := time.Now()
	dateStr := now.Format("2006-01-02")
	return fmt.Sprintf("font-backup-%s.zip", dateStr)
}

// validateOutputPathForBackup validates and normalizes the output path without prompting
// Returns: (normalizedPath, needsOverwriteConfirm, error)
func validateOutputPathForBackup(outputPath string) (string, bool, error) {
	// If no path provided, use date-based default name in current directory
	if outputPath == "" {
		outputPath = generateDefaultBackupFilename()
	}

	// Normalize path separators
	outputPath = filepath.Clean(outputPath)

	// Check if it's a directory (ends with separator or exists as directory)
	info, err := os.Stat(outputPath)
	if err == nil && info.IsDir() {
		// It's a directory, use date-based default filename
		outputPath = filepath.Join(outputPath, generateDefaultBackupFilename())
	} else if err == nil && !info.IsDir() {
		// Path exists and is a file - check if it has .zip extension
		if !strings.HasSuffix(strings.ToLower(outputPath), ".zip") {
			return "", false, fmt.Errorf("output path exists and is not a zip file: %s", outputPath)
		}
		// File exists - will check and prompt later after getting absolute path
	} else if os.IsNotExist(err) {
		// Path doesn't exist - check if parent directory exists
		parentDir := filepath.Dir(outputPath)
		if parentDir != "." && parentDir != outputPath {
			parentInfo, err := os.Stat(parentDir)
			if err != nil {
				// Parent doesn't exist - check if we can create it
				// For safety, only allow creating one level deep
				if !strings.HasSuffix(strings.ToLower(outputPath), ".zip") {
					return "", false, fmt.Errorf("output path must be a .zip file: %s", outputPath)
				}
				// Will create parent directory later
			} else if !parentInfo.IsDir() {
				return "", false, fmt.Errorf("parent path exists but is not a directory: %s", parentDir)
			}
		}

		// Ensure it has .zip extension
		if !strings.HasSuffix(strings.ToLower(outputPath), ".zip") {
			outputPath = outputPath + ".zip"
		}
	}

	// Final validation: ensure it's an absolute or relative path (not just a filename in a non-existent deep path)
	absPath, err := filepath.Abs(outputPath)
	if err != nil {
		return "", false, fmt.Errorf("invalid output path: %v", err)
	}

	// Guard rail: prevent writing to system directories
	systemDirs := []string{
		filepath.Join(os.Getenv("SystemRoot"), "Fonts"),
		"/System/Library/Fonts",
		"/usr/share/fonts",
		"/usr/local/share/fonts",
	}
	for _, sysDir := range systemDirs {
		if sysDir != "" && strings.HasPrefix(strings.ToLower(absPath), strings.ToLower(sysDir)) {
			return "", false, fmt.Errorf("cannot write backup to system font directory: %s", absPath)
		}
	}

	// Check if the final file path already exists
	needsConfirm := false
	if _, err := os.Stat(absPath); err == nil {
		needsConfirm = true
	}

	return absPath, needsConfirm, nil
}

// detectAccessibleScopes detects which font scopes are accessible based on elevation
func detectAccessibleScopes(fm platform.FontManager) ([]platform.InstallationScope, error) {
	isElevated, err := fm.IsElevated()
	if err != nil {
		// If we can't detect elevation, default to user scope only (safer)
		output.GetVerbose().Warning("Unable to detect elevation status: %v. Backing up user scope only.", err)
		return []platform.InstallationScope{platform.UserScope}, nil
	}

	if isElevated {
		// Admin/sudo - can access both scopes
		return []platform.InstallationScope{platform.UserScope, platform.MachineScope}, nil
	}

	// Regular user - only user scope
	return []platform.InstallationScope{platform.UserScope}, nil
}

// performBackup performs the backup operation (for debug mode)
func performBackup(fm platform.FontManager, scopes []platform.InstallationScope, zipPath string) error {
	output.GetDebug().State("Calling performBackup(scopes=%v, zipPath=%s)", scopes, zipPath)

	fonts, err := collectFonts(scopes, fm, "", true) // Suppress verbose for debug mode
	if err != nil {
		output.GetDebug().State("Error collecting fonts for backup: %v", err)
		return err
	}
	output.GetDebug().State("Total fonts to backup: %d", len(fonts))
	output.GetDebug().State("Calling performBackupWithCollectedFonts(scopes=%v, zipPath=%s, fontCount=%d)", scopes, zipPath, len(fonts))
	result, err := performBackupWithCollectedFonts(fm, scopes, zipPath, fonts)
	if err != nil {
		return err
	}

	GetLogger().Info("Backup operation complete - Backed up %d font families, %d files to %s", result.familyCount, result.fileCount, zipPath)
	output.GetDebug().State("Backup operation complete - Families: %d, Files: %d", result.familyCount, result.fileCount)
	fmt.Printf("%s\n", ui.SuccessText.Render(fmt.Sprintf("Successfully backed up %d font families to %s", result.familyCount, zipPath)))
	fmt.Println()
	return nil
}

// runBackupWithProgressBar runs the backup operation with a progress bar
func runBackupWithProgressBar(fm platform.FontManager, scopes []platform.InstallationScope, zipPath string) error {
	// First, collect fonts to determine how many families we'll be backing up
	output.GetVerbose().Info("Scanning fonts to determine backup scope...")
	fonts, err := collectFonts(scopes, fm, "", true) // Suppress verbose - we have our own high-level message
	if err != nil {
		return fmt.Errorf("unable to collect fonts: %v", err)
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
		fmt.Printf("%s\n", ui.WarningText.Render("No fonts found to backup."))
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
			backupResult, err = performBackupWithProgress(fm, scopes, zipPath, fonts, fontMap, matches, sortedFamilyNames, send, cancelChan)
			return err
		},
	)

	if progressErr != nil {
		// Check if it was a cancellation
		if errors.Is(progressErr, shared.ErrOperationCancelled) {
			// Clean up any temp backup files that may have been created
			cleanupTempBackupFiles(zipPath)
			fmt.Printf("%s\n", ui.WarningText.Render("Backup cancelled."))
			fmt.Println()
			return nil // Don't return error for cancellation
		}
		// Print error with proper styling (Cobra won't print it since SilenceErrors is true)
		fmt.Printf("%s\n", ui.ErrorText.Render(fmt.Sprintf("Error: %v", progressErr)))
		fmt.Println()
		return progressErr
	}

	// Show success message after progress bar completes
	if backupResult != nil {
		// Show destination path with InfoText styling
		fmt.Printf("Font files backed up to: %s\n", ui.InfoText.Render(fmt.Sprintf("'%s'", zipPath)))
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

// organizeFontsBySourceAndFamily organizes fonts by source and family name for backup.
//
// It creates a nested map structure: source -> family name -> font files, which is used
// to organize fonts in the backup zip archive. Fonts are matched to repository entries
// to determine their source.
//
// Returns a map: sourceName -> familyName -> []fontFileInfo
func organizeFontsBySourceAndFamily(fm platform.FontManager, fontMap map[string][]ParsedFont, matches map[string]*repo.InstalledFontMatch) map[string]map[string][]fontFileInfo {
	sourceFamilyMap := make(map[string]map[string][]fontFileInfo)
	dedupeMap := make(map[string]bool)

	// Process fonts and organize by source
	for familyName, fontGroup := range fontMap {
		sourceName := "Other"
		if match, exists := matches[familyName]; exists && match != nil {
			sourceName = match.Source
			if sourceName == "" {
				sourceName = "Other"
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

			fontDir := fm.GetFontDir(platform.InstallationScope(font.Scope))
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
//   - send: Function to send progress updates (for TUI). If nil, no progress updates are sent.
//   - cancelChan: Channel to check for cancellation requests. If nil, cancellation is not checked.
//
// Returns:
//   - *backupResult: Contains family and file counts
//   - error: Error if archive creation fails (including cancellation via shared.ErrOperationCancelled)
func createBackupZipArchive(sourceFamilyMap map[string]map[string][]fontFileInfo, zipPath string, familyIndexMap map[string]int, send func(msg tea.Msg), cancelChan <-chan struct{}) (*backupResult, error) {
	// Ensure parent directory exists
	if dir := filepath.Dir(zipPath); dir != "." && dir != zipPath {
		if err := os.MkdirAll(dir, 0755); err != nil {
			GetLogger().Error("Failed to create backup directory: %v", err)
			// Don't print error here - let Cobra handle it when we return
			return nil, fmt.Errorf("unable to create directory for backup archive: %v", err)
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
		return nil, fmt.Errorf("unable to create temp backup archive: %v", err)
	}
	tempPath := tempFile.Name()
	tempFile.Close() // Close so we can delete it if needed
	// Note: We no longer track tempFileMoved - defer will always try to clean up
	// and ignore "file not found" errors (which means the file was successfully moved)

	// Create zip file in temp location
	zipFile, err := os.Create(tempPath)
	if err != nil {
		GetLogger().Error("Failed to create backup zip file: %v", err)
		os.Remove(tempPath) // Clean up temp file
		return nil, fmt.Errorf("unable to create backup archive: %v", err)
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

		// Attempt to clean up temp file
		// If deletion fails (e.g., Windows file handle not yet released), that's acceptable:
		// - Temp files in temp directories are cleaned up by the OS periodically
		// - We'll attempt cleanup on next run if needed
		// - Logging the failure is sufficient for debugging
		if tempPath != "" {
			if err := os.Remove(tempPath); err != nil {
				if !os.IsNotExist(err) {
					// File still exists and couldn't be removed - log for debugging
					// This is non-fatal; temp files will be cleaned up eventually
					GetLogger().Debug("Could not remove temp backup file (may be cleaned up later): %s: %v", tempPath, err)
				}
			} else {
				GetLogger().Debug("Cleaned up temp backup file: %s", tempPath)
			}
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
		return nil, fmt.Errorf("unable to finalize backup archive: %v", err)
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
			return nil, fmt.Errorf("unable to move backup archive to final location: %v (copy also failed: %v)", err, copyErr)
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

// performBackupWithProgress performs the backup operation with progress updates.
// The scopes and fonts parameters are unused but kept for interface consistency with other backup functions.
func performBackupWithProgress(fm platform.FontManager, _ []platform.InstallationScope, zipPath string, _ []ParsedFont, fontMap map[string][]ParsedFont, matches map[string]*repo.InstalledFontMatch, sortedFamilyNames []string, send func(msg tea.Msg), cancelChan <-chan struct{}) (*backupResult, error) {
	// Organize fonts by source -> family name
	sourceFamilyMap := organizeFontsBySourceAndFamily(fm, fontMap, matches)

	// Create a map from family name to index for progress tracking
	familyIndexMap := make(map[string]int)
	for i, familyName := range sortedFamilyNames {
		familyIndexMap[familyName] = i
	}

	// Create zip archive with progress tracking and cancellation support
	return createBackupZipArchive(sourceFamilyMap, zipPath, familyIndexMap, send, cancelChan)
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

	// Remove each temp file found
	// On Windows, file handles may not be released immediately after Close(),
	// so we wait a bit first, then retry with delays
	for _, match := range matches {
		// Give Windows time to release the file handle (defer should have closed it by now)
		time.Sleep(100 * time.Millisecond)

		// Try to remove the file, with retries for Windows file handle release
		var removeErr error
		for i := 0; i < 5; i++ {
			removeErr = os.Remove(match)
			if removeErr == nil || os.IsNotExist(removeErr) {
				// Successfully removed or file doesn't exist
				if removeErr == nil {
					GetLogger().Debug("Cleaned up temp backup file: %s", match)
				}
				break
			}
			// Wait a bit before retrying (Windows file handle release delay)
			if i < 4 {
				time.Sleep(100 * time.Millisecond)
			}
		}

		if removeErr != nil && !os.IsNotExist(removeErr) {
			GetLogger().Debug("Could not remove temp backup file %s after retries: %v", match, removeErr)
		}
	}
}

// performBackupWithCollectedFonts performs the backup operation with pre-collected fonts (for debug mode).
// The scopes parameter is unused but kept for interface consistency with other backup functions.
func performBackupWithCollectedFonts(fm platform.FontManager, _ []platform.InstallationScope, zipPath string, fonts []ParsedFont) (*backupResult, error) {
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
	sourceFamilyMap := organizeFontsBySourceAndFamily(fm, fontMap, matches)

	output.GetVerbose().Info("Creating zip archive...")
	output.GetDebug().State("Organized fonts into %d sources", len(sourceFamilyMap))

	// Create zip archive without progress tracking (debug mode)
	return createBackupZipArchive(sourceFamilyMap, zipPath, make(map[string]int), nil, nil)
}

// backupScopeSelectorModel handles scope selection for backup (TUI logic lives in cmd, not components)
type backupScopeSelectorModel struct {
	Title           string
	Message         string
	AvailableScopes []platform.InstallationScope
	SelectedScopes  []platform.InstallationScope
	CheckboxList    *components.CheckboxList
	Buttons         *components.ButtonGroup
	Navigation      *components.FormNavigation
	Quit            bool
	Confirmed       bool
	Width           int
	Height          int
}

func newBackupScopeSelectorModel(title, message string, availableScopes []platform.InstallationScope) *backupScopeSelectorModel {
	// Build checkbox items from available scopes
	items := make([]components.CheckboxItem, len(availableScopes))
	for i, scope := range availableScopes {
		var label string
		switch scope {
		case platform.UserScope:
			label = "User Scope"
		case platform.MachineScope:
			label = "Machine Scope"
		default:
			label = string(scope)
		}
		items[i] = components.CheckboxItem{
			Label:   label,
			Checked: true, // Default to all checked
			Enabled: true,
		}
	}

	checkboxList := &components.CheckboxList{
		Items:    items,
		Cursor:   0,
		HasFocus: true,
	}

	buttons := components.NewButtonGroup([]string{"Backup", "Cancel"}, 0)
	buttons.SetFocus(false)

	// Create FormNavigation
	nav := components.NewFormNavigation(len(items), buttons)
	nav.ListHasFocus = func() bool { return checkboxList.HasFocus }
	nav.ListSetFocus = func(focused bool) { checkboxList.HasFocus = focused }
	nav.ListGetCursor = func() int { return checkboxList.Cursor }
	nav.ListSetCursor = func(cursor int) { checkboxList.Cursor = cursor }

	model := &backupScopeSelectorModel{
		Title:           title,
		Message:         message,
		AvailableScopes: availableScopes,
		SelectedScopes:  availableScopes, // Default to all selected
		CheckboxList:    checkboxList,
		Buttons:         buttons,
		Navigation:      nav,
		Quit:            false,
		Confirmed:       false,
	}

	// Ensure selected scopes are synced with checkbox states
	model.updateSelectedScopes()

	return model
}

func (m *backupScopeSelectorModel) Init() tea.Cmd {
	return nil
}

func (m *backupScopeSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		// Sync navigation cursor with checkbox list cursor
		m.Navigation.ListCursor = m.CheckboxList.Cursor
		m.Navigation.ListFocused = m.CheckboxList.HasFocus

		// Use FormNavigation to handle navigation
		handled, action, listAction := m.Navigation.HandleKey(key)

		if handled {
			// Sync back after navigation
			m.CheckboxList.Cursor = m.Navigation.ListCursor
			m.CheckboxList.HasFocus = m.Navigation.ListFocused

			// Handle list-specific actions
			if listAction == "toggle" {
				if m.CheckboxList.Cursor >= 0 && m.CheckboxList.Cursor < len(m.CheckboxList.Items) {
					m.CheckboxList.Items[m.CheckboxList.Cursor].Checked = !m.CheckboxList.Items[m.CheckboxList.Cursor].Checked
					m.updateSelectedScopes()
				}
			}

			// Handle button actions (FormNavigation returns the button action)
			if action != "" {
				switch strings.ToLower(action) {
				case "backup":
					// Update selected scopes before confirming
					m.updateSelectedScopes()
					m.Confirmed = true
					m.Quit = true
					return m, nil // Don't quit here - let parent modal handle it
				case "cancel":
					m.Confirmed = false
					m.Quit = true
					return m, nil // Don't quit here - let parent modal handle it
				}
			}

			return m, nil
		}

		// Fallback: handle enter on buttons even if FormNavigation didn't handle it
		// If buttons are visible and enter is pressed, activate the selected button
		if key == "enter" && m.Buttons != nil {
			// Give buttons focus temporarily to get the action
			hadFocus := m.Buttons.HasFocus
			m.Buttons.SetFocus(true)
			buttonAction := m.Buttons.HandleKey(key)
			if !hadFocus {
				m.Buttons.SetFocus(false)
			}

			if buttonAction != "" {
				switch strings.ToLower(buttonAction) {
				case "backup":
					m.updateSelectedScopes()
					m.Confirmed = true
					m.Quit = true
					return m, nil
				case "cancel":
					m.Confirmed = false
					m.Quit = true
					return m, nil
				}
			}
		}

		// Handle escape to cancel
		if key == "esc" {
			m.Confirmed = false
			m.Quit = true
			return m, nil // Don't quit here - let parent modal handle it
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil
	}

	return m, nil
}

func (m *backupScopeSelectorModel) updateSelectedScopes() {
	m.SelectedScopes = []platform.InstallationScope{}
	for i, item := range m.CheckboxList.Items {
		if item.Checked && i < len(m.AvailableScopes) {
			m.SelectedScopes = append(m.SelectedScopes, m.AvailableScopes[i])
		}
	}
}

func (m *backupScopeSelectorModel) View() string {
	var result strings.Builder

	// Title
	if m.Title != "" {
		result.WriteString(ui.PageTitle.Render(m.Title))
		result.WriteString("\n\n")
	}

	// Message
	if m.Message != "" {
		result.WriteString(ui.Text.Render(m.Message))
		result.WriteString("\n\n")
	}

	// Render checkbox list
	if m.CheckboxList != nil {
		result.WriteString(m.CheckboxList.Render())
		result.WriteString("\n\n")
	}

	// Render buttons
	if m.Buttons != nil {
		result.WriteString(m.Buttons.Render())
		result.WriteString("\n")
	}

	// Keyboard help
	commands := []string{
		ui.RenderKeyWithDescription("Tab", "Switch focus"),
		ui.RenderKeyWithDescription("↑/↓", "Navigate"),
		ui.RenderKeyWithDescription("Space", "Toggle"),
		ui.RenderKeyWithDescription("Enter", "Select"),
		ui.RenderKeyWithDescription("Esc", "Cancel"),
	}
	helpText := strings.Join(commands, "  ")
	result.WriteString("\n")
	result.WriteString(helpText)

	return result.String()
}

// backupModalModel manages the backup flow: scope selection + optional overwrite confirmation
type backupModalModel struct {
	ScopeSelector    *backupScopeSelectorModel
	OverwriteConfirm *components.ConfirmModel
	State            string // "scope_selection", "overwrite_confirm", "done"
	AvailableScopes  []platform.InstallationScope
	ZipPath          string
	NeedsOverwrite   bool
	SelectedScopes   []platform.InstallationScope
	Confirmed        bool
	Cancelled        bool
	Width            int
	Height           int
}

func newBackupModalModel(title, message string, availableScopes []platform.InstallationScope, zipPath string, needsOverwrite bool) *backupModalModel {
	scopeSelector := newBackupScopeSelectorModel(title, message, availableScopes)

	return &backupModalModel{
		ScopeSelector:   scopeSelector,
		State:           "scope_selection",
		AvailableScopes: availableScopes,
		ZipPath:         zipPath,
		NeedsOverwrite:  needsOverwrite,
	}
}

func (m *backupModalModel) Init() tea.Cmd {
	if m.ScopeSelector != nil {
		return m.ScopeSelector.Init()
	}
	return nil
}

func (m *backupModalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		// Pass to current modal
		if m.State == "scope_selection" && m.ScopeSelector != nil {
			updated, cmd := m.ScopeSelector.Update(msg)
			m.ScopeSelector = updated.(*backupScopeSelectorModel)
			return m, cmd
		} else if m.State == "overwrite_confirm" && m.OverwriteConfirm != nil {
			updated, cmd := m.OverwriteConfirm.Update(msg)
			m.OverwriteConfirm = updated.(*components.ConfirmModel)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		// Handle escape to cancel
		if msg.String() == "esc" {
			if m.State == "overwrite_confirm" {
				// Cancel overwrite, go back to scope selection
				m.State = "scope_selection"
				m.OverwriteConfirm = nil
				return m, nil
			} else {
				// Cancel everything
				m.Cancelled = true
				m.Confirmed = false
				return m, tea.Quit
			}
		}
	}

	// Route messages to current state
	switch m.State {
	case "scope_selection":
		if m.ScopeSelector != nil {
			updated, cmd := m.ScopeSelector.Update(msg)
			m.ScopeSelector = updated.(*backupScopeSelectorModel)

			// Check if scope selector wants to proceed (but don't let it quit yet)
			// This must be checked immediately after update
			if m.ScopeSelector.Quit {
				if m.ScopeSelector.Confirmed {
					// Ensure selected scopes are up to date
					m.ScopeSelector.updateSelectedScopes()

					// Scope selected - check if we need overwrite confirmation
					m.SelectedScopes = m.ScopeSelector.SelectedScopes

					// Don't let scope selector quit - keep it visible as background
					m.ScopeSelector.Quit = false

					if m.NeedsOverwrite {
						// Show overwrite confirmation as nested overlay
						m.OverwriteConfirm = components.NewConfirmModel(
							"File Already Exists",
							fmt.Sprintf("File already exists. Overwrite '%s'?", ui.SecondaryText.Render(filepath.Base(m.ZipPath))),
						)
						m.State = "overwrite_confirm"
						return m, m.OverwriteConfirm.Init()
					} else {
						// No overwrite needed - we're done
						m.Confirmed = true
						m.State = "done"
						return m, tea.Quit
					}
				} else {
					// Scope selection cancelled
					m.Cancelled = true
					m.Confirmed = false
					return m, tea.Quit
				}
			}
			return m, cmd
		}

	case "overwrite_confirm":
		if m.OverwriteConfirm != nil {
			updated, cmd := m.OverwriteConfirm.Update(msg)
			m.OverwriteConfirm = updated.(*components.ConfirmModel)

			// Check if overwrite confirmation is done
			if m.OverwriteConfirm.Quit {
				if m.OverwriteConfirm.Confirmed {
					// Overwrite confirmed - we're done
					m.Confirmed = true
					m.State = "done"
					return m, tea.Quit
				} else {
					// Overwrite cancelled - go back to scope selection
					m.State = "scope_selection"
					m.OverwriteConfirm = nil
					return m, nil
				}
			}
			return m, cmd
		}
	}

	return m, nil
}

func (m *backupModalModel) View() string {
	switch m.State {
	case "scope_selection":
		if m.ScopeSelector != nil {
			// Render scope selector with overlay
			background := &components.BlankBackgroundModel{
				Width:  m.Width,
				Height: m.Height,
			}
			options := components.OverlayOptions{
				ShowBorder:  true,
				BorderWidth: 0,
			}
			overlay := components.NewOverlayWithOptions(m.ScopeSelector, background, components.Center, components.Center, 0, 0, options)
			return overlay.View()
		}

	case "overwrite_confirm":
		if m.OverwriteConfirm != nil && m.ScopeSelector != nil {
			// Render overwrite confirmation nested on top of scope selector
			// First render the scope selector as background
			scopeBackground := &components.BlankBackgroundModel{
				Width:  m.Width,
				Height: m.Height,
			}
			scopeOptions := components.OverlayOptions{
				ShowBorder:  true,
				BorderWidth: 0,
			}
			scopeOverlay := components.NewOverlayWithOptions(m.ScopeSelector, scopeBackground, components.Center, components.Center, 0, 0, scopeOptions)
			scopeView := scopeOverlay.View()

			// Then overlay the confirmation on top
			confirmOptions := components.OverlayOptions{
				ShowBorder:  true,
				BorderWidth: 0,
			}
			// Create a model that renders the scope view as background
			scopeBackgroundModel := &staticBackgroundModel{content: scopeView}
			confirmOverlay := components.NewOverlayWithOptions(m.OverwriteConfirm, scopeBackgroundModel, components.Center, components.Center, 0, 0, confirmOptions)
			return confirmOverlay.View()
		}
	}

	return ""
}

// staticBackgroundModel is a model that renders static content (used for nested overlays)
type staticBackgroundModel struct {
	content string
}

func (m *staticBackgroundModel) Init() tea.Cmd {
	return nil
}

func (m *staticBackgroundModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *staticBackgroundModel) View() string {
	return m.content
}

// runBackupModal runs the backup modal flow (scope selection + optional overwrite)
func runBackupModal(title, message string, availableScopes []platform.InstallationScope, zipPath string, needsOverwrite bool) ([]platform.InstallationScope, bool, error) {
	model := newBackupModalModel(title, message, availableScopes, zipPath, needsOverwrite)

	background := &components.BlankBackgroundModel{}
	options := components.OverlayOptions{
		ShowBorder:  false,
		BorderWidth: 0,
	}

	overlay := components.NewOverlayWithOptions(model, background, components.Center, components.Center, 0, 0, options)

	program := tea.NewProgram(overlay, tea.WithAltScreen())

	finalModel, err := program.Run()
	if err != nil {
		return nil, false, fmt.Errorf("failed to run backup modal: %w", err)
	}

	// Extract result from overlay
	if overlayModel, ok := finalModel.(*components.OverlayModel); ok {
		if backupModel, ok := overlayModel.Foreground.(*backupModalModel); ok {
			if backupModel.Confirmed && len(backupModel.SelectedScopes) > 0 {
				return backupModel.SelectedScopes, true, nil
			}
			// If confirmed but no scopes, something went wrong
			if backupModel.Confirmed && len(backupModel.SelectedScopes) == 0 {
				return nil, false, fmt.Errorf("backup confirmed but no scopes selected")
			}
			return nil, false, nil
		}
	}

	return nil, false, nil
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
