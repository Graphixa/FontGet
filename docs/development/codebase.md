# FontGet Codebase Documentation

This document provides a comprehensive overview of the FontGet codebase, explaining the purpose and functionality of each file and how they interface with other components.

## Table of Contents

- [Root Files](#root-files)
- [Command Files (`cmd/`)](#command-files-cmd)
- [Internal Packages](#internal-packages)
- [Documentation Files](#documentation-files)
- [Configuration Files](#configuration-files)
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
- Specifies Go version 1.24.4
- Lists all required dependencies including:
  - Cobra (CLI framework)
  - Bubble Tea (TUI framework)
  - Lipgloss (styling and terminal colors)
  - XZ (archive extraction)
  - Pin (spinner/loading indicators)
  - Various other utilities

**Status**: ✅ Active - Essential for Go module system

---

## Command Files (`cmd/`)

### `root.go`
**Purpose**: Root command definition and global configuration
**Functionality**:
- Defines the main `fontget` command
- Sets up global flags (`--verbose`, `--debug`, `--logs`, `--wizard`)
- Initializes logging configuration
- Registers all subcommands
- **Custom Help Templates**: Custom help and usage templates with controlled section ordering (Usage → Description → Commands → Options → Examples)
- **Wizard Flag**: `--wizard` flag allows re-running the onboarding wizard for reconfiguration

**Interfaces**:
- Imports all command packages
- Uses `internal/logging` for log configuration
- Uses `internal/output` for verbose/debug functionality
- Uses `internal/onboarding` for wizard execution

**Status**: ✅ Active - Core command orchestration

### `add.go`
**Purpose**: Font installation command
**Functionality**:
- Installs fonts from various sources (Google Fonts, Nerd Fonts, Font Squirrel)
- Supports both font names (e.g., "Roboto") and Font IDs (e.g., "google.roboto")
- Handles font search and fuzzy matching
- Manages font installation with progress tracking
- Supports different installation scopes (user/system)
- Provides detailed error handling and suggestions
- Uses shared operation infrastructure for consistent behavior
- **Pre-installation Check**: Checks if fonts are already installed before downloading to save bandwidth and time

**Key Functions**:
- `addCmd.RunE`: Main command execution
- `installFontsInDebugMode`: Debug mode installation (plain text output)
- `installFont`: Core font installation logic (includes pre-download check for already-installed fonts)
- `getSourceName`: Source name resolution
- `showFontNotFoundWithSuggestions`: Error handling with suggestions

**Interfaces**:
- Uses `internal/repo` for font data
- Uses `internal/platform` for OS-specific operations
- Uses `internal/output` for verbose/debug output
- Uses `internal/ui` for user interface
- Uses `cmd/operations` for shared operation logic
- Uses `cmd/handlers` for operation output handlers

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
- **Source Filtering**: Supports filtering by source ID, full name, or short prefix (e.g., "google", "nerd", "squirrel")
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
- Uses `internal/functions` for search utilities
- Uses `internal/output` for verbose/debug output

**Status**: ✅ Active - Core functionality

### `list.go`
**Purpose**: Font listing command
**Functionality**:
- Lists installed fonts on the system
- Matches installed fonts to repository entries to show Font IDs, License, Categories, and Source
- Provides filtering and formatting options
- Shows font details and metadata
- Default scope is "all" (shows fonts from both user and machine scopes)
- Displays columns: Name, Font ID, License, Categories, Type, Scope, Source
- **Font ID Filtering**: Query parameter can match either font family names (e.g., "Roboto") or Font IDs (e.g., "google.roboto")
- **Performance Optimizations**: Early type filtering (filters by extension before metadata extraction) and cached lowercased strings for faster filtering

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

**Flags**:
- `--scope, -s`: Filter by installation scope (user or machine)
- `--type, -t`: Filter by font type (TTF, OTF, etc.)
- `--expand, -x`: Show all font variants in hierarchical view

**Interfaces**:
- Uses `internal/platform` for OS-specific font detection
- Uses `internal/output` for verbose/debug output
- Uses `internal/repo` for font matching and repository access
- Uses `internal/shared` for protected font checking and font utilities

**Status**: ✅ Active - Core functionality

### `info.go`
**Purpose**: Font information command
**Functionality**:
- Shows detailed information about specific fonts
- Displays font metadata, variants, and source information
- Provides comprehensive font details

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
- Handles different removal scopes (user, machine, all)
- When removing from "all" scopes, shows separate progress entries for each scope
- Extracts font names from installed font metadata (SFNT name table)
- Protects critical system fonts from removal
- Provides consistent verbose/debug output matching add command
- Auto-detects scope based on elevation (admin/sudo defaults to "all", user defaults to "user")

**Key Functions**:
- `removeCmd.RunE`: Main removal execution
- `findFontFamilyFiles`: Locates font files by family name
- `removeFont`: Core font removal logic
- `resolveFontNameOrID`: Resolves Font IDs to font names for lookup
- `extractFontFamilyNameFromPath`: Extracts font family name from file path using SFNT metadata
- `extractFontDisplayNameFromPath`: Extracts display name from file path using SFNT metadata
- `isCriticalSystemFont`: Checks if a font file is a protected system font

**Interfaces**:
- Uses `internal/platform` for OS-specific operations and font metadata extraction
- Uses `internal/output` for verbose/debug output
- Uses `internal/repo` for font repository access and Font ID resolution
- Uses `internal/components` for progress bar display
- Uses `internal/shared` for protected font checking
- Uses `internal/output` for status reporting

**Status**: ✅ Active - Core functionality

### `backup.go`
**Purpose**: Font backup command
**Functionality**:
- Backs up installed font files to a zip archive
- Organizes fonts by source (e.g., Google Fonts, Nerd Fonts) and then by family name
- Auto-detects accessible scopes based on elevation (user vs admin/sudo)
- Fonts are deduplicated across scopes - if the same font exists in both scopes, only one copy is included
- System fonts are always excluded from backups
- Uses progress bar for backup operation with smooth per-file progress updates
- **Date-based Filenames**: Default filename format is `font-backup-YYYY-MM-DD.zip` (e.g., `font-backup-2024-01-15.zip`)
- **Overwrite Confirmation**: Prompts user before overwriting existing backup files

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
- Uses `internal/shared` for protected font checking

**Status**: ✅ Active - Core functionality

### `export.go`
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
- Uses `internal/repo` for font repository access and Font ID resolution
- Uses `internal/output` for verbose/debug output
- Uses `internal/ui` for user interface
- Uses `cmd/add` for font installation logic

**Status**: ✅ Active - Core functionality (UI/UX improvements pending)

### `sources.go`
**Purpose**: Sources management command
**Functionality**:
- Manages font sources (Google Fonts, Nerd Fonts, Font Squirrel)
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
- Uses `internal/functions` for source sorting
- Uses `internal/repo` for font data
- Uses `internal/output` for verbose/debug output

**Status**: ✅ Active - Core functionality

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
- Uses `internal/functions` for source utilities
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
- Uses `internal/functions` for source sorting
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

**Subcommands**:
- `info` - Display current configuration
- `edit` - Open config file in editor
- `validate` - Validate configuration file integrity
- `reset` - Reset configuration to defaults

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

### `operations.go`
**Purpose**: Shared font operation infrastructure
**Functionality**:
- Defines core operation types and interfaces
- Provides unified operation execution logic
- Handles font installation and removal operations
- Tracks operation status and results
- Manages download size tracking

**Key Types**:
- `OperationHandler`: Interface for operation output handlers
- `FontOperationType`: Distinguishes install vs remove operations
- `OperationStatus`: Tracks success/skipped/failed counts
- `FontToProcess`: Represents a font to be processed
- `FontOperation`: Complete operation definition
- `ItemResult`: Result of processing a single font

**Key Functions**:
- `executeFontOperation`: Orchestrates operation execution
- `processFontInstall`: Handles download, extraction, and installation
- `processFontRemove`: Handles font finding and removal

**Interfaces**:
- Used by `add.go` and `remove.go` commands
- Uses `internal/platform` for font operations
- Uses `internal/repo` for font data

**Status**: ✅ Active - Shared operation infrastructure

### `handlers.go`
**Purpose**: Operation output handlers
**Functionality**:
- Provides different implementations of `OperationHandler` interface
- Handles output formatting for different modes (debug, verbose, normal, TUI)
- Separates output concerns from operation logic

**Key Types**:
- `DebugHandler`: Plain text output for debug mode
- `VerboseHandler`: Detailed output for verbose mode (prepared for future use)
- `NormalHandler`: Standard output handler (prepared for future use)
- `noOpHandler`: Silent handler when TUI manages all output

**Interfaces**:
- Implements `OperationHandler` interface
- Uses `internal/output` for verbose/debug output
- Uses `internal/components` for TUI integration

**Status**: ✅ Active - Output handler implementations


### `operations.go`
**Purpose**: Shared font operation infrastructure
**Functionality**:
- Defines core operation types and interfaces
- Provides unified operation execution logic
- Handles font installation and removal operations
- Tracks operation status and results
- Manages download size tracking

**Key Types**:
- `OperationHandler`: Interface for operation output handlers
- `FontOperationType`: Distinguishes install vs remove operations
- `OperationStatus`: Tracks success/skipped/failed counts
- `FontToProcess`: Represents a font to be processed
- `FontOperation`: Complete operation definition
- `ItemResult`: Result of processing a single font

**Key Functions**:
- `executeFontOperation`: Orchestrates operation execution
- `processFontInstall`: Handles download, extraction, and installation
- `processFontRemove`: Handles font finding and removal

**Interfaces**:
- Used by `add.go` and `remove.go` commands
- Uses `internal/platform` for font operations
- Uses `internal/repo` for font data

**Status**: ✅ Active - Shared operation infrastructure

---

## Internal Packages

### `internal/cmdutils/`
**Purpose**: CLI-specific utilities and helpers
**Files**:
- `init.go`: CLI initialization helpers (`EnsureManifestInitialized`, `CreateFontManager`)
- `cobra.go`: Cobra integration (`CheckElevation`, `PrintElevationHelp`)
- `args.go`: CLI argument parsing (`ParseFontNames`)
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
- `font.go`: Font formatting utilities (`FormatFontNameWithVariant`, `GetFontDisplayName`, etc.)
- `file.go`: File utilities (`FormatFileSize`, `SanitizeForZipPath`, `TruncateString`)
- `matching.go`: Font matching utilities (`FindSimilarFonts`)
- `errors.go`: Error types (`FontNotFoundError`, `FontInstallationError`, etc.)
- `system_fonts.go`: System font utilities (`IsCriticalSystemFont`)
- `repository.go`: Font query resolution (`ResolveFontQuery`, `GetSourceNameFromID`)

**Key Features**:
- **General-Purpose**: Pure utilities with no CLI dependencies
- **Reusable**: Can be used by commands, tests, or other internal packages
- **Domain-Agnostic**: Not tied to any specific feature area

**Guidelines**:
- ✅ Use for pure utility functions
- ✅ Use for code that could be used outside CLI context
- ❌ Don't use for CLI-specific code (use `internal/cmdutils/` instead)

**Status**: ✅ Active - General utilities

### `internal/functions/`
**Purpose**: Domain-specific utilities
**Files**:
- `sort.go`: Source sorting utilities (`SortSources`)
- `validation.go`: Domain-specific validation utilities

**Key Features**:
- **Domain-Specific**: Utilities specific to a particular feature area
- **Type-Specific**: Operates on domain-specific types (e.g., `SourceItem`)

**Guidelines**:
- ✅ Use for utilities specific to a feature domain
- ❌ Don't use for general-purpose utilities (use `internal/shared/` instead)

**Status**: ✅ Active - Domain utilities

### `internal/config/`
**Purpose**: Configuration management
**Files**:
- `user_preferences.go`: User preferences configuration (renamed from `app_config.go`)
  - **AppConfig structure**: Configuration, Logging, Update, Theme, and Search sections
  - **ConfigVersion**: Tracks config schema version for migration support (CurrentConfigVersion = "2.0")
  - **Search section**: Search.ResultLimit for configurable result limiting (0 = unlimited, default)
  - **Theme configuration**: Theme.Name (theme file name) and Theme.Mode (dark/light)
  - Configuration loading, saving, and validation
  - **Simplified Migration System**: Merge-based migration that preserves user values while applying new defaults
    - Field renames handled via `fieldRenameMap` (e.g., "Update.AutoCheck" → "Update.CheckForUpdates")
    - Field moves handled via `fieldMoveMap` (e.g., "Configuration.EnablePopularitySort" → "Search.EnablePopularitySort")
    - Automatic field mapping via `handleLegacyFieldMapping()` before unmarshaling
    - `MigrateConfigAfterUpdate()` merges old config with new defaults after binary updates
  - **Helper functions**: `ExpandLogPath()` (expands $home in log paths), `ParseMaxSize()` (parses "10MB" format)
- `app_state.go`: Core application state types and functions
  - First-run state management
  - Source acceptance tracking
- `manifest.go`: Font sources manifest management
- `validation.go`: Configuration validation

**Key Features**:
- **Simplified Config Migration System**: Merge-based migration that preserves user values
  - Field renames and moves handled automatically via simple maps
  - Old config merged with new defaults during `GetUserPreferences()` load
  - Migration happens automatically during config loading and after binary updates
  - No complex migration rules - just simple field mapping and merge logic
  - Preserves all user customizations while applying new defaults
- **Search Configuration**: Search.ResultLimit allows users to limit search results (0 = unlimited)
- **Theme Configuration**: Users can set theme name and mode in `config.yaml`
  - Theme files must be placed in `~/.fontget/themes/` directory
  - Empty theme name uses embedded default (Catppuccin)
  - Mode can be "dark" or "light" (defaults to "dark")
- **Logging Configuration**: LogPath, MaxSize, and MaxFiles from config.yaml are connected to logger
  - LogPath supports `$home` variable expansion
  - MaxSize parses string format (e.g., "10MB") to integer
- **Update Configuration**: AutoCheck, AutoUpdate, CheckInterval, and LastChecked are fully connected
  - LastChecked uses UTC timezone for consistency
  - AutoUpdate automatically installs updates when enabled and available

**Status**: ✅ Active - Core configuration system with theme support, migration system, and full config connections

### `internal/repo/`
**Purpose**: Font repository management
**Files**:
- `sources.go`: Source data loading and caching
- `manifest.go`: Font manifest operations
- `search.go`: Font search functionality
- `font.go`: Font data structures and Font ID resolution
- `font_matches.go`: Font matching logic for installed fonts to repository entries
- `metadata.go`: Font metadata handling
- `archive.go`: Archive operations
- `types.go`: Type definitions

**Key Features**:
- **Font Matching**: Optimized index-based matching of installed fonts to repository entries
- **Font ID Resolution**: Resolves Font IDs (e.g., "google.roboto") to font names
- **Source Priority**: Handles multiple repository matches using predefined source priority order
- **Nerd Fonts Support**: Special handling for Nerd Fonts naming conventions and variants

**Status**: ✅ Active - Core repository system

### `internal/platform/`
**Purpose**: Cross-platform operations
**Files**:
- `platform.go`: Platform abstraction and font metadata extraction
- `windows.go`: Windows-specific operations
- `darwin.go`: macOS-specific operations
- `linux.go`: Linux-specific operations
- `elevation.go`: Privilege elevation
- `temp.go`: Temporary file operations
- `windows_utils.go`: Windows utilities
- `scope.go`: Scope detection utilities (`AutoDetectScope`)

**Key Features**:
- **Font Metadata Extraction**: `ExtractFontMetadata()` reads font family name, style name, and full name directly from font file SFNT name table
- **Cross-platform Font Management**: Unified interface for font installation/removal across Windows, macOS, and Linux
- **Elevation Detection**: Platform-specific privilege checking
- **Font Directory Management**: Scope-aware font directory resolution
- **Scope Detection**: Auto-detection of installation scope based on elevation

**Status**: ✅ Active - Cross-platform support

### `internal/ui/`
**Purpose**: User interface components
**Files**:
- `components.go`: UI component definitions and utilities
  - `RunSpinner`: Pin spinner wrapper with progress feedback
  - `hexToPinColor`: Converts hex colors to pin package color constants
  - Various rendering utilities for titles, errors, success messages
  - `RenderSourceTag()`: Renders source type tags (`[Built-in]` or `[Custom]`) independently
  - `RenderSourceNameWithTag()`: Renders source names with tags using colored `TableSourceName` style
- `styles.go`: Centralized styling and theming
  - **Theme-aware styling system**: All styles initialized from theme files via `InitStyles()`
  - **Semantic color system**: Theme files define semantic color keys (accent, warning, error, etc.) that are referenced by multiple styles
  - Page structure styles (titles, subtitles, content)
  - Message styles (Text, InfoText, WarningText, ErrorText, SuccessText)
  - Data display styles (tables, lists)
  - Form styles (labels, inputs, placeholders)
  - Command styles (keys, labels, examples)
  - Card styles (titles, labels, content, borders)
  - Button, Checkbox, and Switch component styles
  - Progress bar styles with gradient support
  - **Spinner styles**: Color constants and pin package color mapping (`SpinnerColor`, `SpinnerDoneColor`, `PinColorMap`)
- `theme.go`: Theme management system
  - **Theme abstraction layer**: Loads themes from YAML files in `~/.fontget/themes/`
  - **Embedded default theme**: Catppuccin theme embedded in binary as fallback
  - **Theme configuration**: Theme selection via `config.yaml` (Theme.Name and Theme.Mode)
  - **Mode support**: Dark and light mode support per theme
  - **ThemeManager**: Handles theme loading, mode switching, and color retrieval
- `tables.go`: Table formatting constants and functions
  - Table column width constants (`TableColName`, `TableColID`, etc.)
  - Table header functions (`GetSearchTableHeader`, `GetListTableHeader`, etc.)
  - Table separator function (`GetTableSeparator`)
- `themes/`: Theme definition files
  - `catppuccin.yaml`: Default embedded theme (Catppuccin Mocha/Latte)
  - `gruvbox.yaml`: Gruvbox theme (available for user installation)

**Key Features**:
- **Theme System**: YAML-based theme files with semantic color system
  - Themes stored in `~/.fontget/themes/` directory
  - Default theme (Catppuccin) embedded in binary
  - Theme selection via `config.yaml` Theme section
  - Supports both dark and light modes
- **Semantic Colors**: Theme files define semantic keys (accent, warning, error, grey_light, etc.) that are mapped to multiple UI styles
- **Centralized Styling**: All styles initialized from theme on startup via `InitStyles()`
- **Unified Table API**: All table formatting in one place for consistency
- **Spinner Integration**: Pin spinner colors mapped from hex to pin package constants
- **Color Mapping**: `PinColorMap` provides hex-to-pin color conversion for spinner components

**Status**: ✅ Active - UI system with theme support

### `internal/output/`
**Purpose**: Output management
**Files**:
- `verbose.go`: Verbose output handling with operation details display
- `debug.go`: Debug output handling
- `status.go`: Status report types and functions (`StatusReport`, `PrintStatusReport`)

**Key Features**:
- **Consistent Formatting**: Standardized `[INFO]`, `[WARNING]`, `[ERROR]` prefixes
- **Operation Details Display**: `DisplayFontOperationDetails()` shows formatted installation/removal details
- **Download Size Tracking**: Integrated file size display in verbose output
- **Status Reporting**: Unified status report display for operations
- **Clean API**: Interface-based design prevents circular imports
- **Verbose Output Spacing**: Verbose sections use conditional `fmt.Println()` pattern (only add blank line if verbose was shown) per spacing framework guidelines

**Status**: ✅ Active - Output system

### `internal/functions/`
**Purpose**: Utility functions
**Files**:
- `sort.go`: Source sorting utilities
- `validation.go`: Validation utilities

**Status**: ✅ Active - Utility functions

### `internal/logging/`
**Purpose**: File logging system
**Files**:
- `logger.go`: Logger implementation with file rotation and level management
  - `New()`: Creates logger with default OS-specific log directory
  - `NewWithPath()`: Creates logger with custom log file path (used for config.yaml LogPath)
- `config.go`: Logging configuration

**Key Features**:
- **File-based logging**: All logs written to `fontget.log` in platform-specific log directory OR custom path from config
- **Config Integration**: LogPath, MaxSize, and MaxFiles from `config.yaml` are connected
  - LogPath supports `$home` variable expansion (e.g., `$home/.fontget/logs/fontget.log`)
  - MaxSize parses string format (e.g., "10MB") to integer megabytes
  - MaxFiles controls number of rotated log files to keep
- **Log rotation**: Automatic rotation based on size, age, and backup count
- **Level management**: Log levels (ErrorLevel, InfoLevel, DebugLevel) controlled by verbose/debug flags
- **Always active**: GetLogger() calls should always log to file regardless of verbose/debug flags
  - Logger level is controlled by config (ErrorLevel by default, InfoLevel with --verbose, DebugLevel with --debug)
  - GetLogger() calls should NOT be conditional on `IsVerbose()` or `IsDebug()`
  - Logger writes to file, not console (console output is handled by verbose/debug output system)

**Usage Pattern**:
- All commands should log: operation start, parameters, errors, and completion
- Use `GetLogger().Info()` for operations and parameters
- Use `GetLogger().Error()` for all error cases
- Use `GetLogger().Warn()` for warnings
- Use `GetLogger().Debug()` for detailed debugging information

**Status**: ✅ Active - Logging system with config.yaml integration

### `internal/sources/`
**Purpose**: Source definitions
**Files**:
- `urls.go`: Source URLs and configuration

**Status**: ✅ Active - Source definitions

### `internal/version/`
**Purpose**: Version management
**Files**:
- `version.go`: Version information

**Status**: ✅ Active - Version management

### `internal/update/`
**Purpose**: Self-update system
**Files**:
- `update.go`: Update implementation (UpdateToLatest, UpdateToVersion)
- `check.go`: Update checking logic (CheckForUpdates, PerformStartupCheck)
- `config.go`: Update configuration types (UpdateConfig)

**Key Features**:
- **Auto-check on startup**: Checks config.yaml `AutoCheck` and `CheckInterval` settings
- **Auto-update**: When `AutoUpdate: true` and update is available, automatically installs in background
- **UTC timestamps**: `LastChecked` uses UTC timezone for consistency across timezones
- **Non-blocking**: Startup checks run in goroutine to avoid blocking application startup
- **Error handling**: Graceful fallback if update check fails (silent failure during startup)

**Status**: ✅ Active - Self-update system with config.yaml integration

### `internal/templates/`
**Purpose**: Code templates
**Files**:
- `command_template.go`: Command template for new commands

**Status**: ✅ Active - Development templates

### `internal/components/`
**Purpose**: Reusable UI components
**Files**:
- `progress_bar.go`: Unified progress bar component with inline display and gradient rendering
- `card.go`: Card components with integrated titles and flexible padding
- `form.go`: Form input components for TUI interfaces
- `confirm.go`: Confirmation dialog components

**Key Features**:
- **Unified Progress Bar**: Single component for all progress displays with inline title integration
  - Inline progress bar with gradient color interpolation
  - Compact single-line display (title + item count + progress bar)
  - Manual gradient rendering using lipgloss for accurate color display
  - Supports verbose/debug mode with title suppression
- **Card System**: Modern card components with integrated titles in borders, configurable padding (vertical/horizontal), and consistent styling
- **Form Components**: Reusable form elements for interactive TUI interfaces
- **Confirmation Dialogs**: Standardized confirmation prompts with consistent styling

**Usage Examples**:
- **Add/Remove Commands**: Uses progress bar for font installation/removal progress
- **Info Command**: Uses card components for displaying font details, license info, and metadata
- **Sources Management**: Uses form and confirmation components for interactive source editing
- **Update Operations**: Uses progress components for showing update progress

**Status**: ✅ Active - UI components

### `internal/license/`
**Purpose**: License management
**Files**:
- `license.go`: License information

**Status**: ✅ Active - License management

### `internal/onboarding/`
**Purpose**: First-run onboarding and setup wizard
**Files**:
- `onboarding.go`: Core onboarding flow management
  - `RunFirstRunOnboarding()`: Executes onboarding on first run
  - `RunWizard()`: Executes onboarding wizard regardless of first-run status (for `--wizard` flag)
  - `OnboardingFlow`: Step-based execution system
- `enhanced_flow.go`: Enhanced interactive TUI onboarding flow
  - **EnhancedOnboardingModel**: Bubble Tea model for interactive onboarding
  - **Step System**: Modular step interface for easy extension
  - **Steps**: Welcome → License Agreement → Wizard Choice → Sources → Settings → Theme Selection → Completion
  - **Theme Integration**: Full theme picker TUI integrated into onboarding flow
  - **Conditional Navigation**: Skips customization steps if user chooses "Let it ride"

**Key Features**:
- **Interactive TUI**: Full-screen interactive terminal UI using Bubble Tea
- **Welcome Screen**: First-time user welcome message
- **License Agreement**: Text-based license acceptance (continuing implies agreement)
- **Wizard Choice**: User can choose to customize settings or accept defaults ("Let it ride")
- **Source Selection**: Interactive source enable/disable with checkbox list
- **Settings Configuration**: Update settings (auto-check, auto-update, popularity sort)
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
- Uses `internal/ui` for TUI components and styling
- Uses `internal/components` for reusable UI components (CheckboxList, ButtonGroup)
- Uses Bubble Tea for TUI framework

**Status**: ✅ Active - Enhanced onboarding system with interactive TUI


---

## Documentation Files

### `README.md`
**Purpose**: Main project documentation
**Status**: ✅ Active - Project documentation

### `docs/help.md`
**Purpose**: Command reference documentation
**Status**: ✅ Active - User documentation

### `refactor.md`
**Purpose**: Refactoring plans and documentation
**Status**: ✅ Active - Development documentation

### `docs/`
**Purpose**: User-facing documentation
**Files**:
- `installation.md`: Installation guide
- `terminal-setup.md`: Terminal setup instructions (includes shell completions)
- `contributing.md`: Contributing guidelines

**Status**: ✅ Active - User documentation

### `docs/development/`
**Purpose**: Development documentation and guidelines
**Files**:
- `codebase.md`: This file - comprehensive codebase overview
- `style-guide.md`: Code style guidelines

**Status**: ✅ Active - Development documentation

### `docs/development/guidelines/`
**Purpose**: Development guidelines and best practices
**Files**:
- `codebase-layout-guidelines.md`: Codebase organization guidelines
- `logging-guidelines.md`: Logging best practices
- `spacing-guidelines.md`: Output spacing guidelines
- `verbose-debug-guidelines.md`: Verbose and debug output guidelines
- `versioning-guide.md`: Versioning and release guidelines

**Status**: ✅ Active - Development guidelines

### `docs/maintenance/documentation-sync.md`
**Purpose**: Documentation synchronization
**Status**: ✅ Active - Documentation management

---

## Configuration Files

### `sources/`
**Purpose**: Source data storage
**Files**:
- `google-fonts.json`: Google Fonts data

**Status**: ✅ Active - Source data

---

## Legacy/Deprecated Files

### Files to Review for Potential Cleanup:


### Template Files:

1. **`internal/templates/command_template.go`** - Command template
   - **Purpose**: Template for creating new commands
   - **Status**: ✅ Active template
   - **Usage**: Reference for developers adding new commands
   - **Features**: Includes verbose/debug scaffolding, error handling patterns, and best practices

---

## Recent Architectural Changes

### Major Refactoring (2025-11-28)


#### **Sources Architecture Overhaul**
- **Old System**: `sources.json` with `SourcesConfig` struct
- **New System**: `manifest.json` with `Manifest` struct
- **Benefits**: 
  - Self-healing manifest system
  - Priority-based source ordering
  - Cleaner separation of concerns
  - Better error handling

#### **Output System Redesign**
- **Old System**: Direct `IsVerbose()` and `IsDebug()` functions
- **New System**: Clean interface with `output.GetVerbose().Info()` and `output.GetDebug().Message()`
- **Benefits**:
  - Better separation of concerns
  - Cleaner API design
  - Easier testing and maintenance

#### **Configuration Consolidation**
- **Renamed**: `yaml_config.go` → `app_config.go`
- **Consolidated**: Directory structure and configuration management
- **Added**: Version management system with `internal/version/`

#### **Priority System Implementation**
- **Built-in sources**: Priority 1, 2, 3 (Google Fonts, Nerd Fonts, Font Squirrel)
- **Custom sources**: Priority 100+ (processed after built-in sources)
- **Benefits**: Predictable source processing order

#### **UI Component System (2025-01-02)**
- **Card Components**: Redesigned with integrated titles in borders, flexible padding controls, and consistent styling
- **Form Components**: Extracted reusable form elements from TUI interfaces
- **Confirmation Dialogs**: Standardized confirmation prompts with consistent styling
- **Hierarchical Lists**: Tree-like display components for structured data
- **Benefits**:
  - Consistent UI/UX across all commands
  - Reusable components reduce code duplication
  - Better maintainability and styling control
  - Enhanced visual hierarchy and user experience

#### **Codebase Organization Refactoring**
- **Eliminated `cmd/shared.go`**: Removed anti-pattern of mixing CLI-specific and general-purpose code
- **Created `internal/cmdutils/`**: CLI-specific utilities (initialization, Cobra integration, CLI wrappers)
- **Created `internal/shared/`**: General-purpose utilities (font formatting, file operations, error types)
- **Moved Table Functions**: All table formatting moved to `internal/ui/tables.go` for unified API
- **Moved Status Report**: Status reporting moved to `internal/output/status.go`
- **Moved Scope Detection**: `AutoDetectScope` moved to `internal/platform/scope.go`
- **Cleanup**: Removed unused functions and files (testutil, inferior functions, etc.)
- **Benefits**:
  - Clear separation between CLI-specific and general-purpose code
  - Better testability and reusability
  - Consistent package organization following Go best practices
  - Easier to understand where code belongs

#### **Font Matching and Font ID Support**
- **Font Matching Feature**: Added `internal/repo/font_matches.go` with optimized index-based matching
  - Matches installed fonts to repository entries to display Font IDs, License, Categories, and Source
  - Uses O(1) lookup index instead of O(n) iteration for better performance
  - Handles Nerd Fonts naming conventions and variant suffixes (NL, Propo, etc.)
  - Protects critical system fonts from being matched
- **Font ID Support**: Commands now accept both font names and Font IDs
  - `add` command: Supports Font IDs (e.g., "google.roboto") in addition to font names
  - `remove` command: Supports Font IDs for accurate font removal
  - `list` command: Displays Font IDs for matched fonts
- **List Command Enhancements**:
  - Default scope changed to "all" (shows fonts from both user and machine scopes)
  - New columns: Font ID, License, Categories, Source (replaced "Installed" date)
  - Removed "all" option from --scope flag (now default behavior)
  - Font ID filtering support: Query parameter can match by Font ID (e.g., "google.roboto") in addition to family name
- **Remove Command Enhancements**:
  - Auto-detects scope based on elevation (admin defaults to "all", user defaults to "user")
  - Shows separate progress entries for each scope when removing from "all" scopes
  - Removed `--force` flag (was not functional)
- **Benefits**:
  - Better user experience with Font ID support
  - More informative font listings with license and source information
  - Improved performance with optimized matching algorithm
  - Cleaner command interface with sensible defaults

#### **Pre-Installation Font Checking**
- **Already-Installed Detection**: Added `checkFontsAlreadyInstalled()` function in `cmd/add.go`
  - Checks if fonts are already installed before downloading to save bandwidth and time
  - Uses the same matching logic as the `list` command for consistency
  - Matches by Font ID (most accurate) with family name fallback
  - Respects installation scope (user or machine) for accurate checking
- **Integration**:
  - `add` command: Checks fonts before downloading during installation
  - `import` command: Checks fonts before downloading during import
  - Fonts still appear in UI/progress bar but skip download if already installed
  - Skips are tracked and displayed in status reports
- **Benefits**:
  - Faster installations by skipping unnecessary downloads
  - Bandwidth savings for already-installed fonts
  - Consistent matching logic across commands
  - Accurate detection using Font ID and family name matching

#### **List Command Optimizations and Font ID Filtering**
- **Font ID Filtering Support**: List command now supports filtering by Font ID in addition to family name
  - Query parameter can match either font family names (e.g., "Roboto") or Font IDs (e.g., "google.roboto")
  - Repository matching happens before filtering to make Font IDs available for filtering
  - Filter checks both family name and Font ID with case-insensitive substring matching
- **Performance Optimizations**:
  - **Early Type Filtering**: Modified `collectFonts()` to filter by file extension before expensive metadata extraction
    - When `--type` filter is specified, files are filtered by extension before calling `platform.ExtractFontMetadata()`
    - Significantly reduces processing time when filtering by type
    - Skips metadata extraction for non-matching fonts
  - **Cached Lowercased Strings**: Pre-computes and caches lowercased strings in filtering loop
    - Avoids repeated `ToLower()` calls for family names and Font IDs
    - Reduces string allocations and improves filtering performance
- **Flag Improvements**:
  - Renamed `--full` flag to `--expand` with `-x` alias for better clarity
  - Updated help text and examples to reflect new flag name
- **Benefits**:
  - More flexible filtering with Font ID support
  - Improved performance, especially when using type filters
  - Better user experience with clearer flag naming

#### **File Logging System Review and Fixes**
- **Comprehensive Logger Review**: Reviewed and fixed GetLogger() usage across all commands
  - **Principle Established**: GetLogger() should ALWAYS log to file, regardless of verbose/debug flags
    - Logger level is controlled by config (ErrorLevel/InfoLevel/DebugLevel based on flags)
    - GetLogger() calls should NOT be conditional on `IsVerbose()` or `IsDebug()`
    - Logger writes to file (`fontget.log`), not console (console output handled by verbose/debug system)
  - **Critical Fixes**:
    - **import.go**: Removed `if IsDebug()` wrapper from GetLogger() calls (was preventing logging in normal/verbose mode)
    - **add.go**: Uncommented and activated GetLogger() calls that were previously commented out
  - **Added Comprehensive Logging**:
    - **list.go**: Added operation start, parameters, errors, and completion logging
    - **export.go**: Added operation start, parameters, errors, and completion logging
    - **backup.go**: Added operation start, parameters, errors, and completion logging
  - **Enhanced Logging**:
    - **search.go**: Added parameter logging and error logging
    - **info.go**: Added parameter logging and error logging
  - **Standard Pattern**: All commands now follow consistent logging pattern:
    - Operation start: `GetLogger().Info("Starting [operation] operation")`
    - Parameters: `GetLogger().Info("Parameters - ...")`
    - Errors: `GetLogger().Error("Failed to ...: %v", err)`
    - Completion: `GetLogger().Info("Operation complete - ...")`
- **Benefits**:
  - Complete audit trail in log files for all operations
  - Consistent logging across all commands
  - Proper separation between file logging (GetLogger) and console output (verbose/debug)
  - All operations logged regardless of user's flag choices

#### **Debug Output Standardization and Error Handling Improvements**
- **Debug Output Alignment**: Standardized debug output patterns across `add`, `remove`, and `import` commands
  - **Function Call Traces**: Debug output now shows function calls (e.g., "Calling installFont(...)", "Calling removeFont(...)")
  - **Internal State Values**: Debug output displays internal state values (e.g., "Found X files", "Resolved font name: ...", "Total fonts: X")
  - **Variant Categorization**: Separate debug messages for different variant categories (Installed/Removed/Skipped/Failed) with consistent formatting
  - **Consistent Spacing**: Standardized spacing in debug variant lists (1 space before hyphen: " - fontname")
  - **Font Not Found Messages**: Debug output shows "font not found" messages in console when fonts cannot be located
- **Args Validation Standardization**: All commands now use Pattern 1 (error + hint) for argument validation
  - Commands updated: `add`, `remove`, `import`, `info`
  - Concise error messages with help hints instead of full help display
  - Improved user experience for simple validation errors
- **SilenceUsage Configuration**: All commands now have `SilenceUsage: true` set
  - Prevents full help display on validation errors
  - Ensures consistent error handling behavior across all commands
- **Verbose Output Spacing**: Updated verbose output spacing pattern
  - Replaced `EndSection()` method with conditional `fmt.Println()` pattern
  - Blank lines added only if verbose output was actually shown: `if IsVerbose() { fmt.Println() }`
  - Prevents unnecessary blank lines when verbose mode is disabled
  - Documented in spacing framework and logging guidelines
- **SuppressVerbose Pattern**: Added `suppressVerbose` parameter to `collectFonts()` function
  - Allows internal/helper functions to suppress verbose output when calling `collectFonts`
  - Used by `checkFontsAlreadyInstalled`, `backup.go`, and `export.go` to avoid technical verbose messages in non-list contexts
  - `list` command continues to show verbose output as scanning is its primary operation
- **Benefits**:
  - Consistent debug output across all commands for easier troubleshooting
  - Better user experience with standardized error messages
  - Cleaner verbose output without unnecessary blank lines
  - More appropriate verbose output context (suppressed for internal operations)

#### **Code Quality Polish (2025-01-XX)**
- **Variable Naming Standardization**: Standardized variable naming for consistency
  - Changed `fm` to `fontManager` in `cmd/export.go` and all references
  - Improved code readability and self-documentation
- **File Structure Standardization**: Reorganized command files to follow Go best practices
  - Standardized structure: imports → types → constants → command → helpers
  - Fixed import grouping (stdlib, internal, third-party)
  - Reorganized `cmd/list.go` to follow standard structure
- **Function Documentation**: Added comprehensive godoc comments to key functions
  - **cmd/add.go**: `installFont`, `installFontsInDebugMode`, `checkFontsAlreadyInstalled`
  - **cmd/remove.go**: `removeFont`
  - **cmd/export.go**: `filterFontsForExport`, `buildExportManifest`
  - **cmd/backup.go**: `organizeFontsBySourceAndFamily`, `createBackupZipArchive`
  - **internal/shared/font.go**: `GetDisplayNameFromFilename`
  - **internal/shared/matching.go**: `FindSimilarFonts`
  - **internal/cmdutils/args.go**: `ParseFontNames`
  - **internal/cmdutils/cobra.go**: `CheckElevation`
- **Inline Comments**: Added explanatory comments for complex logic
  - Enhanced comments in `internal/shared/font.go` for camelCase conversion threshold logic
  - Added comments in `cmd/export.go` for font grouping logic
  - Added comments in `cmd/add.go` for early installation checks
- **Code Organization Improvements**: Refactored for better maintainability
  - **sources_manage.go**: Extracted state routing logic into `routeStateUpdate` helper function
  - Improved separation of concerns and code clarity
- **Benefits**:
  - Better code maintainability and readability
  - Improved developer experience with comprehensive documentation
  - Consistent code structure across all command files
  - Self-documenting code with clear function purposes

#### **Config Migration System Simplification (2025-01-XX)**
- **Version Tracking**: Added `ConfigVersion` field to `AppConfig` structure for schema versioning
  - Current version: "2.0" (baseline schema)
  - Backward compatibility: Old configs without version default to "1.0"
- **Simplified Migration System**: Replaced complex rule-based system with simple merge-based approach
  - **Field Mapping**: Simple maps for field renames (`fieldRenameMap`) and field moves (`fieldMoveMap`)
  - **Automatic Handling**: `handleLegacyFieldMapping()` transforms old configs before unmarshaling
  - **Merge Logic**: Old config merged with new defaults via `mergeConfigValues()` reflection-based merge
  - **Update Integration**: `MigrateConfigAfterUpdate()` called automatically after binary updates
  - **Consolidation**: Migration logic consolidated into `user_preferences.go` (removed `migrations.go`)
- **Migration Process**: 
  - Field renames/moves applied automatically during config loading
  - User values preserved, new defaults applied for missing fields
  - Version updated to current after successful merge
- **Benefits**:
  - Simpler, more maintainable codebase
  - No complex migration rules - just add entries to maps
  - Automatic migration during normal operations and after updates
  - All config logic in one place for easier maintenance

#### **Enhanced Onboarding Flow (2025-01-XX)**
- **Redesigned Flow**: Complete redesign of first-run onboarding experience
  - **Welcome Screen**: First-time user welcome message
  - **License Agreement**: Text-based license acceptance (continuing implies agreement)
  - **Wizard Choice**: User can choose to customize settings or accept defaults ("Let it ride")
  - **Conditional Navigation**: Skips customization steps if user chooses "Let it ride"
  - **Theme Integration**: Full theme picker TUI integrated into onboarding flow (only shown if customizing)
- **Interactive TUI**: Enhanced onboarding uses full-screen interactive terminal UI
  - Uses Bubble Tea framework for consistent TUI experience
  - Step-based navigation with back/forward support
  - Modular step interface for easy extension
- **Wizard Flag**: Added `--wizard` flag to root command for re-running onboarding
  - Useful for reconfiguration and testing
  - Runs regardless of first-run status
- **Step System**: Modular step interface (`OnboardingStepInterface`)
  - Each step implements `Name()`, `View()`, `Update()`, `CanGoBack()`, `CanGoNext()`
  - Easy to add new steps or modify existing ones
- **State Management**: Tracks all user selections and saves to config on completion
  - Source selections, settings values, and theme selection are preserved
- **Benefits**:
  - Better first-run experience with clear welcome and guidance
  - Flexible customization path (customize or accept defaults)
  - Integrated theme selection for immediate personalization
  - Re-runnable wizard for easy reconfiguration

#### **Search Command Enhancements (2025-01-XX)**
- **Source-Only Search**: Added support for searching by source alone
  - Example: `fontget search -s google` shows all fonts from Google Fonts
  - Fetches all fonts and filters by source when only `--source` flag is provided
- **Result Limiting**: Added configurable result limiting via `config.yaml`
  - `Search.ResultLimit` field (0 = unlimited, default)
  - Applied after source filtering for accurate limiting
- **Source Filtering Enhancement**: Improved source validation and filtering
  - Supports source ID, full name, and short prefix (e.g., "google", "nerd", "squirrel")
  - `validateSource()` checks all three formats for flexible filtering
- **Code Refactoring**: Improved code quality and maintainability
  - Extracted constants for magic numbers (`SearchTableColumnSpacing`, `SearchPopularityDivisor`, etc.)
  - Extracted `logDebugScoreBreakdown()` helper function for debug output
  - Extracted shared completion functions for category and source flags
  - Removed code duplication and improved code organization
- **Debug Scoring**: Enhanced debug output with detailed score breakdown
  - Shows base score, match bonus, popularity bonus, and final score
  - Displays position in results and match type
- **Benefits**:
  - More flexible search options (source-only search)
  - Better performance with configurable result limiting
  - Improved code maintainability and readability
  - Enhanced debugging capabilities with detailed scoring

#### **Help Template Customization (2025-01-XX)**
- **Custom Help Templates**: Added custom help and usage templates in `cmd/root.go`
  - **Section Ordering**: Controlled section order (Usage → Description → Commands → Options → Examples)
  - **Title Changes**: Changed "Flags:" to "Options:" for better clarity
  - **Spacing Control**: Careful spacing management to prevent double newlines
  - **Consistent Formatting**: Applied to both root command and subcommands
- **Benefits**:
  - Better help output organization and readability
  - Consistent formatting across all commands
  - Professional appearance with proper spacing