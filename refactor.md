# FontGet Refactoring Plan

## ✅ **COMPLETED FOUNDATION**


---

## **List Command Changes**
- [ ] Change --full flag to something else that better matches the hierarchy view (working name)

## 🎯 **CURRENT FOCUS: Command Consistency & Polish**

- [ ] **Color Scheme Enhancements**
  - [ ] Create consistent color hierarchy across all commands
  - [ ] Add color utilities to `cmd/shared.go` for easy access


## 🧹 **Code Quality & Cleanup Milestone**

### **Phase 4: Code Quality Audit & Cleanup**

#### **Debug & Logging Cleanup**
- [ ] **Review debug output consistency** - Ensure all debug messages follow the same format and provide useful information
- [ ] **Remove redundant debug statements** - Clean up duplicate or unnecessary debug output
- [ ] **Standardize debug message formatting** - Use consistent patterns for debug output across all commands
- [ ] **Optimize debug performance** - Ensure debug output doesn't impact performance when disabled

#### **Code Quality Assessment**
- [ ] **Identify code smells** - Review codebase for common issues like:
  - [ ] Long functions that should be broken down
  - [ ] Duplicate code that can be extracted
  - [ ] Complex conditional logic that can be simplified
  - [ ] Inconsistent naming conventions
  - [ ] Unused imports or variables
- [ ] **Refactor complex functions** - Break down large functions into smaller, more maintainable pieces
- [ ] **Improve error handling** - Standardize error handling patterns across the codebase
- [ ] **Add missing documentation** - Ensure all public functions have proper documentation
- [ ] **Optimize imports** - Remove unused imports and organize import statements

#### **Performance & Memory Optimization**
- [ ] **Memory usage audit** - Identify potential memory leaks or excessive allocations
- [ ] **Function call optimization** - Reduce unnecessary function calls in hot paths
- [ ] **String operations optimization** - Use more efficient string operations where possible
- [ ] **Goroutine cleanup** - Ensure proper cleanup of goroutines and channels

#### **Testing & Validation**
- [ ] **Add unit tests for critical functions** - Focus on font matching, source priority, and configuration loading
- [ ] **Integration test improvements** - Test cross-command consistency
- [ ] **Performance benchmarking** - Establish baseline performance metrics
- [ ] **Memory profiling** - Identify memory usage patterns and optimization opportunities

---

## 🧹 Targeted Cleanup: list.go and platform.go

### list.go
- [ ] Extract `buildParsedFont(path, scope) ParsedFont` to encapsulate metadata extraction and display-name choice (uses TypographicFamily/Style when present).
- [ ] Reduce debug noise: keep only
  - [ ] Scan summary (files scanned per scope)
  - [ ] Family grouping summary (family name, file count)
  - [ ] Unique variant count in `--detailed`
- [ ] Keep performance: one metadata read per file; no re-parsing in grouping/rendering.
- [ ] Consider style ordering (Thin→Black) for nicer `--detailed` output (non-blocking).

### internal/platform/platform.go
- [ ] Split `parseNameTable` into smaller helpers:
  - [ ] `forEachNameRecord(data, fn(record))` – decoding bounds/UTF-16 handling.
  - [ ] `selectPreferredNames(records)` – policy to choose NameID16/17 with fallback.
- [ ] Add typed constants for name IDs and platforms (e.g., NameIDFamily=1, NameIDTypoFamily=16, PlatformMicrosoft=3, PlatformUnicode=0, PlatformMacintosh=1).
- [ ] Ensure `extractFontMetadataFullFile` also populates `TypographicFamily/Style` when available from sfnt (or marks as unavailable) to keep parity with header-only path.
- [ ] Unit-test `parseNameTable` with synthetic name tables covering:
  - [ ] Multiple platform/language records
  - [ ] Absence of NameID16/17 (fallback to 1/2)
  - [ ] Mixed encodings and language IDs

### Cross‑Platform Name Selection Policy (Important)
- Current temporary preference leans to Microsoft/English (platform=3, language=1033) when present because it’s the most complete on Windows Nerd Fonts.
- Final cross‑platform policy to implement and test:
  1. Prefer NameID16/17 regardless of platform.
  2. Among multiple 16/17 records, prefer Unicode entries in this order: PlatformUnicode(0) > PlatformMicrosoft(3) > PlatformMacintosh(1).
  3. Within a platform, prefer language matching system locale; fallback to English (1033) if present; otherwise choose a deterministic Unicode record (Platform 0) if any; if not available, choose the lowest language ID under Microsoft (3). Only if no 16/17 fit these, fall back to NameID 1/2 using the same order, and if still empty, finally use filename parsing.
  4. Only if 16/17 absent, fall back to NameID 1/2 using the same platform/language order.
- [ ] Implement the above selection order and verify on macOS/Linux/Windows samples (Roboto, Source Code Pro, JetBrainsMono, ZedMono, Fira Code, Terminess).

### Documentation
- [ ] Document the final name-table selection policy in `docs/codebase.md` and reference it in `list-plan.md` to avoid regressions.

---


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
- [ ] **Network optimizations** - Needs to respect rate limits
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
  - [ ] `--priority <name> <rank>` - Set source priority without TUI
- [ ] **Benefits for automation**
  - [ ] Bypass TUI when not available or desired
  - [ ] Enable AI agents and scripts to manage sources
  - [ ] Better compatibility with CI/CD and automated environments
  - [ ] Consistent with other CLI tools' management patterns

#### **Font ID Resolution & Smart Matching** ✅ **COMPLETED**

##### **Phase 1: Installation Tracking System**
- [ ] **Add installation tracking system** - **NOT IMPLEMENTED**
  - [ ] Create `~/.fontget/installed.json` to track font ID → system name mappings
  - [ ] Update install process to record font ID when installing via FontGet
- [ ] **Smart font detection (winget-style)** - **NOT IMPLEMENTED**
  - [ ] Add tracking for font variants and their system names
  - [ ] Handle font updates and reinstallations

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
  - [ ] Exclude fonts like system fonts or non fontget fonts maybe only;
    - [ ] Export only installed by fontget (some way to track this?? Dicuss); or
    - [ ] Export fonts that match the current sources that are available??? (Needs discussion)
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
- [ ] Complete verbose/debug support across all commands

### **Overall Project Success:**
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

**Current Status**: 7/7 commands fully standardized with visual consistency. Source priority and font matching logic fixed across all commands. Font matching feature completed with optimized index-based matching. Font ID support added to add and remove commands. Ready for Phase 4: Code Quality & Cleanup milestone.
**COMPLETED**: Shared function consolidation, table standardization, performance optimization for suggestion systems, complete UI component system with modern card components, form elements, confirmation dialogs, hierarchical lists, info command card-based layout implementation, remove command visual parity with add command, list command styling and font matching feature, sources command styling, fixed duplicate manage command bug, source priority ordering consistency, font matching logic corrections, Font ID support in add/remove commands, optimized font matching with in-memory index, protected system font filtering, Nerd Fonts variant handling, and unused code cleanup (231 lines removed).