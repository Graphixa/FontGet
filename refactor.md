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
  - [ ] Optimize metadata extraction (one read per file)
  - [ ] Consider style ordering for `--full` output (non-blocking)

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
- [ ] **Flag naming improvement**
  - [ ] Rename `--full` flag to better match hierarchy view (e.g., `--tree`)
  - [ ] Update help text and documentation

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

#### **Update System**
- [ ] **Add `update` command**
  - [ ] `fontget update` - Check and update if newer version available
  - [ ] `fontget update --check` - Just check for updates without installing
  - [ ] Integration with GitHub Releases API
  - [ ] Version comparison and update logic
  - [ ] Backup current version before update
  - [ ] Rollback capability if update fails

#### **Export/Import System**
- [ ] **Add `export` command**
  - [ ] Export installed fonts/collections
  - [ ] Support export by family, source, or all
  - [ ] Output manifest (JSON) with versions and variants
  - [ ] Exclude fonts like system fonts or non fontget fonts maybe only;
    - [ ] Export only installed by fontget (some way to track this?? Dicuss); or
    - [ ] Export fonts that match the current sources that are available??? (Needs discussion)
  - [ ] Include file copy option and dry-run mode
  - [ ] Integrate with verbose/debug output and UI styles

- [ ] **Add `import` command**
  - [ ] Import fonts from a fontget export file
  - [ ] Validate import file structure and font availability
  - [ ] Resolve sources and install missing fonts
  - [ ] Show per-font status with consistent reporting
  - [ ] Integrate with verbose/debug output and UI styles

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

### **Performance Optimizations**
- [x] Font suggestion performance optimization (90ms → 10ms)
- [x] Optimized font matching with in-memory index

### **Code Cleanup**
- [x] Removed 231 lines of unused code
- [x] Fixed duplicate manage command bug
- [x] Source priority ordering consistency
- [x] Font matching logic corrections

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
