# FontGet Installation Guide

This document provides complete installation instructions for FontGet on all supported platforms. FontGet is a single binary with no external dependencies; binaries are available for both `amd64` and `arm64` architectures.

## Quick Reference

| Method | Platform | Section |
|--------|----------|---------|
| Install script | Mac, Linux, Windows | [Install latest version](#install-latest-version) |
| Package manager | Windows, macOS, Linux | [Install via package manager](#install-via-package-manager) |
| Build from source | All | [Build from source](#build-and-install-from-source) |
| Automation / CI | All | [Automation / CI](#automation--ci) |

---

## Install Latest Version

Official installers pull a release binary from GitHub and verify it against **`checksums.txt`** before writing to disk. Default paths are per-user and do not require administrator privileges.

### Shell (macOS, Linux)

```sh
curl -fsSL https://raw.githubusercontent.com/Graphixa/FontGet/main/scripts/install.sh | sh
```

Flags (after `sh -s --`), environment variables, and CI behaviour: [Automation / CI](#automation--ci). 
To pin a release, see [Install specific version](#install-specific-version).

### PowerShell (Windows)

```powershell
irm https://raw.githubusercontent.com/Graphixa/FontGet/main/scripts/install.ps1 | iex
```

To pin a release, see [Install specific version](#install-specific-version).

## Install Specific Version

Install a specific version by setting the `FONTGET_VERSION` environment variable.

### Shell (Mac, Linux)

```sh
FONTGET_VERSION=2.2.0 curl -fsSL https://raw.githubusercontent.com/Graphixa/FontGet/main/scripts/install.sh | sh
```

### PowerShell (Windows)

```powershell
$env:FONTGET_VERSION="2.2.0"; irm https://raw.githubusercontent.com/Graphixa/FontGet/main/scripts/install.ps1 | iex
```

## Install via Package Manager

### Windows

**[WinGet](https://learn.microsoft.com/en-us/windows/package-manager/winget/)**

```powershell
winget install --id "Graphixa.FontGet"
```

**[Scoop](https://scoop.sh/)**

```powershell
scoop bucket add fontget https://github.com/Graphixa/scoop-bucket
scoop install fontget
```

> [!NOTE]
> Add the bucket first; FontGet is not in the main Scoop repository.

**[Chocolatey](https://chocolatey.org/)**

```powershell
choco install fontget
```

### macOS

**[Homebrew](https://formulae.brew.sh/)**

```sh
brew tap Graphixa/homebrew-tap
brew install fontget
```

> [!NOTE]
> Add the tap first; FontGet is not in the main Homebrew repository yet.

### Linux

**Debian/Ubuntu (`.deb` package)**

Visit the [GitHub Releases page](https://github.com/Graphixa/FontGet/releases/latest) and download the `.deb` file for your architecture. Then run:

```bash
ARCH=$(dpkg --print-architecture)
sudo dpkg -i fontget_*_${ARCH}.deb
sudo apt-get install -f  # Fix dependencies if needed
```

**Fedora/RHEL/CentOS (`.rpm` package)**

Visit the [GitHub Releases page](https://github.com/Graphixa/FontGet/releases/latest) and download the `.rpm` file for your architecture. Then run:

```bash
ARCH=$(rpm --eval '%{_arch}')
sudo rpm -i fontget_*_${ARCH}.rpm
```

Both packages install FontGet to `/usr/bin/fontget`. Verify with `fontget version`.

**[AUR (Arch User Repository)](https://aur.archlinux.org/packages/fontget)** (Arch Linux and Arch-based distros)

```bash
yay -S fontget
# or
paru -S fontget
```

The AUR package builds FontGet from source. You need an [AUR helper](https://wiki.archlinux.org/title/AUR_helpers) such as `yay` or `paru`, or you can build manually from the [AUR package page](https://aur.archlinux.org/packages/fontget).

## Build and Install from Source

For instructions on building FontGet from source, see the [Build Guide](development/BUILD.md).

### Prerequisites

- [Go](https://go.dev/) 1.26 or later
- Git

### Build Instructions

**macOS/Linux:**

```bash
git clone https://github.com/Graphixa/FontGet.git
cd FontGet
go build -o fontget

# Move the binary to a directory in your PATH
sudo mv fontget /usr/local/bin/
```

**Windows:**

```powershell
git clone https://github.com/Graphixa/FontGet.git
cd FontGet
go build -o fontget.exe
```

Add the directory containing `fontget.exe` to your PATH, or move it to a directory already in your PATH.


## Verification

After installation, verify that FontGet is installed correctly:

```bash
fontget version
```

This should print the FontGet version. If you see an error, ensure that the installation directory is in your PATH.

### Check Installation Location

**macOS/Linux:**

The shell installer typically installs FontGet to `~/.local/bin/fontget`. If this directory is not in your PATH, add it:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

Add this line to your shell profile (e.g., `~/.bashrc`, `~/.zshrc`) to make it permanent.

**Windows:**

The PowerShell installer typically installs FontGet to `%USERPROFILE%\AppData\Local\Programs\FontGet\fontget.exe` and automatically adds it to your user PATH. You may need to restart your terminal for the PATH changes to take effect.

## Updating FontGet

FontGet includes a built-in self-update system. After the initial installation, you can update FontGet using:

```bash
fontget update
```

This will fetch the latest release from [GitHub Releases](https://github.com/Graphixa/FontGet/releases), download it, and replace your current executable.

You can also update to a specific version:

```bash
fontget update --version 2.2.0
```

> [!NOTE]
> The self-update system works regardless of how you originally installed FontGet (shell script, package manager, etc.).

## Uninstalling FontGet

### macOS

If you installed using the shell script, FontGet is typically installed to `~/.local/bin/fontget`. To uninstall:

```bash
rm -f ~/.local/bin/fontget
```

If you installed to a system directory (e.g., `/usr/local/bin`):

```bash
sudo rm -f /usr/local/bin/fontget
```

If you installed via Homebrew:

```bash
brew uninstall fontget
```

### Linux

If you installed using the shell script, FontGet is typically installed to `~/.local/bin/fontget`. To uninstall:

```bash
rm -f ~/.local/bin/fontget
```

If you installed to a system directory (e.g., `/usr/local/bin`):

```bash
sudo rm -f /usr/local/bin/fontget
```

If you installed via a package manager:

```bash
# Debian/Ubuntu (.deb package)
sudo apt remove fontget

# Fedora/RHEL/CentOS (.rpm package)
sudo rpm -e fontget

# AUR (Arch Linux)
yay -R fontget
# or
paru -R fontget
```

### Windows

If you installed using the PowerShell script, FontGet is typically installed to `%USERPROFILE%\AppData\Local\Programs\FontGet`. To uninstall:

```powershell
Remove-Item "$env:USERPROFILE\AppData\Local\Programs\FontGet" -Recurse -Force
```

You may also need to remove it from your PATH manually via System Settings → Environment Variables.

If you installed via a package manager:

```powershell
# WinGet
winget uninstall Graphixa.FontGet

# Scoop
scoop uninstall fontget

# Chocolatey
choco uninstall fontget
```

## Compatibility

- **macOS**: Both Apple Silicon (arm64) and Intel (x64) are supported
- **Linux**: Both x64 (amd64) and arm64 are supported
- **Windows**: Both x64 (amd64) and arm64 are supported

The Shell installer can be used on Windows with [Windows Subsystem for Linux](https://docs.microsoft.com/en-us/windows/wsl/about), [MSYS](https://www.msys2.org), or equivalent tools.

## Troubleshooting

### FontGet command not found

If you see `fontget: command not found` after installation:

1. **Check if FontGet is installed:**

   **macOS/Linux:**
   ```bash
   ls ~/.local/bin/fontget
   ```

   **Windows:**
   ```powershell
   Test-Path "$env:USERPROFILE\AppData\Local\Programs\FontGet\fontget.exe"
   ```

2. **Verify PATH includes the installation directory:**

   **macOS/Linux:**
   ```bash
   echo $PATH | grep -q "$HOME/.local/bin" && echo "In PATH" || echo "Not in PATH"
   ```

   **Windows:**
   ```powershell
   $env:Path -split ';' | Select-String "FontGet"
   ```

3. **Add to PATH if missing:**

   **macOS/Linux:** Add to your shell profile:
   ```bash
   echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc  # or ~/.zshrc
   source ~/.bashrc  # or source ~/.zshrc
   ```

   **Windows:** The PowerShell installer should add it automatically. If not, add it manually via System Settings → Environment Variables.

### Installation fails

If the installation script fails:

1. **Check your internet connection**
2. **Verify the script URL is accessible:**
   ```bash
   curl -I https://raw.githubusercontent.com/Graphixa/FontGet/main/scripts/install.sh
   ```
3. **Review the script before running** (security best practice):
   ```bash
   curl -fsSL https://raw.githubusercontent.com/Graphixa/FontGet/main/scripts/install.sh -o install.sh
   less install.sh  # Review the script
   sh install.sh    # Run it after reviewing
   ```

### Permission denied errors

If you see permission errors:

- **macOS/Linux:** Ensure the installation directory is writable, or use `sudo` for system-wide installation
- **Windows:** Run PowerShell as Administrator if installing to a system directory

## Automation / CI

## Install Script (Linux / Mac OS)

**`install.sh`:** with `curl … | sh`, stdin is not a TTY, so **Continue?** is skipped by default. Pass script flags after **`sh -s --`**.

| Flag | Description |
|------|-------------|
| `--dry-run`, `--dryrun` | Show what would happen: displays the download and install details, but does not install anything. Exits after displaying information. |
| `-h`, `--help` | Prints usage information for the installer. |

| Variable | Description |
|----------|-------------|
| `FONTGET_VERSION` | Specifies which FontGet version to install (e.g. `2.2.0`). Defaults to `latest`. The `v` prefix is optional. |
| `FONTGET_INSTALL_DIR` | Custom installation location for the binary. Defaults to `$HOME/.local/bin`. |
| `FONTGET_NONINTERACTIVE=1` | Disables interactive prompts (e.g. confirmation to continue) for scripting or automation purposes. |
| `FONTGET_DRY_RUN=1` | Activates dry-run mode, behaving the same as `--dry-run`. |
| `CI` | If set to any non-empty value, disables confirmation prompts, suitable for CI pipelines. |
| `NO_COLOR=1` | Disables colored output in installer messages. |

[!NOTE] User prompts are only shown when both stdin and stdout are terminals, `CI` is not set, and `FONTGET_NONINTERACTIVE` is not `1`.

**`install.ps1` (Windows):** Only the **`FONTGET_VERSION`** environment variable is supported (default: `latest`). There are no flags. The installer places the binary in **`%USERPROFILE%\AppData\Local\Programs\FontGet`** and automatically adds this directory to your user **PATH** if not already present.

Common environment variables for non-interactive automation:

| Flag | Environment Variable | Description |
|------|---------------------|-------------|
| `--accept-agreements` | `FONTGET_ACCEPT_AGREEMENTS=1` | Automatically accepts the license/terms without prompting the user. |
| `--accept-defaults` | `FONTGET_ACCEPT_DEFAULTS=1` | Skips setup wizard or prompts, using all default options. |

Examples: **[Usage — Automation & CI](usage.md#automation--ci)**.

## Additional Resources

- **[Help](usage.md)**: FontGet command reference and usage examples
- **[Contributing](contributing.md)**: How to contribute to FontGet
- **[GitHub Releases](https://github.com/Graphixa/FontGet/releases)**: Download binaries and packages

