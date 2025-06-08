package main

import (
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove [font-name]",
	Short: "Uninstall a font",
	Long:  `Remove a previously installed font. Example: fontget remove "open-sans"`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Implement font removal
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
} 