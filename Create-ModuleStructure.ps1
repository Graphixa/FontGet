# Create base module directory
New-Item -ItemType Directory -Path ".\FontGet" | Out-Null
Set-Location ".\FontGet"

# Create subdirectories
@('Public', 'Private') | ForEach-Object {
    New-Item -ItemType Directory -Path ".\$_" | Out-Null
}

# Create Public function files
@(
    'Install-GoogleFont.ps1',
    'Uninstall-GoogleFont.ps1',
    'Get-GoogleFont.ps1'
) | ForEach-Object {
    New-Item -ItemType File -Path ".\Public\$_" | Out-Null
}

# Create Private function files
@(
    'Write-Log.ps1',
    'Test-FontInstalled.ps1',
    'Get-FontFiles.ps1'
) | ForEach-Object {
    New-Item -ItemType File -Path ".\Private\$_" | Out-Null
}

# Create module files
New-Item -ItemType File -Path ".\FontGet.psm1" | Out-Null
New-Item -ItemType File -Path ".\FontGet.psd1" | Out-Null

# Create the module manifest
$manifestParams = @{
    Path              = ".\FontGet.psd1"
    RootModule       = "FontGet.psm1"
    ModuleVersion    = "0.1.0"
    Author           = "Your Name"
    Description      = "A PowerShell module for installing Google Fonts"
    PowerShellVersion = "5.1"
    FunctionsToExport = @('Install-GoogleFont', 'Uninstall-GoogleFont', 'Get-GoogleFont')
    AliasesToExport   = @('gfont')
    Tags             = @('Font', 'Google Fonts', 'Installation')
}

New-ModuleManifest @manifestParams 