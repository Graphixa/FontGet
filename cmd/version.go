package cmd

import (
	"fmt"

	"fontget/internal/ui"
	"fontget/internal/version"

	"github.com/spf13/cobra"
)

var versionReleaseNotes bool

var versionCmd = &cobra.Command{
	Use:          "version",
	Short:        "Show FontGet version information",
	SilenceUsage: true,
	Long:         `Display version and build information.`,
	Example:      `  fontget version`,
	Args:         cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		v := version.GetVersion()
		display := v
		if v != "dev" {
			display = "v" + v
		}

		// Primary version line - styled with main info color
		versionLine := ui.RenderInfo("FontGet " + display)
		fmt.Println(versionLine)

		// When debug is enabled, show detailed build information
		if IsDebug() {
			commit := version.GitCommit
			if len(commit) > 7 {
				commit = commit[:7]
			}

			fmt.Printf("Commit: %s\n", commit)

			if version.BuildDate != "unknown" {
				fmt.Printf("Build: %s\n", version.BuildDate)
			}
		}

		if versionReleaseNotes {
			if v == "dev" {
				fmt.Println(ui.FeedbackText.Render("Release notes are only available for tagged releases."))
				return
			}
			tag := "v" + v
			fmt.Println(ui.FeedbackInfo.Render(fmt.Sprintf("Release notes for FontGet %s:", tag)))
			fmt.Println(ui.FeedbackText.Render(fmt.Sprintf("https://github.com/Graphixa/FontGet/releases/tag/%s", tag)))
		}
	},
}

func init() {
	versionCmd.Flags().BoolVar(&versionReleaseNotes, "release-notes", false, "Show release notes link for this version")
	rootCmd.AddCommand(versionCmd)
}
