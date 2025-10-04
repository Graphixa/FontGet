package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"fontget/internal/config"
	"fontget/internal/output"
	"fontget/internal/ui"

	"errors"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage FontGet application configuration",
	Long: `Manage FontGet application configuration settings.

The config command allows you to view and edit the FontGet application configuration file (config.yaml).
This includes settings for the default editor, logging preferences, and other application behavior.`,
	Example: `  fontget config info              # Show configuration information
  fontget config edit              # Open config.yaml in default editor
  fontget config --validate        # Validate configuration file`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if --validate flag is set
		validate, _ := cmd.Flags().GetBool("validate")
		if validate {
			// Run validation logic here
			logger := GetLogger()
			if logger != nil {
				logger.Info("Validating configuration file")
			}

			output.GetVerbose().Info("Starting configuration validation")
			output.GetDebug().State("Validation process initiated")

			configPath := config.GetAppConfigPath()
			output.GetVerbose().Info("Loading configuration from: %s", configPath)
			output.GetDebug().State("Configuration file path: %s", configPath)

			appConfig, err := config.LoadUserPreferences()
			if err != nil {
				if logger != nil {
					logger.Error("Failed to load config: %v", err)
				}

				output.GetVerbose().Error("Failed to load configuration file")
				output.GetDebug().Error("Configuration load error: %v", err)

				// Extract validation errors and format them nicely
				if validationErr, ok := err.(config.ValidationErrors); ok {
					output.GetVerbose().Error("Configuration file contains validation errors")
					output.GetDebug().Error("Validation errors: %v", validationErr)
					fmt.Printf("%s\n", ui.RenderError("Your configuration file is malformed. Please fix the following problems with the file and run 'fontget config --validate' again to confirm."))
					fmt.Printf("%s\n", validationErr.Error())
					return nil
				}

				// Check if it's a wrapped validation error by using errors.As
				var validationErrors config.ValidationErrors
				if errors.As(err, &validationErrors) {
					output.GetVerbose().Error("Configuration file contains wrapped validation errors")
					output.GetDebug().Error("Wrapped validation errors: %v", validationErrors)
					fmt.Printf("%s\n", ui.RenderError("Your configuration file is malformed. Please fix the following problems with the file and run 'fontget config --validate' again to confirm."))
					fmt.Printf("%s\n", validationErrors.Error())
					return nil
				}

				// For other errors, show the error but don't show help
				output.GetVerbose().Error("Configuration validation failed with unexpected error")
				output.GetDebug().Error("Unexpected validation error: %v", err)
				fmt.Printf("%s\n", ui.RenderError(fmt.Sprintf("Configuration validation failed: %v", err)))
				return nil
			}

			output.GetVerbose().Info("Configuration file loaded successfully")
			output.GetDebug().State("Configuration loaded: %+v", appConfig)

			output.GetVerbose().Info("Validating configuration structure and values")
			output.GetDebug().State("Running ValidateUserPreferences on loaded config")

			if err := config.ValidateUserPreferences(appConfig); err != nil {
				if logger != nil {
					logger.Error("Configuration validation failed: %v", err)
				}
				output.GetVerbose().Error("Configuration validation failed")
				output.GetDebug().Error("Validation error: %v", err)
				fmt.Printf("%s\n", ui.RenderError(fmt.Sprintf("Configuration validation failed: %v", err)))
				return nil
			}

			output.GetVerbose().Success("Configuration validation completed successfully")
			output.GetDebug().State("Configuration validation process completed")
			fmt.Printf("%s\n", ui.RenderSuccess("Configuration file is valid"))
			return nil
		}

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
			if logger != nil {
				logger.Error("Failed to load config: %v", err)
			}
			output.GetVerbose().Error("Failed to load configuration file")
			output.GetDebug().Error("Configuration load error: %v", err)
			return fmt.Errorf("failed to load config: %w", err)
		}

		output.GetVerbose().Info("Configuration loaded successfully")
		output.GetDebug().State("Configuration loaded: %+v", appConfig)

		fmt.Printf("Configuration Information:\n")
		fmt.Printf("  Location: %s\n", configPath)

		// Show editor information
		actualEditor := config.GetEditorWithFallback(appConfig)
		output.GetVerbose().Info("Resolved editor: %s", actualEditor)
		output.GetDebug().State("Editor resolution - DefaultEditor: '%s', Actual: '%s'", appConfig.Configuration.DefaultEditor, actualEditor)

		if appConfig.Configuration.DefaultEditor == "" {
			fmt.Printf("  Default Editor: %s (system default)\n", actualEditor)
			fmt.Printf("  Editor Note: To customize, edit config.yaml and uncomment a DefaultEditor line\n")
			output.GetVerbose().Info("Using system default editor")
		} else {
			fmt.Printf("  Default Editor: %s (custom)\n", actualEditor)
			output.GetVerbose().Info("Using custom editor from configuration")
		}

		fmt.Printf("  Log Path: %s\n", appConfig.Logging.LogPath)
		fmt.Printf("  Max Log Size: %s\n", appConfig.Logging.MaxSize)

		output.GetVerbose().Info("Logging configuration - Path: %s, MaxSize: %s", appConfig.Logging.LogPath, appConfig.Logging.MaxSize)
		output.GetDebug().State("Logging config: %+v", appConfig.Logging)
		fmt.Printf("  Max Log Files: %d\n", appConfig.Logging.MaxFiles)
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
			if logger != nil {
				logger.Error("Failed to load config: %v", err)
			}
			output.GetVerbose().Error("Failed to load configuration file")
			output.GetDebug().Error("Configuration load error: %v", err)
			return fmt.Errorf("failed to load config: %w", err)
		}

		output.GetVerbose().Info("Configuration loaded successfully")
		output.GetDebug().State("Configuration loaded: %+v", appConfig)

		// Save config to ensure it exists on disk
		output.GetVerbose().Info("Ensuring configuration file exists on disk")
		output.GetDebug().State("Saving configuration to disk")

		if err := config.SaveUserPreferences(appConfig); err != nil {
			if logger != nil {
				logger.Error("Failed to save config: %v", err)
			}
			output.GetVerbose().Error("Failed to save configuration file")
			output.GetDebug().Error("Configuration save error: %v", err)
			return fmt.Errorf("failed to save config: %w", err)
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
			if logger != nil {
				logger.Error("Failed to open editor: %v", err)
			}
			output.GetVerbose().Error("Failed to start editor process")
			output.GetDebug().Error("Editor execution error: %v", err)
			return fmt.Errorf("failed to open editor '%s': %w", editor, err)
		}

		output.GetVerbose().Success("Editor opened successfully")
		output.GetDebug().State("Editor process started successfully")
		fmt.Printf("config.yaml opened in %s\n", editor)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)

	// Add subcommands
	configCmd.AddCommand(configInfoCmd)
	configCmd.AddCommand(configEditCmd)

	// Add validate flag (no alias to avoid conflicts)
	configCmd.Flags().Bool("validate", false, "Validate configuration file")
	// Add reset-defaults flag
	configCmd.Flags().Bool("reset-defaults", false, "Reset configuration to defaults (use with --validate)")

	// Handle reset-defaults logic in PreRunE
	configCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		resetDefaults, _ := cmd.Flags().GetBool("reset-defaults")

		if resetDefaults {
			logger := GetLogger()
			if logger != nil {
				logger.Info("Resetting configuration to defaults")
			}

			output.GetVerbose().Info("Starting configuration reset to defaults")
			output.GetDebug().State("Reset defaults command initiated")

			output.GetVerbose().Info("Generating initial user preferences")
			output.GetDebug().State("Calling config.GenerateInitialUserPreferences()")

			if err := config.GenerateInitialUserPreferences(); err != nil {
				if logger != nil {
					logger.Error("Failed to generate default config: %v", err)
				}
				output.GetVerbose().Error("Failed to generate default configuration")
				output.GetDebug().Error("Configuration generation error: %v", err)
				return fmt.Errorf("failed to reset configuration: %w", err)
			}

			output.GetVerbose().Success("Configuration reset to defaults completed successfully")
			output.GetDebug().State("Configuration reset process completed")
			fmt.Printf("%s\n", ui.RenderSuccess("Configuration reset to defaults successfully"))
			// Exit early to prevent help from showing
			os.Exit(0)
		}
		return nil
	}
}
