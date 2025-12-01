package cmd

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"fontget/internal/config"
	"fontget/internal/platform"
	"fontget/internal/ui"

	"github.com/spf13/cobra"
)

var debugCmd = &cobra.Command{
	Use:          "debug",
	Short:        "Debug and diagnostic commands",
	SilenceUsage: true,
	Long:         `Debug and diagnostic commands for troubleshooting FontGet.`,
}

var debugThemeCmd = &cobra.Command{
	Use:          "theme",
	Short:        "Test terminal theme detection",
	SilenceUsage: true,
	Long:         `Display detailed information about terminal theme detection, including detection method, RGB values, and configuration.`,
	Example:      `  fontget debug theme`,
	Args:         cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(ui.PageTitle.Render("Terminal Theme Detection Debug"))
		fmt.Println()

		// Get current config
		appConfig := config.GetUserPreferences()
		configMode := "auto"
		if appConfig != nil {
			configMode = appConfig.Theme.Mode
		}

		// Check environment variable
		envVar := "FONTGET_THEME_MODE"
		envValue := os.Getenv(envVar)

		fmt.Println(ui.InfoText.Render("Configuration:"))
		fmt.Printf("  Config setting (config.yaml): %s\n", configMode)
		if envValue != "" {
			fmt.Printf("  Environment variable (%s): %s\n", envVar, envValue)
		} else {
			fmt.Printf("  Environment variable (%s): (not set)\n", envVar)
		}
		fmt.Println()

		// Run detection with detailed output
		fmt.Println(ui.InfoText.Render("Detection Results:"))
		result, err := platform.TerminalThemeFromEnvOrDetect(envVar, 2*time.Second)
		if err != nil {
			fmt.Printf("  Final detection error: %v\n", err)
			fmt.Println()
			fmt.Println(ui.WarningText.Render("Detection failed. Falling back to default (dark mode)."))
			return nil
		}

		// Display detection method
		var kindStr string
		switch result.Kind {
		case platform.TerminalKindWin32:
			kindStr = "Win32 API (Legacy Windows Console)"
		case platform.TerminalKindXterm:
			kindStr = "OSC 11 Query (Windows Terminal / xterm-compatible)"
		case platform.TerminalKindOther:
			if envValue != "" {
				kindStr = "Environment Variable Override"
			} else {
				kindStr = "COLORFGBG Environment Variable"
			}
		case platform.TerminalKindUnknown:
			kindStr = "Unknown (Detection Failed)"
		default:
			kindStr = "Unknown"
		}

		fmt.Printf("  Detection method: %s\n", kindStr)
		fmt.Printf("  RGB values: R=%.3f, G=%.3f, B=%.3f\n", result.RGB.R, result.RGB.G, result.RGB.B)

		// Display detected theme
		var themeStr string
		switch result.Theme {
		case platform.TerminalThemeDark:
			themeStr = "dark"
		case platform.TerminalThemeLight:
			themeStr = "light"
		case platform.TerminalThemeUnknown:
			themeStr = "unknown"
		default:
			themeStr = "unknown"
		}

		fmt.Printf("  Detected theme: %s\n", themeStr)
		fmt.Println()

		// Show what FontGet will actually use
		fmt.Println(ui.InfoText.Render("FontGet Theme Selection:"))
		detectedTheme := ui.DetectTerminalTheme()
		fmt.Printf("  Current theme mode: %s\n", detectedTheme)
		fmt.Println()

		// Show environment details
		fmt.Println(ui.InfoText.Render("Environment Details:"))
		fmt.Printf("  OS: %s\n", runtime.GOOS)
		fmt.Printf("  TERM: %s\n", os.Getenv("TERM"))
		fmt.Printf("  WT_SESSION: %s\n", os.Getenv("WT_SESSION"))
		fmt.Printf("  TERM_PROGRAM: %s\n", os.Getenv("TERM_PROGRAM"))
		fmt.Printf("  COLORFGBG: %s\n", os.Getenv("COLORFGBG"))
		fmt.Println()

		// Show final result
		fmt.Println(ui.SuccessText.Render(fmt.Sprintf("✓ Terminal theme detection: %s", themeStr)))
		if configMode == "auto" && envValue == "" {
			fmt.Println(ui.Text.Render("  (Using auto-detection from terminal)"))
		} else if envValue != "" {
			fmt.Println(ui.Text.Render(fmt.Sprintf("  (Using environment variable override: %s)", envValue)))
		} else {
			fmt.Println(ui.Text.Render(fmt.Sprintf("  (Using config setting: %s)", configMode)))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(debugCmd)
	debugCmd.AddCommand(debugThemeCmd)
}
