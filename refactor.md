# FontGet Refactoring Plan

## 🧹 **Code Quality & Polish**

### **Phase 2: Help Text & Documentation**

#### **Documentation Updates**
- [ ] **Update user documentation**
  - [ ] Update README with improved help text examples
  - [x] Update command reference documentation (`docs/usage.md`)
  - [ ] Add troubleshooting guide
  - [x] Update installation instructions for CI/CD releases

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
  - [ ] Add to `docs/development/codebase.md`
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
  - [ ] Add color utilities to `internal/ui` for easy access (shared helpers; not `cmd/shared.go`)
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

#### **Self-Update System** (remaining work)
- [ ] Handle pre-release versions (respect UpdateChannel) - may need custom filtering
- [ ] Integration tests with library (test update flow)
- [ ] **Future**: Code signing verification (library doesn't support, can add later)

#### **Sources Management CLI Flags** ✅ **DONE**
- [x] **`sources add`** — non-TUI; name, URL, optional prefix/priority (`cmd/sources_cli.go`)
- [x] **`sources remove`** — built-ins protected; `--force` / `--yes` for scripts
- [x] **`sources enable` / `sources disable`** — by name or prefix
- [x] **`sources set`** — update URL, prefix, and/or priority for custom sources

#### **Source priority: make it matter everywhere**
**Why**: Priority is stored in the manifest and can be set via `sources add` / `sources set`, but it is only used in a few places (sources update order, sources manage TUI). Search result order and the sources info table ignore it and use hardcoded or non-priority logic. That makes priority feel pointless. The intent is: **lower number = this source is preferred first** (e.g. when the same font appears in multiple sources, or when listing/loading sources).

**How it would work**:
- **Single source of truth**: Manifest `SourceConfig.Priority` (and built-in vs custom) defines order. No duplicate hardcoded maps in the repo.
- **Search result sort** (`internal/repo/sources.go`): When sorting search results by source, use each source’s priority from the manifest instead of the hardcoded `sourcePriority` map. The repo needs access to a name→priority mapping (e.g. pass it in when loading the manifest, or have the caller pass ordered source names / a priority getter). Custom sources keep their manifest priority; built-ins keep 1–3 unless the user has changed them.
- **Sources info table** (`cmd/sources.go`): Sort table rows by priority (built-in first, then by `SourceConfig.Priority`, then by name). Use the same ordering as `GetEnabledSourcesInOrder` (or a shared helper that returns all sources in priority order for display).
- **Repo load order**: When building the font manifest in `loadAllSourcesWithCache`, iterate over sources in priority order (e.g. get ordered list from config/functions) instead of ranging over the map, so load order is deterministic and matches user preference.

**Tasks**:
- [ ] Replace hardcoded `sourcePriority` in `internal/repo/sources.go` with manifest-driven priority (obtain name→priority from config when sorting search results).
- [ ] Sort sources info table rows by priority (built-in first, then manifest priority, then name).
- [ ] In `loadAllSourcesWithCache` (or equivalent), iterate sources in priority order when loading so repo build order is consistent.
- [ ] Document in docs/usage.md that priority controls order in search results, sources info, and sources update (lower = higher preference).

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
- ✅ Installation doc updated (Windows section, package manager layout)
- ✅ `fontget config set` command (reflection-based, schema-driven)
