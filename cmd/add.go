package cmd

import (
	"fmt"
	"os"
	"runtime"

	"fontget/internal/platform"
	"fontget/internal/repo"

	"github.com/spf13/cobra"
)

const (
	fontsDir = "fonts"
)

var (
	scope string
)

var addCmd = &cobra.Command{
	Use:     "add [font-name]",
	Aliases: []string{"install"},
	Short:   "Install a font from Google Fonts",
	Long: `Install a font from Google Fonts by querying the GitHub API and downloading the font files.

The --scope flag determines where the font will be installed:
  user    - Install for current user only (default)
  machine - Install system-wide (requires elevation)`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Validate scope
		installScope := platform.UserScope
		if scope != "" {
			installScope = platform.InstallationScope(scope)
			if installScope != platform.UserScope && installScope != platform.MachineScope {
				fmt.Fprintf(os.Stderr, "Error: invalid scope '%s'. Must be 'user' or 'machine'\n", scope)
				os.Exit(1)
			}
		}

		// Create fonts directory if it doesn't exist
		if err := os.MkdirAll(fontsDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating fonts directory: %v\n", err)
			os.Exit(1)
		}

		// Get platform-specific font manager
		fontManager, err := platform.NewFontManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing font manager: %v\n", err)
			os.Exit(1)
		}

		// Check if elevation is required
		if fontManager.RequiresElevation(installScope) {
			if runtime.GOOS == "windows" {
				// Check if already elevated
				elevated, err := platform.IsElevated()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error checking elevation status: %v\n", err)
					os.Exit(1)
				}

				if !elevated {
					fmt.Println("Installing fonts system-wide requires administrator privileges.")
					fmt.Println("Attempting to relaunch with elevation...")

					// Relaunch with elevation
					if err := platform.RunAsElevated(); err != nil {
						fmt.Fprintf(os.Stderr, "Error relaunching with elevation: %v\n", err)
						fmt.Println("Please run this command as administrator.")
						os.Exit(1)
					}

					// Exit current process as it will be replaced by the elevated one
					os.Exit(0)
				}
			} else {
				// For non-Windows platforms, just print a message
				fmt.Println("Installing fonts system-wide requires administrator privileges.")
				fmt.Println("Please run this command with sudo.")
				os.Exit(1)
			}
		}

		fontName := args[0]
		fonts, err := repo.GetFont(fontName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching font: %v\n", err)
			os.Exit(1)
		}

		installedFiles := []string{}
		for _, font := range fonts {
			fmt.Printf("Downloading %s...\n", font.Name)
			path, err := repo.DownloadFont(&font, fontsDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error downloading %s: %v\n", font.Name, err)
				continue
			}

			fmt.Printf("Installing %s...\n", font.Name)
			if err := fontManager.InstallFont(path, installScope); err != nil {
				fmt.Fprintf(os.Stderr, "Error installing %s: %v\n", font.Name, err)
				continue
			}

			installedFiles = append(installedFiles, font.Name)
		}

		if len(installedFiles) > 0 {
			fmt.Printf("\nSuccessfully installed %d fonts:\n", len(installedFiles))
			for _, file := range installedFiles {
				fmt.Printf("  - %s\n", file)
			}
		} else {
			fmt.Printf("No fonts were installed.\n")
		}
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringVar(&scope, "scope", "", "Installation scope (user or machine)")
}
