# FontGet Refactoring Plan

## 🎯 **CURRENT PRIORITY: Beta Preparation**

### **Phase 1: CI/CD Pipeline (HIGH PRIORITY - Beta Blocking)**

#### **GitHub Actions Setup**
- [ ] **Create cross-platform build workflow**
  - [ ] Windows build (amd64, arm64)
  - [ ] macOS build (amd64, arm64)
  - [ ] Linux build (amd64, arm64)
  - [ ] Build matrix configuration
  - [ ] Artifact upload and naming conventions

#### **Release Automation**
- [ ] **Automated version tagging and releases**
  - [ ] Semantic versioning support
  - [ ] Automated changelog generation
  - [ ] Release notes automation
  - [ ] Binary verification and checksums (SHA256)
  - [ ] GitHub Releases integration

#### **Testing & Quality Assurance**
- [ ] **Automated testing pipeline**
  - [ ] Unit test execution on all platforms
  - [ ] Integration test suite
  - [ ] Cross-platform compatibility testing
  - [ ] Build verification tests

#### **Distribution Preparation**
- [ ] **Package manager preparation**
  - [ ] Homebrew formula (macOS)
  - [ ] Chocolatey package (Windows)
  - [ ] Linux package formats (deb, rpm) - if applicable
  - [ ] Installation script updates

#### **Code Signing & Security** (if applicable)
- [ ] **Code signing setup**
  - [ ] Windows code signing certificate
  - [ ] macOS notarization setup
  - [ ] GPG signing for releases

---

## 🧹 **Code Quality & Polish**

### **Phase 2: Help Text & Documentation**

#### **Help Text Improvements** ✅ **COMPLETED**
- [x] Review all command help text
- [x] Apply verbose/debug principles to all commands
- [x] Standardize terminology and wording
- [x] Remove redundancy and improve clarity

#### **Documentation Updates**
- [ ] **Update user documentation**
  - [ ] Update README with improved help text examples
  - [ ] Update command reference documentation
  - [ ] Add troubleshooting guide
  - [ ] Update installation instructions for CI/CD releases

---

## 🔧 **Code Quality Improvements**

### **Phase 3: Code Cleanup & Refactoring**

#### **Debug & Logging Cleanup**
- [ ] **Review debug output consistency**
  - [ ] Ensure all debug messages follow the same format
  - [ ] Verify debug output provides useful information
  - [ ] Standardize debug message formatting across all commands
  - [ ] Optimize debug performance (ensure no impact when disabled)

#### **Code Quality Assessment**
- [ ] **Identify and fix code smells**
  - [ ] Review for long functions that should be broken down
  - [ ] Extract duplicate code into shared utilities
  - [ ] Simplify complex conditional logic
  - [ ] Standardize naming conventions
  - [ ] Remove unused imports or variables

#### **Targeted Refactoring**
- [ ] **list.go improvements**
  - [ ] Extract `buildParsedFont(path, scope) ParsedFont` function
  - [ ] Reduce debug noise (keep only essential debug output)
  - [x] Optimize metadata extraction (early type filtering - filters by extension before metadata extraction)
  - [x] Cache lowercased strings in filtering loop to avoid repeated ToLower() calls
  - [ ] Consider style ordering for `--expand` output (non-blocking)

- [ ] **internal/platform/platform.go improvements**
  - [ ] Split `parseNameTable` into smaller helper functions
  - [ ] Add typed constants for name IDs and platforms
  - [ ] Ensure `extractFontMetadataFullFile` populates TypographicFamily/Style
  - [ ] Add unit tests for `parseNameTable` with synthetic name tables
  - [ ] Implement cross-platform name selection policy (see details below)

#### **Cross-Platform Name Selection Policy**
- [ ] **Implement final name-table selection order**
  1. Prefer NameID16/17 (Typographic Family/Style) regardless of platform
  2. Among multiple 16/17 records, prefer: PlatformUnicode(0) > PlatformMicrosoft(3) > PlatformMacintosh(1)
  3. Within a platform, prefer language matching system locale; fallback to English (1033)
  4. If 16/17 absent, fall back to NameID 1/2 using same platform/language order
  5. Final fallback: filename parsing
- [ ] **Test on cross-platform samples**
  - [ ] Test on Windows, macOS, and Linux
  - [ ] Verify with: Roboto, Source Code Pro, JetBrainsMono, ZedMono, Fira Code, Terminess
- [ ] **Document the policy**
  - [ ] Add to `docs/codebase.md`
  - [ ] Reference in relevant command documentation

---

## 🎨 **User Experience Enhancements**

### **Phase 4: Command Improvements**

#### **List Command Enhancements**
- [x] **Flag naming improvement**
  - [x] Rename `--full` flag to `--expand` with `-x` alias
  - [x] Update help text and documentation

#### **Color Scheme Consistency**
- [ ] **Standardize color usage**
  - [ ] Create consistent color hierarchy across all commands
  - [ ] Add color utilities to `cmd/shared.go` for easy access
  - [ ] Document color usage guidelines

---

## 🚀 **Performance Optimizations** (LOW PRIORITY - Post-Beta)

### **Phase 5: Performance Improvements**

#### **Parallel Processing**
  - [ ] Rate limiting to avoid overwhelming sources
  - [ ] Retry logic with exponential backoff for failed downloads

#### **Caching Improvements**
- [ ] **Font metadata caching**
  - [ ] Cache font metadata to reduce API calls
  - [ ] Smart cache invalidation based on source timestamps
  - [ ] Compressed cache storage for large font collections

#### **Memory Optimizations**
- [ ] **Stream processing for large files**
  - [ ] Stream font files instead of loading into memory
  - [ ] Lazy loading of font metadata
  - [ ] Memory-efficient font variant processing

#### **Network Optimizations** - Investigate
- [ ] **HTTP/2 support**
  - [ ] HTTP/2 for faster concurrent requests
  - [ ] Connection pooling and keep-alive
  - [ ] Request batching where possible
  - [ ] Respect rate limits from sources

#### **Benchmarking and Metrics**
- [ ] **Performance monitoring**
  - [ ] Performance benchmarks for different scenarios
  - [ ] Memory usage profiling
  - [ ] Download speed metrics and reporting

---

## 📦 **Future Features** (POST-BETA)

### **Phase 6: New Commands & Features**

#### **Self-Update System** (HIGH PRIORITY - Post-Beta)
- [ ] **Library Integration**
  - [ ] Add `github.com/rhysd/go-github-selfupdate/selfupdate` to `go.mod`
  - [ ] Create `internal/update/` package (wrapper around library)
  - [ ] Implement `CheckForUpdates()` function using library
  - [ ] Implement `UpdateToLatest()` function using library
  - [ ] **Note**: Library handles GitHub API, version comparison, platform detection, checksum verification, and binary replacement automatically

- [ ] **Update Command Implementation**
  - [ ] `fontget update` - Check for updates and prompt to install
  - [ ] `fontget update --check` - Only check for updates, don't install
  - [ ] `fontget update -y` - Skip confirmation prompt
  - [ ] `fontget update --version / -v <version>` - Update to specific version (should we support downgrading from here???)
  - [ ] Show current vs. available version
  - [ ] Display release notes/changelog from GitHub release
  - [ ] Atomic binary replacement (cross-platform safe)

- [ ] **Cross-Platform Binary Replacement** ✅ **Handled by Library**
  - [x] Library handles Windows binary replacement (atomic with rollback)
  - [x] Library handles macOS/Linux binary replacement (atomic with rollback)
  - [x] Library handles "file in use" errors
  - [x] Library handles backup and rollback automatically
  - [ ] **Action Required**: Ensure CI/CD builds binaries with naming: `fontget-{os}-{arch}{.ext}`
  - [ ] **Action Required**: Ensure CI/CD generates `checksums.txt` with SHA256 checksums

- [ ] **Configuration Integration**
  - [ ] Add `Update` section to `config.yaml`:
    ```yaml
    Update:
      AutoCheck: true          # Check for updates on startup
      AutoUpdate: false        # Automatically install updates (manual by default)
      CheckInterval: 24        # Hours between checks
      LastChecked: ""          # Timestamp of last check
      UpdateChannel: "stable"  # stable/dev (future)
    ```
  - [ ] First-run prompt: "Would you like FontGet to check for updates automatically?"
  - [ ] Configurable via `fontget config edit`
  - [ ] Respect `--no-check` flag to skip startup checks

- [ ] **Startup Update Check** (Optional)
  - [ ] Check `Update.AutoCheck` and `Update.CheckInterval`
  - [ ] Only check if interval has passed
  - [ ] Non-blocking check (don't delay startup)
  - [ ] Show notification if update available: "FontGet v1.2.3 is available (you have v1.2.2). Run 'fontget update' to upgrade."
  - [ ] Suppress notification if `--quiet` flag used

- [ ] **Error Handling & Edge Cases**
  - [ ] Map library errors to user-friendly messages
  - [ ] Network errors: "Unable to check for updates. Check your internet connection."
  - [ ] GitHub API errors: "Unable to fetch update information. GitHub API may be unavailable."
  - [ ] Invalid checksums: Library handles retry, show user-friendly error
  - [ ] Insufficient permissions: "Insufficient permissions. Try running as administrator/sudo."
  - [ ] Binary locked/in use: "FontGet is currently running. Please close other instances."
  - [ ] Handle pre-release versions (respect UpdateChannel) - may need custom filtering

- [ ] **User Experience**
  - [ ] Clear progress indicators during download
  - [ ] Show download size and speed
  - [ ] Confirmation prompt with version info
  - [ ] Success message with new version
  - [ ] Verbose/debug output support
  - [ ] Integration with existing UI styles

- [ ] **Testing Requirements**
  - [ ] Integration tests with library (test update flow)
  - [ ] Manual testing on Windows, macOS, Linux
  - [ ] Test rollback mechanism (library handles, but verify)
  - [ ] Test edge cases (no internet, API down, etc.)
  - [ ] Test error message mapping
  - [ ] Verify binary naming matches library expectations

- [ ] **Security Considerations** ✅ **Handled by Library**
  - [x] Library verifies SHA256 checksums before installation
  - [x] Library uses HTTPS for all downloads
  - [x] Library doesn't execute binary until verified
  - [x] Library clears temp files after update
  - [ ] **Future**: Code signing verification (library doesn't support, can add later)

#### **Backup System**
- [x] **Add `backup` command**
  - [x] Backup installed font files to zip archive
  - [x] Organize fonts by source and family name
  - [x] Auto-detect accessible scopes based on elevation
  - [x] Deduplicate fonts across scopes
  - [x] Exclude system fonts (always excluded)
  - [x] Progress bar with per-file progress updates
  - [x] **Date-based filenames**: Default format `font-backup-YYYY-MM-DD.zip`
  - [x] **Overwrite confirmation**: Prompt user before overwriting existing files
  - [x] Integrate with verbose/debug output and UI styles

#### **Export/Import System**
- [x] **Add `export` command**
  - [x] Export installed fonts/collections
  - [x] Support export by match string, source, or all
  - [x] Output manifest (JSON) with versions and variants
  - [x] Exclude system fonts (always excluded)
  - [x] Export fonts that match repository entries (Font IDs available)
  - [x] Support directory or file path via -o flag (winget-style)
  - [x] Integrate with verbose/debug output and UI styles
  - [x] Use pin spinner for progress feedback
  - [x] **Date-based filenames**: Default format `fontget-export-YYYY-MM-DD.json`
  - [x] **Overwrite confirmation**: Prompt user before overwriting existing files
  - [x] **Nerd Fonts handling**: Groups families by Font ID (one Font ID can install multiple families like ZedMono, ZedMono Mono, ZedMono Propo)
  - [x] **Backup fonts feature**: Renamed `--copy-files` to `--backup-fonts` and improved to package font files into organized zipped directory (fonts organized by Font ID or family name)

- [x] **Add `import` command**
  - [x] Import fonts from a fontget export file
  - [x] Validate import file structure and font availability
  - [x] Resolve Font IDs and install missing fonts
  - [x] Show per-font status with consistent reporting
  - [x] Integrate with verbose/debug output and UI styles
  - [x] **Nerd Fonts handling**: Deduplicates by Font ID and displays comma-separated family names (e.g., "Installed ZedMono, ZedMono Mono, ZedMono Propo")
  - [ ] **UI/UX improvements**: Line-by-line progress display (items appear as they complete, counter only in progress bar title)
  - [x] **Status message improvements**: Cleaner messaging for install/import/remove commands
    - [x] Single-scope operations: No scope clutter in status messages ("Installed", "Removed", "Skipped... already installed")
    - [x] Multi-scope operations: Show scope in status messages ("Removed from user scope", "Removed from machine scope")
    - [x] Title updates: "for All Users" for machine scope, "for All Scopes (Machine & User)" for --all scope
  - [x] **Source availability detection**: When importing fonts with Font IDs that reference disabled/unavailable sources, detect and inform the user:
    - [x] Check if source from export file exists in current manifest
    - [x] Check if source is enabled (if it exists)
    - [x] Group fonts by missing/disabled source
    - [x] **If source exists but is disabled**: Display message like "The following fonts require '{Source Name}' which is currently disabled. Enable this source via 'fontget sources manage' to import these fonts: [list of font families]"
    - [x] **If source doesn't exist** (custom source): Display message like "The following fonts require '{Source Name}' which is not available in your sources. Add this source via 'fontget sources manage' to import these fonts: [list of font families]"
    - [x] For built-in sources that don't exist: Suggest running `fontget sources update` to refresh sources
    - [x] Handle both built-in sources (can be refreshed) and custom sources (must be manually added with URL/prefix)

#### **Sources Management CLI Flags**
- [ ] **Add non-TUI flags to `sources manage`**
  - [ ] `--add <name> --prefix <prefix> --url <url> [--priority <number>]` - Add source without TUI
  - [ ] `--remove <name>` - Remove source without TUI
  - [ ] `--enable <name>` - Enable source without TUI
  - [ ] `--disable <name>` - Disable source without TUI
  - [ ] `--priority <name> <rank>` - Set source priority without TUI
  - [ ] Benefits: Automation support, CI/CD compatibility, script-friendly

---

## ✅ **COMPLETED WORK**

### **Foundation & Architecture**
- [x] Manifest-based sources system implementation
- [x] Output system redesign (verbose/debug interfaces)
- [x] Configuration consolidation
- [x] Priority system implementation
- [x] UI component system (cards, forms, confirmations, hierarchical lists)

### **Command Standardization**
- [x] Verbose/debug support across all commands (add, remove, search, list, info, config, sources)
- [x] Visual consistency across all commands
- [x] Shared function consolidation
- [x] Table standardization
- [x] Error handling standardization

### **Font Management Features**
- [x] Font matching feature with optimized index-based matching
- [x] Font ID support in add and remove commands
- [x] Protected system font filtering
- [x] Nerd Fonts variant handling
- [x] List command enhancements (Font ID, License, Categories, Source columns)
- [x] Pre-installation font checking (detects already-installed fonts before downloading)
- [x] Backup command with date-based filenames and overwrite confirmation
- [x] Export command enhancements (date-based filenames and overwrite confirmation)

### **Performance Optimizations**
- [x] Font suggestion performance optimization (90ms → 10ms)
- [x] Optimized font matching with in-memory index

### **Code Cleanup**
- [x] Removed 231 lines of unused code
- [x] Fixed duplicate manage command bug
- [x] Source priority ordering consistency
- [x] Font matching logic corrections
- [x] Centralized spinner color configuration in styles.go
- [x] Improved color mapping with PinColorMap for pin package integration

### **User Experience Improvements**
- [ ] Line-by-line progress display for install/import/remove commands (items appear as they complete)
- [x] Cleaner status messages (removed redundant counters from individual items, counter only in progress bar title)
- [x] Improved scope messaging (no scope clutter for single-scope operations, clear scope indication for multi-scope)
- [x] Title updates for machine scope operations ("for All Users", "for All Scopes (Machine & User)")

---

## 📋 **SUCCESS CRITERIA**

### **Beta Release Readiness:**
- [ ] CI/CD pipeline fully operational
- [ ] Automated builds for all target platforms
- [ ] Automated release process
- [ ] All help text reviewed and improved
- [ ] Code quality improvements completed
- [ ] Cross-platform testing verified

### **Post-Beta Goals:**
- [ ] Performance optimizations implemented
- [ ] Export/import functionality added
- [ ] Update system implemented
- [ ] Comprehensive test coverage

---

## 🔧 **DEVELOPMENT WORKFLOW**

### **For Each Command Update:**
1. **Styling** - Use `ui.PageTitle`, `ui.PageSubtitle`, `ui.FeedbackError`, etc.
2. **Verbose Framework** - Use `output.GetVerbose().Info/Warning/Error/Success`
3. **Debug Framework** - Use `output.GetDebug().Message/State/Performance/Error/Warning`
4. **Error Handling** - Use unified helpers (`ui.RenderError`, `ui.RenderWarning`, etc.)
5. **Testing** - Verify with `--verbose`, `--debug`, and `--verbose --debug` flags

### **Quality Checklist:**
- [ ] Visual parity with add.go
- [ ] Verbose and debug produce meaningful output
- [ ] Default mode remains clean
- [ ] Consistent status reporting
- [ ] No direct prints; all routed through output/ui helpers
- [ ] Help text follows CLI best practices

---

## 📊 **Current Status**

**Overall Progress**: Foundation complete, commands standardized, ready for beta preparation

**Next Steps**: 
1. **IMMEDIATE**: Set up CI/CD pipeline (Phase 1)
2. **SHORT TERM**: Code quality improvements (Phase 3)
3. **MEDIUM TERM**: UX enhancements (Phase 4)
4. **LONG TERM**: Performance optimizations and new features (Phases 5-6)

**Blockers for Beta**: CI/CD pipeline setup
