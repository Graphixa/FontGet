package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove [font-name]",
	Aliases: []string{"uninstall"},
	Short:   "Remove a font from your system",
	Long:    `Remove a font from your system by deleting it from the system font directory.`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fontName := args[0]
		fmt.Printf("Removing font: %s\n", fontName)
		// TODO: Implement font removal logic
		fmt.Println("Font removal not yet implemented")
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
