# Spacing Guidelines

## Core Principle
**Commands always start with a blank line, and every output section ends with exactly one blank line (`\n\n` or `fmt.Println()`).**

This simple rule eliminates the need for conditional spacing logic:
- Commands add a blank line at the very start (once, at the top of `RunE`)
- Each section is responsible for its own trailing spacing
- No "add blank line before" logic needed anywhere

## What is a "Section"?

A section is a logical unit of output:
1. **Verbose info block** - `[INFO] Scope: user` etc.
2. **Progress bar output** - The TUI progress bar and item list
3. **Error messages** - "The following fonts were not found..."
4. **Warning messages** - "The following fonts are still installed..."
5. **Status report** - The status report table

## Rules

### 1. Every Section Ends with a Blank Line
- After the last line of content in a section, add `fmt.Println()` or ensure the last `fmt.Printf` ends with `\n\n`
- **No conditionals** - every section always ends with a blank line
- This creates consistent spacing between sections automatically

### 2. Progress Bar Component (`progress_bar.go`)
- **Title line**: Starts with `\n`, ends with `\n` (creates space before items)
- **Items**: Each item ends with `\n`
- **Final output**: Must end with `\n\n` (one `\n` from last item, one `\n` for blank line)

### 3. Status Report (`PrintStatusReport`)
- Starts with `\n` (creates blank line before it)
- Ends with `\n\n` (blank line after it, ready for prompt)
- Only shown in verbose mode

### 4. Error/Warning/Info Messages
- Each message line ends with `\n`
- The section's last line should be followed by `fmt.Println()` to create the trailing blank line

### 5. Verbose Output
- Each verbose info line ends with `\n` (handled by `output.GetVerbose().Info()`)
- After the last verbose info line in a section, add a blank line only if verbose output was enabled:
  ```go
  output.GetVerbose().Info("Scope: %s", scope)
  output.GetVerbose().Info("Removing %d font(s)", count)
  // Verbose section ends with blank line per spacing framework (only if verbose was shown)
  if IsVerbose() {
      fmt.Println()
  }
  ```
- This ensures proper spacing when verbose mode is active, without adding unnecessary blank lines when verbose is disabled

## Examples

### Normal install (no errors, no verbose)
```
Progress bar output (ends with \n\n)
Prompt
```

### Install with "not found" fonts
```
Progress bar output (ends with \n\n)
The following fonts were not found... (ends with \n\n)
Prompt
```

### Verbose install
```
Verbose info (ends with \n\n)
Progress bar output (ends with \n\n)
Status report (ends with \n\n)
Prompt
```

## Implementation

### Before (Complex Conditional Logic)
```go
if len(notFoundFonts) > 0 {
    fmt.Println() // Conditional spacing
    fmt.Printf("The following fonts...\n")
    for _, font := range notFoundFonts {
        fmt.Printf("  - %s\n", font)
    }
    fmt.Println() // More conditional spacing
    fmt.Printf("Try using...\n")
    if len(fontsInOppositeScope) == 0 {
        fmt.Println() // Yet more conditionals
    }
}
```

### After (Simple Rule)
```go
if len(notFoundFonts) > 0 {
    fmt.Printf("The following fonts...\n")
    for _, font := range notFoundFonts {
        fmt.Printf("  - %s\n", font)
    }
    fmt.Println() // Blank line within section (before hint)
    fmt.Printf("Try using...\n")
    fmt.Println() // Section ALWAYS ends with blank line
}
```

## Benefits

1. **Simple**: One rule, easy to remember
2. **Consistent**: Every section behaves the same way
3. **No Conditionals**: No `if IsVerbose()` or `if len(x) == 0` checks for spacing
4. **Easy to Review**: Just check "does this section end with a blank line?"

