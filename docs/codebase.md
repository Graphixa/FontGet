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
  - Lipgloss (styling)
  - Color (terminal colors)
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
- Handles font search and fuzzy matching
- Manages font installation with progress tracking
- Supports different installation scopes (user/system)
- Provides detailed error handling and suggestions

**Key Functions**:
- `addCmd.RunE`: Main command execution
- `installFont`: Core installation logic
- `getSourceDisplayName`: Source name resolution
- `showFontSuggestions`: Error handling with suggestions

**Interfaces**:
- Uses `internal/repo` for font data
- Uses `internal/platform` for OS-specific operations
- Uses `internal/output` for verbose/debug output
- Uses `internal/ui` for user interface

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
- Provides filtering and formatting options
- Shows font details and metadata

**Key Functions**:
- `listCmd.RunE`: Main listing execution
- `listInstalledFonts`: Core listing logic

**Interfaces**:
- Uses `internal/platform` for OS-specific font detection
- Uses `internal/output` for verbose/debug output

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
- Handles different removal scopes
- Provides confirmation prompts

**Key Functions**:
- `removeCmd.RunE`: Main removal execution
- `removeFont`: Core removal logic

**Interfaces**:
- Uses `internal/platform` for OS-specific operations
- Uses `internal/output` for verbose/debug output

**Status**: ✅ Active - Core functionality

### `sources.go`
**Purpose**: Sources management command
**Functionality**:
- Manages font sources (Google Fonts, Nerd Fonts, Font Squirrel)
- Provides subcommands for info, update, and management
- Handles source configuration and updates

**Key Functions**:
- `sourcesCmd`: Main sources command
- `sourcesInfoCmd`: Source information display
- `sourcesUpdateCmd`: Source update functionality
- `runSourcesUpdateVerbose`: Verbose update mode

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

**Key Functions**:
- `configCmd`: Main configuration command
- `showConfig`: Display current configuration
- `resetConfig`: Reset configuration to defaults

**Interfaces**:
- Uses `internal/config` for configuration operations
- Uses `internal/output` for verbose/debug output

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


### `shared.go`
**Purpose**: Shared command utilities
**Functionality**:
- Provides common utilities used across commands
- Handles shared functionality and helpers

**Key Functions**:
- Various utility functions for command operations

**Interfaces**:
- Used by multiple command files

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
- `font.go`: Font data structures
- `metadata.go`: Font metadata handling
- `archive.go`: Archive operations
- `types.go`: Type definitions

**Status**: ✅ Active - Core repository system

### `internal/platform/`
**Purpose**: Cross-platform operations
**Files**:
- `platform.go`: Platform abstraction
- `windows.go`: Windows-specific operations
- `darwin.go`: macOS-specific operations
- `linux.go`: Linux-specific operations
- `elevation.go`: Privilege elevation
- `temp.go`: Temporary file operations
- `windows_utils.go`: Windows utilities

**Status**: ✅ Active - Cross-platform support

### `internal/ui/`
**Purpose**: User interface components
**Files**:
- `components.go`: UI component definitions
- `styles.go`: Styling and theming

**Status**: ✅ Active - UI system

### `internal/output/`
**Purpose**: Output management
**Files**:
- `verbose.go`: Verbose output handling
- `debug.go`: Debug output handling

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
**Purpose**: UI components
**Files**:
- `progress.go`: Progress bar component

**Status**: ✅ Active - UI components

### `internal/license/`
**Purpose**: License management
**Files**:
- `license.go`: License information

**Status**: ✅ Active - License management

### `internal/errors/`
**Purpose**: Error handling
**Files**: (Empty directory)

**Status**: ⚠️ Empty - May need cleanup

### `internal/index/`
**Purpose**: Indexing
**Files**: (Empty directory)

**Status**: ⚠️ Empty - May need cleanup

---

## Documentation Files

### `README.md`
**Purpose**: Main project documentation
**Status**: ✅ Active - Project documentation

### `command-reference.md`
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

### `documentation-sync.md`
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

1. **`internal/errors/`** - Empty directory, may be unused
   - **Status**: ⚠️ Empty directory
   - **Recommendation**: Removed as it was not needed for future error handling

2. **`internal/index/`** - Empty directory, may be unused
   - **Status**: ⚠️ Empty directory
   - **Recommendation**: Removed as it was not needed for future indexing features

3. **`fontget.exe~`** - Backup executable
   - **Status**: ⚠️ Backup file
   - **Recommendation**: Safe to remove

4. **`scripts/audit-flags.go`** - CLI flag audit utility
   - **Purpose**: Analyzes command files to extract flag information and check documentation sync
   - **Status**: ✅ Utility script (not deprecated)
   - **Recommendation**: Keep as development utility

### Files That Were Recently Refactored:

1. **`internal/config/sources_config.go`** - DELETED (replaced by manifest system)
2. **`internal/output/shared.go`** - DELETED (consolidated into verbose.go and debug.go)
3. **`internal/config/app_config.go`** - DELETED (renamed to user_preferences.go)
4. **`internal/config/config.go`** - DELETED (renamed to app_state.go)

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
- ⚠️ Remove empty directories (`internal/errors/`, `internal/index/`)
- ⚠️ Clean up backup files (`fontget.exe~`)
- 🔄 Consider adding more comprehensive tests

**Overall Status**: ✅ Healthy - Well-structured, actively maintained codebase with recent architectural improvements
