package main

import (
	"github.com/spf13/cobra"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage font cache",
	Long:  `Manage the local font cache, including pruning unused downloads.`,
}

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Clear unused downloads",
	Long:  `Remove font files from the cache that haven't been used recently.`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Implement cache pruning
	},
}

func init() {
	rootCmd.AddCommand(cacheCmd)
	cacheCmd.AddCommand(pruneCmd)
} 