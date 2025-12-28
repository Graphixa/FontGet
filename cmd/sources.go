package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"fontget/internal/components"
	"fontget/internal/config"
	"fontget/internal/functions"
	"fontget/internal/output"
	"fontget/internal/repo"
	"fontget/internal/shared"
	"fontget/internal/ui"

	"github.com/spf13/cobra"
)

// UI text constants
const (
	DisabledTag = " [Disabled]"
)

var sourcesCmd = &cobra.Command{
	Use:          "sources",
	Short:        "Manage FontGet font sources",
	SilenceUsage: true,
	Long: `Manage font sources that provide data to discover and install fonts.

Sources define where FontGet retrieves font information. Only add sources from trusted locations.
Use subcommands to view, update, validate, or manage sources interactively.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no subcommand is provided, show help
		return cmd.Help()
	},
}

var sourcesInfoCmd = &cobra.Command{
	Use:          "info",
	Short:        "Show sources information",
	Long:         `Display information about configured font sources, including status, cache size, and last update time.`,
	SilenceUsage: true, // Don't show usage info on errors
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get logger after it's been initialized
		logger := GetLogger()
		if logger != nil {
			logger.Info("Starting sources info operation")
		}

		output.GetDebug().State("Starting sources info operation")

		// Show sources information
		manifestPath := filepath.Join(config.GetAppConfigDir(), "manifest.json")

		output.GetDebug().State("Calling config.LoadManifest()")
		configManifest, err := config.LoadManifest()
		if err != nil {
			GetLogger().Error("Failed to load manifest: %v", err)
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("config.LoadManifest() failed: %v", err)
			output.GetDebug().State("Error loading sources manifest: %v", err)
			return fmt.Errorf("unable to load font repository: %v", err)
		}

		// Check if source files actually exist - if not, force refresh with spinner
		home, err := os.UserHomeDir()
		var r *repo.Repository
		var manifest *repo.FontManifest

		if err == nil {
			sourcesDir := filepath.Join(home, ".fontget", "sources")
			if entries, err := os.ReadDir(sourcesDir); err != nil || len(entries) == 0 {
				// No source files exist, force refresh with spinner
				if logger != nil {
					logger.Info("No source files found, forcing refresh with spinner")
				}
				r, _ = repo.GetRepositoryWithRefresh()
			} else {
				// Source files exist, use normal repository loading
				r, _ = repo.GetRepository()
			}
		} else {
			// Fallback to normal repository loading
			r, err = repo.GetRepository()
		}

		if err != nil {
			if logger != nil {
				logger.Warn("Failed to get repository: %v", err)
			}
		} else {
			manifest, err = r.GetManifest()
			if err != nil {
				if logger != nil {
					logger.Warn("Failed to get manifest: %v", err)
				}
			}
		}

		// Summary card
		var cards []components.Card
		lastUpdated := "Unknown"
		relative := ""
		if manifest != nil && !manifest.LastUpdated.IsZero() {
			lastUpdated = manifest.LastUpdated.Format("Mon, 02 Jan 2006 15:04:05 MST")
			relative = fmt.Sprintf(" (%s ago)", formatDuration(time.Since(manifest.LastUpdated)))
		}
		// enabled/disabled counts are derived below; no need to precompute ordered list here

		sourcesDir := ""
		if err == nil {
			sourcesDir = filepath.Join(home, ".fontget", "sources")
		}
		totalCacheSize := int64(0)
		if sourcesDir != "" {
			if _, err := os.Stat(sourcesDir); err == nil {
				totalCacheSize = getDirSize(sourcesDir)
			}
		}

		// Build summary content with requested layout and spacing
		disabledCount := 0
		for _, s := range configManifest.Sources {
			if !s.Enabled {
				disabledCount++
			}
		}
		var sb strings.Builder
		sb.WriteString(ui.CardLabel.Render("Manifest File: "))
		sb.WriteString(ui.Text.Render(manifestPath))
		sb.WriteString("\n")
		if sourcesDir != "" {
			sb.WriteString(ui.CardLabel.Render("Cache Path: "))
			sb.WriteString(ui.Text.Render(sourcesDir))
			sb.WriteString("\n\n")
		}
		sb.WriteString(ui.CardLabel.Render("Last Updated: "))
		sb.WriteString(ui.Text.Render(lastUpdated + relative))
		sb.WriteString("\n")
		if sourcesDir != "" {
			sb.WriteString(ui.CardLabel.Render("Total Cache Size: "))
			sb.WriteString(ui.Text.Render(formatFileSize(totalCacheSize)))
			sb.WriteString("\n\n")
		} else {
			sb.WriteString("\n")
		}
		totalSources := len(configManifest.Sources)
		label := "Total Sources: "
		sb.WriteString(ui.CardLabel.Render(label))
		sb.WriteString(ui.Text.Render(fmt.Sprintf("%d", totalSources)))
		if disabledCount > 0 {
			sb.WriteString(ui.Text.Render(fmt.Sprintf(" (%d disabled)", disabledCount)))
		}
		cards = append(cards, components.CustomCard("Summary", sb.String()))

		// Render Summary card only (no page title)
		model := components.NewCardModel("", cards)
		model.SetWidth(80)
		fmt.Println()
		fmt.Println(model.Render())

		// Unified Sources table without headings, includes Status
		fmt.Println()
		fmt.Println(ui.GetSourcesInfoTableHeader())
		fmt.Println(ui.GetTableSeparator())
		// Build rows and sort: built-in first, then enabled, then name
		type row struct {
			name  string
			src   config.SourceConfig
			built bool
		}
		var rows []row
		if def, _ := config.GetDefaultManifest(); def != nil {
			for n, s := range configManifest.Sources {
				_, built := def.Sources[n]
				rows = append(rows, row{name: n, src: s, built: built})
			}
		} else {
			for n, s := range configManifest.Sources {
				rows = append(rows, row{name: n, src: s, built: false})
			}
		}
		sort.Slice(rows, func(i, j int) bool {
			if rows[i].built != rows[j].built {
				return rows[i].built
			}
			if rows[i].src.Enabled != rows[j].src.Enabled {
				return rows[i].src.Enabled
			}
			return strings.ToLower(rows[i].name) < strings.ToLower(rows[j].name)
		})
		for _, r := range rows {
			sourceName := r.name
			source := r.src
			// Name column with proper styling and width control
			var nameStyled string
			if !source.Enabled {
				// Keep the red [Disabled] tag within the name column width
				tag := DisabledTag
				nameWidth := ui.TableColSrcName - len(tag)
				if nameWidth < 0 {
					nameWidth = 0
				}
				nameText := shared.TruncateString(sourceName, nameWidth)
				visible := nameText + tag
				pad := 0
				if len(visible) < ui.TableColSrcName {
					pad = ui.TableColSrcName - len(visible)
				}
				nameStyled = ui.FormReadOnly.Render(nameText) + ui.ErrorText.Render(tag) + strings.Repeat(" ", pad)
			} else {
				nameText := shared.TruncateString(sourceName, ui.TableColSrcName)
				paddedName := fmt.Sprintf("%-*s", ui.TableColSrcName, nameText)
				nameStyled = ui.TableSourceName.Render(paddedName)
			}
			last := "Unknown"
			if manifest != nil {
				last = lastUpdated
			}
			// Determine Type
			typ := "Custom"
			if def, _ := config.GetDefaultManifest(); def != nil {
				if _, ok := def.Sources[sourceName]; ok {
					typ = "Built-in"
				}
			}

			// Name already includes disabled tag within the column width
			displayName := nameStyled

			fmt.Printf("%s %-*s %-*s %-*s\n",
				displayName,
				ui.TableColSrcPrefix, shared.TruncateString(source.Prefix, ui.TableColSrcPrefix),
				ui.TableColSrcUpdated, shared.TruncateString(last, ui.TableColSrcUpdated),
				ui.TableColSrcType, typ,
			)
		}

		fmt.Println()
		output.GetDebug().State("Sources operation complete")
		return nil
	},
}

// updateSourceConfigurations updates source configurations to use FontGet-Sources URLs
func updateSourceConfigurations() error {
	// Get logger after it's been initialized
	logger := GetLogger()
	if logger != nil {
		logger.Info("Starting sources configuration update")
	}

	// Load current manifest
	manifest, err := config.LoadManifest()
	if err != nil {
		GetLogger().Error("Failed to load manifest: %v", err)
		output.GetVerbose().Error("%v", err)
		output.GetDebug().Error("config.LoadManifest() failed: %v", err)
		return fmt.Errorf("unable to load font repository: %v", err)
	}

	// Get default manifest for comparison
	defaultManifest, err := config.GetDefaultManifest()
	if err != nil {
		GetLogger().Error("Failed to get default manifest: %v", err)
		output.GetVerbose().Error("%v", err)
		output.GetDebug().Error("config.GetDefaultManifest() failed: %v", err)
		return fmt.Errorf("unable to load default manifest: %v", err)
	}

	// Update sources to use FontGet-Sources URLs
	updated := false
	for name, source := range manifest.Sources {
		if defaultSource, exists := defaultManifest.Sources[name]; exists {
			// Check if URL or prefix needs updating (but NOT the enabled status)
			needsUpdate := source.URL != defaultSource.URL ||
				source.Prefix != defaultSource.Prefix

			if needsUpdate {
				source.URL = defaultSource.URL
				source.Prefix = defaultSource.Prefix
				// Don't change source.Enabled - let the user control that via sources manage
				manifest.Sources[name] = source
				updated = true
				if logger != nil {
					logger.Info("Updated %s source URL and prefix", name)
				}
			}
		}
	}

	if updated {
		// Save updated configuration
		if err := config.SaveManifest(manifest); err != nil {
			GetLogger().Error("Failed to save updated sources config: %v", err)
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("config.SaveManifest() failed: %v", err)
			return fmt.Errorf("unable to save sources configuration: %v", err)
		}
	}

	return nil
}

// runSourcesUpdateVerbose runs the sources update with detailed verbose logging
func runSourcesUpdateVerbose() error {
	// Get logger after it's been initialized
	logger := GetLogger()
	if logger != nil {
		logger.Info("Starting verbose sources update operation")
	}

	// Load manifest
	manifest, err := config.LoadManifest()
	if err != nil {
		GetLogger().Error("Failed to load manifest: %v", err)
		output.GetVerbose().Error("%v", err)
		output.GetDebug().Error("config.LoadManifest() failed: %v", err)
		return fmt.Errorf("unable to load font repository: %v", err)
	}

	// Get enabled sources
	enabledSources := functions.GetEnabledSourcesInOrder(manifest)
	if len(enabledSources) == 0 {
		return fmt.Errorf("no sources are enabled")
	}

	// Use UI components instead of direct color functions

	// Start with a blank line for consistent spacing
	fmt.Println()

	successful := 0
	failed := 0
	failedSources := make(map[string]bool) // Track which sources failed

	// Process each source with detailed logging
	for _, sourceName := range enabledSources {
		source, exists := manifest.Sources[sourceName]
		if !exists {
			fmt.Printf("Checking for updates for %s\n", sourceName)
			fmt.Printf("%s\n\n", ui.RenderError("Source not found in configuration"))
			failed++
			failedSources[sourceName] = true
			continue
		}

		fmt.Printf("Checking for updates for %s\n", sourceName)

		// Create HTTP client with timeout for quick validation
		appConfig := config.GetUserPreferences()
		requestTimeout := config.ParseDuration(appConfig.Network.RequestTimeout, 10*time.Second)
		client := &http.Client{
			Timeout: requestTimeout,
		}

		// First, check if source is reachable with HEAD request (fast validation)
		headResp, err := client.Head(source.URL)
		if err != nil {
			// Provide more specific error messages
			var errorMsg string
			if strings.Contains(err.Error(), "timeout") {
				errorMsg = "request timeout after 5 seconds"
			} else if strings.Contains(err.Error(), "no such host") {
				errorMsg = "host not found"
			} else if strings.Contains(err.Error(), "connection refused") {
				errorMsg = "connection refused"
			} else {
				errorMsg = fmt.Sprintf("network error: %v", err)
			}
			fmt.Printf("%s\n\n", ui.RenderError(errorMsg))
			failed++
			failedSources[sourceName] = true
			continue
		}
		headResp.Body.Close()

		// Check HTTP status code immediately
		if headResp.StatusCode >= 400 {
			fmt.Printf("%s\n\n", ui.RenderError(fmt.Sprintf("Source URL returned status %d", headResp.StatusCode)))
			failed++
			failedSources[sourceName] = true
			continue
		}

		// Source is reachable, now download the full content
		fmt.Printf("Source Found\n")
		fmt.Printf("Downloading from: %s\n", ui.InfoText.Render(source.URL))
		resp, err := client.Get(source.URL)
		if err != nil {
			fmt.Printf("%s\n\n", ui.RenderError(fmt.Sprintf("Failed to download source - %v", err)))
			failed++
			failedSources[sourceName] = true
			continue
		}

		// Read the response body
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close() // Close immediately after reading
		if err != nil {
			fmt.Printf("%s\n\n", ui.RenderError(fmt.Sprintf("Failed to read source content - %v", err)))
			failed++
			failedSources[sourceName] = true
			continue
		}

		// Validate JSON
		var jsonData interface{}
		if err := json.Unmarshal(body, &jsonData); err != nil {
			fmt.Printf("%s\n\n", ui.RenderError(fmt.Sprintf("Source content is not valid JSON - %v", err)))
			failed++
			continue
		}

		// Success - show where it would be cached
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("%s\n\n", ui.RenderError(fmt.Sprintf("Failed to get home directory - %v", err)))
			failed++
			failedSources[sourceName] = true
			continue
		}
		// Sanitize source name for filename (same as in repo package)
		sanitizedName := strings.ToLower(sourceName)
		sanitizedName = strings.ReplaceAll(sanitizedName, " ", "_")
		cachePath := filepath.Join(homeDir, ".fontget", "sources", fmt.Sprintf("%s.json", sanitizedName))
		fmt.Printf("%s\n\n", ui.RenderSuccess(fmt.Sprintf("Downloaded to %s (%d bytes)", cachePath, len(body))))
		successful++
	}

	// Temporarily disable failed sources before refresh to avoid trying to load them again
	originalEnabledStates := make(map[string]bool)
	if len(failedSources) > 0 {
		for sourceName := range failedSources {
			if source, exists := manifest.Sources[sourceName]; exists {
				originalEnabledStates[sourceName] = source.Enabled
				source.Enabled = false
				manifest.Sources[sourceName] = source // Update the map with the modified struct
			}
		}
		// Save the modified manifest so GetManifestWithRefresh will skip failed sources
		if err := config.SaveManifest(manifest); err != nil {
			GetLogger().Error("Failed to save manifest with disabled sources: %v", err)
			output.GetVerbose().Warning("Failed to temporarily disable failed sources, they may be attempted again")
		}
	}

	// Try to load manifest with force refresh
	fmt.Printf("Refreshing font data cache...\n")
	progress := func(current, total int, message string) {
		if current == total {
			fmt.Printf("%s\n", ui.RenderSuccess("\nFont data cache refreshed successfully\n"))
		} else {
			fmt.Printf("   %s\n", message)
		}
	}

	fontManifest, err := repo.GetManifestWithRefresh(nil, progress, true)

	// Restore original enabled states for failed sources
	if len(originalEnabledStates) > 0 {
		for sourceName, wasEnabled := range originalEnabledStates {
			if source, exists := manifest.Sources[sourceName]; exists {
				source.Enabled = wasEnabled
				manifest.Sources[sourceName] = source // Update the map with the modified struct
			}
		}
		// Restore the manifest
		if err := config.SaveManifest(manifest); err != nil {
			GetLogger().Error("Failed to restore manifest: %v", err)
			output.GetVerbose().Warning("Failed to restore original source states")
		}
	}
	if err != nil {
		fmt.Printf("%s\n", ui.RenderWarning(fmt.Sprintf("Failed to refresh font data cache: %v", err)))
	} else {
		// Count total fonts
		totalFonts := 0
		for _, sourceInfo := range fontManifest.Sources {
			totalFonts += len(sourceInfo.Fonts)
		}
		fmt.Printf("%s %d\n", ui.InfoText.Render("Total fonts available:"), totalFonts)
	}

	// Status Report at the bottom (only shown in verbose mode - this function is only called in verbose mode)
	fmt.Printf("\n%s\n", ui.TextBold.Render("Status Report"))
	fmt.Printf("---------------------------------------------\n")
	fmt.Printf("%s: %s  |  %s: %s  |  %s: %s\n\n",
		ui.SuccessText.Render("Updated"), ui.Text.Render(fmt.Sprintf("%d", successful)),
		ui.WarningText.Render("Skipped"), ui.Text.Render("0"),
		ui.ErrorText.Render("Failed"), ui.Text.Render(fmt.Sprintf("%d", failed)))

	return nil
}

// sourcesUpdateCmd handles updating sources and refreshing cache
var sourcesUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update source configuration and refresh cache",
	Long: `Update source configurations and refresh the font database.

Downloads the latest font data from all enabled sources and updates the local cache.`,
	Args:         cobra.NoArgs,
	SilenceUsage: true, // Don't show usage info on errors
	RunE: func(cmd *cobra.Command, args []string) error {
		verbose, _ := cmd.Flags().GetBool("verbose")

		GetLogger().Info("Starting sources update operation")

		// Debug-level information for developers
		output.GetDebug().Message("Debug mode enabled - showing detailed diagnostic information")

		// Clear existing cached sources first
		output.GetVerbose().Info("Clearing existing cached sources")
		output.GetDebug().State("Clearing sources directory before update")

		home, err := os.UserHomeDir()
		if err != nil {
			GetLogger().Error("Failed to get home directory: %v", err)
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("os.UserHomeDir() failed: %v", err)
			return fmt.Errorf("unable to access home directory: %v", err)
		}

		sourcesDir := filepath.Join(home, ".fontget", "sources")
		if err := os.RemoveAll(sourcesDir); err != nil {
			GetLogger().Error("Failed to clear sources directory: %v", err)
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("os.RemoveAll() failed: %v", err)
			return fmt.Errorf("unable to clear sources directory: %v", err)
		}

		// Recreate the sources directory
		if err := os.MkdirAll(sourcesDir, 0755); err != nil {
			GetLogger().Error("Failed to recreate sources directory: %v", err)
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("os.MkdirAll() failed: %v", err)
			return fmt.Errorf("unable to create sources directory: %v", err)
		}

		output.GetVerbose().Success("Cleared existing cached sources")
		output.GetDebug().State("Sources directory cleared and recreated")

		// Update the source configurations
		output.GetVerbose().Info("Updating source configurations")
		output.GetDebug().State("Calling updateSourceConfigurations()")
		if err := updateSourceConfigurations(); err != nil {
			return err
		}

		// If verbose, run in detailed logging mode instead of TUI
		if verbose {
			return runSourcesUpdateVerbose()
		}

		// Run TUI update display with spinners (title shown in TUI)
		return RunSourcesUpdateTUI(verbose)
	},
}

func init() {
	rootCmd.AddCommand(sourcesCmd)

	// Add subcommands
	sourcesCmd.AddCommand(sourcesInfoCmd)
	sourcesCmd.AddCommand(sourcesUpdateCmd)
	sourcesCmd.AddCommand(sourcesValidateCmd)

	// Add flags
	sourcesUpdateCmd.Flags().BoolP("verbose", "v", false, "Show detailed error messages for failed sources")
}

// Helper functions for sources validate

func isValidSourceFile(filePath string) bool {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	var jsonData interface{}
	return json.Unmarshal(data, &jsonData) == nil
}

func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func getDirSize(dir string) int64 {
	var size int64
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	} else {
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd", days)
	}
}

var sourcesValidateCmd = &cobra.Command{
	Use:          "validate",
	Short:        "Validate cached sources integrity",
	SilenceUsage: true,
	Long: `Validate cached source files and report any issues.

If validation fails, run 'fontget sources update' to refresh the source files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		GetLogger().Info("Starting sources validation operation")

		// Debug-level information for developers
		output.GetDebug().Message("Debug mode enabled - showing detailed diagnostic information")

		// Get sources directory
		output.GetVerbose().Info("Getting sources directory")
		output.GetDebug().State("Calling getSourcesDir()")
		home, err := os.UserHomeDir()
		if err != nil {
			GetLogger().Error("Failed to get home directory: %v", err)
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("os.UserHomeDir() failed: %v", err)
			return fmt.Errorf("unable to access home directory: %v", err)
		}

		sourcesDir := filepath.Join(home, ".fontget", "sources")
		output.GetVerbose().Info("Validating sources directory: %s", sourcesDir)

		// Check if sources directory exists
		if _, err := os.Stat(sourcesDir); err != nil {
			fmt.Println(ui.RenderError("Sources directory not found"))
			output.GetVerbose().Warning("Sources directory does not exist")
			output.GetDebug().State("Sources directory not found: %s", sourcesDir)
			return nil
		}

		// Start with a blank line for consistent spacing
		fmt.Println()
		fmt.Printf("%s %s\n\n", ui.TextBold.Render("Sources Path:"), sourcesDir)

		// Validate individual source files
		entries, err := os.ReadDir(sourcesDir)
		if err != nil {
			GetLogger().Error("Failed to read sources directory: %v", err)
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("os.ReadDir() failed: %v", err)
			return fmt.Errorf("unable to read sources directory: %v", err)
		}

		validCount := 0
		invalidCount := 0
		jsonFileCount := 0

		output.GetVerbose().Info("Validating %d source files", len(entries))
		output.GetDebug().State("Starting validation of %d files", len(entries))

		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
				jsonFileCount++
				filePath := filepath.Join(sourcesDir, entry.Name())
				output.GetDebug().State("Validating file: %s", entry.Name())

				if isValidSourceFile(filePath) {
					// Get file size for display
					if info, err := os.Stat(filePath); err == nil {
						size := formatFileSize(info.Size())
						fmt.Printf("  %s %s (%s) | %s\n",
							ui.SuccessText.Render("✓"),
							entry.Name(),
							size,
							ui.SuccessText.Render("Valid"))
						output.GetDebug().State("File %s is valid, size: %s", entry.Name(), size)
					} else {
						fmt.Printf("  %s %s | %s\n",
							ui.SuccessText.Render("✓"),
							entry.Name(),
							ui.SuccessText.Render("Valid"))
						output.GetDebug().State("File %s is valid", entry.Name())
					}
					validCount++
				} else {
					fmt.Printf("  %s %s | %s\n",
						ui.ErrorText.Render("✗"),
						entry.Name(),
						ui.ErrorText.Render("Invalid"))
					invalidCount++
					output.GetVerbose().Warning("File %s is invalid", entry.Name())
					output.GetDebug().State("File %s failed validation", entry.Name())
				}
			}
		}

		// Check if no JSON files were found
		if jsonFileCount == 0 {
			fmt.Println(ui.RenderWarning("No source files found to validate."))
			fmt.Println(ui.RenderWarning("Try running: fontget sources update"))
			output.GetVerbose().Info("No JSON source files found in directory")
			output.GetDebug().State("No .json files found in sources directory")
			return nil
		}

		output.GetVerbose().Info("Validation completed - Valid: %d, Invalid: %d", validCount, invalidCount)
		output.GetDebug().State("Validation results: %d valid, %d invalid", validCount, invalidCount)

		if invalidCount > 0 {
			fmt.Printf("\n%s\n", ui.WarningText.Render("One or more sources failed to validate."))
			fmt.Printf("Run 'fontget sources validate --help' for troubleshooting steps.\n\n")
			output.GetVerbose().Warning("Sources validation found %d invalid files", invalidCount)
		} else {
			fmt.Printf("\n%s\n", ui.SuccessText.Render("All source files are valid\n"))
			output.GetVerbose().Success("All source files are valid")
		}

		output.GetVerbose().Success("Sources validation operation completed")
		output.GetDebug().State("Sources validation operation completed successfully")
		GetLogger().Info("Sources validation operation completed")
		return nil
	},
}
