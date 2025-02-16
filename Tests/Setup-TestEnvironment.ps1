# Create test directories
$testPaths = @(
    "$env:TEMP\GoogleFonts",
    "$env:LOCALAPPDATA\Microsoft\Windows\Fonts",
    "$env:HOMEPATH\AppData\Local\FontGet"
)

foreach ($path in $testPaths) {
    if (-not (Test-Path $path)) {
        New-Item -ItemType Directory -Path $path -Force
    }
}

# Install Pester if not present
if (-not (Get-Module -ListAvailable -Name Pester)) {
    Install-Module -Name Pester -Force -SkipPublisherCheck
} 