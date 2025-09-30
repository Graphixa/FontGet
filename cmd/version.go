package cmd

import (
	"fmt"
	"fontget/internal/version"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show FontGet version information",
	Long: `Display version information for FontGet including build details.

This command shows the current version of FontGet, along with build information
such as git commit hash and build date when available.`,
	Example: `  fontget version`,
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.GetFullVersion())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
