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
│   ├── info.go      # Retreives info about a font from repository without downloading font
│   ├── list.go      # List installed fonts
│   ├── import.go    # Import font files from json file
│   ├── export.go    # Export list of installed font files to json format | Works with --scope parameter
│   └── search.go    # Search available fonts   | Updates every when running first time in 24 hours (keeps manifest fresh)
├── internal/
│   ├── platform/    # Platform-specific font management
│   │   ├── windows.go
│   │   ├── linux.go
│   │   └── darwin.go
│   ├── errors/      # Error handling and user guidance
│   │   └── errors.go
│   └── repo/        # Font repository interaction
│       └── font.go
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

## Platform-Specific Considerations

### Windows
- Use `os.TempDir()` to get `%TEMP%` or `%TMP%`
- Handle long paths if needed
- Ensure proper file permissions

### Linux
- Use `os.TempDir()` to get `/tmp`
- Handle file permissions
- Consider systemd-tmpfiles if available

### macOS
- Use `os.TempDir()` to get `/var/folders/.../T/`
- Handle file permissions
- Consider sandbox restrictions

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

### `fontget completion [bash|zsh|fish|powershell]`

Generate shell completion scripts.

Options:
- `bash`: Generate bash completion script
- `zsh`: Generate zsh completion script
- `fish`: Generate fish completion script
- `powershell`: Generate PowerShell completion script

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
   - [ ] Implement platform-agnostic temp directory handling:
     - [ ] Windows temp directory support
     - [ ] Linux temp directory support
     - [ ] macOS temp directory support
     - [ ] Proper cleanup on all platforms

4. **Core Commands**
   - [x] Implement `add` command
     - [x] Add functionality to check if font installed before downloading or attempting install of font
   - [x] Add scope parameter
   - [ ] Implement `remove` command
   - [x] Implement `list` command
   - [ ] Implement `search` command
   - [ ] Implement `import` command
   - [ ] Implement `export` command
   - [ ] Evaluate commands for required parameters
   - [x] Implement shell completion support:
     - [x] Bash completion
     - [x] Zsh completion
     - [x] Fish completion
     - [x] PowerShell completion

5. **Testing and Documentation**
   - [x] Add unit tests for platform-specific implementations
   - [x] Add unit tests for error handling
   - [ ] Add integration tests for font operations
   - [ ] Add tests for command handlers
   - [ ] Create comprehensive README
   - [ ] Add usage examples
   - [ ] Add platform-specific documentation

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

## Bugfixes

### Font Installation Pre-checks
- [ ] Improve font existence check before download:
  - Current issue: Font download starts before checking if font files are already installed
  - Problem: Case sensitivity mismatch between repository query ("firasans") and installed files ("FiraSans-Black.ttf")
  - Solution:
    1. Normalize font names for comparison (case-insensitive)
    2. Query repository for all font files first
    3. Check each font file against installed fonts before downloading
    4. Add detailed logging of font file matches
    5. Consider implementing a font name mapping system

### Force Flag Implementation
- [ ] Fix --force flag behavior:
  - Current issue: Force flag not working as expected for machine scope installation
  - Problem: Force flag only skips interactive prompts but doesn't override existing font checks
  - Solution:
    1. Modify force flag to override all existence checks
    2. Add force flag handling in platform-specific font managers
    3. Update documentation to clarify force flag behavior
    4. Add warning messages when force installing over existing fonts

### Font Removal Enhancement
- [ ] Improve font removal functionality:
  - Current issue: Font removal fails to find installed fonts
  - Problem: Mismatch between repository font names and installed file names
  - Solution:
    1. Implement font family detection:
       - Query repository for all font files in a family
       - Match installed files against family members
       - Handle partial family removal
    2. Add font name normalization:
       - Strip file extensions for comparison
       - Handle case sensitivity
       - Consider font family prefixes/suffixes
    3. Add verbose mode for debugging font matches
    4. Consider implementing a font registry to track installed fonts

### General Improvements
- [ ] Add better error messages:
  - Include font family information
  - Show available font variants
  - Provide suggestions for similar font names
- [ ] Implement font family awareness:
  - Track font families during installation
  - Support family-wide operations
  - Add family-specific commands
- [ ] Add font metadata caching:
  - Cache repository queries
  - Store font family relationships
  - Improve performance for repeated operations

## Google Fonts Integration Improvements

### 1. Font Repository Structure
- [ ] Implement proper repository structure:
  ```
  .fontget/
  ├── sources/
  │   ├── google-fonts.json     # Main font manifest
  │   ├── metadata/            # Cached METADATA.pb files
  │   │   ├── ofl/            # Open Font License fonts
  │   │   ├── apache/         # Apache License fonts
  │   │   └── ufl/            # Ubuntu Font License fonts
  │   └── licenses/           # Cached license files
  │       ├── OFL.txt         # Open Font License
  │       ├── Apache.txt      # Apache License
  │       └── UFL.txt         # Ubuntu Font License
  └── cache/                  # Temporary download cache
  ```

### 2. Font Metadata Handling
- [ ] Add Protocol Buffer support:
  - [ ] Add protobuf dependency
  - [ ] Define font metadata schema
  - [ ] Implement METADATA.pb parser
- [ ] Enhance FontInfo structure:
  ```go
  type FontInfo struct {
      Name        string       `json:"name"`
      ID          string       `json:"id"`
      Source      string       `json:"source"`
      License     FontLicense  `json:"license"`      // New field for license type
      Category    FontCategory `json:"category"`
      Variants    []string     `json:"variants"`
      Subsets     []string     `json:"subsets"`
      Version     string       `json:"version"`
      Description string       `json:"description"`
      LastModified time.Time   `json:"last_modified"`
  }

  type FontLicense struct {
      Type        string    `json:"type"`         // "OFL", "Apache", "UFL"
      Version     string    `json:"version"`      // License version
      URL         string    `json:"url"`          // License URL
      Description string    `json:"description"`  // Brief description of license terms
  }
  ```

### 3. License Management
- [ ] Implement license handling:
  - [ ] Cache license files locally
  - [ ] Verify license compliance during installation
  - [ ] Display license information in search results
  - [ ] Add license filtering to search
- [ ] Add license-specific features:
  - [ ] License summary in font info
  - [ ] License requirements display
  - [ ] License compliance checks
  - [ ] License update notifications

### 4. Search Improvements
- [ ] Implement advanced search:
  - [ ] Category-based filtering
  - [ ] License-based filtering
  - [ ] Variant-based filtering
  - [ ] Subset-based filtering
- [ ] Add search result sorting:
  - [ ] By popularity
  - [ ] By last modified
  - [ ] By name
  - [ ] By category

### 5. Font Installation
- [ ] Enhance font installation:
  - [ ] Verify license compliance
  - [ ] Support variable fonts
  - [ ] Handle font subsets
  - [ ] Validate font files
- [ ] Add installation options:
  - [ ] Select specific variants
  - [ ] Select specific subsets
  - [ ] Force reinstall
  - [ ] Skip existing

### 6. Platform-Specific Improvements
- [ ] Windows:
  - [ ] Use registry for font tracking
  - [ ] Handle font cache updates
  - [ ] Support font embedding
- [ ] Linux:
  - [ ] Use fontconfig for font tracking
  - [ ] Handle font cache updates
  - [ ] Support system-wide installation
- [ ] macOS:
  - [ ] Use CoreText for font tracking
  - [ ] Handle font cache updates
  - [ ] Support system-wide installation

### 7. Performance Optimizations
- [ ] Implement efficient caching:
  - [ ] Cache font metadata
  - [ ] Cache font files
  - [ ] Cache search results
- [ ] Add parallel processing:
  - [ ] Parallel font downloads
  - [ ] Parallel metadata parsing
  - [ ] Parallel font installation

### 8. Error Handling
- [ ] Improve error messages:
  - [ ] Font-specific errors
  - [ ] Installation errors
  - [ ] Network errors
  - [ ] Permission errors
- [ ] Add recovery mechanisms:
  - [ ] Automatic retry
  - [ ] Fallback options
  - [ ] Cleanup on failure

### 9. Testing
- [ ] Add comprehensive tests:
  - [ ] Unit tests for metadata parsing
  - [ ] Integration tests for font installation
  - [ ] Platform-specific tests
  - [ ] Performance tests
- [ ] Add CI/CD:
  - [ ] Automated testing
  - [ ] Cross-platform builds
  - [ ] Release automation

### 10. Documentation
- [ ] Improve documentation:
  - [ ] Command-line usage
  - [ ] API documentation
  - [ ] Platform-specific guides
  - [ ] Troubleshooting guide
- [ ] Add examples:
  - [ ] Basic usage
  - [ ] Advanced features
  - [ ] Common scenarios
  - [ ] Best practices

## Implementation Priority

1. **Phase 1: Core Improvements**
   - [ ] Implement proper repository structure
   - [ ] Add Protocol Buffer support
   - [ ] Enhance FontInfo structure
   - [ ] Implement license management
   - [ ] Improve search functionality

2. **Phase 2: Platform Support**
   - [ ] Implement platform-specific font tracking
   - [ ] Add platform-specific installation
   - [ ] Handle font cache updates

3. **Phase 3: User Experience**
   - [ ] Add progress indicators
   - [ ] Improve output formatting
   - [ ] Add interactive features

4. **Phase 4: Testing & Documentation**
   - [ ] Add comprehensive tests
   - [ ] Improve documentation
   - [ ] Add examples
