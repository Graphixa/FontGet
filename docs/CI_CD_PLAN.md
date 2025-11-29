# FontGet CI/CD Pipeline Plan

## 📋 Overview

This document outlines the CI/CD strategy for FontGet, including developer workflows, release triggers, and automated build processes. The pipeline uses **GoReleaser** (OSS) with **GitHub Actions** to build, test, and release FontGet across multiple platforms and package managers.

**Related Plans:**
- **Self-Update System**: See `docs/UPDATE_SYSTEM_PLAN.md` for update functionality

---

## 🎯 CI/CD Strategy

### **Release Model: Manual Tag-Based Releases**

FontGet uses **manual, tag-based releases**. This means:
- **Regular commits**: Build and test only (no releases)
- **Tagged commits**: Trigger full release process (build, test, create GitHub release, upload artifacts)

### **Why Manual Releases?**
- Full control over when releases happen
- Time to review changes before release
- Ability to batch multiple commits into a single release
- Prevents accidental releases from every push
- Standard practice for CLI tools

---

## 🔄 Developer Workflow

### **Daily Development**

1. **Make changes** to the codebase
2. **Commit and push** to your branch:
   ```bash
   git add .
   git commit -m "feat: add new feature"
   git push origin your-branch
   ```
3. **CI runs automatically** on push:
   - ✅ Builds on all platforms (Windows, macOS, Linux)
   - ✅ Runs tests
   - ✅ Validates code quality
   - ❌ **Does NOT create a release**

### **Creating a Release**

When you're ready to release:

1. **Ensure everything is ready:**
   - All tests pass
   - Code is reviewed and merged to `main`
   - Version number is appropriate (see Version Management below)

2. **Create and push a git tag:**
   ```bash
   # Tag format: v1.2.3 (semantic versioning)
   git tag -a v1.2.3 -m "Release v1.2.3: Description of changes"
   git push origin v1.2.3
   ```

3. **GitHub Actions automatically:**
   - ✅ Detects the tag push
   - ✅ Builds binaries for all platforms
   - ✅ Runs full test suite
   - ✅ Creates a GitHub Release
   - ✅ Uploads all binaries as release assets
   - ✅ Generates checksums (SHA256)
   - ✅ Updates release notes (if configured)

4. **Verify the release:**
   - Check GitHub Releases page
   - Download and test binaries
   - Update documentation if needed

---

## 🏷️ Version Management

### **Semantic Versioning**

FontGet follows [Semantic Versioning](https://semver.org/): `MAJOR.MINOR.PATCH`

- **MAJOR** (1.0.0): Breaking changes
- **MINOR** (0.1.0): New features, backward compatible
- **PATCH** (0.0.1): Bug fixes, backward compatible

### **Version Sources**

The version is determined in this order:

1. **Git tag** (highest priority) - Used in releases
   - Example: Tag `v1.2.3` → Version `1.2.3`
   - Extracted by: `git describe --tags --always`

2. **Build info** (fallback) - Used in development
   - From `go.mod` version if available
   - Defaults to `dev` if no tag exists

### **Version Injection**

Version information is injected at build time via Go's `ldflags`:

```bash
-ldflags "-X fontget/internal/version.Version=1.2.3 \
         -X fontget/internal/version.GitCommit=abc1234 \
         -X fontget/internal/version.BuildDate=2024-01-15T10:30:00Z"
```

This is handled automatically by the Makefile and GitHub Actions.

---

## 🚀 Release Process

### **Step-by-Step Release Checklist**

#### **Pre-Release**

- [ ] All features for this release are complete
- [ ] All tests pass locally (`make test`)
- [ ] Code is merged to `main` branch
- [ ] CHANGELOG.md is updated (if maintained)
- [ ] Version number is determined (semantic versioning)

#### **Creating the Release**

1. **Check current version:**
   ```bash
   git describe --tags --abbrev=0  # Shows latest tag
   ```

2. **Determine next version:**
   - Bug fix → increment PATCH (1.2.3 → 1.2.4)
   - New feature → increment MINOR (1.2.3 → 1.3.0)
   - Breaking change → increment MAJOR (1.2.3 → 2.0.0)

3. **Create and push the tag:**
   ```bash
   # Create annotated tag with message
   git tag -a v1.2.4 -m "Release v1.2.4: Fix font matching bug"
   
   # Push tag to GitHub (this triggers the release)
   git push origin v1.2.4
   ```

4. **Monitor GitHub Actions:**
   - Go to: `https://github.com/YOUR_USERNAME/FontGet/actions`
   - Watch the "Release" workflow run
   - Wait for all builds to complete (usually 5-10 minutes)

#### **Post-Release**

- [ ] Verify release appears on GitHub Releases page
- [ ] Download and test binaries on each platform
- [ ] Verify checksums are correct
- [ ] Update any external documentation
- [ ] Announce release (if applicable)

---

## 🛠️ Build Tool: GoReleaser

FontGet uses **GoReleaser OSS** (free, open-source) as the build and release automation tool. GoReleaser handles:
- Cross-platform binary compilation
- Archive generation (`.zip`, `.tar.gz`)
- Package creation (`.deb`, `.rpm`, Chocolatey, Homebrew, Scoop, Winget)
- GitHub Releases automation
- Checksum generation

### **Project Structure**

**Actual Project Layout:**
```
FontGet/
  main.go            # Entry point (at root, not ./cmd/Fontget/main.go)
  cmd/               # Command implementations (Cobra commands)
  internal/...       # Internal packages
  go.mod             # module fontget (lowercase)
```

**⚠️ Important Configuration Notes:**
- Binary naming must use **hyphens** (not underscores): `fontget-windows-amd64.exe`
- Binary name must be **lowercase**: `fontget` (not `Fontget`)
- Required for self-update system compatibility (see `docs/UPDATE_SYSTEM_PLAN.md`)
- GoReleaser automatically creates `fontget-{os}-{arch}{.ext}` when `binary: fontget` is set

---

## 🔧 GitHub Actions Workflows

### **Workflow Structure**

The CI/CD pipeline consists of two main workflows:

#### **1. CI Workflow** (`ci.yml`)
- **Triggers:** Every push and pull request
- **Purpose:** Build and test on all platforms
- **Actions:**
  - Build for Windows (amd64, arm64)
  - Build for macOS (amd64, arm64)
  - Build for Linux (amd64, arm64)
  - Run test suite
  - Run linters (if configured)
  - **Does NOT create releases**

#### **2. Release Workflow** (`release.yml`)
- **Triggers:** Only when a tag matching `v*` is pushed (e.g., `v1.2.3`)
- **Purpose:** Create GitHub release with binaries and packages
- **Actions:**
  - Run GoReleaser in release mode
  - Build for all platforms and architectures
  - Run full test suite
  - Create GitHub Release
  - Upload binaries as release assets
  - Generate and upload checksums (SHA256)
  - Create packages (`.deb`, `.rpm`, Chocolatey, Homebrew, Scoop, Winget)
  - Optionally: Generate release notes

### **Workflow Files Location**

```
.github/
  workflows/
    ci.yml          # Continuous Integration (build + test)
    release.yml     # Release automation (tag-triggered, uses GoReleaser)
```

### **GitHub Actions Workflow Example**

Create `.github/workflows/release.yml`:

```yaml
name: Release

on:
  push:
    tags:
      - "v*"

jobs:
  release:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.24.x"  # Match your go.mod version

      - name: Install GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          CHOCOLATEY_API_KEY: ${{ secrets.CHOCOLATEY_API_KEY }}
          HOMEBREW_TAP_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}
          SCOOP_BUCKET_TOKEN: ${{ secrets.SCOOP_BUCKET_TOKEN }}
          WINGET_PUSH_TOKEN: ${{ secrets.WINGET_PUSH_TOKEN }}
```

### **Required GitHub Secrets**

Set these in GitHub repository settings (Settings → Secrets and variables → Actions):

- `GITHUB_TOKEN` - Automatically provided by GitHub
- `CHOCOLATEY_API_KEY` - For publishing to Chocolatey (optional)
- `HOMEBREW_TAP_TOKEN` - Personal Access Token with push access to `ORG/homebrew-tap` (optional)
- `SCOOP_BUCKET_TOKEN` - Personal Access Token with push access to `ORG/scoop-bucket` (optional)
- `WINGET_PUSH_TOKEN` - Personal Access Token for winget manifest repo (optional)

**Note:** Package manager tokens are only needed if you want to auto-publish to those platforms. GitHub Releases will work without them.

---

## 📦 Build Matrix & Distribution

### **Supported Platforms**

| Platform | Architectures | Binary Name |
|----------|--------------|-------------|
| Windows  | amd64, arm64 | `fontget-windows-amd64.exe`, `fontget-windows-arm64.exe` |
| macOS    | amd64, arm64 | `fontget-darwin-amd64`, `fontget-darwin-arm64` |
| Linux    | amd64, arm64 | `fontget-linux-amd64`, `fontget-linux-arm64` |

**⚠️ IMPORTANT**: Binary naming must match exactly for self-update system to work. The `rhysd/go-github-selfupdate` library expects this naming pattern: `{cmd}-{goos}-{goarch}{.ext}` (with hyphens, not underscores).

**Note**: GoReleaser automatically generates the correct binary names when `binary: fontget` is set in the config. The library uses `DetectLatest("Graphixa/FontGet")` with the repository slug format, and automatically finds the correct binary for the current platform.

### **Build Artifacts**

Each release includes:
- **Raw binaries**: 6 binary files (one per platform/arch combination)
- **Archives**: `.zip` (Windows) and `.tar.gz` (macOS/Linux) for each platform
- **Checksums**: `checksums.txt` file with SHA256 hashes (required for self-update system)
- **Packages** (via GoReleaser):
  - `.deb` packages for Debian/Ubuntu
  - `.rpm` packages for Fedora/RHEL/CentOS
  - Chocolatey `.nupkg` package (if configured)
  - Homebrew formula (if tap repo configured)
  - Scoop manifest (if bucket repo configured)
  - Winget manifest (if configured)
- **Source code archive** (automatically included by GitHub)

### **Package Distribution**

GoReleaser can automatically publish to multiple package managers:

1. **GitHub Releases** (always enabled)
   - Raw binaries
   - Archives
   - Checksums
   - Source code

2. **Linux Packages** (via NFPM)
   - `.deb` for Debian/Ubuntu
   - `.rpm` for Fedora/RHEL/CentOS
   - Uploaded to GitHub Releases

3. **Homebrew** (optional, requires tap repo)
   - Auto-generates formula
   - Pushes to `ORG/homebrew-tap` repository
   - Requires `HOMEBREW_TAP_TOKEN` secret

4. **Scoop** (optional, requires bucket repo)
   - Auto-generates manifest
   - Pushes to `ORG/scoop-bucket` repository
   - Requires `SCOOP_BUCKET_TOKEN` secret

5. **Chocolatey** (optional)
   - Auto-generates `.nupkg` package
   - Publishes to Chocolatey.org
   - Requires `CHOCOLATEY_API_KEY` secret

6. **Winget** (optional)
   - Auto-generates manifest
   - Can push to dedicated repo or fork of `microsoft/winget-pkgs`
   - Requires `WINGET_PUSH_TOKEN` secret

**Future Work (Manual):**
- Submitting to official Debian/Ubuntu repositories
- Submitting to official Fedora repositories

### **Checksums File Format**

The `checksums.txt` file must follow this format for the self-update library:
```
SHA256(fontget-windows-amd64.exe)= abc123...
SHA256(fontget-windows-arm64.exe)= def456...
SHA256(fontget-darwin-amd64)= ghi789...
SHA256(fontget-darwin-arm64)= jkl012...
SHA256(fontget-linux-amd64)= mno345...
SHA256(fontget-linux-arm64)= pqr678...
```

**Action Required**: Ensure GitHub Actions workflow generates `checksums.txt` with this exact format.

---

## 🧪 Testing Strategy

### **Automated Tests**

The CI pipeline runs:

1. **Unit Tests:**
   ```bash
   go test ./...
   ```

2. **Integration Tests** (if implemented):
   - Cross-command consistency
   - Font matching logic
   - Source priority handling

3. **Build Verification:**
   - Ensure binaries build successfully
   - Verify version information is injected correctly
   - Check binary size (detect bloat)

### **Manual Testing Before Release**

Before creating a release tag, manually test:

- [ ] Core commands work (`add`, `remove`, `search`, `list`, `info`)
- [ ] Update command works (`update`, `update --check`)
- [ ] Version command shows correct version
- [ ] Help text is accurate
- [ ] Verbose/debug flags work correctly
- [ ] Self-update system can detect releases (test with `update --check`)
- [ ] Cross-platform compatibility (if you have access)

---

## 📝 Release Notes

### **Automatic Generation** (Recommended)

GitHub Actions can auto-generate release notes from:
- Commit messages since last tag
- Pull requests merged since last tag
- Conventional commit format (if used)

### **Manual Release Notes** (Alternative)

You can write release notes manually when creating the tag:

```bash
git tag -a v1.2.4 -m "Release v1.2.4

## What's New
- Fixed font matching bug in search command
- Improved error messages for invalid font IDs
- Updated help text for better clarity

## Bug Fixes
- Fixed crash when sources file is missing
- Fixed incorrect font count in list command

## Improvements
- Performance optimization for font suggestion system
"
```

---

## 🔐 Security & Verification

### **Checksums**

Every release includes a `checksums.txt` file:

```
SHA256(fontget-windows-amd64.exe)= abc123...
SHA256(fontget-darwin-amd64)= def456...
SHA256(fontget-linux-amd64)= ghi789...
...
```

Users can verify downloads:
```bash
# On macOS/Linux
sha256sum -c checksums.txt

# On Windows (PowerShell)
Get-FileHash fontget-windows-amd64.exe -Algorithm SHA256
```

### **Code Signing** (Future Enhancement)

For production releases, consider:
- Windows: Code signing certificate
- macOS: Notarization
- Linux: GPG signing

---

## 🚨 Troubleshooting

### **Release Workflow Failed**

**Problem:** Release workflow fails after pushing tag

**Solutions:**
1. Check GitHub Actions logs for specific error
2. Verify tag format: Must start with `v` (e.g., `v1.2.3`)
3. Ensure you have write permissions to repository
4. Check if release already exists for that version

### **Version Shows as "dev"**

**Problem:** Binary shows version as "dev" instead of tag version

**Solutions:**
1. Ensure tag format is correct: `v1.2.3` (not `1.2.3`)
2. Verify `ldflags` are being passed correctly in workflow
3. Check that `git describe` works in CI environment

### **Build Fails on Specific Platform**

**Problem:** One platform fails to build

**Solutions:**
1. Check platform-specific code (e.g., `platform_windows.go`)
2. Verify Go version compatibility
3. Check for platform-specific dependencies
4. Review build logs for compilation errors

### **Self-Update System Issues**

**Problem:** Self-update system can't find binaries or checksums

**Solutions:**
1. Verify binary naming: Must be `fontget-{os}-{arch}{.ext}` (lowercase, hyphens)
2. Check checksums file format: Must be `SHA256(binary-name)= hash`
3. Ensure raw binaries are uploaded to GitHub Releases (not just archives)
4. Verify repository slug in code matches GitHub repository: `Graphixa/FontGet`
5. Test with `goreleaser release --snapshot` to verify binary naming

---

## 📚 Quick Reference

### **Common Commands**

```bash
# Check current version
git describe --tags --abbrev=0

# Create a release tag
git tag -a v1.2.4 -m "Release v1.2.4: Description"
git push origin v1.2.4

# List all tags
git tag -l

# Delete a tag (if needed, before release is created)
git tag -d v1.2.4
git push origin :refs/tags/v1.2.4

# Test build locally
make build

# Run tests
make test

# Check version info that would be injected
make version-info
```

### **Release Tag Examples**

```bash
# Patch release (bug fix)
git tag -a v1.2.4 -m "Release v1.2.4: Fix font matching bug"

# Minor release (new feature)
git tag -a v1.3.0 -m "Release v1.3.0: Add export command"

# Major release (breaking change)
git tag -a v2.0.0 -m "Release v2.0.0: Redesign configuration system"
```

---

## ⚙️ GoReleaser Configuration

### **Configuration File**

Create `.goreleaser.yaml` at the repository root. This file controls all build and release behavior.

### **GoReleaser Configuration Skeleton**

```yaml
project_name: fontget  # Lowercase to match module name and self-update requirements
dist: dist

before:
  hooks:
    - go mod tidy

builds:
  - id: fontget
    main: .                    # Root main.go (not ./cmd/Fontget/main.go)
    binary: fontget            # Lowercase (not Fontget) - required for self-update system
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - windows
      - darwin
    goarch:
      - amd64
      - arm64
    ignore:
      - goos: windows
        goarch: arm64  # if not supported initially

archives:
  - id: default
    builds:
      - fontget
    format_overrides:
      - goos: windows
        format: zip
    # Archive filename can use underscores, but binary inside will be named correctly
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    # ⚠️ IMPORTANT: GoReleaser automatically names binaries as fontget-{os}-{arch}{.ext}
    # when binary: fontget is set. The self-update library will find these automatically.
    # Raw binaries are also uploaded to GitHub Releases for direct download.

checksum:
  name_template: "checksums.txt"

snapshot:
  name_template: "{{ .Tag }}-SNAPSHOT"

changelog:
  use: git

# Linux packages via NFPM (deb/rpm)
nfpms:
  - id: linux-packages
    package_name: fontget
    file_name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Arch }}"
    formats:
      - deb
      - rpm
    maintainer: "FontGet Maintainers <maintainers@example.com>"
    description: "FontGet CLI tool for managing fonts"
    license: "MIT"
    vendor: "FontGet"
    homepage: "https://github.com/Graphixa/FontGet"
    section: "utils"
    bindir: /usr/bin
    contents:
      - src: ./dist/fontget_{{ .Os }}_{{ .Arch }}/fontget
        dst: /usr/bin/fontget

# Homebrew tap (optional)
brews:
  - name: fontget
    tap:
      owner: Graphixa
      name: homebrew-tap
    commit_author:
      name: goreleaserbot
      email: bot@example.com
    folder: Formula
    homepage: "https://github.com/Graphixa/FontGet"
    description: "FontGet CLI tool for managing fonts"
    license: "MIT"
    install: |
      bin.install "fontget"

# Scoop manifest (optional)
scoop:
  bucket:
    owner: Graphixa
    name: scoop-bucket
  homepage: "https://github.com/Graphixa/FontGet"
  description: "FontGet CLI tool for managing fonts"
  license: "MIT"
  commit_author:
    name: goreleaserbot
    email: bot@example.com
  url_template: "https://github.com/Graphixa/FontGet/releases/download/{{ .Tag }}/{{ .ArtifactName }}"

# Chocolatey package (optional)
chocolatey:
  - name: fontget
    title: "FontGet CLI"
    authors: "FontGet Team"
    project_url: "https://github.com/Graphixa/FontGet"
    tags: "cli tools fonts"
    summary: "FontGet CLI tool for managing fonts"
    description: "FontGet is a command-line font manager for installing and managing fonts."
    license_url: "https://github.com/Graphixa/FontGet/blob/main/LICENSE"
    docs_url: "https://github.com/Graphixa/FontGet"
    bug_tracker_url: "https://github.com/Graphixa/FontGet/issues"
    release_notes_url: "https://github.com/Graphixa/FontGet/releases/tag/{{ .Tag }}"
    files:
      - id: default
    api_key: "{{ .Env.CHOCOLATEY_API_KEY }}"
    source_repo: "https://push.chocolatey.org/"

# Winget manifest (optional)
winget:
  # Option A: generate into current repo under /dist/winget
  # Option B: configure to push to a separate repo (fork of winget-pkgs)
  - publisher: "Graphixa"
    publisher_url: "https://github.com/Graphixa"
    author: "Graphixa"
    short_description: "FontGet CLI tool for managing fonts"
    package_identifier: "Graphixa.FontGet"
    homepage: "https://github.com/Graphixa/FontGet"
    license: "MIT"
    tags:
      - cli
      - tools
      - fonts
    # url and SHA will be auto filled from artifacts
```

### **Configuration Notes**

**⚠️ CRITICAL for Self-Update System:**
- Set `binary: fontget` (lowercase, not `Fontget`) - GoReleaser will automatically create `fontget-{os}-{arch}{.ext}` format
- Set `main: .` (root main.go, not `./cmd/Fontget/main.go`)
- The self-update library uses repository slug format (`Graphixa/FontGet`) and automatically detects the correct binary
- Verify checksums file format matches self-update library expectations (format: `SHA256(binary-name)= hash`)

**Before Using:**
- Update all `ORG`/`Graphixa` placeholders with actual values
- Update email addresses and URLs
- Ensure NFPM `contents.src` path matches actual built binary path (may need adjustment based on GoReleaser output structure)
- Configure package manager sections only if you plan to use them
- Test configuration with `goreleaser release --snapshot` first to verify binary naming and paths

---

## 🎯 Next Steps

### **To Set Up CI/CD Pipeline:**

1. **Create `.goreleaser.yaml` configuration:**
   - Copy the skeleton above
   - Update with your actual repository details
   - Test locally: `goreleaser release --snapshot`

2. **Create `.github/workflows/` directory:**
   ```bash
   mkdir -p .github/workflows
   ```

3. **Create workflow files:**
   - `ci.yml` - For continuous integration
   - `release.yml` - For automated releases (uses GoReleaser)

4. **Test the workflows:**
   - Push a commit to trigger CI workflow
   - Create a test tag to verify release workflow

5. **Configure GitHub repository:**
   - Ensure Actions are enabled
   - Set up required secrets (if using package managers)
   - Test with a pre-release tag first

---

## 📖 Additional Resources

- [GoReleaser Documentation](https://goreleaser.com/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Semantic Versioning](https://semver.org/)
- [Go Build Constraints](https://pkg.go.dev/cmd/go#hdr-Build_constraints)
- [Git Tagging Best Practices](https://git-scm.com/book/en/v2/Git-Basics-Tagging)
- [Self-Update System Plan](docs/UPDATE_SYSTEM_PLAN.md) - Binary naming requirements

---

## ❓ FAQ

**Q: Can I create a release from a branch other than `main`?**  
A: Yes, but it's recommended to only release from `main` to ensure stability.

**Q: What happens if I push a tag that already exists?**  
A: GitHub will reject it. You'll need to delete the old tag first (if no release was created) or use a new version number.

**Q: Can I automate release notes generation?**  
A: Yes, GitHub Actions can generate release notes automatically from commits and PRs. See workflow configuration.

**Q: How do I do a pre-release or beta release?**  
A: Use pre-release version tags like `v1.2.4-beta.1` or `v1.2.4-rc.1`. You may need to adjust the workflow to handle these.

**Q: What if I need to fix a release?**  
A: Create a new patch release (e.g., `v1.2.4` → `v1.2.5`) with the fix. Avoid modifying existing releases.

---

**Last Updated:** 2025-11-29  
**Maintained By:** Graphixa

