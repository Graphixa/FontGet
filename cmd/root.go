package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "fontget",
	Short: "A command-line tool for installing fonts from the Google Fonts repository",
	Long: `fontget is a CLI tool that allows you to install fonts from the Google Fonts repository.
You can specify the installation scope using the --scope flag:
  - user (default): Install for current user only
  - machine: Install system-wide (requires elevation)`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	// Hide the completion command from the main help output
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
}

// Execute executes the root command and returns an error
func Execute() error {
	return rootCmd.Execute()
}
