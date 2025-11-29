package cmd

import (
	"fmt"

	"fontget/internal/ui"
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
		// Primary version line - styled with main info color
		versionLine := ui.RenderInfo("FontGet " + version.GetVersion())
		fmt.Println(versionLine)

		// When debug is enabled, show detailed build information
		if IsDebug() {
			commit := version.GitCommit
			if len(commit) > 7 {
				commit = commit[:7]
			}

			fmt.Printf("Commit: %s\n", commit)

			if version.BuildDate != "unknown" {
				fmt.Printf("Build:  %s\n", version.BuildDate)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
