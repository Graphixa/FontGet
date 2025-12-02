# FontGet Release Guide

## 📋 Overview

This guide provides a step-by-step process for creating releases in FontGet. It covers when to release, how to determine version numbers, and the exact commands needed to create and publish a release.

**Related Documentation:**
- [Versioning Guide](./versioning-guide.md) - Detailed explanation of semantic versioning and version decision logic
- [Contributing Guide](../contributing.md) - General contribution guidelines

---

## 🎯 When to Release

### **Release When You Have:**

1. **New Features** - Added functionality that users will benefit from
2. **Bug Fixes** - Fixed issues that affect users
3. **Security Fixes** - Critical security patches (release immediately)
4. **Breaking Changes** - Incompatible changes that require user action
5. **Documentation Updates** - Significant documentation improvements (usually bundled with features)

### **Don't Release For:**

- ❌ Work-in-progress features (wait until complete)
- ❌ Internal refactoring only (unless it's a breaking change)
- ❌ Every single commit (batch related changes)
- ❌ Unmerged pull requests (must be on `main` branch)

### **Release Frequency**

- **No fixed schedule** - Release when you have meaningful changes
- **Regular releases** - Better to release frequently with small changes than infrequently with large changes
- **Security fixes** - Release immediately, even if it's just a PATCH increment

---

## 📐 Determining Version Numbers

FontGet uses **Semantic Versioning (SemVer)** in the format `MAJOR.MINOR.PATCH`.

### **Quick Decision Tree**

```
Did you make a breaking change?
├─ YES → Increment MAJOR (1.2.3 → 2.0.0)
└─ NO → Did you add new features?
    ├─ YES → Increment MINOR (1.2.3 → 1.3.0)
    └─ NO → Increment PATCH (1.2.3 → 1.2.4)
```

### **Version Increment Rules**

| Change Type | Example | Increment | Example |
|------------|---------|-----------|---------|
| **Breaking change** | Removed command, changed config format | **MAJOR** | `1.2.3` → `2.0.0` |
| **New feature** | Added new command, new flags | **MINOR** | `1.2.3` → `1.3.0` |
| **Bug fix** | Fixed crash, corrected output | **PATCH** | `1.2.3` → `1.2.4` |
| **Security fix** | Fixed vulnerability | **PATCH** | `1.2.3` → `1.2.4` |
| **Performance** | Faster search, reduced memory | **PATCH** | `1.2.3` → `1.2.4` |

### **What Counts as Breaking?**

**✅ Breaking Changes:**
- Removing a command or flag
- Changing config file format (old configs won't work)
- Changing default behavior in a way that breaks existing scripts
- Removing or renaming exported functions (for libraries)

**❌ NOT Breaking Changes:**
- Adding new commands/flags
- Changing internal implementation
- Performance improvements
- Bug fixes that change incorrect behavior

### **Multiple Changes in One Release**

If your release includes multiple types of changes:
- **Breaking + Features + Fixes** → **MAJOR** increment (breaking takes priority)
- **Features + Fixes** → **MINOR** increment (features take priority)
- **Fixes only** → **PATCH** increment

**Example:**
- Current: `v1.2.3`
- Changes: Added backup command (feature) + Fixed crash bug (fix)
- Decision: **MINOR** increment → `v1.3.0` (bug fix included in minor release)

For detailed version decision logic, see the [Versioning Guide](./versioning-guide.md#how-to-decide-version-numbers).

---

## ✅ Pre-Release Checklist

Before creating a release, ensure:

- [ ] **All features complete** - No work-in-progress code
- [ ] **All tests pass** - Run tests locally and verify CI passes
- [ ] **Code merged to `main`** - All changes are on the main branch
- [ ] **Documentation updated** - README, guides, and help text are current
- [ ] **Version number decided** - Use decision tree above
- [ ] **Changelog reviewed** - Know what's changed since last release
- [ ] **No breaking changes** (or documented if there are)

### **Verifying Changes Since Last Release**

```bash
# Check latest tag
git describe --tags --abbrev=0
# Output: v1.2.3

# Review commits since last release
git log v1.2.3..HEAD --oneline

# Review detailed changes
git log v1.2.3..HEAD --pretty=format:"%h - %s (%an, %ar)"
```

---

## 🚀 Release Process

### **Step 1: Ensure You're on Main Branch**

```bash
# Check current branch
git branch --show-current
# Should output: main

# If not on main, switch to it
git checkout main

# Pull latest changes
git pull origin main
```

### **Step 2: Check Current Version**

```bash
# Get the latest tag
git describe --tags --abbrev=0
# Output: v1.2.3

# List all tags (optional, for reference)
git tag -l
```

### **Step 3: Determine Next Version**

1. **Review changes** since last release (see commands above)
2. **Use decision tree** to determine MAJOR/MINOR/PATCH
3. **Calculate next version**:
   - Last: `v1.2.3`
   - Added feature → `v1.3.0` (MINOR)
   - Fixed bug → `v1.2.4` (PATCH)
   - Breaking change → `v2.0.0` (MAJOR)

### **Step 4: Create Release Tag**

Use an **annotated tag** with a descriptive message:

```bash
# Format: git tag -a v<MAJOR>.<MINOR>.<PATCH> -m "Release v<MAJOR>.<MINOR>.<PATCH>: <Description>"

# Example for MINOR release
git tag -a v1.3.0 -m "Release v1.3.0: Added interactive TUI onboarding system and theme improvements"

# Example for PATCH release
git tag -a v1.2.4 -m "Release v1.2.4: Fixed Windows terminal detection and update cleanup"

# Example for MAJOR release
git tag -a v2.0.0 -m "Release v2.0.0: Major refactor with breaking config format changes"
```

**Tag Message Best Practices:**
- ✅ Include version number in message
- ✅ List main features/changes
- ✅ Be descriptive but concise
- ❌ Don't use generic messages like "Release" or "Update"

### **Step 5: Verify Tag Created**

```bash
# Verify tag exists
git tag -l v1.3.0

# View tag details
git show v1.3.0
```

### **Step 6: Push Tag to GitHub**

```bash
# Push the tag (this triggers CI/CD)
git push origin v1.3.0

# Or push all tags (if you have multiple)
git push origin --tags
```

**What happens next:**
- GitHub Actions detects the new tag
- GoReleaser builds binaries for all platforms
- GitHub Release is created automatically
- Binaries and checksums are uploaded
- Self-update system can now detect the new version

### **Step 7: Monitor CI/CD Pipeline**

1. Go to GitHub repository → **Actions** tab
2. Find the workflow run triggered by the tag push
3. Wait for build to complete (usually 5-10 minutes)
4. Verify GitHub Release was created with binaries

### **Step 8: Verify Release**

```bash
# Check GitHub Release was created (via web interface)
# Or verify via GitHub CLI if installed
gh release view v1.3.0

# Test self-update (from a previous version)
fontget update
```

---

## 📝 Complete Release Example

Here's a complete example of releasing `v1.3.0`:

```bash
# Step 1: Ensure on main branch
git checkout main
git pull origin main

# Step 2: Check current version
git describe --tags --abbrev=0
# Output: v1.2.3

# Step 3: Review changes
git log v1.2.3..HEAD --oneline
# Review: Added TUI onboarding (feature), fixed bugs (fixes)
# Decision: MINOR increment (new feature)

# Step 4: Create tag
git tag -a v1.3.0 -m "Release v1.3.0: Added interactive TUI onboarding system, theme improvements, and bug fixes"

# Step 5: Verify tag
git show v1.3.0

# Step 6: Push tag
git push origin v1.3.0

# Step 7: Monitor CI/CD (via GitHub web interface)
# Wait for build to complete

# Step 8: Verify release
# Check GitHub Releases page for v1.3.0
```

---

## 🏷️ Tag Format Requirements

### **Correct Format**

- ✅ `v1.2.3` - Recommended (with `v` prefix)
- ✅ `1.2.3` - Works, but less common

### **Incorrect Format**

- ❌ `v1.2` - Missing PATCH number
- ❌ `v1.2.3.4` - Too many numbers
- ❌ `1-2-3` - Wrong separator
- ❌ `release-1.2.3` - Wrong prefix

### **Tag Type**

Always use **annotated tags** (`-a` flag):
- Store metadata (author, date, message)
- Preferred by GoReleaser
- Better for releases

**Don't use lightweight tags** (no `-a` flag) for releases.

---

## ⚠️ Common Mistakes and How to Avoid Them

### **Mistake 1: Skipping Versions**

❌ **Wrong**: `v1.2.3` → `v1.5.0` (skipped 1.3.0, 1.4.0)  
✅ **Correct**: `v1.2.3` → `v1.3.0` → `v1.4.0` → `v1.5.0`

**Why**: SemVer requires sequential versions. Skipping versions confuses users and tools.

### **Mistake 2: Wrong Increment Type**

❌ **Wrong**: Added new feature but incremented PATCH (`v1.2.3` → `v1.2.4`)  
✅ **Correct**: Added new feature, increment MINOR (`v1.2.3` → `v1.3.0`)

**Why**: Version numbers communicate what changed. Wrong increment misleads users.

### **Mistake 3: Forgetting to Reset**

❌ **Wrong**: `v1.2.3` → `v2.1.4` (should reset MINOR and PATCH)  
✅ **Correct**: `v1.2.3` → `v2.0.0` (MAJOR increments reset others)

**Why**: MAJOR increments indicate breaking changes. MINOR and PATCH should reset to 0.

### **Mistake 4: Reusing Tags**

❌ **Wrong**: Delete and recreate tag `v1.2.3`  
✅ **Correct**: Use new version `v1.2.4` (tags are immutable)

**Why**: Tags are permanent markers. Deleting and recreating breaks references.

### **Mistake 5: Tagging Wrong Commit**

❌ **Wrong**: Tag a commit that's not on `main`  
✅ **Correct**: Ensure all changes are merged to `main` before tagging

**Why**: Releases should only come from the main branch.

### **Mistake 6: Generic Tag Messages**

❌ **Wrong**: `git tag -a v1.3.0 -m "Release"`  
✅ **Correct**: `git tag -a v1.3.0 -m "Release v1.3.0: Added backup command and fixed font matching"`

**Why**: Descriptive messages help users understand what changed.

---

## 🔍 Post-Release Verification

After the CI/CD pipeline completes:

### **1. Check GitHub Release**

- [ ] Release exists on GitHub Releases page
- [ ] Release title matches tag: `v1.3.0`
- [ ] Release description is present
- [ ] Binaries are uploaded for all platforms:
  - `fontget-windows-amd64.exe`
  - `fontget-linux-amd64`
  - `fontget-darwin-amd64` (macOS)
  - `fontget-darwin-arm64` (macOS Apple Silicon)
- [ ] `checksums.txt` file is present

### **2. Test Self-Update**

```bash
# From a previous version, test update
fontget update

# Verify version after update
fontget version
# Should show: FontGet v1.3.0
```

### **3. Verify Version in Binary**

```bash
# Download and test a binary
./fontget version
# Should output: FontGet v1.3.0
```

---

## 🔄 Rollback Procedure

If a release has critical issues:

### **Option 1: Hotfix Release (Recommended)**

1. Create a new PATCH release with fixes
2. Example: `v1.3.0` (broken) → `v1.3.1` (fixed)
3. Follow normal release process

### **Option 2: Delete Tag (Not Recommended)**

```bash
# Delete local tag
git tag -d v1.3.0

# Delete remote tag
git push origin --delete v1.3.0
```

**⚠️ Warning**: Only do this if the release hasn't been downloaded yet. Deleting tags that users have already downloaded can cause confusion.

**Better approach**: Create a new PATCH release with fixes instead of deleting tags.

---

## 📊 Release History Tracking

### **View Release History**

```bash
# List all tags
git tag -l

# List tags with messages
git tag -l -n9

# View specific tag
git show v1.3.0
```

### **Compare Releases**

```bash
# See changes between releases
git log v1.2.3..v1.3.0 --oneline

# See detailed diff
git diff v1.2.3..v1.3.0
```

---

## 🎓 Best Practices

### **1. Start with 0.1.0**

- First release should be `v0.1.0` (not `v1.0.0`)
- `v1.0.0` should be reserved for "first stable release"

### **2. Be Conservative**

- When in doubt, use a smaller increment
- Better to release `v1.2.4` than `v1.3.0` if unsure
- You can always release another PATCH if needed

### **3. Document Breaking Changes**

- Always document what broke in MAJOR releases
- Consider migration guides for breaking changes
- Update changelog/README with migration steps

### **4. Regular Releases**

- Don't wait too long between releases
- Small, frequent releases are better than large, infrequent ones
- Users appreciate regular updates

### **5. Meaningful Tag Messages**

```bash
# ✅ Good
git tag -a v1.3.0 -m "Release v1.3.0: Added backup command and improved font matching"

# ❌ Bad
git tag -a v1.3.0 -m "Release"
```

### **6. Test Before Releasing**

- Always test locally before creating a tag
- Run all tests: `go test ./...`
- Build and test binaries: `go build -o fontget.exe .`

---

## 🚨 Emergency Releases

For **critical security fixes** or **severe bugs**:

1. **Fix the issue** immediately
2. **Test thoroughly** (but quickly)
3. **Create PATCH release** (e.g., `v1.2.3` → `v1.2.4`)
4. **Push tag immediately** to trigger CI/CD
5. **Notify users** if necessary (via GitHub Release notes)

**Example:**
```bash
# Emergency security fix
git tag -a v1.2.4 -m "Release v1.2.4: Critical security fix - CVE-2024-XXXX"
git push origin v1.2.4
```

---

## ❓ FAQ

### **Q: Can I change a version after releasing?**

**A**: No, tags are immutable. If you need to fix a release, create a new PATCH version (e.g., `v1.3.0` → `v1.3.1`).

### **Q: What if I make a mistake in the version number?**

**A**: Create a new tag with the correct version. Don't delete the old tag (it may be referenced). Consider documenting the mistake in release notes.

### **Q: How often should I release?**

**A**: There's no fixed schedule. Release when you have meaningful changes (bug fixes, features, security fixes).

### **Q: Should I version every commit?**

**A**: No, only version releases. Daily commits use `"dev"` version automatically.

### **Q: What about pre-releases (alpha, beta, rc)?**

**A**: FontGet typically releases directly to stable. Pre-releases are supported by SemVer but not commonly used. See [Versioning Guide](./versioning-guide.md#pre-releases-alpha-beta-rc) for details.

### **Q: How do I know if CI/CD succeeded?**

**A**: Check the GitHub Actions tab. A successful build will create a GitHub Release with binaries.

### **Q: What if CI/CD fails?**

**A**: 
1. Check the Actions log for errors
2. Fix the issue
3. Delete the tag (if needed): `git push origin --delete v1.3.0`
4. Create a new tag with fixes: `git tag -a v1.3.1 -m "Release v1.3.1: Fixed build issue"`
5. Push the new tag

---

## 📚 Additional Resources

- [Versioning Guide](./versioning-guide.md) - Detailed versioning explanation
- [Semantic Versioning Specification](https://semver.org/) - Official SemVer spec
- [GoReleaser Documentation](https://goreleaser.com/) - CI/CD tool documentation
- [Git Tagging Best Practices](https://git-scm.com/book/en/v2/Git-Basics-Tagging) - Git tag reference

---

## 📋 Quick Reference

### **Release Command Template**

```bash
# 1. Ensure on main
git checkout main && git pull origin main

# 2. Check current version
git describe --tags --abbrev=0

# 3. Review changes
git log $(git describe --tags --abbrev=0)..HEAD --oneline

# 4. Create and push tag
git tag -a v<MAJOR>.<MINOR>.<PATCH> -m "Release v<MAJOR>.<MINOR>.<PATCH>: <Description>"
git push origin v<MAJOR>.<MINOR>.<PATCH>
```

### **Version Decision Matrix**

| Change Type | Increment | Example |
|------------|-----------|---------|
| Breaking change | MAJOR | `1.2.3` → `2.0.0` |
| New feature | MINOR | `1.2.3` → `1.3.0` |
| Bug fix | PATCH | `1.2.3` → `1.2.4` |
| Security fix | PATCH | `1.2.3` → `1.2.4` |

---

**Last Updated**: 2025-01-XX  
**Status**: Active Guide

