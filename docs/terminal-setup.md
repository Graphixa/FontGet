# Terminal Setup Guide

This guide provides detailed instructions for setting up FontGet completions in various terminal emulators and shells.

## Table of Contents

- [Windows Terminal](#windows-terminal)
- [PowerShell](#powershell)
- [Git Bash](#git-bash)
- [WSL (Windows Subsystem for Linux)](#wsl)
- [Kitty Terminal](#kitty-terminal)
- [iTerm2](#iterm2)
- [Alacritty](#alacritty)
- [WezTerm](#wezterm)

## Windows Terminal

Windows Terminal supports multiple shells. Follow the instructions for your preferred shell:

### PowerShell in Windows Terminal

1. Open Windows Terminal
2. Open PowerShell
3. Enable completions:
   ```powershell
   # For current session
   fontget completion powershell | Out-String | Invoke-Expression

   # For permanent setup
   Add-Content $PROFILE "fontget completion powershell | Out-String | Invoke-Expression"
   ```

### Command Prompt in Windows Terminal

Command Prompt doesn't support completions. Consider using PowerShell or WSL for a better experience.

## PowerShell

### PowerShell 5.1 (Windows PowerShell)

1. Open PowerShell
2. Enable completions:
   ```powershell
   # For current session
   fontget completion powershell | Out-String | Invoke-Expression

   # For permanent setup
   Add-Content $PROFILE "fontget completion powershell | Out-String | Invoke-Expression"
   ```

### PowerShell 7+ (PowerShell Core)

1. Open PowerShell
2. Enable completions:
   ```powershell
   # For current session
   fontget completion powershell | Out-String | Invoke-Expression

   # For permanent setup
   Add-Content $PROFILE "fontget completion powershell | Out-String | Invoke-Expression"
   ```

## Git Bash

1. Open Git Bash
2. Enable completions:
   ```bash
   # For current session
   source <(fontget completion bash)

   # For permanent setup
   echo "source <(fontget completion bash)" >> ~/.bashrc
   ```

## WSL (Windows Subsystem for Linux)

### Ubuntu/Debian

1. Open WSL
2. Enable completions:
   ```bash
   # For current session
   source <(fontget completion bash)

   # For permanent setup
   echo "source <(fontget completion bash)" >> ~/.bashrc
   ```

### Zsh in WSL

1. Open WSL
2. Enable completions:
   ```zsh
   # For current session
   source <(fontget completion zsh)

   # For permanent setup
   echo "source <(fontget completion zsh)" >> ~/.zshrc
   ```

## Kitty Terminal

Kitty Terminal supports multiple shells. Follow the instructions for your preferred shell:

### Bash in Kitty

1. Open Kitty
2. Enable completions:
   ```bash
   # For current session
   source <(fontget completion bash)

   # For permanent setup
   echo "source <(fontget completion bash)" >> ~/.bashrc
   ```

### Zsh in Kitty

1. Open Kitty
2. Enable completions:
   ```zsh
   # For current session
   source <(fontget completion zsh)

   # For permanent setup
   echo "source <(fontget completion zsh)" >> ~/.zshrc
   ```

## iTerm2

### Bash in iTerm2

1. Open iTerm2
2. Enable completions:
   ```bash
   # For current session
   source <(fontget completion bash)

   # For permanent setup
   echo "source <(fontget completion bash)" >> ~/.bashrc
   ```

### Zsh in iTerm2

1. Open iTerm2
2. Enable completions:
   ```zsh
   # For current session
   source <(fontget completion zsh)

   # For permanent setup
   echo "source <(fontget completion zsh)" >> ~/.zshrc
   ```

## Alacritty

Alacritty supports multiple shells. Follow the instructions for your preferred shell:

### Bash in Alacritty

1. Open Alacritty
2. Enable completions:
   ```bash
   # For current session
   source <(fontget completion bash)

   # For permanent setup
   echo "source <(fontget completion bash)" >> ~/.bashrc
   ```

### Zsh in Alacritty

1. Open Alacritty
2. Enable completions:
   ```zsh
   # For current session
   source <(fontget completion zsh)

   # For permanent setup
   echo "source <(fontget completion zsh)" >> ~/.zshrc
   ```

## WezTerm

WezTerm supports multiple shells. Follow the instructions for your preferred shell:

### Bash in WezTerm

1. Open WezTerm
2. Enable completions:
   ```bash
   # For current session
   source <(fontget completion bash)

   # For permanent setup
   echo "source <(fontget completion bash)" >> ~/.bashrc
   ```

### Zsh in WezTerm

1. Open WezTerm
2. Enable completions:
   ```zsh
   # For current session
   source <(fontget completion zsh)

   # For permanent setup
   echo "source <(fontget completion zsh)" >> ~/.zshrc
   ```

## Troubleshooting

If completions are not working:

1. Verify that the completion script is being sourced correctly
2. Check if your shell's completion system is working with other commands
3. Try restarting your terminal
4. Make sure FontGet is in your PATH
5. Check for any error messages when sourcing the completion script

## Contributing

If you'd like to add instructions for another terminal emulator or shell, please submit a pull request. 