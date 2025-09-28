# Shell Completions Setup

FontGet aims to support shell completions for PowerShell, Bash, and Zsh and other terminal emulators. Follow the instructions below for your shell.

## PowerShell

### Current Session
```powershell
# Enable completions for current session
fontget completion powershell | Out-String | Invoke-Expression
```

### Permanent Setup
```powershell
# Add to your PowerShell profile
Add-Content $PROFILE "fontget completion powershell | Out-String | Invoke-Expression"
```

## Bash

### Current Session
```bash
# Enable completions for current session
source <(fontget completion bash)
```

### Permanent Setup
```bash
# Add to your ~/.bashrc
echo "source <(fontget completion bash)" >> ~/.bashrc
```

## Zsh

### Current Session
```zsh
# Enable completions for current session
source <(fontget completion zsh)
```

### Permanent Setup
```zsh
# Add to your ~/.zshrc
echo "source <(fontget completion zsh)" >> ~/.zshrc
```

## Terminal Emulator Setup

For comprehensive setup instructions in different terminal emulators (Windows Terminal, Kitty, etc.), please refer to our [Terminal Setup Guide](terminal-setup.md).
