package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"fontget/internal/components"
	"fontget/internal/config"
	"fontget/internal/output"
	"fontget/internal/ui"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// buildConfigCardsFromYAML builds cards dynamically from the YAML config file structure
func buildConfigCardsFromYAML(configPath string, _ *config.AppConfig, actualEditor string) ([]components.Card, error) {
	// Read the YAML file as raw map to preserve structure
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse as generic map to preserve YAML structure
	var rawData map[string]interface{}
	if err := yaml.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	var cards []components.Card

	// Define section order for consistent display
	sectionOrder := []string{"Configuration", "Logging", "Network", "Limits", "Update", "Theme"}

	// Build cards for each section in order
	for _, sectionName := range sectionOrder {
		sectionData, exists := rawData[sectionName]
		if !exists {
			continue
		}

		sectionMap, ok := sectionData.(map[string]interface{})
		if !ok {
			continue
		}

		// Build card sections from the YAML data
		var cardSections []components.CardSection

		// Special handling for Configuration section - add config path and resolved editor
		if sectionName == "Configuration" {
			cardSections = append(cardSections, components.CardSection{
				Label: "Location",
				Value: configPath,
			})
			cardSections = append(cardSections, components.CardSection{
				Label: "",
				Value: "", // Empty line for spacing
			})
		}

		// Get keys and sort them for consistent display
		keys := make([]string, 0, len(sectionMap))
		for key := range sectionMap {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		// Build sections from sorted keys
		for _, key := range keys {
			value := sectionMap[key]

			// Skip if this is Configuration section and we're processing DefaultEditor
			// (we'll use the resolved editor instead)
			if sectionName == "Configuration" && key == "DefaultEditor" {
				cardSections = append(cardSections, components.CardSection{
					Label: "Default Editor",
					Value: actualEditor,
				})
				continue
			}

			// Format the value as string
			valueStr := formatConfigValue(value)

			// Format the label (convert camelCase to Title Case)
			label := formatConfigLabel(key)

			cardSections = append(cardSections, components.CardSection{
				Label: label,
				Value: valueStr,
			})
		}

		// Create card for this section
		if len(cardSections) > 0 {
			card := components.NewCardWithSections(sectionName, cardSections)
			cards = append(cards, card)
		}
	}

	return cards, nil
}

// formatConfigValue formats a config value as a string for display
func formatConfigValue(value interface{}) string {
	if value == nil {
		return "(empty)"
	}

	switch v := value.(type) {
	case string:
		if v == "" {
			return "(empty)"
		}
		return v
	case bool:
		return strconv.FormatBool(v)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		// YAML numbers might be parsed as float64
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		// For other types, use reflection or string conversion
		return fmt.Sprintf("%v", v)
	}
}

// formatConfigLabel converts camelCase to Title Case
func formatConfigLabel(key string) string {
	// Handle common abbreviations and special cases
	replacements := map[string]string{
		"DefaultEditor":        "Default Editor",
		"EnablePopularitySort": "Enable Popularity Sort",
		"LogPath":              "Log Path",
		"MaxLogSize":           "Max Log Size",
		"MaxLogFiles":          "Max Log Files",
		"RequestTimeout":       "Request Timeout",
		"DownloadTimeout":      "Download Timeout",
		"MaxSourceFileSize":    "Max Source File Size",
		"FileCopyBufferSize":   "File Copy Buffer Size",
		"AutoCheck":            "Auto Check",
		"AutoUpdate":           "Auto Update",
		"UpdateCheckInterval":  "Update Check Interval",
		"LastUpdateCheck":      "Last Update Check",
		"UpdateChannel":        "Update Channel",
	}

	if replacement, exists := replacements[key]; exists {
		return replacement
	}

	// Convert camelCase to Title Case
	var result strings.Builder
	for i, r := range key {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteString(" ")
		}
		// Capitalize first letter, keep rest as-is
		if i == 0 && r >= 'a' && r <= 'z' {
			result.WriteRune(r - 32) // Convert to uppercase
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

var configCmd = &cobra.Command{
	Use:          "config",
	Short:        "Manage FontGet settings and configuration",
	SilenceUsage: true,
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
	Use:          "info",
	Short:        "Show configuration information",
	SilenceUsage: true,
	Long:         `Display detailed information about the current FontGet configuration.`,
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

		output.GetDebug().State("Calling config.LoadUserPreferences()")
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
		output.GetVerbose().Info("Logging configuration - Path: %s, MaxSize: %s", appConfig.Logging.LogPath, appConfig.Logging.MaxLogSize)
		output.GetDebug().State("Editor resolution - DefaultEditor: '%s', Actual: '%s'", appConfig.Configuration.DefaultEditor, actualEditor)
		output.GetDebug().State("Logging config: %+v", appConfig.Logging)

		// Build cards dynamically from YAML structure
		cards, err := buildConfigCardsFromYAML(configPath, appConfig, actualEditor)
		if err != nil {
			GetLogger().Error("Failed to build config cards: %v", err)
			output.GetVerbose().Error("Failed to build config cards: %v", err)
			output.GetDebug().Error("buildConfigCardsFromYAML() failed: %v", err)
			// Fallback to basic display
			return fmt.Errorf("unable to build configuration display: %v", err)
		}

		// Display configuration information using card components
		fmt.Println() // Add space between command and first card

		// Render all cards
		if len(cards) > 0 {
			cardModel := components.NewCardModel("", cards)
			fmt.Println(cardModel.Render())
		}
		return nil
	},
}

var configEditCmd = &cobra.Command{
	Use:          "edit",
	Short:        "Open configuration file in default editor",
	SilenceUsage: true,
	Long:         `Open the FontGet configuration file (config.yaml) in your default editor.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get logger after it's been initialized
		logger := GetLogger()
		if logger != nil {
			logger.Info("Starting config edit operation")
		}

		output.GetDebug().State("Starting config edit operation")
		output.GetVerbose().Info("Starting configuration file edit operation")

		// Open configuration file in editor
		configPath := config.GetAppConfigPath()
		output.GetVerbose().Info("Configuration file path: %s", configPath)
		output.GetDebug().State("Config path resolved: %s", configPath)

		// Try to load current config, but don't block if validation fails
		output.GetVerbose().Info("Loading current configuration")
		output.GetDebug().State("Loading user preferences from disk")

		appConfig, err := config.LoadUserPreferences()
		var editor string
		var validationErrors config.ValidationErrors

		if err != nil {
			// Check if this is a validation error by checking the error message
			// and manually validating to get the actual errors
			errMsg := err.Error()
			if strings.Contains(errMsg, "configuration validation failed") {
				// Manually validate the file to get the validation errors
				data, readErr := os.ReadFile(configPath)
				if readErr == nil {
					var rawData map[string]interface{}
					if yamlErr := yaml.Unmarshal(data, &rawData); yamlErr == nil {
						if valErr := config.ValidateStrictAppConfig(rawData); valErr != nil {
							if valErrs, ok := valErr.(config.ValidationErrors); ok {
								validationErrors = valErrs
							}
						}
					}
				}

				// Show validation errors but continue
				if len(validationErrors) > 0 {
					fmt.Println()
					fmt.Printf("%s\n", ui.WarningText.Render("Configuration validation issues found:"))
					for _, validationErr := range validationErrors {
						fmt.Printf("  - %s: %s\n", validationErr.Field, validationErr.Message)
					}
				} else {
					// Fallback if we couldn't extract errors
					fmt.Printf("%s\n", ui.ErrorText.Render(fmt.Sprintf("Configuration validation failed: %v", err)))
				}

				// Use default editor since we can't load the config
				editor = getDefaultEditorForOS()
			} else {
				// For non-validation errors (file doesn't exist, etc.), try to create it
				// If config file doesn't exist, create it with defaults
				if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
					output.GetVerbose().Info("Config file doesn't exist, creating default")
					if createErr := config.GenerateInitialUserPreferences(); createErr != nil {
						GetLogger().Error("Failed to create config: %v", createErr)
						output.GetVerbose().Error("%v", createErr)
						return fmt.Errorf("unable to create configuration file: %v", createErr)
					}
					// Try loading again after creation
					appConfig, err = config.LoadUserPreferences()
					if err != nil {
						// Still failed, use default editor
						editor = getDefaultEditorForOS()
					} else {
						editor = config.GetEditorWithFallback(appConfig)
					}
				} else {
					// File exists but has other errors, use default editor
					GetLogger().Error("Failed to load config: %v", err)
					output.GetVerbose().Error("%v", err)
					output.GetDebug().Error("config.LoadUserPreferences() failed: %v", err)
					fmt.Printf("%s\n", ui.WarningText.Render(fmt.Sprintf("Unable to load configuration: %v", err)))
					fmt.Printf("%s\n", ui.InfoText.Render("Opening config file in editor anyway..."))
					fmt.Println()
					editor = getDefaultEditorForOS()
				}
			}
		} else {
			output.GetVerbose().Info("Configuration loaded successfully")
			output.GetDebug().State("Configuration loaded: %+v", appConfig)

			// Save config to ensure it exists on disk (only if we successfully loaded it)
			output.GetVerbose().Info("Ensuring configuration file exists on disk")
			output.GetDebug().State("Saving configuration to disk")

			if err := config.SaveUserPreferences(appConfig); err != nil {
				GetLogger().Error("Failed to save config: %v", err)
				output.GetVerbose().Error("%v", err)
				output.GetDebug().Error("config.SaveUserPreferences() failed: %v", err)
				// Don't block on save errors, just continue to open editor
				output.GetVerbose().Info("Continuing despite save error")
			} else {
				output.GetVerbose().Info("Configuration file saved successfully")
				output.GetDebug().State("Configuration file exists on disk")
			}

			// Get the editor to use
			editor = config.GetEditorWithFallback(appConfig)
		}

		output.GetVerbose().Info("Using editor: %s", editor)
		output.GetDebug().State("Editor resolved: %s", editor)

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
		if len(validationErrors) > 0 {
			fmt.Println()
			fmt.Printf("%s\n", ui.InfoText.Render("After fixing the issues, run 'fontget config validate' to verify your changes."))
			fmt.Println()
		}
		output.GetDebug().State("Config operation complete")
		return nil
	},
}

// getDefaultEditorForOS returns the platform-specific default editor
// On Unix-like systems, checks the $EDITOR environment variable first
func getDefaultEditorForOS() string {
	switch runtime.GOOS {
	case "windows":
		return "notepad.exe"
	case "darwin":
		// macOS: check $EDITOR first, then fall back to TextEdit
		if editor := os.Getenv("EDITOR"); editor != "" {
			return editor
		}
		return "open -e"
	default: // Linux and others
		// Check $EDITOR environment variable first (standard Unix convention)
		if editor := os.Getenv("EDITOR"); editor != "" {
			return editor
		}
		// Fallback to common editors, trying vi first as it's more universally available
		// than nano on minimal systems
		return "vi"
	}
}

var configValidateCmd = &cobra.Command{
	Use:          "validate",
	Short:        "Validate configuration file integrity",
	SilenceUsage: true,
	Long: `Validate the configuration file and report any issues.

If validation fails, use 'fontget config edit' to open and fix the configuration file.
If all else fails, use 'fontget config reset' to restore to default settings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		GetLogger().Info("Starting configuration validation operation")

		output.GetDebug().State("Starting config validate operation")

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
				fmt.Printf("  ✗ %s | %s\n", "config.yaml", ui.ErrorText.Render("Invalid"))
				fmt.Printf("\n%s\n", ui.ErrorText.Render("Configuration validation failed"))
				fmt.Printf("Your configuration file is malformed. Please fix the following problems:\n\n%s\n", validationErr.Error())
				return nil
			}

			// Check if it's a wrapped validation error
			var validationErrors config.ValidationErrors
			if errors.As(err, &validationErrors) {
				output.GetVerbose().Error("%v", validationErrors)
				output.GetDebug().Error("Wrapped validation errors: %v", validationErrors)
				fmt.Printf("  ✗ %s | %s\n", "config.yaml", ui.ErrorText.Render("Invalid"))
				fmt.Printf("\n%s\n", ui.ErrorText.Render("Configuration validation failed"))
				fmt.Printf("Your configuration file is malformed. Please fix the following problems:\n\n%s\n", validationErrors.Error())
				return nil
			}

			// For other errors
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("Unexpected validation error: %v", err)
			fmt.Printf("  ✗ %s | %s\n", "config.yaml", ui.ErrorText.Render("Invalid"))
			fmt.Printf("\n%s\n", ui.ErrorText.Render("Configuration validation failed"))
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
			fmt.Printf("  ✗ %s | %s\n", "config.yaml", ui.ErrorText.Render("Invalid"))
			fmt.Printf("\n%s\n", ui.ErrorText.Render("Configuration validation failed"))
			fmt.Printf("Validation error: %v\n", err)
			return nil
		}

		// Success - show validation results
		output.GetVerbose().Success("Configuration validation completed successfully")
		output.GetDebug().State("Configuration validation process completed")
		fmt.Printf("  ✓ %s | %s\n", "config.yaml", ui.SuccessText.Render("Valid"))
		fmt.Printf("\n%s\n", ui.SuccessText.Render("Configuration file is valid"))

		GetLogger().Info("Configuration validation operation completed successfully")
		output.GetDebug().State("Config operation complete")
		return nil
	},
}

var configResetCmd = &cobra.Command{
	Use:          "reset",
	Short:        "Reset configuration to defaults",
	SilenceUsage: true,
	Long: `Reset the configuration file to default values.
Replaces the config with defaults while preserving log files.
Useful when the file is corrupted or you want to start fresh.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		GetLogger().Info("Starting configuration reset operation")

		output.GetDebug().State("Starting config reset operation")

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
			fmt.Printf("%s\n", ui.ErrorText.Render("Confirmation dialog failed"))
			return fmt.Errorf("unable to show confirmation dialog: %v", err)
		}

		if !confirmed {
			output.GetVerbose().Info("User cancelled configuration reset")
			output.GetDebug().State("User chose not to reset configuration")
			fmt.Printf("%s\n", ui.WarningText.Render("Configuration reset cancelled, no changes have been made."))
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
			fmt.Printf("%s\n", ui.ErrorText.Render("Configuration reset failed"))
			fmt.Printf("Failed to generate default configuration: %v\n", err)
			return nil
		}

		// Reset first-run state to trigger onboarding on next run
		output.GetVerbose().Info("Resetting first-run state")
		output.GetDebug().State("Calling config.ResetFirstRunState()")
		if err := config.ResetFirstRunState(); err != nil {
			GetLogger().Error("Failed to reset first-run state: %v", err)
			output.GetVerbose().Warning("Failed to reset first-run state: %v", err)
			output.GetDebug().Error("config.ResetFirstRunState() failed: %v", err)
			// Continue anyway - config reset is still successful
		}

		// Reset accepted sources to clear license acceptances
		output.GetVerbose().Info("Resetting accepted sources")
		output.GetDebug().State("Calling config.ResetAcceptedSources()")
		if err := config.ResetAcceptedSources(); err != nil {
			GetLogger().Error("Failed to reset accepted sources: %v", err)
			output.GetVerbose().Warning("Failed to reset accepted sources: %v", err)
			output.GetDebug().Error("config.ResetAcceptedSources() failed: %v", err)
			// Continue anyway - config reset is still successful
		}

		// Success - show reset results
		output.GetVerbose().Success("Configuration reset completed successfully")
		output.GetDebug().State("Configuration reset process completed")
		fmt.Printf("%s\n", ui.SuccessText.Render("Configuration has been reset to defaults."))
		fmt.Printf("%s\n", ui.InfoText.Render("You will be prompted to accept the license agreements the next time you run FontGet."))

		GetLogger().Info("Configuration reset operation completed successfully")
		output.GetDebug().State("Config operation complete")
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
