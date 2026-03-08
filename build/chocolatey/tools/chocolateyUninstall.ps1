# Remove fontget.exe from tools (shim is removed by Chocolatey automatically).
$ErrorActionPreference = 'Stop'
$toolsDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$exe = Join-Path $toolsDir "fontget.exe"
if (Test-Path $exe) {
  Remove-Item $exe -Force
}
