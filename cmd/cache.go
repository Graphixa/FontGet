package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"fontget/internal/repo"

	"github.com/spf13/cobra"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage FontGet cache",
	Long:  `Manage the FontGet font cache including clearing, status, and validation.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no subcommand is provided, show help
		return cmd.Help()
	},
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all cached data",
	Long:  `Remove all cached font data and sources. This will force a fresh download on next use.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		GetLogger().Info("Starting cache clear operation")

		// Get cache directory
		cache, err := repo.NewCache()
		if err != nil {
			GetLogger().Error("Failed to get cache directory: %v", err)
			return fmt.Errorf("failed to get cache directory: %w", err)
		}

		// Clear sources cache
		sourcesCacheDir := filepath.Join(cache.Dir, "sources")
		if err := os.RemoveAll(sourcesCacheDir); err != nil {
			GetLogger().Error("Failed to clear sources cache: %v", err)
			return fmt.Errorf("failed to clear sources cache: %w", err)
		}

		// Recreate the directory
		if err := os.MkdirAll(sourcesCacheDir, 0755); err != nil {
			GetLogger().Error("Failed to recreate sources cache directory: %v", err)
			return fmt.Errorf("failed to recreate sources cache directory: %w", err)
		}

		fmt.Println(Green("Cache cleared successfully"))
		GetLogger().Info("Cache clear operation completed")
		return nil
	},
}

var cacheStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show cache statistics",
	Long:  `Display information about the current cache including size, age, and status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		GetLogger().Info("Starting cache status operation")

		// Get cache directory
		cache, err := repo.NewCache()
		if err != nil {
			GetLogger().Error("Failed to get cache directory: %v", err)
			return fmt.Errorf("failed to get cache directory: %w", err)
		}

		fmt.Printf("Cache Directory: %s\n", cache.Dir)

		// Check sources cache
		sourcesCacheDir := filepath.Join(cache.Dir, "sources")
		if info, err := os.Stat(sourcesCacheDir); err == nil {
			fmt.Printf("Sources Cache: %s (modified: %s)\n",
				formatFileSize(getDirSize(sourcesCacheDir)),
				info.ModTime().Format("2006-01-02 15:04:05"))
		} else {
			fmt.Println("Sources Cache: Not found")
		}

		// Check individual source files
		if entries, err := os.ReadDir(sourcesCacheDir); err == nil {
			fmt.Printf("Cached Sources: %d\n", len(entries))
			for _, entry := range entries {
				if !entry.IsDir() {
					filePath := filepath.Join(sourcesCacheDir, entry.Name())
					if info, err := os.Stat(filePath); err == nil {
						age := time.Since(info.ModTime())
						fmt.Printf("  - %s: %s (age: %s)\n",
							entry.Name(),
							formatFileSize(info.Size()),
							formatDuration(age))
					}
				}
			}
		}

		GetLogger().Info("Cache status operation completed")
		return nil
	},
}

var cacheValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate cache integrity",
	Long:  `Check the integrity of cached data and report any issues.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		GetLogger().Info("Starting cache validation operation")

		// Get cache directory
		cache, err := repo.NewCache()
		if err != nil {
			GetLogger().Error("Failed to get cache directory: %v", err)
			return fmt.Errorf("failed to get cache directory: %w", err)
		}

		fmt.Printf("Validating cache in: %s\n", cache.Dir)

		// Check sources cache
		sourcesCacheDir := filepath.Join(cache.Dir, "sources")
		if _, err := os.Stat(sourcesCacheDir); err != nil {
			fmt.Println(Red("Sources cache directory not found"))
			return nil
		}

		fmt.Printf("Sources cache directory: %s\n", sourcesCacheDir)

		// Validate individual source files
		entries, err := os.ReadDir(sourcesCacheDir)
		if err != nil {
			GetLogger().Error("Failed to read sources cache directory: %v", err)
			return fmt.Errorf("failed to read sources cache directory: %w", err)
		}

		validCount := 0
		invalidCount := 0

		for _, entry := range entries {
			if !entry.IsDir() {
				filePath := filepath.Join(sourcesCacheDir, entry.Name())
				if isValidCacheFile(filePath) {
					fmt.Printf("  ✓ %s: Valid\n", entry.Name())
					validCount++
				} else {
					fmt.Printf("  ✗ %s: Invalid\n", entry.Name())
					invalidCount++
				}
			}
		}

		fmt.Printf("\nValidation Results:\n")
		fmt.Printf("  Valid files: %d\n", validCount)
		fmt.Printf("  Invalid files: %d\n", invalidCount)

		if invalidCount > 0 {
			fmt.Println(Yellow("Some cache files are invalid. Consider running 'fontget cache clear' to fix."))
		} else {
			fmt.Println(Green("All cache files are valid"))
		}

		GetLogger().Info("Cache validation operation completed")
		return nil
	},
}

// Helper functions

func getDirSize(dir string) int64 {
	var size int64
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%.0fh", d.Hours())
	} else {
		return fmt.Sprintf("%.0fd", d.Hours()/24)
	}
}

func isValidCacheFile(filePath string) bool {
	// Check if file exists and is readable
	if info, err := os.Stat(filePath); err != nil || info.Size() == 0 {
		return false
	}

	// Try to read the file as JSON
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	// Basic JSON validation (check if it starts with { and ends with })
	if len(data) < 2 {
		return false
	}

	// Remove whitespace
	content := string(data)
	for len(content) > 0 && (content[0] == ' ' || content[0] == '\n' || content[0] == '\r' || content[0] == '\t') {
		content = content[1:]
	}
	for len(content) > 0 && (content[len(content)-1] == ' ' || content[len(content)-1] == '\n' || content[len(content)-1] == '\r' || content[len(content)-1] == '\t') {
		content = content[:len(content)-1]
	}

	return len(content) > 0 && content[0] == '{' && content[len(content)-1] == '}'
}

func init() {
	rootCmd.AddCommand(cacheCmd)
	cacheCmd.AddCommand(cacheClearCmd)
	cacheCmd.AddCommand(cacheStatusCmd)
	cacheCmd.AddCommand(cacheValidateCmd)
}
