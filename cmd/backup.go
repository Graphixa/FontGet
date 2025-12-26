package cmd

import (
	"archive/zip"
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

System fonts are always excluded from backups.`,
	Example: `  fontget backup
  fontget backup fonts-backup.zip
  fontget backup D:\Backups\my-fonts.zip
  fontget backup ./backups/`,
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

		// Validate and normalize output path
		zipPath, err := validateAndNormalizeOutputPath(outputPath)
		if err != nil {
			// Check if this is a cancellation (user chose not to overwrite)
			if strings.Contains(err.Error(), "backup cancelled") {
				// User cancelled - show friendly message and return nil (no error)
				fmt.Printf("%s\n", ui.WarningText.Render("Backup cancelled - file already exists."))
				fmt.Println()
				return nil
			}
			return err
		}

		// Auto-detect accessible scopes
		scopes, err := detectAccessibleScopes(fm)
		if err != nil {
			GetLogger().Error("Failed to detect accessible scopes: %v", err)
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("detectAccessibleScopes() failed: %v", err)
			return fmt.Errorf("unable to detect accessible font scopes: %v", err)
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
			return performBackup(fm, scopes, zipPath)
		}

		// Use progress bar for backup operation
		return runBackupWithProgressBar(fm, scopes, zipPath)
	},
}

// generateDefaultBackupFilename generates a date-based backup filename
func generateDefaultBackupFilename() string {
	now := time.Now()
	dateStr := now.Format("2006-01-02")
	return fmt.Sprintf("font-backup-%s.zip", dateStr)
}

// validateAndNormalizeOutputPath validates and normalizes the output path with guard rails
func validateAndNormalizeOutputPath(outputPath string) (string, error) {
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
			return "", fmt.Errorf("output path exists and is not a zip file: %s", outputPath)
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
					return "", fmt.Errorf("output path must be a .zip file: %s", outputPath)
				}
				// Will create parent directory later
			} else if !parentInfo.IsDir() {
				return "", fmt.Errorf("parent path exists but is not a directory: %s", parentDir)
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
		return "", fmt.Errorf("invalid output path: %v", err)
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
			return "", fmt.Errorf("cannot write backup to system font directory: %s", absPath)
		}
	}

	// Check if the final file path already exists and prompt for confirmation
	if _, err := os.Stat(absPath); err == nil {
		// File exists - prompt for confirmation before overwriting
		confirmed, err := components.RunConfirm(
			"File Already Exists",
			fmt.Sprintf("File already exists. Overwrite '%s'?", filepath.Base(absPath)),
		)
		if err != nil {
			return "", fmt.Errorf("unable to show confirmation dialog: %v", err)
		}

		if !confirmed {
			return "", fmt.Errorf("backup cancelled - file already exists: %s", absPath)
		}
	}

	return absPath, nil
}

// formatScopeDisplay formats the scope list for user-friendly display
func formatScopeDisplay(scopes []platform.InstallationScope) string {
	if len(scopes) == 0 {
		return "no scope"
	}
	if len(scopes) == 1 {
		if scopes[0] == platform.UserScope {
			return "user scope"
		}
		return "machine scope"
	}
	// Both scopes
	return "user and machine scope"
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

	// Format scope display for user-friendly output
	scopeDisplay := formatScopeDisplay(scopes)

	// Show pre-operation summary - plain text for count, Primary for scope
	fmt.Printf("Found %d font families to backup %s\n", totalFamilies, ui.QueryText.Render(fmt.Sprintf("(%s)", scopeDisplay)))

	// Create progress bar title with destination path - plain text, Secondary for path
	progressTitle := fmt.Sprintf("Backing up font files to: %s", ui.SecondaryText.Render(zipPath))

	// Create empty operation items - we just want the progress bar, not individual items
	// Set TotalItems to 0 to hide the count text
	operationItems := []components.OperationItem{}

	// Run progress bar with no items (just progress bar, no count, no item list)
	verbose := IsVerbose()
	debug := IsDebug()
	var backupResult *backupResult
	progressErr := components.RunProgressBar(
		progressTitle,
		operationItems,
		verbose, // Verbose mode: show operational details and file/variant listings
		debug,   // Debug mode: show technical details
		func(send func(msg tea.Msg)) error {
			// Perform the actual backup operation
			var err error
			backupResult, err = performBackupWithProgress(fm, scopes, zipPath, fonts, fontMap, matches, totalFamilies, send)
			return err
		},
	)

	if progressErr != nil {
		// Print error with proper styling (Cobra won't print it since SilenceErrors is true)
		fmt.Printf("%s\n", ui.ErrorText.Render(fmt.Sprintf("Error: %v", progressErr)))
		fmt.Println()
		return progressErr
	}

	// Show success message after progress bar completes - simplified
	if backupResult != nil {
		// Just show the count - path was already shown in progress bar title
		successMsg := fmt.Sprintf("Successfully backed up %d font families.", backupResult.familyCount)
		fmt.Printf("%s\n", ui.SuccessText.Render(successMsg))
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

// createBackupZipArchive creates a zip archive from organized font structure.
// If send is not nil, it will be called with progress updates.
// createBackupZipArchive creates a zip archive from organized font files.
//
// It creates a zip archive with fonts organized by source and family name.
// The archive structure is: source/family-name/font-file.ttf
// Progress updates are sent via the send function for TUI display.
//
// Parameters:
//   - sourceFamilyMap: Nested map of source -> family -> font files
//   - zipPath: Path to the output zip file
//   - send: Function to send progress updates (for TUI)
//
// Returns:
//   - *backupResult: Contains family and file counts
//   - error: Error if archive creation fails
func createBackupZipArchive(sourceFamilyMap map[string]map[string][]fontFileInfo, zipPath string, send func(msg tea.Msg)) (*backupResult, error) {
	// Ensure parent directory exists
	if dir := filepath.Dir(zipPath); dir != "." && dir != zipPath {
		if err := os.MkdirAll(dir, 0755); err != nil {
			GetLogger().Error("Failed to create backup directory: %v", err)
			// Don't print error here - let Cobra handle it when we return
			return nil, fmt.Errorf("unable to create directory for backup archive: %v", err)
		}
	}

	// Create zip file
	zipFile, err := os.Create(zipPath)
	if err != nil {
		GetLogger().Error("Failed to create backup zip file: %v", err)
		// Don't print error here - let Cobra handle it when we return
		return nil, fmt.Errorf("unable to create backup archive: %v", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

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

			// Sanitize source and family names for zip paths
			sanitizedSource := shared.SanitizeForZipPath(sourceName)
			sanitizedFamily := shared.SanitizeForZipPath(familyName)

			// Add each font file to the zip
			for _, fontInfo := range fontFiles {
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

				// Update progress after each file for smooth progress bar (if callback provided)
				if send != nil && totalFiles > 0 {
					percent := float64(processedFiles) / float64(totalFiles) * 100
					send(components.ProgressUpdateMsg{Percent: percent})
				}
			}
		}
	}

	// Close zip writer to finalize the archive
	if err := zipWriter.Close(); err != nil {
		GetLogger().Error("Failed to close zip writer: %v", err)
		// Don't print error here - let Cobra handle it when we return
		return nil, fmt.Errorf("unable to finalize backup archive: %v", err)
	}

	output.GetVerbose().Info("Backup archive created: %d font families, %d files", familyCount, fileCount)
	output.GetDebug().State("Backup operation complete - Families: %d, Files: %d", familyCount, fileCount)

	return &backupResult{
		familyCount: familyCount,
		fileCount:   fileCount,
	}, nil
}

func performBackupWithProgress(fm platform.FontManager, _ []platform.InstallationScope, zipPath string, _ []ParsedFont, fontMap map[string][]ParsedFont, matches map[string]*repo.InstalledFontMatch, _ int, send func(msg tea.Msg)) (*backupResult, error) {
	// Organize fonts by source -> family name
	sourceFamilyMap := organizeFontsBySourceAndFamily(fm, fontMap, matches)

	// Create zip archive with progress tracking
	return createBackupZipArchive(sourceFamilyMap, zipPath, send)
}

// performBackupWithCollectedFonts performs the backup operation with pre-collected fonts (for debug mode)
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
	return createBackupZipArchive(sourceFamilyMap, zipPath, nil)
}

func init() {
	rootCmd.AddCommand(backupCmd)
}
