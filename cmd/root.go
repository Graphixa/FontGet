package cmd

import (
	"fmt"
	"fontget/internal/license"
	"fontget/internal/logging"
	"fontget/internal/platform"
	"os"

	"github.com/spf13/cobra"
)

const (
	version = "1.0"
)

var (
	verbose bool
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
		// Initialize logger with appropriate level based on verbose flag
		config := logging.DefaultConfig()
		if verbose {
			config.Level = logging.DebugLevel
			config.ConsoleOutput = true // Enable console output for debug/info logs when verbose is set
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
	// Add verbose flag
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	// Add logs flag
	rootCmd.PersistentFlags().BoolVar(&logs, "logs", false, "Open logs directory")

	// Set custom help template
	rootCmd.SetHelpTemplate(`
{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

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
{{end}}{{end}}{{end}}
`)

	// Set custom usage template with extra spacing
	rootCmd.SetUsageTemplate(`
{{if .Runnable}}Usage:
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
{{end}}
`)

	// Add completion command
	completionCmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate completion script",
		Long: `To load completions:

Bash:
  $ source <(go run main.go completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ go run main.go completion bash > ~/.fontget-completion.bash
  $ source ~/.fontget-completion.bash
  # macOS:
  $ go run main.go completion bash > /usr/local/etc/bash_completion.d/fontget

Zsh:
  $ source <(go run main.go completion zsh)

  # To load completions for each session, execute once:
  $ go run main.go completion zsh > "${fpath[1]}/_fontget"

PowerShell:
  PS> go run main.go completion powershell > fontget.ps1
  PS> . ./fontget.ps1`,
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
	err := rootCmd.Execute()
	if err != nil {
		// Check if it's our custom error type
		if _, ok := err.(*FontInstallationError); ok {
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
