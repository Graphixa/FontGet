# FontGet

A PowerShell module that provides a simple CLI interface for installing and managing Google Fonts on Windows systems. Requires administrator privileges for font installation.

## Installation

Install from PowerShell Gallery:
Install-Module -Name FontGet

## Quick Start

gfont install "font name"                  # Install a font (requires admin)
gfont list                                 # List installed fonts
gfont search                              # Search Google Fonts
gfont uninstall                           # Remove a font (requires admin)

## Commands

| Command | Description |
|---------|-------------|
| install, add | Install Google fonts (requires admin) |
| uninstall, remove | Remove installed fonts (requires admin) |
| list | List installed fonts |
| search | Search available Google fonts |
| help | Show help information |

## Options

- --force: Skip confirmation prompts
- --system: Show system fonts only (with list command)

## Examples

1. Install multiple fonts:
   gfont install "Roboto, Open Sans, Lato"

2. Force installation:
   gfont install "Roboto" --force

3. List fonts:
   gfont list

4. Search fonts:
   gfont search "sans"

## Requirements

- Windows PowerShell 5.1 or PowerShell Core 6.0+
- Windows Operating System
- Administrator privileges

## License

This project is licensed under the MIT License.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Features

- Download fonts from Google Fonts repository
- Install fonts system-wide or per-user
- Check for existing font installations
- Logging functionality
- Support for TTF and OTF font formats

## Installation

1. Clone this repository or download the module files:
