package cmd

import (
	"errors"
	"fmt"
	"fontget/internal/config"
	"fontget/internal/logging"
	"fontget/internal/onboarding"
	"fontget/internal/output"
	"fontget/internal/platform"
	"fontget/internal/shared"
	"fontget/internal/ui"
	"fontget/internal/update"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

// Version is now managed centrally in internal/version package

var (
	verbose bool
	debug   bool
	logs    bool
	wizard  bool
	logger  *logging.Logger
)

var rootCmd = &cobra.Command{
	Use:   "fontget <command> [options]",
	Short: "A command-line tool for managing fonts",
	Long:  `FontGet is a powerful command-line font manager for installing and managing fonts on your system.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if wizard {
			// Run the wizard
			if err := onboarding.RunWizard(); err != nil {
				// Handle cancellation/incomplete gracefully
				if errors.Is(err, shared.ErrOnboardingCancelled) || errors.Is(err, shared.ErrOnboardingIncomplete) {
					os.Exit(0)
				}
				return err
			}
			return nil
		}

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
		// Load user preferences to get logging config
		appConfig := config.GetUserPreferences()

		// Initialize logger with appropriate level based on flags
		logConfig := logging.DefaultConfig()

		// Debug flag enables full logging with timestamps to console
		if debug {
			logConfig.Level = logging.DebugLevel
			logConfig.ConsoleOutput = true // Show all logs to console in debug mode
		} else if verbose {
			// Verbose flag enables INFO level but no console logging (cleaner output)
			logConfig.Level = logging.InfoLevel
			logConfig.ConsoleOutput = false // Don't show regular logs in verbose mode
		} else {
			// Default mode: Log Info/Warn/Error to file (standard CLI behavior)
			// Console output is minimal (errors only), but file has full audit trail
			logConfig.Level = logging.InfoLevel
			logConfig.ConsoleOutput = false
		}

		// Apply logging config from config.yaml if available
		if appConfig != nil {
			// Parse MaxLogSize from config (e.g., "10MB" -> 10)
			if appConfig.Logging.MaxLogSize != "" {
				maxSize, err := config.ParseMaxSize(appConfig.Logging.MaxLogSize)
				if err == nil {
					logConfig.MaxSize = maxSize
				}
			}

			// Use MaxLogFiles from config
			if appConfig.Logging.MaxLogFiles > 0 {
				logConfig.MaxBackups = appConfig.Logging.MaxLogFiles
			}
		}

		var err error
		// Use LogPath from config if available, otherwise use default
		if appConfig != nil && appConfig.Logging.LogPath != "" {
			expandedLogPath, expandErr := config.ExpandLogPath(appConfig.Logging.LogPath)
			if expandErr == nil {
				logger, err = logging.NewWithPath(logConfig, expandedLogPath)
				if err != nil {
					// Fallback to default if NewWithPath fails
					logger, err = logging.New(logConfig)
				}
			} else {
				// Fallback to default if expansion fails
				logger, err = logging.New(logConfig)
			}
		} else {
			// Use default log path
			logger, err = logging.New(logConfig)
		}

		if err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}

		// Initialize theme system (non-blocking - uses fast timeout)
		// Theme detection happens quickly and doesn't delay command execution
		if err := ui.InitThemeManager(); err != nil {
			// Non-fatal: continue with default theme if detection fails
			// This prevents theme detection issues from breaking commands
		}

		// Initialize styles based on theme
		if err := ui.InitStyles(); err != nil {
			// Non-fatal: continue with default styles if initialization fails
			// This prevents theme issues from breaking commands
		}

		// Skip license check for certain commands
		skipLicenseCommands := map[string]bool{
			"help":       true,
			"version":    true, // Version command doesn't need license acceptance
			"completion": true,
		}

		// Skip first-run onboarding if --wizard flag is set (wizard will be run in RunE)
		if !wizard && !skipLicenseCommands[cmd.Name()] {
			// Run first-run onboarding (welcome, license, settings)
			if err := onboarding.RunFirstRunOnboarding(); err != nil {
				// If onboarding was cancelled or incomplete, exit gracefully
				// The first run status remains false, so onboarding will restart on next command
				// Don't show error message for cancellation - it's expected behavior
				if errors.Is(err, shared.ErrOnboardingCancelled) || errors.Is(err, shared.ErrOnboardingIncomplete) {
					// Exit silently - user cancelled, will be prompted again next time
					os.Exit(0)
				}
				// For other errors, return them normally
				return err
			}
		}

		// Perform startup update check (non-blocking)
		// Skip for certain commands to avoid unnecessary checks
		skipUpdateCheckCommands := map[string]bool{
			"help":       true,
			"version":    true, // Don't check for updates when just checking version
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

	// Add logs flag (non-persistent - only available on root command)
	rootCmd.Flags().BoolVar(&logs, "logs", false, "Open logs directory")

	// Add wizard flag (not persistent - only applies to root command)
	rootCmd.Flags().BoolVar(&wizard, "wizard", false, "Run the setup wizard to configure FontGet")

	// Inject flag checkers into output package to avoid circular imports
	output.SetVerboseChecker(IsVerbose)
	output.SetDebugChecker(IsDebug)

	// Register custom template function to replace "[flags]" with "options"
	cobra.AddTemplateFunc("replaceFlags", func(s string) string {
		return strings.ReplaceAll(s, "[flags]", "[options]")
	})

	// Set custom help template
	rootCmd.SetHelpTemplate(`{{if .Runnable}}
Usage: {{replaceFlags .UseLine}}
{{end}}{{with (or .Long .Short)}}
{{. | trimTrailingWhitespaces}}
{{end}}{{if .HasAvailableSubCommands}}
Available Commands:
{{range .Commands}}{{if .IsAvailableCommand}}  {{rpad .Name .NamePadding }} {{.Short}}
{{end}}{{end}}{{end}}{{if or .HasAvailableLocalFlags .HasAvailableInheritedFlags}}
Options:
{{if .HasAvailableLocalFlags}}{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}
{{end}}{{if .HasAvailableInheritedFlags}}{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}
{{end}}{{end}}{{if .HasExample}}
Examples:
{{.Example}}
{{end}}
`)

	// Set custom usage template with extra spacing
	rootCmd.SetUsageTemplate(`{{if .Runnable}}
Usage: {{replaceFlags .UseLine}}
{{end}}{{if .HasAvailableSubCommands}}
Available Commands:
{{range .Commands}}{{if .IsAvailableCommand}}  {{rpad .Name .NamePadding }} {{.Short}}
{{end}}{{end}}
{{end}}{{if or .HasAvailableLocalFlags .HasAvailableInheritedFlags}}
Options:
{{if .HasAvailableLocalFlags}}{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}
{{end}}{{if .HasAvailableInheritedFlags}}{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}
{{end}}{{end}}{{if .HasExample}}
Examples:
{{.Example}}
{{end}}`)

	// Completion command is now in cmd/completion.go
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
		appConfig.Update.UpdateCheckInterval,
		appConfig.Update.LastUpdateCheck,
		func(result *update.CheckResult) {
			// Update LastUpdateCheck timestamp in config (UTC)
			appConfig.Update.LastUpdateCheck = update.GetLastCheckedTimestamp()
			if err := config.SaveUserPreferences(appConfig); err != nil {
				// Silently fail - don't interrupt user experience
				if logger != nil {
					logger.Error("Failed to save LastUpdateCheck timestamp: %v", err)
				}
			}

			// Show notification if update is available
			if result.UpdateAvailable {
				// Check if AutoUpdate is enabled
				if appConfig.Update.AutoUpdate {
					// Auto-install update
					if logger != nil {
						logger.Info("AutoUpdate enabled - automatically installing update from %s to %s", result.CurrentVersion, result.LatestVersion)
					}
					// Perform update in background (non-blocking)
					go func() {
						err := update.UpdateToLatest()
						if err != nil {
							if logger != nil {
								logger.Error("Auto-update failed: %v", err)
							}
							fmt.Printf("\n%s\n", ui.ErrorText.Render(fmt.Sprintf("Auto-update failed: %v. Run 'fontget update' manually.", err)))
						} else {
							if logger != nil {
								logger.Info("Auto-update successful - updated to %s", result.LatestVersion)
							}
							fmt.Printf("\n%s\n", ui.SuccessText.Render(fmt.Sprintf("FontGet has been automatically updated to v%s", result.LatestVersion)))
						}
					}()
				} else {
					// Just notify user
					fmt.Printf("\n%s\n", ui.InfoText.Render(update.FormatUpdateNotification(result.CurrentVersion, result.LatestVersion)))
				}
			}
		},
	)
}
