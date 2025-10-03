# FontGet Commands Reference

This document provides a comprehensive overview of all FontGet commands, their purpose, and usage examples.

---

## Core Commands

### `add`
**Purpose**: Install fonts from various sources (Google Fonts, Font Squirrel, Nerd Fonts)  
**Why it matters**: Core functionality - this is what users use to get fonts  
**Subcommands**: None (takes font name or ID as argument)

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

#### Usage Example
```bash
# Show sources information
fontget sources info

# Update all sources
fontget sources update

# Interactive management
fontget sources manage

# Update with verbose output
fontget sources update --verbose
```

---

### `cache`

#### Purpose
Manage the font cache for performance and storage

#### Why It Matters
Cache can grow large, users need to manage it and troubleshoot issues

#### Subcommands
- `status` - Show cache statistics
- `clear` - Clear all cached data
- `validate` - Check cache integrity

#### Usage Example
```bash
# Show cache information
fontget cache status

# Clear all cached data
fontget cache clear

# Validate cache integrity
fontget cache validate
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

#### Usage Example
```bash
# Show current configuration
fontget config info

# Edit configuration file
fontget config edit

# Validate configuration
fontget config --validate

# Reset to defaults
fontget config --reset-defaults --validate
```

---

## Planned Commands

### `update`

#### Purpose
Update FontGet to the latest version

#### Why It Matters
Users need to get new features and bug fixes

#### Subcommands
None (updates the entire application)

#### Usage Example
```bash
# Update to latest version
fontget update
```

---

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
| `list` | Show installed fonts | `--scope, -s`, `--family, -a`, `--type, -t` |
| `remove` | Uninstall fonts | `--scope, -s`, `--force, -f` |
| `info` | Show font details | `--license, -l`, `--metadata, -m` |
| `sources` | Manage font sources |  |
| &nbsp;&nbsp;&nbsp;- `info` | Show sources information |  |
| &nbsp;&nbsp;&nbsp;- `update` | Update source data | `--verbose, -v` |
| &nbsp;&nbsp;&nbsp;- `manage` | Interactive management |  |
| `cache` | Manage font cache |  |
| &nbsp;&nbsp;&nbsp;- `status` | Show cache statistics |  |
| &nbsp;&nbsp;&nbsp;- `clear` | Clear cached data |  |
| &nbsp;&nbsp;&nbsp;- `validate` | Validate cache integrity |  |
| `config` | Manage configuration | `--validate`, `--reset-defaults` |
| &nbsp;&nbsp;&nbsp;- `info` | Display current config |  |
| &nbsp;&nbsp;&nbsp;- `edit` | Open config file in editor |  |
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

#### Configuration
- `--validate` - Validate the configuration file for errors
- `--reset-defaults` - Reset configuration to default values (use with --validate)

#### Sources Management
- `--verbose, -v` - Show detailed error messages during source updates

---

## Getting Help

- Use `fontget --help` for general help
- Use `fontget <command> --help` for command-specific help
- Use `fontget <command> <subcommand> --help` for subcommand help