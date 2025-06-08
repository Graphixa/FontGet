package main

import (
	"github.com/spf13/cobra"
)

var (
	googleOnly bool
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export installed fonts to JSON",
	Long:  `Export all installed fonts to a JSON file. Use --google to export only Google Fonts.`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Implement font export
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.Flags().BoolVar(&googleOnly, "google", false, "Export only Google Fonts")
} 