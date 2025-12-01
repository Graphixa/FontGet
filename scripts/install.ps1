# FontGet Installer Script for Windows
# Installs FontGet CLI tool on Windows (PowerShell)
#
# Usage:
#   irm https://raw.githubusercontent.com/Graphixa/FontGet/main/scripts/install.ps1 | iex
#   # Or with a specific version:
#   $env:FONTGET_VERSION="1.0.0"; irm https://raw.githubusercontent.com/Graphixa/FontGet/main/scripts/install.ps1 | iex
#

# Set error handling
$ErrorActionPreference = "Stop"
$ProgressPreference = 'SilentlyContinue'

# Repository information
$Repo = "Graphixa/FontGet"
$RepoUrl = "https://github.com/$Repo"

# Detect architecture
$IsArm64 = $env:PROCESSOR_ARCHITECTURE -eq "ARM64" -or $env:PROCESSOR_ARCHITEW6432 -eq "ARM64"
if ($IsArm64) {
    $Arch = "arm64"
} elseif ([Environment]::Is64BitOperatingSystem) {
    $Arch = "amd64"
} else {
    Write-Error "32-bit Windows is not supported. FontGet requires 64-bit Windows."
    exit 1
}

# Determine version to install
$Version = $env:FONTGET_VERSION
if (-not $Version -or $Version -eq "") {
    $Version = "latest"
}

if ($Version -eq "latest") {
    $BaseUrl = "$RepoUrl/releases/latest/download"
    Write-Host "Installing latest version of FontGet..." -ForegroundColor Blue
} else {
    # Remove 'v' prefix if present
    $Version = $Version -replace '^v', ''
    $BaseUrl = "$RepoUrl/releases/download/v$Version"
    Write-Host "Installing FontGet v$Version..." -ForegroundColor Blue
}

# Binary name (with .exe extension for Windows)
$BinaryName = "fontget-windows-$Arch.exe"
$DownloadUrl = "$BaseUrl/$BinaryName"

# Installation directory (user-local, no admin required)
$InstallDir = "$env:USERPROFILE\AppData\Local\Programs\FontGet"
$InstalledBin = Join-Path $InstallDir "fontget.exe"

# Check if fontget is already installed
if (Test-Path $InstalledBin) {
    try {
        $CurrentVersion = & $InstalledBin version 2>$null | Select-Object -First 1
        Write-Host "FontGet is already installed at: $InstalledBin" -ForegroundColor Yellow
        Write-Host "Current version: $CurrentVersion" -ForegroundColor Yellow
        Write-Host "This will be overwritten." -ForegroundColor Yellow
        Write-Host ""
    } catch {
        # Ignore errors when checking version
    }
}

# Create installation directory
Write-Host "Creating installation directory..." -ForegroundColor Blue
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

# Create temporary file for download
$TempFile = Join-Path $env:TEMP "fontget-installer-$Arch.exe"

# Download binary
Write-Host "Downloading FontGet..." -ForegroundColor Blue
try {
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $TempFile -UseBasicParsing
} catch {
    Write-Error "Failed to download FontGet"
    if ($Version -ne "latest") {
        Write-Host "Version v$Version may not exist. Check available versions at:" -ForegroundColor Yellow
        Write-Host "$RepoUrl/releases" -ForegroundColor Blue
    }
    exit 1
}

# Verify file was downloaded
if (-not (Test-Path $TempFile)) {
    Write-Error "Downloaded file not found"
    exit 1
}

# Verify binary works (basic check)
try {
    $TestOutput = & $TempFile version 2>&1
    if ($LASTEXITCODE -ne 0 -and $TestOutput -match "error|Error|ERROR") {
        throw "Binary validation failed"
    }
} catch {
    Write-Error "Downloaded binary appears to be invalid"
    Remove-Item $TempFile -Force -ErrorAction SilentlyContinue
    exit 1
}

# Install binary
Write-Host "Installing to $InstallDir..." -ForegroundColor Blue
Move-Item -Path $TempFile -Destination $InstalledBin -Force

# Get installed version
try {
    $InstalledVersion = & $InstalledBin version 2>$null | Select-Object -First 1
} catch {
    $InstalledVersion = "FontGet"
}

Write-Host ""
Write-Host "✓ FontGet installed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "  Location: $InstalledBin"
Write-Host "  Version:  $InstalledVersion"
Write-Host ""

# Check if install directory is in PATH
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
$PathEntries = if ($UserPath) { $UserPath -split ';' } else { @() }

if ($PathEntries -contains $InstallDir) {
    Write-Host "✓ $InstallDir is already in your PATH" -ForegroundColor Green
} else {
    Write-Host "⚠ $InstallDir is not in your PATH" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Adding to PATH..." -ForegroundColor Blue
    
    $NewPath = if ($UserPath) { "$UserPath;$InstallDir" } else { $InstallDir }
    [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
    
    # Update current session PATH
    $env:Path = "$env:Path;$InstallDir"
    
    Write-Host "✓ Added to PATH" -ForegroundColor Green
    Write-Host ""
    Write-Host "Note: You may need to restart your terminal for 'fontget' to be available." -ForegroundColor Yellow
    Write-Host "Or run FontGet directly:" -ForegroundColor Yellow
    Write-Host "  $InstalledBin --help" -ForegroundColor Blue
}

Write-Host ""
Write-Host "You can now use 'fontget' to manage your fonts!" -ForegroundColor Green
Write-Host "  fontget search `"roboto`"" -ForegroundColor Blue
Write-Host "  fontget add google.roboto" -ForegroundColor Blue
Write-Host ""

