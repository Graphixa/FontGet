package cmd

import (
	"errors"
	"fmt"
	"fontget/internal/components"
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

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
)

// Version is now managed centrally in internal/version package

var (
	verbose          bool
	debug            bool
	logs             bool
	wizard           bool
	acceptAgreements bool
	acceptDefaults   bool
	logger           *logging.Logger
	pendingUpdate    *update.CheckResult // Stores update check result for post-command prompt
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

		// Skip onboarding/license for certain commands (e.g. help, version, config reset)
		// Shell tab completion invokes cobra.ShellCompRequestCmd / ShellCompNoDescRequestCmd, not "completion".
		skipLicenseCommands := map[string]bool{
			"help":                          true,
			"version":                       true,
			"completion":                    true,
			cobra.ShellCompRequestCmd:       true,
			cobra.ShellCompNoDescRequestCmd: true,
			"reset":                         true, // config reset: do not prompt for wizard
		}

		// Apply automation flags (persistent: available on all commands, e.g. fontget add X --accept-agreements --accept-defaults)
		root := cmd.Root()
		acceptAgreements := os.Getenv("FONTGET_ACCEPT_AGREEMENTS") == "1"
		acceptDefaults := os.Getenv("FONTGET_ACCEPT_DEFAULTS") == "1" || os.Getenv("FONTGET_SKIP_ONBOARDING") == "1"
		if f := root.PersistentFlags().Lookup("accept-agreements"); f != nil && f.Value.String() == "true" {
			acceptAgreements = true
		}
		if f := root.PersistentFlags().Lookup("accept-defaults"); f != nil && f.Value.String() == "true" {
			acceptDefaults = true
		}

		// Skip first-run onboarding if --wizard flag is set (wizard will be run in RunE)
		if !wizard && !skipLicenseCommands[cmd.Name()] {
			isFirstRun, errFirst := config.IsFirstRun()
			if errFirst != nil {
				return fmt.Errorf("check first run: %w", errFirst)
			}
			isTTY := term.IsTerminal(os.Stdin.Fd())

			// Non-interactive: require both flags to avoid blocking
			if !isTTY && isFirstRun && (!acceptDefaults || !acceptAgreements) {
				return fmt.Errorf("onboarding requires an interactive terminal. In CI/automation, use --accept-agreements --accept-defaults (e.g. fontget add <font-ID> --accept-agreements --accept-defaults) or set FONTGET_ACCEPT_AGREEMENTS=1 and FONTGET_ACCEPT_DEFAULTS=1")
			}

			if acceptAgreements || acceptDefaults {
				fmt.Println() // Section start: blank line per spacing guidelines
				if acceptAgreements {
					fmt.Println("Terms Accepted: Thank you for acknowledging and agreeing to the Terms of Use and to comply with each font's applicable license.")
					_ = config.MarkAgreementsAccepted()
				}
				// Both flags: accept agreements, create defaults, mark complete, no TUI
				if acceptDefaults && acceptAgreements {
					if isFirstRun {
						if err := config.GenerateInitialUserPreferences(); err != nil {
							return fmt.Errorf("create default config: %w", err)
						}
						if err := config.EnsureManifestExists(); err != nil {
							return fmt.Errorf("create manifest: %w", err)
						}
						if err := config.MarkFirstRunCompleted(); err != nil {
							return fmt.Errorf("mark first run completed: %w", err)
						}
					}

					fmt.Println("Defaults Accepted: FontGet will use the default configuration.")
					// Skip running any onboarding TUI
				} else if acceptDefaults {
					fmt.Println("Defaults Accepted: FontGet will use the default configuration.")
					// --accept-defaults only: on first run, show terms TUI only if agreements not already accepted
					if isFirstRun {
						agreed, _ := config.IsAgreementsAccepted()
						if agreed {
							// Already accepted (e.g. from previous --accept-agreements); create defaults, no TUI
							if err := config.GenerateInitialUserPreferences(); err != nil {
								return fmt.Errorf("create default config: %w", err)
							}
							if err := config.EnsureManifestExists(); err != nil {
								return fmt.Errorf("create manifest: %w", err)
							}
							if err := config.MarkFirstRunCompleted(); err != nil {
								return fmt.Errorf("mark first run completed: %w", err)
							}
						} else {
							if err := onboarding.RunLicenseAgreementOnly(); err != nil {
								if errors.Is(err, shared.ErrOnboardingCancelled) {
									os.Exit(0)
								}
								return err
							}
						}
					}
				}
				// No trailing blank: next command adds its own leading blank to avoid double space
			}

			if isFirstRun && !acceptDefaults {
				// Run full onboarding (starts after license step if already accepted)
				if err := onboarding.RunFirstRunOnboarding(); err != nil {
					if errors.Is(err, shared.ErrOnboardingCancelled) || errors.Is(err, shared.ErrOnboardingIncomplete) {
						os.Exit(0)
					}
					return err
				}
			}
		}

		// Perform startup update check (non-blocking)
		// Skip for certain commands to avoid unnecessary checks
		skipUpdateCheckCommands := map[string]bool{
			"help":                          true,
			"version":                       true, // Don't check for updates when just checking version
			"completion":                    true,
			cobra.ShellCompRequestCmd:       true,
			cobra.ShellCompNoDescRequestCmd: true,
			"update":                        true, // Don't check for updates when running update command
		}

		if !skipUpdateCheckCommands[cmd.Name()] {
			go performStartupUpdateCheck()
		}

		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		// Handle pending update prompt after command completes
		if pendingUpdate != nil && pendingUpdate.UpdateAvailable {
			// Load config to check grace period
			appConfig := config.GetUserPreferences()
			if appConfig == nil {
				// If we can't load config, clear pending update and continue
				pendingUpdate = nil
				if logger != nil {
					return logger.Close()
				}
				return nil
			}

			// Check if we should show the prompt based on grace period
			if !update.ShouldShowUpdatePrompt(appConfig.Update.NextUpdateCheck) {
				// Still in grace period - don't show prompt
				if logger != nil {
					logger.Info("Update prompt skipped - still in grace period")
				}
				// Clear pending update
				pendingUpdate = nil
				if logger != nil {
					return logger.Close()
				}
				return nil
			}

			// Show confirmation prompt
			confirmed, err := components.RunConfirm(
				"Update Available",
				fmt.Sprintf("FontGet v%s is available (you have v%s).\nUpdate now?",
					pendingUpdate.LatestVersion, pendingUpdate.CurrentVersion),
			)
			if err != nil {
				if logger != nil {
					logger.Error("Confirmation dialog failed: %v", err)
				}
				// Clear pending update and continue
				pendingUpdate = nil
				if logger != nil {
					return logger.Close()
				}
				return nil
			}

			if confirmed {
				// User confirmed - perform update
				if logger != nil {
					logger.Info("User confirmed update from %s to %s", pendingUpdate.CurrentVersion, pendingUpdate.LatestVersion)
				}

				// Clear the grace period since user accepted
				appConfig.Update.NextUpdateCheck = ""
				if err := config.SaveUserPreferences(appConfig); err != nil {
					if logger != nil {
						logger.Warn("Failed to clear NextUpdateCheck: %v", err)
					}
				}

				// Show spinner while updating
				err := ui.RunSpinner(
					fmt.Sprintf("Updating FontGet from v%s to v%s...", pendingUpdate.CurrentVersion, pendingUpdate.LatestVersion),
					fmt.Sprintf("Successfully updated to FontGet v%s", pendingUpdate.LatestVersion),
					func() error {
						return update.UpdateToLatest()
					},
				)
				if err != nil {
					if logger != nil {
						logger.Error("Update failed: %v", err)
					}
					fmt.Printf("%s\n", ui.ErrorText.Render(fmt.Sprintf("Update failed: %v. Run 'fontget update' manually.", err)))
				} else {
					if logger != nil {
						logger.Info("Update successful - updated to %s", pendingUpdate.LatestVersion)
					}

					// Config migration is handled automatically by update.UpdateToLatest() after binary update
					// No explicit migration call needed here - it happens as part of the update process
				}
			} else {
				// User declined - set grace period so we don't prompt again until next interval
				if logger != nil {
					logger.Info("User declined update - setting grace period")
				}
				appConfig.Update.NextUpdateCheck = update.GetUpdateDeclinedUntilTimestamp(appConfig.Update.UpdateCheckInterval)
				if err := config.SaveUserPreferences(appConfig); err != nil {
					if logger != nil {
						logger.Warn("Failed to save NextUpdateCheck: %v", err)
					}
				}
			}

			// Clear pending update
			pendingUpdate = nil
		}

		if logger != nil {
			return logger.Close()
		}
		return nil
	},
}

func init() {
	// Add verbose flag - shows detailed operation information including file/variant listings (user-friendly)
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed verbose output of operations")

	// Add debug flag - shows full diagnostic logs with timestamps (for developers)
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Show debug logs with timestamps (for troubleshooting)")

	// Add logs flag (non-persistent - only available on root command)
	rootCmd.Flags().BoolVar(&logs, "logs", false, "Open logs directory")

	// Add wizard flag (not persistent - only applies to root command)
	rootCmd.Flags().BoolVar(&wizard, "wizard", false, "Run the setup wizard to configure FontGet")

	// Automation / CI: available on all commands (e.g. fontget add X --accept-agreements --accept-defaults)
	rootCmd.PersistentFlags().BoolVar(&acceptAgreements, "accept-agreements", false, "Accept the FontGet terms of use without showing the prompt (for scripts/CI)")
	rootCmd.PersistentFlags().BoolVar(&acceptDefaults, "accept-defaults", false, "Use default configuration and skip the setup wizard; use with --accept-agreements for fully non-interactive use")

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
		appConfig.Update.CheckForUpdates,
		appConfig.Update.UpdateCheckInterval,
		appConfig.Update.LastUpdateCheck,
		func(result *update.CheckResult) {
			// Always update LastUpdateCheck timestamp (even on errors)
			// This prevents repeated failed checks
			appConfig.Update.LastUpdateCheck = update.GetLastCheckedTimestamp()
			if err := config.SaveUserPreferences(appConfig); err != nil {
				// Log error but don't interrupt user experience
				if logger != nil {
					logger.Error("Failed to save LastUpdateCheck timestamp: %v", err)
				}
			}

			// Log errors if any
			if result.Error != nil {
				if logger != nil {
					logger.Warn("Update check failed: %v", result.Error)
				}
				// Don't show error to user during startup - will be visible in logs
				// Timestamp is updated so we won't keep retrying
				return
			}

			// If update is available, check grace period before storing result
			if result.UpdateAvailable {
				// Check if we should show the prompt based on grace period
				if update.ShouldShowUpdatePrompt(appConfig.Update.NextUpdateCheck) {
					pendingUpdate = result
					// Don't show prompt here - defer to PersistentPostRunE
					// This ensures prompt appears after command output
				} else {
					// Still in grace period - don't set pending update
					if logger != nil {
						logger.Info("Update available but grace period active - skipping prompt")
					}
				}
			}
		},
	)
}
