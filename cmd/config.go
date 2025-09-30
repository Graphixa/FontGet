package cmd

import (
	"fmt"
	"os/exec"
	"runtime"

	"fontget/internal/config"

	"errors"

	"github.com/fatih/color"
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
			// Validation is handled in PreRunE, so just return here
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

		// Show configuration information
		configPath := config.GetAppConfigPath()

		appConfig, err := config.LoadUserPreferences()
		if err != nil {
			if logger != nil {
				logger.Error("Failed to load config: %v", err)
			}
			return fmt.Errorf("failed to load config: %w", err)
		}

		fmt.Printf("Configuration Information:\n")
		fmt.Printf("  Location: %s\n", configPath)

		// Show editor information
		actualEditor := config.GetEditorWithFallback(appConfig)
		if appConfig.Configuration.DefaultEditor == "" {
			fmt.Printf("  Default Editor: %s (system default)\n", actualEditor)
			fmt.Printf("  Editor Note: To customize, edit config.yaml and uncomment a DefaultEditor line\n")
		} else {
			fmt.Printf("  Default Editor: %s (custom)\n", actualEditor)
		}

		fmt.Printf("  Log Path: %s\n", appConfig.Logging.LogPath)
		fmt.Printf("  Max Log Size: %s\n", appConfig.Logging.MaxSize)
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

		// Open configuration file in editor
		configPath := config.GetAppConfigPath()

		// Load current config to ensure it exists
		appConfig, err := config.LoadUserPreferences()
		if err != nil {
			if logger != nil {
				logger.Error("Failed to load config: %v", err)
			}
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Save config to ensure it exists on disk
		if err := config.SaveUserPreferences(appConfig); err != nil {
			if logger != nil {
				logger.Error("Failed to save config: %v", err)
			}
			return fmt.Errorf("failed to save config: %w", err)
		}

		// Get the editor to use
		editor := config.GetEditorWithFallback(appConfig)

		// Open the configuration file in the editor
		fmt.Printf("Opening config.yaml in %s...\n", editor)

		var execCmd *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			execCmd = exec.Command("cmd", "/c", "start", editor, configPath)
		case "darwin":
			execCmd = exec.Command("open", "-e", configPath)
		default: // Linux and others
			execCmd = exec.Command(editor, configPath)
		}

		if err := execCmd.Start(); err != nil {
			if logger != nil {
				logger.Error("Failed to open editor: %v", err)
			}
			return fmt.Errorf("failed to open editor '%s': %w", editor, err)
		}

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

	// Move validate logic to PreRunE
	configCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		validate, _ := cmd.Flags().GetBool("validate")
		resetDefaults, _ := cmd.Flags().GetBool("reset-defaults")

		if validate {
			logger := GetLogger()
			if logger != nil {
				logger.Info("Validating configuration file")
			}

			// If reset-defaults is specified, restore default config first
			if resetDefaults {
				logger := GetLogger()
				if logger != nil {
					logger.Info("Resetting configuration to defaults")
				}

				if err := config.GenerateInitialUserPreferences(); err != nil {
					if logger != nil {
						logger.Error("Failed to generate default config: %v", err)
					}
					return fmt.Errorf("failed to reset configuration: %w", err)
				}

				green := color.New(color.FgGreen).SprintFunc()
				fmt.Printf("%s\n", green("Configuration reset to defaults successfully"))
				return nil
			}

			appConfig, err := config.LoadUserPreferences()
			if err != nil {
				if logger != nil {
					logger.Error("Failed to load config: %v", err)
				}

				// Extract validation errors and format them nicely
				if validationErr, ok := err.(config.ValidationErrors); ok {
					red := color.New(color.FgRed).SprintFunc()
					fmt.Printf("%s\n", red("Your configuration file is malformed. Please fix the following problems with the file and run 'fontget config --validate' again to confirm."))
					fmt.Printf("%s\n", validationErr.Error())
					// Return nil to prevent help text from showing
					return nil
				}

				// Check if it's a wrapped validation error by using errors.As
				var validationErrors config.ValidationErrors
				if errors.As(err, &validationErrors) {
					red := color.New(color.FgRed).SprintFunc()
					fmt.Printf("%s\n", red("Your configuration file is malformed. Please fix the following problems with the file and run 'fontget config --validate' again to confirm."))
					fmt.Printf("%s\n", validationErrors.Error())
					// Return nil to prevent help text from showing
					return nil
				}

				// For other errors, show the error but don't show help
				red := color.New(color.FgRed).SprintFunc()
				fmt.Printf("%s\n", red(fmt.Sprintf("Configuration validation failed: %v", err)))
				return nil
			}

			if err := config.ValidateUserPreferences(appConfig); err != nil {
				red := color.New(color.FgRed).SprintFunc()
				if logger != nil {
					logger.Error("Configuration validation failed: %v", err)
				}
				fmt.Printf("%s\n", red(fmt.Sprintf("Configuration validation failed: %v", err)))
				return nil
			}

			green := color.New(color.FgGreen).SprintFunc()
			fmt.Printf("%s\n", green("Configuration file is valid"))
			return nil
		}
		return nil
	}
}
