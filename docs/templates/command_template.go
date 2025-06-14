// This is a template for building new commands.
// It is not used in the project.
// It is only used to help you build new commands.

package cmd

import (
	"fmt"
	"strings"

	"fontget/internal/repo"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Template for new commands. Replace "command" with your command name
var commandCmd = &cobra.Command{
	Use:   "command <required-arg>",
	Short: "One-line description of what the command does",
	Long:  "Detailed description of what the command does and how it works.",
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
	Args: cobra.MaximumNArgs(1),
	// Optional: Add argument completion
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Get repository
		r, err := repo.GetRepository()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		// Get all fonts
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
		// 1. Get flags
		flagValue, _ := cmd.Flags().GetString("flag-name")

		// 2. Get arguments
		var argValue string
		if len(args) > 0 {
			argValue = args[0]
		}

		// 3. Validate input
		if argValue == "" {
			red := color.New(color.FgRed).SprintFunc()
			fmt.Printf("\n%s\n\n", red("A required argument is missing"))
			fmt.Println(cmd.Long)
			fmt.Println()
			fmt.Println("Usage:")
			fmt.Printf("  %s\n\n", cmd.UseLine())
			fmt.Println("Flags:")
			cmd.Flags().VisitAll(func(flag *pflag.Flag) {
				if flag.Shorthand != "" {
					fmt.Printf("  -%s, --%s\t%s\n", flag.Shorthand, flag.Name, flag.Usage)
				} else {
					fmt.Printf("  --%s\t%s\n", flag.Name, flag.Usage)
				}
			})
			if cmd.Example != "" {
				fmt.Println("\nExamples:")
				fmt.Println(cmd.Example)
			}
			return nil
		}

		// 4. Get repository
		r, err := repo.GetRepository()
		if err != nil {
			return fmt.Errorf("failed to initialize repository: %w", err)
		}

		// 5. Get manifest
		manifest, err := r.GetManifest()
		if err != nil {
			return fmt.Errorf("failed to get manifest: %w", err)
		}

		// 6. Print results in a table format
		fmt.Printf("\nFound %d items matching '%s'", 0, argValue)
		if flagValue != "" {
			fmt.Printf(" with flag '%s'", flagValue)
		}
		fmt.Println("\n")

		// Define column widths
		columns := map[string]int{
			"Name":   40,
			"ID":     38,
			"Value":  20,
			"Status": 15,
			"Source": 15,
		}

		// Print header
		header := fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s",
			columns["Name"], "Name",
			columns["ID"], "ID",
			columns["Value"], "Value",
			columns["Status"], "Status",
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
	rootCmd.AddCommand(commandCmd)

	// 2. Add flags
	// String flag with short version
	commandCmd.Flags().StringP("flag-name", "f", "", "Description of the flag")
	// Boolean flag
	commandCmd.Flags().BoolP("bool-flag", "b", false, "Description of the boolean flag")
	// String slice flag
	commandCmd.Flags().StringSliceP("slice-flag", "s", []string{}, "Description of the slice flag")

	// 3. Add flag completion if needed
	commandCmd.RegisterFlagCompletionFunc("flag-name", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Example flag completion
		completions := []string{
			"flag-value1",
			"flag-value2",
			"flag-value3",
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	})

	// 4. Add required flags if any
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

Example for an "add" command:

var addCmd = &cobra.Command{
	Use:   "add <font-name>",
	Short: "Add a font to your system",
	Long:  "Downloads and installs a font from the Google Fonts repository or other added sources.",
	Example: `  fontget add "Fira Sans"
  fontget add "Roboto" --style "Regular"
  fontget add "Open Sans" -s "Bold"`,
	Args: cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Font name completion logic
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
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
