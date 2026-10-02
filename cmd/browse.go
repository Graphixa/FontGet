package cmd

import (
	"errors"
	"fmt"

	"fontget/internal/cmdutils"
	"fontget/internal/output"
	"fontget/internal/platform"
	"fontget/internal/repo"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var browseCmd = &cobra.Command{
	Use:   "browse",
	Short: "Interactive font catalog",
	Long: `Browse, search, and install or remove fonts in a terminal UI.

Tab / Shift+Tab cycles categories. --scope and --force match fontget add.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		GetLogger().Info("Starting browse TUI")

		if err := cmdutils.EnsureManifestInitialized(func() cmdutils.Logger { return GetLogger() }); err != nil {
			return err
		}

		output.GetVerbose().Info("Loading font repository")
		output.GetDebug().State("Calling repo.GetRepository()")
		r, err := repo.GetRepository()
		if err != nil {
			if IsCancelErr(err) {
				return nil
			}
			if lg := GetLogger(); lg != nil {
				lg.Error("Failed to get repository: %v", err)
			}
			output.GetVerbose().Error("%v", err)
			output.GetDebug().Error("repo.GetRepository() failed: %v", err)
			return fmt.Errorf("unable to load font repository: %w", err)
		}

		fontManager, err := cmdutils.CreateFontManager(func() cmdutils.Logger { return GetLogger() })
		if err != nil {
			return err
		}

		scope, _ := cmd.Flags().GetString("scope")
		force, _ := cmd.Flags().GetBool("force")

		if scope == "" {
			scope, err = platform.AutoDetectScope(fontManager, "user", "machine", GetLogger())
			if err != nil {
				scope = "user"
			}
		}

		installScope := platform.UserScope
		if scope != "user" {
			installScope = platform.InstallationScope(scope)
			if installScope != platform.UserScope && installScope != platform.MachineScope {
				return fmt.Errorf("invalid scope %q: use user or machine", scope)
			}
		}

		if err := cmdutils.CheckElevation(cmd, fontManager, installScope); err != nil {
			if errors.Is(err, cmdutils.ErrElevationRequired) {
				return nil
			}
			return err
		}

		fontDir := fontManager.GetFontDir(installScope)

		model, err := newBrowseModel(r, fontManager, installScope, fontDir, force)
		if err != nil {
			return fmt.Errorf("failed to start browse UI: %w", err)
		}

		p := tea.NewProgram(model, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("browse UI: %w", err)
		}

		GetLogger().Info("Browse TUI finished")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(browseCmd)
	browseCmd.Flags().StringP("scope", "s", "", "Installation scope (user or machine)")
	browseCmd.Flags().BoolP("force", "f", false, "Reinstall even if already installed")
}
