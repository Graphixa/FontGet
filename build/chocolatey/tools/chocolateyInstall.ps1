# Download FontGet Windows zip from GitHub release and extract.
# Checksum is read from the release's checksums.txt so it always matches the published zip (avoids verification failures).
# GoReleaser zip contains fontget_<version>_windows_amd64/fontget.exe; we extract and place exe in tools for shim.
$ErrorActionPreference = 'Stop'
$version = $env:ChocolateyPackageVersion
$tag = "v$version"
$zipName = "fontget_$version_windows_amd64.zip"
$zipUrl = "https://github.com/Graphixa/FontGet/releases/download/$tag/$zipName"
$checksumsUrl = "https://github.com/Graphixa/FontGet/releases/download/$tag/checksums.txt"
$toolsDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$unzipLocation = Join-Path $env:TEMP "fontget-choco-$version"
if (Test-Path $unzipLocation) { Remove-Item $unzipLocation -Recurse -Force }
New-Item -ItemType Directory -Path $unzipLocation -Force | Out-Null

# Get SHA256 for the Windows zip from release checksums.txt (single source of truth; avoids baked-in checksum mismatch)
$checksumsPath = Join-Path $env:TEMP "fontget-checksums-$version.txt"
Get-ChocolateyWebFile -PackageName 'fontget' -FileFullPath $checksumsPath -Url $checksumsUrl
$checksumsContent = Get-Content -Path $checksumsPath -Raw -ErrorAction Stop
$checksumLine = ($checksumsContent -split "`n") | Where-Object { $_ -match [regex]::Escape($zipName) } | Select-Object -First 1
if (-not $checksumLine) { throw "Could not find checksum for $zipName in checksums.txt" }
$checksum64 = ($checksumLine -split '\s+', 2)[0].Trim()
if (-not $checksum64 -or $checksum64.Length -ne 64) { throw "Invalid SHA256 for $zipName in checksums.txt" }

$packageArgs = @{
  PackageName    = 'fontget'
  Url64bit       = $zipUrl
  UnzipLocation  = $unzipLocation
  Checksum64     = $checksum64
  ChecksumType64 = 'sha256'
}
Install-ChocolateyZipPackage @packageArgs

# Zip contains folder fontget_<version>_windows_amd64/fontget.exe; copy exe to tools so Chocolatey creates shim
$exe = Get-ChildItem -Path $unzipLocation -Filter "fontget.exe" -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
if ($exe) {
  Copy-Item $exe.FullName -Destination (Join-Path $toolsDir "fontget.exe") -Force
}
Remove-Item $unzipLocation -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item $checksumsPath -Force -ErrorAction SilentlyContinue
