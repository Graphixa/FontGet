# Clean up test directories
$testDirs = @(
    "$env:TEMP\GoogleFonts",
    "$env:LOCALAPPDATA\FontGet"
)

foreach ($dir in $testDirs) {
    if (Test-Path $dir) {
        Write-Host "Cleaning up $dir"
        Remove-Item -Path $dir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# Clean up test registry entries only
Write-Host "Cleaning test entries from HKCU:\Software\Microsoft\Windows NT\CurrentVersion\Fonts"
Remove-ItemProperty -Path "HKCU:\Software\Microsoft\Windows NT\CurrentVersion\Fonts" -Name "OpenSans*" -ErrorAction SilentlyContinue

Write-Host "Cleaning test entries from HKLM:\Software\Microsoft\Windows NT\CurrentVersion\Fonts"
Remove-ItemProperty -Path "HKLM:\Software\Microsoft\Windows NT\CurrentVersion\Fonts" -Name "OpenSans*" -ErrorAction SilentlyContinue

Write-Host "Cleanup complete" 