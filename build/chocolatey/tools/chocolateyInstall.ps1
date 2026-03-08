# Download FontGet Windows zip from GitHub release and extract.
# Checksum is injected at pack time by the release workflow so the zip download is validated (Chocolatey requirement).
$ErrorActionPreference = 'Stop'
$version = $env:ChocolateyPackageVersion
$tag = "v$version"
$zipName = "fontget_$version_windows_amd64.zip"
$zipUrl = "https://github.com/Graphixa/FontGet/releases/download/$tag/$zipName"
$toolsDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$unzipLocation = Join-Path $env:TEMP "fontget-choco-$version"
if (Test-Path $unzipLocation) { Remove-Item $unzipLocation -Recurse -Force }
New-Item -ItemType Directory -Path $unzipLocation -Force | Out-Null

$packageArgs = @{
  PackageName    = 'fontget'
  Url64bit       = $zipUrl
  UnzipLocation  = $unzipLocation
  Checksum64     = 'CHECKSUM_PLACEHOLDER'
  ChecksumType64 = 'sha256'
}
Install-ChocolateyZipPackage @packageArgs

# Zip contains folder fontget_<version>_windows_amd64/fontget.exe; copy exe to tools for shim
$exe = Get-ChildItem -Path $unzipLocation -Filter "fontget.exe" -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
if ($exe) {
  Copy-Item $exe.FullName -Destination (Join-Path $toolsDir "fontget.exe") -Force
}
Remove-Item $unzipLocation -Recurse -Force -ErrorAction SilentlyContinue
