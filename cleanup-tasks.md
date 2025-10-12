# FontGet Cleanup Tasks

This file contains actionable tasks for cleaning up the FontGet codebase. Each task includes specific file locations and implementation details.

---

## 🗑️ **PHASE 1: Remove Unused Functions (Inferior Code)**

### **cmd/shared.go - Remove Unused Functions**
- [ ] Remove `GetColorFunctions()` function from `cmd/shared.go`
- [ ] Remove `FontNotFoundError.Error()` method from `cmd/shared.go`
- [ ] Remove `FontInstallationError.Error()` method from `cmd/shared.go`
- [ ] Remove `FontRemovalError.Error()` method from `cmd/shared.go`
- [ ] Remove `ConfigurationError.Error()` method from `cmd/shared.go`
- [ ] Remove `ElevationError.Error()` method from `cmd/shared.go`
- [ ] Remove `GetDynamicSearchTableHeader()` function from `cmd/shared.go`
- [ ] Remove `GetDynamicSearchTableSeparator()` function from `cmd/shared.go`
- [ ] Remove `GetSourcesTableHeader()` function from `cmd/shared.go`

### **internal/platform/testutil/ - Remove Test Utilities**
- [ ] Remove `CreateTestFont()` function from `internal/platform/testutil/testutil.go`
- [ ] Remove `CleanupTestFont()` function from `internal/platform/testutil/testutil.go`
- [ ] Remove `CreateTestFontDir()` function from `internal/platform/testutil/testutil.go`
- [ ] Remove `CleanupTestFontDir()` function from `internal/platform/testutil/testutil.go`

### **internal/functions/sort.go - Remove Unused Sorting Functions**
- [ ] Remove `SortSourcesByEnabled()` function from `internal/functions/sort.go`
- [ ] Remove `SortSourcesByName()` function from `internal/functions/sort.go`
- [ ] Remove `SortSourcesByType()` function from `internal/functions/sort.go`
- [ ] Remove `ConvertManifestToSourceItems()` function from `internal/functions/sort.go`

### **internal/templates/command_template.go - LEAVE ALONE OR MAKE IT MAKE SENSE AS IT'S A TEMPLATE**
- [ ] Look at `countTotalItems()` function from `internal/templates/command_template.go`

### **internal/platform/ - Remove Inferior Font Analysis Functions**
- [ ] Remove `AnalyzeFontComprehensively()` function from `internal/platform/`
- [ ] Remove `extractAllNameTableEntries()` function from `internal/platform/`
- [ ] Remove `parseNameTableComprehensive()` function from `internal/platform/`
- [ ] Remove `findNameTable()` function from `internal/platform/`
- [ ] Remove `extractFontProperties()` function from `internal/platform/`
- [ ] Remove `extractTechnicalDetails()` function from `internal/platform/`

### **internal/platform/windows_utils.go - Remove Unnecessary Windows Functions**
- [ ] Remove `FindWindowEx()` function from `internal/platform/windows_utils.go`
- [ ] Remove `RunAsElevated()` function from `internal/platform/windows_utils.go`

### **internal/repo/ - Remove Inferior Repository Functions**
- [ ] Remove `FetchURLContent()` function from `internal/repo/`
- [ ] Remove `NewFontRepository()` function from `internal/repo/`
- [ ] Remove `FontRepository.GetFontInfo()` method from `internal/repo/`
- [ ] Remove `FontRepository.DownloadFont()` method from `internal/repo/`
- [ ] Remove `GetFont()` function from `internal/repo/`
- [ ] Remove `ListInstalledFonts()` function from `internal/repo/`
- [ ] Remove `GetAllFonts()` function from `internal/repo/`
- [ ] Remove `GetFontInfo()` function from `internal/repo/`
- [ ] Remove `GetFontFiles()` function from `internal/repo/`

### **internal/components/card.go - Remove Inferior Card Functions**
- [ ] Remove `AvailableFilesCard()` function from `internal/components/card.go`
- [ ] Remove `MetadataCard()` function from `internal/components/card.go`
- [ ] Remove `SourceInfoCard()` function from `internal/components/card.go`
- [ ] Remove `ValidationResultCard()` function from `internal/components/card.go`

### **internal/components/hierarchy.go - Remove Unused Hierarchy Functions**
- [ ] Remove entire `internal/components/hierarchy.go` file (12 unused functions)
  - **Functions**: `NewHierarchyItem()`, `NewHierarchyModel()`, `AddChild()`, `AddChildren()`, `Render()`, `SetWidth()`, `SetShowDetails()`, `ToggleItem()`, `toggleItemRecursive()`, `getDefaultItemStyle()`, `getChildItemStyle()`, `FontFamilyItem()`, `getFontFamilyStyle()`, `getFontVariantStyle()`, `CreateFontHierarchy()`, `RenderHierarchy()`
  - **Reason**: No value for sources (no variants), current table display is better

---

## 🔧 **PHASE 2: Fix Inconsistencies**

### **Table Display Consistency Fixes**
- [ ] Replace `strings.Repeat("-", sepWidth)` with `GetTableSeparator()` in `cmd/sources.go` line 160
  - **Note**: `GetTableSeparator()` is already defined in `cmd/shared.go` (line 436) and available in same package
  - **No import needed**: Function is in same `cmd` package
- [ ] Replace `strings.Repeat("-", totalWidth)` with `GetTableSeparator()` in `cmd/search.go` line 270
  - **Note**: `GetTableSeparator()` is already defined in `cmd/shared.go` (line 436) and available in same package
  - **No import needed**: Function is in same `cmd` package
  - **Status**: `cmd/search.go` already uses `GetTableSeparator()` on line 215, so this task may already be complete

---

## 🚀 **PHASE 3: Implement Superior Functions**

### **Replace Basic Temp File Handling with Superior Functions**
- [ ] Replace `filepath.Join(os.TempDir(), "Fontget", "fonts")` with `GetTempFontsDir()` in `cmd/add.go` line 438
  - **Note**: `GetTempFontsDir()` is defined in `internal/platform/temp.go` (line 34)
  - **Import status**: `"fontget/internal/platform"` is already imported in `cmd/add.go` (line 13)
- [ ] Add proper cleanup using `CleanupTempFontsDir()` in `cmd/add.go` after font installation
  - **Note**: `CleanupTempFontsDir()` is defined in `internal/platform/temp.go` (line 66)
  - **Import status**: Same `"fontget/internal/platform"` import already exists
- [ ] Update any other temp file usage to use `GetTempDir()` and `GetTempFontsDir()` functions
  - **Note**: Both functions are in `internal/platform/temp.go`
  - **Import needed**: `"fontget/internal/platform"` where these functions are used

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
