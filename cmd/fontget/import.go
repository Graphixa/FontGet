package main

import (
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import [file]",
	Short: "Import fonts from JSON file",
	Long:  `Import and install fonts from a JSON file containing font definitions.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Implement font import
	},
}

func init() {
	rootCmd.AddCommand(importCmd)
} 