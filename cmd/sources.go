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

var sourcesCmd = &cobra.Command{
	Use:   "sources",
	Short: "Manage FontGet font sources",
	Long: `Manage FontGet font sources configuration.

The sources command allows you to view and edit the FontGet sources configuration file (sources.json).
This includes managing font repositories, their priorities, and enable/disable states.`,
	Example: `  fontget sources info              # Show sources information
  fontget sources edit              # Open sources.json in default editor
  fontget sources --validate        # Validate sources configuration
  fontget sources add               # Add a new font source
  fontget sources remove <name>     # Remove a font source`,
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

var sourcesInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show sources information",
	Long:  `Display detailed information about the current FontGet sources configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get logger after it's been initialized
		logger := GetLogger()
		if logger != nil {
			logger.Info("Starting sources info operation")
		}

		// Show sources information
		sourcesPath, err := config.GetSourcesConfigPath()
		if err != nil {
			if logger != nil {
				logger.Error("Failed to get sources path: %v", err)
			}
			return fmt.Errorf("failed to get sources path: %w", err)
		}

		sourcesConfig, err := config.LoadSourcesConfig()
		if err != nil {
			if logger != nil {
				logger.Error("Failed to load sources config: %v", err)
			}
			return fmt.Errorf("failed to load sources config: %w", err)
		}

		fmt.Printf("Sources Information:\n")
		fmt.Printf("  File: %s\n", sourcesPath)
		fmt.Printf("  Total Sources: %d\n", len(sourcesConfig.Sources))

		enabledSources := config.GetEnabledSources(sourcesConfig)
		fmt.Printf("  Enabled Sources: %d\n", len(enabledSources))

		if len(enabledSources) > 0 {
			fmt.Printf("  Enabled Source List:\n")
			for i, name := range enabledSources {
				if source, exists := config.GetSourceByName(sourcesConfig, name); exists {
					fmt.Printf("    %d. %s (%s)\n", i+1, name, source.Prefix)
				}
			}
		}

		// Show disabled sources
		var disabledSources []string
		for name, source := range sourcesConfig.Sources {
			if !source.Enabled {
				disabledSources = append(disabledSources, name)
			}
		}

		if len(disabledSources) > 0 {
			fmt.Printf("  Disabled Sources: %d\n", len(disabledSources))
			for i, name := range disabledSources {
				if source, exists := config.GetSourceByName(sourcesConfig, name); exists {
					fmt.Printf("    %d. %s (%s)\n", i+1, name, source.Prefix)
				}
			}
		}

		return nil
	},
}

var sourcesEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open sources configuration file in default editor",
	Long:  `Open the FontGet sources configuration file (sources.json) in your default editor.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get logger after it's been initialized
		logger := GetLogger()
		if logger != nil {
			logger.Info("Starting sources edit operation")
		}

		// Open sources configuration file in editor
		sourcesPath, err := config.GetSourcesConfigPath()
		if err != nil {
			if logger != nil {
				logger.Error("Failed to get sources path: %v", err)
			}
			return fmt.Errorf("failed to get sources path: %w", err)
		}

		// Load current config to ensure it exists
		sourcesConfig, err := config.LoadSourcesConfig()
		if err != nil {
			if logger != nil {
				logger.Error("Failed to load sources config: %v", err)
			}
			return fmt.Errorf("failed to load sources config: %w", err)
		}

		// Save config to ensure it exists on disk
		if err := config.SaveSourcesConfig(sourcesConfig); err != nil {
			if logger != nil {
				logger.Error("Failed to save sources config: %v", err)
			}
			return fmt.Errorf("failed to save sources config: %w", err)
		}

		// Get the editor to use from main config
		yamlConfig, err := config.LoadYAMLConfig()
		if err != nil {
			if logger != nil {
				logger.Error("Failed to load YAML config for editor: %v", err)
			}
			return fmt.Errorf("failed to load YAML config: %w", err)
		}

		editor := config.GetEditorWithFallback(yamlConfig)

		// Open the sources configuration file in the editor
		fmt.Printf("Opening sources.json in %s...\n", editor)

		var execCmd *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			execCmd = exec.Command("cmd", "/c", "start", editor, sourcesPath)
		case "darwin":
			execCmd = exec.Command("open", "-e", sourcesPath)
		default: // Linux and others
			execCmd = exec.Command(editor, sourcesPath)
		}

		if err := execCmd.Start(); err != nil {
			if logger != nil {
				logger.Error("Failed to open editor: %v", err)
			}
			return fmt.Errorf("failed to open editor '%s': %w", editor, err)
		}

		fmt.Printf("sources.json opened in %s\n", editor)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(sourcesCmd)

	// Add subcommands
	sourcesCmd.AddCommand(sourcesInfoCmd)
	sourcesCmd.AddCommand(sourcesEditCmd)

	// Add validate flag (no alias to avoid conflicts)
	sourcesCmd.Flags().Bool("validate", false, "Validate sources configuration")
	// Add reset-defaults flag
	sourcesCmd.Flags().Bool("reset-defaults", false, "Reset sources configuration to defaults (use with --validate)")

	// Move validate logic to PreRunE
	sourcesCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		validate, _ := cmd.Flags().GetBool("validate")
		resetDefaults, _ := cmd.Flags().GetBool("reset-defaults")

		if validate {
			logger := GetLogger()
			if logger != nil {
				logger.Info("Validating sources configuration")
			}

			sourcesConfig, err := config.LoadSourcesConfig()
			if err != nil {
				if logger != nil {
					logger.Error("Failed to load sources config: %v", err)
				}

				// If reset-defaults is specified, restore default config
				if resetDefaults {
					logger := GetLogger()
					if logger != nil {
						logger.Info("Resetting sources configuration to defaults")
					}

					defaultConfig := config.DefaultSourcesConfig()
					if err := config.SaveSourcesConfig(defaultConfig); err != nil {
						if logger != nil {
							logger.Error("Failed to save default sources config: %v", err)
						}
						return fmt.Errorf("failed to reset sources configuration: %w", err)
					}

					green := color.New(color.FgGreen).SprintFunc()
					fmt.Printf("%s\n", green("Sources configuration reset to defaults successfully"))
					return nil
				}

				// Extract validation errors and format them nicely
				if validationErr, ok := err.(config.ValidationErrors); ok {
					red := color.New(color.FgRed).SprintFunc()
					fmt.Printf("%s\n", red("Your sources configuration file is malformed. Please fix the following problems with the file and run 'fontget sources --validate' again to confirm."))
					fmt.Printf("%s\n", validationErr.Error())
					// Return nil to prevent help text from showing
					return nil
				}

				// Check if it's a wrapped validation error by using errors.As
				var validationErrors config.ValidationErrors
				if errors.As(err, &validationErrors) {
					red := color.New(color.FgRed).SprintFunc()
					fmt.Printf("%s\n", red("Your sources configuration file is malformed. Please fix the following problems with the file and run 'fontget sources --validate' again to confirm."))
					fmt.Printf("%s\n", validationErrors.Error())
					// Return nil to prevent help text from showing
					return nil
				}

				// For other errors, show the error but don't show help
				red := color.New(color.FgRed).SprintFunc()
				fmt.Printf("%s\n", red(fmt.Sprintf("Sources configuration validation failed: %v", err)))
				return nil
			}

			if err := config.ValidateSourcesConfig(sourcesConfig); err != nil {
				red := color.New(color.FgRed).SprintFunc()
				if logger != nil {
					logger.Error("Sources configuration validation failed: %v", err)
				}
				fmt.Printf("%s\n", red(fmt.Sprintf("Sources configuration validation failed: %v", err)))
				return nil
			}

			green := color.New(color.FgGreen).SprintFunc()
			fmt.Printf("%s\n", green("Sources configuration is valid"))
			return nil
		}
		return nil
	}
}
