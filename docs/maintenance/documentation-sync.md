# FontGet Documentation Sync Process

This document outlines the process for keeping FontGet's command reference documentation in sync with the actual codebase.

## Overview

The FontGet CLI tool has many flags and commands that need to be accurately documented. This process ensures that the `docs/help.md` file stays current with the actual implementation.

## Audit Script

### Location
`docs/maintenance/audit-flags.go`

### Purpose
Automatically scans all command files in the `cmd/` directory and extracts:
- All command definitions
- All flag registrations (StringP, BoolP, IntP)
- Flag types, short forms, defaults, and usage descriptions
- Global vs local flag classification

### Usage
```bash
# Run from project root
go run docs/maintenance/audit-flags.go cmd/
```

### Output
The script provides:
1. **Complete flag inventory** - Lists all commands and their flags
2. **Documentation sync check** - Identifies flags missing from documentation
3. **Summary statistics** - Total commands, flags, and global vs local counts

## Documentation Checklist

Before each release, verify the following:

### ✅ Command Coverage
- [ ] All commands in `cmd/` directory are documented
- [ ] All subcommands are listed in the Quick Reference table
- [ ] Command purposes and examples are accurate

### ✅ Flag Coverage
- [ ] All flags from audit script are in documentation
- [ ] Flag short forms (-v, -f, etc.) are documented
- [ ] Flag descriptions match the actual usage strings
- [ ] Global flags are clearly marked as "Available on all commands"

### ✅ Flag Descriptions
- [ ] Flag descriptions are clear and helpful
- [ ] Flag values and options are documented (e.g., scope: user/machine/all)
- [ ] Flag combinations are explained where relevant

### ✅ Examples
- [ ] Examples show real flag usage
- [ ] Examples cover common use cases
- [ ] Examples are tested and work correctly

## Manual Verification Steps

### 1. Run the Audit Script
```bash
go run docs/maintenance/audit-flags.go cmd/
```

### 2. Check for Missing Flags
Look for any flags listed in the "Missing flags in documentation" section and add them to `docs/help.md`.

### 3. Verify Flag Descriptions
Compare the audit output with the documentation to ensure:
- Flag names match exactly
- Short forms are correct
- Usage descriptions are accurate
- Flag types are appropriate

### 4. Test Examples
Run the documented examples to ensure they work:
```bash
# Test global flags
fontget --verbose --help
fontget --logs

# Test command-specific flags
fontget add --help
fontget search --help
fontget list --help
# ... etc for all commands
```

### 5. Check Help Output
Compare `--help` output with documentation:
```bash
fontget --help
fontget add --help
fontget search --help
# ... etc
```

## Common Issues to Watch For

### Flag Name Mismatches
- Documentation shows `--limit` but code uses `--category`
- Short form inconsistencies (`-l` vs `-c`)

### Missing Global Flags
- Global flags (like `--verbose`) should be documented as available on all commands
- Don't repeat global flags in individual command sections

### Outdated Examples
- Examples that use removed or renamed flags
- Examples that don't reflect current behavior

### Incomplete Flag Descriptions
- Flags without usage descriptions
- Flags without value explanations (e.g., what values `--scope` accepts)

## Maintenance Schedule

### Before Each Release
1. Run the audit script
2. Update documentation for any new flags
3. Verify all examples work
4. Check help output consistency

### When Adding New Commands
1. Add command to `command-reference.md`
2. Include all flags in Quick Reference table
3. Add detailed flag descriptions
4. Provide working examples

### When Adding New Flags
1. Add flag to appropriate command section
2. Update Quick Reference table
3. Add to Flag Reference section
4. Test and document examples

## Quick Reference Template

When adding new commands, use this template:

```markdown
### `command-name`
**Purpose**: Brief description of what the command does  
**Why it matters**: Why users would need this command  
**Subcommands**: List any subcommands (or "None" if none)

```bash
# Basic usage example
fontget command-name "example"

# Example with flags
fontget command-name "example" --flag value
```
```

## Flag Reference Template

When adding new flags, use this template:

```markdown
#### Category Name
- `--flag-name, -f` - Description of what the flag does:
  - `value1` - Description of this value
  - `value2` - Description of this value
```

## Automation Ideas

### Future Enhancements
- Automated documentation generation from code comments
- CI/CD integration to fail builds when documentation is out of sync
- Automated example testing
- Flag usage validation

### Current Limitations
- Manual flag extraction (could be improved with AST parsing)
- No automated example testing
- No validation of flag descriptions against actual usage

## Contact

If you find discrepancies between the code and documentation, please:
1. Run the audit script to confirm
2. Update the documentation
3. Test the changes
4. Submit a pull request

This process ensures FontGet's documentation remains accurate and helpful for all users.
