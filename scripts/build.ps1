# FontGet Build Script for Windows
# Automatically detects version from git tags

param(
    [string]$Version = "",
    [switch]$Dev,
    [switch]$Help
)

function Show-Help {
    Write-Host "FontGet Build Script" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Usage: .\scripts\build.ps1 [options]"
    Write-Host ""
    Write-Host "Options:"
    Write-Host "  -Version <version>  Build with specific version (e.g., 2.0.0)"
    Write-Host "  -Dev                Build as development version (latest release + commit hash)"
    Write-Host "  -Help               Show this help message"
    Write-Host ""
    Write-Host "Examples:"
    Write-Host "  .\scripts\build.ps1              # Auto-detect version from git tag"
    Write-Host "  .\scripts\build.ps1 -Dev         # Build as 'dev' version"
    Write-Host "  .\scripts\build.ps1 -Version 2.1.0  # Build with specific version"
    Write-Host ""
}

if ($Help) {
    Show-Help
    exit 0
}

# Get git info
$commit = git rev-parse --short HEAD 2>$null
if (-not $commit) { $commit = "unknown" }

$date = [System.DateTime]::UtcNow.ToString("yyyy-MM-ddTHH:mm:ssZ")

# Determine version
if ($Dev) {
    # For dev builds: get latest release version + commit hash
    $tag = git describe --tags --abbrev=0 2>$null
    if ($tag) {
        $baseVersion = $tag -replace '^v', ''
        $version = "$baseVersion-dev+$commit"
        Write-Host "Building FontGet (dev build: $version)..." -ForegroundColor Yellow
    } else {
        # No tag found, use plain dev+commit
        $version = "dev+$commit"
        Write-Host "Building FontGet (dev build: $version - no release tag found)..." -ForegroundColor Yellow
    }
} elseif ($Version) {
    $version = $Version
    Write-Host "Building FontGet v$version (release)..." -ForegroundColor Green
} else {
    # Auto-detect from git tag
    $tag = git describe --tags --abbrev=0 2>$null
    if ($tag) {
        $version = $tag -replace '^v', ''
        Write-Host "Building FontGet v$version (auto-detected from git tag)..." -ForegroundColor Cyan
    } else {
        $version = "dev"
        Write-Host "Building FontGet (dev build - no git tag found)..." -ForegroundColor Yellow
    }
}

Write-Host "  Version: $version" -ForegroundColor Gray
Write-Host "  Commit:  $commit" -ForegroundColor Gray
Write-Host "  Date:    $date" -ForegroundColor Gray
Write-Host ""

# Build flags
$ldflags = @(
    "-s",
    "-w",
    "-X fontget/internal/version.Version=$version",
    "-X fontget/internal/version.GitCommit=$commit",
    "-X fontget/internal/version.BuildDate=$date"
)

# Build
$ldflagsStr = $ldflags -join " "
go build -ldflags $ldflagsStr -o fontget.exe .

if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "Build successful! Binary: fontget.exe" -ForegroundColor Green
    Write-Host ""
    Write-Host "Version info:" -ForegroundColor Cyan
    .\fontget.exe version
} else {
    Write-Host ""
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}

