# FontGet Installation Guide

This document provides complete installation instructions for FontGet on all supported platforms. FontGet is a single binary with no external dependencies; binaries are available for both `amd64` and `arm64` architectures.

## Quick Reference

| Method | Platform | Section |
|--------|----------|---------|
| Install script | Mac, Linux, Windows | [Install latest version](#install-latest-version) |
| Package manager | Windows, macOS, Linux | [Install via package manager](#install-via-package-manager) |
| Build from source | All | [Build from source](#build-and-install-from-source) |

---

## Install Latest Version

The recommended way to install FontGet. Downloads the latest release and installs to your user directory.

### Shell (Mac, Linux)

```sh
curl -fsSL https://raw.githubusercontent.com/Graphixa/FontGet/main/scripts/install.sh | sh
```

### PowerShell (Windows)

```powershell
irm https://raw.githubusercontent.com/Graphixa/FontGet/main/scripts/install.ps1 | iex
```

## Install Specific Version

Install a specific version by setting the `FONTGET_VERSION` environment variable.

### Shell (Mac, Linux)

```sh
FONTGET_VERSION=1.0.0 curl -fsSL https://raw.githubusercontent.com/Graphixa/FontGet/main/scripts/install.sh | sh
```

### PowerShell (Windows)

```powershell
$env:FONTGET_VERSION="1.0.0"; irm https://raw.githubusercontent.com/Graphixa/FontGet/main/scripts/install.ps1 | iex
```

## Install via Package Manager

### Windows

**[WinGet](https://winstall.app/)**

```powershell
winget install --id "fontget.fontget"
```

**[Scoop](https://scoop.sh/)**

```powershell
scoop bucket add fontget https://github.com/Graphixa/scoop-bucket
scoop install fontget
```

> [!NOTE]
> Add the bucket first; FontGet is not in the main Scoop repository.

**[Chocolatey](https://chocolatey.org/)** (coming soon)

```powershell
choco install fontget
```

> [!NOTE]
> Chocolatey package is coming soon. Use the [PowerShell install script](#powershell-windows) for now.

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

**AUR (Arch Linux)** (coming soon)

```bash
yay -S fontget
# or
paru -S fontget
```

> [!NOTE]
> AUR package is coming soon. Use the [shell install script](#shell-mac-linux) or download from [GitHub Releases](https://github.com/Graphixa/FontGet/releases).

## Build and Install from Source

For contributors or users who prefer to build from source. See the [Contributing guide](contributing.md) for detailed development setup.

### Prerequisites

- [Go](https://go.dev/) 1.24 or later
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
fontget update --version 1.0.0
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
winget uninstall fontget.fontget

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

## Additional Resources

- **[Help](usage.md)**: FontGet command reference and usage examples
- **[Contributing](contributing.md)**: How to contribute to FontGet
- **[GitHub Releases](https://github.com/Graphixa/FontGet/releases)**: Download binaries and packages

