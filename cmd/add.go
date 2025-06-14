package cmd

import (
	"fmt"
	"runtime"
	"strings"

	"fontget/internal/errors"
	"fontget/internal/platform"
	"fontget/internal/repo"

	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <font-name>",
	Short: "Install a font from Google Fonts",
	Long: `Install a font from the Google Fonts repository.
You can specify the installation scope using the --scope flag:
  - user (default): Install for current user only
  - machine: Install system-wide (requires elevation)

Use --force to skip interactive prompts.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fontName := args[0]

		// Create font manager
		fontManager, err := platform.NewFontManager()
		if err != nil {
			return fmt.Errorf("failed to initialize font manager: %w", err)
		}

		// Convert scope string to InstallationScope
		installScope := platform.UserScope
		if scope != "user" {
			installScope = platform.InstallationScope(scope)
			if installScope != platform.UserScope && installScope != platform.MachineScope {
				return fmt.Errorf("invalid scope '%s'. Must be 'user' or 'machine'", scope)
			}
		}

		// Get font information from repository first
		fmt.Printf("Checking font '%s'...\n", fontName)
		fonts, err := repo.GetFont(fontName)
		if err != nil {
			return fmt.Errorf("failed to get font information: %w", err)
		}

		// Check if any of the font files are already installed in the requested scope
		fontDir := fontManager.GetFontDir(installScope)
		installedFonts, err := platform.ListInstalledFonts(fontDir)
		if err != nil {
			return fmt.Errorf("error checking installed fonts: %w", err)
		}

		// Create a map of installed fonts for faster lookup
		installedFontMap := make(map[string]bool)
		for _, installedFont := range installedFonts {
			installedFontMap[strings.ToLower(installedFont)] = true
		}

		// Check each font file
		var alreadyInstalled []string
		for _, font := range fonts {
			if installedFontMap[strings.ToLower(font.Name)] {
				alreadyInstalled = append(alreadyInstalled, font.Name)
			}
		}

		if len(alreadyInstalled) > 0 {
			if !force {
				return fmt.Errorf("Font files already installed in %s scope: %s", installScope, strings.Join(alreadyInstalled, ", "))
			}
			fmt.Printf("Font files already installed in %s scope: %s. Proceeding with installation as --force is set.\n",
				installScope, strings.Join(alreadyInstalled, ", "))
		}

		// If installing for user scope, check if it's already installed system-wide
		if installScope == platform.UserScope {
			systemFontDir := fontManager.GetFontDir(platform.MachineScope)
			systemFonts, err := platform.ListInstalledFonts(systemFontDir)
			if err != nil {
				return fmt.Errorf("error checking system fonts: %w", err)
			}

			// Create a map of system fonts for faster lookup
			systemFontMap := make(map[string]bool)
			for _, systemFont := range systemFonts {
				systemFontMap[strings.ToLower(systemFont)] = true
			}

			// Check each font file
			var systemInstalled []string
			for _, font := range fonts {
				if systemFontMap[strings.ToLower(font.Name)] {
					systemInstalled = append(systemInstalled, font.Name)
				}
			}

			if len(systemInstalled) > 0 {
				if !force {
					return fmt.Errorf("Font files already available system-wide: %s. No need to install for user",
						strings.Join(systemInstalled, ", "))
				}
				fmt.Printf("Font files already available system-wide: %s. Proceeding with user installation as --force is set.\n",
					strings.Join(systemInstalled, ", "))
			}
		}

		// Check if elevation is required
		if fontManager.RequiresElevation(installScope) {
			elevated, err := fontManager.IsElevated()
			if err != nil {
				return fmt.Errorf("failed to check elevation status: %w", err)
			}
			if !elevated {
				errors.PrintElevationHelp(cmd, runtime.GOOS)
				return errors.ElevationRequired(runtime.GOOS)
			}
		}

		// Get temporary fonts directory
		tempFontsDir, err := platform.GetTempFontsDir()
		if err != nil {
			return fmt.Errorf("failed to get temporary directory: %w", err)
		}
		defer platform.CleanupTempFontsDir()

		// Download font files
		fmt.Printf("Downloading font '%s'...\n", fontName)
		var downloadedPaths []string
		for _, font := range fonts {
			path, err := repo.DownloadFont(&font, tempFontsDir)
			if err != nil {
				return fmt.Errorf("failed to download font: %w", err)
			}
			downloadedPaths = append(downloadedPaths, path)
		}

		// Install the font
		fmt.Printf("Installing font '%s' to %s scope...\n", fontName, installScope)
		for _, fontPath := range downloadedPaths {
			if err := fontManager.InstallFont(fontPath, installScope); err != nil {
				return fmt.Errorf("failed to install font: %w", err)
			}
		}

		fmt.Printf("Successfully installed font '%s' to %s scope\n", fontName, installScope)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringVar(&scope, "scope", "user", "Installation scope (user or machine)")
	addCmd.Flags().BoolVarP(&force, "force", "f", false, "Skip interactive prompts")
}
