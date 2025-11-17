# Help Text Analysis and Improvement Recommendations

This document analyzes all command help text in FontGet for clarity, consistency, terminology, and alignment with CLI best practices.

## Analysis Criteria

- **Wording**: Clear, concise, and grammatically correct
- **Terminology**: Consistent use of terms (font ID, font name, source, etc.)
- **Simplicity**: Not overly wordy, easy to scan
- **Helpfulness**: Provides useful information without redundancy
- **CLI Conventions**: Follows patterns from popular CLI tools (git, docker, kubectl, etc.)

---

## Root Command

### `fontget`

**Current:**
- **Use**: `fontget <command> [flags]`
- **Short**: `A command-line tool for managing fonts`
- **Long**: `FontGet is a powerful command-line font manager for installing and managing fonts on your system.`

**Status**: ✅ **Fine** - Clear and concise

**Notes**: Standard root command description. No changes needed.

---

## Core Commands

### `add` / `install`

**Current:**
- **Use**: `add <font-id> [<font-id2> <font-id3> ...]`
- **Short**: `Install fonts on your system`
- **Long**: 
  ```
  Install fonts from available font repositories.

  You can install fonts by their name or ID, e.g. "Roboto" or "google.roboto".
  You can specify multiple fonts by separating them with a space or comma. 

  Font names with spaces in their name should be wrapped in quotes, e.g. "Open Sans".

  You can specify the installation scope using the --scope flag:
    - user (default): Install font for current user
    - machine: Install font system-wide (requires elevation)
  ```
- **Example**: 
  ```
  fontget add "Roboto"
  fontget add "google.roboto"
  fontget add "Open Sans" "Fira Sans" "Noto Sans"
  fontget add "roboto firasans notosans"
  fontget add "Open Sans" -s machine
  fontget add "roboto" -f
  ```

**Status**: ⚠️ **Needs Improvement**

**Issues:**
1. "font repositories" is redundant - just say "available sources" or "font sources"
2. "You can install fonts by their name or ID" - should clarify what "ID" means (Font ID)
3. "You can specify multiple fonts" - redundant, already shown in Use syntax
4. Examples show `-f` flag but it's not explained in Long description
5. Mixing "font-id" in Use with "name or ID" in Long is confusing

**Recommended Changes:**

**Long:**
```
Install fonts from available font sources.

Fonts can be specified by name (e.g., "Roboto") or Font ID (e.g., "google.roboto").
Multiple fonts can be installed in a single command.

Font names with spaces must be wrapped in quotes (e.g., "Open Sans").

Installation scope can be specified with the --scope flag:
  - user (default): Install for current user only
  - machine: Install system-wide (requires administrator privileges)
```

**Notes:**
- Remove redundant explanations about multiple fonts (shown in syntax)
- Clarify "Font ID" terminology
- Add explanation for `-f` flag if it's important, or remove from examples
- Use "administrator privileges" instead of "elevation" (more user-friendly)

---

### `remove` / `uninstall`

**Current:**
- **Use**: `remove <font-id> [<font-id2> <font-id3> ...]`
- **Short**: `Remove fonts from your system`
- **Long**: 
  ```
  Remove fonts from your system. 
  
  You can remove fonts by their name or ID, e.g. "Roboto" or "google.roboto".
  You can remove multiple fonts by separating them with a space or comma. 

  Font names with spaces in their name should be wrapped in quotes, e.g. "Open Sans".

  You can specify the removal scope using the --scope flag:
    - user (default): Remove font from current user
    - machine: Remove font system-wide (requires elevation)
    - all: Remove from both user and machine scopes (requires elevation)
  ```
- **Example**: 
  ```
  fontget remove "Roboto"
  fontget remove "google.roboto"
  fontget remove "Open Sans" "Fira Sans" "Noto Sans"
  fontget remove "roboto, firasans, notosans"
  fontget remove "Open Sans" -s machine
  fontget remove "Roboto" -s user
  ```

**Status**: ⚠️ **Needs Improvement**

**Issues:**
1. Same issues as `add` command
2. "Remove fonts from your system" is redundant (already in Short)
3. "You can remove fonts" - redundant, shown in syntax
4. "all" scope not clearly explained

**Recommended Changes:**

**Long:**
```
Remove fonts from your system.

Fonts can be specified by name (e.g., "Roboto") or Font ID (e.g., "google.roboto").
Multiple fonts can be removed in a single command.

Font names with spaces must be wrapped in quotes (e.g., "Open Sans").

Removal scope can be specified with the --scope flag:
  - user (default): Remove from current user only
  - machine: Remove from system-wide installation (requires administrator privileges)
  - all: Remove from both user and system-wide locations (requires administrator privileges)
```

---

### `search`

**Current:**
- **Use**: `search <query>`
- **Short**: `Search for fonts that are downloadable with FontGet`
- **Long**: `Searches for fonts from Google Fonts and other added sources.`
- **Example**: 
  ```
  fontget search fira
  fontget search "Fira Sans"
  fontget search -c "Sans Serif"
  fontget search "roboto" -c "Sans Serif"
  fontget search -c
  ```

**Status**: ⚠️ **Needs Improvement**

**Issues:**
1. Short description is wordy - "that are downloadable with FontGet" is redundant
2. Long description is too brief and mentions "Google Fonts" specifically (should be generic)
3. Doesn't explain the `-c` flag or category filtering
4. Example shows `-c` without value but doesn't explain what it does

**Recommended Changes:**

**Short:**
```
Search for available fonts
```

**Long:**
```
Search for fonts from all configured sources.

The search query matches font names. Use the --category flag to filter by
font category (e.g., "Sans Serif", "Serif", "Monospace").

Examples:
  fontget search fira              # Search for fonts matching "fira"
  fontget search -c "Sans Serif"   # List all fonts in "Sans Serif" category
  fontget search -c                # List all available categories
```

**Notes:**
- Make Short more concise
- Expand Long to explain category filtering
- Move examples to Long description (more standard for CLI tools)
- Remove redundant "Searches" verb (command name already implies action)

---

### `list`

**Current:**
- **Use**: `list [query]`
- **Short**: `List installed fonts`
- **Long**: 
  ```
  List fonts installed on your system.

  You can filter the results by providing an optional query string to filter font family names.

  By default, fonts from both user and machine scopes are shown.

  You can filter to a specific scope using the --scope flag:
    - user: Show fonts installed for current user only
    - machine: Show fonts installed system-wide only
  ```
- **Example**: 
  ```
  fontget list
  fontget list "jet"
  fontget list roboto -t ttf
  fontget list "fira" -f
  fontget list -s user
  ```

**Status**: ⚠️ **Needs Improvement**

**Issues:**
1. "You can filter the results by providing an optional query string to filter font family names" - redundant ("filter" appears twice)
2. Doesn't explain `-t` (type) or `-f` (full) flags
3. "By default, fonts from both user and machine scopes are shown" - could be clearer

**Recommended Changes:**

**Long:**
```
List fonts installed on your system.

By default, shows fonts from both user and system-wide installations.
Results can be filtered by font family name, type, or scope.

Flags:
  --scope, -s    Filter by installation scope (user, machine)
  --type, -t     Filter by font type (TTF, OTF, etc.)
  --full, -f     Show all font variants in hierarchical view
```

**Notes:**
- Remove redundant wording
- Add flag explanations (standard CLI practice)
- Clarify default behavior more concisely

---

### `info`

**Current:**
- **Use**: `info <font-id> [flags]`
- **Short**: `Display detailed information about a font`
- **Long**: `Show comprehensive information about a font including variants, license, and metadata.`
- **Example**: 
  ```
  fontget info "Noto Sans"
  fontget info "Roboto" -l
  ```

**Status**: ⚠️ **Needs Improvement**

**Issues:**
1. Long description is very brief - doesn't explain what information is shown
2. Doesn't explain the `-l` flag
3. "comprehensive" is vague

**Recommended Changes:**

**Long:**
```
Display detailed information about a font.

Shows font metadata including:
  - Font name, ID, and source
  - Available variants
  - License information
  - Categories and tags

Use the --license flag to show only license information.
```

**Notes:**
- Be specific about what information is shown
- Explain the `-l` flag
- Use bullet points for clarity (common in CLI help)

---

## Configuration Commands

### `config`

**Current:**
- **Use**: `config`
- **Short**: `Manage FontGet settings and configuration`
- **Long**: 
  ```
  Manage FontGet application configuration settings.

  The config command allows you to view and edit the FontGet application configuration file (config.yaml).
  This includes settings for the default editor, logging preferences, and other application behavior.
  ```
- **Example**: 
  ```
  fontget config info              # Show configuration information
  fontget config edit              # Open config.yaml in default editor
  fontget config validate          # Validate configuration file
  ```

**Status**: ✅ **Fine** - Clear and helpful

**Notes**: Good description. Examples with comments are helpful.

---

### `config info`

**Current:**
- **Use**: `info`
- **Short**: `Show configuration information`
- **Long**: `Display detailed information about the current FontGet configuration.`

**Status**: ✅ **Fine** - Clear and concise

---

### `config edit`

**Current:**
- **Use**: `edit`
- **Short**: `Open configuration file in default editor`
- **Long**: `Open the FontGet configuration file (config.yaml) in your default editor.`

**Status**: ✅ **Fine** - Clear and concise

---

### `config validate`

**Current:**
- **Use**: `validate`
- **Short**: `Validate configuration file integrity`
- **Long**: 
  ```
  Check the integrity of the configuration file and report any issues. Useful for troubleshooting configuration problems.

  If validation fails, you can try:
  1. Run 'fontget config edit' to open the configuration file in your editor
  2. Fix any syntax errors or invalid values
  3. Run 'fontget config validate' again to verify the configuration is valid

  For more help, visit: https://github.com/Graphixa/FontGet
  ```

**Status**: ⚠️ **Needs Improvement**

**Issues:**
1. Numbered list in help text is unusual - most CLI tools keep help text simpler
2. GitHub URL in help text is not standard practice (should be in README/docs)
3. "Useful for troubleshooting configuration problems" is redundant

**Recommended Changes:**

**Long:**
```
Validate the configuration file and report any issues.

If validation fails, use 'fontget config edit' to open and fix the configuration file.
If all else fails, use 'fontget config reset' to restore to default settings.
```

**Notes:**
- Simplify the troubleshooting steps
- Remove GitHub URL (not standard in CLI help)
- Keep it concise

---

### `config reset`

**Current:**
- **Use**: `reset`
- **Short**: `Reset configuration to defaults`
- **Long**: 
  ```
  Reset the FontGet configuration file to default values. This is useful when your configuration file becomes corrupted or you want to start fresh.

  This command will:
  1. Generate a new default configuration file
  2. Replace the existing configuration file
  3. Preserve any existing log files

  For more help, visit: https://github.com/Graphixa/FontGet
  ```

**Status**: ⚠️ **Needs Improvement**

**Issues:**
1. Numbered list is verbose for help text
2. GitHub URL should be removed
3. "This is useful when..." is a bit wordy

**Recommended Changes:**

**Long:**
```
Reset the configuration file to default values.

Replaces the existing configuration file with defaults while preserving log files.
Useful when the configuration file is corrupted or you want to start fresh.
```

**Notes:**
- Condense numbered list into prose
- Remove GitHub URL
- Keep explanation concise

---

## Source Management Commands

### `sources`

**Current:**
- **Use**: `sources`
- **Short**: `Manage FontGet font sources`
- **Long**: 
  ```
  Manage sources with the sub-commands. A source provides the data for you to discover and install fonts. Only add a new source if you trust it as a secure location.

  usage: fontget sources [<command>] [<options>]
  ```

**Status**: ⚠️ **Needs Improvement**

**Issues:**
1. "Manage sources with the sub-commands" - awkward phrasing
2. "A source provides the data for you to discover and install fonts" - wordy
3. Security warning is good but could be more concise
4. "usage:" line is redundant (Cobra shows usage automatically)

**Recommended Changes:**

**Long:**
```
Manage font sources.

Font sources provide the data needed to discover and install fonts. Only add
sources from trusted locations.
```

**Notes:**
- Simplify the explanation
- Keep security warning but make it more concise
- Remove redundant "usage:" line

---

### `sources info`

**Current:**
- **Use**: `info`
- **Short**: `Show sources information`
- **Long**: `Display detailed information about the current FontGet sources configuration.`

**Status**: ⚠️ **Minor Improvement**

**Issues:**
1. "Display detailed information about the current FontGet sources configuration" - wordy and redundant

**Recommended Changes:**

**Long:**
```
Display information about configured font sources.
```

**Notes:**
- More concise
- Remove redundant "current" and "FontGet"

---

### `sources update`

**Current:**
- **Use**: `update`
- **Short**: `Update source configuration and refresh cache`
- **Long**: 
  ```
  Update source configuration to use FontGet-Sources URLs and refresh the font data cache.

  usage: fontget sources update [--verbose]
  ```

**Status**: ⚠️ **Needs Improvement**

**Issues:**
1. Mentions "FontGet-Sources URLs" specifically - too implementation-specific for help text
2. "refresh the font data cache" - technical jargon
3. "usage:" line is redundant

**Recommended Changes:**

**Long:**
```
Update source configurations and refresh the font database.

Downloads the latest font data from all enabled sources.
```

**Notes:**
- Remove implementation details
- Use "font database" instead of "cache" (more user-friendly)
- Remove redundant "usage:" line

---

### `sources validate`

**Current:**
- **Use**: `validate`
- **Short**: `Validate cached sources integrity`
- **Long**: 
  ```
  Check the integrity of cached source files and report any issues. Useful for troubleshooting custom sources.

  If validation fails, you can try:
  1. Run 'fontget sources update' to re-download source files and rebuild manifest
  2. Run 'fontget sources validate' again to verify sources have been fixed

  For more help, visit: https://github.com/Graphixa/FontGet
  ```

**Status**: ⚠️ **Needs Improvement**

**Issues:**
1. "cached sources integrity" - technical jargon
2. Numbered list is verbose
3. GitHub URL should be removed
4. "Useful for troubleshooting custom sources" - redundant

**Recommended Changes:**

**Long:**
```
Validate source files and report any issues.

If validation fails, run 'fontget sources update' to refresh the source files.
```

**Notes:**
- Simplify language
- Remove numbered list
- Remove GitHub URL
- Keep it concise

---

### `sources manage`

**Current:**
- **Use**: `manage`
- **Short**: `Interactive source management with TUI`
- **Long**: 
  ```
  Launch an interactive TUI for managing font sources.

  Navigation:
    ↑/↓ or j/k  - Move cursor
    Space/Enter - Toggle source enabled state
    a           - Add new source
    e           - Edit selected source (view built-in details)
    d           - Delete selected source (non-built-in only)
    esc         - Quit (prompts to save if changes made)

  usage: fontget sources manage
  ```

**Status**: ✅ **Fine** - Good TUI documentation

**Notes**: 
- TUI commands need detailed documentation
- Navigation section is helpful
- Remove redundant "usage:" line

**Recommended Changes:**

**Long:**
```
Launch an interactive TUI for managing font sources.

Navigation:
  ↑/↓ or j/k  - Move cursor
  Space/Enter - Toggle source enabled state
  a           - Add new source
  e           - Edit selected source (view built-in details)
  d           - Delete selected source (non-built-in only)
  esc         - Quit (prompts to save if changes made)
```

---

## Utility Commands

### `version`

**Current:**
- **Use**: `version`
- **Short**: `Show FontGet version information`
- **Long**: 
  ```
  Display version information for FontGet including build details.

  This command shows the current version of FontGet, along with build information
  such as git commit hash and build date when available.
  ```
- **Example**: `fontget version`

**Status**: ⚠️ **Minor Improvement**

**Issues:**
1. Long description is redundant - "Display version information" and "shows the current version" say the same thing
2. "when available" is vague

**Recommended Changes:**

**Long:**
```
Display version and build information.
```

**Notes:**
- Much more concise
- Standard for version commands (see git, docker, etc.)

---

### `completion`

**Current:**
- **Use**: `completion`
- **Short**: `Generate completion script`
- **Long**: 
  ```
  To load completions:

  Bash:
    $ source <(go run main.go completion bash)

    # To load completions for each session, execute once:
    # Linux:
    $ go run main.go completion bash > ~/.fontget-completion.bash
    $ source ~/.fontget-completion.bash
    # macOS:
    $ go run main.go completion bash > /usr/local/etc/bash_completion.d/fontget

  Zsh:
    $ source <(go run main.go completion zsh)

    # To load completions for each session, execute once:
    $ go run main.go completion zsh > "${fpath[1]}/_fontget"

  PowerShell:
    PS> go run main.go completion powershell > fontget.ps1
    PS> . ./fontget.ps1
  ```

**Status**: ⚠️ **Needs Improvement**

**Issues:**
1. Examples use `go run main.go` instead of `fontget` - incorrect for installed binary
2. Too verbose for help text - should reference documentation
3. Help text should be brief, detailed instructions belong in docs

**Recommended Changes:**

**Long:**
```
Generate shell completion scripts.

Supports bash, zsh, and PowerShell. See documentation for installation instructions.
```

**Notes:**
- Keep it brief
- Reference documentation for details
- Standard practice for completion commands

---

## Summary

### Commands Needing Improvement:
1. **add** - Clarify terminology, remove redundancy, explain flags
2. **remove** - Same issues as add
3. **search** - Expand Long description, explain flags
4. **list** - Explain flags, remove redundancy
5. **info** - Expand Long description, explain flags
6. **config validate** - Simplify, remove GitHub URL
7. **config reset** - Simplify, remove GitHub URL
8. **sources** - Simplify wording
9. **sources info** - More concise
10. **sources update** - Less technical jargon
11. **sources validate** - Simplify, remove GitHub URL
12. **version** - More concise
13. **completion** - Brief reference to docs

### Commands That Are Fine:
- **root** (fontget)
- **config** (main command)
- **config info**
- **config edit**
- **sources manage**

---

## General Recommendations

1. **Remove GitHub URLs from help text** - These belong in documentation, not CLI help
2. **Avoid numbered lists in Long descriptions** - Keep help text concise
3. **Explain flags in Long descriptions** - Especially for commands with multiple flags
4. **Use consistent terminology** - "Font ID" vs "font ID", "administrator privileges" vs "elevation"
5. **Remove redundant phrases** - "You can...", "This command...", etc.
6. **Keep examples in Examples field** - Don't duplicate in Long description
7. **Use "font database" instead of "cache"** - More user-friendly terminology
8. **Remove "usage:" lines** - Cobra shows usage automatically

