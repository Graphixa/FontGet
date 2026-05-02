# FontGet Installer Script for Windows
# Installs FontGet CLI tool on Windows (PowerShell)
#
# Usage:
#   irm https://raw.githubusercontent.com/Graphixa/FontGet/main/scripts/install.ps1 | iex
#   # Or with a specific version:
#   $env:FONTGET_VERSION="1.0.0"; irm https://raw.githubusercontent.com/Graphixa/FontGet/main/scripts/install.ps1 | iex
#
# Environment:
#   FONTGET_VERSION          Version or latest (default: latest)
#   FONTGET_NONINTERACTIVE=1 Skip "Continue?" (non-interactive install)
#   CI                       When non-empty, prompt is skipped (common on CI runners)
#   NO_COLOR=1               Disable ANSI colors / subdued styling where supported
#

# Set error handling
$ErrorActionPreference = "Stop"
$ProgressPreference = 'SilentlyContinue'

function Write-InstallDimLine {
    param([Parameter(Mandatory)][string]$Message)
    if ($env:NO_COLOR -eq '1') {
        Write-Host $Message
    } else {
        Write-Host $Message -ForegroundColor DarkGray
    }
}

function Write-InstallOsLine {
    param([Parameter(Mandatory)][string]$OsName, [Parameter(Mandatory)][string]$ArchName)
    if ($env:NO_COLOR -eq '1') {
        Write-Host "OS: $OsName | ARCH: $ArchName"
        return
    }
    Write-Host -NoNewline -ForegroundColor Blue "OS: "
    Write-Host -NoNewline "$OsName "
    Write-Host -NoNewline -ForegroundColor Blue "| "
    Write-Host -NoNewline -ForegroundColor Blue "ARCH: "
    Write-Host $ArchName
}

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
} else {
    # Remove 'v' prefix if present
    $Version = $Version -replace '^v', ''
    $BaseUrl = "$RepoUrl/releases/download/v$Version"
}

$DisplayVersion = $Version

# Binary name (with .exe extension for Windows)
$BinaryName = "fontget-windows-$Arch.exe"
$DownloadUrl = "$BaseUrl/$BinaryName"

# Installation directory (user-local, no admin required)
$InstallDir = "$env:USERPROFILE\AppData\Local\Programs\FontGet"
$InstalledBin = Join-Path $InstallDir "fontget.exe"

# --- splash (default terminal foreground — no accent color on banner); aligned with install.sh ---
$SplashBanner = @'

███████╗░█████╗░███╗░░██╗████████╗░██████╗░███████╗████████╗
██╔════╝██╔══██╗████╗░██║╚══██╔══╝██╔════╝░██╔════╝╚══██╔══╝
█████╗░░██║░░██║██╔██╗██║░░░██║░░░██║░░██╗░█████╗░░░░░██║░░░
██╔══╝░░██║░░██║██║╚████║░░░██║░░░██║░░╚██╗██╔══╝░░░░░██║░░░
██║░░░░░╚█████╔╝██║░╚███║░░░██║░░░╚██████╔╝███████╗░░░██║░░░
╚═╝░░░░░░╚════╝░╚═╝░░╚══╝░░░╚═╝░░░░╚═════╝░╚══════╝░░░╚═╝░░░

'@
Write-Host $SplashBanner

$Tagline = "Discover, install & manage fonts from the command line."
$SplashW = 60
if ($Tagline.Length -gt $SplashW) {
    Write-InstallDimLine $Tagline
} else {
    $padL = [math]::Floor(($SplashW - $Tagline.Length) / 2)
    $padR = $SplashW - $Tagline.Length - $padL
    Write-InstallDimLine ((" " * $padL) + $Tagline + (" " * $padR))
}
Write-Host ""

Write-InstallOsLine -OsName "windows" -ArchName $Arch
Write-Host ""

Write-Host "This will install FontGet $DisplayVersion to $InstalledBin"
Write-Host ""

# Existing install notice before prompt (aligned with install.sh)
if (Test-Path $InstalledBin) {
    try {
        $CurrentVersion = & $InstalledBin version 2>$null | Select-Object -First 1
        if ($env:NO_COLOR -eq '1') {
            Write-Host "FontGet is already installed at: $InstalledBin"
            Write-Host "Current version: $CurrentVersion"
            Write-Host "This will be overwritten."
        } else {
            Write-Host "FontGet is already installed at: $InstalledBin" -ForegroundColor Yellow
            Write-Host "Current version: $CurrentVersion" -ForegroundColor Yellow
            Write-Host "This will be overwritten." -ForegroundColor Yellow
        }
        Write-Host ""
    } catch {
        # Ignore errors when checking version
    }
}

# Continue prompt: only when interactive; CI / non-TTY / NONINTERACTIVE skip (aligned with install.sh)
$ShouldPrompt = $true
if ($env:FONTGET_NONINTERACTIVE -eq '1') {
    $ShouldPrompt = $false
} elseif ($env:CI) {
    $ShouldPrompt = $false
} else {
    try {
        if ([Console]::IsInputRedirected -or [Console]::IsOutputRedirected) {
            $ShouldPrompt = $false
        }
    } catch {
        $ShouldPrompt = $false
    }
}

if ($ShouldPrompt) {
    Write-Host -NoNewline "Continue? [y/N] "
    try {
        $reply = [Console]::ReadLine()
    } catch {
        $reply = "n"
    }
    if ([string]::IsNullOrWhiteSpace($reply)) {
        $reply = "n"
    }
    $replyNorm = $reply.Trim().ToLowerInvariant()
    if ($replyNorm -ne "y" -and $replyNorm -ne "yes") {
        if ($env:NO_COLOR -eq '1') {
            Write-Host "Cancelled."
        } else {
            Write-Host "Cancelled." -ForegroundColor Yellow
        }
        exit 0
    }
    Write-Host ""
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

# Verify SHA256 against release checksums.txt (GoReleaser)
$ChecksumsUrl = "$BaseUrl/checksums.txt"
$ChecksumsTemp = Join-Path $env:TEMP "fontget-checksums-$Arch.txt"
Write-Host "Downloading checksums..." -ForegroundColor Blue
try {
    Invoke-WebRequest -Uri $ChecksumsUrl -OutFile $ChecksumsTemp -UseBasicParsing
} catch {
    Write-Error "Failed to download checksums.txt"
    Remove-Item $TempFile -Force -ErrorAction SilentlyContinue
    exit 1
}

$expectedHash = $null
Get-Content -LiteralPath $ChecksumsTemp -ErrorAction SilentlyContinue | ForEach-Object {
    $line = $_.TrimEnd("`r")
    if ($line -match '^\s*([A-Fa-f0-9]{64})\s+[\*]?\s*(.+)$') {
        $name = $Matches[2].Trim()
        if ($name -eq $BinaryName) {
            $expectedHash = $Matches[1].ToLowerInvariant()
        }
    }
}
Remove-Item $ChecksumsTemp -Force -ErrorAction SilentlyContinue

if (-not $expectedHash) {
    Write-Error "No checksum line for $BinaryName in checksums.txt"
    Remove-Item $TempFile -Force -ErrorAction SilentlyContinue
    exit 1
}

$actualHash = (Get-FileHash -LiteralPath $TempFile -Algorithm SHA256).Hash.ToLowerInvariant()
if ($actualHash -ne $expectedHash) {
    Write-Error "Checksum mismatch for $BinaryName"
    Remove-Item $TempFile -Force -ErrorAction SilentlyContinue
    exit 1
}

Write-Host "✓ Checksum verified" -ForegroundColor Green

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

