# Logging Guidelines for FontGet

This document establishes clear guidelines for verbose and debug messaging across all FontGet commands to ensure consistent, useful, and non-redundant output.

## Overview

FontGet uses three types of logging:
- **File Logging (GetLogger())**: Persistent log file for all operations, errors, and important events
- **Verbose**: User-relevant operational details (console output)
- **Debug**: Developer-focused technical details (console output)

## File Logging Guidelines (GetLogger())

### Purpose
File logging provides a persistent record of all FontGet operations, errors, and important events. This log file is always written regardless of verbose or debug flags, making it essential for troubleshooting issues that occurred in the past.

### Key Principles
- **Always Active**: File logging is always enabled and writes to `fontget.log`
- **Not Conditional**: `GetLogger()` calls should NEVER be wrapped in `IsVerbose()` or `IsDebug()` checks
- **Level Controlled by Config**: The log level (Error/Info/Debug) is controlled by the application's configuration, not by conditional code
- **Separate from Console**: File logging is independent of console output (verbose/debug)

### Use Cases
- **Operation Lifecycle**: Start, parameters, completion, and results of all operations
- **Error Tracking**: All errors should be logged with context for troubleshooting
- **Important State Changes**: Significant state transitions (e.g., scope auto-detection, configuration changes)
- **Performance Metrics**: Operation completion with counts (installed, skipped, failed)
- **Parameter Logging**: Command parameters, filters, scopes, and configuration values

### Format
- Structured, machine-readable format with timestamps
- Include operation context (command name, parameters)
- Use appropriate log levels (Info, Error, Warn, Debug)
- Include relevant data (counts, paths, IDs) for troubleshooting

### Examples
```go
// Good file logging
GetLogger().Info("Starting font installation operation")
GetLogger().Info("Installation parameters - Scope: %s, Force: %v, Fonts: %v", scope, force, fontNames)
GetLogger().Error("Failed to install font %s: %v", fontName, err)
GetLogger().Info("Installation complete - Installed: %d, Skipped: %d, Failed: %d", installed, skipped, failed)

// Bad file logging (conditional)
if IsDebug() {
    GetLogger().Info("Operation started") // WRONG - should always log
}

// Bad file logging (console output)
GetLogger().Info("Installing fonts...") // This goes to file, not console
// Use output.GetVerbose().Info() for console output instead
```

### Log Levels
- **Error**: All errors and failures that need attention
- **Warn**: Warnings about non-critical issues (e.g., failed checks that don't block operation)
- **Info**: Normal operation flow, start/completion, parameters, results
- **Debug**: Detailed technical information (controlled by config level)

### When to Use File Logging vs Console Output
- **File Logging (GetLogger())**: 
  - Operation lifecycle (start, parameters, completion)
  - All errors (for troubleshooting)
  - Important state changes
  - Results and counts
  
- **Console Output (Verbose/Debug)**:
  - User-facing progress information
  - Real-time feedback during operations
  - User-friendly error messages
  - Technical debugging details

### Implementation Pattern
```go
// At operation start
GetLogger().Info("Starting [operation name] operation")
GetLogger().Info("[Operation] parameters - [key details]")

// During operation (important events only)
GetLogger().Info("Auto-detected scope: %s", scope)

// On errors
GetLogger().Error("Failed to [operation]: %v", err)

// At completion
GetLogger().Info("[Operation] complete - [results summary]")
```

---

## Verbose Messaging Guidelines

### Purpose
Verbose messages provide operational details that users care about when troubleshooting or understanding what FontGet is doing.

### Use Cases
- **File paths and locations**: Where fonts are being installed/removed
- **Installation scope**: User vs machine scope operations
- **Source information**: Which font source is being used
- **Operation progress**: What actions are being performed
- **Configuration values**: Settings being used
- **Search results**: What fonts were found
- **Error context**: User-friendly error explanations

### Format
- User-friendly language, no technical jargon
- Present tense for ongoing operations
- Past tense for completed operations
- Include relevant context (paths, scopes, counts)

### Examples
```go
// Good verbose messages
output.GetVerbose().Info("Installing fonts to: %s", fontDir)
output.GetVerbose().Detail("Info", "Font exists at: %s", expectedPath)
output.GetVerbose().Success("Successfully installed font: %s to %s scope", fontName, scope)
output.GetVerbose().Warning("Skipping protected system font: %s", fontName)

// Bad verbose messages (too technical)
output.GetVerbose().Info("Calling fontManager.InstallFont()")
output.GetVerbose().Detail("Debug", "Variable fontPaths length: %d", len(fontPaths))
```

## Debug Messaging Guidelines

### Purpose
Debug messages provide technical details for developers debugging issues or understanding internal behavior.

### Use Cases
- **Function call traces**: Which functions are being called
- **Internal state values**: Variable contents and state changes
- **API responses**: Raw data from external sources
- **Error stack traces**: Technical error details
- **Performance timing**: How long operations take
- **Conditional logic paths**: Which code paths are taken
- **Configuration parsing**: Technical details of config loading

### Format
- Technical language with function names
- Include variable names and values
- Show function call chains
- Include line numbers or context when helpful

### Examples
```go
// Good debug messages
output.GetDebug().State("Starting font removal process for: %s", fontName)
output.GetDebug().State("Found %d matching font files in %s scope", len(matchingFonts), scope)
output.GetDebug().Error("Font removal failed for %s in %s scope: %v", fontName, scope, err)
output.GetDebug().State("Calling config.LoadUserPreferences() -> returned %d sources", len(sources))

// Bad debug messages (too user-focused)
output.GetDebug().Info("Downloading font from Google Fonts")
output.GetDebug().Success("Font installed successfully")
```

## Classification Criteria

### File Logging (GetLogger())
- **Always log**: Operation lifecycle, errors, important state changes
- **Purpose**: Persistent record for troubleshooting and auditing
- **Audience**: Developers, support, and users reviewing logs
- **Format**: Structured with timestamps, machine-readable

### If user would care → Verbose
- Shows "what" is happening
- Helps with troubleshooting
- Explains user-visible behavior
- Provides operational context
- **Output**: Console (when `--verbose` flag is used)

### If only developer would care → Debug
- Shows "how" something works
- Internal implementation details
- Technical error information
- Performance metrics
- **Output**: Console (when `--debug` flag is used)

## Command-Specific Guidelines

### Add Command
- **File Logging**: Operation start, installation parameters (scope, force, font names), auto-detected scope, errors, completion with counts
- **Verbose**: Download progress, installation paths, source names, scope information
- **Debug**: Download function calls, file extraction details, internal state changes

### Remove Command
- **File Logging**: Operation start, removal parameters (scope, font names), errors, completion with counts
- **Verbose**: Removal progress, font locations, scope information, protection status
- **Debug**: File search algorithms, removal function calls, error handling details

### List Command
- **File Logging**: Operation start, list parameters (scope, type filter, family filter), errors, completion with counts
- **Verbose**: Font scanning progress, directory paths, font counts
- **Debug**: File system traversal, font parsing details, filtering logic

### Search Command
- **File Logging**: Operation start, search parameters (query, filters), errors, completion with result counts
- **Verbose**: Search progress, result counts, source information
- **Debug**: Search algorithms, scoring logic, API calls

### Export Command
- **File Logging**: Operation start, export parameters (output file, filters), errors, completion with counts
- **Verbose**: Export progress, file paths, filter information
- **Debug**: File matching logic, manifest generation details

### Import Command
- **File Logging**: Operation start, import parameters (file path, scope, force), errors, completion with counts
- **Verbose**: Import progress, source information, font names
- **Debug**: File parsing, validation logic, installation details

### Backup Command
- **File Logging**: Operation start, backup parameters (output file, scopes), errors, completion with counts
- **Verbose**: Backup progress, archive creation, file counts
- **Debug**: Archive creation details, file collection logic

### Config Command
- **File Logging**: Operation start, config operations (view/edit/reset), errors, completion
- **Verbose**: Configuration file paths, setting values, validation results
- **Debug**: YAML parsing details, validation logic, file I/O operations

### Sources Command
- **File Logging**: Operation start, source operations (info/update/manage/validate), errors, completion
- **Verbose**: Source operations, URL information, validation results
- **Debug**: HTTP request details, JSON parsing, cache operations

## Implementation Examples

### Good File Logging Implementation
```go
// Always log operation lifecycle and errors
GetLogger().Info("Starting font installation operation")
GetLogger().Info("Installation parameters - Scope: %s, Force: %v, Fonts: %v", scope, force, fontNames)
GetLogger().Error("Failed to install font %s: %v", fontName, err)
GetLogger().Info("Installation complete - Installed: %d, Skipped: %d, Failed: %d", installed, skipped, failed)
```

### Good Verbose Implementation
```go
// Show user-relevant information (console output)
output.GetVerbose().Info("Scope: %s", scope)
output.GetVerbose().Info("Installing %d font(s)", count)
// Verbose section ends with blank line per spacing framework (only if verbose was shown)
if IsVerbose() {
    fmt.Println()
}
```

### Good Debug Implementation
```go
// Show technical details (console output)
output.GetDebug().State("Calling findFontFamilyFiles(%s, %s, %s)", fontName, fontManager, scope)
output.GetDebug().State("Font search returned %d results", len(results))
output.GetDebug().Error("FontManager.RemoveFont() failed: %v", err)
```

### Bad Implementation (Cross-contamination)
```go
// Don't mix user and technical information
output.GetVerbose().Info("Calling fontManager.InstallFont()") // Too technical for verbose
output.GetDebug().Success("Font installed successfully") // Too user-focused for debug

// Don't make file logging conditional
if IsDebug() {
    GetLogger().Info("Operation started") // WRONG - should always log
}
```

### Bad Implementation (Wrong Output Channel)
```go
// Don't use GetLogger() for console output
GetLogger().Info("Installing fonts...") // Goes to file, not console
// Use output.GetVerbose().Info() for console output instead

// Don't use verbose/debug for file logging
output.GetVerbose().Info("Operation started") // Goes to console, not file
// Use GetLogger().Info() for file logging instead
```

## Common Patterns to Avoid

### Redundant Messages
```go
// Don't repeat the same information
output.GetVerbose().Info("Starting download")
output.GetVerbose().Info("Download in progress") // Redundant
```

### Useless Messages
```go
// Don't log obvious things
output.GetVerbose().Info("Processing started") // Obvious
output.GetDebug().State("Variable i = %d", i) // Not useful
```

### Cross-contamination
```go
// Don't put technical details in verbose
output.GetVerbose().Detail("Debug", "Function call: %s", functionName)

// Don't put user messages in debug
output.GetDebug().Success("Operation completed successfully")
```

### Suppressing Verbose for Internal/Helper Functions

When shared functions are used for both primary operations and internal checks, suppress verbose output for internal usage:

```go
// Good: Function accepts suppressVerbose parameter
func collectFonts(scopes []platform.InstallationScope, fm platform.FontManager, typeFilter string, suppressVerbose ...bool) ([]ParsedFont, error) {
    shouldSuppressVerbose := false
    if len(suppressVerbose) > 0 {
        shouldSuppressVerbose = suppressVerbose[0]
    }
    
    for _, scope := range scopes {
        if !shouldSuppressVerbose {
            output.GetVerbose().Info("Scanning %s scope: %s", scope, fontDir)
        }
        // ... rest of function
    }
}

// Primary operation (list command) - show verbose
fonts, err := collectFonts(scopes, fm, typeFilter)

// Internal check (add command) - suppress verbose
fonts, err := collectFonts(scopes, fontManager, "", true)
```

**Why**: Internal checks (like checking if a font is already installed) are implementation details that users don't need to see. Only show verbose output when the operation is the primary purpose of the command (e.g., `list` command scanning fonts).

## Testing Guidelines

### File Logging Testing
- Test without any flags (default mode)
- Verify `fontget.log` file is created and contains entries
- Check that operation start, parameters, errors, and completion are logged
- Verify log entries have timestamps and proper formatting
- Test with different log levels (Error/Info/Debug) via config
- Ensure GetLogger() calls are NOT conditional on flags

### Verbose Testing
- Test with `--verbose` flag
- Verify messages are user-relevant
- Check that technical details are not shown
- Ensure messages help with troubleshooting
- Verify messages appear on console, not just in log file

### Debug Testing
- Test with `--debug` flag
- Verify messages are technical
- Check that user-friendly messages are not shown
- Ensure messages help with debugging
- Verify messages appear on console, not just in log file

### Cross-contamination Testing
- Test both flags together
- Verify no overlap between verbose and debug
- Check that messages are properly categorized
- Verify file logging is independent of console output flags

## Maintenance

### Regular Audits
- Review verbose/debug usage quarterly
- Check for new cross-contamination
- Verify messages are still relevant
- Remove outdated or useless messages

### Code Reviews
- Check new verbose/debug messages
- Verify proper classification
- Ensure no redundant information
- Test both flag combinations

## Tools and Utilities

### Linting
- Use static analysis to find unused logging calls
- Check for proper message formatting
- Verify consistent usage patterns

### Testing
- Unit tests for logging behavior
- Integration tests with flags
- User acceptance testing for message clarity

## Examples by Command

### Add Command Examples
```go
// File Logging (always active, goes to fontget.log)
GetLogger().Info("Starting font installation operation")
GetLogger().Info("Installation parameters - Scope: %s, Force: %v, Fonts: %v", scope, force, fontNames)
GetLogger().Error("Failed to install font %s: %v", fontName, err)
GetLogger().Info("Installation complete - Installed: %d, Skipped: %d, Failed: %d", installed, skipped, failed)

// Verbose (user-relevant, console output)
output.GetVerbose().Info("Scope: %s", scope)
output.GetVerbose().Info("Installing %d font(s)", len(fontNames))
// Verbose section ends with blank line per spacing framework (only if verbose was shown)
if IsVerbose() {
    fmt.Println()
}

// Debug (technical, console output)
output.GetDebug().State("Calling repo.DownloadAndExtractFont()")
output.GetDebug().State("Font extraction returned %d files", len(fontPaths))
output.GetDebug().Error("fontManager.InstallFont() failed: %v", err)
```

### Remove Command Examples
```go
// File Logging (always active, goes to fontget.log)
GetLogger().Info("Starting font removal operation")
GetLogger().Info("Removal parameters - Scope: %s, Fonts: %v", scope, fontNames)
GetLogger().Error("Failed to remove font %s: %v", fontName, err)
GetLogger().Info("Removal complete - Removed: %d, Skipped: %d, Failed: %d", removed, skipped, failed)

// Verbose (user-relevant, console output)
output.GetVerbose().Info("Scope: %s", scope)
output.GetVerbose().Info("Removing %d font(s)", len(fontNames))
// Verbose section ends with blank line per spacing framework (only if verbose was shown)
if IsVerbose() {
    fmt.Println()
}

// Debug (technical, console output)
output.GetDebug().State("Calling findFontFamilyFiles(%s, %s, %s)", fontName, fontManager, scope)
output.GetDebug().State("Found %d matching fonts", len(matchingFonts))
output.GetDebug().Error("fontManager.RemoveFont() failed: %v", err)
```

### List Command Examples
```go
// File Logging (always active, goes to fontget.log)
GetLogger().Info("Starting font list operation")
GetLogger().Info("List parameters - Scope: %v, Type filter: %s, Family filter: %s", scopes, typeFilter, familyFilter)
GetLogger().Error("Failed to list fonts: %v", err)
GetLogger().Info("List operation complete - Found %d fonts", len(fonts))

// Verbose (user-relevant, console output)
output.GetVerbose().Info("Scanning %s scope: %s", scope, fontDir)
output.GetVerbose().Info("Found %d files in %s", len(names), fontDir)
output.GetVerbose().Info("Scan complete: parsed %d files across %d scope(s)", len(parsed), len(scopes))
// Verbose section ends with blank line per spacing framework (only if verbose was shown)
if IsVerbose() {
    fmt.Println()
}

// Debug (technical, console output)
output.GetDebug().State("Matching %d font families against repository", len(families))
output.GetDebug().State("Filtering fonts - Type: %s, Family: %s", typeFilter, familyFilter)
```

## Summary

### Three Logging Channels

1. **File Logging (GetLogger())**
   - Always active, writes to `fontget.log`
   - Never conditional on flags
   - Logs: operation lifecycle, parameters, errors, completion
   - Purpose: Persistent record for troubleshooting

2. **Verbose (output.GetVerbose())**
   - Console output when `--verbose` flag is used
   - User-relevant operational details
   - Conditional: `if IsVerbose() && !IsDebug()`
   - Purpose: Help users understand what's happening

3. **Debug (output.GetDebug())**
   - Console output when `--debug` flag is used
   - Technical details for developers
   - Always shown when `--debug` is active
   - Purpose: Help developers debug issues

### Key Rules

- ✅ **Always use GetLogger()** for file logging (operation start, parameters, errors, completion)
- ✅ **Never make GetLogger() conditional** on `IsVerbose()` or `IsDebug()`
- ✅ **Use verbose for user-facing console output** (conditional on `IsVerbose() && !IsDebug()`)
- ✅ **Use debug for technical console output** (shown when `--debug` is active)
- ❌ **Don't use GetLogger() for console output** (use verbose/debug instead)
- ❌ **Don't use verbose/debug for file logging** (use GetLogger() instead)

This document should be referenced during code reviews and when adding new logging statements to ensure consistent, useful output across all FontGet commands.
