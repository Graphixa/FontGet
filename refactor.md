# FontGet Refactor Checklist

## 🎯 **High Priority Fixes**

### **1. Critical Code Issues (Immediate Fixes)**
- [ ] **Fix main.go error handling**
  - Currently has empty error handling: `if err := cmd.Execute(); err != nil { // handle error, e.g. print and exit }`
  - Add proper error handling and exit codes

- [ ] **Fix Windows registry code complexity**
  - `addFontToRegistry()` and `removeFontFromRegistry()` are overly complex
  - Simplify registry operations with proper error handling
  - Add rollback mechanism for failed operations

- [ ] **Fix search algorithm inefficiency**
  - Replace bubble sort with Go's built-in `sort.Slice()`
  - Optimize string comparison operations
  - Add search result caching

### **2. Command Output Cleanup**
- [ ] **Remove debug output from add command**
  - Remove "Found 7044 fonts in manifest" message
  - Remove individual font name headers when no match found
  - Only show status report when there are actual operations (installed/skipped/failed > 0)

- [ ] **Fix search command caching behavior**
  - Confirm sources are NOT updated on every search run
  - Ensure 24-hour cache is working properly
  - Add manual cache refresh option

- [ ] **Fix remove command output issues**
  - Remove "Found 7044 fonts in manifest" debug message
  - Remove individual font name headers when no match found
  - Only show status report when there are actual operations (removed/skipped/failed > 0)

### **3. Sources Command Improvements**
- [ ] **Implement Bubble Tea TUI for sources management**
  - Create `cmd/sources_manage.go` with Bubble Tea interface
  - Add table layout with proper headers
  - Implement navigation (arrow keys, space to toggle, etc.)
  - Add form interface for adding/editing sources
  - Use Bubble Tea table component for better layout

- [ ] **Remove redundant sources subcommands**
  - Remove `fontget sources list` subcommand
  - Enhance `fontget sources info` with useful information:
    - Last updated sources date
    - Source file locations
    - Cache status and size
    - Source validation status

### **4. Command Alias Standardization**
- [ ] **Fix alias conflicts in list command**
  - Change `--family` alias from `-f` to `-a` (conflicts with `--force`)
  - Keep `-s` for `--scope` across all commands
  - Keep `-t` for `--type` in list command
  - Keep `-c` for `--category` in search command

- [ ] **Add missing aliases**
  - Add `-s` alias for `--scope` in all commands that use it
  - Add `-f` alias for `--force` in add command
  - Ensure consistent alias usage across all commands

## 🔧 **Code Quality Improvements**

### **5. Extract Common Patterns**
- [ ] **Create shared command utilities**
  - Extract auto-scope detection logic into `cmd/shared.go`
  - Extract font name processing logic
  - Extract status reporting functions
  - Extract error handling patterns
  - Extract color function creation (green, red, yellow, etc.)

- [ ] **Refactor large command functions**
  - Break down `remove.go` RunE function (600+ lines)
  - Break down `add.go` RunE function (300+ lines)
  - Extract font processing loops into separate functions
  - Extract scope handling logic
  - Extract font family file finding logic
  - Extract similar font suggestion logic

### **6. Data Structure Consolidation**
- [ ] **Consolidate font data structures**
  - Merge `FontData`, `FontInfo`, `Font`, and `FontFileInfo` into single structure
  - Remove legacy compatibility structures
  - Update all references to use consolidated structure

- [ ] **Simplify type definitions**
  - Remove duplicate struct definitions
  - Create clear inheritance hierarchy
  - Add proper validation methods

### **7. Error Handling Standardization**
- [ ] **Create custom error types**
  - `FontNotFoundError` with suggestions
  - `FontInstallationError` with details
  - `FontRemovalError` with details
  - `ConfigurationError` with hints

- [ ] **Standardize error patterns**
  - Consistent error wrapping with context
  - Consistent logging patterns
  - Consistent user-facing error messages

## 🚀 **Performance & Algorithm Improvements**

### **8. Search Algorithm Optimization**
- [ ] **Replace manual sorting with Go's built-in sort**
  - Use `sort.Slice()` instead of bubble sort
  - Implement proper comparison functions
  - Add performance benchmarks

- [ ] **Optimize search matching**
  - Implement fuzzy matching for better results
  - Add search result caching
  - Optimize string comparison operations

### **9. Caching System Enhancement**
- [ ] **Implement proper cache validation**
  - Add cache integrity checks
  - Implement cache corruption recovery
  - Add cache size monitoring

- [ ] **Add cache management commands**
  - `fontget cache clear` - Clear all cached data
  - `fontget cache status` - Show cache statistics
  - `fontget cache validate` - Validate cache integrity

## 🏗️ **Architecture Improvements**

### **10. Configuration System Cleanup**
- [ ] **Choose single configuration format**
  - Decide between JSON and YAML
  - Complete migration to chosen format
  - Remove unused configuration code

- [ ] **Simplify configuration loading**
  - Consolidate configuration loading logic
  - Add configuration validation
  - Implement configuration migration

### **11. Platform-Specific Code Refactoring**
- [ ] **Improve Windows registry operations**
  - Consider using higher-level libraries instead of raw syscalls
  - Add better error handling for registry operations
  - Implement registry operation rollback
  - Simplify complex registry manipulation code in `windows.go`
  - Add proper UTF-16 string handling

- [ ] **Standardize platform interfaces**
  - Ensure consistent behavior across platforms
  - Add platform-specific validation
  - Improve cross-platform testing
  - Standardize font directory detection
  - Standardize elevation checking

- [ ] **Fix Windows-specific issues**
  - Simplify `addFontToRegistry()` and `removeFontFromRegistry()` functions
  - Add proper error handling for Windows API calls
  - Implement proper cleanup on registry operation failures

## 📋 **Missing Features Implementation**

### **12. Archive Handling (Critical Missing Feature)**
- [ ] **Implement ZIP extraction for Font Squirrel**
  - Add ZIP extraction functionality in `internal/platform/`
  - Handle font file extraction from ZIP archives
  - Add cleanup for temporary extracted files
  - Update font installation logic to handle extracted files

- [ ] **Implement TAR.XZ extraction for Nerd Fonts**
  - Add TAR.XZ extraction functionality in `internal/platform/`
  - Handle font file extraction from TAR.XZ archives
  - Add cleanup for temporary extracted files
  - Update font installation logic to handle extracted files

- [ ] **Update font file type detection**
  - Modify `GetFont()` to handle new variant structure with file type keys
  - Implement file type detection based on source and variant files
  - **Google Fonts**: Direct TTF/OTF files (no extraction needed)
  - **Font Squirrel**: ZIP archives (extract before installation)
  - **Nerd Fonts**: TAR.XZ archives (extract before installation)

### **13. Installation Tracking (Critical Missing Feature)**
- [ ] **Add installation tracking system**
  - Create `~/.fontget/installations.json` with proper schema
  - Track font installations with metadata (font_id, source, installed date, scope)
  - Add installation history commands
  - Update `cmd/list.go` to show source information for installed fonts
  - Add filtering by source option

- [ ] **Implement font export/import**
  - `fontget export` - Export installation list
  - `fontget import` - Import installation list
  - Add cross-platform compatibility
  - Support both user and machine scope exports

### **14. Complete Missing Command Updates**
- [ ] **Update `cmd/info.go`**
  - Modify `infoCmd` to display enhanced metadata
  - Add support for clean font IDs
  - Show source information and variant details
  - Display unicode ranges, languages, and sample text

- [ ] **Update `cmd/list.go`**
  - Modify `listCmd` to show source information for installed fonts
  - Add filtering by source option
  - Update installation tracking format

## 🧪 **Testing & Quality Assurance**

### **15. Add Comprehensive Testing**
- [ ] **Unit tests for core functions**
  - Test font processing logic
  - Test configuration loading
  - Test platform-specific operations

- [ ] **Integration tests for commands**
  - Test complete command workflows
  - Test error scenarios
  - Test cross-platform compatibility

### **16. Code Documentation**
- [ ] **Add comprehensive documentation**
  - Document all public functions
  - Add package-level documentation
  - Create architecture documentation

- [ ] **Improve inline comments**
  - Add comments for complex logic
  - Document platform-specific code
  - Add TODO comments for future improvements

## 📊 **Monitoring & Metrics**

### **17. Add Performance Monitoring**
- [ ] **Implement performance metrics**
  - Track command execution times
  - Monitor memory usage
  - Track cache hit rates

- [ ] **Add diagnostic commands**
  - `fontget diagnose` - System health check
  - `fontget stats` - Usage statistics
  - `fontget version` - Version and build info

## 🎨 **User Experience Enhancements**

### **18. Improve Command Help**
- [ ] **Standardize help text across commands**
  - Consistent description format
  - Consistent example format
  - Consistent flag descriptions

- [ ] **Add interactive help**
  - `fontget help <command>` - Detailed command help
  - `fontget examples` - Usage examples
  - `fontget tutorial` - Interactive tutorial

### **19. Enhanced Output Formatting**
- [ ] **Improve table formatting**
  - Consistent column widths
  - Better alignment
  - Color coding improvements

- [ ] **Add progress indicators**
  - Show download progress
  - Show installation progress
  - Show cache refresh progress

## 🔍 **Code Review Checklist**

### **20. Code Quality Review**
- [ ] **Remove dead code**
  - Remove unused functions
  - Remove unused variables
  - Remove commented-out code

- [ ] **Fix code style issues**
  - Consistent naming conventions
  - Consistent formatting
  - Consistent error handling

- [ ] **Add input validation**
  - Validate all user inputs
  - Add bounds checking
  - Add format validation

## 📝 **Documentation Updates**

### **21. Update Documentation**
- [ ] **Update README.md**
  - Reflect new features
  - Update installation instructions
  - Add troubleshooting section

- [ ] **Create migration guide**
  - Document breaking changes
  - Provide upgrade instructions
  - Add compatibility notes

### **22. Create Developer Documentation**
- [ ] **Add development setup guide**
  - Environment setup
  - Build instructions
  - Testing instructions

- [ ] **Create contribution guidelines**
  - Code style guide
  - Pull request process
  - Issue reporting guidelines

---

## 🎯 **Implementation Priority**

1. **Week 1**: Critical code issues, command output cleanup, alias fixes
2. **Week 2**: Sources TUI implementation, common pattern extraction
3. **Week 3**: Data structure consolidation, error handling standardization
4. **Week 4**: Performance improvements, missing features (archive handling, installation tracking)
5. **Week 5**: Testing, documentation, final polish

## 📋 **Task Dependencies**

- **Before starting**: Complete all "Critical Code Issues" (Section 1)
- **Before data structure consolidation**: Complete "Extract Common Patterns" (Section 5)
- **Before missing features**: Complete "Data Structure Consolidation" (Section 6)
- **Before testing**: Complete all code quality improvements (Sections 5-11)

## ✅ **Success Criteria**

- [ ] All commands have consistent behavior and output
- [ ] Code duplication reduced by 80%
- [ ] Command functions under 100 lines each
- [ ] All tests passing
- [ ] Documentation complete and up-to-date
- [ ] Performance improved by 50%
- [ ] User experience significantly enhanced
