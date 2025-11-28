# Code Quality Assessment and Tracking

This document tracks code quality improvements identified during the assessment phase. Items are organized by priority and category.

## Assessment Date
2025-01-XX

---

## 1. Long Functions (Should be Broken Down)

### High Priority

#### `cmd/add.go`
- [x] **`installFontsInDebugMode`** (lines ~717-833) ✅ **COMPLETED**
  - **Issue**: ~116 lines, handles multiple responsibilities
  - **Breakdown**:
    1. **`processInstallResult`** (in `cmd/add.go`)
       - **Purpose**: Process and categorize install result details (installed/skipped/failed variants)
       - **Signature**: `func processInstallResult(result *InstallResult) (installedFiles, skippedFiles, failedFiles []string)`
       - **Placement**: `cmd/add.go` (command-specific logic)
    2. **`logInstallResultDetails`** (in `cmd/add.go`)
       - **Purpose**: Log detailed variant information in debug mode
       - **Signature**: `func logInstallResultDetails(result *InstallResult, fontName, scopeLabel string)`
       - **Placement**: `cmd/add.go` (command-specific logging)
    3. **`updateInstallStatus`** (in `cmd/add.go`)
       - **Purpose**: Update installation status from result
       - **Signature**: `func updateInstallStatus(status *InstallationStatus, result *InstallResult)`
       - **Placement**: `cmd/add.go` (command-specific status tracking)
  - **Target**: Break into 3 helper functions + main function (~40 lines each)
  - **Priority**: HIGH (affects maintainability)

- [x] **`installFont`** (lines ~835-999) ✅ **COMPLETED**
  - **Issue**: ~165 lines, complex logic with multiple responsibilities
  - **Breakdown**:
    1. **`checkFontAlreadyInstalled`** (in `cmd/shared.go`)
       - **Purpose**: Check if font is already installed (reusable across commands)
       - **Signature**: `func checkFontAlreadyInstalled(fontID, fontName string, installScope platform.InstallationScope, fontManager platform.FontManager, force bool) (bool, error)`
       - **Placement**: `cmd/shared.go` (shared utility, used by add and import)
    2. **`downloadFontVariants`** (in `cmd/add.go`)
       - **Purpose**: Download all variants of a font family
       - **Signature**: `func downloadFontVariants(fontFiles []repo.FontFile, tempDir string) ([]string, error)`
       - **Placement**: `cmd/add.go` (command-specific download logic)
    3. **`installDownloadedFonts`** (in `cmd/add.go`)
       - **Purpose**: Install downloaded font files to system
       - **Signature**: `func installDownloadedFonts(fontPaths []string, fontManager platform.FontManager, installScope platform.InstallationScope, fontDir string) (installed, skipped, failed int, details []string, errors []string)`
       - **Placement**: `cmd/add.go` (command-specific installation logic)
    4. **`buildInstallResult`** (in `cmd/add.go`)
       - **Purpose**: Build InstallResult from installation outcomes
       - **Signature**: `func buildInstallResult(status string, message string, installed, skipped, failed int, details []string, errors []string) *InstallResult`
       - **Placement**: `cmd/add.go` (command-specific result building)
  - **Target**: Break into 4 helper functions + main function (~40 lines each)
  - **Priority**: HIGH (core functionality, hard to test)

#### `cmd/remove.go`
- [x] **`removeFontsInDebugMode`** (lines ~800-1170) ✅ **COMPLETED**
  - **Issue**: ~370 lines, very long function handling multiple scopes and operations
  - **Breakdown**:
    1. **`processSingleScopeRemoval`** (in `cmd/remove.go`)
       - **Purpose**: Process removal for a single scope (single-scope mode)
       - **Signature**: `func processSingleScopeRemoval(foundFonts []FoundFontInfo, fontManager platform.FontManager, scope platform.InstallationScope, fontDir string, r *repo.Repository, isCriticalSystemFont func(string) bool, status *RemovalStatus) error`
       - **Placement**: `cmd/remove.go` (command-specific logic)
    2. **`processMultiScopeRemoval`** (in `cmd/remove.go`)
       - **Purpose**: Process removal for multiple scopes (all-scope mode)
       - **Signature**: `func processMultiScopeRemoval(fontScopeItems []FontScopeItem, fontManager platform.FontManager, r *repo.Repository, isCriticalSystemFont func(string) bool, status *RemovalStatus) error`
       - **Placement**: `cmd/remove.go` (command-specific logic)
    3. **`buildRemovalStatusMessage`** (in `cmd/remove.go`)
       - **Purpose**: Build status message from removal result
       - **Signature**: `func buildRemovalStatusMessage(result *RemoveResult, err error) (status, message string)`
       - **Placement**: `cmd/remove.go` (command-specific formatting)
    4. **`updateRemovalStatus`** (in `cmd/remove.go`)
       - **Purpose**: Update removal status from result
       - **Signature**: `func updateRemovalStatus(status *RemovalStatus, result *RemoveResult, err error)`
       - **Placement**: `cmd/remove.go` (command-specific status tracking)
  - **Target**: Break into 4 helper functions + main function (~80 lines each)
  - **Priority**: HIGH (very long, affects readability)

- [x] **`removeFont`** (lines ~900-1100) ✅ **COMPLETED**
  - **Issue**: ~200 lines, complex removal logic
  - **Breakdown**:
    1. **`findFontFilesForRemoval`** (in `cmd/remove.go`)
       - **Purpose**: Find all font files matching the font name
       - **Signature**: `func findFontFilesForRemoval(fontName string, fontManager platform.FontManager, scope platform.InstallationScope, fontDir string, r *repo.Repository, isCriticalSystemFont func(string) bool) ([]string, error)`
       - **Placement**: `cmd/remove.go` (command-specific file finding)
    2. **`removeFontFiles`** (in `cmd/remove.go`)
       - **Purpose**: Remove font files from system
       - **Signature**: `func removeFontFiles(fontPaths []string, fontManager platform.FontManager, scope platform.InstallationScope) (removed, skipped, failed int, details []string, errors []string)`
       - **Placement**: `cmd/remove.go` (command-specific removal logic)
    3. **`buildRemoveResult`** (in `cmd/remove.go`)
       - **Purpose**: Build RemoveResult from removal outcomes
       - **Signature**: `func buildRemoveResult(status string, message string, removed, skipped, failed int, details []string, errors []string) *RemoveResult`
       - **Placement**: `cmd/remove.go` (command-specific result building)
  - **Target**: Break into 3 helper functions + main function (~50 lines each)
  - **Priority**: MEDIUM (long but manageable)

#### `cmd/sources_manage.go`
- [ ] **`Update` method** (lines ~108-133)
  - **Issue**: Multiple state handlers, could be cleaner
  - **Recommendation**: Already delegates to updateList/updateForm, but could extract state routing
  - **Target**: Minor refactoring
  - **Priority**: LOW (already well-structured)

### Medium Priority

#### `cmd/export.go`
- [x] **`performFullExportWithResult`** (lines ~288-554) ✅ **COMPLETED**
  - **Issue**: ~266 lines, handles collection, matching, filtering, and export
  - **Breakdown**:
    1. **`matchInstalledFontsToRepository`** (in `cmd/shared.go`)
       - **Purpose**: Match installed fonts to repository entries (reusable for list/export)
       - **Signature**: `func matchInstalledFontsToRepository(familyNames []string) (map[string]*repo.InstalledFontMatch, error)`
       - **Placement**: `cmd/shared.go` (shared utility, used by list and export)
    2. **`populateFontMatchData`** (in `cmd/export.go`)
       - **Purpose**: Populate ParsedFont structs with match data
       - **Signature**: `func populateFontMatchData(families map[string][]ParsedFont, matches map[string]*repo.InstalledFontMatch)`
       - **Placement**: `cmd/export.go` (command-specific data population)
    3. **`filterFontsForExport`** (in `cmd/export.go`)
       - **Purpose**: Apply match/source/exportAll filters to font families
       - **Signature**: `func filterFontsForExport(families map[string][]ParsedFont, names []string, matchFilter, sourceFilter string, exportAll, onlyMatched bool) (map[string][]ParsedFont, skippedSystem, skippedUnmatched, skippedByFilter int)`
       - **Placement**: `cmd/export.go` (command-specific filtering)
    4. **`buildExportManifest`** (in `cmd/export.go`)
       - **Purpose**: Build export manifest from filtered fonts
       - **Signature**: `func buildExportManifest(fontIDGroups map[string]*fontIDGroup, totalVariants int) *ExportManifest`
       - **Placement**: `cmd/export.go` (command-specific manifest building)
  - **Target**: Break into 4 helper functions + main function (~60 lines each)
  - **Priority**: MEDIUM (long but clear flow)

#### `cmd/backup.go`
- [x] **`performBackupWithProgress`** (lines ~550-650) ✅ **COMPLETED**
  - **Issue**: ~100 lines, handles organization and zip creation
  - **Breakdown**:
    1. **`organizeFontsBySourceAndFamily`** (in `cmd/backup.go`)
       - **Purpose**: Organize fonts by source and family name for zip structure
       - **Signature**: `func organizeFontsBySourceAndFamily(fonts []ParsedFont, matches map[string]*repo.InstalledFontMatch) (map[string]map[string][]fontFileInfo, error)`
       - **Placement**: `cmd/backup.go` (command-specific organization)
    2. **`createBackupZipArchive`** (in `cmd/backup.go`)
       - **Purpose**: Create zip archive from organized font structure
       - **Signature**: `func createBackupZipArchive(sourceFamilyMap map[string]map[string][]fontFileInfo, zipPath string) (familyCount, fileCount int, err error)`
       - **Placement**: `cmd/backup.go` (command-specific zip creation)
  - **Target**: Break into 2 helper functions + main function (~50 lines each)
  - **Priority**: MEDIUM (moderate length)

---

## 2. Duplicate Code Patterns

### High Priority

#### Error Handling Pattern
- [x] **Standardize error handling across all commands** ✅ **COMPLETED** (via ensureManifestInitialized and createFontManager with inlined logging)
  - **Issue**: Some commands have inconsistent error handling patterns
  - **Current Pattern** (from shared.go):
    ```go
    if err != nil {
        GetLogger().Error("Failed to ...: %v", err)
        output.GetVerbose().Error("%v", err)
        output.GetDebug().Error("functionName() failed: %v", err)
        return fmt.Errorf("user-friendly message: %v", err)
    }
    ```
  - **Function to create**:
    - **Name**: `handleError`
    - **Signature**: `func handleError(err error, functionName string, userMessage string) error`
    - **Placement**: `cmd/shared.go` (shared utility for all commands)
    - **Purpose**: Standardize error logging and user-facing error messages
  - **Files to update**: All command files
  - **Priority**: HIGH (affects consistency)

#### Manifest Initialization
- [x] **Extract `config.EnsureManifestExists()` pattern** ✅ **COMPLETED**
  - **Issue**: Repeated error handling pattern in every command
  - **Current Pattern**:
    ```go
    if err := config.EnsureManifestExists(); err != nil {
        GetLogger().Error("Failed to ensure manifest exists: %v", err)
        output.GetVerbose().Error("%v", err)
        output.GetDebug().Error("config.EnsureManifestExists() failed: %v", err)
        return fmt.Errorf("unable to load font repository: %v", err)
    }
    ```
  - **Function to create**:
    - **Name**: `ensureManifestInitialized`
    - **Signature**: `func ensureManifestInitialized() error`
    - **Placement**: `cmd/shared.go` (shared utility for all commands)
    - **Purpose**: Ensure manifest exists with standardized error handling
  - **Files to update**: add.go, remove.go, list.go, search.go, info.go, export.go, import.go, backup.go
  - **Priority**: HIGH (reduces duplication across 8 files)

#### Font Manager Creation
- [x] **Extract font manager creation pattern** ✅ **COMPLETED**
  - **Issue**: Repeated error handling in multiple commands
  - **Current Pattern**:
    ```go
    fm, err := platform.NewFontManager()
    if err != nil {
        GetLogger().Error("Failed to create font manager: %v", err)
        output.GetVerbose().Error("%v", err)
        output.GetDebug().Error("platform.NewFontManager() failed: %v", err)
        return fmt.Errorf("unable to access system fonts: %v", err)
    }
    ```
  - **Function to create**:
    - **Name**: `createFontManager`
    - **Signature**: `func createFontManager() (platform.FontManager, error)`
    - **Placement**: `cmd/shared.go` (shared utility for commands that need font manager)
    - **Purpose**: Create font manager with standardized error handling
  - **Files to update**: list.go, export.go, backup.go, remove.go
  - **Priority**: MEDIUM (affects 4 files)

### Medium Priority

#### Scope Detection/Auto-detection
- [x] **Extract scope auto-detection logic** ✅ **COMPLETED**
  - **Issue**: Similar logic in add.go and remove.go
  - **Current Pattern**: Auto-detect scope based on elevation
  - **Function to create**:
    - **Name**: `autoDetectScope`
    - **Signature**: `func autoDetectScope(fontManager platform.FontManager, defaultScope string) (string, error)`
    - **Placement**: `cmd/shared.go` (shared utility for add and remove commands)
    - **Purpose**: Auto-detect installation scope based on elevation status
  - **Files to update**: add.go, remove.go
  - **Priority**: MEDIUM (affects 2 files)

#### Repository Initialization
- [x] **Extract repository initialization pattern** ✅ **COMPLETED**
  - **Issue**: Similar pattern in search.go and info.go
  - **Current Pattern**: Get repository with optional refresh
  - **Function to create**:
    - **Name**: `getRepository`
    - **Signature**: `func getRepository(refresh bool) (*repo.Repository, error)`
    - **Placement**: `cmd/shared.go` (shared utility for search and info commands)
    - **Purpose**: Get repository with optional refresh, standardized error handling
  - **Files to update**: search.go, info.go
  - **Priority**: LOW (affects 2 files, pattern is clear)

---

## 3. Complex Conditional Logic

### High Priority

#### `cmd/list.go`
- [x] **Filtering logic** (lines ~277-314) ✅ **COMPLETED**
  - **Issue**: Complex nested conditionals for filtering by family, type, and Font ID
  - **Breakdown**:
    1. **`filterFontsByFamilyAndID`** (in `cmd/list.go`)
       - **Purpose**: Filter font families by family name or Font ID
       - **Signature**: `func filterFontsByFamilyAndID(families map[string][]ParsedFont, familyFilter string) map[string][]ParsedFont`
       - **Placement**: `cmd/list.go` (command-specific filtering, but could be shared if export uses similar logic)
       - **Note**: Type filtering already handled in `collectFonts`, so this only handles family/ID filtering
  - **Target**: Extract into 1 helper function (~40 lines)
  - **Priority**: HIGH (improves readability)

#### `cmd/add.go`
- [x] **Font resolution logic** (lines ~420-470) ✅ **COMPLETED**
  - **Issue**: Complex logic for resolving Font IDs vs font names
  - **Breakdown**:
    1. **`resolveFontQuery`** (in `cmd/shared.go`)
       - **Purpose**: Resolve font query (Font ID or name) to FontFile list (reusable for add/import)
       - **Signature**: `func resolveFontQuery(fontName string, allFonts []repo.Font) ([]repo.FontFile, error)`
       - **Placement**: `cmd/shared.go` (shared utility, used by add and import)
  - **Target**: Extract into 1 helper function (~50 lines)
  - **Priority**: MEDIUM (already somewhat modular, but extraction improves reusability)

#### `cmd/remove.go`
- [ ] **Scope processing logic** (lines ~800-900)
  - **Issue**: Complex nested loops for processing multiple scopes
  - **Note**: This is already addressed in `removeFontsInDebugMode` breakdown above
  - **Priority**: MEDIUM (covered in long function breakdown)

### Medium Priority

#### `cmd/export.go`
- [ ] **Filtering logic** (lines ~328-450)
  - **Issue**: Complex conditional logic for match/source/exportAll filters
  - **Note**: This is already addressed in `performFullExportWithResult` breakdown above as `filterFontsForExport`
  - **Priority**: MEDIUM (covered in long function breakdown)

---

## 4. Naming Conventions

### Low Priority

#### Function Names
- [ ] **Review function naming consistency**
  - **Issue**: Some functions use different naming patterns
  - **Examples**:
    - `installFont` vs `installFontsInDebugMode` (singular vs plural)
    - `removeFont` vs `removeFontsInDebugMode`
    - `performFullExportWithResult` vs `performFullExport`
  - **Recommendation**: Standardize naming (prefer singular for single-item functions, plural for collections)
  - **Priority**: LOW (cosmetic, but improves consistency)

#### Variable Names
- [ ] **Review variable naming consistency**
  - **Issue**: Some inconsistencies in variable naming
  - **Examples**:
    - `fm` vs `fontManager` (abbreviation vs full name)
    - `r` vs `repo` (single letter vs descriptive)
  - **Recommendation**: Prefer descriptive names over abbreviations
  - **Priority**: LOW (cosmetic)

---

## 5. Unused Imports/Variables

### Medium Priority

- [ ] **Audit all command files for unused imports**
  - **Method**: Use `goimports` or `golangci-lint` to detect unused imports
  - **Files to check**: All files in `cmd/` directory
  - **Priority**: MEDIUM (reduces clutter)

- [ ] **Audit for unused variables**
  - **Method**: Use `golangci-lint` or `staticcheck` to detect unused variables
  - **Files to check**: All files in `cmd/` directory
  - **Priority**: MEDIUM (reduces clutter)

---

## 6. Code Organization

### Medium Priority

#### `cmd/shared.go`
- [ ] **Review shared.go organization**
  - **Issue**: File is 851 lines, contains many different utilities
  - **Recommendation**: Consider splitting into:
    - `shared_errors.go` - Error types and error handling
    - `shared_fonts.go` - Font-related utilities
    - `shared_formatting.go` - Formatting and display utilities
    - `shared_platform.go` - Platform-related utilities
  - **Priority**: MEDIUM (improves organization, but not critical)

#### Command File Organization
- [ ] **Review command file structure**
  - **Issue**: Some command files mix command definition, helper functions, and types
  - **Recommendation**: Standardize structure:
    1. Imports
    2. Types
    3. Constants
    4. Command definition
    5. Helper functions
  - **Priority**: LOW (cosmetic, but improves readability)

---

## 7. Testing Coverage

### High Priority

- [ ] **Add unit tests for shared utilities**
  - **Files**: `cmd/shared.go`
  - **Functions to test**:
    - `ParseFontNames`
    - `GetFontFamilyNameFromFilename`
    - `GetDisplayNameFromFilename`
    - `checkFontsAlreadyInstalled`
  - **Priority**: HIGH (shared utilities should be well-tested)

- [ ] **Add unit tests for complex functions**
  - **Functions**:
    - `installFont` (add.go)
    - `removeFont` (remove.go)
    - Filtering logic (list.go, export.go)
  - **Priority**: MEDIUM (improves reliability)

---

## 8. Documentation

### Low Priority

- [ ] **Add function documentation**
  - **Issue**: Some complex functions lack documentation
  - **Recommendation**: Add godoc comments for all exported functions and complex internal functions
  - **Priority**: LOW (improves maintainability)

- [ ] **Review inline comments**
  - **Issue**: Some complex logic lacks explanatory comments
  - **Recommendation**: Add comments for non-obvious logic
  - **Priority**: LOW (improves readability)

---

## Priority Summary

### High Priority (Do First)
1. Extract error handling helper functions
2. Extract manifest initialization helper
3. Break down long functions in add.go and remove.go
4. Extract filtering logic in list.go
5. Add unit tests for shared utilities

### Medium Priority (Do Next)
1. Extract font manager creation helper
2. Extract scope auto-detection logic
3. Break down long functions in export.go and backup.go
4. Extract complex conditional logic
5. Audit for unused imports/variables
6. Consider splitting shared.go

### Low Priority (Polish)
1. Standardize naming conventions
2. Improve code organization
3. Add function documentation
4. Review inline comments

---

## Function Placement Guidelines

### `cmd/shared.go` - Shared Utilities
Place functions here if they are:
- Used by **2+ commands**
- General-purpose utilities (error handling, initialization, common operations)
- Not command-specific logic

**Examples**:
- `handleError` - Used by all commands
- `ensureManifestInitialized` - Used by all commands
- `createFontManager` - Used by 4+ commands
- `checkFontAlreadyInstalled` - Used by add and import
- `resolveFontQuery` - Used by add and import
- `matchInstalledFontsToRepository` - Used by list and export
- `autoDetectScope` - Used by add and remove
- `getRepository` - Used by search and info

### Command-Specific Files (e.g., `cmd/add.go`)
Place functions here if they are:
- Used by **only one command**
- Command-specific business logic
- Command-specific formatting/logging

**Examples**:
- `processInstallResult` - Only used by add command
- `downloadFontVariants` - Only used by add command
- `filterFontsByFamilyAndID` - Only used by list command
- `organizeFontsBySourceAndFamily` - Only used by backup command

### `internal/components/` - UI Components
Place functions here if they are:
- UI/display-related utilities
- Reusable UI components
- Formatting for display

**Note**: Most UI components already exist in `internal/components/`. Only add new ones if they're truly reusable across multiple commands.

## Implementation Guidelines

### When Extracting Functions
1. **Single Responsibility**: Each function should do one thing well
2. **Clear Naming**: Function names should clearly describe what they do
3. **Error Handling**: Follow standard error handling pattern (use `handleError` helper when created)
4. **Testing**: Add tests for extracted functions, especially those in `shared.go`
5. **Documentation**: Add godoc comments for all exported functions and complex internal functions
6. **Placement**: Follow placement guidelines above - prefer `shared.go` for reusable utilities

### When Refactoring
1. **Test First**: Ensure existing tests pass before refactoring
2. **Small Steps**: Make small, incremental changes (extract one function at a time)
3. **Verify**: Test after each change to ensure no regressions
4. **Document**: Update `docs/codebase.md` when making significant changes
5. **Review**: Consider code review for complex refactorings

---

## Progress Tracking

### Completed ✅
- [x] Initial code quality assessment
- [x] Identified long functions
- [x] Identified duplicate code patterns
- [x] Identified complex conditional logic
- [x] Extract manifest initialization helper (`ensureManifestInitialized`)
- [x] Extract font manager creation helper (`createFontManager`)
- [x] Break down `installFontsInDebugMode` in `cmd/add.go` (into `processInstallResult`, `logInstallResultDetails`, `updateInstallStatus`)
- [x] Break down `installFont` in `cmd/add.go` (into `downloadFontVariants`, `installDownloadedFonts`, `buildInstallResult`)
- [x] Extract filtering logic in `cmd/list.go` (`filterFontsByFamilyAndID`)
- [x] Break down `removeFontsInDebugMode` in `cmd/remove.go` (into `processRemoveResult`, `logRemoveResultDetails`, `updateRemovalStatus`)
- [x] Break down `removeFont` in `cmd/remove.go` (into `findFontFilesForRemoval`, `removeFontFiles`, `buildRemoveResult`)
- [x] Extract font resolution logic in `cmd/add.go` (`resolveFontQuery` in `cmd/shared.go`)
- [x] Extract scope auto-detection logic in `cmd/shared.go` (`autoDetectScope`)
- [x] Break down `performFullExportWithResult` in `cmd/export.go` (into `matchInstalledFontsToRepository`, `populateFontMatchData`, `filterFontsForExport`, `buildExportManifest`)
- [x] Break down `performBackupWithProgress` in `cmd/backup.go` (into `organizeFontsBySourceAndFamily`, `createBackupZipArchive`)
- [x] Audit and fix unused imports/variables

### Remaining Tasks

#### High Priority
- [x] **Add unit tests for shared utilities** (Section 7) ✅ **COMPLETED**
  - Test `ParseFontNames`, `GetFontFamilyNameFromFilename`, `GetDisplayNameFromFilename`, `checkFontsAlreadyInstalled`
  - Test new shared functions: `resolveFontQuery`, `autoDetectScope`, `matchInstalledFontsToRepository`, `getSourceNameFromID`

#### Medium Priority
- [ ] **Extract repository initialization pattern** (Section 2, LOW priority but listed)
  - Create `getRepository(refresh bool)` helper in `cmd/shared.go`
  - Update `search.go` and `info.go` to use it
- [ ] **Review shared.go organization** (Section 6)
  - Consider splitting into: `shared_errors.go`, `shared_fonts.go`, `shared_formatting.go`, `shared_platform.go`
- [ ] **Add unit tests for complex functions** (Section 7)
  - Test `installFont`, `removeFont`, filtering logic

#### Low Priority (Polish)
- [ ] **Review function naming consistency** (Section 4)
  - Standardize `installFont` vs `installFontsInDebugMode` patterns
- [ ] **Review variable naming consistency** (Section 4)
  - Standardize `fm` vs `fontManager`, `r` vs `repo`
- [ ] **Review command file structure** (Section 6)
  - Standardize: Imports → Types → Constants → Command → Helpers
- [ ] **Add function documentation** (Section 8)
  - Add godoc comments for exported and complex internal functions
- [ ] **Review inline comments** (Section 8)
  - Add comments for non-obvious logic
- [ ] **`Update` method in `cmd/sources_manage.go`** (Section 1)
  - Minor refactoring (already well-structured, LOW priority)

---

## Notes

- **Reference**: Use `cmd/shared.go` as a model for well-organized utility functions
- **Testing**: Run tests after each refactoring to ensure no regressions
- **Code Review**: Review changes with team before merging
- **Documentation**: Update `docs/codebase.md` when making significant changes

