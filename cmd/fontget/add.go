package main

import (
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add [font-name]",
	Short: "Install a font",
	Long:  `Install a font from the repository. Example: fontget add "open-sans"`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Implement font installation
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
} 