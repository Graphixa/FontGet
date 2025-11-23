package cmd

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"

	"fontget/internal/components"
	"fontget/internal/config"
	"fontget/internal/output"
	"fontget/internal/ui"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage FontGet settings and configuration",
	Long: `Manage FontGet application configuration settings.

The config command allows you to view and edit the FontGet application configuration file (config.yaml).
This includes settings for the default editor, logging preferences, and other application behavior.`,
	Example: `  fontget config info              # Show configuration information
  fontget config edit              # Open config.yaml in default editor
  fontget config validate          # Validate configuration file`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no subcommand is provided, show help
		return cmd.Help()
	},
}

var configInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show configuration information",
	Long:  `Display detailed information about the current FontGet configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get logger after it's been initialized
		logger := GetLogger()
		if logger != nil {
			logger.Info("Starting config info operation")
		}

		output.GetVerbose().Info("Starting configuration information display")
		output.GetDebug().State("Config info command initiated")

		// Show configuration information
		configPath := config.GetAppConfigPath()
		output.GetVerbose().Info("Configuration file location: %s", configPath)
		output.GetDebug().State("Config path resolved: %s", configPath)

		appConfig, err := config.LoadUserPreferences()
		if err != nil {
			GetLogger().Error("Failed to load config: %v", err)
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("config.LoadUserPreferences() failed: %v", err)
			return fmt.Errorf("unable to load configuration: %v", err)
		}

		output.GetVerbose().Info("Configuration loaded successfully")
		output.GetDebug().State("Configuration loaded: %+v", appConfig)

		// Show editor information
		actualEditor := config.GetEditorWithFallback(appConfig)
		output.GetVerbose().Info("Resolved editor: %s", actualEditor)
		if appConfig.Configuration.DefaultEditor == "" {
			output.GetVerbose().Info("Using system default editor")
		} else {
			output.GetVerbose().Info("Using custom editor from configuration")
		}
		output.GetVerbose().Info("Logging configuration - Path: %s, MaxSize: %s", appConfig.Logging.LogPath, appConfig.Logging.MaxSize)
		output.GetDebug().State("Editor resolution - DefaultEditor: '%s', Actual: '%s'", appConfig.Configuration.DefaultEditor, actualEditor)
		output.GetDebug().State("Logging config: %+v", appConfig.Logging)

		// Display configuration information using card components
		fmt.Println() // Add space between command and first card
		var cards []components.Card

		// Configuration info card
		cards = append(cards, components.ConfigurationInfoCard(
			configPath,
			actualEditor,
			fmt.Sprintf("%t", appConfig.Configuration.UsePopularitySort),
		))

		// Logging configuration card
		cards = append(cards, components.LoggingConfigCard(
			appConfig.Logging.LogPath,
			appConfig.Logging.MaxSize,
			fmt.Sprintf("%d", appConfig.Logging.MaxFiles),
		))

		// Render all cards
		if len(cards) > 0 {
			cardModel := components.NewCardModel("", cards)
			fmt.Println(cardModel.Render())
		}
		return nil
	},
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open configuration file in default editor",
	Long:  `Open the FontGet configuration file (config.yaml) in your default editor.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get logger after it's been initialized
		logger := GetLogger()
		if logger != nil {
			logger.Info("Starting config edit operation")
		}

		output.GetVerbose().Info("Starting configuration file edit operation")
		output.GetDebug().State("Config edit command initiated")

		// Open configuration file in editor
		configPath := config.GetAppConfigPath()
		output.GetVerbose().Info("Configuration file path: %s", configPath)
		output.GetDebug().State("Config path resolved: %s", configPath)

		// Load current config to ensure it exists
		output.GetVerbose().Info("Loading current configuration")
		output.GetDebug().State("Loading user preferences from disk")

		appConfig, err := config.LoadUserPreferences()
		if err != nil {
			GetLogger().Error("Failed to load config: %v", err)
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("config.LoadUserPreferences() failed: %v", err)
			return fmt.Errorf("unable to load configuration: %v", err)
		}

		output.GetVerbose().Info("Configuration loaded successfully")
		output.GetDebug().State("Configuration loaded: %+v", appConfig)

		// Save config to ensure it exists on disk
		output.GetVerbose().Info("Ensuring configuration file exists on disk")
		output.GetDebug().State("Saving configuration to disk")

		if err := config.SaveUserPreferences(appConfig); err != nil {
			GetLogger().Error("Failed to save config: %v", err)
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("config.SaveUserPreferences() failed: %v", err)
			return fmt.Errorf("unable to save configuration: %v", err)
		}

		output.GetVerbose().Info("Configuration file saved successfully")
		output.GetDebug().State("Configuration file exists on disk")

		// Get the editor to use
		editor := config.GetEditorWithFallback(appConfig)
		output.GetVerbose().Info("Using editor: %s", editor)
		output.GetDebug().State("Editor resolved: %s", editor)

		// Open the configuration file in the editor
		fmt.Printf("Opening config.yaml in %s...\n", editor)

		output.GetVerbose().Info("Preparing to open editor with configuration file")
		output.GetDebug().State("Operating system: %s", runtime.GOOS)

		var execCmd *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			execCmd = exec.Command("cmd", "/c", "start", editor, configPath)
			output.GetDebug().State("Windows command: cmd /c start %s %s", editor, configPath)
		case "darwin":
			execCmd = exec.Command("open", "-e", configPath)
			output.GetDebug().State("macOS command: open -e %s", configPath)
		default: // Linux and others
			execCmd = exec.Command(editor, configPath)
			output.GetDebug().State("Linux/Unix command: %s %s", editor, configPath)
		}

		output.GetVerbose().Info("Executing editor command")
		output.GetDebug().State("Command: %+v", execCmd)

		if err := execCmd.Start(); err != nil {
			GetLogger().Error("Failed to open editor: %v", err)
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("execCmd.Start() failed: %v", err)
			return fmt.Errorf("unable to open editor '%s': %v", editor, err)
		}

		output.GetVerbose().Success("Editor opened successfully")
		output.GetDebug().State("Editor process started successfully")
		fmt.Printf("config.yaml opened in %s\n", editor)
		return nil
	},
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration file integrity",
	Long: `Validate the configuration file and report any issues.

If validation fails, use 'fontget config edit' to open and fix the configuration file.
If all else fails, use 'fontget config reset' to restore to default settings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		GetLogger().Info("Starting configuration validation operation")

		// Debug-level information for developers
		output.GetDebug().Message("Debug mode enabled - showing detailed diagnostic information")

		// Get configuration file path
		output.GetVerbose().Info("Getting configuration file path")
		output.GetDebug().State("Calling config.GetAppConfigPath()")
		configPath := config.GetAppConfigPath()
		output.GetVerbose().Info("Configuration file path: %s", configPath)
		output.GetDebug().State("Configuration file path: %s", configPath)

		// Start with a blank line for consistent spacing
		fmt.Println()
		fmt.Printf("Configuration Path: %s\n\n", configPath)

		// Load and validate configuration
		output.GetVerbose().Info("Loading configuration from: %s", configPath)
		output.GetDebug().State("Calling config.LoadUserPreferences()")
		appConfig, err := config.LoadUserPreferences()
		if err != nil {
			GetLogger().Error("Failed to load config: %v", err)
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("config.LoadUserPreferences() failed: %v", err)

			// Handle different types of validation errors
			if validationErr, ok := err.(config.ValidationErrors); ok {
				output.GetVerbose().Error("%v", validationErr)
				output.GetDebug().Error("Validation errors: %v", validationErr)
				fmt.Printf("  ✗ %s | %s\n", "config.yaml", ui.FeedbackError.Render("Invalid"))
				fmt.Printf("\n%s\n", ui.FeedbackError.Render("Configuration validation failed"))
				fmt.Printf("Your configuration file is malformed. Please fix the following problems:\n\n%s\n", validationErr.Error())
				return nil
			}

			// Check if it's a wrapped validation error
			var validationErrors config.ValidationErrors
			if errors.As(err, &validationErrors) {
				output.GetVerbose().Error("%v", validationErrors)
				output.GetDebug().Error("Wrapped validation errors: %v", validationErrors)
				fmt.Printf("  ✗ %s | %s\n", "config.yaml", ui.FeedbackError.Render("Invalid"))
				fmt.Printf("\n%s\n", ui.FeedbackError.Render("Configuration validation failed"))
				fmt.Printf("Your configuration file is malformed. Please fix the following problems:\n\n%s\n", validationErrors.Error())
				return nil
			}

			// For other errors
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("Unexpected validation error: %v", err)
			fmt.Printf("  ✗ %s | %s\n", "config.yaml", ui.FeedbackError.Render("Invalid"))
			fmt.Printf("\n%s\n", ui.FeedbackError.Render("Configuration validation failed"))
			fmt.Printf("Unexpected error: %v\n", err)
			return nil
		}

		output.GetVerbose().Info("Configuration file loaded successfully")
		output.GetDebug().State("Configuration loaded: %+v", appConfig)

		// Validate configuration structure and values
		output.GetVerbose().Info("Validating configuration structure and values")
		output.GetDebug().State("Calling config.ValidateUserPreferences()")
		if err := config.ValidateUserPreferences(appConfig); err != nil {
			GetLogger().Error("Configuration validation failed: %v", err)
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("config.ValidateUserPreferences() failed: %v", err)
			fmt.Printf("  ✗ %s | %s\n", "config.yaml", ui.FeedbackError.Render("Invalid"))
			fmt.Printf("\n%s\n", ui.FeedbackError.Render("Configuration validation failed"))
			fmt.Printf("Validation error: %v\n", err)
			return nil
		}

		// Success - show validation results
		output.GetVerbose().Success("Configuration validation completed successfully")
		output.GetDebug().State("Configuration validation process completed")
		fmt.Printf("  ✓ %s | %s\n", "config.yaml", ui.FeedbackSuccess.Render("Valid"))
		fmt.Printf("\n%s\n", ui.FeedbackSuccess.Render("Configuration file is valid"))

		GetLogger().Info("Configuration validation operation completed successfully")
		return nil
	},
}

var configResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset configuration to defaults",
	Long: `Reset the configuration file to default values.

Replaces the existing configuration file with defaults while preserving log files.
Useful when the configuration file is corrupted or you want to start fresh.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		GetLogger().Info("Starting configuration reset operation")

		// Debug-level information for developers
		output.GetDebug().Message("Debug mode enabled - showing detailed diagnostic information")

		// Get configuration file path
		output.GetVerbose().Info("Getting configuration file path")
		output.GetDebug().State("Calling config.GetAppConfigPath()")
		configPath := config.GetAppConfigPath()
		output.GetVerbose().Info("Configuration file path: %s", configPath)
		output.GetDebug().State("Configuration file path: %s", configPath)

		// Show confirmation dialog
		output.GetVerbose().Info("Showing confirmation dialog")
		output.GetDebug().State("Creating confirmation dialog for config reset")

		confirmed, err := components.RunConfirm(
			"Reset Configuration",
			"Are you sure you want to reset your configuration to defaults?\nThis will set all settings back to their default values.",
		)
		if err != nil {
			GetLogger().Error("Confirmation dialog failed: %v", err)
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("components.RunConfirm() failed: %v", err)
			fmt.Printf("%s\n", ui.FeedbackError.Render("Confirmation dialog failed\n"))
			return fmt.Errorf("unable to show confirmation dialog: %v", err)
		}

		if !confirmed {
			output.GetVerbose().Info("User cancelled configuration reset")
			output.GetDebug().State("User chose not to reset configuration")
			fmt.Printf("%s\n", ui.FeedbackWarning.Render("Configuration reset cancelled, no changes have been made.\n"))
			return nil
		}

		// User confirmed - proceed with reset
		output.GetVerbose().Info("User confirmed configuration reset")
		output.GetDebug().State("Proceeding with configuration reset")

		// Generate default configuration
		output.GetVerbose().Info("Generating default configuration")
		output.GetDebug().State("Calling config.GenerateInitialUserPreferences()")
		if err := config.GenerateInitialUserPreferences(); err != nil {
			GetLogger().Error("Failed to generate default config: %v", err)
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("config.GenerateInitialUserPreferences() failed: %v", err)
			fmt.Printf("%s\n", ui.FeedbackError.Render("Configuration reset failed\n"))
			fmt.Printf("Failed to generate default configuration: %v\n", err)
			return nil
		}

		// Success - show reset results
		output.GetVerbose().Success("Configuration reset completed successfully")
		output.GetDebug().State("Configuration reset process completed")
		fmt.Printf("%s\n", ui.FeedbackSuccess.Render("Configuration has been reset to defaults\n"))

		GetLogger().Info("Configuration reset operation completed successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)

	// Add subcommands
	configCmd.AddCommand(configInfoCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configResetCmd)
}
