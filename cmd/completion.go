package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var completionInstallFlag bool

var completionCmd = &cobra.Command{
	Use:   "completion [shell]",
	Short: "Generate or install shell completion scripts",
	Long: `Generate or install shell completion scripts.

Supports bash, zsh, and PowerShell. Use --install to auto-install to your shell config.

Examples:
  # Generate completion script (output to stdout)
  fontget completion bash

  # Auto-detect shell and install
  fontget completion --install

  # Install for specific shell
  fontget completion bash --install

See documentation for more details.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If --install flag is set
		if completionInstallFlag {
			// If no shell specified, auto-detect
			if len(args) == 0 {
				detectedShell, err := detectShell()
				if err != nil {
					return fmt.Errorf("failed to detect shell: %w\n\nPlease specify shell: fontget completion <shell> --install", err)
				}

				fmt.Printf("Detected shell: %s\n", detectedShell)
				if err := installCompletion(detectedShell); err != nil {
					return fmt.Errorf("failed to install completion: %w", err)
				}

				fmt.Printf("✓ Completion installed successfully for %s!\n", detectedShell)
				fmt.Println("Please restart your terminal or reload your shell configuration.")
				return nil
			}

			// Shell specified, install for that shell
			shell := args[0]
			if err := installCompletion(shell); err != nil {
				return fmt.Errorf("failed to install completion: %w", err)
			}

			fmt.Printf("✓ Completion installed successfully for %s!\n", shell)
			fmt.Println("Please restart your terminal or reload your shell configuration.")
			return nil
		}

		// No --install flag, output completion script
		// Require shell argument
		if len(args) == 0 {
			return fmt.Errorf("shell required. Use: fontget completion <shell> or fontget completion --install")
		}

		shell := args[0]

		// Output the completion script (default behavior)
		switch shell {
		case "bash":
			return rootCmd.GenBashCompletion(cmd.OutOrStdout())
		case "zsh":
			return rootCmd.GenZshCompletion(cmd.OutOrStdout())
		case "powershell":
			return rootCmd.GenPowerShellCompletion(cmd.OutOrStdout())
		default:
			return fmt.Errorf("unsupported shell: %s. Supported shells: bash, zsh, powershell", shell)
		}
	},
	ValidArgs: []string{"bash", "zsh", "powershell"},
}

func init() {
	completionCmd.Flags().BoolVar(&completionInstallFlag, "install", false, "Install completion script to shell configuration file")
	rootCmd.AddCommand(completionCmd)
}

// detectShell attempts to detect the current shell
func detectShell() (string, error) {
	// Try environment variables first
	if shell := os.Getenv("SHELL"); shell != "" {
		shellName := extractShellName(shell)
		if isValidShell(shellName) {
			return shellName, nil
		}
	}

	// Try PowerShell-specific environment variable
	if runtime.GOOS == "windows" {
		if os.Getenv("PSModulePath") != "" {
			return "powershell", nil
		}
	}

	// Try detection functions
	if detected, err := detectBash(); err == nil && detected {
		return "bash", nil
	}
	if detected, err := detectZsh(); err == nil && detected {
		return "zsh", nil
	}
	if detected, err := detectPowerShell(); err == nil && detected {
		return "powershell", nil
	}

	return "", fmt.Errorf("could not detect shell")
}

// extractShellName extracts the shell name from a path (e.g., "/bin/bash" -> "bash")
func extractShellName(shellPath string) string {
	base := filepath.Base(shellPath)
	// Remove extension if present
	if ext := filepath.Ext(base); ext != "" {
		base = strings.TrimSuffix(base, ext)
	}
	return strings.ToLower(base)
}

// isValidShell checks if a shell name is supported
func isValidShell(shell string) bool {
	validShells := []string{"bash", "zsh", "powershell"}
	for _, v := range validShells {
		if shell == v {
			return true
		}
	}
	return false
}

// detectBash detects if bash is the current shell
func detectBash() (bool, error) {
	if runtime.GOOS == "windows" {
		// On Windows, check if Git Bash or WSL bash
		if os.Getenv("MSYSTEM") != "" {
			return true, nil
		}
		// Check if running in WSL
		if _, err := os.Stat("/proc/version"); err == nil {
			if data, err := os.ReadFile("/proc/version"); err == nil {
				if strings.Contains(strings.ToLower(string(data)), "microsoft") {
					return true, nil
				}
			}
		}
		return false, nil
	}

	// On Unix-like systems, check SHELL or try to detect
	shell := os.Getenv("SHELL")
	if shell != "" {
		return strings.Contains(strings.ToLower(shell), "bash"), nil
	}

	// Try to run bash --version
	cmd := exec.Command("bash", "--version")
	if err := cmd.Run(); err == nil {
		return true, nil
	}

	return false, nil
}

// detectZsh detects if zsh is the current shell
func detectZsh() (bool, error) {
	shell := os.Getenv("SHELL")
	if shell != "" {
		return strings.Contains(strings.ToLower(shell), "zsh"), nil
	}

	// Try to run zsh --version
	cmd := exec.Command("zsh", "--version")
	if err := cmd.Run(); err == nil {
		return true, nil
	}

	return false, nil
}

// detectPowerShell detects if PowerShell is the current shell
func detectPowerShell() (bool, error) {
	if runtime.GOOS == "windows" {
		// Check for PowerShell-specific environment variables
		if os.Getenv("PSModulePath") != "" {
			return true, nil
		}
		// Try to run pwsh or powershell
		for _, cmdName := range []string{"pwsh", "powershell"} {
			cmd := exec.Command(cmdName, "-Command", "exit")
			if err := cmd.Run(); err == nil {
				return true, nil
			}
		}
	}
	return false, nil
}

// expandPath expands ~ to home directory and environment variables
func expandPath(path string) (string, error) {
	// Expand ~
	if strings.HasPrefix(path, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		path = strings.Replace(path, "~", homeDir, 1)
	}

	// Expand environment variables
	path = os.ExpandEnv(path)

	return path, nil
}

// installCompletion installs completion for the specified shell
func installCompletion(shellName string) error {
	// Validate shell
	if !isValidShell(shellName) {
		return fmt.Errorf("unsupported shell: %s. Supported shells: bash, zsh, powershell", shellName)
	}

	// PowerShell is handled differently (inline in profile)
	if shellName == "powershell" {
		return installPowerShellCompletion()
	}

	// Generate completion script for bash/zsh
	script, err := generateCompletionScript(shellName)
	if err != nil {
		return fmt.Errorf("failed to generate completion script: %w", err)
	}

	// Get shell-specific paths
	var completionPath, configPath, sourceLine string
	switch shellName {
	case "bash":
		completionPath = "~/.fontget-completion.bash"
		configPath = "~/.bashrc"
		sourceLine = "source ~/.fontget-completion.bash"
	case "zsh":
		completionPath = "~/.zsh/completions/_fontget"
		configPath = "~/.zshrc"
		sourceLine = "fpath=(~/.zsh/completions $fpath)"
	}

	// Expand paths
	completionPathExpanded, err := expandPath(completionPath)
	if err != nil {
		return fmt.Errorf("failed to expand completion file path: %w", err)
	}

	configPathExpanded, err := expandPath(configPath)
	if err != nil {
		return fmt.Errorf("failed to expand config file path: %w", err)
	}

	// Create directory for completion file if needed
	if shellName == "zsh" {
		completionDir := filepath.Dir(completionPathExpanded)
		if err := os.MkdirAll(completionDir, 0755); err != nil {
			return fmt.Errorf("failed to create completion directory: %w", err)
		}
	}

	// Write completion script
	if err := os.WriteFile(completionPathExpanded, script, 0644); err != nil {
		return fmt.Errorf("failed to write completion file: %w", err)
	}

	// Update shell config file
	if err := updateShellConfig(configPathExpanded, sourceLine, shellName); err != nil {
		return fmt.Errorf("failed to update shell config: %w", err)
	}

	return nil
}

// generateCompletionScript generates the completion script for a shell
func generateCompletionScript(shellName string) ([]byte, error) {
	// Create a temporary file to capture output
	tmpFile, err := os.CreateTemp("", "fontget-completion-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Generate completion to temp file
	switch shellName {
	case "bash":
		if err := rootCmd.GenBashCompletion(tmpFile); err != nil {
			return nil, fmt.Errorf("failed to generate bash completion: %w", err)
		}
	case "zsh":
		if err := rootCmd.GenZshCompletion(tmpFile); err != nil {
			return nil, fmt.Errorf("failed to generate zsh completion: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported shell: %s", shellName)
	}

	// Read the generated script
	if err := tmpFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temp file: %w", err)
	}

	script, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read generated script: %w", err)
	}

	return script, nil
}

// installPowerShellCompletion installs PowerShell completion (inline in profile)
func installPowerShellCompletion() error {
	// Get PowerShell profile path
	profilePath := os.ExpandEnv("$PROFILE")
	if profilePath == "$PROFILE" {
		// $PROFILE wasn't expanded, try to get it from PowerShell
		cmd := exec.Command("powershell", "-Command", "$PROFILE")
		output, err := cmd.Output()
		if err != nil {
			// Try pwsh
			cmd = exec.Command("pwsh", "-Command", "$PROFILE")
			output, err = cmd.Output()
		}
		if err != nil {
			return fmt.Errorf("failed to get PowerShell profile path: %w", err)
		}
		profilePath = strings.TrimSpace(string(output))
	}

	// Create profile directory if it doesn't exist
	profileDir := filepath.Dir(profilePath)
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		return fmt.Errorf("failed to create profile directory: %w", err)
	}

	// Check if already installed
	existingContent, _ := os.ReadFile(profilePath)
	if strings.Contains(string(existingContent), "fontget completion powershell") {
		return fmt.Errorf("completion already installed in PowerShell profile")
	}

	// Add completion line to profile
	sourceLine := "fontget completion powershell | Out-String | Invoke-Expression"
	content := string(existingContent)
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += "\n# FontGet completion\n"
	content += sourceLine + "\n"

	if err := os.WriteFile(profilePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write PowerShell profile: %w", err)
	}

	return nil
}

// updateShellConfig adds the source line to the shell config file if not already present
func updateShellConfig(configPath, sourceLine, shellName string) error {
	// Read existing config
	existingContent, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	content := string(existingContent)

	// Check if already installed
	if strings.Contains(content, "fontget completion") || strings.Contains(content, "_fontget") {
		return fmt.Errorf("completion already installed in %s", configPath)
	}

	// For zsh, check if fpath line already exists
	if shellName == "zsh" {
		// Check if fpath for our completions directory already exists
		if strings.Contains(content, "~/.zsh/completions") || strings.Contains(content, "$HOME/.zsh/completions") {
			// Already configured, don't add duplicate
			return nil
		}
	}

	// Add source line
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	content += "\n# FontGet completion\n"
	content += sourceLine + "\n"

	// Write updated config
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
