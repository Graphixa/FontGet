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
- [ ] **Update `cmd/sources.go`** - Replace direct color functions with UI components
  - [ ] Replace `red()`, `green()`, `yellow()` with `ui.RenderError()`, `ui.RenderSuccess()`, `ui.RenderWarning()`
  - [ ] Standardize error message formatting
- [ ] **Update `cmd/config.go`** - Replace direct color functions with UI components
  - [ ] Replace `color.New(color.FgRed).SprintFunc()` with `ui.RenderError()`
  - [ ] Standardize success/warning messages

#### **Command Visual Consistency**
- [ ] **Apply "Gold Standard" to remaining commands** (using `cmd/add.go` and `cmd/search.go` as baseline)
  - [ ] `cmd/remove.go` - Visual parity with add.go
  - [ ] `cmd/list.go` - Styling and headers
  - [ ] `cmd/info.go` - Styling and content sections
  - [ ] `cmd/sources.go` - Styling parity (info, update, manage) use table like search for sources showing source info such as size in mb

#### **Verbose/Debug Implementation**
- [ ] **Complete verbose/debug framework for remaining commands**
  - [ ] `cmd/remove.go` - Add verbose details (files removed, scope/elevation, protected detection)
  - [ ] `cmd/search.go` - Add verbose details (parameters, filters, counts)
  - [ ] `cmd/list.go` - Add verbose details (scan directories, parsed files, filters)
  - [ ] `cmd/info.go` - Add verbose details (lookup flow, source resolution)
  - [ ] `cmd/sources.go` - Add verbose details (update plan, per-source outcomes)

#### **UI Component Extraction**
- [ ] **Create reusable UI components**
  - [ ] Extract table components to `internal/components/table.go`
  - [ ] Extract form components to `internal/components/form.go`
  - [ ] Extract progress indicators to `internal/components/progress.go`
  - [ ] Extract confirmation dialogs to `internal/components/confirm.go`

#### **Evaluate Performance Optimisations** (LOW PRIORITY)
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
- [ ] Reusable UI components implemented
- [ ] Error handling standardized across all commands
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

**Current Status**: 5/7 commands fully standardized, 2 commands need error handling updates, all commands need visual consistency review.