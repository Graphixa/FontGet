<#
.SYNOPSIS
    Uninstalls Google Fonts from Windows.

.DESCRIPTION
    The Uninstall-GoogleFont cmdlet removes installed Google Fonts from the system.
    It removes both the font files and their registry entries.
    Supports uninstallation from both current user and all users contexts.

.PARAMETER Name
    The name of the font(s) to uninstall. Multiple fonts can be specified using a comma-separated list.
    Font names are case-insensitive and partial matches are supported.

.PARAMETER Scope
    Specifies whether to uninstall the font from current user or all users.
    Valid values are 'CurrentUser' and 'AllUsers'. Default is 'AllUsers'.
    Note: 'AllUsers' requires administrator privileges.

.PARAMETER Force
    Suppresses confirmation prompts and forces the uninstallation.

.EXAMPLE
    Uninstall-GoogleFont -Name "Roboto"
    Uninstalls the Roboto font family from all users (requires admin rights).

.EXAMPLE
    Uninstall-GoogleFont -Name "Roboto,Open Sans" -Scope CurrentUser
    Uninstalls both Roboto and Open Sans font families from the current user's fonts.

.EXAMPLE
    Uninstall-GoogleFont "Roboto" -Force
    Forces uninstallation of the Roboto font family without confirmation prompts.

.NOTES
    Author: Graphixa
    Module: FontGet
    Requires: Windows PowerShell 5.1 or PowerShell Core 7.0+
    Requires: Administrator rights for AllUsers scope

.LINK
    https://github.com/Graphixa/FontGet
#>

function Uninstall-GoogleFont {
    [CmdletBinding(SupportsShouldProcess)]
    param (
        [Parameter(Mandatory=$true)]
        [string]$Name
    )

    try {
        # Normalize font name and create variations for matching
        $fontName = $Name.ToLower().Trim()
        $fontNameNoSpace = $fontName.Replace(" ", "")
        Write-Debug "Normalized font name: $fontName"
        Write-Debug "Font name without spaces: $fontNameNoSpace"

        Write-Log "Starting font uninstallation for: $Name" -Level INFO

        # Get installed fonts that match the name
        $fontFolder = "$env:windir\Fonts"
        $registryPath = "HKLM:\Software\Microsoft\Windows NT\CurrentVersion\Fonts"

        # Get all font registry entries
        $fontRegistry = Get-ItemProperty -Path $registryPath
        
        # Filter for matching font entries
        $matchingFonts = $fontRegistry.PSObject.Properties |
            Where-Object { $_.Name -notmatch '^\$|^PS' } |  # Exclude PowerShell metadata properties
            Where-Object {
                $regName = $_.Name.ToLower()
                $regValue = $_.Value.ToLower()
                
                # Match against font name and file name
                $regName.StartsWith($fontName) -or 
                $regName.StartsWith($fontNameNoSpace) -or
                $regValue.StartsWith($fontName) -or
                $regValue.StartsWith($fontNameNoSpace)
            }

        if (-not $matchingFonts) {
            Write-Log "No fonts found matching '$fontName'" -Level WARNING
            Write-Warning "No fonts found matching '$fontName'"
            return
        }

        Write-Debug "Found $($matchingFonts.Count) matching entries"
        foreach ($font in $matchingFonts) {
            $fontFileName = $font.Value
            $fontFilePath = Join-Path $fontFolder $fontFileName

            Write-Debug "Processing: $($font.Name) -> $fontFileName"

            if ($PSCmdlet.ShouldProcess($fontFileName, "Uninstall font")) {
                Write-Log "Removing font: $($font.Name) ($fontFileName)" -Level INFO

                # Remove registry entry first
                try {
                    Remove-ItemProperty -Path $registryPath -Name $font.Name -ErrorAction Stop
                    Write-Log "Removed registry entry: $($font.Name)" -Level INFO
                }
                catch {
                    Write-Log "Failed to remove registry entry: $($font.Name) - $_" -Level ERROR
                    Write-Warning "Failed to remove registry entry: $($font.Name)"
                    continue
                }

                # Then try to remove the font file
                if (Test-Path $fontFilePath) {
                    try {
                        Remove-Item $fontFilePath -Force -ErrorAction Stop
                        Write-Log "Removed font file: $fontFileName" -Level INFO
                    }
                    catch {
                        Write-Log "Could not remove font file: $fontFileName - $_" -Level WARNING
                        Write-Warning "Could not remove font file: $fontFileName. It may be in use."
                    }
                }
                else {
                    Write-Log "Font file not found: $fontFileName" -Level WARNING
                }
            }
        }

        Write-Log "Font uninstallation completed for: $Name" -Level INFO
    }
    catch {
        Write-Log "Error uninstalling font: $_" -Level ERROR
        throw $_
    }
} 