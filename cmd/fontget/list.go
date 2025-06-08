package main

import (
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed fonts",
	Long:  `Display a list of all fonts currently installed on the system.`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Implement font listing
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
} 