# FontGet Refactoring Plan

## 🎯 **CURRENT PRIORITY: Beta Preparation**

### **Phase 1: CI/CD Pipeline (HIGH PRIORITY - Beta Blocking)** ✅ **COMPLETED**

#### **Distribution Preparation** ✅ **MOSTLY COMPLETED**
- [x] **Package manager preparation**
  - [x] Homebrew formula (macOS)
  - [ ] Chocolatey package (Windows) - Optional, not blocking
  - [x] Linux package formats (deb, rpm)
  - [x] Scoop manifest (Windows)
  - [x] Installation script updates
  - [x] GoReleaser configuration for automated releases
  - [x] GitHub Actions CI/CD workflows (ci.yml, release.yml)

#### **Code Signing & Security** (if applicable)
- [ ] **Code signing setup**
  - [ ] Windows code signing certificate
  - [ ] macOS notarization setup
  - [ ] GPG signing for releases

---

## 🧹 **Code Quality & Polish**

### **Phase 2: Help Text & Documentation**

#### **Documentation Updates**
- [ ] **Update user documentation**
  - [ ] Update README with improved help text examples
  - [ ] Update command reference documentation
  - [ ] Add troubleshooting guide
  - [ ] Update installation instructions for CI/CD releases

---

## 🔧 **Code Quality Improvements**

### **Phase 3: Code Cleanup & Refactoring**

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

#### **Self-Update System** ✅ **MOSTLY COMPLETE** (HIGH PRIORITY - Post-Beta)
- [x] **Library Integration**
  - [x] Add `github.com/rhysd/go-github-selfupdate/selfupdate` to `go.mod`
  - [x] Create `internal/update/` package (wrapper around library)
  - [x] Implement `CheckForUpdates()` function using library
  - [x] Implement `UpdateToLatest()` function using library
  - [x] **Note**: Library handles GitHub API, version comparison, platform detection, checksum verification, and binary replacement automatically

- [x] **Update Command Implementation**
  - [x] `fontget update` - Check for updates and prompt to install
  - [x] `fontget update --check` - Only check for updates, don't install
  - [x] `fontget update -y` - Skip confirmation prompt
  - [x] `fontget update --version / -v <version>` - Update to specific version (supports downgrading)
  - [x] Show current vs. available version
  - [x] Display changelog link (in normal mode)
  - [x] Atomic binary replacement (cross-platform safe)

- [x] **Cross-Platform Binary Replacement** ✅ **Handled by Library**
  - [x] Library handles Windows binary replacement (atomic with rollback)
  - [x] Library handles macOS/Linux binary replacement (atomic with rollback)
  - [x] Library handles "file in use" errors
  - [x] Library handles backup and rollback automatically
  - [x] **CI/CD**: GoReleaser generates binaries with proper naming and `checksums.txt` with SHA256 checksums

- [x] **Configuration Integration**
  - [x] Add `Update` section to `config.yaml`:
    ```yaml
    Update:
      AutoCheck: true          # Check for updates on startup
      AutoUpdate: false        # Automatically install updates (manual by default)
      UpdateCheckInterval: 24  # Hours between checks
      LastChecked: ""          # Timestamp of last check (auto-updated)
      UpdateChannel: "stable"  # stable/beta/nightly
    ```
  - [x] First-run prompt in onboarding flow
  - [x] Configurable via `fontget config edit` and `fontget config info`
  - [x] Respects `Update.AutoCheck` and `Update.UpdateCheckInterval` settings

- [x] **Startup Update Check**
  - [x] Check `Update.AutoCheck` and `Update.UpdateCheckInterval`
  - [x] Only check if interval has passed
  - [x] Non-blocking check (don't delay startup)
  - [x] Show notification if update available
  - [x] Auto-update support when `Update.AutoUpdate` is enabled

- [x] **Error Handling & Edge Cases**
  - [x] Map library errors to user-friendly messages
  - [x] Network errors handled with user-friendly messages
  - [x] GitHub API errors handled gracefully
  - [x] Invalid checksums: Library handles retry, shows user-friendly error
  - [x] Insufficient permissions: User-friendly error messages
  - [x] Binary locked/in use: Handled by library
  - [ ] Handle pre-release versions (respect UpdateChannel) - may need custom filtering

- [ ] **Testing Requirements**
  - [ ] Integration tests with library (test update flow)
  - [x] Manual testing on Windows, macOS, Linux
  - [x] Test rollback mechanism (library handles, verified)
  - [x] Test edge cases (no internet, API down, etc.)
  - [x] Test error message mapping
  - [x] Verify binary naming matches library expectations

- [x] **Security Considerations** ✅ **Handled by Library**
  - [x] Library verifies SHA256 checksums before installation
  - [x] Library uses HTTPS for all downloads
  - [x] Library doesn't execute binary until verified
  - [x] Library clears temp files after update
  - [ ] **Future**: Code signing verification (library doesn't support, can add later)

#### **Sources Management CLI Flags**
**Goal**: Enable automation-friendly, non-interactive source management for scripts and CI/CD

- [ ] **Add `sources add` subcommand** (non-TUI alternative to `sources manage`)
  - [ ] `fontget sources add --name <name> --url <url> [--prefix <prefix>] [--priority <number>]`
  - [ ] Auto-generate prefix from name if not provided
  - [ ] Validate URL format and source accessibility
  - [ ] Error if source name/prefix already exists

- [ ] **Add `sources remove` subcommand**
  - [ ] `fontget sources remove --name <name>`
  - [ ] Prevent removal of built-in sources (error message)
  - [ ] Confirm removal or add `--force` flag

- [ ] **Add `sources enable/disable` subcommands**
  - [ ] `fontget sources enable --name <name>`
  - [ ] `fontget sources disable --name <name>`
  - [ ] Work with both custom and built-in sources

- [ ] **Add `sources set` subcommand** (update source properties)
  - [ ] `fontget sources set --name <name> --priority <number>` - Update priority
  - [ ] `fontget sources set --name <name> --prefix <prefix>` - Update prefix
  - [ ] `fontget sources set --name <name> --url <url>` - Update URL
  - [ ] Support multiple properties: `--name <name> --priority <num> --prefix <prefix>`
  - [ ] Prevent modifying built-in source properties (error message)

---

### **User Experience Improvements**
- [ ] Line-by-line progress display for install/import/remove commands (items appear as they complete)
- [x] Cleaner status messages (removed redundant counters from individual items, counter only in progress bar title)
- [x] Improved scope messaging (no scope clutter for single-scope operations, clear scope indication for multi-scope)
- [x] Title updates for machine scope operations ("for All Users", "for All Scopes (Machine & User)")
- [x] Button-based confirmation dialogs (replaced Y/N prompts with interactive button UI)
- [x] Improved update command output (styled version display, changelog links)

---

## 📋 **SUCCESS CRITERIA**

### **Beta Release Readiness:**
- [x] CI/CD pipeline fully operational
- [x] Automated builds for all target platforms
- [x] Automated release process (GoReleaser)
- [x] All help text reviewed and improved
- [ ] Code quality improvements completed (ongoing)
- [x] Cross-platform testing verified

### **Post-Beta Goals:**
- [ ] Performance optimizations implemented
- [x] Export/import functionality added
- [x] Update system implemented
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

**Overall Progress**: Foundation complete, commands standardized, CI/CD operational, update system implemented, ready for beta

**Next Steps**: 
1. **SHORT TERM**: Code quality improvements (Phase 3)
2. **MEDIUM TERM**: UX enhancements (Phase 4)
3. **LONG TERM**: Performance optimizations and new features (Phases 5-6)

**Recent Completions**: 
- ✅ CI/CD pipeline fully operational
- ✅ Self-update system implemented
- ✅ Export/import functionality complete
- ✅ Help text reviewed and improved
