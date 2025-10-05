# FontGet Refactoring Plan

## ✅ **COMPLETED FOUNDATION**

### **Core Infrastructure (COMPLETED)**
- [x] **Style System** - Centralized UI components with Catppuccin Mocha palette
- [x] **Verbose/Debug Framework** - Clean output interface with `output.GetVerbose()` and `output.GetDebug()`
- [x] **Sources Architecture** - Manifest-based system with auto-bootstrapping
- [x] **Archive Handling** - ZIP/TAR.XZ extraction for all font sources
- [x] **Documentation Structure** - Organized docs with help.md, codebase.md, contributing.md
- [x] **Error Handling** - Standardized UI components (5/7 commands updated)
- [x] **Font Installation Feedback** - Detailed variant reporting with symbols

---

## 🎯 **CURRENT FOCUS: Command Consistency & Polish**

### **Phase 3: Complete Command Standardization**

#### **Remaining Error Handling Standardization**
- [x] **Update `cmd/sources.go`** - Replace direct color functions with UI components
  - [x] Replace `red()`, `green()`, `yellow()` with `ui.RenderError()`, `ui.RenderSuccess()`, `ui.RenderWarning()`
  - [x] Standardize error message formatting
- [x] **Update `cmd/config.go`** - Replace direct color functions with UI components
  - [x] Replace `color.New(color.FgRed).SprintFunc()` with `ui.RenderError()`
  - [x] Standardize success/warning messages

#### **Command Visual Consistency**
- [ ] **Apply "Gold Standard" to remaining commands** (using `cmd/add.go` and `cmd/search.go` as baseline)
  - [x] `cmd/remove.go` - Visual parity with add.go (already matches)
  - [ ] `cmd/list.go` - Styling and headers
  - [x] `cmd/info.go` - Card-based layout implemented with modern UI components
  - [ ] `cmd/sources.go` - Styling parity (info, update, manage) use table like search for sources showing source info such as size in mb

#### **Enhanced Command Layouts** (READY FOR IMPLEMENTATION - Based on ideas.md)
- [x] **Info Command Card-Based Layout** - **COMPLETED**
  - [x] Card components implemented with integrated titles and flexible padding
  - [x] Helper functions available: `FontDetailsCard()`, `LicenseInfoCard()`, `AvailableFilesCard()`, `MetadataCard()`
  - [x] Implement bordered card sections for different information categories
  - [x] Font Details card (Name, ID, Category) - **IMPLEMENTED**
  - [x] License Info card (License, URL) - **IMPLEMENTED**
  - [x] Available Files card (Download URLs) - **IMPLEMENTED**
  - [x] Metadata card (Last Modified, Source URL, Popularity) - **IMPLEMENTED**
  - [x] Use charmbracelet TUI components for consistent styling - **COMPLETED**
- [ ] **List Command Hierarchical Display** - **COMPONENTS READY**
  - [x] Hierarchy components implemented with proper indentation and arrows
  - [x] Font families display without indentation, variants with `↳` arrows
  - [x] Proper spacing between different font families
  - [ ] Show font families with their variants in a tree-like structure
  - [ ] Use pink color for font family names
  - [ ] Use regular console color for font variants
  - [ ] Add `--detailed` or `--full` flag to show font variants
  - [ ] Default mode shows only font families (compact view)
  - [ ] Detailed mode shows all variants with indentation
- [ ] **Dynamic Table System Integration**
  - [ ] Apply dynamic table widths to all commands using search/add/remove tables
  - [ ] Update `cmd/add.go` and `cmd/remove.go` to use dynamic widths for suggestion tables
  - [ ] Ensure consistent table behavior across all commands
- [ ] **Color Scheme Enhancements**
  - [ ] Implement pink color for font family names (matching ideas.md)
  - [ ] Use regular console color for font variants and secondary text
  - [ ] Create consistent color hierarchy across all commands
  - [ ] Add color utilities to `cmd/shared.go` for easy access

#### **Verbose/Debug Implementation**
- [ ] **Complete verbose/debug framework for remaining commands**
  - [ ] `cmd/remove.go` - Add verbose details (files removed, scope/elevation, protected detection)
  - [ ] `cmd/search.go` - Add verbose details (parameters, filters, counts)
  - [ ] `cmd/list.go` - Add verbose details (scan directories, parsed files, filters)
  - [ ] `cmd/info.go` - Add verbose details (lookup flow, source resolution)
  - [ ] `cmd/sources.go` - Add verbose details (update plan, per-source outcomes)

#### **UI Component Extraction** ✅ **COMPLETED**
- [x] **Create reusable UI components**
  - [x] ~~Extract table components to `internal/components/table.go`~~ **NOT NEEDED** - Table system already well-centralized in `cmd/shared.go`
  - [x] Extract form components to `internal/components/form.go` - **COMPLETED** - Reusable form elements for TUI interfaces
  - [x] Extract progress indicators to `internal/components/progress.go` - **COMPLETED** - Already existed, enhanced with Bubble Tea integration
  - [x] Extract confirmation dialogs to `internal/components/confirm.go` - **COMPLETED** - Standardized confirmation prompts with consistent styling
  - [x] Extract card components to `internal/components/card.go` - **COMPLETED** - Modern card system with integrated titles and flexible padding
  - [x] Extract hierarchical list components to `internal/components/hierarchy.go` - **COMPLETED** - Tree-like display for structured data (font families with variants)
  - [x] ~~Extract color scheme utilities to `internal/components/colors.go`~~ **NOT NEEDED** - Color scheme already well-centralized in `internal/ui/styles.go`
- [x] **Card System Enhancement** - **COMPLETED**
  - [x] Integrated title rendering in card borders with proper styling
  - [x] Flexible padding controls (vertical and horizontal) for different use cases
  - [x] Consistent styling using `ui.CardTitle`, `ui.CardLabel`, `ui.CardContent` styles
  - [x] Proper border color matching and ANSI escape code handling
  - [x] Backward compatibility maintained with existing API

#### **Shared Function Consolidation** (HIGH PRIORITY)
- [x] **Table standardization system** - Created flexible table system in `cmd/shared.go`
  - [x] `GetSearchTableHeader()` - Font search/add/remove tables (5 columns)
  - [x] `GetListTableHeader()` - Font list tables (5 columns) 
  - [x] `GetTableSeparator()` - Consistent separator line
  - [x] Column width constants for all table types
  - [x] Maximum table width: 120 characters (full terminal width)
  - [x] `GetSourcesTableHeader()` - Reserved for future sources info table (not currently used)
- [x] **Update commands to use shared table system**
  - [x] `cmd/list.go` - Replace custom table formatting with `GetListTableHeader()`
  - [x] `cmd/search.go` - Replace custom table formatting with `GetSearchTableHeader()`
  - [x] `cmd/sources.go` - Currently uses simple text formatting, no table needed
  - [x] `cmd/add.go` - Already using `GetSearchTableHeader()`
  - [x] `cmd/remove.go` - Already using `GetSearchTableHeader()`
- [x] **Move remaining duplicate functions to `cmd/shared.go`**
  - [x] `truncateString()` - Used in both add.go and remove.go
  - [x] `findSimilarFonts()` and `findSimilarInstalledFonts()` - Font suggestion logic (consolidated into unified `findSimilarFonts()`)
  - [x] `showFontNotFoundWithSuggestions()` and `showInstalledFontNotFoundWithSuggestionsCached()` - Suggestion display (kept command-specific as they differ significantly)
  - [x] `formatFontNameWithVariant()` - Font name formatting (only in add.go, not duplicated)
  - [x] `extractFontDisplayNameFromFilename()` - Font filename parsing (only in remove.go, not duplicated)
  - [x] `convertCamelCaseToSpaced()` - String formatting utilities (only in remove.go, not duplicated)
  - [x] `buildInstalledFontsCache()` - Font discovery caching (only in remove.go, not duplicated)
- [x] **Benefits of consolidation**
  - [x] Eliminate code duplication between add/remove commands
  - [x] Centralize font suggestion and display logic
  - [x] Easier maintenance and consistency
  - [x] Reduced binary size
  - [x] Single source of truth for font handling utilities

#### **Evaluate Performance Optimisations** (LOW PRIORITY)
- [x] **Font suggestion performance optimization** - Optimized add command suggestion table performance
  - [x] Analyzed performance bottlenecks in suggestion display
  - [x] Implemented fresh data approach (90ms vs 10ms - imperceptible to humans)
  - [x] Maintained dynamic source detection without caching complexity
  - [x] Verified remove command performance (~10ms) remains optimal
- [ ] **Add parallel processing**
  - [ ] Parallel font downloads for multiple fonts
  - [ ] Worker pool with configurable concurrency (default: 3-5 workers)
  - [ ] Rate limiting to avoid overwhelming sources
  - [ ] Retry logic with exponential backoff for failed downloads
- [ ] **Caching improvements**
  - [ ] Font metadata caching to reduce API calls
  - [ ] Smart cache invalidation based on source timestamps
  - [ ] Compressed cache storage for large font collections
- [ ] **Memory optimizations**
  - [ ] Stream processing for large font files instead of loading into memory
  - [ ] Lazy loading of font metadata
  - [ ] Memory-efficient font variant processing
- [ ] **Network optimizations**
  - [ ] HTTP/2 support for faster concurrent requests
  - [ ] Connection pooling and keep-alive
  - [ ] Request batching where possible
- [ ] **Benchmarking and metrics**
  - [ ] Performance benchmarks for different scenarios
  - [ ] Memory usage profiling
  - [ ] Download speed metrics and reporting

#### **Sources Management CLI Flags** (LOW PRIORITY)
- [ ] **Add CLI flags to `sources manage` command**
  - [ ] `--add <name> --prefix <prefix> --url <url> [--priority <number>]` - Add new source without TUI
  - [ ] `--remove <name>` - Remove source without TUI (can't remove built-in sources, can only disable/enable)
  - [ ] `--enable <name>` - Enable source without TUI
  - [ ] `--disable <name>` - Disable source without TUI
  - [ ] `--priority <name> --priority <number>` - Set source priority without TUI
- [ ] **Benefits for automation**
  - [ ] Bypass TUI when not available or desired
  - [ ] Enable AI agents and scripts to manage sources
  - [ ] Better compatibility with CI/CD and automated environments
  - [ ] Consistent with other CLI tools' management patterns

#### **Font ID Resolution & Smart Matching** (HIGH PRIORITY)

##### **Phase 1: Installation Tracking System**
- [ ] **Add installation tracking system**
  - [ ] Create `~/.fontget/installed.json` to track font ID → system name mappings
  - [ ] Update install process to record font ID when installing via FontGet
- [ ] **Smart font detection (winget-style)**
  - [ ] Add tracking for font variants and their system names
  - [ ] Handle font updates and reinstallations

##### **Phase 2: Smart Font Detection (winget-style)**
- [ ] **Smart font detection**
  - [ ] Scan system fonts against all FontGet sources
  - [ ] Match system fonts to FontGet sources by priority order
  - [ ] Show Font ID for detected fonts, blank for unknown fonts
  - [ ] Cache detection results for performance
- [ ] **Dynamic source detection for remove command**
  - [ ] Replace "System" placeholder with actual source name (e.g., "Google Fonts", "Nerd Fonts")
  - [ ] Show Font ID instead of "N/A" when font is detected in sources
  - [ ] Display license and categories when available from source
  - [ ] Fall back to "System Font" only for fonts not found in any source
- [ ] **Enhanced list command**
  - [ ] Show Font Name, Font ID, Variants, Categories (e.g. Display, Serif etc. if possible) and Source columns
  - [ ] Display detected fonts with their Font ID from highest priority source
  - [ ] Leave Font ID blank for fonts not in any FontGet source
  - [ ] Add filtering by source and Font ID

##### **Phase 3: Enhanced Remove Command**
- [ ] **Enhanced remove command**
  - [ ] Support both font IDs (`nerd.fira-code`) and system names (`Fira Code`)
  - [ ] Add suggestion system like add command
  - [ ] Support partial matches (`open-sans` matches across multiple sources)
  - [ ] Show source resolution when multiple sources match
  - [ ] Integrate with installation tracking system

#### **Update System** (MEDIUM PRIORITY)

##### **Phase 1: Update Command Implementation**
- [ ] **Add `update` command**
  - [ ] `fontget update` - Check and update if newer version available
  - [ ] `fontget update --check` - Just check for updates without installing
  - [ ] Integration with GitHub Releases API
  - [ ] Version comparison and update logic
  - [ ] Backup current version before update
  - [ ] Rollback capability if update fails

##### **Phase 2: GitHub Actions Setup**
- [ ] **Set up GitHub Actions for automated builds**
  - [ ] Cross-platform build workflow (Windows, macOS, Linux)
  - [ ] Automated version tagging and releases
  - [ ] Build matrix for different architectures
  - [ ] Artifact upload and release creation
  - [ ] Automated testing on multiple platforms

##### **Phase 3: Build System & Distribution**
- [ ] **Complete CI/CD pipeline**
  - [ ] Code signing for Windows/macOS
  - [ ] Automated changelog generation
  - [ ] Release notes automation
  - [ ] Binary verification and checksums
  - [ ] Distribution to package managers (Homebrew, Chocolatey, etc.)

#### **Future Commands**
- [ ] **Add `export` command** - Export installed fonts/collections
  - [ ] Support export by family, source, or all
  - [ ] Output manifest (JSON) with versions and variants
  - [ ] Include file copy option and dry-run mode
  - [ ] Integrate with verbose/debug output and UI styles
- [ ] **Add `import` command** - Import fonts from an export manifest
  - [ ] Validate manifest structure and availability
  - [ ] Resolve sources and install missing fonts
  - [ ] Show per-font status with consistent reporting
  - [ ] Integrate with verbose/debug output and UI styles

---

## 📋 **SUCCESS CRITERIA**

### **Phase 3 Completion Criteria:**
- [ ] All commands use centralized style system
- [ ] Consistent visual hierarchy across all commands
- [x] Reusable UI components implemented
- [x] Error handling standardized across all commands
- [ ] All commands follow same interaction patterns
- [ ] Complete verbose/debug support across all commands

### **Overall Project Success:**
- [x] All major font sources functional (Google Fonts, Nerd Fonts, Font Squirrel)
- [x] Archive handling implemented (ZIP/TAR.XZ support)
- [x] Smart font naming for extracted archives
- [x] Documentation restructure and organization
- [x] Dead code cleanup and reference consistency
- [x] Audit script improvements and maintenance tools
- [ ] **REMAINING**: Complete visual consistency across all commands
- [ ] **REMAINING**: Reusable UI components implemented

### **Testing & Quality**
- [ ] **Add comprehensive testing**
  - [ ] Unit tests for new components and utilities
  - [ ] Integration tests for updated commands
  - [ ] Cross-platform compatibility testing
  - [ ] Documentation accuracy testing

---

## 🔧 **DEVELOPMENT WORKFLOW**

### **For Each Command Update:**
1. **Styling** - Use `ui.PageTitle`, `ui.PageSubtitle`, `ui.FeedbackError`, etc.
2. **Verbose Framework** - Replace prints with `output.GetVerbose().Info/Warning/Error/Success`
3. **Debug Framework** - Add `output.GetDebug().Message/State/Performance/Error/Warning`
4. **Error Handling** - Use unified helpers (`ui.RenderError`, `ui.RenderWarning`, etc.)
5. **Testing** - Verify with `--verbose`, `--debug`, and `--verbose --debug` flags

### **Quality Checklist:**
- [ ] Visual parity with add.go
- [ ] Verbose and debug produce meaningful output
- [ ] Default mode remains clean
- [ ] Consistent status reporting
- [ ] No direct prints; all routed through output/ui helpers

---

**Current Status**: 6/7 commands fully standardized, 1 command needs error handling updates, most commands have visual consistency. 
**COMPLETED**: Shared function consolidation, table standardization, performance optimization for suggestion systems, complete UI component system with modern card components, form elements, confirmation dialogs, hierarchical lists, info command card-based layout implementation, and remove command visual parity with add command.