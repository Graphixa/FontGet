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
- Sets up global flags (`--verbose`, `--debug`)
- Initializes logging configuration
- Registers all subcommands

**Interfaces**:
- Imports all command packages
- Uses `internal/logging` for log configuration
- Uses `internal/output` for verbose/debug functionality

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

**Key Functions**:
- `searchCmd.RunE`: Main search execution
- `performSearch`: Core search logic
- `displaySearchResults`: Result formatting

**Interfaces**:
- Uses `internal/repo` for font data access
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

**Key Functions**:
- `listCmd.RunE`: Main listing execution
- `collectFonts`: Collects fonts from specified scopes
- `IsCriticalSystemFont`: Checks if a font is a protected system font

**Interfaces**:
- Uses `internal/platform` for OS-specific font detection
- Uses `internal/output` for verbose/debug output
- Uses `internal/repo` for font matching and repository access
- Uses `cmd/shared` for protected font checking

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
- Uses `cmd/shared` for protected font checking and status reporting

**Status**: ✅ Active - Core functionality

### `export.go`
**Purpose**: Font export command
**Functionality**:
- Exports installed fonts to a JSON manifest file
- Matches installed fonts to repository entries to include Font IDs, License, Categories, and Source
- Supports filtering by match string, source, or export all fonts
- System fonts are always excluded from exports
- Supports output to directory (creates fonts-export.json) or specific file path via -o flag
- Uses pin spinner for progress feedback during export
- Provides verbose/debug output following logging guidelines
- **Nerd Fonts Support**: Groups families by Font ID to handle cases where one Font ID installs multiple families (e.g., ZedMono installs ZedMono, ZedMono Mono, and ZedMono Propo)

**Key Functions**:
- `exportCmd.RunE`: Main export execution
- `performFullExportWithResult`: Complete export process with result tracking (groups by Font ID)
- `performFullExport`: Export process for debug mode
- `collectFonts`: Collects fonts from specified scopes (reused from list.go)
- `groupByFamily`: Groups fonts by family name (reused from list.go)

**Key Features**:
- **Directory Support**: `-o` flag accepts directories (creates default filename) or file paths (winget-style)
- **Font Matching**: Uses optimized index-based matching to repository entries
- **Filtering**: Supports `--match`, `--source`, `--all`, and `--matched` flags
- **Export Manifest**: JSON structure with metadata, font details, and variants
- **Nerd Fonts Handling**: Groups multiple families under one Font ID entry with `family_names` array

**Interfaces**:
- Uses `internal/platform` for OS-specific font detection
- Uses `internal/output` for verbose/debug output
- Uses `internal/repo` for font matching and repository access
- Uses `internal/ui` for spinner components
- Uses `cmd/shared` for protected font checking

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

**Key Functions**:
- `NewSourcesModel`: TUI model initialization
- `addSource`: Adding new sources
- `updateSource`: Editing existing sources
- `saveChanges`: Persisting changes to manifest

**Interfaces**:
- Uses `internal/config` for manifest operations
- Uses `internal/functions` for source utilities
- Uses `internal/ui` for TUI components
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


### `shared.go`
**Purpose**: Shared command utilities
**Functionality**:
- Provides common utilities used across commands
- Handles shared functionality and helpers
- Font name parsing and formatting utilities
- File size formatting utilities
- Table formatting constants and helpers
- Error types for font operations
- Protected system font checking
- Font installation status checking

**Key Functions**:
- `ParseFontNames`: Parses comma-separated font names from arguments
- `FormatFontNameWithVariant`: Formats font names with variants
- `GetFontDisplayName`: Returns human-friendly display names
- `GetFontFamilyNameFromFilename`: Extracts font family name from filename
- `FormatFileSize`: Formats bytes into human-readable format (KB, MB)
- `findSimilarFonts`: Fuzzy matching for font names
- `PrintStatusReport`: Prints formatted status reports
- `IsCriticalSystemFont`: Checks if a font is a protected system font (used by list and remove commands)
- `checkFontsAlreadyInstalled`: Checks if a font is already installed in the specified scope using the same matching logic as the list command (matches by Font ID and family name)

**Interfaces**:
- Used by multiple command files
- Uses `internal/ui` for styling
- Uses `internal/platform` for platform operations

**Status**: ✅ Active - Utility functions

---

## Internal Packages

### `internal/config/`
**Purpose**: Configuration management
**Files**:
- `user_preferences.go`: User preferences configuration (renamed from `app_config.go`)
- `app_state.go`: Core application state types and functions
- `manifest.go`: Font sources manifest management
- `validation.go`: Configuration validation

**Status**: ✅ Active - Core configuration system

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
- `testutil/`: Test utilities for platform operations

**Key Features**:
- **Font Metadata Extraction**: `ExtractFontMetadata()` reads font family name, style name, and full name directly from font file SFNT name table
- **Cross-platform Font Management**: Unified interface for font installation/removal across Windows, macOS, and Linux
- **Elevation Detection**: Platform-specific privilege checking
- **Font Directory Management**: Scope-aware font directory resolution

**Status**: ✅ Active - Cross-platform support

### `internal/ui/`
**Purpose**: User interface components
**Files**:
- `components.go`: UI component definitions and utilities
  - `RunSpinner`: Pin spinner wrapper with progress feedback
  - `hexToPinColor`: Converts hex colors to pin package color constants
  - Various rendering utilities for titles, errors, success messages
- `styles.go`: Centralized styling and theming
  - Catppuccin Mocha color palette
  - Page structure styles (titles, subtitles, content)
  - User feedback styles (info, warning, error, success)
  - Data display styles (tables, lists)
  - Form styles (labels, inputs, placeholders)
  - Command styles (keys, labels, examples)
  - Card styles (titles, labels, content, borders)
  - Progress bar styles (headers, text, containers)
  - **Spinner styles**: Color constants and pin package color mapping (`SpinnerColor`, `SpinnerDoneColor`, `PinColorMap`)

**Key Features**:
- **Centralized Styling**: All colors and styles defined in `styles.go` for consistency
- **Spinner Integration**: Pin spinner colors mapped from hex to pin package constants
- **Color Mapping**: `PinColorMap` provides hex-to-pin color conversion for spinner components

**Status**: ✅ Active - UI system

### `internal/output/`
**Purpose**: Output management
**Files**:
- `verbose.go`: Verbose output handling with operation details display
- `debug.go`: Debug output handling

**Key Features**:
- **Consistent Formatting**: Standardized `[INFO]`, `[WARNING]`, `[ERROR]` prefixes
- **Operation Details Display**: `DisplayFontOperationDetails()` shows formatted installation/removal details
- **Download Size Tracking**: Integrated file size display in verbose output
- **Clean API**: Interface-based design prevents circular imports

**Status**: ✅ Active - Output system

### `internal/functions/`
**Purpose**: Utility functions
**Files**:
- `sort.go`: Source sorting utilities
- `validation.go`: Validation utilities

**Status**: ✅ Active - Utility functions

### `internal/logging/`
**Purpose**: Logging system
**Files**:
- `logger.go`: Logger implementation
- `config.go`: Logging configuration

**Status**: ✅ Active - Logging system

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
- `hierarchy.go`: Hierarchical list components for structured data display

**Key Features**:
- **Unified Progress Bar**: Single component for all progress displays with inline title integration
  - Inline progress bar with gradient color interpolation
  - Compact single-line display (title + item count + progress bar)
  - Manual gradient rendering using lipgloss for accurate color display
  - Supports verbose/debug mode with title suppression
- **Card System**: Modern card components with integrated titles in borders, configurable padding (vertical/horizontal), and consistent styling
- **Form Components**: Reusable form elements for interactive TUI interfaces
- **Confirmation Dialogs**: Standardized confirmation prompts with consistent styling
- **Hierarchical Lists**: Tree-like display components for structured data (e.g., font families with variants)

**Usage Examples**:
- **Add/Remove Commands**: Uses progress bar for font installation/removal progress
- **Info Command**: Uses card components for displaying font details, license info, and metadata
- **Sources Management**: Uses form and confirmation components for interactive source editing
- **List Command**: Uses hierarchy components for displaying font families with variants
- **Update Operations**: Uses progress components for showing update progress

**Status**: ✅ Active - UI components

### `internal/license/`
**Purpose**: License management
**Files**:
- `license.go`: License information

**Status**: ✅ Active - License management


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
**Purpose**: Additional documentation
**Files**:
- `shell-completions.md`: Shell completion setup
- `STYLE_GUIDE.md`: Code style guidelines
- `terminal-setup.md`: Terminal setup instructions

**Status**: ✅ Active - Documentation

### `docs/maintenance/documentation-sync.md`
**Purpose**: Documentation synchronization
**Status**: ✅ Active - Documentation management

---

## Configuration Files

### `Makefile`
**Purpose**: Build automation
**Functionality**:
- Build commands
- Version injection
- Development utilities

**Status**: ✅ Active - Build system

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

### Major Refactoring (2025-09-30)

The codebase underwent a significant refactoring to implement a new manifest-based sources system:

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

#### **Font Matching and Font ID Support (2025-01-XX)**
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
- **Remove Command Enhancements**:
  - Auto-detects scope based on elevation (admin defaults to "all", user defaults to "user")
  - Shows separate progress entries for each scope when removing from "all" scopes
  - Removed `--force` flag (was not functional)
- **Benefits**:
  - Better user experience with Font ID support
  - More informative font listings with license and source information
  - Improved performance with optimized matching algorithm
  - Cleaner command interface with sensible defaults

#### **Pre-Installation Font Checking (2025-01-XX)**
- **Already-Installed Detection**: Added `checkFontsAlreadyInstalled()` function in `cmd/shared.go`
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

---

## Summary

The FontGet codebase is well-structured with clear separation of concerns:

- **Commands** (`cmd/`): CLI command implementations
- **Internal packages**: Core functionality and utilities
- **Configuration**: Centralized config and manifest management
- **Platform support**: Cross-platform compatibility
- **UI/UX**: User interface and output management

The codebase has undergone recent refactoring to implement a new manifest system, replacing the old sources configuration approach. Most files are active and well-maintained, with only a few empty directories that may need cleanup.

### Key Strengths:
- ✅ Clean architecture with clear separation of concerns
- ✅ Comprehensive cross-platform support
- ✅ Modern CLI framework (Cobra) with TUI support (Bubble Tea)
- ✅ Well-documented and maintained
- ✅ Recent refactoring improved maintainability

### Areas for Improvement:
- 🔄 Consider adding more comprehensive tests

**Overall Status**: ✅ Healthy - Well-structured, actively maintained codebase with recent architectural improvements
