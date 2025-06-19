package cmd

import (
	"fmt"
	"runtime"

	"fontget/internal/platform"

	"github.com/spf13/cobra"
)

// printElevationHelp prints platform-specific elevation instructions
func printElevationHelp(cmd *cobra.Command, platform string) {
	fmt.Println()
	switch platform {
	case "windows":
		cmd.Println("This operation requires administrator privileges.")
		cmd.Println("To run as administrator:")
		cmd.Println("  1. Right-click on Command Prompt or PowerShell.")
		cmd.Println("  2. Select 'Run as administrator'.")
		cmd.Printf("  3. Run: %s\n", cmd.CommandPath())
	case "darwin", "linux":
		cmd.Println("This operation requires root privileges.")
		cmd.Println("To run as root, prepend 'sudo' to your command, for example:")
		cmd.Printf("  sudo %s\n", cmd.CommandPath())
	default:
		cmd.Println("This operation requires elevated privileges. Please re-run as administrator or root.")
	}
	fmt.Println()
}

// checkElevation checks if the current process has elevated privileges
// and prints help if elevation is required but not present
func checkElevation(cmd *cobra.Command, fontManager platform.FontManager, scope platform.InstallationScope) error {
	if fontManager.RequiresElevation(scope) {
		// Create elevation manager
		elevationManager, err := platform.NewElevationManager()
		if err != nil {
			return fmt.Errorf("failed to initialize elevation manager: %w", err)
		}

		// Check if already elevated
		elevated, err := elevationManager.IsElevated()
		if err != nil {
			return fmt.Errorf("failed to check elevation status: %w", err)
		}

		if !elevated {
			// Print help message
			printElevationHelp(cmd, runtime.GOOS)
			return fmt.Errorf("this command requires elevated privileges. Please follow the instructions above to re-run as administrator/root")
		}
	}
	return nil
}
