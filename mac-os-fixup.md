# macOS Compatibility Analysis and Fix Plan

## Executive Summary

FontGet has several macOS-specific issues that prevent it from working correctly on modern macOS systems (macOS 14 Sonoma and later). The primary issue is the use of deprecated `atsutil` commands that were removed in macOS 14. Additionally, there are potential architecture detection issues for ARM vs Intel Macs in the Homebrew tap.

**Status**: 🔴 **CRITICAL** - Font installation, removal, and list commands fail on macOS 14+

---

## Issues Identified

### 1. ⚠️ **CRITICAL: `atsutil` Deprecation (macOS 14+)**

**Problem:**
- The `updateFontCache()` function in `internal/platform/darwin.go` uses `atsutil` commands
- `atsutil` was deprecated in Xcode 4.6 (2012) and **completely removed in macOS 14 Sonoma**
- This causes font installation and removal to fail with errors like:
  ```
  atsutil command failed: exec: "atsutil": executable file not found in $PATH
  ```

**Current Code Location:**
- `internal/platform/darwin.go:148-180` - `updateFontCache()` function
- Uses commands:
  - `atsutil databases -removeUser` (user scope)
  - `atsutil databases -remove` (machine scope)
  - `atsutil server -shutdown`
  - `atsutil server -ping`

**Impact:**
- ❌ `fontget add` fails after copying font files (cache update fails)
- ❌ `fontget remove` fails after deleting font files (cache update fails)
- ⚠️ Fonts may be copied/deleted but not recognized by macOS until manual intervention

**Solution:**
Replace `atsutil` commands with modern macOS font cache refresh method:
```go
// Modern approach: restart fontd service
exec.Command("pkill", "fontd")
// fontd will automatically restart and refresh cache
```

**Alternative (if pkill doesn't work):**
- Use `killall fontd` (more reliable on some systems)
- Or simply skip cache refresh (macOS auto-detects fonts in `~/Library/Fonts` and `/Library/Fonts`)

---

### 2. ⚠️ **System Integrity Protection (SIP) Considerations**

**Problem:**
- System Integrity Protection (SIP) introduced in macOS 10.11 El Capitan
- Prevents modification of `/System/Library/Fonts` (system fonts)
- However, `/Library/Fonts` (system-wide user fonts) should still be accessible with sudo

**Current Code:**
- Uses `/Library/Fonts` for machine scope (correct)
- Uses `~/Library/Fonts` for user scope (correct)
- Does NOT attempt to modify `/System/Library/Fonts` (correct)

**Impact:**
- ✅ Current implementation should work with SIP
- ⚠️ May need to verify permissions on `/Library/Fonts` directory

**Solution:**
- Verify that `/Library/Fonts` is accessible with sudo
- Add better error messages if SIP prevents access
- Consider checking SIP status and warning users

---

### 3. ⚠️ **List Command Issues**

**Problem:**
- List command uses `platform.ListInstalledFonts()` which walks font directories
- May fail if:
  - Directory permissions are incorrect
  - Font directories don't exist
  - Font metadata extraction fails

**Current Code:**
- `cmd/list.go:316-320` - Uses `ListInstalledFonts()`
- `internal/platform/platform.go:99-128` - `ListInstalledFonts()` implementation

**Potential Issues:**
1. **Directory Access**: May not have read permissions on `/Library/Fonts` without sudo
2. **Metadata Extraction**: Font metadata extraction may fail on corrupted fonts
3. **Error Handling**: Errors may be silently ignored

**Solution:**
- Add better error handling and logging
- Check directory permissions before attempting to read
- Handle metadata extraction failures gracefully
- Add verbose/debug output for troubleshooting

---

### 4. ⚠️ **ARM vs Intel Binary Selection**

#### 4a. Install Script (`scripts/install.sh`)

**Current Implementation:**
```bash
# Lines 42-54
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
esac

BINARY_NAME="fontget-${OS}-${ARCH}"
```

**Status:** ✅ **CORRECT**
- Properly detects `x86_64` → `amd64` (Intel)
- Properly detects `arm64`/`aarch64` → `arm64` (Apple Silicon)
- Downloads correct binary: `fontget-darwin-amd64` or `fontget-darwin-arm64`

**No Changes Needed**

#### 4b. GoReleaser Configuration (`.goreleaser.yaml`)

**Current Implementation:**
```yaml
goos:
  - darwin
goarch:
  - amd64
  - arm64
```

**Status:** ✅ **CORRECT**
- Builds both `darwin_amd64` and `darwin_arm64` binaries
- Extra files section correctly names binaries:
  - `fontget-darwin-amd64`
  - `fontget-darwin-arm64`

**No Changes Needed**

#### 4c. Homebrew Cask (`.goreleaser.yaml`)

**Current Implementation:**
```yaml
homebrew_casks:
  - name: fontget
    # ... uses default archive (darwin builds only)
    ids:
      - default
```

**Problem:** ⚠️ **POTENTIAL ISSUE**
- Homebrew Cask may not automatically select the correct architecture-specific binary
- GoReleaser generates a single archive that may contain only one architecture
- Need to verify if Homebrew can handle universal binaries or architecture-specific casks

**Solution:**
- Check if GoReleaser creates universal binaries for macOS (fat binaries)
- If not, may need to use `on_arm` and `on_intel` blocks in Homebrew Cask formula
- Or create separate casks for ARM and Intel (not recommended)

**Research Needed:**
- Verify what GoReleaser actually generates for macOS archives
- Check if Homebrew tap repository has architecture-specific handling
- Test installation on both ARM and Intel Macs

---

## Recommended Fixes

### Priority 1: Fix `atsutil` Deprecation (CRITICAL)

**File:** `internal/platform/darwin.go`

**Replace `updateFontCache()` function:**

```go
// updateFontCache refreshes the font cache on macOS
// Uses modern method compatible with macOS 14+ (Sonoma)
func (m *darwinFontManager) updateFontCache(scope InstallationScope) error {
	// On macOS 14+, atsutil was removed
	// Modern approach: restart fontd service to refresh cache
	// fontd automatically restarts and picks up new fonts
	
	// Try pkill first (more reliable)
	cmd := exec.Command("pkill", "fontd")
	if output, err := cmd.CombinedOutput(); err != nil {
		// If pkill fails, try killall as fallback
		cmd = exec.Command("killall", "fontd")
		if output, err := cmd.CombinedOutput(); err != nil {
			// If both fail, log warning but don't fail installation
			// macOS will auto-detect fonts in ~/Library/Fonts and /Library/Fonts
			// Fonts will be available after app restart or system refresh
			return fmt.Errorf("failed to refresh font cache (non-critical): %v\nOutput: %s", err, string(output))
		}
	}
	
	// Small delay to allow fontd to restart
	time.Sleep(500 * time.Millisecond)
	
	return nil
}
```

**Alternative (Simpler):**
```go
// updateFontCache refreshes the font cache on macOS
// On macOS 14+, we skip cache refresh as fonts are auto-detected
func (m *darwinFontManager) updateFontCache(scope InstallationScope) error {
	// macOS 14+ automatically detects fonts in ~/Library/Fonts and /Library/Fonts
	// No manual cache refresh needed - fonts will be available after app restart
	// For immediate availability, we can try to restart fontd (non-critical)
	
	// Try to restart fontd (non-blocking)
	go func() {
		exec.Command("pkill", "fontd").Run()
	}()
	
	return nil
}
```

**Testing:**
- Test on macOS 14+ (Sonoma)
- Test on macOS 13 (Ventura) - should still work
- Verify fonts appear in Font Book after installation
- Verify fonts are available in applications

---

### Priority 2: Improve Error Handling for List Command

**File:** `cmd/list.go` and `internal/platform/platform.go`

**Add better error handling:**

```go
// In collectFonts() function
fontDir := fm.GetFontDir(scope)

// Check if directory exists and is accessible
if _, err := os.Stat(fontDir); os.IsNotExist(err) {
    output.GetVerbose().Warning("Font directory does not exist: %s", fontDir)
    output.GetDebug().State("Directory %s does not exist, skipping", fontDir)
    return []ParsedFont{}, nil // Return empty, not error
}

// Check read permissions
if _, err := os.Open(fontDir); err != nil {
    if os.IsPermission(err) {
        output.GetVerbose().Warning("No read permission for font directory: %s", fontDir)
        output.GetDebug().Error("Permission denied accessing %s", fontDir)
        // For machine scope, suggest using sudo
        if scope == platform.MachineScope {
            return nil, fmt.Errorf("insufficient permissions to read %s. Try running with sudo", fontDir)
        }
    }
    return nil, fmt.Errorf("unable to access font directory %s: %w", fontDir, err)
}
```

---

### Priority 3: Verify Homebrew Cask Architecture Handling

**Action Items:**
1. Check what GoReleaser actually generates for macOS releases
2. Verify if the archive contains universal binaries or architecture-specific binaries
3. Test Homebrew installation on both ARM and Intel Macs
4. If needed, update Homebrew Cask formula to handle architecture-specific binaries

**Potential Solution (if needed):**

If GoReleaser creates separate binaries, may need to update Homebrew tap manually:

```ruby
# In homebrew-tap repository
cask "fontget" do
  version "1.1.1"
  
  on_arm do
    url "https://github.com/Graphixa/FontGet/releases/download/v#{version}/fontget-darwin-arm64"
    sha256 "..." # ARM binary checksum
  end
  
  on_intel do
    url "https://github.com/Graphixa/FontGet/releases/download/v#{version}/fontget-darwin-amd64"
    sha256 "..." # Intel binary checksum
  end
  
  # ... rest of cask definition
end
```

**Note:** This would require manual updates to the Homebrew tap repository, which may not be ideal for automated releases.

---

## Testing Plan

### Test Environment Setup
1. **macOS 14 Sonoma (ARM)** - Primary test target
2. **macOS 13 Ventura (Intel)** - Compatibility check
3. **macOS 13 Ventura (ARM)** - Architecture check

### Test Cases

#### Test 1: Font Installation (User Scope)
```bash
fontget add google.roboto
# Expected: Font installs successfully, no atsutil errors
# Verify: Font appears in Font Book
# Verify: Font available in applications
```

#### Test 2: Font Installation (Machine Scope)
```bash
sudo fontget add google.roboto --scope machine
# Expected: Font installs to /Library/Fonts
# Verify: Font appears system-wide
```

#### Test 3: Font Removal
```bash
fontget remove google.roboto
# Expected: Font removed successfully
# Verify: Font no longer in Font Book
```

#### Test 4: List Command
```bash
fontget list
# Expected: Lists all installed fonts
# Verify: Shows fonts from both user and machine scopes
```

#### Test 5: List Command (Machine Scope)
```bash
fontget list --scope machine
# Expected: Lists system-wide fonts
# Verify: No permission errors
```

#### Test 6: Install Script (ARM Mac)
```bash
curl -fsSL https://raw.githubusercontent.com/Graphixa/FontGet/main/scripts/install.sh | sh
# Expected: Downloads fontget-darwin-arm64
# Verify: Binary works correctly
```

#### Test 7: Install Script (Intel Mac)
```bash
curl -fsSL https://raw.githubusercontent.com/Graphixa/FontGet/main/scripts/install.sh | sh
# Expected: Downloads fontget-darwin-amd64
# Verify: Binary works correctly
```

#### Test 8: Homebrew Installation (ARM)
```bash
brew install --cask fontget
# Expected: Installs correct ARM binary
# Verify: Binary works correctly
```

#### Test 9: Homebrew Installation (Intel)
```bash
brew install --cask fontget
# Expected: Installs correct Intel binary
# Verify: Binary works correctly
```

---

## Additional Considerations

### 1. Font Cache Behavior on macOS 14+

**Research Findings:**
- macOS 14+ automatically detects fonts in `~/Library/Fonts` and `/Library/Fonts`
- No manual cache refresh required
- Fonts may take a few seconds to appear in Font Book
- Applications may need to be restarted to see new fonts

**Recommendation:**
- Make font cache refresh optional/non-blocking
- Add informational message: "Fonts will be available shortly. You may need to restart applications."

### 2. Permission Handling

**Current Implementation:**
- User scope: `~/Library/Fonts` (no elevation needed)
- Machine scope: `/Library/Fonts` (requires sudo)

**Recommendation:**
- Add clear error messages for permission issues
- Suggest using `--scope user` if machine scope fails
- Check if user has write permissions before attempting installation

### 3. Error Messages

**Current Issues:**
- `atsutil` errors are cryptic for end users
- No guidance on what to do when cache refresh fails

**Recommendation:**
- Add user-friendly error messages
- Explain that fonts may still work even if cache refresh fails
- Provide troubleshooting steps

---

## Implementation Checklist

### Phase 1: Critical Fixes (Immediate)
- [ ] Replace `atsutil` commands with `pkill fontd` or skip cache refresh
- [ ] Test font installation on macOS 14+
- [ ] Test font removal on macOS 14+
- [ ] Update error messages for better user experience

### Phase 2: Error Handling Improvements
- [ ] Add permission checks for font directories
- [ ] Improve error messages for list command
- [ ] Add verbose/debug output for troubleshooting
- [ ] Handle metadata extraction failures gracefully

### Phase 3: Architecture Verification
- [ ] Test install script on ARM Mac
- [ ] Test install script on Intel Mac
- [ ] Verify Homebrew Cask works on both architectures
- [ ] Update documentation if architecture-specific handling needed

### Phase 4: Testing & Validation
- [ ] Test all commands on macOS 14 Sonoma
- [ ] Test all commands on macOS 13 Ventura
- [ ] Verify fonts appear in Font Book
- [ ] Verify fonts work in applications
- [ ] Test both user and machine scopes

---

## References

1. **Apple Type Services Deprecation:**
   - https://en.wikipedia.org/wiki/Apple_Type_Services_for_Unicode_Imaging
   - `atsutil` removed in macOS 14 Sonoma

2. **System Integrity Protection:**
   - https://support.apple.com/guide/security/system-integrity-protection-secb7ea06b49/web
   - Prevents modification of `/System/Library/Fonts`

3. **Font Installation on macOS:**
   - https://support.apple.com/en-mide/guide/font-book/fntbk1000/mac
   - Fonts in `~/Library/Fonts` and `/Library/Fonts` are auto-detected

4. **Font Cache Refresh:**
   - https://discussions.apple.com/thread/255757042
   - Use `pkill fontd` to refresh font cache on macOS 14+

5. **Homebrew Cask Architecture Handling:**
   - https://docs.brew.sh/Cask-Cookbook
   - Use `on_arm` and `on_intel` blocks for architecture-specific binaries

---

## Notes

- The search command works fine because it doesn't interact with the file system or use `atsutil`
- The issue is isolated to commands that modify fonts (add, remove) or read font directories (list)
- The install script correctly handles architecture detection
- GoReleaser builds both ARM and Intel binaries correctly
- Homebrew Cask may need manual updates if architecture-specific handling is required
