# Codebase Layout Guidelines

This document provides guidelines for organizing code in the FontGet codebase, explaining where different types of code should be placed and why.

## Table of Contents

- [Directory Structure Overview](#directory-structure-overview)
- [Package Organization Principles](#package-organization-principles)
- [Package-Specific Guidelines](#package-specific-guidelines)
- [Decision Tree: Where Should This Code Go?](#decision-tree-where-should-this-code-go)
- [Common Patterns](#common-patterns)
- [Anti-Patterns to Avoid](#anti-patterns-to-avoid)

---

## Directory Structure Overview

```
fontget/
├── cmd/                    # CLI command implementations
├── internal/               # Internal packages (not exported)
│   ├── cmdutils/          # CLI-specific utilities
│   ├── shared/            # General-purpose utilities
│   ├── functions/         # Domain-specific utilities
│   ├── platform/          # Platform abstraction layer
│   ├── repo/              # Font repository management
│   ├── config/            # Configuration management
│   ├── ui/                # User interface components
│   ├── output/            # Output management (verbose/debug)
│   ├── components/        # Reusable UI components
│   ├── logging/           # File logging system
│   └── ...                # Other internal packages
└── docs/                  # Documentation
```

---

## Package Organization Principles

### 1. **Separation of Concerns**
Each package should have a single, well-defined responsibility.

### 2. **Dependency Direction**
- `cmd/` depends on `internal/` packages
- `internal/` packages should not depend on `cmd/`
- Lower-level packages should not depend on higher-level packages

### 3. **Reusability**
- General-purpose code goes in `internal/shared/`
- CLI-specific code goes in `internal/cmdutils/`
- Domain-specific code goes in appropriate domain packages

### 4. **Testability**
- Packages should be easily testable in isolation
- Avoid circular dependencies
- Use interfaces for abstraction

---

## Package-Specific Guidelines

### `cmd/` - Command Implementations

**Purpose**: CLI command implementations using Cobra framework

**Contains**:
- Command definitions (`add.go`, `remove.go`, `list.go`, etc.)
- Command-specific logic and workflows
- Operation orchestration (`operations.go`, `handlers.go`)

**Guidelines**:
- ✅ Each command should be in its own file
- ✅ Commands should delegate to `internal/` packages for business logic
- ✅ Commands should use `cmdutils` for CLI-specific utilities
- ✅ Commands should use `shared` for general-purpose utilities
- ❌ Don't put reusable utilities directly in `cmd/`
- ❌ Don't create `cmd/shared.go` (use `internal/cmdutils/` or `internal/shared/` instead)

**Example**:
```go
// cmd/add.go
func addCmd.RunE(cmd *cobra.Command, args []string) error {
    // Use cmdutils for CLI-specific helpers
    if err := cmdutils.EnsureManifestInitialized(...); err != nil {
        return err
    }
    
    // Use shared for general utilities
    fontNames := cmdutils.ParseFontNames(args)
    similar := shared.FindSimilarFonts(fontName, ...)
    
    // Command-specific logic here
}
```

---

### `internal/cmdutils/` - CLI-Specific Utilities

**Purpose**: Utilities that are specific to CLI commands and depend on CLI context

**Contains**:
- CLI initialization helpers (`EnsureManifestInitialized`, `CreateFontManager`)
- Cobra integration (`CheckElevation`, `PrintElevationHelp`)
- CLI argument parsing (`ParseFontNames`)
- CLI wrappers for repository operations with logging (`GetRepository`, `MatchInstalledFontsToRepository`)

**When to Use**:
- ✅ Code that needs Cobra command context
- ✅ Code that provides CLI-specific error handling with verbose/debug output
- ✅ Code that wraps `internal/` packages with CLI-specific logging/error handling
- ✅ Code that's only used by commands, not by other internal packages

**When NOT to Use**:
- ❌ General-purpose utilities (use `internal/shared/` instead)
- ❌ Business logic (use domain-specific packages)
- ❌ Code that might be used by non-CLI code

**Key Characteristics**:
- Functions often accept `Logger` interface for CLI logging
- Functions integrate with `output.GetVerbose()` and `output.GetDebug()`
- Functions provide standardized error messages for CLI users

**Example**:
```go
// internal/cmdutils/init.go
func EnsureManifestInitialized(getLogger func() Logger) error {
    if err := config.EnsureManifestExists(); err != nil {
        if logger := getLogger(); logger != nil {
            logger.Error("Failed to ensure manifest exists: %v", err)
        }
        output.GetVerbose().Error("%v", err)
        output.GetDebug().Error("config.EnsureManifestExists() failed: %v", err)
        return fmt.Errorf("unable to load font repository: %w", err)
    }
    return nil
}
```

---

### `internal/shared/` - General-Purpose Utilities

**Purpose**: Reusable utilities that are domain-agnostic and could be used anywhere

**Contains**:
- Font formatting utilities (`FormatFontNameWithVariant`, `GetFontDisplayName`)
- File utilities (`FormatFileSize`, `SanitizeForZipPath`, `TruncateString`)
- Font matching utilities (`FindSimilarFonts`)
- Error types (`FontNotFoundError`, `FontInstallationError`, etc.)
- System font utilities (`IsCriticalSystemFont`)
- Repository query resolution (`ResolveFontQuery`)

**When to Use**:
- ✅ Pure utility functions with no CLI dependencies
- ✅ Functions that could be used by commands, tests, or other internal packages
- ✅ Domain-agnostic utilities (font formatting, file operations, string manipulation)
- ✅ Error types that are part of the domain model

**When NOT to Use**:
- ❌ CLI-specific code (use `internal/cmdutils/` instead)
- ❌ Code that depends on Cobra or CLI context
- ❌ Domain-specific business logic (use domain packages like `internal/repo/`)

**Key Characteristics**:
- Functions are pure (no side effects beyond their return values)
- Functions don't depend on CLI context
- Functions are easily testable in isolation
- Functions could theoretically be used in a library context

**Example**:
```go
// internal/shared/font.go
func FormatFontNameWithVariant(fontName, variant string) string {
    // Pure function, no CLI dependencies
    if variant == "" || strings.EqualFold(variant, "regular") {
        return fontName
    }
    // ... formatting logic
}

// internal/shared/file.go
func FormatFileSize(bytes int64) string {
    // General-purpose utility, could be used anywhere
    // ...
}
```

---

### `internal/functions/` - Domain-Specific Utilities

**Purpose**: Utilities that are specific to a particular domain or feature area

**Contains**:
- Source sorting utilities (`SortSources`)
- Validation utilities (domain-specific validation)

**When to Use**:
- ✅ Utilities that are specific to a particular domain (e.g., sources management)
- ✅ Code that operates on domain-specific types
- ✅ Code that's tightly coupled to a specific feature area

**When NOT to Use**:
- ❌ General-purpose utilities (use `internal/shared/` instead)
- ❌ CLI-specific code (use `internal/cmdutils/` instead)

**Key Characteristics**:
- Functions operate on domain-specific types (e.g., `SourceItem`)
- Functions are specific to a feature area (e.g., sources management)
- Functions might not be reusable outside their domain

**Example**:
```go
// internal/functions/sort.go
func SortSources(sources []SourceItem) {
    // Domain-specific: operates on SourceItem type
    // Specific to sources management feature
    sort.Slice(sources, func(i, j int) bool {
        // ... sorting logic
    })
}
```

---

### `internal/ui/` - User Interface Components

**Purpose**: UI styling, components, and table formatting

**Contains**:
- Styling definitions (`styles.go`)
- UI component utilities (`components.go`)
- Table formatting (`tables.go`)

**Guidelines**:
- ✅ All table formatting constants and functions
- ✅ All styling definitions
- ✅ UI rendering utilities
- ❌ Business logic
- ❌ Platform-specific code

---

### `internal/output/` - Output Management

**Purpose**: Console output management (verbose/debug)

**Contains**:
- Verbose output (`verbose.go`)
- Debug output (`debug.go`)
- Status reporting (`status.go`)

**Guidelines**:
- ✅ All console output formatting
- ✅ Status report types and functions
- ❌ File logging (use `internal/logging/` instead)
- ❌ Business logic

---

### `internal/platform/` - Platform Abstraction

**Purpose**: Cross-platform operations and platform-specific implementations

**Contains**:
- Platform abstraction (`platform.go`)
- Platform-specific implementations (`windows.go`, `darwin.go`, `linux.go`)
- Platform utilities (`scope.go`, `temp.go`, `windows_utils.go`)

**Guidelines**:
- ✅ Platform-specific code
- ✅ Cross-platform abstractions
- ✅ Platform-related utilities (e.g., scope detection, temp file management)
- ❌ Business logic
- ❌ CLI-specific code

---

### `internal/repo/` - Font Repository

**Purpose**: Font repository management and data access

**Contains**:
- Repository operations
- Font data structures
- Font matching logic
- Source management

**Guidelines**:
- ✅ Repository data access
- ✅ Font matching and search
- ✅ Repository-specific business logic
- ❌ General utilities (use `internal/shared/` instead)
- ❌ CLI-specific wrappers (use `internal/cmdutils/` instead)

---

## Decision Tree: Where Should This Code Go?

### Is it a CLI command implementation?
→ **`cmd/`**

### Does it need Cobra context or CLI-specific error handling?
→ **`internal/cmdutils/`**
- Examples: `EnsureManifestInitialized`, `CheckElevation`, `ParseFontNames`

### Is it a general-purpose utility that could be used anywhere?
→ **`internal/shared/`**
- Examples: `FormatFileSize`, `FormatFontNameWithVariant`, `FindSimilarFonts`

### Is it specific to a particular domain/feature?
→ **`internal/functions/`** or domain-specific package
- Examples: `SortSources` (sources domain)

### Is it UI-related (styling, tables, components)?
→ **`internal/ui/`** or **`internal/components/`**

### Is it output-related (verbose/debug/status)?
→ **`internal/output/`**

### Is it platform-specific or cross-platform abstraction?
→ **`internal/platform/`**

### Is it repository/data access related?
→ **`internal/repo/`**

---

## Common Patterns

### Pattern 1: CLI Wrapper in `cmdutils`, Business Logic in `shared`

```go
// internal/cmdutils/repository.go
func GetRepository(refresh bool, logger Logger) (*repo.Repository, error) {
    // CLI-specific: logging, error formatting
    r, err := repo.GetRepository() // Business logic in repo package
    if err != nil {
        if logger != nil {
            logger.Error("Failed to get repository: %v", err)
        }
        output.GetVerbose().Error("%v", err)
        return nil, fmt.Errorf("unable to load font repository: %w", err)
    }
    return r, nil
}

// internal/shared/repository.go
func ResolveFontQuery(fontName string) (*FontResolutionResult, error) {
    // Pure business logic, no CLI dependencies
    // ...
}
```

### Pattern 2: General Utility in `shared`

```go
// internal/shared/file.go
func FormatFileSize(bytes int64) string {
    // Pure utility, no dependencies
    // Can be used by commands, tests, or any package
}
```

### Pattern 3: Domain-Specific Utility in `functions`

```go
// internal/functions/sort.go
func SortSources(sources []SourceItem) {
    // Domain-specific: operates on SourceItem type
    // Specific to sources management
}
```

---

## Anti-Patterns to Avoid

### ❌ Don't Create `cmd/shared.go`
**Why**: Mixes CLI-specific and general-purpose code
**Solution**: Use `internal/cmdutils/` for CLI-specific, `internal/shared/` for general-purpose

### ❌ Don't Put General Utilities in `cmdutils`
**Why**: `cmdutils` is for CLI-specific code only
**Solution**: Use `internal/shared/` for general utilities

### ❌ Don't Put CLI-Specific Code in `shared`
**Why**: `shared` should be CLI-agnostic
**Solution**: Use `internal/cmdutils/` for CLI-specific code

### ❌ Don't Create Circular Dependencies
**Why**: Makes code hard to test and maintain
**Solution**: Follow dependency direction: `cmd/` → `internal/` → lower-level packages

### ❌ Don't Mix Concerns in One Package
**Why**: Violates separation of concerns
**Solution**: Split into appropriate packages based on responsibility

---

## Summary

| Package | Purpose | When to Use |
|---------|---------|-------------|
| `cmd/` | CLI commands | Command implementations |
| `internal/cmdutils/` | CLI-specific utilities | Code that needs CLI context, Cobra, or CLI error handling |
| `internal/shared/` | General-purpose utilities | Pure utilities that could be used anywhere |
| `internal/functions/` | Domain-specific utilities | Utilities specific to a feature domain |
| `internal/ui/` | UI components | Styling, tables, UI rendering |
| `internal/output/` | Output management | Verbose/debug/status output |
| `internal/platform/` | Platform abstraction | Cross-platform or platform-specific code |
| `internal/repo/` | Repository | Font repository and data access |

**Key Principle**: When in doubt, ask: "Could this code be used outside of a CLI context?" If yes → `shared`, if no → `cmdutils`.

