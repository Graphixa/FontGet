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
│   ├── list.go      # List installed fonts
│   ├── import.go    # Import font files from json file
│   ├── export.go    # Export list of installed font files to json format | Works with --scope parameter
│   └── search.go    # Search available fonts
├── internal/
│   ├── platform/    # Platform-specific font management
│   │   ├── windows.go
│   │   ├── linux.go
│   │   └── darwin.go
│   └── repo/        # Font repository interaction
│       └── font.go
├── go.mod
├── go.sum
└── README.md
```

## Local Cache Layout

```
.fontget/
└── fonts/          # Downloaded font files
```

## Font Installation Process

1. Query GitHub API for a specific font:
   - Endpoint: `https://api.github.com/repos/google/fonts/contents/ofl/{font-name}`
   - Response: JSON array of font files with metadata

2. Download and verify font files:
   - Download each font file
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

4. Platform-specific elevation:
   - Windows: UAC prompt for machine scope
   - Linux: `sudo` for machine scope
   - macOS: `sudo` for machine scope

## Installation Scopes

### User Scope (Default)
- Installs fonts for the current user only
- No elevation required
- Fonts are available only to the installing user
- Default installation locations:
  - Windows: `%LOCALAPPDATA%\Microsoft\Windows\Fonts`
  - Linux: `~/.local/share/fonts`
  - macOS: `~/Library/Fonts`

### Machine Scope
- Installs fonts system-wide
- Requires elevation
- Fonts are available to all users
- Installation locations:
  - Windows: `C:\Windows\Fonts`
  - Linux: `/usr/local/share/fonts`
  - macOS: `/Library/Fonts`

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

### `fontget list [--scope <user|machine>]`

List installed fonts.

### `fontget search <query>`

Search for available fonts.

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
   - [ ] Add elevation support:
     - [ ] Windows UAC prompt
     - [ ] Linux sudo
     - [ ] macOS sudo

4. **Core Commands**
   - [x] Implement `add` command
   - [ ] Add scope parameter
   - [ ] Implement `remove` command
   - [ ] Implement `list` command
   - [ ] Implement `search` command

5. **Testing and Documentation**
   - [ ] Add unit tests
   - [ ] Add integration tests
   - [ ] Create comprehensive README
   - [ ] Add usage examples

## CI Configuration

```yaml
name: CI

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        os: [ubuntu-latest, windows-latest, macos-latest]
    steps:
      - uses: actions/checkout@v2
      - name: Set up Go
        uses: actions/setup-go@v2
        with:
          go-version: 1.21
      - name: Test
        run: go test -v ./...
      - name: Build
        run: go build -v ./...
```

## Testing Requirements

### Windows
- Test user scope installation
- Test machine scope installation with UAC
- Verify font cache updates
- Test font removal

### Linux
- Test user scope installation
- Test machine scope installation with sudo
- Verify font cache updates
- Test font removal

### macOS
- Test user scope installation
- Test machine scope installation with sudo
- Verify font cache updates
- Test font removal
