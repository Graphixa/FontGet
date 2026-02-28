# Building FontGet

This guide covers how to build FontGet for local testing. **For releases, GitHub Actions handles everything automatically when you push a git tag.**

## Prerequisites

Before building, you need:

| Requirement | Purpose |
|-------------|---------|
| **Go 1.24+** | Compiler (see `go.mod` for exact version) |
| **Git** | Clone repo and for build metadata (commit hash) |

### Installing Go

- **macOS (Homebrew):** `brew install go`
- **Windows:** [Download the installer](https://go.dev/dl/) or `winget install GoLang.Go`
- **Linux:** Use your package manager (e.g. `sudo apt install golang-go` on Debian/Ubuntu) or [official install](https://go.dev/doc/install)

Check your version:
```bash
go version   # Should be 1.24 or higher
```

### Getting dependencies

After cloning the repo, fetch Go module dependencies once:

```bash
cd /path/to/FontGet
go mod download
# Or simply run a build – Go will download modules automatically
```

Optional (not required for a basic build):

- **Make** – For `make build` / `make build-dev` on Linux/macOS. On Windows use the PowerShell script or install make via Chocolatey/WSL.
- **go-winres** – Only for Windows; used by the Makefile to embed version/icon resources. Builds work without it (you’ll see a warning).

## Quick Start

### Local Testing Builds

**Windows (PowerShell):**
```powershell
# Simple build for local testing (uses 'dev' version)
.\scripts\build.ps1

# Test a specific version locally (optional)
.\scripts\build.ps1 -Version 2.1.0
```

**Linux/macOS:**
```bash
# Simple build for local testing (uses 'dev' version)
./scripts/build.sh

# Test a specific version locally (optional)
./scripts/build.sh -v 2.1.0
```

**Using Makefile (if make is installed):**
```bash
make build          # Local dev build
make build-dev      # Same as above
make version        # Show version info
make help           # Show all targets
```

## How It Works

**For Local Testing:**
- Default build uses `"dev"` version (simple, no complexity)
- Builds include commit hash and build date for debugging
- Perfect for testing changes locally

**For Releases:**
- Just create and push a git tag: `git tag -a v2.1.0 -m "Release v2.1.0" && git push origin v2.1.0`
- GitHub Actions automatically:
  - Detects the tag
  - Builds for all platforms
  - Creates GitHub Release
  - Uploads binaries
- No manual build needed!

## Workflow

### Daily Development

Just build locally for testing:
```powershell
.\scripts\build.ps1
# Output: FontGet dev
```

That's it! Simple and straightforward.

### Testing a Release Version Locally

If you want to test what a release build will look like:
```powershell
.\scripts\build.ps1 -Version 2.1.0
# Output: FontGet v2.1.0
```

### Creating a Release

1. **Commit your changes:**
   ```bash
   git add .
   git commit -m "Add new feature"
   git push
   ```

2. **Create and push a tag:**
   ```bash
   git tag -a v2.1.0 -m "Release v2.1.0"
   git push origin v2.1.0
   ```

3. **GitHub Actions automatically:**
   - Builds binaries for Windows, macOS, Linux (amd64, arm64)
   - Creates GitHub Release
   - Uploads all binaries
   - Users can update via `fontget update`

**That's it!** No manual building needed.

## CI/CD Integration

**GitHub Actions + GoReleaser** handle everything automatically:

- When you push a tag like `v2.1.0`, GitHub Actions triggers
- GoReleaser detects the tag and extracts version
- Builds binaries for all platforms automatically
- Creates GitHub Release with all artifacts
- Self-update system can then detect the new version

**You never need to build releases manually!** Just tag and push.

See `.goreleaser.yaml` for configuration details.

## Manual Build (Advanced)

If you prefer to build without scripts:

```bash
# Simple dev build
go build -o fontget .

# With version info (optional)
COMMIT=$(git rev-parse --short HEAD)
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
go build -ldflags \
  "-X fontget/internal/version.Version=dev \
   -X fontget/internal/version.GitCommit=$COMMIT \
   -X fontget/internal/version.BuildDate=$DATE" \
  -o fontget .
```

## Troubleshooting

### Version shows as "dev"

This is **normal and expected** for local builds. The binary will show `FontGet dev` - this is perfect for local testing.

### I want to test a release version locally

Use the `-Version` flag:
```powershell
.\scripts\build.ps1 -Version 2.1.0
```

### Build fails

- **Go not found** – Install Go (see [Prerequisites](#prerequisites)) and ensure it’s on your `PATH`. Check with `go version`.
- **Wrong directory** – Run the build from the FontGet repo root (where `go.mod` is).
- **Missing or stale modules** – Run `go mod download` or `go mod tidy`, then try again.
- **Version mismatch** – Use Go 1.24 or newer (see `go.mod`).

### Makefile not found (Windows)

Make isn't installed by default on Windows. Use the PowerShell script instead:
```powershell
.\scripts\build.ps1
```

Or install make via:
- WSL (Windows Subsystem for Linux)
- Chocolatey: `choco install make`
- Or use the build scripts directly

