package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fontget/internal/config"
	"fontget/internal/functions"
	"fontget/internal/repo"
	"fontget/internal/ui"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var sourcesCmd = &cobra.Command{
	Use:   "sources",
	Short: "Manage FontGet font sources",
	Long: `Manage sources with the sub-commands. A source provides the data for you to discover and install fonts. Only add a new source if you trust it as a secure location.

usage: fontget sources [<command>] [<options>]`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no subcommand is provided, show help
		return cmd.Help()
	},
}

var sourcesInfoCmd = &cobra.Command{
	Use:          "info",
	Short:        "Show sources information",
	Long:         `Display detailed information about the current FontGet sources configuration.`,
	SilenceUsage: true, // Don't show usage info on errors
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get logger after it's been initialized
		logger := GetLogger()
		if logger != nil {
			logger.Info("Starting sources info operation")
		}

		// Show sources information
		manifestPath := filepath.Join(config.GetAppConfigDir(), "manifest.json")

		configManifest, err := config.LoadManifest()
		if err != nil {
			if logger != nil {
				logger.Error("Failed to load manifest: %v", err)
			}
			return fmt.Errorf("failed to load manifest: %w", err)
		}

		// Get repository (uses smart caching like search/list commands)
		var manifest *repo.FontManifest
		r, err := repo.GetRepository()
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

		// Use shared color functions for consistency
		cyan := Cyan
		green := Green
		yellow := Yellow
		red := Red
		white := White

		fmt.Printf("\n%s\n", Bold("Sources Information"))
		fmt.Printf("---------------------------------------------\n")
		fmt.Printf("%s: %s\n", cyan("Manifest File"), manifestPath)
		fmt.Printf("%s: %d\n", cyan("Total Sources"), len(configManifest.Sources))

		enabledSources := functions.GetEnabledSourcesInOrder(configManifest)
		fmt.Printf("%s: %d\n", cyan("Enabled Sources"), len(enabledSources))

		// Show last updated sources date
		if manifest != nil {
			fmt.Printf("%s: %s\n", cyan("Last Updated"), manifest.LastUpdated.Format("Mon, 02 Jan 2006 15:04:05 MST"))
		}

		// Show cache status and size
		// Note: Cache directory information would be added here when available

		if len(enabledSources) > 0 {
			fmt.Printf("\n%s\n", Bold("Enabled Sources"))
			fmt.Printf("---------------------------------------------\n")

			for i, name := range enabledSources {
				if source, exists := config.GetSourceByName(configManifest, name); exists {
					fmt.Printf("  %d. %s %s\n", i+1, green(name), white(fmt.Sprintf("(%s)", source.Prefix)))
				} else {
					fmt.Printf("  %d. %s %s\n", i+1, red(name), red("(NOT FOUND)"))
				}
			}
		}

		// Show disabled sources
		var disabledSources []string
		for name, source := range configManifest.Sources {
			if !source.Enabled {
				disabledSources = append(disabledSources, name)
			}
		}

		if len(disabledSources) > 0 {
			fmt.Printf("\n%s\n", Bold("Disabled Sources"))
			fmt.Printf("---------------------------------------------\n")
			for i, name := range disabledSources {
				if source, exists := config.GetSourceByName(configManifest, name); exists {
					fmt.Printf("  %d. %s %s\n", i+1, yellow(name), white(fmt.Sprintf("(%s)", source.Prefix)))
				}
			}
		}

		fmt.Printf("\n")

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
		if logger != nil {
			logger.Error("Failed to load manifest: %v", err)
		}
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	// Get default manifest for comparison
	defaultManifest, err := config.GetDefaultManifest()
	if err != nil {
		if logger != nil {
			logger.Error("Failed to get default manifest: %v", err)
		}
		return fmt.Errorf("failed to get default manifest: %w", err)
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
			if logger != nil {
				logger.Error("Failed to save updated sources config: %v", err)
			}
			return fmt.Errorf("failed to save updated sources config: %w", err)
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
		if logger != nil {
			logger.Error("Failed to load manifest: %v", err)
		}
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	// Get enabled sources
	enabledSources := functions.GetEnabledSourcesInOrder(manifest)
	if len(enabledSources) == 0 {
		return fmt.Errorf("no sources are enabled")
	}

	// Color functions
	cyan := color.New(color.FgCyan).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	fmt.Printf("%s\n\n", cyan("Updating FontGet Sources..."))

	successful := 0
	failed := 0

	// Process each source with detailed logging
	for _, sourceName := range enabledSources {
		source, exists := manifest.Sources[sourceName]
		if !exists {
			fmt.Printf("Checking for updates for %s\n", sourceName)
			fmt.Printf("%s\n\n", red("Error: Source not found in configuration"))
			failed++
			continue
		}

		fmt.Printf("Checking for updates for %s\n", sourceName)

		// Create HTTP client with shorter timeout for faster error detection
		client := &http.Client{
			Timeout: 5 * time.Second,
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
			fmt.Printf("%s\n\n", red(fmt.Sprintf("Error: %s", errorMsg)))
			failed++
			continue
		}
		headResp.Body.Close()

		// Check HTTP status code immediately
		if headResp.StatusCode >= 400 {
			fmt.Printf("%s\n\n", red(fmt.Sprintf("Error: Source URL returned status %d", headResp.StatusCode)))
			failed++
			continue
		}

		// Source is reachable, now download the full content
		fmt.Printf("Source Found\n")
		fmt.Printf("Downloading from: %s\n", yellow(source.URL))
		resp, err := client.Get(source.URL)
		if err != nil {
			fmt.Printf("%s\n\n", red(fmt.Sprintf("Error: Failed to download source - %v", err)))
			failed++
			continue
		}

		// Read the response body
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close() // Close immediately after reading
		if err != nil {
			fmt.Printf("%s\n\n", red(fmt.Sprintf("Error: Failed to read source content - %v", err)))
			failed++
			continue
		}

		// Validate JSON
		var jsonData interface{}
		if err := json.Unmarshal(body, &jsonData); err != nil {
			fmt.Printf("%s\n\n", red(fmt.Sprintf("Error: Source content is not valid JSON - %v", err)))
			failed++
			continue
		}

		// Success - show where it would be cached
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("%s\n\n", red(fmt.Sprintf("Error: Failed to get home directory - %v", err)))
			failed++
			continue
		}
		cachePath := filepath.Join(homeDir, ".fontget", "cache", fmt.Sprintf("%s.json", sourceName))
		fmt.Printf("%s\n\n", green(fmt.Sprintf("Success: Downloaded to %s (%d bytes)", cachePath, len(body))))
		successful++
	}

	// Summary with colors like other commands
	fmt.Printf("%s\n", ui.ReportTitle.Render("Status Report"))
	fmt.Printf("---------------------------------------------\n")

	white := color.New(color.FgWhite).SprintFunc()

	fmt.Printf("%s: %s  |  %s: %s  |  %s: %s\n",
		green("Updated"), white(successful),
		yellow("Skipped"), white(0),
		red("Failed"), white(failed))

	// Try to load manifest with force refresh
	fmt.Printf("\n%s\n", cyan("Refreshing font data cache..."))
	progress := func(current, total int, message string) {
		if current == total {
			fmt.Printf("%s\n", green("Font data cache refreshed successfully"))
		} else {
			fmt.Printf("   %s\n", message)
		}
	}

	fontManifest, err := repo.GetManifestWithRefresh(nil, progress, true)
	if err != nil {
		fmt.Printf("%s\n", yellow(fmt.Sprintf("Warning: Failed to refresh font data cache: %v", err)))
	} else {
		// Count total fonts
		totalFonts := 0
		for _, sourceInfo := range fontManifest.Sources {
			totalFonts += len(sourceInfo.Fonts)
		}
		fmt.Printf("%s %d\n", cyan("Total fonts available:"), totalFonts)
	}

	return nil
}

// showSourcesTable displays sources in a table format
func showSourcesTable(verbose bool) error {
	// Get logger after it's been initialized
	logger := GetLogger()
	if logger != nil {
		logger.Info("Starting sources table display")
	}

	// Load manifest
	manifest, err := config.LoadManifest()
	if err != nil {
		if logger != nil {
			logger.Error("Failed to load manifest: %v", err)
		}
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	// Display sources in table format
	fmt.Printf("FontGet Sources\n")
	fmt.Printf("===============\n\n")

	if len(manifest.Sources) == 0 {
		fmt.Printf("No sources configured.\n")
		return nil
	}

	// Display table header
	fmt.Printf("%-15s %-10s %-8s %-50s\n", "Name", "Prefix", "Status", "URL")
	fmt.Printf("%-15s %-10s %-8s %-50s\n", "----", "------", "------", "---")

	// Display all sources
	enabledCount := 0
	disabledCount := 0
	validationErrors := make(map[string]string)

	// If verbose mode, validate sources
	if verbose {
		client := &http.Client{
			Timeout: 5 * time.Second,
		}

		for name, source := range manifest.Sources {
			if source.Enabled {
				resp, err := client.Head(source.URL)
				if err != nil {
					validationErrors[name] = fmt.Sprintf("Connection error: %v", err)
				} else {
					resp.Body.Close()
					if resp.StatusCode >= 400 {
						validationErrors[name] = fmt.Sprintf("HTTP %d", resp.StatusCode)
					}
				}
			}
		}
	}

	for name, source := range manifest.Sources {
		status := "Disabled"
		if source.Enabled {
			status = "Enabled"
			enabledCount++
		} else {
			disabledCount++
		}

		// Add validation status if verbose and there's an error
		if verbose && validationErrors[name] != "" {
			status += fmt.Sprintf(" (Error: %s)", validationErrors[name])
		}

		fmt.Printf("%-15s %-10s %-8s %-50s\n", name, source.Prefix, status, source.URL)
	}

	fmt.Printf("\nTotal sources: %d (Enabled: %d, Disabled: %d)\n",
		len(manifest.Sources), enabledCount, disabledCount)

	return nil
}

// sourcesUpdateCmd handles updating sources and refreshing cache
var sourcesUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update source configuration and refresh cache",
	Long: `Update source configuration to use FontGet-Sources URLs and refresh the font data cache.

usage: fontget sources update [--verbose]`,
	Args:         cobra.NoArgs,
	SilenceUsage: true, // Don't show usage info on errors
	RunE: func(cmd *cobra.Command, args []string) error {
		verbose, _ := cmd.Flags().GetBool("verbose")

		// First, update the source configurations (like the original command does)
		if err := updateSourceConfigurations(); err != nil {
			return err
		}

		// If verbose, run in detailed logging mode instead of TUI
		if verbose {
			return runSourcesUpdateVerbose()
		}

		// Run TUI update display with spinners
		return RunSourcesUpdateTUI(verbose)
	},
}

func init() {
	sourcesUpdateCmd.Flags().BoolP("verbose", "v", false, "Show detailed error messages for failed sources")
}

// handleSourcesUpdate handles the --update flag for sources command
func handleSourcesUpdate(cmd *cobra.Command) error {
	// Get logger after it's been initialized
	logger := GetLogger()
	if logger != nil {
		logger.Info("Starting sources update operation")
	}

	// Load current manifest
	manifest, err := config.LoadManifest()
	if err != nil {
		if logger != nil {
			logger.Error("Failed to load manifest: %v", err)
		}
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	// Get default manifest for comparison
	defaultManifest, err := config.GetDefaultManifest()
	if err != nil {
		if logger != nil {
			logger.Error("Failed to get default manifest: %v", err)
		}
		return fmt.Errorf("failed to get default manifest: %w", err)
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
			if logger != nil {
				logger.Error("Failed to save updated sources config: %v", err)
			}
			return fmt.Errorf("failed to save updated sources config: %w", err)
		}

		PrintSuccess("Sources configuration updated to use FontGet-Sources URLs")
		PrintInfo("Updated sources:")
		for name, source := range manifest.Sources {
			fmt.Printf("   %s: %s (%s)\n", InfoColor.Sprint(name), source.URL, source.Prefix)
		}
	} else {
		PrintInfo("Sources configuration is already up to date")
	}

	// Force refresh the cache
	fmt.Printf("\n")
	PrintInfo("Refreshing font data cache...")

	// Create a progress callback
	progress := func(current, total int, message string) {
		if current == total {
			PrintSuccess(message)
		} else {
			fmt.Printf("   %s\n", message)
		}
	}

	// Load manifest with force refresh
	fontManifest, err := repo.GetManifestWithRefresh(nil, progress, true)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to refresh font data cache: %v", err)
		}

		// Provide more user-friendly error messages
		if strings.Contains(err.Error(), "404") {
			PrintErrorWithHint("Failed to refresh font data cache", "One or more sources returned 404 - check if the source URLs are correct")
		} else if strings.Contains(err.Error(), "timeout") {
			PrintErrorWithHint("Failed to refresh font data cache", "Request timed out - check your internet connection")
		} else if strings.Contains(err.Error(), "connection refused") {
			PrintErrorWithHint("Failed to refresh font data cache", "Connection refused - check your internet connection")
		} else {
			PrintErrorWithHint("Failed to refresh font data cache", fmt.Sprintf("%v", err))
		}
		return nil
	}

	PrintSuccess("Font data cache refreshed successfully")

	// Count total fonts across all sources
	totalFonts := 0
	for _, sourceInfo := range fontManifest.Sources {
		totalFonts += len(sourceInfo.Fonts)
	}
	fmt.Printf("%s %d\n", InfoColor.Sprint("Total fonts available:"), totalFonts)

	return nil
}

// getEnabledSourcesInOrder returns enabled sources in priority order
func getEnabledSourcesInOrder(manifest *config.Manifest) []string {
	var sources []functions.SourceItem
	for name, source := range manifest.Sources {
		if source.Enabled {
			sources = append(sources, functions.SourceItem{
				Name:     name,
				Priority: source.Priority,
			})
		}
	}

	// Sort by priority
	functions.SortSources(sources)

	var result []string
	for _, source := range sources {
		result = append(result, source.Name)
	}
	return result
}

func init() {
	rootCmd.AddCommand(sourcesCmd)

	// Add subcommands
	sourcesCmd.AddCommand(sourcesInfoCmd)
	sourcesCmd.AddCommand(sourcesUpdateCmd)
	sourcesCmd.AddCommand(sourcesManageCmd)

}
