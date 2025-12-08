# FontGet Commands Reference

This document provides a comprehensive overview of all FontGet commands, their Purpose, and usage examples.

---

## Core Commands

### `add`

**Purpose:** Install fonts from various sources (Google Fonts, Font Squirrel, Nerd Fonts)

**Subcommands:** None (takes font name or ID as argument)

#### Examples:

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

---

### `search`

#### Purpose
Search for fonts across all sources with fuzzy matching

#### Why It Matters
Users need to find fonts before installing them

#### Subcommands
None (takes search query as argument)

#### Usage Example
```bash
# Find all sans-serif fonts
fontget search "sans"

# Find specific font
fontget search "roboto"

# Find fonts by category
fontget search "mono" --category "Monospace"
```

---

### `list`

#### Purpose
Show installed fonts with details (name, variants, source, etc.)

#### Why It Matters
Users need to see what's installed and manage their font collection

#### Subcommands
None (shows all installed fonts)

#### Usage Example
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

---

### `remove`

#### Purpose
Uninstall fonts from the system

#### Why It Matters
Users need to clean up unused fonts and manage storage

#### Subcommands
None (takes font name as argument)

#### Usage Example
```bash
# Remove font from user scope
fontget remove "roboto"

# Remove font from machine scope
fontget remove "roboto" --scope machine

# Force removal of protected fonts
fontget remove "roboto" --force
```

---

### `info`

#### Purpose
Show detailed information about a specific font

#### Why It Matters
Users need to see font details before installing

#### Subcommands
None (takes font name or ID as argument)

#### Usage Example
```bash
# Show complete font details
fontget info "roboto"

# Show only license information
fontget info "roboto" --license

# Show only metadata
fontget info "roboto" --metadata
```

---

## Management Commands

### `sources`

#### Purpose
Manage font sources (Google Fonts, Font Squirrel, Nerd Fonts)

#### Why It Matters
Users need to control which sources are available and update them

#### Subcommands
- `info` - Show sources information
- `update` - Update source data
- `manage` - Interactive source management
- `validate` - Validate cached sources integrity

#### Usage Example
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

---

### `config`

#### Purpose
Manage FontGet configuration settings

#### Why It Matters
Users need to customize behavior and settings

#### Subcommands
- `info` - Display current config
- `edit` - Open config file in editor
- `validate` - Validate configuration file integrity
- `reset` - Reset configuration to defaults

#### Usage Example
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

---

### `update`

#### Purpose
Update FontGet to the latest version from GitHub Releases

#### Why It Matters
Users need to get new features, bug fixes, and security updates

#### Subcommands
None (updates the entire application)

#### Flags
- `--check, -c` - Only check for updates, don't install
- `-y` - Skip confirmation prompt and auto-confirm update
- `--version <version>` - Update to specific version (e.g., 1.2.3)

#### Usage Example
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

#### Configuration
Update behavior can be configured in `config.yaml`:

```yaml
Update:
  AutoCheck: true        # Check for updates on startup
  AutoUpdate: false      # Automatically install updates (manual by default)
  CheckInterval: 24      # Hours between update checks
  LastChecked: ""         # ISO timestamp (automatically updated)
  UpdateChannel: "stable" # Update channel: stable, beta, or nightly
```

#### Notes
- Updates are downloaded from GitHub Releases
- Checksums are automatically verified for security
- Binary replacement is atomic and safe across all platforms
- Failed updates automatically roll back to the previous version
- Startup update checks are non-blocking and respect the `CheckInterval` setting

---

## Planned Commands

### `export`

#### Purpose
Export font list to various formats (JSON, CSV, etc.)

#### Why It Matters
Users need to backup or share their font collections

#### Subcommands
None (takes format as argument)

#### Usage Example
```bash
# Export to JSON
fontget export --format json

# Export to CSV
fontget export --format csv
```

---

### `import`

#### Purpose
Import fonts from other backup files (JSON, CSV, etc.)

#### Why It Matters
Users migrating from other tools need import functionality

#### Subcommands
None (takes file path as argument)

#### Usage Example
```bash
# Import from JSON file
fontget import --file fonts.json

# Import from CSV file
fontget import --file fonts.csv
```

---

## Quick Reference

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
| `completion` | Generate completion script |  |
| _Global Flags_ | Available on all commands | `--verbose, -v`, `--debug`, `--logs` |

---

## Flag Reference

### Global Flags
- `--verbose, -v` - Show detailed operation information (user-friendly)
- `--debug` - Show debug logs with timestamps (for troubleshooting)
- `--logs` - Open the logs directory in your file manager

### Flag Combinations
- Use `--verbose` alone for user-friendly detailed output
- Use `--debug` alone for developer diagnostic output with timestamps
- Use `--verbose --debug` together for maximum detail (both user info + diagnostics)

### Command-Specific Flags

#### Installation & Management
- `--scope, -s` - Specify installation/removal scope:
  - `user` (default) - Install/remove for current user only
  - `machine` - Install/remove system-wide (requires elevation)
  - `all` - For list/remove: show/remove from both scopes
- `--force, -f` - Force operation even if font already exists or is protected

#### Search & Discovery
- `--category, -c` - Filter fonts by category:
  - `Sans Serif`, `Serif`, `Display`, `Handwriting`, `Monospace`, `Other`

#### Font Information
- `--license, -l` - Show only license information for a font
- `--metadata, -m` - Show only metadata information for a font

#### Font Listing
- `--family, -a` - Filter installed fonts by family name
- `--type, -t` - Filter installed fonts by file type (TTF, OTF, etc.)
- `--full, -f` - Show font styles in hierarchical view

#### Configuration
- No command-specific flags (use subcommands: `validate`, `reset`)

#### Sources Management
- `--verbose, -v` - Show detailed error messages during source updates

---

## Getting Help

- Use `fontget --help` for general help
- Use `fontget <command> --help` for command-specific help
- Use `fontget <command> <subcommand> --help` for subcommand help
