package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const (
	version = "1.0"
)

var rootCmd = &cobra.Command{
	Use:   "fontget",
	Short: "A command-line tool for managing fonts",
	Long: `Fontget is a command-line tool for managing fonts on your system.
It allows you to add, remove, and list fonts, with support for both user and system-wide installation.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	// Hide the completion command from the main help output
	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	// Set custom help template
	rootCmd.SetHelpTemplate(`{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if .Runnable}}Usage:
  {{.UseLine}}{{end}}

{{if .HasAvailableFlags}}Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}

{{if .HasExample}}Examples:
{{.Example}}{{end}}

{{if .HasHelpSubCommands}}Additional Commands:
{{range .Commands}}{{if .IsAvailableCommand}}  {{rpad .Name .NamePadding }} {{.Short}}
{{end}}{{end}}{{end}}`)

	// Set custom usage template with extra spacing
	rootCmd.SetUsageTemplate(`{{if .Runnable}}Usage:
  {{.UseLine}}{{end}}

{{if .HasAvailableFlags}}Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}

{{if .HasExample}}Examples:
{{.Example}}{{end}}

{{if .HasHelpSubCommands}}Additional Commands:
{{range .Commands}}{{if .IsAvailableCommand}}  {{rpad .Name .NamePadding }} {{.Short}}
{{end}}{{end}}{{end}}`)

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
