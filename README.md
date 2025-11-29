# FontGet

A tiny, cross-platform CLI tool to install and manage fonts from the command line.

FontGet is a fast, font manager that makes it easy to discover, install, and organize fonts from Google Fonts, Nerd Fonts, and custom sources. It's made to be lightweight and to work on almost every system.

> [!NOTE]
> Binaries are available for both `amd64` and `arm64` architectures.

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

>A comprehensive list of installation options can be found in the [Installation Guide](docs/installation.md).

#### Mac, Linux (Shell)

```sh
curl -fsSL https://raw.githubusercontent.com/Graphixa/FontGet/main/scripts/install.sh | sh
```

#### PowerShell (Windows)

```powershell
irm https://raw.githubusercontent.com/Graphixa/FontGet/main/scripts/install.ps1 | iex
```

## Installation via Package Manager 

### Windows

**[WinGet](https://winstall.app/)**
```

> [!NOTE]
> Winget package is coming soon. For now, use the PowerShell install script.

**[Chocolatey](https://chocolatey.org/)**

```powershell
choco install fontget
```

> [!NOTE]
> Chocolatey package is coming soon. For now, use the PowerShell install script.

**[Scoop](https://scoop.sh/)**

```powershell
scoop install fontget
```

> [!NOTE]
> Scoop bucket is coming soon. For now, use the PowerShell install script.

### macOS

**[Homebrew](https://formulae.brew.sh/)**

```sh
brew install fontget
```

> [!NOTE]
> Homebrew installation is coming soon. For now, use the shell install script.

### Linux

**[AUR (Arch User Repository)](https://aur.archlinux.org/packages/fontget)**

```bash
yay -S fontget
# or
paru -S fontget
```

### Build and install from source

Complete instructions for building FontGet from source can be found in the [Contributing guide](docs/contributing.md).

```bash
git clone https://github.com/Graphixa/FontGet.git
cd FontGet
go build -o fontget
```

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
