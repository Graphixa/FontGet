// This is a template for building new commands.
// It is not used in the project.
// It is only used to help you build new commands.

package templates

import (
	"fmt"
	"strings"

	"fontget/internal/repo"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

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

		// Validate input
		if argValue == "" && flagValue == "" {
			red := color.New(color.FgRed).SprintFunc()
			fmt.Printf("\n%s\n\n", red("A required argument is missing"))
			return cmd.Help()
		}
		return nil
	},
	// Optional: Add argument completion
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Get manifest
		manifest, err := repo.GetManifest(nil, nil)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		// Get all fonts from all sources
		var completions []string
		for _, sourceInfo := range manifest.Sources {
			for fontID, fontInfo := range sourceInfo.Fonts {
				if strings.HasPrefix(strings.ToLower(fontInfo.Name), strings.ToLower(toComplete)) {
					completions = append(completions, fontID)
				}
			}
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Double check args to prevent panic
		flagValue, _ := cmd.Flags().GetString("flag-name")
		var argValue string
		if len(args) > 0 {
			argValue = args[0]
		}
		if argValue == "" && flagValue == "" {
			return nil // Args validator will have already shown the help
		}

		// Get manifest (using current architecture)
		manifest, err := repo.GetManifest(nil, nil)
		if err != nil {
			return fmt.Errorf("failed to get manifest: %w", err)
		}

		// Print results in a table format (matching search command style)
		fmt.Printf("\nFound %d items matching '%s'", 0, argValue)
		if flagValue != "" {
			fmt.Printf(" with flag '%s'", flagValue)
		}
		fmt.Println("\n")

		// Define column widths (matching search command)
		columns := map[string]int{
			"Name":       30,
			"ID":         30,
			"License":    12,
			"Categories": 15,
			"Source":     15,
		}

		// Print header (matching search command style)
		header := fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s",
			columns["Name"], "Name",
			columns["ID"], "ID",
			columns["License"], "License",
			columns["Categories"], "Categories",
			columns["Source"], "Source")
		fmt.Println(header)
		fmt.Println(strings.Repeat("-", len(header)))

		// Print manifest info
		fmt.Printf("\nManifest last updated: %s\n", manifest.LastUpdated.Format("Mon, 02 Jan 2006 15:04:05 MST"))
		fmt.Printf("Total items available: %d\n", countTotalItems(manifest))

		return nil
	},
}

// countTotalItems counts the total number of items in the manifest
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

Standard Help Formatting:
- Use winget-style help with "usage:" line
- Include subcommands in "The following sub-commands are available:" section
- Include flags in "The following options are available:" section
- End with "For more details on a specific command, pass it the help argument. [-?]"

Table Formatting (for list/search commands):
- Use consistent column widths matching search command
- Standard columns: Name, ID, License, Categories, Source
- Use dashes for header separator
- Include manifest info at bottom

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
			red := color.New(color.FgRed).SprintFunc()
			fmt.Printf("\n%s\n\n", red("A font name is required"))
			return cmd.Help()
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Double check args to prevent panic
		if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
			return nil // Args validator will have already shown the help
		}

		style, _ := cmd.Flags().GetString("style")
		fontName := args[0]
		// Add font logic here
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringP("style", "s", "", "Font style to install")
}
*/
