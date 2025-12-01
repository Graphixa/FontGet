// This is a template for building new commands.
// It is not used in the project - it's a development template.
// Use this as a starting point when creating new FontGet commands.

package templates

import (
	"fmt"
	"strings"

	fontgetCmd "fontget/cmd"
	"fontget/internal/cmdutils"
	"fontget/internal/output"
	"fontget/internal/repo"
	"fontget/internal/ui"

	"github.com/spf13/cobra"
)

// Template for new commands. Replace "command" with your command name
var commandCmd = &cobra.Command{
	Use:          "command <required-arg>",
	Short:        "One-line description of what the command does",
	SilenceUsage: true, // Prevents full help display on validation errors
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

		// Validate input using modern error handling pattern
		// Pattern: Print error with ui.RenderError, show hint, return nil
		if argValue == "" && flagValue == "" {
			fmt.Printf("\n%s\n", ui.RenderError("A required argument is missing"))
			fmt.Printf("Use 'fontget command --help' for more information.\n\n")
			return nil // Return nil to prevent duplicate error from Cobra
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
		// Always log operation start (file logging, not console)
		fontgetCmd.GetLogger().Info("Starting command operation")

		// Ensure manifest system is initialized (required for repository access)
		if err := cmdutils.EnsureManifestInitialized(func() cmdutils.Logger { return fontgetCmd.GetLogger() }); err != nil {
			fontgetCmd.GetLogger().Error("Failed to initialize manifest: %v", err)
			return err
		}

		// Double check args to prevent panic
		flagValue, _ := cmd.Flags().GetString("flag-name")
		var argValue string
		if len(args) > 0 {
			argValue = args[0]
		}
		if argValue == "" && flagValue == "" {
			return nil // Args validator will have already shown the error
		}

		// Log parameters (always log to file)
		fontgetCmd.GetLogger().Info("Command parameters - Arg: %s, Flag: %s", argValue, flagValue)

		// Debug-level information for developers
		output.GetDebug().Message("Debug mode enabled - showing detailed diagnostic information")

		// Print styled title using modern UI components (if needed)
		// Note: Not all commands need PageTitle - use only when appropriate
		// fmt.Printf("\n%s\n", ui.PageTitle.Render("Command Results"))

		// Verbose-level information for users
		output.GetVerbose().Info("Processing command with argument: %s", argValue)
		if flagValue != "" {
			output.GetVerbose().Info("Using flag value: %s", flagValue)
		}

		// Debug state information for developers
		output.GetDebug().State("Arguments received: %d, Flag provided: %t", len(args), flagValue != "")

		// Use optimized repository access (smart caching like search/list commands)
		r, err := repo.GetRepository()
		if err != nil {
			fontgetCmd.GetLogger().Error("Failed to initialize repository: %v", err)
			return fmt.Errorf("failed to initialize repository: %w", err)
		}

		// Get manifest from repository
		manifest, err := r.GetManifest()
		if err != nil {
			fontgetCmd.GetLogger().Error("Failed to get manifest: %v", err)
			return fmt.Errorf("failed to get manifest: %w", err)
		}

		// Example: Process fonts from manifest
		// Replace this with your actual command logic
		for _, sourceInfo := range manifest.Sources {
			for fontID, fontInfo := range sourceInfo.Fonts {
				// Example processing - replace with your actual logic
				_ = fontID
				_ = fontInfo
				// Add your processing logic here
			}
		}

		// Print results using modern UI styling
		fmt.Printf("\nFound %d items matching '%s'", 0, ui.TableSourceName.Render(argValue))
		if flagValue != "" {
			fmt.Printf(" with flag '%s'", ui.TableSourceName.Render(flagValue))
		}
		fmt.Println()

		// Verbose information about the operation
		output.GetVerbose().Info("Operation completed successfully")
		output.GetVerbose().Detail("Results", "Found %d matches", 0)

		// Debug performance information
		output.GetDebug().Performance("Operation completed in <timing>")

		// Log operation completion (always log to file)
		fontgetCmd.GetLogger().Info("Command operation completed successfully")
		return nil
	},
}

func init() {
	// 1. Add the command to the root command
	// Note: Replace rootCmd with the actual root command variable from cmd package
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

1. Copy this template to a new file in the cmd/ directory (e.g., cmd/command.go)
2. Replace "command" with your command name in the variable name and all references
3. Update the Use, Short, Long, and Example fields
4. Choose the appropriate Args validator
5. Implement the ValidArgsFunction if needed
6. Implement the RunE function with your command's logic
7. Add and configure flags in the init function
8. Add flag completion if needed
9. Mark flags as required if needed
10. For commands with subcommands, follow the sources command pattern
11. Register the command in cmd/root.go: rootCmd.AddCommand(commandCmd)

IMPORTANT NOTES:

- GetLogger() is available from cmd/root.go - when copying this template to cmd/ package, remove the "fontgetCmd" import alias and change fontgetCmd.GetLogger() to GetLogger()
- Always use SilenceUsage: true to prevent full help display on validation errors
- Always call cmdutils.EnsureManifestInitialized() before using repository
- Always log operation start, parameters, errors, and completion to file
- Use ui.RenderError() for error messages, return nil (not cmd.Help())
- Use output.GetVerbose() for user-friendly detailed output
- Use output.GetDebug() for developer diagnostic output

PERFORMANCE BEST PRACTICES:
- ALWAYS use repo.GetRepository() for normal operations (smart caching)
- ONLY use repo.GetManifest() directly when you need fresh data
- ONLY use repo.GetManifestWithRefresh() when forcing updates
- This ensures consistent performance across all commands

STYLING BEST PRACTICES:
- Use ui.RenderError() for error messages
- Use ui.TableHeader.Render() for table headers
- Use ui.TableSourceName.Render() for highlighted text
- Use ui.Text.Render() for regular text
- Use ui.SuccessText.Render() for success messages
- Use ui.WarningText.Render() for warnings
- Use ui.InfoText.Render() for info messages
- Use ui.ErrorText.Render() for error messages

LOGGING BEST PRACTICES:
- Always use GetLogger() from cmd/root.go (no placeholder needed)
- In this template, we use fontgetCmd.GetLogger() to avoid naming conflict with the cmd parameter
- When copying to cmd/ package, remove the import alias and use GetLogger() directly
- Use logger.Info() for operation start/completion and parameters
- Use logger.Error() for all errors
- Use logger.Warn() for warnings
- Use logger.Debug() for detailed debugging information
- ALWAYS log to file regardless of verbose/debug flags
- Logger level is controlled by config (ErrorLevel/InfoLevel/DebugLevel based on flags)

VERBOSE/DEBUG MODE BEST PRACTICES:
- Use output.GetVerbose().Info(format, args...) for user-friendly detailed output
- Use output.GetVerbose().Warning/Error/Success(format, args...) for different message types
- Use output.GetVerbose().Detail(prefix, format, args...) for indented details
- Use output.GetDebug().Message(format, args...) for developer diagnostic output
- Use output.GetDebug().State/Performance/Error/Warning(format, args...) for debug diagnostics
- Clean, consistent interface - no manual styling needed
- Users can combine --verbose --debug for maximum detail
- Keep normal output clean and ensure verbose/debug doesn't interfere with operation

ERROR HANDLING PATTERN:
- In Args validator: Print error with ui.RenderError(), show hint, return nil
- In RunE: Return fmt.Errorf() with wrapped errors for actual failures
- Always log errors to file with GetLogger().Error()

EXAMPLES:

// Verbose output (user-friendly)
output.GetVerbose().Info("Installing fonts to: %s", fontDir)
output.GetVerbose().Detail("Info", "Font exists at: %s", path)
output.GetVerbose().Warning("Font may be corrupted")
output.GetVerbose().Error("Installation failed: %s", err.Error())

// Debug output (developer diagnostics)
output.GetDebug().Message("Debug mode enabled - detailed diagnostics")
output.GetDebug().State("Current working directory: %s", dir)
output.GetDebug().Performance("Operation completed in %v", duration)
output.GetDebug().Error("Critical system error: %v", err)

// Error handling in Args validator
Args: func(cmd *cobra.Command, args []string) error {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		fmt.Printf("\n%s\n", ui.RenderError("A font name is required"))
		fmt.Printf("Use 'fontget command --help' for more information.\n\n")
		return nil // Prevents duplicate error from Cobra
	}
	return nil
},

// Error handling in RunE
if err != nil {
	fontgetCmd.GetLogger().Error("Operation failed: %v", err)
	return fmt.Errorf("operation failed: %w", err)
}

Standard Help Formatting:
- Use winget-style help with "usage:" line
- Include subcommands in "The following sub-commands are available:" section
- Include flags in "The following options are available:" section
- End with "For more details on a specific command, pass it the help argument. [-?]"

Table Formatting (for list/search commands):
- Use consistent column widths matching search command
- Standard columns: Name, ID, License, Categories, Source
- Use ui.Text.Render() for header separator
- Include manifest info at bottom using ui.Text.Render()

IMPORT STRUCTURE:
Follow standard Go import grouping:
1. Standard library (fmt, strings, etc.)
2. Internal packages (fontget/internal/...)
3. Third-party packages (github.com/...)

COMMAND STRUCTURE:
1. Package declaration
2. Imports (grouped: stdlib, internal, third-party)
3. Constants (if any)
4. Types (if any)
5. Command definition (var commandCmd)
6. Helper functions (if any)
7. init() function

For commands with subcommands, see cmd/sources.go for reference.
*/
