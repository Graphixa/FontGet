# FontGet CLI Tool

A command-line tool for installing fonts from multiple font repositories.

## Project Purpose

The `fontget` CLI tool queries multiple font repositories for fonts on demand, allowing users to easily install fonts on their system. It supports both user-level and system-wide font installation with support for Google Fonts, Nerd Fonts, Font Squirrel, and custom sources.

## Repository Layout

```
fontget/
├── cmd/
│   ├── root.go      # Root command and initialization
│   ├── add.go       # Font installation command
│   ├── remove.go    # Font removal command
│   ├── list.go      # List installed fonts
│   ├── search.go    # Search available fonts
│   ├── info.go      # Font information command
│   ├── sources.go   # Source management command
│   ├── config.go    # Configuration management command
│   └── shared.go    # Shared utilities
├── internal/
│   ├── platform/    # Platform-specific font management
│   ├── config/      # Configuration management
│   ├── sources/     # Source management and translators
│   ├── repo/        # Font repository interaction
│   └── logging/     # Logging functionality
├── sources/         # Font source manifests
│   ├── google-fonts.json
│   ├── nerd-fonts.json
│   └── font-squirrel.json
├── docs/
├── go.mod
├── go.sum
└── README.md
```

## Local Cache Layout

```
{TEMP_DIR}/Fontget/     # Temporary download directory
└── fonts/              # Downloaded font files (cleaned up after installation)

Where {TEMP_DIR} is:
- Windows: %TEMP% or %TMP%
- Linux: /tmp
- macOS: /var/folders/.../T/
```

## Configuration Structure

### Main Config (`~/.fontget/config.yaml`)
```yaml
Configuration:
  DefaultEditor: "notepad.exe"

Logging:
  LogPath: "$home/.fontget/logs/fontget.log"
  MaxSize: "10MB"
  MaxFiles: 5

Sources:
  Google:
    Path: "https://raw.githubusercontent.com/graphixa/FontGet/main/sources/google-fonts.json"
    Prefix: "google"
    Enabled: true
  NerdFonts:
    Path: "https://raw.githubusercontent.com/graphixa/FontGet/main/sources/nerd-fonts.json"
    Prefix: "nerd"
    Enabled: true
  FontSquirrel:
    Path: "https://raw.githubusercontent.com/graphixa/FontGet/main/sources/font-squirrel.json"
    Prefix: "sqrl"
    Enabled: false
```

### Installations (`~/.fontget/installations.json`)
```json
{
  "installations": [
    {
      "font_id": "google.roboto",
      "source": "google",
      "installed": "2024-01-15T10:30:00Z",
      "scope": "user"
    }
  ]
}
```

## Commands

### Core Commands
- `fontget add <font-name>` - Install a font
- `fontget remove <font-name>` - Remove an installed font
- `fontget list` - List installed fonts
- `fontget search <query>` - Search for available fonts
- `fontget info <font-id>` - Show detailed font information

### Source Management
- `fontget sources add` - Add a new font source
- `fontget sources remove <source-name>` - Remove a font source
- `fontget sources list` - List all sources
- `fontget sources <source> --enable/--disable` - Enable/disable source
- `fontget sources -oU/-oD <source>` - Reorder source priority
- `fontget sources template --output <file>` - Generate manifest template

### Configuration
- `fontget config` - Open configuration in editor
- `fontget config --show` - Show configuration file location

## Multi-Source Implementation Plan

### Phase 1: Foundation (Week 1-2)
- [ ] **Backward Compatibility Strategy**
  - [ ] Replace existing `sources/google-fonts.json` with new `sources/googleapi-manifest.json`
  - [ ] Update `GetManifest()` function to load from new manifest structure
  - [ ] Ensure all current commands (add, remove, list, search, info) work with new manifest
  - [ ] Test that Google Fonts functionality continues working with new API-based manifest
  - [ ] All enabled sources will be searched simultaneously by default

- [ ] **Font Format Support**
  - [ ] Support TTF, OTF, FON formats (no WOFF/WOFF2 for local installation)
  - [ ] Validate font files during download and installation
  - [ ] Extract font metadata for installation tracking
  - [ ] Handle font file integrity verification

- [ ] **YAML Configuration System**
  - [ ] Create YAML configuration parser
  - [ ] Implement configuration validation
  - [ ] Add configuration file management
  - [ ] Create default configuration template

- [ ] **Built-in Source Translators**
  - [ ] Google Fonts API translator (https://www.googleapis.com/webfonts/v1/webfonts)
  - [ ] Nerd Fonts GitHub API translator (https://api.github.com/repos/ryanoasis/nerd-fonts)
  - [ ] Font Squirrel API translator (https://www.fontsquirrel.com/api/fontlist/all)
  - [ ] Source manifest generation system
  - [ ] Github actions to translate api data from sources into fontget manifests hosted in github repo
  - [ ] API key management for Google Fonts (optional, fallback to public data) (Discuss with user beforehand)
  - [ ] Rate limiting and error handling for all APIs

- [ ] **Local Directory Scanning**
  - [ ] Implement recursive font scanning
  - [ ] Support TTF, OTF, FON formats (no WOFF/WOFF2)
  - [ ] Generate manifests from local directories
  - [ ] Validate font files during scanning

- [ ] **Source Validation System**
  - [ ] JSON syntax validation
  - [ ] Required field validation
  - [ ] Font ID format validation
  - [ ] File path validation (for local sources)
  - [ ] License and category validation

- [ ] **Basic Source Management Commands**
  - [ ] `fontget sources add` (interactive)
  - [ ] `fontget sources add --name --path` (local directory)
  - [ ] `fontget sources add --manifest` (custom manifest with validation)
  - [ ] `fontget sources remove`
  - [ ] `fontget sources list`
  - [ ] `fontget sources <source> --enable/--disable`
  - [ ] `fontget sources -oU <source>` (move source up in priority order)
  - [ ] `fontget sources -oD <source>` (move source down in priority order)
  - [ ] Validate manifests on add (JSON syntax, required fields, font ID format)

### Phase 2: Integration (Week 3-4)
- [ ] **Simple Installation Tracking**
  - [ ] Track current installations only (no removal history)
  - [ ] Store in `~/.fontget/installations.json`
  - [ ] Simple format: font_id, source, installed_date, scope
  - [ ] Support for backup/restore operations (export/import current list of fonts within sources only - Not system fonts or font's installed outside of fontget)
  - [ ] No complex audit trails or removal tracking

- [ ] **Source Collision Resolution**
  - [ ] Detect font name conflicts across sources
  - [ ] User-friendly collision UI
  - [ ] Clear source identification in results
  - [ ] Explicit source specification support

- [ ] **Order-based Priority System**
  - [ ] YAML order determines source priority
  - [ ] First enabled source wins by default
  - [ ] Support for explicit source specification
  - [ ] Priority reordering functionality

- [ ] **Update Existing Commands for Multi-Source**
  - [ ] `fontget add` - support source priority and explicit sources
  - [ ] `fontget search` - search all enabled sources simultaneously by default, filter by source
  - [ ] `fontget list` - support source filtering and grouping
  - [ ] `fontget info` - support source-specific font IDs

- [ ] **Custom Manifest Template Generation**
  - [ ] `fontget sources template --output <file>`
  - [ ] Generate complete manifest template
  - [ ] Include examples and documentation
  - [ ] Validate generated templates

### Phase 3: Polish (Week 5-6)
- [ ] **BIOS-Style Interactive Menu**
  - [ ] Interactive source management UI
  - [ ] Visual priority reordering
  - [ ] Enable/disable toggles
  - [ ] Real-time configuration preview

- [ ] **Custom Manifest Upload Support**
  - [ ] Support for user-provided manifests
  - [ ] Manifest validation and error reporting
  - [ ] Template-based manifest creation
  - [ ] Local manifest file support

- [ ] **Advanced Configuration Options**
  - [ ] Source-specific settings
  - [ ] Advanced logging configuration
  - [ ] Performance optimization options
  - [ ] User preference management

- [ ] **Backup/Restore Functionality**
  - [ ] Export current installation list to JSON/CSV
  - [ ] Import installation list from JSON/CSV
  - [ ] Cross-platform compatibility for installation data
  - [ ] Simple format: font_id, source, scope (no complex metadata)

## Source Types

### Built-in Sources
1. **Google Fonts** - Via Google Web Fonts API (https://www.googleapis.com/webfonts/v1/webfonts)
2. **Nerd Fonts** - Via GitHub API (https://api.github.com/repos/ryanoasis/nerd-fonts)
3. **Font Squirrel** - Via Font Squirrel API (https://www.fontsquirrel.com/api/fontlist/all)

### Custom Sources
1. **Local Directories** - Scan local font folders
2. **Custom Manifests** - User-provided JSON manifests
3. **HTTP Endpoints** - Simple API endpoints (future)

## Font Installation Process

1. **Source Resolution**:
   - Check explicit source specification (e.g., `google.roboto`)
   - Use priority order for source selection
   - Handle collisions with user choice

2. **Font Download and Verification**:
   - Download from appropriate source
   - Verify file integrity
   - Extract font metadata

3. **Installation**:
   - Install to user or machine scope
   - Update font cache
   - Track installation

4. **Cleanup**:
   - Remove temporary files
   - Update installation tracking

## Error Handling

- **Source not found**: Clear error with available sources
- **Font not found**: Show similar fonts from all sources
- **Collision resolution**: Present all options clearly
- **Network errors**: Retry logic with clear error messages
- **Permission errors**: Clear elevation requirements

## Future Enhancements

- **Interactive font selection**: Browse fonts visually
- **Font preview**: Show font samples
- **Batch operations**: Install multiple fonts
- **Font comparison**: Compare fonts across sources
- **Advanced filtering**: Filter by license, category, etc.
- **Performance optimization**: Caching and parallel downloads 