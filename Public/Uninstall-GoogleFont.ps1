<#
.SYNOPSIS
    Uninstalls Google Fonts from Windows.

.DESCRIPTION
    The Uninstall-GoogleFont cmdlet removes installed Google Fonts from the system.
    It removes both the font files and their registry entries.

.PARAMETER Name
    The name of the font to uninstall.
    Font names are case-insensitive and partial matches are supported.

.EXAMPLE
    Uninstall-GoogleFont -Name "Roboto"
    Uninstalls the Roboto font family.

.EXAMPLE
    Uninstall-GoogleFont "Open Sans"
    Uninstalls the Open Sans font family.

.NOTES
    Author: Graphixa
    Module: FontGet
    Requires: Windows PowerShell 5.1 or PowerShell Core 7.0+
    Requires: Administrator rights

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
        Write-Progress -Activity "Uninstalling Font" -Status "Searching for font files" -PercentComplete 10
        
        # Normalize font name and create variations for matching
        $fontName = $Name.ToLower().Trim()
        $fontNameNoSpace = $fontName.Replace(" ", "")
        Write-Verbose "Searching for font: $Name"
        Write-Verbose "Using normalized names: '$fontName', '$fontNameNoSpace'"

        Write-Log "Starting font uninstallation for: $Name" -Level INFO

        # Get installed fonts that match the name
        $fontFolder = "$env:windir\Fonts"
        $registryPath = "HKLM:\Software\Microsoft\Windows NT\CurrentVersion\Fonts"

        Write-Progress -Activity "Uninstalling Font" -Status "Checking registry entries" -PercentComplete 25
        Write-Verbose "Checking registry for matching fonts at: $registryPath"
        
        # Get all font registry entries
        $fontRegistry = Get-ItemProperty -Path $registryPath
        
        # Filter for matching font entries
        $matchingFonts = $fontRegistry.PSObject.Properties |
            Where-Object { $_.Name -notmatch '^\$|^PS' } |
            Where-Object {
                $regName = $_.Name.ToLower()
                $regValue = $_.Value.ToLower()
                
                # Match against font name and file name
                $regName.StartsWith($fontName) -or 
                $regName.StartsWith($fontNameNoSpace) -or
                $regValue.StartsWith($fontName) -or
                $regValue.StartsWith($fontNameNoSpace)
            }

        Write-Progress -Activity "Uninstalling Font" -Status "Found $($matchingFonts.Count) matching fonts" -PercentComplete 50

        if (-not $matchingFonts) {
            Write-Log "No fonts found matching '$fontName'" -Level WARNING
            Write-Warning "No fonts found matching '$fontName'"
            Write-Progress -Activity "Uninstalling Font" -Status "Completed" -Completed
            return
        }

        Write-Verbose "Found $($matchingFonts.Count) matching font entries"
        
        $currentFont = 0
        $totalFonts = $matchingFonts.Count
        
        foreach ($font in $matchingFonts) {
            $currentFont++
            $percentComplete = 50 + (($currentFont / $totalFonts) * 50)
            
            $fontFileName = $font.Value
            $fontFilePath = Join-Path $fontFolder $fontFileName

            Write-Progress -Activity "Uninstalling Font" -Status "Removing font: $($font.Name)" -PercentComplete $percentComplete
            Write-Verbose "Processing font: $($font.Name) ($fontFileName)"

            if ($PSCmdlet.ShouldProcess($fontFileName, "Uninstall font")) {
                Write-Log "Removing font: $($font.Name) ($fontFileName)" -Level INFO
                Write-Verbose "Removing registry entry: $($font.Name)"

                # Remove registry entry first
                try {
                    Remove-ItemProperty -Path $registryPath -Name $font.Name -ErrorAction Stop
                    Write-Verbose "Successfully removed registry entry"
                    Write-Log "Removed registry entry: $($font.Name)" -Level INFO
                }
                catch {
                    Write-Log "Failed to remove registry entry: $($font.Name) - $_" -Level ERROR
                    Write-Warning "Failed to remove registry entry: $($font.Name)"
                    continue
                }

                # Then try to remove the font file
                if (Test-Path $fontFilePath) {
                    Write-Verbose "Removing font file: $fontFilePath"
                    try {
                        Remove-Item $fontFilePath -Force -ErrorAction Stop
                        Write-Verbose "Successfully removed font file"
                        Write-Log "Removed font file: $fontFileName" -Level INFO
                    }
                    catch {
                        Write-Log "Could not remove font file: $fontFileName - $_" -Level WARNING
                        Write-Warning "Could not remove font file: $fontFileName. It may be in use."
                    }
                }
                else {
                    Write-Verbose "Font file not found: $fontFilePath"
                    Write-Log "Font file not found: $fontFileName" -Level WARNING
                }
            }
        }

        Write-Progress -Activity "Uninstalling Font" -Status "Completed" -Completed
        Write-Verbose "Font uninstallation completed for: $Name"
        Write-Log "Font uninstallation completed for: $Name" -Level INFO
    }
    catch {
        Write-Progress -Activity "Uninstalling Font" -Status "Error" -Completed
        Write-Log "Error uninstalling font: $_" -Level ERROR
        throw $_
    }
} 