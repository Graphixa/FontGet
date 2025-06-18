package cmd

import (
	"fmt"
	"runtime"

	"fontget/internal/platform"

	"github.com/spf13/cobra"
)

// printElevationHelp prints platform-specific elevation instructions
func printElevationHelp(cmd *cobra.Command, platform string) {
	switch platform {
	case "windows":
		cmd.Println("To run with administrator privileges:")
		cmd.Println("1. Right-click on the command prompt or PowerShell")
		cmd.Println("2. Select 'Run as administrator'")
		cmd.Println("3. Run the command again")
	case "macos":
		cmd.Println("To run with root privileges:")
		cmd.Println("1. Open Terminal")
		cmd.Println("2. Run 'sudo fontget'")
		cmd.Println("3. Enter your password when prompted")
		cmd.Println("4. Run the command again")
	case "linux", "darwin":
		cmd.Println("To run with root privileges:")
		cmd.Printf("sudo %s\n", cmd.CommandPath())
	}
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
			return fmt.Errorf("this command requires elevated privileges; please re-run as administrator/root")
		}
	}
	return nil
}
