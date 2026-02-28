# Building FontGet

This guide covers how to build FontGet for local testing. **For releases, GitHub Actions handles everything automatically when you push a git tag.**

## Prerequisites

- **Go 1.24+** (see `go.mod`). Install: [go.dev/dl](https://go.dev/dl), or `brew install go` on macOS.
- **Git** (for clone and build metadata).

From the repo root, dependencies are fetched automatically on first build. Optionally run `go mod download` once.

## Build

### Linux/macOS

Run the following from your terminal:

```bash
sh scripts/build.sh
```

- **Output:** `/tmp/fontget-dev` (so the binary runs on all drives, including cloud-synced ones).
- **Run:** `/tmp/fontget-dev version` and `/tmp/fontget-dev search roboto`

**Options:**

```bash
sh scripts/build.sh -h              # Help
sh scripts/build.sh -v 2.1.0        # Build with a specific version string
FONTGET_OUTPUT=./fontget sh scripts/build.sh   # Build in repo (e.g. if not on cloud drive)
```

**Do not run the build script with sudo** — the binary must be owned by you so you can run it.

### Windows

From the repo root in PowerShell:

```powershell
.\scripts\build.ps1
.\scripts\build.ps1 -Version 2.1.0
.\scripts\build.ps1 -Help
```

## Dev vs release

- **Local build:** Version is `dev-YYYYMMDDHHMMSS-<commit>` (e.g. `dev-20260228020445-6c41181`). Binary is at `/tmp/fontget-dev` (Linux/macOS) or in repo working directory (Windows).
- **Release:** Tag and push: `git tag -a v2.1.0 -m "Release v2.1.0" && git push origin v2.1.0`. GitHub Actions builds and publishes; no local build required.

## Manual build (optional)

Without the script:

```bash
go build -o fontget .
```

For version/commit/date in the binary, use the same ldflags as in `scripts/build.sh` (see that file for the exact `-X` flags).

## Troubleshooting

- **Build fails:** Ensure Go 1.24+ (`go version`), you’re in the repo root (where `go.mod` is), and run `go mod tidy` if needed.
- **Permission denied on script:** Run with `sh scripts/build.sh` (or `bash scripts/build.sh`), not `./scripts/build.sh`.
- **Binary won’t run (e.g. on pCloud):** Default output is already `/tmp/fontget-dev`; run `/tmp/fontget-dev`. If you used `FONTGET_OUTPUT=./fontget`, the filesystem may not allow execute — use the default or build to another local path.
- **Windows:** Use the PowerShell script; Make is optional (e.g. via WSL or Chocolatey).
