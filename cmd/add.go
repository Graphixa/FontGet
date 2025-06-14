package cmd

import (
	"fmt"

	"fontget/internal/platform"
	"fontget/internal/repo"

	"github.com/spf13/cobra"
)

var (
	// addCmd represents the add command
	addCmd = &cobra.Command{
		Use:   "add <font ID> [flags]",
		Short: "Add a font to your system",
		Long: `Add a font to your system. The font ID should match the ID in the catalog.
For example:
  fontget add "roboto"
  fontget add "Open Sans"
  fontget add "Source Code Pro"

You can also specify multiple fonts at once:
  fontget add "roboto, firasans, notosans"

To see available fonts, use the 'list' command.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get repository
			repository, err := repo.GetRepository()
			if err != nil {
				return fmt.Errorf("failed to get repository: %v", err)
			}

			// Get manifest
			manifest, err := repository.GetManifest()
			if err != nil {
				return fmt.Errorf("failed to get font catalog: %v", err)
			}

			// Get font manager
			fontManager, err := platform.NewFontManager()
			if err != nil {
				return fmt.Errorf("failed to initialize font manager: %v", err)
			}

			// Process each font
			for _, fontName := range args {
				// Find font in manifest
				_, found := manifest.Sources["google-fonts"].Fonts[fontName]
				if !found {
					return fmt.Errorf("font '%s' not found in catalog. Use 'list' command to see available fonts", fontName)
				}

				// Get font files
				files, err := repo.GetFontFiles(fontName)
				if err != nil {
					return fmt.Errorf("failed to get font files for %s: %v", fontName, err)
				}

				// Install each font file
				for _, file := range files {
					if err := fontManager.InstallFont(file, platform.UserScope); err != nil {
						return fmt.Errorf("failed to install font file %s: %v", file, err)
					}
				}

				fmt.Printf("Successfully installed %s\n", fontName)
			}

			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(addCmd)
}
