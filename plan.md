# FontGet CLI Tool - Implementation Plan

A command-line tool for installing fonts from multiple repositories via the centralized [FontGet-Sources repository](https://github.com/Graphixa/FontGet-Sources).

## 🎉 Phase 1 COMPLETED - FontGet-Sources Integration

**Major accomplishments:**
- ✅ **FontGet-Sources Integration**: Successfully integrated with external repository
- ✅ **Local Caching System**: Implemented 24-hour auto-refresh caching
- ✅ **Multi-Source Support**: Google Fonts, Nerd Fonts, Font Squirrel
- ✅ **Clean Font Resolution**: Table-based conflict resolution
- ✅ **Enhanced Search**: Rich metadata display with source information
- ✅ **Source Management**: `fontget sources --update` command
- ✅ **Backward Compatibility**: No breaking changes to existing functionality

**Current Status**: Core functionality complete, ready for Phase 2 enhancements.

## Project Purpose

The `fontget` CLI tool queries multiple font repositories for fonts on demand, allowing users to easily install fonts on their system. It supports both user-level and system-wide font installation with support for Google Fonts, Nerd Fonts, Font Squirrel, and custom sources.

## Repository Structure

```
fontget/
├── cmd/
│   ├── root.go      # Root command and initialization
│   ├── add.go       # Font installation command
│   ├── remove.go    # Font removal command
│   ├── list.go      # List installed fonts
│   ├── search.go    # Search available fonts
│   ├── info.go      # Font information command
│   ├── sources.go   # Source management command
│   ├── config.go    # Configuration management command
│   └── shared.go    # Shared utilities
├── internal/
│   ├── platform/    # Platform-specific font management
│   ├── config/      # Configuration management
│   ├── repo/        # Font repository interaction
│   └── logging/     # Logging functionality
├── docs/
├── go.mod
├── go.sum
└── README.md
```

**Note:** Font source data is now managed externally in the [FontGet-Sources repository](https://github.com/Graphixa/FontGet-Sources). The CLI fetches source data from this centralized repository instead of maintaining local source files.

## Local Cache Layout

```
{TEMP_DIR}/Fontget/     # Temporary download directory
└── fonts/              # Downloaded font files (cleaned up after installation)

Where {TEMP_DIR} is:
- Windows: %TEMP% or %TMP%
- Linux: /tmp
- macOS: /var/folders/.../T/
```

## Configuration Structure

### Main Config (`~/.fontget/config.yaml`)
```yaml
Configuration:
  DefaultEditor: "notepad.exe"

Logging:
  LogPath: "$home/.fontget/logs/fontget.log"
  MaxSize: "10MB"
  MaxFiles: 5

Sources:
  Google:
    Path: "https://raw.githubusercontent.com/Graphixa/FontGet-Sources/main/sources/google-fonts.json"
    Prefix: "google"
    Enabled: true
  NerdFonts:
    Path: "https://raw.githubusercontent.com/Graphixa/FontGet-Sources/main/sources/nerd-fonts.json"
    Prefix: "nerd"
    Enabled: true
  FontSquirrel:
    Path: "https://raw.githubusercontent.com/Graphixa/FontGet-Sources/main/sources/font-squirrel.json"
    Prefix: "squirrel"
    Enabled: false
```


## Current Codebase Analysis

### Existing Source Management
- **Current Location**: `internal/repo/sources.go` and `internal/repo/manifest.go`
- **Current URLs**: Point to `https://raw.githubusercontent.com/graphixa/FontGet/main/sources/`
- **Current Structure**: Flat manifest with `sources` map containing `SourceInfo` and `FontInfo`
- **Current Font ID Format**: Uses source prefixes (e.g., `google.roboto`)

### Key Files to Update
- `internal/repo/sources.go` - Update source URLs and constants
- `internal/repo/manifest.go` - Update `GetManifest()` function
- `internal/repo/types.go` - Update struct definitions
- `internal/config/sources_config.go` - Update default source URLs
- `cmd/sources.go` - Update source management commands

### Current Data Structures (to be replaced)
```go
// Current FontManifest structure
type FontManifest struct {
    Version     string                `json:"version"`
    LastUpdated time.Time             `json:"last_updated"`
    Sources     map[string]SourceInfo `json:"sources"`
}

// Current FontInfo structure
type FontInfo struct {
    Name         string            `json:"name"`
    License      string            `json:"license"`
    Variants     []string          `json:"variants"`
    Subsets      []string          `json:"subsets"`
    Version      string            `json:"version"`
    Description  string            `json:"description"`
    LastModified time.Time         `json:"last_modified"`
    MetadataURL  string            `json:"metadata_url"`
    SourceURL    string            `json:"source_url"`
    Categories   []string          `json:"categories,omitempty"`
    Tags         []string          `json:"tags,omitempty"`
    Popularity   int               `json:"popularity,omitempty"`
    Files        map[string]string `json:"files,omitempty"`
}
```

## Implementation Tasks

### Phase 1: FontGet-Sources Integration

#### 1.1 Update Data Structures
- [x] Create new Go structs matching FontGet-Sources schema:
  ```go
  type SourceInfo struct {
      Name        string    `json:"name"`
      Description string    `json:"description"`
      URL         string    `json:"url"`
      Version     string    `json:"version"`
      LastUpdated time.Time `json:"last_updated"`
      TotalFonts  int       `json:"total_fonts"`
  }
  
  type FontVariant struct {
      Name    string            `json:"name"`
      Weight  int               `json:"weight"`
      Style   string            `json:"style"`
      Files   map[string]string `json:"files"`
  }
  
  type Font struct {
      Name          string        `json:"name"`
      Family        string        `json:"family"`
      License       string        `json:"license"`
      LicenseURL    string        `json:"license_url"`
      Designer      string        `json:"designer"`
      Foundry       string        `json:"foundry"`
      Categories    []string      `json:"categories"`
      Tags          []string      `json:"tags"`
      Popularity    int           `json:"popularity"`
      Variants      []FontVariant `json:"variants"`
      UnicodeRanges []string      `json:"unicode_ranges"`
      Languages     []string      `json:"languages"`
      SampleText    string        `json:"sample_text"`
  }
  ```

#### 1.2 Update Source Loading
- [x] **Update `internal/repo/sources.go`**:
  - [x] Change `fontManifestURL` constant to point to FontGet-Sources
  - [x] Update `getSourcesDir()` to handle multiple source files
  - [x] Add functions to load individual source files (google-fonts.json, nerd-fonts.json, font-squirrel.json)
  - [x] Implement source priority ordering (Google → Nerd → Font Squirrel)

- [x] **Update `internal/repo/manifest.go`**:
  - [x] Modify `GetManifest()` to load from multiple FontGet-Sources URLs
  - [x] Add function to merge multiple source files into single manifest
  - [x] Implement JSON schema validation using FontGet-Sources schema
  - [x] Add error handling for network failures with retry logic
  - [x] Cache source data locally with 24-hour refresh per source

- [x] **Update `internal/config/sources_config.go`**:
  - [x] Change default source URLs to FontGet-Sources repository
  - [x] Update `DefaultSourcesConfig()` function with new URLs
  - [x] Add validation for FontGet-Sources URL format

#### 1.3 Font Resolution System
- [x] **Update `internal/repo/font.go`**:
  - [x] Modify `GetFont()` function to search across all sources
  - [x] Implement clean font ID resolution (`roboto` instead of `google.roboto`)
  - [x] Add function to detect font name conflicts across sources
  - [x] When multiple sources have same font name, return all matches for user selection
  - [x] Support explicit source specification: `fontget add google.roboto`
  - [x] Remove source priority logic - let user choose explicitly

- [x] **Update `cmd/add.go`**:
  - [x] Modify font resolution logic to handle new clean IDs
  - [x] Add collision resolution UI that displays table similar to search results:
    ```
    Font 'roboto' found in multiple sources:
    
    Font Name    Font ID           Source
    ----------------------------------------
    Roboto       google.roboto     Google Fonts
    Roboto       squirrel.roboto   Font Squirrel
    
    Select font ID to install: 
    ```
  - [x] Update error messages for font not found scenarios
  - [x] Add support for explicit source specification in command line
  - [x] Use same table formatting as `fontget search` command

#### 1.4 File Type Handling
- [ ] **Update `internal/repo/font.go`**:
  - [ ] Modify `GetFont()` to handle new variant structure with file type keys
  - [ ] Implement file type detection based on source and variant files
  - [ ] **Google Fonts**: Direct TTF/OTF files (no extraction needed)
  - [ ] **Font Squirrel**: ZIP archives (extract before installation)
  - [ ] **Nerd Fonts**: TAR.XZ archives (extract before installation)
  - [ ] Add archive extraction functions for ZIP/TAR.XZ files
  - [ ] Update font file validation to handle different file types

- [ ] **Update `internal/platform/` files**:
  - [ ] Add ZIP extraction functionality for Font Squirrel fonts
  - [ ] Add TAR.XZ extraction functionality for Nerd Fonts
  - [ ] Update font installation logic to handle extracted files
  - [ ] Add cleanup for temporary extracted files

#### 1.5 Update Commands
- [x] **Update `cmd/add.go`**:
  - [x] Modify `addCmd` to handle new font resolution system
  - [x] Add source conflict resolution UI
  - [x] Update progress messages for different file types
  - [x] Add support for explicit source specification

- [x] **Update `cmd/search.go`**:
  - [x] Modify `searchCmd` to search across all FontGet-Sources
  - [x] Add source information to search results in table format:
    ```
    Font Name    Font ID           Source         Category      Popularity
    ---------------------------------------------------------------------
    Roboto       google.roboto     Google Fonts   Sans Serif    95
    Roboto       squirrel.roboto   Font Squirrel  Sans Serif    85
    ```
  - [x] Implement enhanced metadata display (categories, tags, popularity)
  - [x] Add filtering by source option
  - [x] Ensure table formatting is consistent with collision resolution UI

- [ ] **Update `cmd/info.go`**:
  - [ ] Modify `infoCmd` to display enhanced metadata
  - [ ] Add support for clean font IDs
  - [ ] Show source information and variant details
  - [ ] Display unicode ranges, languages, and sample text

- [x] **Update `cmd/sources.go`**:
  - [x] Modify `sourcesCmd` to work with FontGet-Sources
  - [x] Add `fontget sources --update` command (replaced refresh subcommand)
  - [x] Update source listing to show FontGet-Sources status
  - [x] Add source enable/disable functionality
  - [x] Implement local caching with 24-hour auto-refresh
  - [x] Add manual cache refresh functionality

- [ ] **Update `cmd/list.go`**:
  - [ ] Modify `listCmd` to show source information for installed fonts
  - [ ] Add filtering by source option
  - [ ] Update installation tracking format

### Phase 2: Enhanced Features

#### 2.1 Installation Tracking
- [ ] Track installations in `~/.fontget/installations.json`:
  ```json
  {
    "installations": [
      {
        "font_id": "google.roboto",
        "source": "google",
        "installed": "2024-01-15T10:30:00Z",
        "scope": "user"
      }
    ]
  }
  ```

#### 2.2 Enhanced Search & Filtering
- [ ] Search by font name, family, designer, tags
- [ ] Filter by category (Sans Serif, Serif, Display, etc.)
- [ ] Sort by popularity
- [ ] Show source information in results

#### 2.3 Source Management
- [ ] `fontget sources <source> --enable/--disable`
- [ ] `fontget sources list` - Show all available sources and their status
- [ ] `fontget sources refresh` - Force refresh all source data
- [ ] Remove priority reordering commands (no longer needed with explicit selection)

### Phase 3: Polish & Advanced Features

#### 3.1 Interactive UI
- [ ] BIOS-style interactive menu for source management
- [ ] Visual font browsing with categories
- [ ] Real-time configuration preview

#### 3.2 Backup/Restore
- [ ] `fontget export` - Export installation list
- [ ] `fontget import` - Import installation list
- [ ] Cross-platform compatibility

## Key Implementation Details

### Current System (Phase 1-2)
- **Active**: JSON-based configuration in `internal/config/config.go`
- **Storage**: System-wide directories following package manager conventions
  - Linux: `/etc/fontget/` (system configs), `~/.local/share/fontget/` (user data)
  - Windows: `C:\ProgramData\FontGet\` (system configs), `%LOCALAPPDATA%\FontGet\` (user data)
  - macOS: `/etc/fontget/` (system configs), `~/Library/Application Support/FontGet/` (user data)
- **Purpose**: Follows system package manager standards and enterprise deployment patterns

### Future System (Phase 3+)
- **Target**: YAML-based configuration in `internal/config/yaml_config.go`
- **Storage**: Maintains system-wide structure with YAML format
- **Benefits**: Modern configuration format while preserving system integration

### Migration Strategy
1. **Phase 1-2**: Continue using current JSON system while developing YAML system
2. **Phase 3**: Implement YAML system alongside JSON system
3. **Post-Phase 3**: Deprecate JSON system, rename `yaml_config.go` to `config.go`
4. **Final**: Remove old JSON system entirely

### Directory Structure Rationale
**System-wide approach chosen because:**
- FontGet will be distributed through official package managers
- Follows system package manager conventions (like apt, pacman, winget)
- Enables enterprise deployment and centralized management
- System admins can manage FontGet configuration centrally
- Users can still install fonts to their user directories without admin privileges
- Proper separation of system configs (admin-managed) and user data (user-managed)

## Commands

### Core Commands
- `fontget add <font-name>` - Install a font
- `fontget remove <font-name>` - Remove an installed font
- `fontget list` - List installed fonts
- `fontget search <query>` - Search for available fonts
- `fontget info <font-id>` - Show detailed font information

### Source Management
- `fontget sources --update` - Update source URLs to FontGet-Sources and refresh cache
- `fontget sources info` - Show sources information and status
- `fontget sources edit` - Open sources configuration in editor
- `fontget sources --validate` - Validate sources configuration
- `fontget sources --reset-defaults` - Reset sources to defaults

### Configuration
- `fontget config` - Open configuration in editor
- `fontget config --show` - Show configuration file location

## FontGet-Sources Integration ✅ COMPLETED

The FontGet CLI tool now integrates with the external [FontGet-Sources repository](https://github.com/Graphixa/FontGet-Sources) for standardized font data management. This separation of concerns allows:

- **Centralized Data Management**: All font source data is maintained in a dedicated repository
- **Automated Updates**: GitHub Actions workflows automatically update font data daily
- **Standardized Schema**: Consistent JSON schema across all font sources
- **Enhanced Data Quality**: Automated validation and sanitization of font data

### New Source Structure

The FontGet-Sources repository provides:
- **Schema Definition**: [font-source-schema.json](https://raw.githubusercontent.com/Graphixa/FontGet-Sources/main/schemas/font-source-schema.json)
- **Source Files**:
  - [Google Fonts](https://raw.githubusercontent.com/Graphixa/FontGet-Sources/main/sources/google-fonts.json)
  - [Nerd Fonts](https://raw.githubusercontent.com/Graphixa/FontGet-Sources/main/sources/nerd-fonts.json)
  - [Font Squirrel](https://raw.githubusercontent.com/Graphixa/FontGet-Sources/main/sources/font-squirrel.json)

### Key Changes from Previous Format

1. **Nested Structure**: `source_info` + `fonts` separation instead of flat structure
2. **Variant Objects**: Variants are now objects with `name`, `weight`, `style`, and `files` properties
3. **File Type Keys**: Files are organized by type (`ttf`, `otf`, `zip`, `tar.xz`) instead of variant names
4. **Clean Font IDs**: Font IDs are clean (e.g., `roboto`) without source prefixes
5. **Enhanced Metadata**: Additional fields like `unicode_ranges`, `languages`, `sample_text`, `tags`, `popularity`

### File Type Detection
```go
func GetDownloadURL(font Font, variant FontVariant) string {
    // Priority: TTF > OTF > FON
    if ttf, exists := variant.Files["ttf"]; exists {
        return ttf
    }
    if otf, exists := variant.Files["otf"]; exists {
        return otf
    }
    if fon, exists := variant.Files["fon"]; exists {
        return fon
    }
    return ""
}
```

### Archive Handling
```go
func HandleDownload(url string, source string) error {
    switch {
    case strings.HasSuffix(url, ".zip"):
        return extractZip(url)
    case strings.HasSuffix(url, ".tar.xz"):
        return extractTarXz(url)
    default:
        return downloadDirect(url)
    }
}
```

## Local Caching System ✅ COMPLETED

The FontGet CLI now implements a sophisticated local caching system for improved performance and reduced network usage:

### Cache Features
- **Local Storage**: Font source data cached in `~/.fontget/sources/` directory
- **Automatic Refresh**: Cache automatically refreshes if older than 24 hours
- **Manual Refresh**: `fontget sources --update` forces immediate cache refresh
- **Fallback**: If cache is corrupted or missing, fetches fresh data
- **Progress Feedback**: Shows loading progress during cache operations

### Cache Files
- `google-fonts.json` - Google Fonts source data (~1.7MB)
- `nerdfonts.json` - Nerd Fonts source data (~120KB)
- `fontsquirrel.json` - Font Squirrel source data (~1MB)

### Cache Behavior
- **First Run**: Downloads and caches all sources
- **Subsequent Runs**: Uses cached data if < 24 hours old
- **After 24 Hours**: Automatically downloads fresh data
- **Manual Refresh**: `fontget sources --update` forces immediate refresh

## Error Handling

- **Font not found**: Show similar fonts from all sources
- **Multiple sources**: Display table with all available options for user selection:
  ```
    Font Name    Font ID           Source         Category      Popularity
    ---------------------------------------------------------------------
    Roboto       google.roboto     Google Fonts   Sans Serif    95
    Roboto       squirrel.roboto   Font Squirrel  Sans Serif    85
  
  Select font ID to install: 
  ```
- **Network errors**: Retry with exponential backoff
- **Permission errors**: Clear elevation requirements
- **Invalid files**: Validate font files before installation

## Migration Steps

### Step 1: Update Data Structures
1. Create new structs in `internal/repo/types.go` matching FontGet-Sources schema
2. Keep old structs temporarily for backward compatibility
3. Update all references to use new structs

### Step 2: Update Source Loading
1. Modify `internal/repo/sources.go` to load from FontGet-Sources URLs
2. Update `internal/repo/manifest.go` to handle multiple source files
3. Update `internal/config/sources_config.go` with new URLs

### Step 3: Update Font Resolution
1. Modify `internal/repo/font.go` to search across all sources
2. Implement clean font ID resolution
3. Add source conflict resolution

### Step 4: Update Commands
1. Update all command files to use new data structures
2. Add enhanced metadata display
3. Implement source management features

### Step 5: Testing
1. Test with each FontGet-Sources file individually
2. Test font resolution with conflicts
3. Test all file types (TTF/OTF/ZIP/TAR.XZ)
4. Test cross-platform compatibility

## Testing Requirements

### Unit Tests
- [ ] Test new data structure parsing
- [ ] Test font resolution logic
- [ ] Test file type detection
- [ ] Test archive extraction
- [ ] Test source conflict resolution

### Integration Tests
- [ ] Test loading from FontGet-Sources URLs
- [ ] Test font installation from each source
- [ ] Test search across all sources
- [ ] Test source management commands

### End-to-End Tests
- [ ] Test complete font installation workflow
- [ ] Test source conflict resolution UI
- [ ] Test all command combinations
- [ ] Test error handling scenarios

## Success Criteria

- [x] All commands work with FontGet-Sources data
- [x] Clean font ID resolution with table-based conflict resolution
- [ ] Support for all file types (TTF/OTF/ZIP/TAR.XZ)
- [x] Enhanced metadata display
- [x] Source management commands
- [x] Local caching with automatic refresh
- [ ] Installation tracking
- [x] Cross-platform compatibility
- [ ] All tests pass
- [x] No breaking changes to existing functionality