# FontGet Codebase Documentation

This document provides a comprehensive overview of the FontGet codebase, explaining the purpose and functionality of each file and how they interface with other components.

> For codebase rules (e.g. “where should new code go?”), see `docs/development/guidelines/codebase-layout-guidelines.md`.

## Table of Contents

- [Root Files](#root-files)
- [Static analysis and security scanning](#static-analysis-and-security-scanning)
- [Command Files (`cmd/`)](#command-files-cmd)
- [Internal Packages](#internal-packages)
- [Documentation Files](#documentation-files)
- [Runtime data (not in the git tree)](#runtime-data-not-in-the-git-tree)
- [Legacy/Deprecated Files](#legacydeprecated-files)

---

## Root Files

### `main.go`
**Purpose**: Application entry point
**Functionality**: 
- Initializes the CLI application
- Calls `cmd.Execute()` to start the command processing
- Handles fatal errors and exits with appropriate codes

**Interfaces**: 
- Imports `fontget/cmd` package
- Uses standard `os` package for error handling

**Status**: ✅ Active - Core application entry point

### `go.mod`
**Purpose**: Go module definition and dependency management
**Functionality**:
- Defines module name as `fontget`
- Specifies Go version 1.26.0 (toolchain `go1.26.6`)
- Lists all required dependencies including:
  - Cobra (CLI framework)
  - Bubble Tea / Bubbles (TUI framework and widgets)
  - Lipgloss (styling and terminal colors)
  - XZ (archive extraction)
  - Various other utilities (spinners and progress UI live in `internal/ui` and `internal/components`)

**Status**: ✅ Active - Essential for Go module system

---

## Static analysis and security scanning

CI (see [`.github/workflows/ci.yml`](../../.github/workflows/ci.yml)) runs:

- **`staticcheck ./...`** — must be clean; no project-wide suppressions.
- **`gosec -conf=.gosec.json ./...`** — gosec with the repo’s configuration, not default rules only.

The Security workflow ([`.github/workflows/security.yml`](../../.github/workflows/security.yml)) runs:

- **`govulncheck ./...`** — reports only vulnerabilities that are reachable from this module (stdlib and dependencies). It must be clean.

Configuration lives in [`.gosec.json`](../../.gosec.json) at the repository root. It **excludes** several high-noise rules that are weak fits for a local CLI (`G104`, `G304`, `G204`, `G122`, `G703`). **`G404`** (weak RNG) is not globally excluded: intentional `math/rand` use for **non-cryptographic** retry jitter is marked with **`#nosec G404`** at those call sites. The config also sets **permission thresholds** (`G301`/`G302`/`G306`/`G307`) so some directory and file modes used for caches or platform conventions do not fail the build. That is intentional: the gate matches product tradeoffs (shell completion paths, shared dirs, argv-only subprocesses) rather than “strictest possible” default gosec output.

**Stricter local audit:** To see what default rules report (useful before tightening config), run from the repo root without `-conf`:

```bash
go run github.com/securego/gosec/v2/cmd/gosec@latest ./...
```

Treat that output as **informational triage**, not as the same bar as CI.

**Subprocess and archive handling (invariants):** FontGet invokes external programs only with **`exec.Command` (or `CommandContext`) and discrete arguments**—not through a shell—so URLs and paths are passed as argv elements (e.g. download fallbacks in `internal/network/download_fallbacks.go`, platform helpers in `internal/platform/platform.go`). User-supplied strings must not be concatenated into a single shell command line. Archive extraction uses pure-Go ZIP / TAR.XZ / TAR.GZ readers only (no external archiver) and applies **path safety** (e.g. `safeArchiveRelPath`, symlink skipping where applicable in `internal/repo/archive.go`) so extracted files stay under the intended destination tree; keep those checks when changing extraction code.

---

## Command Files (`cmd/`)

### `root.go`
**Purpose**: Root command definition and global configuration
**Functionality**:
- Defines the main `fontget` command
- **Flags**: Persistent flags (apply to all commands) — `--verbose`, `-v`, `--debug`, `--accept-agreements`, `--accept-defaults`. Root-only flags — `--logs`, `--wizard`. CI flags work with any command (e.g. `fontget add <font-id> --accept-agreements --accept-defaults`).
- Initializes logging configuration
- Registers all subcommands
- **Custom Help Templates**: Custom help and usage templates with controlled section ordering (Usage → Description → Commands → Options → Examples)
- **Wizard Flag**: `--wizard` re-runs the onboarding wizard. **CI flags**: `--accept-agreements` and `--accept-defaults` skip first-run terms/onboarding for scripts/CI (env: `FONTGET_ACCEPT_AGREEMENTS`, `FONTGET_ACCEPT_DEFAULTS`; `FONTGET_SKIP_ONBOARDING` still supported).

**Interfaces**:
- Imports all command packages
- Uses `internal/logging` for log configuration
- Uses `internal/output` for verbose/debug functionality
- Uses `internal/onboarding` for wizard execution

**Status**: ✅ Active - Core command orchestration

### `add.go`
**Purpose**: Font installation command
**Functionality**:
- Installs fonts from enabled built-in sources (see `internal/sources` defaults) and custom sources
- Supports both font names (e.g., "Roboto") and Font IDs (e.g., "google.roboto")
- **Local install**: also accepts a font file, folder, or zip (including FontGet backups); see `add_local.go`
- Handles font search and fuzzy matching
- Manages font installation with progress tracking
- Supports different installation scopes (user/system)
- Provides detailed error handling and suggestions
- Uses in-command types and helpers (`FontOperationDetails`, `installFont`, progress integration) together with shared internal packages for consistent behavior
- **Architecture**: Installation orchestration lives in this file and `internal/*` packages; there are no `cmd/operations.go` or `cmd/handlers.go` sources
- **Pre-installation Check**: Checks if fonts are already installed before downloading to save bandwidth and time
- **Installation registry**: After a fully successful install, writes provenance to **`~/.fontget/installation_registry.json`** via **`internal/installations.RecordInstallation`** (grouped families/files, optional **`installation_source`** from cached manifest lookup). Local installs use Font IDs `local.<family>` with source `local` or `fontget-backup`.

**Key Functions**:
- `addCmd.RunE`: Main command execution (classifies local vs catalog args)
- `installFontsInDebugMode`: Debug mode installation (plain text output)
- `installFont`: Core font installation logic (includes pre-download check for already-installed fonts)
- `installDownloadedFonts`: Shared per-file install loop (`deleteSources` protects user paths for local adds)
- `getSourceName`: Source name resolution
- `showFontNotFoundWithSuggestions`: Error handling with suggestions

**Related**: `add_local.go` — `classifyAddArg`, `installLocalPath`, local confirm/`--yes`

**Interfaces**:
- Uses `internal/cmdutils` for CLI helpers (manifest checks, elevation, file existence, argument handling)
- Uses `internal/repo` for font data
- Uses `internal/installations` for **`installation_registry.json`** (record after successful install)
- Uses `internal/platform` for OS-specific operations and local discover/dedupe (`local_candidates.go`)
- Uses `internal/output` for verbose/debug output
- Uses `internal/ui` for user interface styling and spinners
- Uses `internal/components` for progress bar and operation UI
- Uses `internal/shared` for shared utilities (matching, formatting, errors, backup zip comment)

**Status**: ✅ Active - Core functionality

### `search.go`
**Purpose**: Font search command
**Functionality**:
- Searches for fonts across all enabled sources
- Provides fuzzy matching and filtering
- Displays search results in formatted tables
- Supports various search options and filters
- **Source-Only Search**: Supports searching by source alone (e.g., `fontget search -s google`) to show all fonts from a specific source
- **Result Limiting**: Configurable result limit via `config.yaml` Search.ResultLimit (0 = unlimited, default)
- **Source Filtering**: Supports filtering by source ID, full name, or short prefix (e.g. `google`, `nerd`, `league`, `fontshare`, `fontsource`, `squirrel`)
- **Refactored Code**: Extracted constants, helper functions, and removed code duplication for better maintainability

**Key Functions**:
- `searchCmd.RunE`: Main search execution
- `validateSource`: Validates source by ID, name, or prefix
- `logDebugScoreBreakdown`: Debug output for search scoring details
- `displaySearchResults`: Result formatting with dynamic table sizing

**Key Features**:
- **Source Validation**: Checks source ID, full name, and short prefix for flexible filtering
- **Result Limiting**: Applies configurable limit after source filtering (if enabled)
- **Debug Scoring**: Detailed score breakdown in debug mode showing base score, match bonus, popularity bonus, and final score
- **Code Quality**: Extracted constants for magic numbers, shared completion functions, and improved code organization

**Interfaces**:
- Uses `internal/repo` for font data access
- Uses `internal/config` for user preferences and result limiting
- Uses `internal/output` for verbose/debug output

**Status**: ✅ Active - Core functionality

### `list.go`
**Purpose**: Font listing command
**Functionality**:
- Lists installed fonts on the system
- Matches installed fonts to repository entries to show Font IDs, License, Categories, and Source
- Provides filtering and formatting options
- Shows font details and metadata
- Default scope is both user and machine (omit `--scope`); valid values are `user` or `machine` only
- Displays columns: Name, Font ID, License, Categories, Type, Scope, Source
- **Font ID Filtering**: Query parameter can match either font family names (e.g., "Roboto") or Font IDs (e.g., "google.roboto")
- **Performance Optimizations**:
  - Early type filtering (filters by extension before metadata extraction)
  - Concurrent metadata extraction for faster scans on large font directories
  - Precomputed case-insensitive sort keys to avoid repeated `strings.ToLower()` in sort comparators
  - Avoids holding the spinner open for fast operations (no artificial 2.5s delay when spinner clears the line)
  - Disables spinner when `--debug` is enabled to prevent carriage-return UI rendering from mangling debug output

**Key Functions**:
- `listCmd.RunE`: Main listing execution
- `collectFonts`: Collects fonts from specified scopes with optional type filtering and optional verbose output suppression
- `buildParsedFont`: Extracts font metadata from file path and builds ParsedFont struct
- `groupByFamily`: Groups fonts by family name
- `filterFontsByFamilyAndID`: Filters font families by family name or Font ID

**Key Features**:
- **Font ID Support**: Filter by Font ID in addition to family name
- **Early Type Filtering**: Filters by file extension before expensive metadata extraction when type filter is specified
- **Optimized Filtering**: Caches lowercased strings to avoid repeated ToLower() calls
- **Verbose Output Suppression**: `collectFonts` accepts optional `suppressVerbose` parameter to suppress verbose output when called from internal/helper functions (e.g., `checkFontsAlreadyInstalled`, `backup.go`, `export.go`)
- **Standardized Structure**: File follows Go best practices with imports → types → command → helpers structure
- **Installation registry**: Merges **`internal/installations.Load`** into listed families when repository matching left **`FontID`** empty (path index, then family index; **`repo.MatchRepositoryFontByID`**)

**Flags**:
- `--scope, -s`: Filter by installation scope (`user` or `machine`; omit for both)
- `--type, -t`: Filter by font type (TTF, OTF, etc.)
- `--expand, -x`: Show all font variants in hierarchical view
- `--fontget-installed`: Only fonts installed by FontGet (from the installation registry)

**Interfaces**:
- Uses `internal/platform` for OS-specific font detection
- Uses `internal/output` for verbose/debug output
- Uses `internal/repo` for font matching and repository access
- Uses `internal/installations` for **`installation_registry.json`** (merge into list results)
- Uses `internal/shared` for protected font checking and font utilities

**Status**: ✅ Active - Core functionality

### `info.go`
**Purpose**: Font information command
**Functionality**:
- Shows detailed information about specific fonts
- Displays font metadata, variants, and source information
- Provides comprehensive font details

**Flags**:
- `--license, -l`: Show license information

**Key Functions**:
- `infoCmd.RunE`: Main info execution
- `showFontInfo`: Detailed information display

**Interfaces**:
- Uses `internal/repo` for font data
- Uses `internal/output` for verbose/debug output

**Status**: ✅ Active - Core functionality

### `remove.go`
**Purpose**: Font removal command
**Functionality**:
- Removes fonts from the system
- Supports both font names (e.g., "Roboto") and Font IDs (e.g., "google.roboto")
- Handles removal scopes via `--scope` (`user` or `machine`); when omitted, behaves according to elevation auto-detect
- Extracts font names from installed font metadata (SFNT name table)
- **Installation registry**: For Font IDs, prefers registry-resolved basenames in scope when present; drops the registry entry after a full successful registry-backed removal
- Protects critical system fonts from removal
- Provides consistent verbose/debug output matching add command
- Auto-detects scope based on elevation (admin/sudo may cover both scopes; user defaults to `user`)

**Key Functions**:
- `removeCmd.RunE`: Main removal execution
- `findFontFamilyFiles`: Locates font files by family name
- `removeFont`: Core font removal logic
- `resolveFontNameOrID`: Resolves Font IDs to font names for lookup
- `extractFontFamilyNameFromPath`: Extracts font family name from file path using SFNT metadata
- `extractFontDisplayNameFromPath`: Extracts display name from file path using SFNT metadata
- `isCriticalSystemFont`: Checks if a font file is a protected system font

**Interfaces**:
- Uses `internal/cmdutils` for CLI helpers (elevation, manifest checks, and related command utilities)
- Uses `internal/platform` for OS-specific operations and font metadata extraction
- Uses `internal/installations` for **`installation_registry.json`** (resolve/remove Font ID paths; preflight helpers)
- Uses `internal/repo` for font repository access and Font ID resolution
- Uses `internal/output` for verbose/debug output and status reporting
- Uses `internal/components` for progress bar display
- Uses `internal/shared` for protected font checking
- Uses `internal/ui` for user interface styling where needed

**Status**: ✅ Active - Core functionality

### `backup.go`
**Purpose**: Font backup command
**Functionality**:
- Backs up installed font files to a zip archive
- Organizes fonts by source (repository source name) and then by family name
- Auto-detects accessible scopes based on elevation (user vs admin/sudo)
- Fonts are deduplicated across scopes - if the same font exists in both scopes, only one copy is included
- System fonts are always excluded from backups
- Uses progress bar for backup operation with smooth per-file progress updates
- **Date-based Filenames**: Default filename format is `font-backup-YYYY-MM-DD.zip` (e.g., `font-backup-2024-01-15.zip`)
- **Overwrite Confirmation**: Prompts user before overwriting existing backup files
- **Zip comment marker**: Writes `FontGet backup;format=1` (`shared.BackupZipComment`) so `fontget add` can detect FontGet backups

**Key Functions**:
- `backupCmd.RunE`: Main backup execution
- `runBackupWithProgressBar`: Backup operation with progress bar display
- `performBackupWithProgress`: Backup operation with progress updates (per-file progress tracking)
- `performBackup`: Backup operation for debug mode
- `validateAndNormalizeOutputPath`: Path validation with overwrite confirmation
- `generateDefaultBackupFilename`: Generates date-based default filename
- `detectAccessibleScopes`: Auto-detects accessible font scopes based on elevation

**Key Features**:
- **Progress Tracking**: Updates progress bar per file for smooth progress indication
- **Scope Detection**: Automatically detects which scopes are accessible (user vs machine)
- **File Organization**: Organizes fonts by source → family name in zip archive
- **Deduplication**: Prevents duplicate font files across scopes
- **Date-based Naming**: Default filenames include date for easy organization
- **Safe Overwrite**: Confirmation dialog prevents accidental overwrites

**Interfaces**:
- Uses `internal/platform` for OS-specific font detection and scope management
- Uses `internal/output` for verbose/debug output
- Uses `internal/repo` for font matching and repository access
- Uses `internal/components` for progress bar and confirmation dialogs
- Uses `internal/ui` for user interface styling
- Uses `internal/shared` for protected font checking and backup zip comment constant

**Status**: ✅ Active - Core functionality

**Purpose**: Font export command
**Functionality**:
- Exports installed fonts to a JSON manifest file
- Matches installed fonts to repository entries to include Font IDs, License, Categories, and Source
- Supports filtering by match string, source, or export all fonts
- System fonts are always excluded from exports
- Supports output to directory (creates date-based filename) or specific file path via -o flag
- Uses pin spinner for progress feedback during export
- Provides verbose/debug output following logging guidelines
- **Date-based Filenames**: Default filename format is `fontget-export-YYYY-MM-DD.json` (e.g., `fontget-export-2024-01-15.json`)
- **Overwrite Confirmation**: Prompts user before overwriting existing export files
- **Nerd Fonts Support**: Groups families by Font ID to handle cases where one Font ID installs multiple families (e.g., ZedMono installs ZedMono, ZedMono Mono, and ZedMono Propo)

**Key Functions**:
- `exportCmd.RunE`: Main export execution
- `performFullExportWithResult`: Complete export process with result tracking (groups by Font ID)
- `performFullExport`: Export process for debug mode
- `validateAndNormalizeExportPath`: Path validation with overwrite confirmation
- `generateDefaultExportFilename`: Generates date-based default filename
- `collectFonts`: Collects fonts from specified scopes (reused from list.go)
- `groupByFamily`: Groups fonts by family name (reused from list.go)

**Key Features**:
- **Directory Support**: `-o` flag accepts directories (creates date-based default filename) or file paths (winget-style)
- **Date-based Naming**: Default filenames include date for easy organization
- **Safe Overwrite**: Confirmation dialog prevents accidental overwrites
- **Font Matching**: Uses optimized index-based matching to repository entries
- **Filtering**: Supports `--match`, `--source`, `--all`, and `--matched` flags
- **Export Manifest**: JSON structure with metadata, font details, and variants
- **Nerd Fonts Handling**: Groups multiple families under one Font ID entry with `family_names` array

**Interfaces**:
- Uses `internal/platform` for OS-specific font detection
- Uses `internal/output` for verbose/debug output
- Uses `internal/repo` for font matching and repository access
- Uses `internal/components` for confirmation dialogs
- Uses `internal/ui` for spinner components
- Uses `internal/shared` for protected font checking

**Status**: ✅ Active - Core functionality

### `import.go`
**Purpose**: Font import command
**Functionality**:
- Imports fonts from an export manifest file
- Validates export file structure and font availability
- Resolves Font IDs and installs missing fonts
- Shows per-font installation status
- Provides progress feedback during import
- **Nerd Fonts Support**: Deduplicates by Font ID and displays comma-separated family names in success messages
- **Pre-installation Check**: Checks if fonts are already installed before downloading to save bandwidth and time

**Key Functions**:
- `importCmd.RunE`: Main import execution
- `importFontsInDebugMode`: Debug mode import processing
- Font deduplication by Font ID to prevent duplicate installations

**Key Features**:
- **Manifest Validation**: Validates export file structure and version
- **Font Resolution**: Resolves Font IDs to font names for installation
- **Status Reporting**: Shows installation status for each font with comma-separated family names for Nerd Fonts
- **Error Handling**: Handles missing fonts, invalid Font IDs, and installation failures
- **Backward Compatibility**: Handles both old format (`family_name`) and new format (`family_names` array)
- **Nerd Fonts Handling**: Deduplicates by Font ID and shows all families in success message (e.g., "Installed ZedMono, ZedMono Mono, ZedMono Propo")
- **Already-Installed Detection**: Uses same matching logic as list command to detect already-installed fonts before downloading

**Interfaces**:
- Uses `internal/cmdutils` for CLI helpers (file checks, elevation)
- Uses `internal/config` for manifest access where needed
- Uses `internal/repo` for font repository access and Font ID resolution
- Uses `internal/platform` for installation scope and font directories
- Uses `internal/output` for verbose/debug output
- Uses `internal/ui` for user interface styling
- Uses `internal/components` for progress bar and operation items
- Uses `internal/shared` for shared utilities
- Calls `installFont` and related installation helpers defined in `add.go` (same `cmd` package—not a separate import path)

**Status**: ✅ Active - Core functionality (UI/UX improvements pending)

### `sources.go`
**Purpose**: Sources management command
**Functionality**:
- Manages font sources (built-ins from `internal/sources` defaults plus custom entries in the manifest)
- Provides subcommands for info, update, management, and validation
- Handles source configuration and updates
- Validates cached source integrity

**Key Functions**:
- `sourcesCmd`: Main sources command
- `sourcesInfoCmd`: Source information display
- `sourcesUpdateCmd`: Source update functionality
- `sourcesValidateCmd`: Validate cached sources integrity
- `runSourcesUpdateVerbose`: Verbose update mode

**Subcommands**:
- `info` - Show sources information
- `update` - Update source data
- `manage` - Interactive source management (TUI)
- `validate` - Validate cached sources integrity

**Interfaces**:
- Uses `internal/config` for manifest management
- Uses `cmd/sources_helpers.go` for source sorting (`SortSources`)
- Uses `internal/repo` for font data
- Uses `internal/sources` for default source name ordering in `sources info`
- Uses `internal/output` for verbose/debug output

**Status**: ✅ Active - Core functionality

### `sources_cli.go`
**Purpose**: Non-interactive sources subcommands (`add`, `remove`, `enable`, `disable`, `set`) for scripts and CI
**Functionality**: Cobra wiring and flags for manifest-backed source changes without the TUI
**Interfaces**: Uses `internal/config`, `internal/repo`, and related packages as appropriate per subcommand
**Status**: ✅ Active

### `sources_helpers.go`
**Purpose**: Source list sorting/validation helpers shared by sources TUI and CLI
**Functionality**: `SortSources`, `SourceItem`, and form validators used by sources management
**Status**: ✅ Active

### `sources_manage.go`
**Purpose**: Interactive sources management TUI
**Functionality**:
- Provides interactive terminal UI for managing sources
- Allows adding, editing, and removing custom sources
- Handles source priority and configuration
- Supports built-in source management
- Uses reusable TUI components (CheckboxList, ButtonGroup) for consistent UI

**Key Functions**:
- `NewSourcesModel`: TUI model initialization
- `Update`: Main message handler that routes to state-specific handlers
- `routeStateUpdate`: Routes messages to appropriate state handler based on current state
- `addSource`: Adding new sources
- `updateSource`: Editing existing sources
- `saveChanges`: Persisting changes to manifest
- `initCheckboxList`: Initializes checkbox list component from sources
- `syncCheckboxListToSources`: Syncs checkbox state to sources
- `syncSourcesToCheckboxList`: Syncs source state to checkbox list

**Key Features**:
- **Checkbox Component**: Uses `components.CheckboxList` for source enable/disable management
- **Button Components**: Uses `components.ButtonGroup` for confirmation dialogs (save, delete)
- **Plain Source Names**: Source names use `ui.Text` (plain text) with styled tags via `ui.RenderSourceTag()`
- **Consistent UI**: Shares components with enhanced onboarding for unified experience

**Interfaces**:
- Uses `internal/config` for manifest operations
- Uses `cmd/sources_helpers.go` for source utilities (`SortSources`)
- Uses `internal/ui` for TUI components and styling
- Uses `internal/components` for reusable TUI components (CheckboxList, ButtonGroup)
- Uses Bubble Tea for TUI framework

**Status**: ✅ Active - Core functionality

### `sources_update.go`
**Purpose**: Sources update TUI
**Functionality**:
- Provides interactive progress display for source updates
- Shows real-time update progress with spinners
- Handles both verbose and non-verbose modes
- Displays update results and error handling

**Key Functions**:
- `NewUpdateModel`: Update model initialization
- `RunSourcesUpdateTUI`: TUI execution
- `updateNextSource`: Source update logic

**Interfaces**:
- Uses `internal/config` for manifest operations
- Uses `cmd/sources_helpers.go` for source sorting (`SortSources`)
- Uses `internal/ui` for TUI components
- Uses Bubble Tea for TUI framework

**Status**: ✅ Active - Core functionality

### `config.go`
**Purpose**: Configuration management command
**Functionality**:
- Manages FontGet application configuration
- Handles configuration file operations
- Provides configuration validation and migration
- Supports subcommands for different configuration operations

**Key Functions**:
- `configCmd`: Main configuration command
- `configInfoCmd`: Display current configuration
- `configEditCmd`: Open configuration file in editor
- `configValidateCmd`: Validate configuration file integrity
- `configResetCmd`: Reset configuration to defaults
- `configSetCmd`: Set a single config key (via `internal/config.SetConfigKey`)

**Subcommands**:
- `info` - Display current configuration
- `edit` - Open config file in editor
- `validate` - Validate configuration file integrity
- `reset` - Reset configuration to defaults
- `set` - Set a configuration key

**Interfaces**:
- Uses `internal/config` for configuration operations
- Uses `internal/output` for verbose/debug output
- Uses `internal/components` for confirmation dialogs
- Uses `internal/ui` for user interface

**Status**: ✅ Active - Core functionality

### `version.go`
**Purpose**: Version information command
**Functionality**:
- Displays FontGet version information
- Shows build details and manifest version

**Key Functions**:
- `versionCmd`: Version command execution

**Interfaces**:
- Uses `internal/version` for version information

**Status**: ✅ Active - Core functionality

### `browse.go` / `browse_model.go`
**Purpose**: Interactive font browser TUI (Bubble Tea) over repository data
**Status**: ✅ Active

### `theme.go` / `theme_layout.go`
**Purpose**: `fontget theme` — inspect and change the active terminal theme (non-TUI and layout helpers)
**Status**: ✅ Active

### `update.go`
**Purpose**: `fontget update` — check and apply self-updates via `internal/update`
**Status**: ✅ Active

### `completion.go`
**Purpose**: Shell completion generation for Cobra commands
**Status**: ✅ Active

### `progress_steps.go`
**Purpose**: Shared install progress step identifiers used by add/import flows and UI
**Status**: ✅ Active

### `cancel_messages.go`
**Purpose**: Cancellation user-facing copy and helpers (`IsCancelErr`, finish-cancel messaging) for install/remove flows
**Status**: ✅ Active

### `install_tracking.go`
**Purpose**: Per-package install/remove progress persistence into the installation registry (`installTracker`, incomplete state)
**Status**: ✅ Active

---

## Internal Packages

### `internal/cmdutils/`
**Purpose**: CLI-specific utilities and helpers
**Files**:
- `doc.go`: Package docs
- `init.go`: CLI initialization helpers (`EnsureManifestInitialized`, `CreateFontManager`)
- `cobra.go`: Cobra integration and elevation-required sentinel (`CheckElevation`, `PrintElevationHelp`)
- `args.go`: CLI argument parsing (`ParseFontNames`)
- `errors.go`: Standardized error/warning/info printers
- `path.go`: Validate/normalize export and backup output paths
- `scopes.go`: Detect accessible install scopes
- `repository.go`: CLI wrappers for repository operations with logging

**Key Features**:
- **CLI-Specific**: All functions are designed for CLI command context
- **Standardized Error Handling**: Provides consistent error messages with verbose/debug output
- **Logger Interface**: Uses minimal `Logger` interface to avoid circular dependencies
- **Repository Wrappers**: CLI-specific wrappers around `internal/repo/` with logging

**Guidelines**:
- ✅ Use for code that needs Cobra context or CLI-specific error handling
- ✅ Use for CLI wrappers around internal packages
- ❌ Don't use for general-purpose utilities (use `internal/shared/` instead)

**Status**: ✅ Active - CLI utilities

### `internal/shared/`
**Purpose**: General-purpose utilities that are domain-agnostic
**Files**:
- `doc.go`: Package docs
- `font.go`: Font formatting utilities (`FormatFontNameWithVariant`, `GetFontDisplayName`, etc.)
- `file.go`: File utilities (`FormatFileSize`, `SanitizeForZipPath`, `TruncateString`)
- `matching.go`: Font matching utilities (`FindSimilarFonts`)
- `errors.go`: Error types (`FontNotFoundError`, `FontInstallationError`, cancel sentinel, etc.)
- `system_fonts.go`: System font utilities (`IsCriticalSystemFont`)
- `repository.go`: Font query resolution (`ResolveFontQuery`, `GetSourceNameFromID`)
- `progress.go`: `Clamp01` progress helper
- `textwrap.go`: Word-aware text wrapping

**Key Features**:
- **General-Purpose**: Pure utilities with no CLI dependencies
- **Reusable**: Can be used by commands, tests, or other internal packages
- **Domain-Agnostic**: Not tied to any specific feature area

**Guidelines**:
- ✅ Use for pure utility functions
- ✅ Use for code that could be used outside CLI context
- ❌ Don't use for CLI-specific code (use `internal/cmdutils/` instead)

**Status**: ✅ Active - General utilities

### `internal/config/`
**Purpose**: Configuration management
**Files**:
- `user_preferences.go`: User preferences configuration (renamed from `app_config.go`)
  - **AppConfig structure**: Configuration, Logging, Update, Theme, and Search sections
  - **ConfigVersion**: Tracks config schema version for migration support (CurrentConfigVersion = "2.0", stored as `version` in YAML)
  - **Search section**: Search.ResultLimit for configurable result limiting (0 = unlimited, default)
  - **Theme configuration**: `Theme` section with `Name` (theme file name) and `Use256ColorSpace` (bool to downsample theme hex colors to ANSI 256 for terminals without true color support)
  - Configuration loading, saving, validation, and schema-aware migration
  - **Schema Migration System**:
    - Schema defaults and comments defined in embedded `default_config.yaml`
    - `CurrentConfigVersion` compared to `version` in YAML to decide if migration is needed
    - `migrate.go` provides `NeedsSchemaMigration()` / `MigrateToCurrentSchema()` plus explicit rules (e.g., `Theme: \"arasaka\"` → `Theme: { Name: \"arasaka\", Use256ColorSpace: false }`, field renames/moves via `fieldRenameMap` / `fieldMoveMap`)
    - `MigrateConfigAfterUpdate()` ensures configs are bumped to the latest version after binary updates while preserving user values
  - **Helper functions**: `ExpandLogPath()` (expands $home in log paths), `ParseMaxSize()` (parses "10MB" format)
- `default_config.yaml`: Embedded default configuration template and schema used for initial config generation and as the baseline for migrations
- `migrate.go`: Schema migration helpers (`NeedsSchemaMigration`, `MigrateToCurrentSchema`, `copyMatchingKeys`, `applyExplicitMigrationRules`)
- `set.go`: `SetConfigKey` and settable key enumeration for `fontget config set`
- `app_state.go`: Core application state types and functions
  - First-run state management
  - Source acceptance tracking
- `manifest.go`: Font sources manifest management; on load, merges any **missing built-in** source rows from current defaults (URL, prefix, priority, enabled) and persists when the manifest was updated
- `validation.go`: Configuration validation

**Key Features**:
- **Schema-Aware Config Migration**:
  - Versioned schema defined in embedded `default_config.yaml` with `CurrentConfigVersion`
  - Explicit migration rules in `migrate.go` for field renames/moves and structural changes (e.g., scalar → object `Theme` section)
  - Existing configs automatically migrated to the current schema on load while preserving user customizations
- **Search Configuration**: Search.ResultLimit allows users to limit search results (0 = unlimited)
- **Theme Configuration**: Users can set theme name and 256-color behavior in `config.yaml`
  - Theme files must be placed in `~/.fontget/themes/` directory
  - Empty theme name uses embedded default (Catppuccin)
  - `Theme.Use256ColorSpace` controls optional ANSI 256 downsampling for terminals without true color support
- **Logging Configuration**: LogPath, MaxSize, and MaxFiles from `config.yaml` are connected to logger
  - LogPath supports `$home` variable expansion
  - MaxSize parses string format (e.g., "10MB") to integer
- **Update Configuration**: `CheckForUpdates`, `UpdateCheckInterval`, `LastUpdateCheck`, and `NextUpdateCheck` are fully connected
  - `UpdateCheckInterval` controls how often update checks and prompts are shown
  - `NextUpdateCheck` is advanced when the user declines an update so prompts are suppressed until the next interval

**Status**: ✅ Active - Core configuration system with theme support, schema versioning, and full config connections

### `internal/repo/`
**Purpose**: Font repository management
**Files**:
- `sources.go`: Source data loading and caching; `SourceURLs` and search tie-break priority align with `internal/sources` constants and default source order
- `manifest.go`: Font manifest operations
- `search.go`: Font search functionality
- `font.go`: Font data structures, Font ID resolution, downloads, and `DownloadAndExtractFont` (including `DownloadFontOptions` such as `ArchiveSourcePrefix` for archive layout selection after extract)
- `font_matches.go`: Font matching logic for installed fonts to repository entries
- `normalize.go`: `FontKey` / `BaseFamilyName` string normalizers for family matching (e.g. Nerd Fonts-style suffixes)
- `metadata.go`: Font metadata handling
- `checksum.go`: SHA-256 expected-digest parsing/verification
- `download_candidates.go`: Ranked download URL candidates from file maps
- `download_headers.go`: HTTP response header parsing/inference helpers for downloads (e.g., detecting ZIP archives served behind `.ttf` URLs)
- `progress_reader.go`: Throttled download progress reader
- `source_cache.go`: Per-source JSON cache paths/IO
- `archive.go`: Archive type detection and helpers (path safety, symlink skipping)
- `archive_extract.go`: Extraction budget/size enforcement
- `archive_extract_selection.go`: Webfont-path filtering and static vs variable font install policy on extracted paths
- `archive_inspect.go`: ZIP central-directory inspection
- `archive_install_pick.go`: Choosing installable paths inside archives (known upstream layouts when prefix matches, otherwise agnostic directory scoring with fallback)
- `archive_policy.go`: Source-specific archive install policy (e.g. Nerd Fonts)
- `archive_types.go`: `ArchiveEntry` types and safety sentinels
- `archive_validate.go`: Font magic-byte format detection
- `types.go`: Type definitions

**Key Features**:
- **Font Matching**: Optimized index-based matching of installed fonts to repository entries
- **Font ID Resolution**: Resolves Font IDs (e.g., "google.roboto") to font names
- **Name Normalization**: `FontKey` / `BaseFamilyName` live here (not a separate `internal/normalize` package)
- **Source Priority**: Handles multiple repository matches using predefined source priority order
- **Nerd Fonts Support**: Special handling for Nerd Fonts naming conventions and variants
- **Robust Download/Archive Handling**:
  - Supported archives: ZIP, TAR.XZ, TAR.GZ (pure Go; FontGet does not invoke an external archiver)
  - Archive detection uses extension, HTTP headers (Content-Type / Content-Disposition), and file magic bytes (final truth)
  - Prevents archives from being mis-installed as `.ttf` when upstream naming is misleading (notably Font Squirrel)
  - Only ZIP / TAR.XZ / TAR.GZ are extracted; other formats are not treated as known archives
  - **Post-extract selection**: Validated font paths may be narrowed by `PickInstallableFontPathsFromArchive` (invoked from `DownloadAndExtractFont`) when an archive contains both desktop and web-kit trees or mixed static/variable layouts

**Status**: ✅ Active - Core repository system

### `internal/network/`
**Purpose**: HTTP download clients, fallback runners, and process/stall helpers used by repository downloads
**Files**:
- `download_http_client.go`: Shared HTTP transports/clients for downloads
- `download_fallbacks.go`: curl/wget/PowerShell fallback download runners (argv-only `exec.Command`)
- `bot_challenge.go`: Detect WAF/bot-challenge HTTP responses
- `classify.go`: Download retry/action classification sentinels
- `exec.go`: Cancellable external process exec with stall detection
- `exec_unix.go` / `exec_windows.go`: Platform process terminate/kill
- `stall_detector.go`: `StallDetectingReader` inactivity watchdog

**Status**: ✅ Active - Download networking layer

### `internal/installations/`
**Purpose**: Persist FontGet install provenance beside the sources manifest (`installation_registry.json` under **`~/.fontget/`**, basename **`installations.FileName`**).
**Files**:
- `registry.go`: Types, **`Load`** / **`Save`**, **`RecordInstallation`** / **`RemoveInstallation`**, **`PathIndex`** / **`FamilyInstallationsIndex`**, **`BasenamesForDir`**, **`NormalizePathKey`**, **`RegistryPath`**
- `registry_migrate.go`: **`schema_version`** migration — **`buildRegistryMigrations()`** lists allowed one-hop transitions from older on-disk labels to the current constant in **`registry.go`**; **`Load`** runs migrations after JSON decode and rewrites the file when any step applied; unknown versions fail **`Load`**
- `registry_merge.go`: Merge registry metadata into list family groups
- `registry_removal.go`: When/how remove consults the registry
- `recovery.go`: Persist incomplete-rollback recovery records
- `lock.go` / `lock_unix.go` / `lock_windows.go`: Registry file lock orchestration

**Key Features**:
- **`schema_version`** uses semver-style strings (e.g. **`1.0`**); bump **`schemaVersion`** and extend **`buildRegistryMigrations()`** when the persisted JSON contract changes
- **`CurrentRegistrySchemaVersion()`** exposes the schema string for this binary

**Status**: ✅ Active

### `internal/platform/`
**Purpose**: Cross-platform operations
**Files**:
- `platform.go`: Platform abstraction, scopes, `FontManager` API, `OpenURL`, `ExtractFontMetadata`
- `opentype_tables.go`: Lightweight SFNT table directory checks (e.g. `fvar` for variable fonts) used by repository archive install policy
- `windows.go`: Windows-specific `FontManager`
- `darwin.go`: macOS-specific `FontManager`
- `linux.go`: Linux-specific `FontManager`
- `windows_utils.go`: Win32 GDI/registry font registration helpers
- `scope.go`: Scope detection utilities (`AutoDetectScope`)
- `temp.go`: FontGet temporary directory helpers
- `mutation.go`: Place/rollback font file mutations
- `mutation_windows.go` / `mutation_other.go`: Platform registration undo/restore
- `machine_remove.go`: Machine-scope Fonts registry remove ops

**Key Features**:
- **Font Metadata Extraction**: `ExtractFontMetadata()` reads font family name, style name, and full name directly from font file SFNT name table
- **Cross-platform Font Management**: Unified interface for font installation/removal across Windows, macOS, and Linux
- **Elevation Detection**: Privilege checking lives in platform OS files and `internal/cmdutils` (there is no `elevation.go`)
- **Font Directory Management**: Scope-aware font directory resolution
- **Scope Detection**: Auto-detection of installation scope based on elevation
- **Safe Mutations**: Place/rollback helpers for install/remove file operations

**Status**: ✅ Active - Cross-platform support

### `internal/ui/`
**Purpose**: Terminal UI styling, layout helpers, and theme loading (distinct from `internal/components`, which holds reusable Bubble Tea widgets)

**Files**:
- `components.go` — High-level render helpers (`RenderTitleWithSubtitle`, `RenderStatusReport`, `RenderSearchResults`, loading/error/success screens), `RunSpinner` (blocking spinner around a function), and `SimpleProgressBar` for lightweight progress
- `styles.go` — Lipgloss style variables, `InitStyles()` / theme wiring, semantic colors, table and form styles, `RenderSourceTag` / `RenderSourceNameWithTag`, dialog/modal styles (`DialogModal`, etc.), spinner color fields (`SpinnerColor`, `SpinnerDoneColor`), and related theme-driven adjustments (including optional ANSI 256 downsampling via `Theme.Use256ColorSpace` in `config.yaml`, plus helpers like `ColorOrNoColor` for system-theme terminal defaults)
- `theme.go` — Loads embedded and user themes from YAML (`~/.fontget/themes/`), `ThemeManager`, mode handling, color lookup for `InitStyles()`
- `theme_discovery.go` — `DiscoverThemes`, `ThemeInfo` / `ThemeOption` for theme picker and onboarding
- `tables.go` — Table column width constants, `GetSearchTableHeader`, `GetListTableHeader`, `GetTableSeparator`, etc.
- `spinner_model.go` — `NewSpinnerModel` for Bubble Tea–driven blocking spinners (used where a full `tea.Program` model is needed)
- `url_format.go` — `FormatTerminalURL` / `FormatTerminalURLChunk` for OSC 8 hyperlinks in supporting terminals
- `url_format_test.go` — Tests for URL formatting
- `themes/` — Bundled YAML theme definitions (multiple files; includes embedded defaults such as Catppuccin plus additional bundled themes—see the directory for the current set). Users can add more under `~/.fontget/themes/`.

**Key Features**:
- **Theme system**: YAML themes with semantic color keys; default theme embedded; user themes override by filename
- **Centralized styling**: `InitStyles()` applies theme colors across commands
- **Unified table API**: Shared headers and column conventions
- **Spinners**: Both simple blocking (`RunSpinner`) and full Bubble Tea model (`NewSpinnerModel`) paths, with theme-based spinner colors
  - Spinner model enforces a minimum display time only when showing a completion message; fast operations that clear the line return immediately

**Status**: ✅ Active - UI system with theme support

### `internal/output/`
**Purpose**: Output management
**Files**:
- `verbose.go`: Verbose output handling with operation details display
- `debug.go`: Debug output handling
- `status.go`: Status report types and functions (`StatusReport`, `PrintStatusReport`)

**Key Features**:
- **Consistent Formatting**: Standardized `[INFO]`, `[WARNING]`, `[ERROR]` prefixes
- **Debug Mode Helpers**: `IsDebugOutputEnabled()` supports disabling interactive UI elements (e.g., spinners) when debug output must stay readable
- **Operation Details Display**: `DisplayFontOperationDetails()` shows formatted installation/removal details
- **Download Size Tracking**: Integrated file size display in verbose output
- **Status Reporting**: Unified status report display for operations
- **Clean API**: Interface-based design prevents circular imports
- **Verbose Output Spacing**: Verbose sections use conditional `fmt.Println()` pattern (only add blank line if verbose was shown) per spacing framework guidelines

**Status**: ✅ Active - Output system

### `internal/logging/`
**Purpose**: File logging system
**Files**:
- `logger.go`: Logger implementation with file rotation and level management
  - `New()`: Creates logger with default OS-specific log directory
  - `NewWithPath()`: Creates logger with custom log file path (used for config.yaml LogPath)
  - `SetGlobal()` / `CloseGlobal()`: Register and tear down the single process-wide logger (called from `cmd/root.go` after resolving `Logging.LogPath`)
  - `GetLogger()`: Returns the active logger set by `SetGlobal()` (nil before CLI init, e.g. in isolated tests unless they register a logger)
  - `ActiveLogDir()`: Directory containing the active `fontget.log` (used by `fontget --logs`)
- `config.go`: Logging configuration

**Key Features**:
- **Single log file**: `cmd` resolves `Logging.LogPath` from `config.yaml` (with `$home` expansion) and falls back to the OS default log directory when unset or invalid; one `fontget.log` path is registered for the whole process (older builds may have left stray files under the OS-only path, e.g. `~/Library/Logs/fontget` on macOS; they are no longer written by current versions when `LogPath` is set)
- **Config Integration**: LogPath, MaxSize, and MaxFiles from `config.yaml` are connected
  - LogPath supports `$home` variable expansion (e.g., `$home/.fontget/logs/fontget.log`)
  - MaxSize parses string format (e.g., "10MB") to integer megabytes
  - MaxFiles controls number of rotated log files to keep
- **Log rotation**: Automatic rotation based on size, age, and backup count
- **Level management**: Log levels (ErrorLevel, InfoLevel, DebugLevel) are set from CLI flags and config in `cmd/root.go`
- **Always active (after init)**: `logging.GetLogger()` (and `cmd.GetLogger()`, which delegates to it) writes to the same file regardless of verbose/debug flags
  - `GetLogger()` calls should NOT be conditional on `IsVerbose()` or `IsDebug()`
  - Logger writes to file, not console (console output is handled by verbose/debug output system)

**Usage Pattern**:
- All commands should log: operation start, parameters, errors, and completion
- In `cmd/`, use `GetLogger().Info()` (delegates to `logging.GetLogger()`); in `internal/` packages, use `logging.GetLogger()` directly
- Use `GetLogger().Error()` for all error cases
- Use `GetLogger().Warn()` for warnings
- Use `GetLogger().Debug()` for detailed debugging information

**Status**: ✅ Active - Logging system with config.yaml integration

### `internal/sources/`
**Purpose**: Built-in FontGet-Sources definitions shared across config defaults, repository URL maps, and ordering helpers
**Files**:
- `urls.go`: Base URL constants, per-source JSON URLs, `DefaultSources()` (name → URL, prefix, filename, priority, enabled), and `DefaultSourceNamesInPriorityOrder()` for consistent ordering in search, repo, onboarding, and CLI

**Status**: ✅ Active - Source definitions

### `internal/version/`
**Purpose**: Version management
**Files**:
- `version.go`: Version information

**Status**: ✅ Active - Version management

### `internal/update/`
**Purpose**: Self-update system
**Files**:
- `update.go`: Update implementation (CheckForUpdates, UpdateToLatest, UpdateToVersion)
- `release.go`: Latest-release permalink discovery and immutable tagged asset downloads
- `checksum.go`: SHA-256 verification against GoReleaser `checksums.txt`
- `extract.go`: Extract `fontget` / `fontget.exe` from zip or tar.gz
- `apply.go`: Atomic binary replace with `.old` rollback
- `archive.go`: GoReleaser archive naming for the current platform
- `check.go`: Update checking logic (ShouldCheckForUpdates, PerformStartupCheck)

**Key Features**:
- **GitHub Releases**: Resolves the public `/releases/latest` redirect, validates its version tag, then uses immutable tagged download URLs
- **No runtime credentials**: Public release downloads do not use the GitHub API, `GITHUB_TOKEN`, or `GH_TOKEN`
- **Checksum verification**: Downloads `checksums.txt` and verifies the archive SHA-256 before install
- **Startup check**: Reads `config.yaml` `CheckForUpdates` and `UpdateCheckInterval` (there is no `AutoUpdate` toggle)
- **Timestamps**: `LastUpdateCheck` / `NextUpdateCheck` track when checks ran and when the next prompt is allowed
- **Non-blocking**: Startup checks run in goroutine to avoid blocking application startup
- **Error handling**: Graceful fallback if update check fails (silent failure during startup)

**Status**: ✅ Active - Self-update system with config.yaml integration

### `internal/templates/`
**Purpose**: Embedded YAML theme templates for theme authoring / reset flows
**Files**:
- `dark-theme-template.yaml`: Dark theme template
- `light-theme-template.yaml`: Light theme template

**Status**: ✅ Active - Theme templates (no Go sources)

### `internal/components/`
**Purpose**: Reusable Bubble Tea / CLI UI widgets
**Files**:
- `progress_bar.go`: Multi-item progress display model
- `card.go` / `card_sections.go`: Bordered card and multi-section card body layout
- `confirm.go`: Confirmation dialog model
- `dialog.go`: Modal dialog renderer
- `overlay.go`: Overlay/modal compositing model
- `preview.go`: Theme preview TUI model
- `status_popup.go`: Centered install/status popup renderer
- `switch.go`: Toggle switch widget
- `checkbox.go`: Navigable checkbox list
- `button.go`: Button / button-group widgets
- `table_custom.go`: Viewport-controlled custom table
- `table_interactive.go`: Bubble Tea interactive table model
- `table_static.go`: Static CLI table renderer
- `table_utils.go`: Table config/modes/column helpers

**Key Features**:
- **Progress**: Shared progress model for add/remove/import/update flows
- **Tables**: Static, interactive, and custom table paths with shared helpers
- **Cards / dialogs / overlays**: Info display and modal confirmation UI
- **Switches / checkboxes / buttons**: Onboarding and sources-manage controls
- **Theme preview / status popup**: Theme picker and install status surfaces

**Usage Examples**:
- **Add/Remove Commands**: Progress bar for font installation/removal
- **Info Command**: Card components for font details and license info
- **Sources Management / Onboarding**: CheckboxList, ButtonGroup, Switch
- **Browse / Theme**: Interactive tables and preview models

**Status**: ✅ Active - UI components

### `internal/testutil/`
**Purpose**: Test helpers shared across packages
**Files**:
- `home.go`: Set `HOME` / `USERPROFILE` for tests
- `minifont.go`: Minimal TTF bytes for metadata tests

**Status**: ✅ Active - Test utilities

### `internal/onboarding/`
**Purpose**: First-run onboarding, Terms of Use, and setup wizard (there is no separate `internal/license` package)
**Files**:
- `terms_of_use.yaml`: Single source of truth for the Terms of Use screen. **Section-based**: `sections` is an ordered list; each section has `name`, `style` (e.g. PageTitle, Text, InfoText, SourceName), and either `content` (string) or `items` (list for bullets). No layout or colors are defined in code—reorder, add, or restyle by editing the file. Same structure would work as JSON.
- `terms.go`: Embeds `terms_of_use.yaml`, unmarshals into `[]Section`, and exports `TermsOfUseSections()`, `StyleRenderer(styleName)` (maps style keys to ui renderers), plus backward-compat getters (`TermsOfUseTitle()`, `TermsOfUseIntroText()`, etc.) that look up by section name.
- `onboarding.go`: Core onboarding flow management
  - `RunFirstRunOnboarding()`: Executes onboarding on first run
  - `RunWizard()`: Executes onboarding wizard regardless of first-run status (for `--wizard` flag)
  - `OnboardingFlow`: Step-based execution system
- `enhanced_flow.go`: Enhanced interactive TUI onboarding flow
  - **EnhancedOnboardingModel**: Bubble Tea model for interactive onboarding
  - **Step System**: Modular step interface for easy extension
  - **Steps**: Welcome → Terms of Use → Wizard Choice → Sources → Settings → Theme Selection → Completion
  - **Theme Integration**: Full theme picker TUI integrated into onboarding flow
  - **Conditional Navigation**: Skips customization steps if user chooses "Let it ride"

**Key Features**:
- **Interactive TUI**: Full-screen interactive terminal UI using Bubble Tea
- **Welcome Screen**: First-time user welcome message
- **Terms of Use**: Text-based terms and disclaimer acceptance (continuing implies agreement)
- **Wizard Choice**: User can choose to customize settings or accept defaults ("Let it ride")
- **Source Selection**: Interactive source enable/disable with checkbox list
- **Settings Configuration**: Update settings (`CheckForUpdates`, popularity sort, related prefs)
- **Theme Selection**: Full theme picker TUI with preview (only shown if user chose to customize)
- **Completion Screen**: Summary of selections and next steps
- **State Management**: Tracks selections and saves to config on completion
- **Re-runnable**: Can be re-run via `--wizard` flag for reconfiguration

**Key Functions**:
- `NewEnhancedOnboardingModel()`: Creates new onboarding model with all steps
- `SaveSelections()`: Saves user selections to config file
- Step-specific functions: `NewWelcomeStepEnhanced()`, `NewLicenseAgreementStepEnhanced()`, `NewWizardChoiceStepEnhanced()`, etc.

**Interfaces**:
- Uses `internal/config` for configuration management
- Uses `internal/sources` for default source ordering and metadata in source steps
- Uses `internal/ui` for TUI components and styling
- Uses `internal/components` for reusable UI components (CheckboxList, ButtonGroup)
- Uses Bubble Tea for TUI framework

**Status**: ✅ Active - Enhanced onboarding system with interactive TUI


---

## Documentation Files

### `README.md`
**Purpose**: Main project documentation
**Status**: ✅ Active - Project documentation

### `docs/`
**Purpose**: User-facing documentation
**Files**:
- `README.md`: Documentation index
- `usage.md`: Command reference
- `installation.md`: Installation guide (includes Automation / CI for install scripts)
- `troubleshooting.md`: Common issues
- `custom-sources.md`: Adding custom font sources
- `terminal-setup.md`: Terminal setup and shell completions
- `contributing.md`: Contributing guidelines

**Status**: ✅ Active - User documentation

### `docs/development/`
**Purpose**: Development documentation
**Files**:
- `codebase.md`: This file — comprehensive codebase overview
- `build-guide.md`: Build and release steps
- `style-guide.md`: Theming / UI style notes
- `theming.md`: Theme files, structure, and configuration
- `component-architecture.md`: TUI components and patterns
- `bubbletea-layout-patterns.md`: Bubble Tea layout patterns

**Status**: ✅ Active - Development documentation

### `docs/development/guidelines/`
**Purpose**: Development guidelines and best practices
**Files**:
- `codebase-layout-guidelines.md`: Where new code should go
- `logging-guidelines.md`: Logging best practices
- `spacing-guidelines.md`: Output spacing guidelines
- `verbose-debug-guidelines.md`: Verbose and debug output guidelines
- `versioning-guide.md`: Versioning guidelines
- `release-guide.md`: Release process

**Status**: ✅ Active - Development guidelines

### `docs/maintenance/documentation-sync.md`
**Purpose**: Documentation synchronization process and flag-audit stats
**Related**: `docs/maintenance/audit-flags.go` scans `cmd/` for flag registrations

**Status**: ✅ Active - Documentation management

---

## Runtime data (not in the git tree)

### `~/.fontget/` (typical)
**Purpose**: Per-user FontGet home — `config.yaml`, `manifest.json`, `installation_registry.json`, logs, themes, and downloaded FontGet-Sources JSON cache files (filenames correspond to built-in `Filename` fields in `internal/sources.DefaultSources()`). Exact paths are resolved by `internal/config` / `internal/repo` / `internal/installations`.

**Status**: ✅ Active - Runtime data (not committed)

---

## Legacy/Deprecated Files

None currently tracked. Removed packages/paths that docs formerly referenced: `internal/functions/`, `internal/normalize/` (now `internal/repo/normalize.go`), `internal/license/` (terms live in `internal/onboarding/`), `internal/templates/command_template.go`, root `refactor.md`, and a repo-root `sources/` cache directory.

---