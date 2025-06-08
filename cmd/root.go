package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "fontget",
	Short: "A CLI tool to install fonts from Google Fonts",
	Long:  `fontget is a CLI tool that allows you to install fonts from the Google Fonts repository.`,
}

// Execute executes the root command
func Execute() error {
	return rootCmd.Execute()
} 