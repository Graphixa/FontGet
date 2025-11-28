# FontGet Cleanup Tasks

This file contains actionable tasks for cleaning up the FontGet codebase. Each task includes specific file locations and implementation details.

> **⚠️ IMPORTANT**: This file was created before the major refactoring that eliminated `cmd/shared.go`. Many references have been updated, but some tasks may need review to ensure they're still relevant after the refactoring to `internal/shared/` and `internal/cmdutils/`.

---

## 🗑️ **PHASE 1: Remove Unused Functions (Inferior Code)**

### **✅ cmd/shared.go - REMOVED (Refactoring Complete)**
- [x] ~~`cmd/shared.go` has been completely eliminated~~ - All functions moved to `internal/shared/` and `internal/cmdutils/`
- [x] ~~Error types moved to `internal/shared/errors.go`~~
- [x] ~~Table functions moved to `internal/cmdutils/tables.go`~~

### **✅ Table Functions - MOVED to `internal/ui/tables.go`**
- [x] **All table functions moved** from `internal/cmdutils/tables.go` to `internal/ui/tables.go`
- [x] **All table constants moved** to `internal/ui/tables.go`
- [x] **All cmd files updated** to use `ui.Get*TableHeader()` and `ui.GetTableSeparator()`
- [x] **Unused functions removed**: `GetDynamicSearchTableHeader()`, `GetDynamicSearchTableSeparator()`, `GetSourcesTableHeader()`
- [x] **`internal/cmdutils/tables.go` deleted** - no longer needed
- [x] **Unified API**: All table functions now come from `ui` package for consistency

### **✅ internal/platform/testutil/ - REMOVED**
- [x] **Entire file deleted** - `internal/platform/testutil/testutil.go` removed (all functions unused)

### **✅ internal/functions/sort.go - REMOVED Unused Sorting Functions**
- [x] Removed `SortSourcesByEnabled()` function
- [x] Removed `SortSourcesByName()` function
- [x] Removed `SortSourcesByType()` function
- [x] Removed `ConvertManifestToSourceItems()` function

### **internal/templates/command_template.go - TEMPLATE FILE (No Action Needed)**
- [x] **Reviewed** `countTotalItems()` function - it's in a template file for developers to reference
- [x] **Decision**: Leave as-is - template files are meant to be examples, not production code

### **✅ internal/platform/ - REMOVED Inferior Font Analysis Functions**
- [x] Removed `AnalyzeFontComprehensively()` function and `ComprehensiveFontAnalysis` type
- [x] Removed `extractAllNameTableEntries()` function
- [x] Removed `parseNameTableComprehensive()` function
- [x] Removed `findNameTable()` function
- [x] Removed `extractFontProperties()` function
- [x] Removed `extractTechnicalDetails()` function

### **✅ internal/platform/windows_utils.go - REMOVED Unnecessary Windows Functions**
- [x] Removed `FindWindowEx()` function from `internal/platform/windows.go` (and variable declaration)
- [x] Removed `RunAsElevated()` function from `internal/platform/windows_utils.go`

### **✅ internal/repo/ - REMOVED Inferior Repository Functions**
- [x] **KEPT** `FetchURLContent()` - Used in `internal/license/license.go`
- [x] Removed `FontRepository` struct and `NewFontRepository()` function
- [x] Removed `FontRepository.GetFontInfo()` method
- [x] Removed `FontRepository.DownloadFont()` method (standalone `DownloadFont()` kept - it's used)
- [x] Removed `FontFileInfo` type (unused)
- [x] Removed `GetFont()` function
- [x] Removed `ListInstalledFonts()` function
- [x] Removed `GetAllFonts()` function (GetAllFontsCached is used instead)
- [x] Removed `GetFontInfo()` function from `internal/repo/sources.go`
- [x] Removed `GetFontFiles()` function from `internal/repo/sources.go`

### **✅ internal/components/card.go - REMOVED Inferior Card Functions**
- [x] Removed `AvailableFilesCard()` function
- [x] Removed `MetadataCard()` function
- [x] Removed `SourceInfoCard()` function
- [x] Removed `ValidationResultCard()` function

### **✅ internal/components/hierarchy.go - ALREADY REMOVED**
- [x] File does not exist - already removed in previous cleanup

---

## 🔧 **PHASE 2: Fix Inconsistencies**

### **✅ Table Display Consistency Fixes - COMPLETED**
- [x] **All table functions moved to `internal/ui/tables.go`**
- [x] **`cmd/sources.go` updated** - Now uses `ui.GetTableSeparator()` instead of `strings.Repeat("-", sepWidth)`
- [x] **All cmd files updated** - Now use unified `ui.Get*TableHeader()` and `ui.GetTableSeparator()` API
- [x] **Consistent API**: All table building now comes from single `ui` package
- [ ] **Note**: `cmd/search.go` line 290 still uses `strings.Repeat("-", totalWidth)` for dynamic tables - this is intentional for dynamic width calculation

---

## 🚀 **PHASE 3: Implement Superior Functions**

### **✅ Replace Basic Temp File Handling with Superior Functions - COMPLETED**
- [x] **Replaced** `filepath.Join(os.TempDir(), "Fontget", "fonts")` with `platform.GetTempFontsDir()` in `cmd/add.go`
  - Uses superior function with proper error handling
  - Creates temp directory if it doesn't exist
- [x] **Added** proper cleanup using `platform.CleanupTempFontsDir()` in `cmd/add.go` after font installation
  - Cleanup happens after all fonts are installed
  - Errors are logged but don't fail the installation
- [x] **Verified** no other temp file usage found - only `cmd/add.go` uses temp directories

---

## ✅ **PHASE 5: Progress Components Implementation (COMPLETED)**

### **Unified Progress Bar Implementation**
- [x] **Create reusable `OperationProgressModel` component** in `internal/components/operation_progress.go`
  - Supports real-time updates via Bubble Tea
  - Single progress bar for entire operation
  - Dynamic item list (items appear as completed)
  - Verbose mode: hierarchical variant display
  - Status indicators: ✓ ⏳ ○ ✗

- [x] **Implement unified progress in `cmd/add.go`**
  - Replace per-font progress with single operation progress
  - Show "Downloading and Installing Fonts (X of Y)"
  - Real-time updates as fonts complete
  - Verbose: Show variant hierarchy with indentation

- [x] **Implement unified progress in `cmd/remove.go`**
  - Add progress indication (was missing)
  - Show "Finding and Removing Fonts (X of Y)"
  - Real-time updates as fonts are removed
  - Verbose: Show removed file list with indentation

- [x] **Establish verbose vs debug messaging guidelines**
  - Create `docs/maintenance/logging-guidelines.md`
  - Comprehensive guidelines for message classification
  - Clear criteria: verbose for user info, debug for technical details

### **Visual Design Implementation**
- [x] **Single overall progress bar** for both add and remove commands
- [x] **Real-time updates** showing current operation status
- [x] **Hierarchical variant display** in verbose mode
- [x] **Status indicators** for completed, in-progress, pending, and failed items
- [x] **Consistent visual style** with existing non-card design

### **Files Modified**
- [x] `internal/components/operation_progress.go` (new)
- [x] `internal/ui/styles.go` (added OperationTitle style)
- [x] `cmd/add.go` (replaced per-font progress with unified progress)
- [x] `cmd/remove.go` (added unified progress bar)
- [x] `docs/maintenance/logging-guidelines.md` (new)

---

## 🧪 **TESTING CRITERIA**

### **Phase 1 Tests - Function Removal**
- [ ] **Test 1.1**: Run `go build` - should compile without errors
- [ ] **Test 1.2**: Run `go test ./...` - all existing tests should pass
- [ ] **Test 1.3**: Run `staticcheck ./...` - should show no unused function warnings for removed functions
- [ ] **Test 1.4**: Run `deadcode ./...` - should show reduced number of unreachable functions

### **Phase 2 Tests - Consistency Fixes**
- [ ] **Test 2.1**: Run `fontget sources info` - table separator should be consistent with other commands
- [ ] **Test 2.2**: Run `fontget search "test"` - table separator should be consistent with other commands
- [ ] **Test 2.3**: Visual inspection - all table separators should look identical across commands

### **Phase 3 Tests - Superior Function Implementation**
- [ ] **Test 3.1**: Run `fontget add "test-font"` - should use proper temp directory management
- [ ] **Test 3.2**: Check temp directory cleanup - no leftover temp files after font installation
- [ ] **Test 3.3**: Run with `--debug` flag - should show proper temp directory paths

### **Phase 4 Tests - UI Component Implementation**
- [ ] **Test 4.1**: Run `fontget list --full` - should display hierarchical font family structure
- [ ] **Test 4.2**: Run `fontget sources manage` - should show confirmation dialogs for changes
- [ ] **Test 4.3**: Run `fontget add "large-font"` - should show progress indication during download
- [ ] **Test 4.4**: Run `fontget config edit` - should open interactive form (if implemented)

### **Phase 5 Tests - Progress Components Implementation**
- [x] **Test 5.1**: Run `fontget add "test-font"` - should show unified progress bar
- [x] **Test 5.2**: Run `fontget add "test-font" --verbose` - should show variant hierarchy
- [x] **Test 5.3**: Run `fontget remove "test-font"` - should show unified progress bar
- [x] **Test 5.4**: Run `fontget remove "test-font" --verbose` - should show removed file hierarchy
- [x] **Test 5.5**: Test with multiple fonts - should show real-time updates
- [x] **Test 5.6**: Test with failures - should show ✗ status correctly
- [x] **Test 5.7**: Test with skipped fonts - should show appropriate status
- [x] **Test 5.8**: Visual consistency - add and remove commands should have similar format

### **Integration Tests**
- [ ] **Test I.1**: Run all commands with `--help` - should work without errors
- [ ] **Test I.2**: Run `fontget add "test"` followed by `fontget list` - should show installed font
- [ ] **Test I.3**: Run `fontget remove "test"` - should remove font successfully
- [ ] **Test I.4**: Run `fontget sources update` - should update sources without errors

### **Performance Tests**
- [ ] **Test P.1**: Measure startup time - should be same or better than before cleanup
- [ ] **Test P.2**: Measure memory usage - should be same or better than before cleanup
- [ ] **Test P.3**: Run font installation - should be same or better performance

### **Code Quality Tests**
- [ ] **Test Q.1**: Run `gofmt -d .` - should show no formatting issues
- [ ] **Test Q.2**: Run `golint ./...` - should show no new linting issues
- [ ] **Test Q.3**: Run `go vet ./...` - should show no vet issues
- [ ] **Test Q.4**: Check code coverage - should maintain or improve coverage

---

## 📋 **SUCCESS CRITERIA**

### **Phase 1 Success Criteria**
- ✅ All unused functions removed without breaking existing functionality
- ✅ Code compiles and all tests pass
- ✅ No new linting errors introduced
- ✅ Reduced codebase size by ~300-500 lines

### **Phase 2 Success Criteria**
- ✅ All table separators use consistent `GetTableSeparator()` function
- ✅ Visual consistency across all command outputs
- ✅ No functional changes to user experience

### **Phase 3 Success Criteria**
- ✅ Superior temp file management implemented
- ✅ Better error recovery and cleanup
- ✅ Improved reliability of font installation process

### **Phase 4 Success Criteria**
- ✅ Enhanced user experience with new UI components
- ✅ Hierarchical font display working correctly
- ✅ Interactive forms and confirmations working
- ✅ Progress indicators providing better feedback

### **Overall Success Criteria**
- ✅ Codebase is cleaner and more maintainable
- ✅ No regression in existing functionality
- ✅ Improved user experience where components are implemented
- ✅ Better code organization and reusability
- ✅ All tests passing and code quality maintained

---

## 🚨 **ROLLBACK PLAN**

If any phase introduces issues:
1. **Phase 1**: Revert specific function removals that cause compilation errors
2. **Phase 2**: Revert table separator changes if they break formatting
3. **Phase 3**: Revert temp file changes if they cause installation issues
4. **Phase 4**: Disable new UI components if they cause runtime errors

Each phase should be tested independently before proceeding to the next phase.
