# Terminal Setup Guide

This guide provides quick links to set up FontGet completions in popular terminal emulators. Each terminal emulator section links to the detailed shell-specific instructions.

## Table of Contents

- [Windows Terminal](#windows-terminal)
- [PowerShell](#powershell)
- [Git Bash](#git-bash)
- [WSL (Windows Subsystem for Linux)](#wsl-windows-subsystem-for-linux)
- [macOS Terminal](#macos-terminal)
- [Kitty Terminal](#kitty-terminal)
- [iTerm2](#iterm2)
- [Alacritty](#alacritty)
- [WezTerm](#wezterm)
- [Troubleshooting](#troubleshooting)


## Quick Start

**Easiest method - detects shell automatically and installs**

```bash
# Auto-detect your shell and install
fontget completion --install
```
>Should work in any terminal (macOS Terminal, Kitty, Powershell etc.)

## Setup by Shell
Alternatively you can specify your shell manually.

### PowerShell

**Automatic Installation (Recommended):**
```powershell
fontget completion powershell --install
```

**Test in Current Session (before permanent install):**
```powershell
# Enable completions for current session only
fontget completion powershell | Out-String | Invoke-Expression
```

### Bash {#bash}

**Automatic Installation (Recommended):**
```bash
fontget completion bash --install
```

**Test in Current Session (before permanent install):**
```bash
# Enable completions for current session only
source <(fontget completion bash)
```

### Zsh {#zsh}

**Automatic Installation (Recommended):**
```bash
fontget completion zsh --install
```

**Test in Current Session (before permanent install):**
```zsh
# Enable completions for current session only
source <(fontget completion zsh)
```

## Windows Terminal

Windows Terminal supports multiple shells. Follow the instructions for your preferred shell:

### Windows Terminal (PowerShell)

👉 **[Goto PowerShell Setup Instructions →](#powershell)**

### Windows Terminal (WSL or Git Bash)

👉 **[Goto Bash Setup Instructions →](#bash)**

>[!Note]
> Command Prompt doesn't support completions. Use PowerShell or WSL instead.

## macOS Terminal

### macOS Terminal (macOS Mojave 10.14 and earlier)

Uses **Bash** completions.

👉 **[Goto Bash Setup Instructions →](#bash)**

### macOS Terminal (macOS Catalina 10.15 and later)

Uses **Zsh** completions.

👉 **[Goto Zsh Setup Instructions →](#zsh)**


## Kitty Terminal

Kitty Terminal supports multiple shells. Follow the instructions for your preferred shell.

### Kitty Terminal (Bash)

👉 **[Goto Bash Setup Instructions →](#bash)**

### Kitty Terminal (Zsh)

👉 **[Goto Zsh Setup Instructions →](#zsh)**

---

## iTerm2

iTerm2 supports multiple shells. Follow the instructions for your preferred shell.

### iTerm2 (Bash)

👉 **[Goto Bash Setup Instructions →](#bash)**

### iTerm2 (Zsh)

👉 **[Goto Zsh Setup Instructions →](#zsh)**

---

## Alacritty

Alacritty supports multiple shells. Follow the instructions for your preferred shell.

### Alacritty (Bash)

👉 **[Goto Bash Setup Instructions →](#bash)**

### Alacritty (Zsh)

👉 **[Goto Zsh Setup Instructions →](#zsh)**

---

## WezTerm

WezTerm supports multiple shells. Follow the instructions for your preferred shell.

### WezTerm (Bash)

👉 **[Goto Bash Setup Instructions →](#bash)**

### WezTerm (Zsh)

👉 **[Goto Zsh Setup Instructions →](#zsh)**

---

## Troubleshooting

If completions are not working:

1. **Verify installation:**
   ```bash
   # Check if completion file exists
   ls ~/.fontget-completion.bash  # Bash
   ls ~/.zsh/completions/_fontget  # Zsh
   ```

2. **Check shell configuration:**
   ```bash
   # Verify source line is in config file
   grep "fontget" ~/.bashrc  # Bash
   grep "fontget" ~/.zshrc   # Zsh
   ```

3. **Reload your shell:**
   ```bash
   source ~/.bashrc  # Bash
   source ~/.zshrc   # Zsh
   # Or restart your terminal
   ```

4. **Make sure FontGet is in your PATH:**
   ```bash
   which fontget
   fontget version
   ```

5. **Check for any error messages when sourcing the completion script**

### PowerShell-Specific Issues

1. **Profile not loading:**
   ```powershell
   # Check if profile exists
   Test-Path $PROFILE
   
   # Create profile if it doesn't exist
   if (!(Test-Path $PROFILE)) {
       New-Item -Path $PROFILE -Type File -Force
   }
   ```

2. **Execution policy:**
   ```powershell
   # Check execution policy
   Get-ExecutionPolicy
   
   # If needed, set to RemoteSigned (requires admin)
   Set-ExecutionPolicy RemoteSigned
   ```

### Zsh-Specific Issues

1. **Completion system not initialized:**
   ```zsh
   # Make sure compinit is in your .zshrc
   grep "compinit" ~/.zshrc
   
   # If not, add it
   echo "autoload -Uz compinit && compinit" >> ~/.zshrc
   ```

---

## Additional Resources

- [FontGet Help](help.md) - Complete command reference

---

## Contributing

If you'd like to add instructions for another terminal emulator or shell, please submit a pull request.
