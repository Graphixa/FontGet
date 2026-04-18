# Terminal Setup Guide

This guide provides quick links to set up FontGet completions in popular terminal emulators. Refer to the section that matches your terminal's current shell (e.g. Bash, Zsh, Powershell or Fish)

## Table of Contents

- [Quick Start](#quick-start)
- [Setup by Shell](#setup-by-shell)
- [Which shell am I using?](#which-shell-am-i-using)
- [Platform notes](#platform-notes) (includes [clickable URLs](#any-terminal-emulator))
- [Troubleshooting](#troubleshooting)
- [Additional Resources](#additional-resources)

## Quick Start

**Easiest method — detects your shell and installs completions**

```bash
fontget completion --install
```

Works from any terminal once `fontget` is on your `PATH`.

## Setup by Shell

Use the section for the shell you run (not the name of the terminal app).

### PowerShell

**Automatic installation (recommended):**

```powershell
fontget completion powershell --install
```

**Current session only (before a permanent install):**

```powershell
fontget completion powershell | Out-String | Invoke-Expression
```

### Bash {#bash}

**Automatic installation (recommended):**

```bash
fontget completion bash --install
```

**Current session only:**

```bash
source <(fontget completion bash)
```

### Zsh {#zsh}

**Automatic installation (recommended):**

```bash
fontget completion zsh --install
```

**Current session only:**

```zsh
source <(fontget completion zsh)
```

**Zsh note:** The installer appends a line that adds `~/.zsh/completions` to `fpath`. That line must run **before** `compinit` in `~/.zshrc`, or completions may not load until you run `compinit` again (or remove `~/.zcompdump` and restart the shell). If completions are missing after install, move the FontGet `fpath` block above your `compinit` call.

### Fish {#fish}

Fish loads completions from `~/.config/fish/completions/` automatically; no edit to `config.fish` is required.

**Automatic installation (recommended):**

```bash
fontget completion fish --install
```

**Current session only:**

```fish
fontget completion fish | source
```

Ensure `fontget` is on your `PATH` (`which fontget`).

## Which shell am I using?

```bash
echo "$SHELL"
```

On **macOS**, Apple’s default login shell is **zsh** (Catalina 10.15 and later) and was **bash** on older releases. Your terminal app’s preferences may override the default.

On **Windows**, use PowerShell, Git Bash, or WSL—each matches one of the shell sections above.

## Platform notes

### Windows

- **Windows Terminal** can host PowerShell, Command Prompt, or WSL—open the profile that matches how you use FontGet.
- **Command Prompt** does not support these completions; use **PowerShell** or **WSL** instead.
- **Git Bash** and **WSL** use **Bash** completions ([Bash](#bash)).

### macOS

- **Login shells** often read `~/.bash_profile` instead of `~/.bashrc`. If you installed Bash completions but they never load, ensure `~/.bashrc` is sourced from `~/.bash_profile` (e.g. `[[ -f ~/.bashrc ]] && source ~/.bashrc`).

### Any terminal emulator

There is nothing special to configure per app (Kitty, iTerm2, Alacritty, WezTerm, etc.). Pick **PowerShell**, **Bash**, **Zsh**, or **Fish** according to the shell your profile runs.

**Clickable URLs (OSC 8):** Commands such as `fontget info` render `http`/`https` links using terminal hyperlinks. In supporting terminals (e.g. recent **Windows Terminal**, **iTerm2**, **Kitty**, **WezTerm**, **Ghostty**), you can **ctrl+click** (or **cmd+click**) the underlined URL to open it in a browser. Other terminals still show the URL with normal styling; behavior depends on the emulator.

## Troubleshooting

If completions are not working:

1. **Verify installation:**
   ```bash
   ls ~/.fontget-completion.bash
   ls ~/.zsh/completions/_fontget
   ls ~/.config/fish/completions/fontget.fish
   ```

2. **Check shell configuration:**
   ```bash
   grep "fontget" ~/.bashrc
   grep "fontget" ~/.zshrc
   ```
   Fish does not need a source line; the file under `~/.config/fish/completions/` is enough.

3. **Reload your shell:**
   ```bash
   source ~/.bashrc   # Bash
   source ~/.zshrc    # Zsh
   ```
   Or restart the terminal. See [Platform notes](#platform-notes) for login-shell behavior on macOS.

4. **Confirm FontGet is on your PATH:**
   ```bash
   which fontget
   fontget version
   ```

5. **Watch for errors** when sourcing or running completion commands.

### PowerShell-specific issues

1. **Profile not loading:**
   ```powershell
   Test-Path $PROFILE
   if (!(Test-Path $PROFILE)) {
       New-Item -Path $PROFILE -Type File -Force
   }
   ```

2. **Execution policy:**
   ```powershell
   Get-ExecutionPolicy
   # If needed (may require admin):
   Set-ExecutionPolicy RemoteSigned
   ```

### Zsh-specific issues

1. **Completion system not initialized:**
   ```zsh
   grep "compinit" ~/.zshrc
   echo "autoload -Uz compinit && compinit" >> ~/.zshrc
   ```

2. **`fpath` after `compinit`:** Ensure `fpath=(~/.zsh/completions $fpath)` runs before `compinit`, or run `compinit` again after fixing order (see [Zsh](#zsh) above).

### Fish-specific issues

1. **Completion file missing or stale:** Re-run `fontget completion fish --install`, or remove `~/.config/fish/completions/fontget.fish` and install again.

---

## Additional Resources

- [FontGet usage / commands](usage.md)

---

## Contributing

Improvements to shell- or platform-specific notes are welcome via pull request.
