# Debug Output Alignment Review

This document identifies commands that don't align with the `add` and `remove` commands' debug output patterns and logging guidelines.

## Reference Pattern (Add/Remove Commands)

The `add` and `remove` commands follow this debug output structure:

1. **Starting operation message**
   - `output.GetDebug().State("Starting font [operation] operation")`
   - `output.GetDebug().State("Total fonts: %d", count)`

2. **Per-font processing**
   - `output.GetDebug().State("[Operation] font %d/%d: %s", i+1, total, fontName)`
   - `output.GetDebug().State("[Operation] font %s in %s (directory: %s)", fontName, scopeLabel, fontDir)`

3. **Function call traces** (inside helper functions)
   - `output.GetDebug().State("Calling [functionName](...)")`
   - `output.GetDebug().State("Found %d matching font file(s)", count)`

4. **Internal state values**
   - Font name resolution, repository searches, file counts, etc.

5. **Separate variant categorization**
   - `output.GetDebug().State("[Status] variants:")` (e.g., "Installed variants:", "Removed variants:", "Skipped variants:", "Failed variants:")
   - Each variant listed with `output.GetDebug().State(" - %s", variantName)`

6. **Completion status**
   - `output.GetDebug().State("Font %s in %s completed: %s - %s ([Status]: %d, Skipped: %d, Failed: %d)", ...)`

7. **Operation complete summary**
   - `output.GetDebug().State("Operation complete - [Status]: %d, Skipped: %d, Failed: %d", ...)`

## Commands Requiring Alignment

### 1. Import Command (`cmd/import.go`)

**Status**: ⚠️ **Partially Aligned** - Missing several key elements

**Current Issues**:

1. **Missing function call trace** for `installFont()`
   - **Location**: Line 606
   - **Issue**: No debug message before calling `installFont()`
   - **Fix**: Add `output.GetDebug().State("Calling installFont(...)")` before line 606

2. **"scope scope" typo** in error message
   - **Location**: Line 604, 616, 635
   - **Issue**: Uses `"in %s scope"` but `scopeLabel` already includes "scope"
   - **Fix**: Change to `"in %s"` (remove "scope" from format string)

3. **Missing separate variant categorization**
   - **Location**: Lines 625-636
   - **Issue**: Doesn't show separate "Installed variants:", "Skipped variants:", "Failed variants:" sections
   - **Fix**: Add variant categorization logic similar to `add` command (lines 905-941 in `add.go`)

4. **Missing detailed variant output**
   - **Location**: Lines 625-636
   - **Issue**: Only shows completion status, not individual variants
   - **Fix**: Parse `result.Details` and show variants in separate categories

5. **Missing internal state values**
   - **Location**: Throughout `importFontsInDebugMode`
   - **Issue**: Doesn't show font name resolution, file counts, etc.
   - **Fix**: Add debug messages for internal operations (similar to `remove` command's `removeFont` function)

**Required Changes**:

```go
// Line 603-604: Fix "scope scope" typo
output.GetDebug().State("Importing font %d/%d: %s", i+1, len(fontsToInstall), fontGroup.FontName)
output.GetDebug().State("Installing font %s in %s (directory: %s)", fontGroup.FontName, scopeLabel, fontDir)

// Line 605: Add function call trace
output.GetDebug().State("Calling installFont(%s, %s, %s, %v, %s)", fontGroup.FontID, scopeLabel, fontDir, force, "...")

// Line 616: Fix "scope scope" typo
output.GetDebug().State("Error installing font %s in %s: %v", fontGroup.FontName, scopeLabel, err)

// Lines 625-636: Add variant categorization (similar to add.go lines 905-941)
if len(result.Details) > 0 {
    installedCount := result.Success
    skippedCount := result.Skipped
    failedCount := result.Failed
    var installedFiles, skippedFiles, failedFiles []string
    idx := 0
    if installedCount > 0 && idx < len(result.Details) {
        installedFiles = result.Details[idx : idx+installedCount]
        idx += installedCount
    }
    if skippedCount > 0 && idx < len(result.Details) {
        skippedFiles = result.Details[idx : idx+skippedCount]
        idx += skippedCount
    }
    if failedCount > 0 && idx < len(result.Details) {
        failedFiles = result.Details[idx : idx+failedCount]
    }

    if len(installedFiles) > 0 {
        output.GetDebug().State("Installed variants:")
        for _, file := range installedFiles {
            output.GetDebug().State(" - %s", file)
        }
    }
    if len(skippedFiles) > 0 {
        output.GetDebug().State("Skipped variants:")
        for _, file := range skippedFiles {
            output.GetDebug().State(" - %s", file)
        }
    }
    if len(failedFiles) > 0 {
        output.GetDebug().State("Failed variants:")
        for _, file := range failedFiles {
            output.GetDebug().State(" - %s", file)
        }
    }
}

// Line 635: Fix "scope scope" typo
output.GetDebug().State("Font %s in %s completed: %s - %s (Installed: %d, Skipped: %d, Failed: %d)",
    fontGroup.FontName, scopeLabel, result.Status, result.Message, result.Success, result.Skipped, result.Failed)
```

---

### 2. Export Command (`cmd/export.go`)

**Status**: ⚠️ **Partially Aligned** - Has debug output but not structured like add/remove

**Current Issues**:

1. **No structured debug mode function**
   - **Location**: No dedicated `exportFontsInDebugMode()` function
   - **Issue**: Debug output is scattered throughout the main function
   - **Fix**: Consider creating a dedicated debug mode function if export operations become more complex

2. **Missing operation start message**
   - **Location**: Line 79
   - **Issue**: Only has file logging, no debug console output for operation start
   - **Fix**: Add `output.GetDebug().State("Starting font export operation")` after line 79

3. **Missing total fonts count**
   - **Location**: After font collection
   - **Issue**: No debug message showing total fonts to export
   - **Fix**: Add `output.GetDebug().State("Total fonts to export: %d", count)` after font collection

4. **Inconsistent debug message format**
   - **Location**: Various locations
   - **Issue**: Some debug messages use different formats
   - **Fix**: Standardize to match add/remove pattern

**Required Changes**:

```go
// After line 79: Add operation start
output.GetDebug().State("Starting font export operation")

// After font collection: Add total count
output.GetDebug().State("Total fonts to export: %d", len(fontsToExport))

// Standardize existing debug messages to match add/remove format
```

**Note**: Export command is simpler than add/remove, so full alignment may not be necessary. Consider if the current debug output is sufficient for troubleshooting.

---

### 3. Backup Command (`cmd/backup.go`)

**Status**: ⚠️ **Partially Aligned** - Has debug output but not structured like add/remove

**Current Issues**:

1. **No structured debug mode function**
   - **Location**: No dedicated `backupFontsInDebugMode()` function
   - **Issue**: Debug output is scattered throughout the main function
   - **Fix**: Consider creating a dedicated debug mode function if backup operations become more complex

2. **Missing operation start message**
   - **Location**: Line 56
   - **Issue**: Only has file logging, no debug console output for operation start
   - **Fix**: Add `output.GetDebug().State("Starting font backup operation")` after line 56

3. **Missing total fonts count**
   - **Location**: After font collection
   - **Issue**: No debug message showing total fonts to backup
   - **Fix**: Add `output.GetDebug().State("Total fonts to backup: %d", count)` after font collection

4. **Inconsistent debug message format**
   - **Location**: Various locations
   - **Issue**: Some debug messages use different formats
   - **Fix**: Standardize to match add/remove pattern

**Required Changes**:

```go
// After line 56: Add operation start
output.GetDebug().State("Starting font backup operation")

// After font collection: Add total count
output.GetDebug().State("Total fonts to backup: %d", len(fontsToBackup))

// Standardize existing debug messages to match add/remove format
```

**Note**: Backup command is simpler than add/remove, so full alignment may not be necessary. Consider if the current debug output is sufficient for troubleshooting.

---

### 4. List Command (`cmd/list.go`)

**Status**: ✅ **Mostly Aligned** - Has appropriate debug output for its operation type

**Current State**:
- Has debug output for font collection
- Has debug output for filtering
- Has debug output for matching operations
- Debug output is appropriate for a read-only operation

**No changes required** - List command is a read-only operation and doesn't need the same level of detail as add/remove.

---

### 5. Search Command (`cmd/search.go`)

**Status**: ✅ **Mostly Aligned** - Has appropriate debug output for its operation type

**Current State**:
- Has debug output for search parameters
- Has debug output for repository operations
- Has debug output for search results
- Debug output is appropriate for a read-only operation

**No changes required** - Search command is a read-only operation and doesn't need the same level of detail as add/remove.

---

### 6. Info Command (`cmd/info.go`)

**Status**: ✅ **Mostly Aligned** - Has appropriate debug output for its operation type

**Current State**:
- Has debug output for font lookup
- Has debug output for repository operations
- Has debug output for matching operations
- Debug output is appropriate for a read-only operation

**No changes required** - Info command is a read-only operation and doesn't need the same level of detail as add/remove.

---

## Summary

### High Priority (Must Fix)

1. **Import Command** - Missing function call traces, variant categorization, and has "scope scope" typo
   - **Impact**: High - Import is a write operation similar to add/remove
   - **Effort**: Medium - Requires adding variant categorization logic

### Medium Priority (Consider Fixing)

2. **Export Command** - Missing structured debug output
   - **Impact**: Medium - Export is a read operation but could benefit from better debug output
   - **Effort**: Low - Mostly adding operation start messages

3. **Backup Command** - Missing structured debug output
   - **Impact**: Medium - Backup is a write operation but simpler than add/remove
   - **Effort**: Low - Mostly adding operation start messages

### Low Priority (No Changes Needed)

4. **List Command** - Appropriate debug output for read-only operation
5. **Search Command** - Appropriate debug output for read-only operation
6. **Info Command** - Appropriate debug output for read-only operation

## Implementation Priority

1. **First**: Fix Import command (most similar to add/remove, highest impact)
2. **Second**: Consider adding operation start messages to Export and Backup commands
3. **Third**: Review if any other commands need alignment based on usage patterns

## Notes

- Read-only operations (list, search, info) don't need the same level of detail as write operations (add, remove, import)
- Write operations that modify system state should have comprehensive debug output for troubleshooting
- The add/remove pattern should be the reference for all write operations

