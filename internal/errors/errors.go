package errors

import (
	"fmt"

	"github.com/spf13/cobra"
)

// ElevationRequired returns a formatted error message for when elevation is needed
func ElevationRequired(platform string) error {
	var msg string
	switch platform {
	case "windows":
		msg = "This operation requires administrator privileges. Please run the command again with administrator rights."
	case "linux", "darwin":
		msg = "This operation requires root privileges. Please run the command again with sudo."
	default:
		msg = "This operation requires elevated privileges."
	}
	return fmt.Errorf("%s", msg)
}

// HandleError prints the error message and exits with status code 1
func HandleError(cmd *cobra.Command, err error) {
	if err != nil {
		cmd.PrintErrf("Error: %v\n", err)
		cmd.Usage()
	}
}

// PrintElevationHelp prints platform-specific elevation instructions
func PrintElevationHelp(cmd *cobra.Command, platform string) {
	switch platform {
	case "windows":
		cmd.Println("To run with administrator privileges:")
		cmd.Println("1. Right-click on the command prompt or PowerShell")
		cmd.Println("2. Select 'Run as administrator'")
		cmd.Println("3. Run the command again")
	case "linux", "darwin":
		cmd.Println("To run with root privileges:")
		cmd.Printf("sudo %s\n", cmd.CommandPath())
	}
}
