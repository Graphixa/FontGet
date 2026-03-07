package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"fontget/internal/cmdutils"
	"fontget/internal/config"
	"fontget/internal/functions"
	"fontget/internal/output"
	"fontget/internal/ui"

	"github.com/spf13/cobra"
)

var sourcesAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a font source",
	Long: `Add a custom font source. Name and URL are required; prefix and priority are optional.
Prefix defaults to a slug from the name (lowercase, spaces to hyphens). Priority defaults to the next available (custom sources start after built-in).

Example:
  fontget sources add --name "My Fonts" --url https://example.com/fonts.json
  fontget sources add --name "Custom" --url https://example.com/sources.json --prefix custom --priority 10`,
	Example: `  fontget sources add --name "My Source" --url https://example.com/fonts.json
  fontget sources add --name "Custom" --url https://example.com/sources.json --prefix custom`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runSourcesAdd,
}

var sourcesRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a custom font source",
	Long: `Remove a custom font source by name or prefix. Built-in sources (Google Fonts, Nerd Fonts, Font Squirrel) cannot be removed.
Use --force or --yes to skip the confirmation prompt (for scripts).`,
	Example: `  fontget sources remove --name "My Source"
  fontget sources remove --name mysource --force`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runSourcesRemove,
}

var sourcesEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable a font source",
	Long:  `Enable a font source by name or prefix. Works for both built-in and custom sources.`,
	Example: `  fontget sources enable --name "Nerd Fonts"
  fontget sources enable --name nerd`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runSourcesEnable,
}

var sourcesDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable a font source",
	Long:  `Disable a font source by name or prefix. Works for both built-in and custom sources.`,
	Example: `  fontget sources disable --name "Nerd Fonts"
  fontget sources disable --name nerd`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runSourcesDisable,
}

var sourcesSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Update a custom source's properties",
	Long: `Update URL, prefix, or priority of a custom font source by name or prefix. Built-in sources cannot be modified (use enable/disable to change availability).
At least one of --url, --prefix, or --priority must be provided.`,
	Example: `  fontget sources set --name "My Source" --url https://new.example.com/fonts.json
  fontget sources set --name mysource --priority 5 --prefix mysource`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runSourcesSet,
}

func initSourcesCLI() {
	sourcesCmd.AddCommand(sourcesAddCmd)
	sourcesCmd.AddCommand(sourcesRemoveCmd)
	sourcesCmd.AddCommand(sourcesEnableCmd)
	sourcesCmd.AddCommand(sourcesDisableCmd)
	sourcesCmd.AddCommand(sourcesSetCmd)

	sourcesAddCmd.Flags().StringP("name", "n", "", "Source name (required)")
	sourcesAddCmd.Flags().StringP("url", "u", "", "Source URL (required, must be http(s)://)")
	sourcesAddCmd.Flags().StringP("prefix", "p", "", "Source prefix (optional; default from name)")
	sourcesAddCmd.Flags().Int("priority", 0, "Priority, lower = higher priority (optional; default next available)")
	_ = sourcesAddCmd.MarkFlagRequired("name")
	_ = sourcesAddCmd.MarkFlagRequired("url")

	sourcesRemoveCmd.Flags().StringP("name", "n", "", "Source name or prefix to remove (required)")
	sourcesRemoveCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	sourcesRemoveCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
	_ = sourcesRemoveCmd.MarkFlagRequired("name")

	sourcesEnableCmd.Flags().StringP("name", "n", "", "Source name or prefix to enable (required)")
	_ = sourcesEnableCmd.MarkFlagRequired("name")

	sourcesDisableCmd.Flags().StringP("name", "n", "", "Source name or prefix to disable (required)")
	_ = sourcesDisableCmd.MarkFlagRequired("name")

	sourcesSetCmd.Flags().StringP("name", "n", "", "Source name or prefix to update (required)")
	sourcesSetCmd.Flags().StringP("url", "u", "", "New source URL")
	sourcesSetCmd.Flags().StringP("prefix", "p", "", "New source prefix")
	sourcesSetCmd.Flags().Int("priority", -1, "New priority (positive integer)")
	_ = sourcesSetCmd.MarkFlagRequired("name")
}

func runSourcesAdd(cmd *cobra.Command, _ []string) error {
	name, _ := cmd.Flags().GetString("name")
	url, _ := cmd.Flags().GetString("url")
	prefix, _ := cmd.Flags().GetString("prefix")
	priority, _ := cmd.Flags().GetInt("priority")

	name = strings.TrimSpace(name)
	url = strings.TrimSpace(url)
	prefix = strings.TrimSpace(prefix)

	if GetLogger() != nil {
		GetLogger().Info("sources add: name=%q url=%q prefix=%q priority=%d", name, url, prefix, priority)
	}
	output.GetDebug().State("sources add: name=%q url=%q prefix=%q priority=%d", name, url, prefix, priority)

	manifest, err := config.LoadManifest()
	if err != nil {
		fmt.Println()
		cmdutils.PrintErrorf("Cannot load sources manifest: %v", err)
		fmt.Println()
		return err
	}

	existing := convertManifestToSourceItems(manifest)
	result := functions.ValidateSourceForm(name, url, prefix, existing, -1)
	if !result.IsValid {
		fmt.Println()
		cmdutils.PrintErrorf("%s", result.GetFirstError())
		fmt.Println()
		return fmt.Errorf("validation: %s", result.GetFirstError())
	}

	if prefix == "" {
		prefix = functions.AutoGeneratePrefix(name)
	}
	prefix = strings.ToLower(prefix)

	if priority <= 0 {
		priority = nextCustomSourcePriority(manifest)
	}

	filename := generateFilename(name)
	manifest.Sources[name] = config.SourceConfig{
		URL:      url,
		Prefix:   prefix,
		Enabled:  true,
		Filename: filename,
		Priority: priority,
	}

	if err := config.SaveManifest(manifest); err != nil {
		fmt.Println()
		cmdutils.PrintErrorf("Failed to save manifest: %v", err)
		fmt.Println()
		return err
	}

	fmt.Println()
	fmt.Printf("%s\n", ui.SuccessText.Render(fmt.Sprintf("Added source %q.", name)))
	return nil
}

func runSourcesRemove(cmd *cobra.Command, _ []string) error {
	name, _ := cmd.Flags().GetString("name")
	force, _ := cmd.Flags().GetBool("force")
	yes, _ := cmd.Flags().GetBool("yes")
	name = strings.TrimSpace(name)
	skipConfirm := force || yes

	if GetLogger() != nil {
		GetLogger().Info("sources remove: name=%q force=%v", name, skipConfirm)
	}

	manifest, err := config.LoadManifest()
	if err != nil {
		fmt.Println()
		cmdutils.PrintErrorf("Cannot load sources manifest: %v", err)
		fmt.Println()
		return err
	}

	resolvedName, err := resolveSourceName(manifest, name)
	if err != nil {
		fmt.Println()
		cmdutils.PrintErrorf("Source %q not found.", name)
		fmt.Println()
		return err
	}
	name = resolvedName

	if config.IsBuiltInSource(name) {
		fmt.Println()
		cmdutils.PrintErrorf("Cannot remove built-in source %q. Only custom sources can be removed.", name)
		fmt.Println()
		return fmt.Errorf("cannot remove built-in source")
	}

	if _, exists := manifest.Sources[name]; !exists {
		fmt.Println()
		cmdutils.PrintErrorf("Source %q not found.", name)
		fmt.Println()
		return fmt.Errorf("source not found")
	}

	if !skipConfirm {
		fmt.Println()
		fmt.Printf("Remove source %q? [y/N] ", name)
		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			fmt.Println()
			return nil
		}
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer != "y" && answer != "yes" {
			fmt.Println("Not removed.")
			return nil
		}
	}

	delete(manifest.Sources, name)
	if err := config.SaveManifest(manifest); err != nil {
		fmt.Println()
		cmdutils.PrintErrorf("Failed to save manifest: %v", err)
		fmt.Println()
		return err
	}

	fmt.Println()
	fmt.Printf("%s\n", ui.SuccessText.Render(fmt.Sprintf("Removed source %q.", name)))
	return nil
}

func runSourcesEnable(cmd *cobra.Command, _ []string) error {
	name, _ := cmd.Flags().GetString("name")
	name = strings.TrimSpace(name)

	manifest, err := config.LoadManifest()
	if err != nil {
		fmt.Println()
		cmdutils.PrintErrorf("Cannot load sources manifest: %v", err)
		fmt.Println()
		return err
	}

	resolvedName, err := resolveSourceName(manifest, name)
	if err != nil {
		fmt.Println()
		cmdutils.PrintErrorf("Source %q not found.", name)
		fmt.Println()
		return err
	}
	name = resolvedName

	source := manifest.Sources[name]
	source.Enabled = true
	manifest.Sources[name] = source

	if err := config.SaveManifest(manifest); err != nil {
		fmt.Println()
		cmdutils.PrintErrorf("Failed to save manifest: %v", err)
		fmt.Println()
		return err
	}

	fmt.Println()
	fmt.Printf("%s\n", ui.SuccessText.Render(fmt.Sprintf("Enabled source %q.", name)))
	return nil
}

func runSourcesDisable(cmd *cobra.Command, _ []string) error {
	name, _ := cmd.Flags().GetString("name")
	name = strings.TrimSpace(name)

	manifest, err := config.LoadManifest()
	if err != nil {
		fmt.Println()
		cmdutils.PrintErrorf("Cannot load sources manifest: %v", err)
		fmt.Println()
		return err
	}

	resolvedName, err := resolveSourceName(manifest, name)
	if err != nil {
		fmt.Println()
		cmdutils.PrintErrorf("Source %q not found.", name)
		fmt.Println()
		return err
	}
	name = resolvedName

	source := manifest.Sources[name]
	source.Enabled = false
	manifest.Sources[name] = source

	if err := config.SaveManifest(manifest); err != nil {
		fmt.Println()
		cmdutils.PrintErrorf("Failed to save manifest: %v", err)
		fmt.Println()
		return err
	}

	fmt.Println()
	fmt.Printf("%s\n", ui.SuccessText.Render(fmt.Sprintf("Disabled source %q.", name)))
	return nil
}

func runSourcesSet(cmd *cobra.Command, _ []string) error {
	name, _ := cmd.Flags().GetString("name")
	url, _ := cmd.Flags().GetString("url")
	prefix, _ := cmd.Flags().GetString("prefix")
	priority, _ := cmd.Flags().GetInt("priority")
	name = strings.TrimSpace(name)
	url = strings.TrimSpace(url)
	prefix = strings.TrimSpace(prefix)

	hasURL := url != ""
	hasPrefix := prefix != ""
	hasPriority := priority >= 0
	if !hasURL && !hasPrefix && !hasPriority {
		fmt.Println()
		cmdutils.PrintErrorf("At least one of --url, --prefix, or --priority must be provided.")
		fmt.Println()
		return fmt.Errorf("no properties to update")
	}

	manifest, err := config.LoadManifest()
	if err != nil {
		fmt.Println()
		cmdutils.PrintErrorf("Cannot load sources manifest: %v", err)
		fmt.Println()
		return err
	}

	resolvedName, err := resolveSourceName(manifest, name)
	if err != nil {
		fmt.Println()
		cmdutils.PrintErrorf("Source %q not found.", name)
		fmt.Println()
		return err
	}
	name = resolvedName

	if config.IsBuiltInSource(name) {
		fmt.Println()
		cmdutils.PrintErrorf("Cannot modify built-in source %q. Use sources enable/disable to change availability.", name)
		fmt.Println()
		return fmt.Errorf("cannot modify built-in source")
	}

	source := manifest.Sources[name]
	existing := convertManifestToSourceItems(manifest)
	editingIndex := functions.FindSourceIndex(existing, name)
	if hasURL {
		if err := functions.ValidateURL(url); err != nil {
			fmt.Println()
			cmdutils.PrintErrorf("%v", err)
			fmt.Println()
			return err
		}
		for i, s := range existing {
			if s.URL == url && (i != editingIndex) {
				fmt.Println()
				cmdutils.PrintErrorf("Another source already has URL %q.", url)
				fmt.Println()
				return fmt.Errorf("duplicate URL")
			}
		}
		source.URL = url
	}
	if hasPrefix {
		prefix = strings.ToLower(prefix)
		if err := functions.ValidatePrefix(prefix); err != nil {
			fmt.Println()
			cmdutils.PrintErrorf("%v", err)
			fmt.Println()
			return err
		}
		for i, s := range existing {
			if s.Prefix == prefix && (i != editingIndex) {
				fmt.Println()
				cmdutils.PrintErrorf("Another source already has prefix %q.", prefix)
				fmt.Println()
				return fmt.Errorf("duplicate prefix")
			}
		}
		source.Prefix = prefix
	}
	if hasPriority {
		if priority < 1 {
			fmt.Println()
			cmdutils.PrintErrorf("Priority must be a positive integer.")
			fmt.Println()
			return fmt.Errorf("invalid priority")
		}
		source.Priority = priority
	}

	manifest.Sources[name] = source

	if err := config.SaveManifest(manifest); err != nil {
		fmt.Println()
		cmdutils.PrintErrorf("Failed to save manifest: %v", err)
		fmt.Println()
		return err
	}

	fmt.Println()
	fmt.Printf("%s\n", ui.SuccessText.Render(fmt.Sprintf("Updated source %q.", name)))
	return nil
}

// resolveSourceName returns the manifest source name (key) for the given identifier, which may be either the full source name or the source prefix. If not found, returns "" and an error.
func resolveSourceName(manifest *config.Manifest, identifier string) (string, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return "", fmt.Errorf("source not found")
	}
	if _, exists := manifest.Sources[identifier]; exists {
		return identifier, nil
	}
	identLower := strings.ToLower(identifier)
	var match string
	for name, source := range manifest.Sources {
		if strings.ToLower(source.Prefix) == identLower {
			if match != "" {
				return "", fmt.Errorf("source not found")
			}
			match = name
		}
	}
	if match != "" {
		return match, nil
	}
	return "", fmt.Errorf("source not found")
}

// nextCustomSourcePriority returns the next priority for a custom source (max custom priority + 1, or 100 if none).
func nextCustomSourcePriority(manifest *config.Manifest) int {
	maxPriority := 99
	for n, s := range manifest.Sources {
		if !config.IsBuiltInSource(n) && s.Priority > maxPriority {
			maxPriority = s.Priority
		}
	}
	return maxPriority + 1
}
