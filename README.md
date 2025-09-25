# FontGet
A tiny, cross-platform font package manager written in Go.

> [!WARNING]  
> This project is currently under heavy development so expect bugs.

## Features

- Install fonts from Google Fonts, Nerd Fonts out of the box
- You can use your own custom sources
- Cross-platform support (Windows, macOS, Linux)
- Font management (install, remove, list, search)
- Import/export font collections (planned)
- Sources are cached and updated regularly for fast performance
- Beautiful terminal UI with Catppuccin color scheme

## Testing

Feel free to test FontGet on your platform by building from source:

### Windows

```powershell
git clone https://github.com/graphixa/fontget.git
cd fontget
go build -o fontget.exe
.\fontget.exe search "roboto"
```

### macOS/Linux:
```bash
# Clone the repository
git clone https://github.com/graphixa/fontget.git
cd fontget
go build -o fontget
./fontget search "roboto"
```

## Usage

```bash
# Search for fonts
fontget search "noto"

# Install a specific font
fontget add google.roboto

# Handle multiple font matches
fontget add roboto  # Shows selection if multiple sources

# List installed fonts
fontget list

# Remove a font
fontget remove "roboto"

# Manage repositories
fontget sources info
fontget sources update
fontget sources manage # TUI based source manager
```

## Shell Completions
> [!WARNING]  
> Shell completion is still a work in progress but is planned.
> 
For shell completion setup, see our [Shell Completions Guide](docs/shell-completions.md).


## License

MIT License