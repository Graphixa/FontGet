# FontGet Commands Reference

This document provides a comprehensive overview of all FontGet commands, their Purpose, and usage examples.

## Commands

| Command | Purpose | Example |
|--------|---------|---------|
| [Help](#help) | Show all FontGet commands | `fontget help` |
| [Search](#search) | Search for fonts across sources | `fontget search "roboto"` |
| [Add](#add) | Install a font from available sources | `fontget add "google.roboto"` |
| [Remove](#remove) | Uninstall fonts from the system | `fontget remove "google.roboto"` |
| [List](#list) | List installed fonts on the system | `fontget list` |
| [Info](#info) | Show detailed info about a font | `fontget info "google.roboto"` |
| [Sources](#sources) | Manage font sources | `fontget sources` |
| [Config](#config) | Manage configuration | `fontget config` |
| [Export](#export) | Export fonts to a manifest | `fontget export --output fonts.json` |
| [Import](#import) | Import fonts from a manifest | `fontget import fonts.json` |
| [Backup](#backup) | Backup installed fonts files to a zip | `fontget backup --scope user` |
| [Version](#version) | Show version/build info | `fontget version` |
| [Update](#update) | Update FontGet | `fontget update` |
| [Completion](#completion) | Generate shell completions | `fontget completion` |
---

## Help

### Purpose
Show all FontGet commands and available options.

### Examples
```bash
fontget help
fontget help add
fontget help sources update
```

## Search

### Purpose
Search for available fonts.

### Flags
- `--category, -c` - Filter by category (e.g., Sans Serif, Serif, Monospace). Use `-c` alone to list categories.
- `--source, -s` - Filter by source (short ID like "google", "nerd", "squirrel" or full name like "Google Fonts")

### Examples
```bash
# Find all fonts containing the word 'sans'
fontget search "sans"

# Find all fonts named 'roboto'
fontget search "roboto"

# Find fonts by category
fontget search "robo" --category "Monospace"

# Find fonts from a specific source
fontget search "roboto" --source "google"
fontget search "roboto" --source "Google Fonts"

# List possible search categories
fontget search -c
```

## Add

**Alias:** `install`

### Purpose
Install fonts from configured sources. You can install one or multiple fonts in a single command.

### Flags
- `--scope, -s` - Installation scope: `user` (default) or `machine` (admin required)
- `--force, -f` - Overwrite existing fonts

### Notes
- Fonts can be specified by name (e.g., "Roboto") or Font ID (e.g., "google.roboto").
- Names with spaces must be quoted: "Open Sans".
- You can pass multiple font names or IDs in one command.

### Examples
```bash
# Install from any available source
fontget add "roboto"

# Install from specific source
fontget add "google.roboto"

# Install multiple fonts at once
fontget add "google.roboto" "google.open-sans" "nerd.jetbrains-mono"

# Force reinstall existing font
fontget add "roboto" --force
```

## Remove

**Alias:** `uninstall`

### Purpose
Remove fonts from your system.

### Flags
- `--scope, -s` - Removal scope (`user` default, `machine`, or `all`; system-wide requires admin)
- `--force, -f` - Force removal even if protected

### Notes
- Fonts can be specified by name (e.g., "Roboto") or Font ID (e.g., "google.roboto").
- Names with spaces must be quoted: "Open Sans".
- You can remove multiple fonts in one command.

### Examples
```bash
# Remove font from user scope
fontget remove "roboto"

# Remove multiple fonts at once
fontget remove "google.roboto" "google.open-sans" "nerd.jetbrains-mono"

# Remove font from machine scope
fontget remove "roboto" --scope machine

# Force removal of protected fonts
fontget remove "roboto" --force
```

## List

### Purpose
List installed fonts.

### Flags
- `--scope, -s` - Filter by installation scope (`user`, `machine`, or `all` default)
- `--type, -t` - Filter by font type (e.g., TTF, OTF)
- `--expand, -x` - Show font styles in hierarchical view

### Notes
- Pass an optional query as a positional argument to filter by family name or Font ID (e.g., `fontget list "roboto"` or `fontget list "google.roboto"`).

### Examples
```bash
# List all installed fonts
fontget list

# List fonts from specific scope
fontget list --scope machine

# List fonts matching a query (family name or Font ID)
fontget list "Roboto"

# List fonts by type
fontget list --type TTF

# Show font styles in hierarchical view
fontget list --expand
```

## Info

### Purpose
Display detailed information about a font.

### Flags
- `--license, -l` - Show license information only
- `--metadata, -m` - Show metadata only

### Notes
- Shows font name, ID, source, available variants, license, categories, and tags.

### Examples
```bash
# Show complete font details
fontget info "roboto"

# Show only license information
fontget info "roboto" --license

# Show only metadata
fontget info "roboto" --metadata
```

## Sources

### Purpose
Manage font sources.

### Subcommands
- `info` - Show sources information
- `update` - Refresh source configurations and font database
- `validate` - Validate source files
- `manage` - Interactive source management (TUI)

### Flags
- `--verbose, -v` - Show detailed output during updates

### Notes
- Only add sources from trusted locations.

### Examples
```bash
# Show sources information
fontget sources info

# Update all sources
fontget sources update

# Interactive management
fontget sources manage

# Validate sources integrity
fontget sources validate


```

## Config

### Purpose
Manage FontGet configuration settings

### Subcommands
- `info` - Display current config
- `edit` - Open config file in editor
- `validate` - Validate configuration file integrity
- `reset` - Reset configuration to defaults

### Examples
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

### Notes
- `config validate`: checks config integrity. If it fails, edit with `fontget config edit` or reset with `fontget config reset`.
- `config reset`: replaces the config with defaults while preserving log files.

## Export

### Purpose
Export installed fonts to a JSON manifest file that can be used to restore fonts on another system

### Flags
- `--output, -o` - Output file path (default: fonts-export.json). Can specify directory or full file path
- `--match, -m` - Export only fonts that match the specified string (name or family)
- `--source, -s` - Export only fonts from the given source (e.g., "Google Fonts", "Nerd Fonts")
- `--all, -a` - Export all installed fonts, including those without Font IDs (cannot be used with `--matched`)
- `--matched` - Export only fonts that match repository entries (default; cannot be used with `--all`)
- `--force, -f` - Overwrite existing file without confirmation

### Notes
- You cannot use `--match` and `--source` together in the same command.
- By default, only fonts that have a Font ID in the repository are exported; use `--all` to include everything.

### Examples
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

# Force overwrite existing export file
fontget export "fonts.json" --force
```

## Import

### Purpose
Import fonts from a FontGet export manifest file (JSON format)


### Flags
- `--scope, -s` - Installation scope override:
  - `user` - Install for current user only
  - `machine` - Install system-wide (requires elevation)
  - If not specified, auto-detects based on elevation
- `--force, -f` - Force installation even if font is already installed

### Examples
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

## Backup

### Purpose
Backup installed font files to a zip archive organized by source and family name

### Flags
- `--force, -f` - Force overwrite existing archive without confirmation
- `--scope, -s` - Scope to backup: `user`, `machine`, or `both` (default: all accessible)

### Notes
- Automatically detects accessible scopes (user vs machine) based on elevation
- Fonts are deduplicated across scopes
- System fonts are always excluded
- Prompts before overwriting existing backup files (unless `--force` is used)

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

# Backup only user scope fonts
fontget backup "fonts-backup.zip" --scope user

# Backup only machine scope fonts
fontget backup "fonts-backup.zip" --scope machine

# Backup both scopes explicitly
fontget backup "fonts-backup.zip" --scope both

# Force overwrite existing backup without confirmation
fontget backup "fonts-backup.zip" --force
```


## Version

### Purpose
Display version and build information.


### Flags
- `--release-notes` - Show release notes link for the current version

### Examples
```bash
fontget version

# Show version with release notes link
fontget version --release-notes

# Show detailed build information (with --debug flag)
fontget version --debug
```

## Update

### Purpose
Update FontGet to the latest version

### Flags
- `--check, -c` - Only check for updates, don't install
- `--yes, -y` - Skip confirmation prompt and auto-confirm update
- `--version <version>` - Update or downgrade to specific version (e.g., 1.2.1)

### Notes
- Updates are downloaded from GitHub Releases
- Checksums are automatically verified for security
- Binary replacement is atomic and safe across all platforms
- Failed updates automatically roll back to the previous version
- Startup update checks are non-blocking and respect the `CheckInterval` setting

### Examples
```bash
# Check for updates and prompt to install
fontget update

# Only check for updates (don't install)
fontget update --check

# Auto-confirm update (skip confirmation)
fontget update --yes

# Update to specific version
fontget update --version 1.2.3

```

### Configuration
Update behavior can be configured in `config.yaml`:

```yaml
Update:
  AutoCheck: true        # Check for updates on startup
  AutoUpdate: false      # Automatically install updates (manual by default)
  CheckInterval: 24      # Hours between update checks
  LastChecked: ""         # ISO timestamp (automatically updated)
```

## Completion

### Purpose
Generate shell completion scripts.

### Flags
- `--install` - Automatically install the completion script to the shell configuration

### Notes
- Supports bash, zsh, and PowerShell. See documentation for installation steps.
- After installation, restart your terminal or reload your shell configuration.

### Examples
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


# Full Command Reference

| Command / Subcommand | Purpose | Flags |
|---------------------|---------|-------|
| `add` | Install fonts | `--scope, -s`, `--force, -f` |
| `search` | Find fonts | `--category, -c`, `--source, -s` |
| `list` | Show installed fonts | `--scope, -s`, `--type, -t`, `--expand, -x`, `[query]` |
| `remove` | Uninstall fonts | `--scope, -s`, `--force, -f` |
| `info` | Show font details | `--license, -l`, `--metadata, -m` |
| `sources` | Manage font sources |  |
| &nbsp;&nbsp;&nbsp;- `info` | Show sources information |  |
| &nbsp;&nbsp;&nbsp;- `update` | Refresh source configurations and font database | `--verbose, -v` |
| &nbsp;&nbsp;&nbsp;- `manage` | Interactive management |  |
| &nbsp;&nbsp;&nbsp;- `validate` | Validate sources integrity |  |
| `config` | Manage configuration |  |
| &nbsp;&nbsp;&nbsp;- `info` | Display current config |  |
| &nbsp;&nbsp;&nbsp;- `edit` | Open config file in editor |  |
| &nbsp;&nbsp;&nbsp;- `validate` | Validate configuration |  |
| &nbsp;&nbsp;&nbsp;- `reset` | Reset to defaults |  |
| `export` | Export fonts to manifest | `--output, -o`, `--match, -m`, `--source, -s`, `--all, -a`, `--force, -f`, `--matched` |
| `backup` | Backup font files to zip | `--force, -f`, `--scope, -s` |
| `import` | Import fonts from manifest | `--scope, -s`, `--force, -f` |
| `version` | Display version and build information | `--release-notes` |
| `completion` | Generate completion script | `--install` |
| `update` | Update FontGet | `--check, -c`, `--yes, -y`, `--version` |
| _Global Flags_ | Available on all commands | `--verbose, -v`, `--debug`, `--logs`, `--wizard` |

---

## Global Flags

These flags are available on all commands:

- `--verbose, -v` - Show detailed operation information (user-friendly)
- `--debug` - Show debug logs with timestamps (for troubleshooting)
- `--logs` - Open the logs directory in your file manager
- `--wizard` - Run the setup wizard to configure FontGet

### Flag Combinations
- Use `--verbose` alone for user-friendly detailed output
- Use `--debug` alone for developer diagnostic output with timestamps
- Use `--verbose --debug` together for maximum detail (both user info + diagnostics)

---

## Getting Help

- Use `fontget --help` for general help
- Use `fontget <command> --help` for command-specific help
- Use `fontget <command> <subcommand> --help` for subcommand help
