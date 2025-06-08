package cmd

import (
	"fmt"
	"os"

	"fontget/internal/platform"
	"fontget/internal/repo"

	"github.com/spf13/cobra"
)

const (
	fontsDir = "fonts"
)

var addCmd = &cobra.Command{
	Use:     "add [font-name]",
	Aliases: []string{"install"},
	Short:   "Install a font from Google Fonts",
	Long:    `Install a font from Google Fonts by querying the GitHub API and downloading the font files.`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
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
			if err := fontManager.InstallFont(path); err != nil {
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
}
