package cmd

import (
	"errors"
	"fmt"

	"fontget/internal/cmdutils"
	"fontget/internal/platform"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var browseCmd = &cobra.Command{
	Use:   "browse",
	Short: "Interactive font catalog: search, filter by category, install or remove",
	Long: `Browse the font catalog in the terminal: search, view details, open source pages, and install
or remove fonts. Installation behavior matches fontget add (scope, force).

In the TUI, use Tab / Shift+Tab to cycle the category filter (All, then each category from your manifest,
including custom sources). Type in the search box to narrow results; an empty search with a category
shows all fonts in that category.

Flags --scope (-s) and --force (-f) match fontget add (user/machine install scope; force reinstall).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		GetLogger().Info("Starting browse TUI")

		if err := cmdutils.EnsureManifestInitialized(func() cmdutils.Logger { return GetLogger() }); err != nil {
			return err
		}

		refresh, _ := cmd.Flags().GetBool("refresh")
		r, err := cmdutils.GetRepository(refresh, GetLogger())
		if err != nil {
			return err
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
	browseCmd.Flags().BoolP("force", "f", false, "Force installation even if font is already installed")
	browseCmd.Flags().Bool("refresh", false, "Force refresh of font manifest before browse")
	_ = browseCmd.Flags().MarkHidden("refresh")
}
