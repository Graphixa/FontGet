package cmd

import (
	"fmt"
	"fontget/internal/config"
	"fontget/internal/license"
	"fontget/internal/logging"
	"fontget/internal/output"
	"fontget/internal/platform"
	"fontget/internal/shared"
	"fontget/internal/ui"
	"fontget/internal/update"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

// Version is now managed centrally in internal/version package

var (
	verbose bool
	debug   bool
	logs    bool
	logger  *logging.Logger
)

var rootCmd = &cobra.Command{
	Use:   "fontget <command> [flags]",
	Short: "A command-line tool for managing fonts",
	Long:  `FontGet is a powerful command-line font manager for installing and managing fonts on your system.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if logs {
			// Open logs directory
			logDir, err := logging.GetLogDirectory()
			if err != nil {
				return fmt.Errorf("failed to get log directory: %w", err)
			}

			// Create directory if it doesn't exist
			if err := os.MkdirAll(logDir, 0755); err != nil {
				return fmt.Errorf("failed to create log directory: %w", err)
			}

			// Open the directory using platform-specific method
			if err := platform.OpenDirectory(logDir); err != nil {
				return fmt.Errorf("failed to open logs directory: %w", err)
			}

			fmt.Printf("Opened logs directory: %s\n", logDir)
			return nil
		}

		return cmd.Help()
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize logger with appropriate level based on flags
		config := logging.DefaultConfig()

		// Debug flag enables full logging with timestamps to console
		if debug {
			config.Level = logging.DebugLevel
			config.ConsoleOutput = true // Show all logs to console in debug mode
		} else if verbose {
			// Verbose flag enables INFO level but no console logging (cleaner output)
			config.Level = logging.InfoLevel
			config.ConsoleOutput = false // Don't show regular logs in verbose mode
		} else {
			// Default mode: minimal logging, no console output
			config.Level = logging.ErrorLevel
			config.ConsoleOutput = false
		}

		var err error
		logger, err = logging.New(config)
		if err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}

		// Skip license check for certain commands
		skipLicenseCommands := map[string]bool{
			"help":       true,
			"completion": true,
		}

		if !skipLicenseCommands[cmd.Name()] {
			// Check first run and license acceptance
			if err := license.CheckFirstRunAndPrompt(); err != nil {
				return err
			}
		}

		// Perform startup update check (non-blocking)
		// Skip for certain commands to avoid unnecessary checks
		skipUpdateCheckCommands := map[string]bool{
			"help":       true,
			"completion": true,
			"update":     true, // Don't check for updates when running update command
		}

		if !skipUpdateCheckCommands[cmd.Name()] {
			go performStartupUpdateCheck()
		}

		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if logger != nil {
			return logger.Close()
		}
		return nil
	},
}

func init() {
	// Add verbose flag - shows detailed operation information including file/variant listings (user-friendly)
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed operation information including file/variant listings")

	// Add debug flag - shows full diagnostic logs with timestamps (for developers)
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Show debug logs with timestamps (for troubleshooting)")

	// Add logs flag
	rootCmd.PersistentFlags().BoolVar(&logs, "logs", false, "Open logs directory")

	// Inject flag checkers into output package to avoid circular imports
	output.SetVerboseChecker(IsVerbose)
	output.SetDebugChecker(IsDebug)

	// Set custom help template
	rootCmd.SetHelpTemplate(`{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if .Runnable}}Usage:
  {{.UseLine}}
{{end}}{{if .HasAvailableFlags}}
Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}
{{end}}{{if .HasExample}}
Examples:
{{.Example}}
{{end}}{{if .HasAvailableSubCommands}}
Available Commands:
{{range .Commands}}{{if .IsAvailableCommand}}  {{rpad .Name .NamePadding }} {{.Short}}
{{end}}{{end}}{{end}}`)

	// Set custom usage template with extra spacing
	rootCmd.SetUsageTemplate(`{{if .Runnable}}Usage:
  {{.UseLine}}
{{end}}{{if .HasAvailableFlags}}
Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}
{{end}}{{if .HasExample}}
Examples:
{{.Example}}
{{end}}{{if .HasAvailableSubCommands}}
Available Commands:
{{range .Commands}}{{if .IsAvailableCommand}}  {{rpad .Name .NamePadding }} {{.Short}}
{{end}}{{end}}
{{end}}`)

	// Add completion command
	completionCmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate completion script",
		Long: `Generate shell completion scripts.

Supports bash, zsh, and PowerShell. See documentation for installation instructions.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := args[0]
			switch shell {
			case "bash":
				return rootCmd.GenBashCompletion(cmd.OutOrStdout())
			case "zsh":
				return rootCmd.GenZshCompletion(cmd.OutOrStdout())
			case "powershell":
				return rootCmd.GenPowerShellCompletion(cmd.OutOrStdout())
			default:
				return fmt.Errorf("unsupported shell: %s", shell)
			}
		},
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"bash", "zsh", "powershell"},
	}
	rootCmd.AddCommand(completionCmd)
}

// Execute runs the root command
func Execute() error {
	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		// Force exit on interrupt
		os.Exit(1)
	}()

	err := rootCmd.Execute()
	if err != nil {
		// Check if it's our custom error type
		if _, ok := err.(*shared.FontInstallationError); ok {
			// Just return the error without showing help
			return err
		}
		// For other errors, let Cobra handle them
		return err
	}
	return nil
}

// GetLogger returns the global logger instance
func GetLogger() *logging.Logger {
	return logger
}

// IsVerbose returns true if verbose mode is enabled
func IsVerbose() bool {
	return verbose
}

// IsDebug returns true if debug mode is enabled
func IsDebug() bool {
	return debug
}

// performStartupUpdateCheck performs a non-blocking update check on startup
func performStartupUpdateCheck() {
	// Load configuration
	appConfig := config.GetUserPreferences()
	if appConfig == nil {
		return
	}

	// Perform check in background
	update.PerformStartupCheck(
		appConfig.Update.AutoCheck,
		appConfig.Update.CheckInterval,
		appConfig.Update.LastChecked,
		func(result *update.CheckResult) {
			// Update LastChecked timestamp in config
			appConfig.Update.LastChecked = update.GetLastCheckedTimestamp()
			if err := config.SaveUserPreferences(appConfig); err != nil {
				// Silently fail - don't interrupt user experience
				if logger != nil {
					logger.Error("Failed to save LastChecked timestamp: %v", err)
				}
			}

			// Show notification if update is available
			if result.UpdateAvailable {
				fmt.Printf("\n%s\n", ui.FeedbackInfo.Render(update.FormatUpdateNotification(result.CurrentVersion, result.LatestVersion)))
			}
		},
	)
}
