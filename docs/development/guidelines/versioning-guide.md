# FontGet Versioning Guide

## 📋 Overview

This guide explains how versioning works in FontGet, from development builds to production releases. It covers semantic versioning, git tags, CI/CD integration, and the self-update system.

**Related Documents:**
- `docs/CI_CD_PLAN.md` - How CI/CD uses versions

---

## 🎯 What is Versioning?

**Versioning** is a way to identify different releases of your software. Each version number tells users:
- **What changed** (major features, bug fixes, breaking changes)
- **When it was released** (chronological order)
- **If they should update** (compatibility, security fixes)

Think of it like book editions:
- **1st edition** → **2nd edition** (major rewrite = MAJOR version)
- **2nd edition** → **2nd edition, revised** (new chapters = MINOR version)
- **2nd edition, revised** → **2nd edition, revised (typo fix)** (small fixes = PATCH version)

---

## 📐 Semantic Versioning (SemVer)

FontGet uses **Semantic Versioning** (SemVer), the industry standard for version numbers.

### **Format: `MAJOR.MINOR.PATCH`**

Examples: `1.0.0`, `1.2.3`, `2.0.0`

### **What Each Number Means**

#### **MAJOR** (first number)
- **When to increment**: Breaking changes that break compatibility
- **Examples**:
  - Removed a command or flag
  - Changed config file format (old configs won't work)
  - Changed API/behavior in a way that breaks existing scripts
- **Example**: `1.2.3` → `2.0.0`

#### **MINOR** (second number)
- **When to increment**: New features, backward compatible
- **Examples**:
  - Added a new command (`fontget backup`)
  - Added new flags to existing commands
  - New functionality that doesn't break existing behavior
- **Example**: `1.2.3` → `1.3.0`

#### **PATCH** (third number)
- **When to increment**: Bug fixes, backward compatible
- **Examples**:
  - Fixed a crash
  - Fixed incorrect output
  - Performance improvements
  - Security fixes
- **Example**: `1.2.3` → `1.2.4`

### **Rules of SemVer**

1. **Start at 0.1.0** for initial development
2. **Increment PATCH** for bug fixes
3. **Increment MINOR** for new features (reset PATCH to 0)
4. **Increment MAJOR** for breaking changes (reset MINOR and PATCH to 0)
5. **Never skip versions** (don't go from 1.2.3 to 1.5.0)

---

## 🔄 How Versioning Works in FontGet

### **Current Setup**

FontGet has a version system in `internal/version/version.go`:

```go
var Version = "dev"  // Default for development builds
```

### **Version Sources (Priority Order)**

1. **Git Tag** (source of truth for releases) - Used in releases
   - Tag format: `v1.2.3` (with `v` prefix)
   - Created by: Developer when creating a release
   - Read by: GoReleaser in GitHub Actions
   - Example: `git tag -a v1.2.3 -m "Release v1.2.3"`

2. **Build-time Injection** (how version gets into binary) - Used in releases
   - GoReleaser reads git tag and injects via `ldflags`
   - Format: `-X fontget/internal/version.Version=1.2.3`
   - Extracted from: Git tag (strips `v` prefix)
   - Used by: GoReleaser in GitHub Actions
   - Example: `fontget version` → `FontGet v1.2.3`

3. **Build Info** (fallback) - Used with `go install`
   - From `go.mod` if available
   - Extracted from `runtime/debug.BuildInfo`
   - Example: `go install github.com/Graphixa/FontGet@v1.2.3`

4. **Default: "dev" or "dev+{hash}"** - Used for local builds
   - When building with `go build` (no ldflags)
   - When no git tag is available
   - Automatically detects git commit hash at runtime (if available)
   - Identifies development/local builds
   - Example: `fontget version` → `FontGet dev+124d611` (or `FontGet dev` if git unavailable)

### **Version Flow**

```
Developer creates tag: v1.2.3
         ↓
GitHub Actions detects tag
         ↓
GoReleaser extracts version: 1.2.3
         ↓
Builds binaries with: -X version.Version=1.2.3
         ↓
Creates GitHub Release: "v1.2.3"
         ↓
Self-update system checks: "Is 1.2.3 > current version?"
         ↓
User runs: fontget update → Downloads v1.2.3
```

---

## 🏷️ Git Tags and Versions

### **What are Git Tags?**

Git tags are **bookmarks** in your git history that mark specific commits as important (like releases).

Think of them like:
- **Commits** = pages in a book
- **Tags** = bookmarks marking "Chapter 1", "Chapter 2", etc.

### **Tag Format**

FontGet uses **annotated tags** with the `v` prefix:

```bash
v1.2.3    # ✅ Correct format
1.2.3     # ⚠️ Works, but less common
v1.2      # ❌ Missing PATCH number
v1.2.3.4  # ❌ Too many numbers
```

### **Creating a Release Tag**

```bash
# 1. Make sure you're on main branch and up to date
git checkout main
git pull origin main

# 2. Create an annotated tag with a message
git tag -a v1.2.3 -m "Release v1.2.3: Added backup command and fixed font matching bug"

# 3. Push the tag to GitHub (this triggers CI/CD)
git push origin v1.2.3
```

### **Why Annotated Tags?**

- **Annotated tags** (`-a` flag) store extra metadata (author, date, message)
- **Lightweight tags** are just pointers (less useful for releases)
- GoReleaser and self-update systems prefer annotated tags

---

## 🤔 How to Decide Version Numbers

### **Decision Tree**

```
Did you make a breaking change?
├─ YES → Increment MAJOR (1.2.3 → 2.0.0)
└─ NO → Did you add new features?
    ├─ YES → Increment MINOR (1.2.3 → 1.3.0)
    └─ NO → Increment PATCH (1.2.3 → 1.2.4)
```

### **Common Scenarios**

#### **Scenario 1: Bug Fix**
- **Current**: `v1.2.3`
- **Change**: Fixed crash when font name contains special characters
- **Decision**: PATCH increment
- **New Version**: `v1.2.4`

#### **Scenario 2: New Feature**
- **Current**: `v1.2.3`
- **Change**: Added `fontget backup` command
- **Decision**: MINOR increment
- **New Version**: `v1.3.0`

#### **Scenario 3: Multiple Changes**
- **Current**: `v1.2.3`
- **Changes**: 
  - Added `fontget backup` command (new feature)
  - Fixed font matching bug (bug fix)
- **Decision**: MINOR increment (new feature takes priority)
- **New Version**: `v1.3.0`
- **Note**: Bug fixes are included in MINOR releases

#### **Scenario 4: Breaking Change**
- **Current**: `v1.2.3`
- **Change**: Removed `--force` flag (breaking change)
- **Decision**: MAJOR increment
- **New Version**: `v2.0.0`

#### **Scenario 5: Security Fix**
- **Current**: `v1.2.3`
- **Change**: Fixed security vulnerability
- **Decision**: PATCH increment (but high priority)
- **New Version**: `v1.2.4`

### **What Counts as "Breaking"?**

**✅ Breaking Changes:**
- Removing a command or flag
- Changing config file format
- Changing default behavior in a way that breaks scripts
- Removing or renaming exported functions (for libraries)

**❌ NOT Breaking Changes:**
- Adding new commands/flags
- Changing internal implementation
- Performance improvements
- Bug fixes that change incorrect behavior

---

## 🚀 Versioning Workflow

### **Local Development Builds**

When you build FontGet locally with `go build`, the version automatically includes the git commit hash:

```bash
$ go build -o fontget
$ ./fontget version
FontGet dev+124d611
```

**How it works:**
- ✅ Simple `go build` - no special flags needed!
- ✅ Automatically detects git commit hash at runtime (when run from git repo)
- ✅ Shows `dev+{hash}` format (e.g., `dev+124d611`)
- ✅ Falls back to `dev` if git is not available or not in a git repo

**Why `dev+{hash}`?**
- ✅ Clearly identifies it's a development build (not from a release)
- ✅ Includes commit hash for precise tracking (`dev+124d611`)
- ✅ SemVer-compliant format (build metadata uses `+`)
- ✅ Self-update system recognizes "dev" builds and will always suggest updates
- ✅ Standard practice in Go projects

**Optional: Build-time Injection (for distribution)**

If you want to bake the commit hash into the binary (useful for distributing local builds):

```bash
# Linux/macOS
COMMIT=$(git rev-parse --short HEAD)
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
go build -ldflags "-X fontget/internal/version.GitCommit=$COMMIT -X fontget/internal/version.BuildDate=$DATE" -o fontget

# Windows (PowerShell)
$commit = git rev-parse --short HEAD
$date = (Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ" -AsUTC)
go build -ldflags "-X fontget/internal/version.GitCommit=$commit -X fontget/internal/version.BuildDate=$date" -o fontget.exe
```

**Note:** The version will remain `"dev"` or `"dev+{hash}"` unless you explicitly set it via `-X fontget/internal/version.Version=<version>`. This is intentional—local builds should use "dev" to distinguish them from releases.

### **Pre-Release Checklist**

Before creating a release:

1. ✅ All features complete
2. ✅ All tests pass
3. ✅ Code merged to `main`
4. ✅ Documentation updated
5. ✅ **Version number decided** (using decision tree above)

### **Creating a Release**

1. **Check current version:**
   ```bash
   git describe --tags --abbrev=0
   # Output: v1.2.3
   ```

2. **Determine next version:**
   - Review changes since last release
   - Use decision tree to determine MAJOR/MINOR/PATCH
   - Example: Last was `v1.2.3`, added new feature → `v1.3.0`

3. **Create and push tag:**
   ```bash
   git tag -a v1.3.0 -m "Release v1.3.0: Added backup command"
   git push origin v1.3.0
   ```

4. **CI/CD automatically:**
   - Builds binaries for all platforms
   - Creates GitHub Release
   - Uploads binaries and checksums
   - Self-update system can now detect the new version

---

## 🔗 Integration with CI/CD

### **How GoReleaser Uses Versions**

GoReleaser automatically:

1. **Detects git tag**: When you push `v1.2.3`, GoReleaser detects it
2. **Extracts version**: Strips `v` prefix → `1.2.3`
3. **Injects version**: Uses `ldflags` to set `version.Version=1.2.3`
4. **Creates release**: GitHub Release titled "v1.2.3"
5. **Names binaries**: `fontget-windows-amd64.exe` (for self-update compatibility, version is in the binary, not filename)

### **Version in Build Process**

```yaml
# GoReleaser config (simplified)
builds:
  - ldflags:
      - -X fontget/internal/version.Version={{ .Version }}
      - -X fontget/internal/version.GitCommit={{ .Commit }}
      - -X fontget/internal/version.BuildDate={{ .Date }}
```

GoReleaser automatically fills in:
- `{{ .Version }}` → `1.2.3` (from git tag)
- `{{ .Commit }}` → `abc1234` (from git commit)
- `{{ .Date }}` → `2025-01-15T10:30:00Z` (build timestamp)

---

## 🔄 Integration with Self-Update System

### **How Self-Update Uses Versions**

The self-update system (`rhysd/go-github-selfupdate`) uses versions to:

1. **Check for updates**: Compares current version vs. latest GitHub release
2. **Version comparison**: Uses semantic versioning to determine if update is available
3. **Download correct binary**: Finds binary matching current platform and version

### **Version Comparison**

```go
// Library automatically compares:
currentVersion := "1.2.3"
latestVersion := "1.2.4"

// Library determines: 1.2.4 > 1.2.3 → Update available

// Special handling for dev builds:
// "dev" and "dev+{hash}" are treated as 0.0.0 for comparison
// This ensures dev builds always see updates as available
```

### **GitHub Release Requirements**

For self-update to work, GitHub releases must:

1. ✅ Use semantic version tags: `v1.2.3`
2. ✅ Include binaries named: `fontget-{os}-{arch}{.ext}`
3. ✅ Include `checksums.txt` file
4. ✅ Release title matches tag: `v1.2.3`

---

## 📊 Version Examples

### **Example Release History**

```
v0.1.0  - Initial release (basic font installation)
v0.2.0  - Added search command (new feature)
v0.2.1  - Fixed search crash bug (bug fix)
v0.3.0  - Added list command (new feature)
v0.3.1  - Fixed list performance (bug fix)
v0.3.2  - Security fix (bug fix)
v1.0.0  - First stable release (breaking: removed experimental features)
v1.1.0  - Added backup command (new feature)
v1.1.1  - Fixed backup zip corruption (bug fix)
v1.2.0  - Added export/import commands (new feature)
v1.2.1  - Fixed export manifest format (bug fix)
v2.0.0  - Major refactor (breaking: changed config format)
v2.1.0  - Added new source support (new feature)
```

### **Reading Version History**

- **v0.x.x**: Pre-1.0 (development, may have breaking changes)
- **v1.0.0**: First stable release
- **v1.x.x**: Stable releases (backward compatible)
- **v2.0.0**: Major version (may have breaking changes)

---

## 🎓 Pre-Releases (Alpha, Beta, RC)

> [!NOTE]
> **FontGet currently uses stable releases only.** Pre-release tags (alpha, beta, rc) are supported by SemVer and the versioning system, but FontGet typically releases directly to stable. This section documents pre-releases for reference, but they may not be used in practice.

### **Pre-Release Formats**

SemVer supports pre-release versions:

- **Alpha**: `v1.2.3-alpha.1`, `v1.2.3-alpha.2`
- **Beta**: `v1.2.3-beta.1`, `v1.2.3-beta.2`
- **Release Candidate**: `v1.2.3-rc.1`, `v1.2.3-rc.2`

### **When to Use Pre-Releases**

- **Alpha**: Early testing, unstable features
- **Beta**: Feature complete, testing for bugs
- **RC**: Release candidate, final testing before stable

### **Pre-Release Workflow** (If Needed)

```bash
# 1. Create beta release (if needed)
git tag -a v1.3.0-beta.1 -m "Beta release: Testing backup command"
git push origin v1.3.0-beta.1

# 2. After testing, create stable release
git tag -a v1.3.0 -m "Stable release: Backup command"
git push origin v1.3.0
```

### **Self-Update and Pre-Releases**

By default, self-update system only checks **stable releases** (no pre-release tags).

To check pre-releases, users would need to:
- Manually download beta releases
- Or configure update channel (future feature)

---

## ⚠️ Common Pitfalls

### **Pitfall 1: Skipping Versions**
❌ **Wrong**: `v1.2.3` → `v1.5.0` (skipped 1.3.0, 1.4.0)
✅ **Correct**: `v1.2.3` → `v1.3.0` → `v1.4.0` → `v1.5.0`

### **Pitfall 2: Wrong Increment Type**
❌ **Wrong**: Added new feature but incremented PATCH (`v1.2.3` → `v1.2.4`)
✅ **Correct**: Added new feature, increment MINOR (`v1.2.3` → `v1.3.0`)

### **Pitfall 3: Forgetting to Reset**
❌ **Wrong**: `v1.2.3` → `v2.1.4` (should reset MINOR and PATCH)
✅ **Correct**: `v1.2.3` → `v2.0.0` (MAJOR increments reset others)

### **Pitfall 4: Tag Format**
❌ **Wrong**: `1.2.3` (missing `v` prefix)
✅ **Correct**: `v1.2.3` (with `v` prefix)

### **Pitfall 5: Reusing Tags**
❌ **Wrong**: Delete and recreate tag `v1.2.3`
✅ **Correct**: Use new version `v1.2.4` (tags are immutable)

---

## 📝 Best Practices

### **1. Start with 0.1.0**
- First release should be `v0.1.0` (not `v1.0.0`)
- `v1.0.0` should be reserved for "first stable release"

### **2. Be Conservative**
- When in doubt, use a smaller increment
- Better to release `v1.2.4` than `v1.3.0` if unsure

### **3. Document Breaking Changes**
- Always document what broke in MAJOR releases
- Consider migration guides

### **4. Regular Releases**
- Don't wait too long between releases
- Small, frequent releases are better than large, infrequent ones

### **5. Use Meaningful Tag Messages**
```bash
# ✅ Good
git tag -a v1.3.0 -m "Release v1.3.0: Added backup command and improved font matching"

# ❌ Bad
git tag -a v1.3.0 -m "Release"
```

---

## 🔍 Checking Versions

### **Check Current Version**
```bash
fontget version
# Output: FontGet v1.2.3

# With detailed build info (--debug flag)
fontget version --debug
# Output:
# FontGet v1.2.3
# Commit: abc1234
# Build: 2025-01-15T10:30:00Z
```

### **Check Latest Tag**
```bash
git describe --tags --abbrev=0
# Output: v1.2.3
```

### **Check All Tags**
```bash
git tag -l
# Output:
# v0.1.0
# v0.2.0
# v1.0.0
# v1.1.0
# v1.2.3
```

### **Check Version in Code**
```go
import "fontget/internal/version"

currentVersion := version.GetVersion()
// Returns: "1.2.3" or "dev"
```

---

## 🎯 Quick Reference

### **Version Decision Matrix**

| Change Type | Example | Increment | Example |
|------------|---------|-----------|---------|
| Bug fix | Fixed crash | PATCH | `1.2.3` → `1.2.4` |
| New feature | Added command | MINOR | `1.2.3` → `1.3.0` |
| Breaking change | Removed flag | MAJOR | `1.2.3` → `2.0.0` |
| Security fix | Fixed vulnerability | PATCH | `1.2.3` → `1.2.4` |
| Performance | Faster search | PATCH | `1.2.3` → `1.2.4` |

### **Tag Creation Command**
```bash
git tag -a v<MAJOR>.<MINOR>.<PATCH> -m "Release v<MAJOR>.<MINOR>.<PATCH>: <Description>"
git push origin v<MAJOR>.<MINOR>.<PATCH>
```

### **Version Format**
- ✅ `v1.2.3` (recommended)
- ✅ `1.2.3` (works, but less common)
- ❌ `v1.2` (missing PATCH)
- ❌ `v1.2.3.4` (too many numbers)

---

## ❓ FAQ

### **Q: What version should I use for the first release?**
**A**: Start with `v0.1.0`. Reserve `v1.0.0` for when you're confident the API is stable.

### **Q: Can I change a version after releasing?**
**A**: No, tags are immutable. If you need to fix a release, create a new version (e.g., `v1.2.4`).

### **Q: What if I make a mistake in version number?**
**A**: Create a new tag with the correct version. Don't delete the old tag (it may be referenced).

### **Q: How often should I release?**
**A**: There's no fixed schedule. Release when you have meaningful changes (bug fixes, features, security fixes).

### **Q: Should I version every commit?**
**A**: No, only version releases. Daily commits don't need versions (they use `"dev"`).

### **Q: What about pre-1.0 versions?**
**A**: Pre-1.0 versions (0.x.x) can have breaking changes without incrementing MAJOR. After 1.0.0, follow SemVer strictly.

---

## 📚 Additional Resources

- [Semantic Versioning Specification](https://semver.org/)
- [GoReleaser Documentation](https://goreleaser.com/)
- [Git Tagging Best Practices](https://git-scm.com/book/en/v2/Git-Basics-Tagging)
- [GitHub Releases API](https://docs.github.com/en/rest/releases/releases)

---

**Last Updated**: 2025-01-XX  
**Status**: Active Guide

