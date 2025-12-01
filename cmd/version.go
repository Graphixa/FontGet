package cmd

import (
	"fmt"
	"strings"

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
		// Handle dev+hash format (build metadata)
		if strings.HasPrefix(v, "dev+") {
			display = v // Keep as "dev+abc123"
		} else if v != "dev" {
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
			// Check if it's a dev build (with or without commit hash)
			if v == "dev" || strings.HasPrefix(v, "dev+") {
				fmt.Println(ui.Text.Render("Release notes are only available for tagged releases."))
				return
			}
			tag := "v" + v
			fmt.Println(ui.InfoText.Render(fmt.Sprintf("Release notes for FontGet %s:", tag)))
			fmt.Println(ui.Text.Render(fmt.Sprintf("https://github.com/Graphixa/FontGet/releases/tag/%s", tag)))
		}
	},
}

func init() {
	versionCmd.Flags().BoolVar(&versionReleaseNotes, "release-notes", false, "Show release notes link for this version")
	rootCmd.AddCommand(versionCmd)
}
