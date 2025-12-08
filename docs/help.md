# FontGet Commands Reference

This document provides a comprehensive overview of all FontGet commands, their Purpose, and usage examples.

## `add`

### Purpose
Install fonts from various sources (Google Fonts, Font Squirrel, Nerd Fonts)

### Subcommands
None (takes font name or ID as argument)

### Flags
- `--scope, -s` - Specify installation scope:
  - `user` (default) - Install for current user only
  - `machine` - Install system-wide (requires elevation)
- `--force, -f` - Force reinstall even if font already exists

### Examples

```bash
# Install from any available source
fontget add "roboto"

# Install from specific source
fontget add "google.roboto"

# Force reinstall existing font
fontget add "roboto" --force

# Install with verbose output
fontget add "roboto" --verbose

# Install with debug diagnostics
fontget add "roboto" --debug

```

## `search`

### Purpose
Search for fonts across all sources with fuzzy matching

### Subcommands
None (takes search query as argument)

### Flags
- `--category, -c` - Filter fonts by category:
  - `Sans Serif`, `Serif`, `Display`, `Handwriting`, `Monospace`, `Other`

### Usage Example
```bash
# Find all sans-serif fonts
fontget search "sans"

# Find specific font
fontget search "roboto"

# Find fonts by category
fontget search "mono" --category "Monospace"
```

## `list`

### Purpose
Show installed fonts with details (name, variants, source, etc.)

### Subcommands
None (shows all installed fonts)

### Flags
- `--scope, -s` - Filter by installation scope:
  - `user` - Show fonts from user scope only
  - `machine` - Show fonts from machine scope only
  - `all` (default) - Show fonts from both scopes
- `--family, -a` - Filter installed fonts by family name
- `--type, -t` - Filter installed fonts by file type (TTF, OTF, etc.)
- `--full, -f` - Show font styles in hierarchical view

### Usage Example
```bash
# List all installed fonts
fontget list

# List fonts from specific scope
fontget list --scope machine

# List fonts by family
fontget list --family "Roboto"

# List fonts by type
fontget list --type TTF

# Show font styles in hierarchical view
fontget list --full
```

## `remove`

### Purpose
Uninstall fonts from the system

### Subcommands
None (takes font name as argument)

### Flags
- `--scope, -s` - Specify removal scope:
  - `user` (default) - Remove from user scope only
  - `machine` - Remove from machine scope only (requires elevation)
  - `all` - Remove from both scopes
- `--force, -f` - Force removal even if font is protected

### Usage Example
```bash
# Remove font from user scope
fontget remove "roboto"

# Remove font from machine scope
fontget remove "roboto" --scope machine

# Force removal of protected fonts
fontget remove "roboto" --force
```

## `info`

### Purpose
Show detailed information about a specific font

### Subcommands
None (takes font name or ID as argument)

### Flags
- `--license, -l` - Show only license information for a font
- `--metadata, -m` - Show only metadata information for a font

### Usage Example
```bash
# Show complete font details
fontget info "roboto"

# Show only license information
fontget info "roboto" --license

# Show only metadata
fontget info "roboto" --metadata
```

## `sources`

### Purpose
Manage font sources (Google Fonts, Font Squirrel, Nerd Fonts)

### Subcommands
- `info` - Show sources information
- `update` - Update source data
- `manage` - Interactive source management
- `validate` - Validate cached sources integrity

### Flags
- `--verbose, -v` - Show detailed error messages during source updates

### Usage Example
```bash
# Show sources information
fontget sources info

# Update all sources
fontget sources update

# Interactive management
fontget sources manage

# Validate sources integrity
fontget sources validate

# Update with verbose output
fontget sources update --verbose
```

## `config`

### Purpose
Manage FontGet configuration settings

### Subcommands
- `info` - Display current config
- `edit` - Open config file in editor
- `validate` - Validate configuration file integrity
- `reset` - Reset configuration to defaults

### Flags
No command-specific flags (use subcommands: `validate`, `reset`)

### Usage Example
```bash
# Show current configuration
fontget config info

# Edit configuration file
fontget config edit

# Validate configuration
fontget config validate

# Reset to defaults
fontget config reset
```

## `update`

### Purpose
Update FontGet to the latest version from GitHub Releases

### Subcommands
None (updates the entire application)

### Flags
- `--check, -c` - Only check for updates, don't install
- `-y` - Skip confirmation prompt and auto-confirm update
- `--version <version>` - Update to specific version (e.g., 1.2.3)

### Usage Example
```bash
# Check for updates and prompt to install
fontget update

# Only check for updates (don't install)
fontget update --check

# Auto-confirm update (skip confirmation)
fontget update -y

# Update to specific version
fontget update --version 1.2.3

# Check with verbose output
fontget update --check --verbose
```

### Configuration
Update behavior can be configured in `config.yaml`:

```yaml
Update:
  AutoCheck: true        # Check for updates on startup
  AutoUpdate: false      # Automatically install updates (manual by default)
  CheckInterval: 24      # Hours between update checks
  LastChecked: ""         # ISO timestamp (automatically updated)
  UpdateChannel: "stable" # Update channel: stable, beta, or nightly
```

### Notes
- Updates are downloaded from GitHub Releases
- Checksums are automatically verified for security
- Binary replacement is atomic and safe across all platforms
- Failed updates automatically roll back to the previous version
- Startup update checks are non-blocking and respect the `CheckInterval` setting

## `export`

### Purpose
Export installed fonts to a JSON manifest file that can be used to restore fonts on another system

### Subcommands
None (takes output file as optional argument)

### Flags
- `--output, -o` - Output file path (default: fonts-export.json). Can specify directory or full file path
- `--match, -m` - Export fonts that match the specified string
- `--source, -s` - Filter by font source (e.g., "Google Fonts")
- `--all, -a` - Export all installed fonts (including those without Font IDs)
- `--matched` - Export only fonts that match repository entries (default behavior, cannot be used with --all)

### Usage Example
```bash
# Export to default file (fonts-export.json)
fontget export

# Export to specific file
fontget export "fonts.json"

# Export to directory (creates fonts-export.json in that directory)
fontget export -o "D:\Exports"

# Export to specific file path
fontget export -o "D:\Exports\my-fonts.json"

# Export fonts matching a string
fontget export "fonts.json" --match "Roboto"

# Export fonts from specific source
fontget export "google-fonts.json" --source "Google Fonts"

# Export all installed fonts (including unmatched)
fontget export "fonts.json" --all
```

## `import`

### Purpose
Import fonts from a FontGet export manifest file (JSON format)

### Subcommands
None (takes manifest file path as argument)

### Flags
- `--scope, -s` - Installation scope override:
  - `user` - Install for current user only
  - `machine` - Install system-wide (requires elevation)
  - If not specified, auto-detects based on elevation
- `--force, -f` - Force installation even if font is already installed

### Usage Example
```bash
# Import from manifest file
fontget import "fonts.json"

# Import and install to the current user's scope (user only)
fontget import "fonts.json" --scope user

# Import and force install (if fonts are already installed, they're re-installed)
fontget import "fonts.json" --force

# Import and install to the machine scope (all users install)
fontget import "fonts.json" --scope machine
```

## `backup`

### Purpose
Backup installed font files to a zip archive organized by source and family name

### Subcommands
None (takes output path as optional argument)

### Flags
No command-specific flags

### Usage Example
```bash
# Backup to default file (font-backup-YYYY-MM-DD.zip)
fontget backup

# Backup to specific file
fontget backup "fonts-backup.zip"

# Backup to specific directory (creates date-based filename)
fontget backup "D:\Backups"

# Backup to full file path
fontget backup "D:\Backups\my-fonts.zip"
```

### Notes
- Automatically detects accessible scopes (user vs machine) based on elevation
- Fonts are deduplicated across scopes
- System fonts are always excluded
- Prompts before overwriting existing backup files

## `version`

### Purpose
Show FontGet version and build information

### Subcommands
None

### Flags
- `--release-notes` - Show release notes link for the current version

### Usage Example
```bash
# Show version information
fontget version

# Show version with release notes link
fontget version --release-notes

# Show detailed build information (with --debug flag)
fontget version --debug
```

## `completion`

### Purpose
Generate or install shell completion scripts for bash, zsh, and PowerShell

### Subcommands
None (takes shell name as argument)

### Flags
- `--install` - Automatically install completion script to shell configuration file

### Usage Example
```bash
# Generate completion script for bash (output to stdout)
fontget completion bash

# Generate completion script for zsh
fontget completion zsh

# Generate completion script for PowerShell
fontget completion powershell

# Auto-detect shell and install
fontget completion --install

# Install for specific shell
fontget completion bash --install
fontget completion zsh --install
fontget completion powershell --install
```

### Notes
- Supported shells: `bash`, `zsh`, `powershell`
- After installation, restart your terminal or reload your shell configuration
- Auto-detection works on Unix-like systems (Linux, macOS)


# Quick Reference

| Command / Subcommand | Purpose | Flags |
|---------------------|---------|-------|
| `add` | Install fonts | `--scope, -s`, `--force, -f` |
| `search` | Find fonts | `--category, -c` |
| `list` | Show installed fonts | `--scope, -s`, `--family, -a`, `--type, -t`, `--full, -f` |
| `remove` | Uninstall fonts | `--scope, -s`, `--force, -f` |
| `info` | Show font details | `--license, -l`, `--metadata, -m` |
| `sources` | Manage font sources |  |
| &nbsp;&nbsp;&nbsp;- `info` | Show sources information |  |
| &nbsp;&nbsp;&nbsp;- `update` | Update source data | `--verbose, -v` |
| &nbsp;&nbsp;&nbsp;- `manage` | Interactive management |  |
| &nbsp;&nbsp;&nbsp;- `validate` | Validate sources integrity |  |
| `config` | Manage configuration |  |
| &nbsp;&nbsp;&nbsp;- `info` | Display current config |  |
| &nbsp;&nbsp;&nbsp;- `edit` | Open config file in editor |  |
| &nbsp;&nbsp;&nbsp;- `validate` | Validate configuration |  |
| &nbsp;&nbsp;&nbsp;- `reset` | Reset to defaults |  |
| `export` | Export fonts to manifest | `--output, -o`, `--match, -m`, `--source, -s`, `--all, -a`, `--matched` |
| `import` | Import fonts from manifest | `--scope, -s`, `--force, -f` |
| `backup` | Backup font files to zip |  |
| `version` | Show version information | `--release-notes` |
| `completion` | Generate completion script | `--install` |
| `update` | Update FontGet | `--check, -c`, `-y`, `--version` |
| _Global Flags_ | Available on all commands | `--verbose, -v`, `--debug`, `--logs` |

---

## Global Flags

These flags are available on all commands:

- `--verbose, -v` - Show detailed operation information (user-friendly)
- `--debug` - Show debug logs with timestamps (for troubleshooting)
- `--logs` - Open the logs directory in your file manager

### Flag Combinations
- Use `--verbose` alone for user-friendly detailed output
- Use `--debug` alone for developer diagnostic output with timestamps
- Use `--verbose --debug` together for maximum detail (both user info + diagnostics)

---

## Getting Help

- Use `fontget --help` for general help
- Use `fontget <command> --help` for command-specific help
- Use `fontget <command> <subcommand> --help` for subcommand help
