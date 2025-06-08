# fontget

A tiny, cross-platform font package manager written in Go.

## Installation

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