# fontget

A tiny, cross-platform font package manager written in Go.

## Installation

1. Download the latest release for your platform from the [releases page](https://github.com/yourusername/fontget/releases)
2. Extract the binary to a location in your PATH
3. Enable shell completions (optional but recommended):

### Shell Completions

#### PowerShell
```powershell
# Enable completions for current session
fontget completion powershell | Out-String | Invoke-Expression

# To make it permanent, add to your PowerShell profile:
Add-Content $PROFILE "fontget completion powershell | Out-String | Invoke-Expression"
```

#### Bash
```bash
# Enable completions for current session
source <(fontget completion bash)

# To make it permanent, add to your ~/.bashrc:
echo "source <(fontget completion bash)" >> ~/.bashrc
```

#### Zsh
```zsh
# Enable completions for current session
source <(fontget completion zsh)

# To make it permanent, add to your ~/.zshrc:
echo "source <(fontget completion zsh)" >> ~/.zshrc
```

> Note: For a comprehensive guide on setting up completions in different terminal emulators (Windows Terminal, Kitty, etc.), please refer to our [Terminal Setup Guide](docs/terminal-setup.md).

```bash
# Using Go
go install fontget@latest

# Using Homebrew (macOS)
brew install fontget

# Using winget (Windows)
winget install fontget
```

## Usage

```bash
# Install a font
fontget add "open-sans"

# Remove a font
fontget remove "open-sans"

# List installed fonts
fontget list

# Search for fonts
fontget search "noto"

# Import fonts from JSON
fontget import fonts.json

# Export installed fonts
fontget export
fontget export --google  # Export only Google Fonts

# Manage repositories
fontget repo            # List repositories
fontget repo --update   # Update font index
fontget repo --add      # Add a repository
fontget repo --remove   # Remove a repository

# Manage cache
fontget cache prune     # Clear unused downloads
```

## Features

- Install fonts from Google Fonts and other repositories
- Offline installation support
- Cross-platform (Windows, macOS, Linux)
- Font management (install, remove, list)
- Import/export font collections
- Cache management

## Development

```bash
# Clone the repository
git clone https://github.com/yourusername/fontget.git
cd fontget

# Build
go build -o fontget ./cmd/fontget

# Run tests
go test ./...
```

## License

MIT License 