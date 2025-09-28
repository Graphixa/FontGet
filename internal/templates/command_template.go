// This is a template for building new commands.
// It is not used in the project.
// It is only used to help you build new commands.

package templates

import (
	"fmt"
	"strings"

	"fontget/internal/config"
	"fontget/internal/logging"
	"fontget/internal/repo"
	"fontget/internal/ui"

	"github.com/spf13/cobra"
)

// GetLogger is a placeholder - replace with actual logger function from your cmd package
// The actual implementation is in cmd/root.go:
// func GetLogger() *logging.Logger { return logger }
func GetLogger() *logging.Logger {
	return nil // Replace with actual logger implementation
}

// Template for new commands. Replace "command" with your command name
var commandCmd = &cobra.Command{
	Use:   "command <required-arg>",
	Short: "One-line description of what the command does",
	Long: `Detailed description of what the command does and how it works.

usage: fontget command [<options>]`,
	Example: `  fontget command example1
  fontget command "example with quotes"
  fontget command example3 --flag value
  fontget command example4 -f value`,
	// Use one of these Args validators:
	// cobra.NoArgs - Command doesn't accept any arguments
	// cobra.ExactArgs(n) - Command requires exactly n arguments
	// cobra.MinimumNArgs(n) - Command requires at least n arguments
	// cobra.MaximumNArgs(n) - Command accepts at most n arguments
	// cobra.RangeArgs(min, max) - Command accepts between min and max arguments
	Args: func(cmd *cobra.Command, args []string) error {
		// Get flags
		flagValue, _ := cmd.Flags().GetString("flag-name")

		// Get arguments
		var argValue string
		if len(args) > 0 {
			argValue = args[0]
		}

		// Validate input using modern UI styling
		if argValue == "" && flagValue == "" {
			fmt.Printf("\n%s\n\n", ui.RenderError("A required argument is missing"))
			return cmd.Help()
		}
		return nil
	},
	// Optional: Add argument completion using optimized repository access
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Use optimized repository access (smart caching)
		r, err := repo.GetRepository()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		// Get all fonts using repository method
		results, err := r.SearchFonts("", "")
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		// Filter and return font names
		var completions []string
		for _, result := range results {
			if strings.HasPrefix(strings.ToLower(result.Name), strings.ToLower(toComplete)) {
				completions = append(completions, result.Name)
			}
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get logger for consistent logging
		logger := GetLogger()
		if logger != nil {
			logger.Info("Starting command operation")
		}

		// Double check args to prevent panic
		flagValue, _ := cmd.Flags().GetString("flag-name")
		var argValue string
		if len(args) > 0 {
			argValue = args[0]
		}
		if argValue == "" && flagValue == "" {
			return nil // Args validator will have already shown the help
		}

		// Print styled title using modern UI components
		fmt.Printf("\n%s\n", ui.PageTitle.Render("Command Results"))

		// Use optimized repository access (smart caching like search/list commands)
		r, err := repo.GetRepository()
		if err != nil {
			if logger != nil {
				logger.Error("Failed to initialize repository: %v", err)
			}
			return fmt.Errorf("failed to initialize repository: %w", err)
		}

		// Get manifest from repository
		manifest, err := r.GetManifest()
		if err != nil {
			if logger != nil {
				logger.Error("Failed to get manifest: %v", err)
			}
			return fmt.Errorf("failed to get manifest: %w", err)
		}

		// Print results using modern UI styling
		fmt.Printf("\nFound %d items matching '%s'", 0, ui.TableSourceName.Render(argValue))
		if flagValue != "" {
			fmt.Printf(" with flag '%s'", ui.TableSourceName.Render(flagValue))
		}
		fmt.Println()

		// Define column widths (matching search command style)
		columns := map[string]int{
			"Name":       30, // For display name
			"ID":         30, // For longer font IDs
			"License":    12, // For license type
			"Categories": 15, // For categories
			"Source":     12, // For source name
		}

		// Print header with modern UI styling (matching search command)
		header := fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s",
			columns["Name"], "Name",
			columns["ID"], "ID",
			columns["License"], "License",
			columns["Categories"], "Categories",
			columns["Source"], "Source")
		fmt.Println(ui.TableHeader.Render(header))
		fmt.Println(ui.FeedbackText.Render(strings.Repeat("-", len(header))))

		// Example: Process fonts from manifest
		for _, sourceInfo := range manifest.Sources {
			for fontID, fontInfo := range sourceInfo.Fonts {
				// Example processing - replace with your actual logic
				_ = fontID
				_ = fontInfo
				// Add your processing logic here
			}
		}

		// Show when FontGet last updated sources (matching search command)
		if lastUpdated, err := config.GetSourcesLastUpdated(); err == nil && !lastUpdated.IsZero() {
			fmt.Printf("\n%s: %s\n", ui.FeedbackText.Render("Sources Last Updated"), lastUpdated.Format("Mon, 02 Jan 2006 15:04:05 MST"))
		}

		if logger != nil {
			logger.Info("Command operation completed successfully")
		}
		return nil
	},
}

// Helper function to count total items in manifest
func countTotalItems(manifest *repo.FontManifest) int {
	total := 0
	for _, source := range manifest.Sources {
		total += len(source.Fonts)
	}
	return total
}

func init() {
	// 1. Add the command to the root command
	// Note: Replace rootCmd with the actual root command variable from your cmd package
	// rootCmd.AddCommand(commandCmd)

	// 2. Add subcommands if needed (for commands like sources, config, etc.)
	// commandCmd.AddCommand(subCommand1)
	// commandCmd.AddCommand(subCommand2)

	// 3. Add flags
	// String flag with short version
	commandCmd.Flags().StringP("flag-name", "f", "", "Description of the flag")
	// Boolean flag
	commandCmd.Flags().BoolP("bool-flag", "b", false, "Description of the boolean flag")
	// String slice flag
	commandCmd.Flags().StringSliceP("slice-flag", "s", []string{}, "Description of the slice flag")

	// 4. Add flag completion if needed
	commandCmd.RegisterFlagCompletionFunc("flag-name", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Example flag completion
		completions := []string{
			"flag-value1",
			"flag-value2",
			"flag-value3",
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	})

	// 5. Add required flags if any
	// commandCmd.MarkFlagRequired("flag-name")
}

/*
Usage Instructions:

1. Copy this template to a new file named after your command (e.g., add.go, remove.go)
2. Replace "command" with your command name in the variable name and all references
3. Update the Use, Short, Long, and Example fields
4. Choose the appropriate Args validator
5. Implement the ValidArgsFunction if needed
6. Implement the RunE function with your command's logic
7. Add and configure flags in the init function
8. Add flag completion if needed
9. Mark flags as required if needed
10. For commands with subcommands, follow the sources command pattern

PERFORMANCE BEST PRACTICES:
- ALWAYS use repo.GetRepository() for normal operations (smart caching)
- ONLY use repo.GetManifest() directly when you need fresh data
- ONLY use repo.GetManifestWithRefresh() when forcing updates
- This ensures consistent performance across all commands

STYLING BEST PRACTICES:
- Use ui.RenderError() for error messages
- Use ui.PageTitle.Render() for main titles
- Use ui.TableHeader.Render() for table headers
- Use ui.TableSourceName.Render() for highlighted text
- Use ui.FeedbackText.Render() for regular text
- Use ui.FeedbackSuccess.Render() for success messages
- Use ui.FeedbackWarning.Render() for warnings
- Use ui.FeedbackInfo.Render() for info messages

LOGGING BEST PRACTICES:
- Always get logger with GetLogger()
- Use logger.Info() for operation start/completion
- Use logger.Error() for errors
- Use logger.Warn() for warnings

Standard Help Formatting:
- Use winget-style help with "usage:" line
- Include subcommands in "The following sub-commands are available:" section
- Include flags in "The following options are available:" section
- End with "For more details on a specific command, pass it the help argument. [-?]"

Table Formatting (for list/search commands):
- Use consistent column widths matching search command
- Standard columns: Name, ID, License, Categories, Source
- Use ui.FeedbackText.Render() for header separator
- Include manifest info at bottom using ui.FeedbackText.Render()

Example for an "add" command:

var addCmd = &cobra.Command{
	Use:   "add <font-name>",
	Short: "Add a font to your system",
	Long: `Downloads and installs a font from available sources.

usage: fontget add [<options>]`,
	Example: `  fontget add "Fira Sans"
  fontget add "Roboto" --style "Regular"
  fontget add "Open Sans" -s "Bold"`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
			fmt.Printf("\n%s\n\n", ui.RenderError("A font name is required"))
			return cmd.Help()
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get logger
		logger := GetLogger()
		if logger != nil {
			logger.Info("Starting font add operation")
		}

		// Double check args to prevent panic
		if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
			return nil // Args validator will have already shown the help
		}

		// Print styled title
		fmt.Printf("\n%s\n", ui.PageTitle.Render("Adding Font"))

		// Use optimized repository access
		r, err := repo.GetRepository()
		if err != nil {
			if logger != nil {
				logger.Error("Failed to initialize repository: %v", err)
			}
			return fmt.Errorf("failed to initialize repository: %w", err)
		}

		style, _ := cmd.Flags().GetString("style")
		fontName := args[0]

		// Add font logic here
		fmt.Printf("Adding font: %s", ui.TableSourceName.Render(fontName))
		if style != "" {
			fmt.Printf(" (style: %s)", ui.TableSourceName.Render(style))
		}
		fmt.Println()

		if logger != nil {
			logger.Info("Font add operation completed successfully")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringP("style", "s", "", "Font style to install")
}
*/
