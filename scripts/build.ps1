# FontGet Build Script for Windows
# Simple build script for local testing
# Note: Release builds are handled automatically by GitHub Actions on tag push

param(
    [string]$Version = "",
    [switch]$Help
)

function Show-Help {
    Write-Host "FontGet Build Script" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Usage: .\scripts\build.ps1 [options]"
    Write-Host ""
    Write-Host "Options:"
    Write-Host "  -Version <version>  Build with specific version (for testing release builds locally)"
    Write-Host "  -Help               Show this help message"
    Write-Host ""
    Write-Host "Examples:"
    Write-Host "  .\scripts\build.ps1              # Build for local testing (uses 'dev' version)"
    Write-Host "  .\scripts\build.ps1 -Version 2.1.0  # Test a specific version locally"
    Write-Host ""
    Write-Host "Note: For releases, just create and push a git tag. GitHub Actions will build automatically."
    Write-Host ""
}

if ($Help) {
    Show-Help
    exit 0
}

# Get git info (for build metadata)
$commit = git rev-parse --short HEAD 2>$null
if (-not $commit) { $commit = "unknown" }

$date = [System.DateTime]::UtcNow.ToString("yyyy-MM-ddTHH:mm:ssZ")

# Determine version
if ($Version) {
    # User specified a version (for testing release builds locally)
    $version = $Version
    Write-Host "Building FontGet v$version (local test build)..." -ForegroundColor Cyan
} else {
    # Default: simple dev build for local testing
    $version = "dev"
    Write-Host "Building FontGet (local dev build)..." -ForegroundColor Yellow
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

