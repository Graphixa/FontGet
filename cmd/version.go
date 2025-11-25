package cmd

import (
	"fmt"
	"fontget/internal/version"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:          "version",
	Short:        "Show FontGet version information",
	SilenceUsage: true,
	Long:         `Display version and build information.`,
	Example:      `  fontget version`,
	Args:         cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.GetFullVersion())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
