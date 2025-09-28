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
- Powered by [FontGet-Sources](https://github.com/Graphixa/FontGet-Sources) for reliable font data

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

## How It Works

FontGet uses the [FontGet-Sources](https://github.com/Graphixa/FontGet-Sources) repository to provide reliable, up-to-date font data. This centralized approach ensures:

- **Consistent data**: All users get the same font information
- **Regular updates**: Font data is maintained and updated regularly
- **Fast performance**: Sources are cached locally for quick access
- **Reliability**: One centralized sources for consistency

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
For shell completion setup, see the [Shell Completions Guide](docs/shell-completions.md).


## License

MIT License