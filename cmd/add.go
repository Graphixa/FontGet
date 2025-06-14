package cmd

import (
	"fmt"
	"strings"

	"fontget/internal/platform"
	"fontget/internal/repo"

	"github.com/fatih/color"
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
				// Split by comma if multiple fonts
				fontNames := strings.Split(fontName, ",")
				for _, name := range fontNames {
					name = strings.TrimSpace(name)
					if name == "" {
						continue
					}

					// Find font in manifest
					var fontInfo *repo.FontInfo
					var sourceName string
					for _, source := range manifest.Sources {
						if info, found := source.Fonts[name]; found {
							fontInfo = &info
							sourceName = source.Name
							break
						}
					}

					if fontInfo == nil {
						red := color.New(color.FgRed).SprintFunc()
						fmt.Printf("%s: Font '%s' not found in catalog. Use 'list' command to see available fonts\n", red("Error"), name)
						continue
					}

					// Get font files
					files, err := repo.GetFontFiles(fontInfo.Name)
					if err != nil {
						red := color.New(color.FgRed).SprintFunc()
						fmt.Printf("%s: Failed to get font files for %s: %v\n", red("Error"), name, err)
						continue
					}

					// Show progress
					yellow := color.New(color.FgYellow).SprintFunc()
					fmt.Printf("Installing %s from %s...\n", yellow(fontInfo.Name), yellow(sourceName))

					// Install each font file
					success := true
					for variant, file := range files {
						if err := fontManager.InstallFont(file, platform.UserScope); err != nil {
							red := color.New(color.FgRed).SprintFunc()
							fmt.Printf("%s: Failed to install %s variant: %v\n", red("Error"), variant, err)
							success = false
						}
					}

					if success {
						green := color.New(color.FgGreen).SprintFunc()
						fmt.Printf("%s: Successfully installed %s\n", green("Success"), fontInfo.Name)
					}
				}
			}

			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(addCmd)
}
