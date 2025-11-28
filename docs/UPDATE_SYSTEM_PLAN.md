# FontGet Self-Update System - Implementation Plan

## 📋 Overview

This document outlines the implementation plan for FontGet's self-update system, following CLI tool best practices and ensuring cross-platform compatibility (Windows, macOS, Linux).

**Related Plans:**
- **CI/CD Pipeline**: See `docs/CI_CD_PLAN.md` for release automation and GoReleaser configuration

---

## 🎯 Goals

1. **Manual Update Command**: `fontget update` to check and install updates
2. **Optional Auto-Check**: Configurable check for updates on startup
3. **User Control**: Full control over update behavior via configuration
4. **Cross-Platform**: Works seamlessly on Windows, macOS, and Linux
5. **Security**: Verify checksums, use HTTPS, safe binary replacement
6. **User Experience**: Clear progress, helpful errors, rollback capability

---

## 🏗️ Architecture

### **Library Choice**

**Using**: `github.com/rhysd/go-github-selfupdate` ⭐

**Why**: Built-in GitHub Releases API integration, version comparison, platform detection, checksum verification, and cross-platform binary replacement. Perfect fit for our use case.

**Reference**: See `docs/maintenance/UPDATE_LIBRARY_ANALYSIS.md` for full comparison.

### **Package Structure**

```
internal/update/
  ├── update.go          # Main update logic (wrapper around library)
  ├── config.go          # Update configuration management
  └── check.go           # Startup check logic (optional)
```

**Note**: Much simpler structure since the library handles:
- ✅ GitHub Releases API integration
- ✅ Version comparison
- ✅ Platform detection
- ✅ Binary download and verification
- ✅ Cross-platform binary replacement
- ✅ Rollback mechanism

### **Dependencies**

- **Self-Update Library**: `github.com/rhysd/go-github-selfupdate/selfupdate`
- **HTTP Client**: Standard `net/http` (already in use, library uses it internally)
- **File Operations**: Standard `os`, `path/filepath` (already in use, library uses it internally)

---

## 🔧 Implementation Details

### **1. Library Integration**

The `rhysd/go-github-selfupdate` library handles most of the complexity:

#### **What the Library Provides**
- ✅ **GitHub Releases API integration** - Automatically fetches latest release
- ✅ **Version comparison** - Compares current vs. latest version
- ✅ **Platform detection** - Automatically detects OS/arch and finds correct binary
- ✅ **Binary naming** - Handles naming conventions automatically
- ✅ **Checksum verification** - Verifies SHA256 checksums from release assets
- ✅ **Cross-platform replacement** - Safe binary replacement on all platforms
- ✅ **Rollback** - Automatic rollback on failure

#### **What We Need to Implement**
- ⚠️ **Configuration management** - Update settings in config.yaml
- ⚠️ **Startup check logic** - Optional auto-check on startup
- ⚠️ **Command interface** - `fontget update` command with flags
- ⚠️ **User experience** - Progress indicators, error messages, confirmations

### **2. Basic Library Usage**

#### **Simple Update Flow**
```go
import (
    "github.com/rhysd/go-github-selfupdate/selfupdate"
    "fontget/internal/version"
)

func CheckForUpdates() (*selfupdate.Release, bool, error) {
    updater, err := selfupdate.NewUpdater(selfupdate.Config{
        Owner: "Graphixa",
        Repo:  "FontGet",
    })
    if err != nil {
        return nil, false, err
    }
    
    currentVersion := version.GetVersion()
    latest, found, err := updater.DetectLatest("fontget")
    if err != nil {
        return nil, false, err
    }
    
    if !found {
        return nil, false, nil
    }
    
    // Library handles version comparison, but we can also do semantic comparison
    needsUpdate := latest.Version != currentVersion
    
    return latest, needsUpdate, nil
}

func UpdateToLatest() error {
    updater, err := selfupdate.NewUpdater(selfupdate.Config{
        Owner: "Graphixa",
        Repo:  "FontGet",
    })
    if err != nil {
        return err
    }
    
    latest, found, err := updater.DetectLatest("fontget")
    if err != nil {
        return err
    }
    
    if !found {
        return fmt.Errorf("no releases found")
    }
    
    // Library handles: download, checksum verification, binary replacement, rollback
    err = updater.UpdateTo(latest, "fontget")
    return err
}
```

### **3. Binary Naming Convention**

The library automatically detects the correct binary based on:
- Binary name pattern: `{cmd}_{goos}_{goarch}{.ext}` (with **hyphens**, not underscores)
- For FontGet: `fontget-windows-amd64.exe`, `fontget-darwin-amd64`, `fontget-linux-amd64`
- Library handles all platform detection automatically

**⚠️ CRITICAL: GoReleaser Configuration Required**

The `goReleaser_Plan.md` shows binary naming with **underscores** (`Fontget_windows_amd64.exe`), but the self-update library requires **hyphens** (`fontget-windows-amd64.exe`).

**Action Required**: Update GoReleaser configuration to:
1. Use lowercase binary name: `binary: fontget` (not `Fontget`)
2. Configure binary naming in archives to use hyphens: `fontget-{os}-{arch}{.ext}`
3. Ensure binaries inside archives are named correctly for the library to find them

**See Integration Notes below for GoReleaser config adjustments.**

### **4. Checksum Verification**

The library automatically:
- ✅ Looks for `checksums.txt` in release assets
- ✅ Parses checksums for the current platform binary
- ✅ Verifies SHA256 checksum after download
- ✅ Fails if checksum doesn't match

**No custom implementation needed** - library handles it all.

### **5. Cross-Platform Binary Replacement**

The library handles:
- ✅ **Windows**: Safe atomic replacement with rollback
- ✅ **macOS/Linux**: Atomic rename with rollback
- ✅ **File locking**: Handles "file in use" errors
- ✅ **Permissions**: Handles permission errors gracefully
- ✅ **Backup**: Keeps backup for rollback

**Error Handling** (library provides, we just need to display):
- If binary is locked: Library returns error, we show: "FontGet is currently running. Please close other instances and try again."
- If permissions denied: Library returns error, we show: "Insufficient permissions. Try running as administrator/sudo."
- If update fails: Library automatically rolls back, we show success or error message

### **6. Configuration Integration**

#### **Config Structure**
Add to `internal/config/user_preferences.go`:

```go
type UpdateSection struct {
    AutoCheck    bool   `yaml:"AutoCheck"`     // Check on startup
    AutoUpdate   bool   `yaml:"AutoUpdate"`    // Auto-install (default: false)
    CheckInterval int   `yaml:"CheckInterval"` // Hours between checks
    LastChecked  string `yaml:"LastChecked"`   // ISO timestamp
    UpdateChannel string `yaml:"UpdateChannel"` // stable/beta/nightly
}
```

#### **Default Values**
```go
Update: UpdateSection{
    AutoCheck:    true,   // Check by default
    AutoUpdate:   false,  // Manual install by default
    CheckInterval: 24,    // Check daily
    LastChecked:  "",     // Never checked
    UpdateChannel: "stable",
}
```

#### **First-Run Prompt**
During first run (if config doesn't exist):
```
Would you like FontGet to check for updates automatically? [Y/n]
```

### **7. Startup Update Check**

#### **Implementation**
- Check in `root.go` `PersistentPreRunE` (after config loaded)
- Only if `Update.AutoCheck == true`
- Check `Update.LastChecked` and `Update.CheckInterval`
- Run in goroutine (non-blocking)
- Show notification if update available
- Don't delay startup

#### **Notification Format**
```
FontGet v1.2.3 is available (you have v1.2.2).
Run 'fontget update' to upgrade.
```

### **8. Update Command**

#### **Command Structure**
```go
var updateCmd = &cobra.Command{
    Use:   "update",
    Short: "Update FontGet to the latest version",
    Long: `Check for updates and optionally install the latest version.
    
By default, this command checks for updates and prompts for confirmation
before installing. Use flags to customize behavior.`,
    Example: `  fontget update
  fontget update --check
  fontget update -y
  fontget update --version 1.2.3`,
}
```

#### **Flags**
- `--check`: Only check, don't install
- `-y`: Skip confirmation prompt
- `--version <version>`: Update to specific version

#### **Flow**
1. Create updater with `selfupdate.NewUpdater()`
2. Call `updater.DetectLatest("fontget")` - library handles GitHub API call
3. Compare versions (library provides `latest.Version`, compare with `version.GetVersion()`)
4. If update available:
   - Show current vs. new version
   - Show release notes from `latest.ReleaseNotes` (first 10 lines)
   - Prompt for confirmation (unless `-y` flag)
   - Call `updater.UpdateTo(latest, "fontget")` - library handles:
     - Download binary
     - Verify checksum
     - Backup current binary
     - Replace binary
     - Rollback on failure
   - Show success message
5. If no update: "FontGet is up to date (v1.2.2)"

---

## 🔒 Security Considerations

### **1. Checksum Verification** ✅ **Handled by Library**
- ✅ Library **always verifies SHA256 checksums** before installation
- ✅ Library never installs binary without valid checksum
- ✅ Library automatically retries on checksum mismatch

### **2. HTTPS Only** ✅ **Handled by Library**
- ✅ Library uses HTTPS for all API calls and downloads
- ✅ Library verifies TLS certificates
- ✅ No HTTP fallback

### **3. Binary Validation** ✅ **Handled by Library**
- ✅ Library doesn't execute downloaded binary until verified
- ✅ Library verifies file size matches expected size
- ✅ Library checks file permissions after download

### **4. Future Enhancements**
- ⚠️ **Code signing verification** - Library doesn't support this, but we can add it later if needed
- ⚠️ **GPG signature verification** - Would need custom implementation
- ⚠️ **Certificate pinning** - Library uses standard Go TLS, can be enhanced if needed

**Note**: For most use cases, GitHub's security + checksum verification is sufficient. Code signing can be added later if needed.

---

## 🎨 User Experience

### **Progress Indicators**
- Use existing spinner for API check
- Use progress bar for download (if available)
- Show download speed and ETA
- Clear error messages

### **Error Messages**
- **Network error**: "Unable to check for updates. Check your internet connection."
- **API error**: "Unable to fetch update information. GitHub API may be unavailable."
- **Checksum error**: "Download verification failed. Retrying..."
- **Permission error**: "Insufficient permissions. Try running as administrator/sudo."
- **File locked**: "FontGet is currently running. Please close other instances."

### **Success Messages**
- "FontGet updated successfully from v1.2.2 to v1.2.3"
- "Update complete. Restart FontGet to use the new version."

### **Verbose/Debug Output**
- Show API response details in verbose mode
- Show download progress in verbose mode
- Show file operations in debug mode
- Show checksum verification in debug mode

---

## 🧪 Testing Strategy

### **Unit Tests**
- Version comparison logic
- Platform detection
- Checksum parsing and verification
- Config loading and defaults

### **Integration Tests**
- Mock GitHub API responses
- Test download and verification flow
- Test binary replacement (use temp directories)
- Test rollback mechanism

### **Manual Testing**
- Test on Windows, macOS, Linux
- Test with different architectures (amd64, arm64)
- Test error scenarios (no internet, API down, etc.)
- Test with locked binary
- Test with insufficient permissions
- Test rollback after failed update

---

## 📝 Implementation Checklist

### **Phase 1: Library Integration & Configuration**
- [ ] Add `github.com/rhysd/go-github-selfupdate/selfupdate` to `go.mod`
- [ ] Create `internal/update/` package
- [ ] Implement basic `CheckForUpdates()` function using library
- [ ] Implement basic `UpdateToLatest()` function using library
- [ ] Add update config section to `internal/config/user_preferences.go`
- [ ] Test basic update flow (check for updates, update to latest)

### **Phase 2: Update Command**
- [ ] Create `cmd/update.go`
- [ ] Implement `fontget update` command
- [ ] Implement `--check` flag (only check, don't install)
- [ ] Implement `-y` flag (skip confirmation)
- [ ] Implement `--version / -v <version>` flag (update to specific version)
- [ ] Add confirmation prompts
- [ ] Show release notes from `latest.ReleaseNotes`
- [ ] Integrate with verbose/debug output
- [ ] Test all flags and scenarios

### **Phase 3: Configuration & Startup Check**
- [ ] Implement startup check logic in `root.go`
- [ ] Add notification display when update available
- [ ] Respect `Update.AutoCheck` config setting
- [ ] Respect `Update.CheckInterval` config setting
- [ ] Make startup check non-blocking (goroutine)
- [ ] Update `Update.LastChecked` timestamp
- [ ] Test startup check behavior

### **Phase 4: Error Handling & UX**
- [ ] Add comprehensive error messages for library errors
- [ ] Handle "file locked" errors gracefully
- [ ] Handle "permissions denied" errors gracefully
- [ ] Handle network errors gracefully
- [ ] Show helpful error messages with suggestions
- [ ] Integrate with existing UI styles
- [ ] Test error scenarios

### **Phase 5: Testing & Polish**
- [ ] Test on Windows, macOS, Linux
- [ ] Test with different architectures (amd64, arm64)
- [ ] Test error scenarios (no internet, API down, etc.)
- [ ] Test with locked binary
- [ ] Test with insufficient permissions
- [ ] Test rollback mechanism (library handles, but verify)
- [ ] Update documentation
- [ ] Add to help text
- [ ] Verify binary naming matches library expectations

---

## 🔗 References

### **Library Documentation**
- [rhysd/go-github-selfupdate](https://pkg.go.dev/github.com/rhysd/go-github-selfupdate/selfupdate) ⭐ **Primary Library**
- [Library Repository](https://github.com/rhysd/go-github-selfupdate)

### **Related Plans**
- [CI/CD Plan](docs/CI_CD_PLAN.md) - Release automation, build pipeline, and GoReleaser configuration

### **Best Practices**
- [CLI Best Practices - Self-Updates](https://clig.dev/#self-updates)
- [Semantic Versioning](https://semver.org/)
- [GitHub Releases API](https://docs.github.com/en/rest/releases/releases)

### **Similar Implementations**
- `gh` CLI (GitHub CLI) - uses similar approach
- `golangci-lint` - self-update command
- `kubectl` - plugin update mechanism

---

## ❓ Open Questions

1. **Update Channel**: Should we support dev channels initially, or just stable?
   - **Recommendation**: Start with stable only, add channels later

2. **Auto-Update**: Should auto-update be opt-in or opt-out?
   - **Recommendation**: Opt-in (manual by default) for security

3. **Rollback**: How many versions to keep for rollback?
   - **Recommendation**: Keep last 2-3 versions

4. **Notification Frequency**: How often to show update notifications?
   - **Recommendation**: Once per day max, respect CheckInterval

5. **Pre-release Versions**: Should we check for pre-releases?
   - **Recommendation**: Only if UpdateChannel is "dev"

---

---

## 📦 **Library Integration Notes**

### **Binary Naming Requirements**

The library expects binaries to be named: `{cmd}_{goos}_{goarch}{.ext}` (with **hyphens**, not underscores)

**For FontGet:**
- Windows: `fontget-windows-amd64.exe`, `fontget-windows-arm64.exe`
- macOS: `fontget-darwin-amd64`, `fontget-darwin-arm64`
- Linux: `fontget-linux-amd64`, `fontget-linux-arm64`

**⚠️ GoReleaser Configuration Alignment**

The CI/CD plan (`docs/CI_CD_PLAN.md`) includes GoReleaser configuration that must match these requirements:

1. **Binary Name**: Use lowercase `fontget` (not `Fontget`)
2. **Main Path**: Current project has `main.go` at root (not `./cmd/Fontget/main.go`)
3. **Binary Naming**: Use hyphens in binary names, not underscores
4. **Archive Contents**: Binaries inside archives must be named `fontget-{os}-{arch}{.ext}`

**Required GoReleaser Config Changes:**
```yaml
builds:
  - id: fontget
    main: .                    # Root main.go (not ./cmd/Fontget)
    binary: fontget            # Lowercase (not Fontget)
    # ... rest of config

archives:
  - id: default
    builds:
      - fontget
    # Ensure binaries inside archives are named correctly
    # GoReleaser will extract binaries, library needs them named: fontget-{os}-{arch}{.ext}
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    # May need to configure file renaming inside archives
```

**Action Required**: 
- Ensure `.goreleaser.yaml` reflects actual project structure (see `docs/CI_CD_PLAN.md`)
- Configure GoReleaser to produce binaries with hyphenated names
- Test that library can find binaries in release assets

### **Checksums File**

The library automatically looks for `checksums.txt` in release assets with format:
```
SHA256(fontget-windows-amd64.exe)= abc123...
SHA256(fontget-darwin-amd64)= def456...
SHA256(fontget-linux-amd64)= ghi789...
```

**GoReleaser Integration:**
- GoReleaser can generate `checksums.txt` automatically
- Ensure checksum file uses **hyphenated** binary names (not underscores)
- Format must match exactly: `SHA256(binary-name)= hash`

**Action Required**: 
- Configure GoReleaser `checksum` section to generate correct format
- Verify checksum file matches library expectations

### **Version Tag Format**

The library works with semantic versioning tags:
- ✅ `v1.2.3` (recommended)
- ✅ `1.2.3` (also works)
- ⚠️ Pre-releases: `v1.2.3-beta.1` (library handles, but may need filtering)

**GoReleaser Integration:**
- GoReleaser is triggered by git tags matching `v*`
- This aligns perfectly with library expectations
- No changes needed

**Action Required**: Ensure releases use semantic versioning tags (already planned in GoReleaser plan).

### **Project Structure Alignment**

**Current Project Structure:**
```
FontGet/
  main.go              # Entry point (not ./cmd/Fontget/main.go)
  cmd/                 # Command implementations
  internal/            # Internal packages
  go.mod               # Module: fontget (lowercase)
```

**GoReleaser Configuration Requirements:**
- ✅ Use `main: .` (root main.go, not `./cmd/Fontget/main.go`)
- ✅ Use `binary: fontget` (lowercase, not `Fontget`)
- ✅ Ensure binary naming uses hyphens for self-update compatibility

**Action Required**: Verify `.goreleaser.yaml` matches these requirements (see `docs/CI_CD_PLAN.md` for full configuration).

---

---

## ⚠️ **Integration with GoReleaser Plan**

### **Conflicts to Resolve**

1. **Project Structure Mismatch**
   - GoReleaser plan assumes: `./cmd/Fontget/main.go`
   - Actual project: `main.go` at root
   - **Fix**: Update GoReleaser config to use `main: .`

2. **Binary Naming Mismatch**
   - GoReleaser plan shows: `Fontget_windows_amd64.exe` (capital F, underscores)
   - Self-update library needs: `fontget-windows-amd64.exe` (lowercase, hyphens)
   - **Fix**: Configure GoReleaser to use lowercase and hyphens

3. **Archive Binary Names**
   - Binaries inside archives must be named correctly for library to find them
   - **Fix**: Configure GoReleaser archive contents to rename binaries

### **Recommended GoReleaser Config Updates**

```yaml
project_name: fontget  # Lowercase to match module name

builds:
  - id: fontget
    main: .                    # Root main.go (not ./cmd/Fontget/main.go)
    binary: fontget            # Lowercase (not Fontget)
    # ... rest of config

archives:
  - id: default
    builds:
      - fontget
    # Archive name can use underscores, but binaries inside must use hyphens
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    # May need to configure file renaming inside archives
    # Or ensure binary is named correctly when built

checksum:
  name_template: "checksums.txt"
  # Ensure checksums use hyphenated names matching library expectations
```

**Action Required**: Ensure `.goreleaser.yaml` aligns with these requirements (see `docs/CI_CD_PLAN.md` for configuration details).

---

**Last Updated**: 2024-01-15  
**Status**: Planning Phase - **Using `rhysd/go-github-selfupdate` library**  
**⚠️ Note**: GoReleaser configuration needs alignment (see Integration Notes above)

