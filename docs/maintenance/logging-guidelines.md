# Logging Guidelines for FontGet

This document establishes clear guidelines for verbose and debug messaging across all FontGet commands to ensure consistent, useful, and non-redundant output.

## Overview

FontGet uses two types of user-facing logging:
- **Verbose**: User-relevant operational details
- **Debug**: Developer-focused technical details

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

### If user would care → Verbose
- Shows "what" is happening
- Helps with troubleshooting
- Explains user-visible behavior
- Provides operational context

### If only developer would care → Debug
- Shows "how" something works
- Internal implementation details
- Technical error information
- Performance metrics

## Command-Specific Guidelines

### Add Command
- **Verbose**: Download progress, installation paths, source names, scope information
- **Debug**: Download function calls, file extraction details, internal state changes

### Remove Command
- **Verbose**: Removal progress, font locations, scope information, protection status
- **Debug**: File search algorithms, removal function calls, error handling details

### List Command
- **Verbose**: Font scanning progress, directory paths, font counts
- **Debug**: File system traversal, font parsing details, filtering logic

### Search Command
- **Verbose**: Search progress, result counts, source information
- **Debug**: Search algorithms, scoring logic, API calls

### Config Command
- **Verbose**: Configuration file paths, setting values, validation results
- **Debug**: YAML parsing details, validation logic, file I/O operations

### Sources Command
- **Verbose**: Source operations, URL information, validation results
- **Debug**: HTTP request details, JSON parsing, cache operations

## Implementation Examples

### Good Verbose Implementation
```go
// Show user-relevant information
output.GetVerbose().Info("Processing font: %s", fontName)
output.GetVerbose().Detail("Info", "Installing to: %s", installPath)
output.GetVerbose().Success("Successfully installed %d font variants", len(variants))
```

### Good Debug Implementation
```go
// Show technical details
output.GetDebug().State("Calling findFontFamilyFiles(%s, %s, %s)", fontName, fontManager, scope)
output.GetDebug().State("Font search returned %d results", len(results))
output.GetDebug().Error("FontManager.RemoveFont() failed: %v", err)
```

### Bad Implementation (Cross-contamination)
```go
// Don't mix user and technical information
output.GetVerbose().Info("Calling fontManager.InstallFont()") // Too technical for verbose
output.GetDebug().Success("Font installed successfully") // Too user-focused for debug
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

## Testing Guidelines

### Verbose Testing
- Test with `--verbose` flag
- Verify messages are user-relevant
- Check that technical details are not shown
- Ensure messages help with troubleshooting

### Debug Testing
- Test with `--debug` flag
- Verify messages are technical
- Check that user-friendly messages are not shown
- Ensure messages help with debugging

### Cross-contamination Testing
- Test both flags together
- Verify no overlap between verbose and debug
- Check that messages are properly categorized

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
// Verbose (user-relevant)
output.GetVerbose().Info("Downloading and Installing Fonts")
output.GetVerbose().Detail("Info", "Installing to: %s", fontDir)
output.GetVerbose().Success("Successfully installed %s to %s scope", fontName, scope)

// Debug (technical)
output.GetDebug().State("Calling repo.DownloadAndExtractFont()")
output.GetDebug().State("Font extraction returned %d files", len(fontPaths))
output.GetDebug().Error("fontManager.InstallFont() failed: %v", err)
```

### Remove Command Examples
```go
// Verbose (user-relevant)
output.GetVerbose().Info("Finding and Removing Fonts")
output.GetVerbose().Detail("Info", "Removing from: %s", fontDir)
output.GetVerbose().Success("Successfully removed %s from %s scope", fontName, scope)

// Debug (technical)
output.GetDebug().State("Calling findFontFamilyFiles(%s, %s, %s)", fontName, fontManager, scope)
output.GetDebug().State("Found %d matching fonts", len(matchingFonts))
output.GetDebug().Error("fontManager.RemoveFont() failed: %v", err)
```

This document should be referenced during code reviews and when adding new logging statements to ensure consistent, useful output across all FontGet commands.
