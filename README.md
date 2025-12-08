# FontGet

[![Version](https://img.shields.io/github/v/release/Graphixa/FontGet)](https://github.com/Graphixa/FontGet/releases)
[![License](https://img.shields.io/github/license/Graphixa/Fontget)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Graphixa/FontGet)](go.mod)
[![CI](https://img.shields.io/github/actions/workflow/status/Graphixa/FontGet/ci.yml?label=CI)](https://github.com/Graphixa/FontGet/actions)

A tiny, cross-platform CLI tool to install and manage fonts from the command line.

FontGet is a fast, font manager that makes it easy to discover, install, and organize fonts from Google Fonts, Nerd Fonts, and custom sources. It's made to be lightweight and to work on almost every system.

## Features

- Install fonts from **Google Fonts** and **Nerd Fonts** out of the box
- Support Font Squirrel source (disabled by default at the moment)
- Custom sources
- Cross-platform (Windows, macOS, Linux)
- Font management (install, remove, list, search)
- Import/export font collections
- Automatic source updates with caching
- Beautiful terminal UI with Catppuccin color scheme
- Built-in self-update system


## Installation

Install FontGet on your system using one of the commands below. FontGet is a single binary executable with no external dependencies. 

> Binaries are available for both `amd64` and `arm64` architectures.

#### Mac, Linux (Shell)

```sh
curl -fsSL https://raw.githubusercontent.com/Graphixa/FontGet/main/scripts/install.sh | sh
```

#### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/Graphixa/FontGet/main/scripts/install.ps1 | iex
```

More installation options can be found in the [Installation Guide](docs/installation.md).

## Installation via Package Manager 

### Windows

**[WinGet](https://winstall.app/)**
```
winget install --id "graphixa.fontget"
```

**[Chocolatey](https://chocolatey.org/)**

```powershell
choco install fontget
```

**[Scoop](https://scoop.sh/)**

```powershell
scoop bucket add fontget https://github.com/Graphixa/scoop-bucket
scoop install fontget
```

### macOS

**[Homebrew](https://formulae.brew.sh/)**

```sh
brew tap Graphixa/homebrew-tap
brew install fontget
```

### Linux

**[AUR (Arch User Repository)](https://aur.archlinux.org/packages/fontget)**

```bash
yay -S fontget
# or
paru -S fontget
```

### Build and Install from Source

Instructions for building FontGet from source can be found in the [Contributing guide](docs/contributing.md).


## Using FontGet

FontGet makes it easy to search, install, and manage fonts from various sources. Here are some common commands to get you started:

**Search for fonts:**

```bash
fontget search "roboto"
```

**Install a font:**

```bash
fontget add "google.roboto"
```

**List installed fonts:**

```bash
# List all installed fonts
fontget list

# List all fonts matching "sans"
fontget list "sans" 
```

**Remove a font:**

```bash
fontget remove "roboto"
```

For a full list of commands refer to the [📖 Help Guide](docs/help.md) or by running `fontget help` in your terminal after installing.

## Additional resources

- **[⬇️ Installation Guide](docs/installation.md)**: Complete installation instructions for all platforms
- **[🛟 Help](docs/help.md)**: Complete command reference and usage examples
- **[🤝 Contributing](docs/contributing.md)**: How to contribute to the project
- **[🔎 FontGet-Sources](https://github.com/Graphixa/FontGet-Sources)**: The font data repository that powers FontGet



## Contributing

To contribute, please read our [contributing instructions](docs/contributing.md).

## License

MIT License
