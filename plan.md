# FontGet CLI Tool

A command-line tool for installing fonts from the Google Fonts repository.

## Project Purpose

The `fontget` CLI tool queries the Google Fonts repository for each font on demand, allowing users to easily install fonts on their system. It supports both user-level and system-wide font installation.

## Repository Layout

```
fontget/
├── cmd/
│   ├── root.go      # Root command and initialization
│   ├── add.go       # Font installation command | alias: fontget install
│   ├── remove.go    # Font removal command | alias: fontget uninstall
│   ├── info.go      # Retrieves info about a font from repository without downloading font
│   ├── list.go      # List installed fonts
│   ├── import.go    # Import font files from json file
│   ├── export.go    # Export list of installed font files to json format | Works with --scope parameter
│   └── search.go    # Search available fonts | Updates manifest when running first time in 24 hours
├── internal/
│   ├── platform/    # Platform-specific font management
│   │   ├── windows.go
│   │   ├── linux.go
│   │   └── darwin.go
│   ├── errors/      # Error handling and user guidance
│   │   └── errors.go
│   ├── logging/     # Logging functionality
│   │   ├── logger.go
│   │   └── config.go
│   └── repo/        # Font repository interaction
│       └── font.go
├── docs/
│   └── templates/   # Command templates and documentation
│       └── command_template.go
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

## Font Installation Process

1. Query GitHub API for a specific font:
   - Endpoint: `https://api.github.com/repos/google/fonts/contents/ofl/{font-name}`
   - Response: JSON array of font files with metadata

2. Download and verify font files:
   - Get platform-specific temp directory using `os.TempDir()`
   - Create `Fontget/fonts` subdirectory if it doesn't exist
   - Download to platform temp directory
   - Calculate SHA-256 hash
   - Verify file integrity

3. Install fonts:
   - User scope (default): Install to user's font directory
     - Windows: `%LOCALAPPDATA%\Microsoft\Windows\Fonts`
     - Linux: `~/.local/share/fonts`
     - macOS: `~/Library/Fonts`
   - Machine scope: Install system-wide (requires elevation)
     - Windows: `C:\Windows\Fonts`
     - Linux: `/usr/local/share/fonts`
     - macOS: `/Library/Fonts`

4. Clean up:
   - Remove downloaded files from temp directory
   - Ensure proper cleanup on all platforms
   - Handle cleanup errors gracefully
   - No persistent storage needed

## Error Handling Improvements

### Command Error Handling
- [x] Implement consistent error handling pattern:
  - [x] Validate args in `Args` function
  - [x] Show formatted error messages in red
  - [x] Return `cmd.Help()` for empty queries
  - [x] Double-check args in `RunE` to prevent panics
  - [x] Let other errors flow through cobra's error handling

### Font Installation Errors
- [x] Improve font existence checks:
  - [x] Normalize font names for case-insensitive comparison
  - [x] Query repository for all font files first
  - [x] Check each font file against installed fonts
  - [x] Add detailed logging of font file matches
  - [x] Implement font name mapping system

### Force Flag Behavior
- [x] Fix --force flag implementation:
  - [x] Override all existence checks
  - [x] Add force flag handling in platform-specific managers
  - [x] Update documentation
  - [x] Add warning messages for force installs

## Installation Scopes

### User Scope (Default)
- [x] Installs fonts for the current user only
- [x] No elevation required
- [x] Fonts are available only to the installing user
- [x] Default installation locations:
  - [x] Windows: `%LOCALAPPDATA%\Microsoft\Windows\Fonts`
  - [x] Linux: `~/.local/share/fonts`
  - [x] macOS: `~/Library/Fonts`

### Machine Scope
- [x] Installs fonts system-wide
- [x] Requires elevation
- [x] Fonts are available to all users
- [x] Installation locations:
  - [x] Windows: `C:\Windows\Fonts`
  - [x] Linux: `/usr/local/share/fonts`
  - [x] macOS: `/Library/Fonts`

## Platform-Specific Elevation

### Windows
- Uses UAC (User Account Control) for elevation
- Detection:
  - Check if running as administrator using `windows.IsElevated()`
  - If not elevated and machine scope requested:
    - Print clear message about elevation requirement
    - Optionally attempt to relaunch with elevation using `runas`
- Implementation:
  - Use `windows.IsElevated()` to check current privileges
  - Use `windows.RunAsElevated()` to relaunch if needed
  - Handle UAC prompt gracefully

### Linux
- Uses `sudo` for elevation
- Detection:
  - Check if running as root using `os.Geteuid() == 0`
  - If not root and machine scope requested:
    - Print clear message about `sudo` requirement
    - Provide example command with `sudo`
- Implementation:
  - Use `os.Geteuid()` to check current privileges
  - Provide clear error messages and instructions

### macOS
- Uses `sudo` for elevation
- Detection:
  - Check if running as root using `os.Geteuid() == 0`
  - If not root and machine scope requested:
    - Print clear message about `sudo` requirement
    - Provide example command with `sudo`
- Implementation:
  - Use `os.Geteuid()` to check current privileges
  - Provide clear error messages and instructions

## Commands

### `fontget add <font-name> [--scope <user|machine>]`

Install a font from Google Fonts.

Options:
- `--scope`: Installation scope (default: user)
  - `user`: Install for current user only
  - `machine`: Install system-wide (requires elevation)

### `fontget remove <font-name> [--scope <user|machine>]`

Remove an installed font.

### `fontget list [--scope <user|machine|all>]`

List installed fonts.

Options:
- `-s, --scope`: Installation scope (default: auto-detected)
  - `user`: List fonts for current user only
  - `machine`: List system-wide fonts (requires elevation)
  - `all`: List fonts from both user and machine scopes
- `-a, --family`: Filter by font family name
- `-t, --type`: Filter by font type (TTF, OTF, etc.)

Examples:
```bash
fontget list
fontget list -s machine
fontget list -s all
fontget list -a "Roboto"
fontget list -t TTF
fontget list -s all -t TTF
```

### `fontget search <query>`

Search for available fonts.

### `fontget completion [bash|zsh|fish|powershell]`

Generate shell completion scripts.

Options:
- `bash`: Generate bash completion script
- `zsh`: Generate zsh completion script
- `fish`: Generate fish completion script
- `powershell`: Generate PowerShell completion script

### Font Removal Enhancement
- [x] Improve font removal:
  - [x] Implement font family detection
  - [x] Add font name normalization
  - [x] Add verbose mode for debugging
  - [ ] Consider implementing font registry (pending)

## Milestones

1. **Basic Setup**
   - [x] Initialize Go module
   - [x] Set up Cobra CLI structure
   - [x] Create basic command structure

2. **Font Repository Integration**
   - [x] Implement GitHub API client
   - [x] Add font metadata parsing
   - [x] Implement font download with SHA-256 verification

3. **Platform-Specific Implementation**
   - [x] Create platform-agnostic interface
   - [x] Implement Windows font manager
   - [x] Implement Linux font manager
   - [x] Implement macOS font manager
   - [x] Add elevation support:
     - [x] Windows UAC prompt
     - [x] Linux sudo
     - [x] macOS sudo
   - [x] Implement platform-agnostic temp directory handling:
     - [x] Windows temp directory support
     - [x] Linux temp directory support
     - [x] macOS temp directory support
     - [x] Proper cleanup on all platforms

4. **Core Commands**
   - [x] Implement `add` command
     - [x] Add functionality to check if font installed before downloading
   - [x] Add scope parameter
   - [x] Implement `remove` command
   - [x] Implement `list` command
   - [x] Implement `search` command
   - [ ] Implement `import` command (pending)
   - [ ] Implement `export` command (pending)
   - [x] Implement shell completion support:
     - [x] Bash completion
     - [x] Zsh completion
     - [x] Fish completion
     - [x] PowerShell completion

5. **Testing and Documentation**
   - [x] Add unit tests for platform-specific implementations
   - [x] Add unit tests for error handling
   - [ ] Add integration tests for font operations (pending)
   - [ ] Add tests for command handlers (pending)
   - [ ] Create comprehensive README (pending)
   - [ ] Add usage examples (pending)
   - [ ] Add platform-specific documentation (pending)

## Implementation Priority

1. **Phase 1: Logging System**
   - [x] Create logging package (basic logging, --verbose, debug prints)
   - [x] Implement log file handling (basic)
   - [x] Add --verbose flag to commands
   - [x] Clean up debug prints
   - [ ] Implement log rotation, cleanup, advanced config (pending)

2. **Phase 2: Command Completion**
   - [x] Implement `remove` command
   - [x] Add proper logging to all commands
   - [x] Add verbose mode to all commands
   - [ ] Implement `import` and `export` commands (pending)

3. **Phase 3: Testing & Documentation**
   - [ ] Improve documentation (pending)
   - [ ] Add examples (pending)

4. **Phase 4: User Experience**
   - [ ] Add progress indicators (pending)
   - [ ] Improve output formatting (pending)
   - [ ] Add interactive features (pending)

## Platform-Specific Improvements

### Windows:
  - [x] Use registry for font tracking (basic add/remove)
  - [x] Handle font cache updates
  - [ ] Support font embedding (pending)
### Linux:
  - [x] Use fontconfig for font cache update
  - [x] Handle font cache updates
  - [ ] Support system-wide installation (pending advanced features)
### macOS:
  - [x] Use CoreText/atsutil for font cache update
  - [x] Handle font cache updates
  - [ ] Support system-wide installation (pending advanced features)

## Logging System

### Log File Structure
- [x] Implement structured logging system (basic)
  - [x] Create logging package in `internal/logging`
  - [x] Define log levels (INFO, DEBUG, ERROR)
  - [x] Add timestamps to all log entries
- [ ] Support log rotation, cleanup, and advanced config (pending)

### Log File Location
- [x] Configure default log location (basic)
- [ ] Add configuration options, cleanup command, and audit trail (pending)

### Logging Features
- [x] Add --verbose flag to commands
- [ ] Implement log rotation, cleanup, and advanced config (pending)

### Log Content
- [x] Define standard log format
- [x] Remove debug print statements
- [x] Add context to log messages

## License Management

### First-Time Setup Flow
- [ ] Implement first-time user detection
- [ ] Add license acceptance prompt for Google Fonts
- [ ] Store acceptance in user config
- [ ] Silent operation after acceptance

### Sources Command
- [ ] Implement `fontget sources` command
- [ ] Add `--add` flag for adding new sources
- [ ] Add `--list` flag for viewing accepted sources
- [ ] Add `--reset-licenses` flag for re-accepting licenses
- [ ] Trigger license acceptance when adding new sources

### Multiple Sources Support
- [ ] Support for multiple font sources
- [ ] Individual license acceptance tracking per source
- [ ] Auto-detect new sources in manifest
- [ ] Handle source-specific license requirements

### User Config Storage
```json
{
  "first_run_completed": true,
  "accepted_sources": {
    "google-fonts": {
      "accepted": true,
      "accepted_date": "2024-01-15T10:30:00Z"
    }
  }
}
```

### Implementation Priority

1. **Phase 1: First-Time Setup**
   - [ ] Add first-time user detection
   - [ ] Implement Google Fonts license acceptance prompt
   - [ ] Store acceptance in user config
   - [ ] Silent operation after acceptance

2. **Phase 2: Sources Command**
   - [ ] Create `fontget sources` command
   - [ ] Implement `--add` flag with license acceptance
   - [ ] Add `--list` and `--reset-licenses` flags
   - [ ] Handle new source license acceptance

3. **Phase 3: Multiple Sources**
   - [ ] Support multiple sources in manifest
   - [ ] Individual source acceptance tracking
   - [ ] Auto-detect and prompt for new sources
   - [ ] Source-specific license handling

## Command Improvements

### Add Command ✅
- [x] Add `-s` alias for `--scope`
- [x] Add `--force` flag with `-f` alias
- [x] Improve description and examples
- [x] Add auto-scope detection
- [x] Show installation scope in success messages

### Remove Command ✅
- [x] Add `-s` alias for `--scope`
- [x] Add `--force` flag with `-f` alias
- [x] Improve description and examples
- [x] Add auto-scope detection
- [x] Prevent removal of protected system fonts
- [x] Show removal scope in success messages

### List Command ✅
- [x] Fix `-f` alias conflict (changed to `-a` for `--family`)
- [x] Improve description and examples
- [x] Add auto-scope detection
- [x] Update examples to use short flags consistently

### Search Command
- [ ] Fix alias conflicts
- [ ] Improve description and examples
- [ ] Add auto-scope detection

### Info Command
- [ ] Fix alias conflicts
- [ ] Improve description and examples
- [ ] Add auto-scope detection
