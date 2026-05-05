package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"fontget/internal/cmdutils"
	"fontget/internal/ui"

	"github.com/spf13/cobra"
)

var completionInstallFlag bool

// Markers for idempotent installs (avoid broad substring checks like "_fontget").
const (
	zshFpathMarker       = "# fontget: zsh completion fpath (managed by fontget install)"
	bashCompletionMarker = "# fontget: bash completion source (managed by fontget install)"
)

var completionCmd = &cobra.Command{
	Use:   "completion [shell]",
	Short: "Generate or install shell completion scripts",
	Long: `Generate or install shell completion scripts.

Supports bash, zsh, fish, and PowerShell.
Use --install to auto-install completion for your shell.
When no shell is provided with --install, FontGet auto-detects your current shell.

For zsh, --install prepends a small fpath block to the top of ~/.zshrc so completions load before compinit (required for Oh My Zsh and similar setups on macOS).`,
	Example: `  fontget completion bash
  fontget completion --install
  fontget completion bash --install`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If --install flag is set
		if completionInstallFlag {
			// If no shell specified, auto-detect
			if len(args) == 0 {
				detectedShell, err := detectShell()
				if err != nil {
					return fmt.Errorf("failed to detect shell: %w\n\nPlease specify shell: fontget completion <shell> --install", err)
				}

				cmdutils.PrintInfof("Detected shell: %s", detectedShell)
				completionFile, shellConfig, err := installCompletion(detectedShell)
				if err != nil {
					return fmt.Errorf("failed to install completion: %w", err)
				}
				printCompletionInstallSummary(cmd.OutOrStdout(), detectedShell, completionFile, shellConfig)
				return nil
			}

			// Shell specified, install for that shell
			shell := args[0]
			completionFile, shellConfig, err := installCompletion(shell)
			if err != nil {
				return fmt.Errorf("failed to install completion: %w", err)
			}
			printCompletionInstallSummary(cmd.OutOrStdout(), shell, completionFile, shellConfig)
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
		case "fish":
			return rootCmd.GenFishCompletion(cmd.OutOrStdout(), true)
		case "powershell":
			return rootCmd.GenPowerShellCompletion(cmd.OutOrStdout())
		default:
			return fmt.Errorf("unsupported shell: %s. Supported shells: bash, zsh, fish, powershell", shell)
		}
	},
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
}

func init() {
	completionCmd.Flags().BoolVar(&completionInstallFlag, "install", false, "Install completion script to shell configuration file")
	rootCmd.AddCommand(completionCmd)
}

// printCompletionInstallSummary prints human-readable paths after --install.
// Raw completion script output (without --install) must stay machine-parseable; only this path uses styling.
func printCompletionInstallSummary(out io.Writer, shell, completionFile, shellConfig string) {
	fmt.Fprintln(out)
	fmt.Fprintf(out, "  %s %s\n",
		ui.SuccessText.Render("✓"),
		ui.Text.Render(fmt.Sprintf("Shell completion installed for %s.", shell)))
	if completionFile != "" {
		fmt.Fprintf(out, "  %s %s\n", ui.TextBold.Render("Completion file:"), ui.InfoText.Render(completionFile))
	}
	if shellConfig != "" {
		label := "Shell config:"
		if shell == "powershell" {
			label = "PowerShell profile:"
		}
		fmt.Fprintf(out, "  %s %s\n", ui.TextBold.Render(label), ui.InfoText.Render(shellConfig))
	} else if shell == "fish" {
		fmt.Fprintf(out, "  %s\n", ui.Text.Render("Fish loads completions from that path automatically; config.fish was not modified."))
	}
	fmt.Fprintf(out, "\n%s\n", ui.Text.Render("Open a new terminal tab or reload your shell configuration for tab completion to take effect."))
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
	if detected, err := detectFish(); err == nil && detected {
		return "fish", nil
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
	validShells := []string{"bash", "zsh", "fish", "powershell"}
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

// detectFish detects if fish is the current shell
func detectFish() (bool, error) {
	shell := os.Getenv("SHELL")
	if shell != "" {
		return strings.Contains(strings.ToLower(shell), "fish"), nil
	}

	cmd := exec.Command("fish", "--version")
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

// installCompletion installs completion for the specified shell.
// It returns absolute paths: completion script file (if any), and shell rc/profile updated (if any).
func installCompletion(shellName string) (completionFile, shellConfig string, err error) {
	// Validate shell
	if !isValidShell(shellName) {
		return "", "", fmt.Errorf("unsupported shell: %s. Supported shells: bash, zsh, fish, powershell", shellName)
	}

	// PowerShell is handled differently (inline in profile)
	if shellName == "powershell" {
		return installPowerShellCompletion()
	}

	if shellName == "fish" {
		return installFishCompletion()
	}

	// Generate completion script for bash/zsh
	script, err := generateCompletionScript(shellName)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate completion script: %w", err)
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
		sourceLine = "" // zsh: fpath block is injected via prependZshFpathBlock
	}

	// Expand paths
	completionPathExpanded, err := expandPath(completionPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to expand completion file path: %w", err)
	}

	configPathExpanded, err := expandPath(configPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to expand config file path: %w", err)
	}

	// Create directory for completion file if needed
	if shellName == "zsh" {
		completionDir := filepath.Dir(completionPathExpanded)
		if err := os.MkdirAll(completionDir, 0755); err != nil {
			return "", "", fmt.Errorf("failed to create completion directory: %w", err)
		}
	}

	// Write completion script
	if err := os.WriteFile(completionPathExpanded, script, 0644); err != nil {
		return "", "", fmt.Errorf("failed to write completion file: %w", err)
	}

	// Update shell config file (zsh prepends fpath so it runs before compinit / oh-my-zsh)
	if err := updateShellConfig(configPathExpanded, sourceLine, shellName); err != nil {
		return "", "", fmt.Errorf("failed to update shell config: %w", err)
	}

	return completionPathExpanded, configPathExpanded, nil
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
	case "fish":
		if err := rootCmd.GenFishCompletion(tmpFile, true); err != nil {
			return nil, fmt.Errorf("failed to generate fish completion: %w", err)
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

// installFishCompletion writes the fish completion script; Fish loads ~/.config/fish/completions/*.fish automatically.
func installFishCompletion() (completionFile, shellConfig string, err error) {
	completionPath := "~/.config/fish/completions/fontget.fish"
	completionPathExpanded, err := expandPath(completionPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to expand completion file path: %w", err)
	}

	if data, err := os.ReadFile(completionPathExpanded); err == nil {
		if len(data) > 0 && strings.Contains(string(data), "fish completion for fontget") {
			return "", "", fmt.Errorf("completion already installed at %s", completionPathExpanded)
		}
	} else if !os.IsNotExist(err) {
		return "", "", fmt.Errorf("failed to read existing fish completion file: %w", err)
	}

	completionDir := filepath.Dir(completionPathExpanded)
	if err := os.MkdirAll(completionDir, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create fish completions directory: %w", err)
	}

	script, err := generateCompletionScript("fish")
	if err != nil {
		return "", "", fmt.Errorf("failed to generate completion script: %w", err)
	}

	if err := os.WriteFile(completionPathExpanded, script, 0644); err != nil {
		return "", "", fmt.Errorf("failed to write fish completion file: %w", err)
	}

	return completionPathExpanded, "", nil
}

// installPowerShellCompletion installs PowerShell completion (inline in profile).
// Returns ("", profilePath) because there is no separate completion script file.
func installPowerShellCompletion() (completionFile, shellConfig string, err error) {
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
			return "", "", fmt.Errorf("failed to get PowerShell profile path: %w", err)
		}
		profilePath = strings.TrimSpace(string(output))
	}

	// Create profile directory if it doesn't exist
	profileDir := filepath.Dir(profilePath)
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create profile directory: %w", err)
	}

	// Check if already installed
	existingContent, _ := os.ReadFile(profilePath)
	if strings.Contains(string(existingContent), "fontget completion powershell") {
		return "", "", fmt.Errorf("completion already installed in PowerShell profile")
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
		return "", "", fmt.Errorf("failed to write PowerShell profile: %w", err)
	}

	return "", profilePath, nil
}

// updateShellConfig adds completion hooks to the shell config file.
// For zsh, the fpath snippet is prepended so it runs before compinit (required on macOS with Oh My Zsh, etc.).
func updateShellConfig(configPath, sourceLine, shellName string) error {
	existingContent, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	content := string(existingContent)

	switch shellName {
	case "zsh":
		if strings.Contains(content, zshFpathMarker) {
			return fmt.Errorf("completion already installed in %s", configPath)
		}
		newContent := prependZshFpathBlock(content)
		if err := os.WriteFile(configPath, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}
		return nil
	case "bash":
		if strings.Contains(content, bashCompletionMarker) {
			return fmt.Errorf("completion already installed in %s", configPath)
		}
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += "\n" + bashCompletionMarker + "\n"
		content += sourceLine + "\n"
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("updateShellConfig: unsupported shell %q", shellName)
	}
}

// prependZshFpathBlock prepends the completions directory to fpath before the rest of .zshrc.
// If fpath is only extended after compinit has run, zsh will not load _fontget until compinit is re-run.
func prependZshFpathBlock(existingZshrc string) string {
	block := zshFpathMarker + "\n" +
		`# Must run before compinit / oh-my-zsh (see fontget completion zsh --help).` + "\n" +
		`fpath=("${HOME}/.zsh/completions" $fpath)` + "\n" +
		"\n"

	if existingZshrc == "" {
		return block
	}
	return block + existingZshrc
}
