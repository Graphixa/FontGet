<#
.SYNOPSIS
    Checks if a font is installed on the system.

.DESCRIPTION
    Internal function that verifies if a specified font is installed.
    Checks the Windows registry for font entries.
    Supports partial name matching.

.PARAMETER Name
    The name of the font to check. Case-insensitive partial matches are supported.

.EXAMPLE
    Test-FontInstalled -Name "Roboto"
    Checks if any Roboto fonts are installed.

.NOTES
    Internal function, not exported.
    Author: Graphixa
    Module: FontGet

.OUTPUTS
    System.Boolean
    Returns $true if the font is installed, $false otherwise.

.LINK
    https://github.com/Graphixa/FontGet
#>

function Test-FontInstalled {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)]
        [string]$Name
    )

    try {
        Write-Log "Checking if font '$Name' is installed" -Level INFO
        
        # Get the font registry path
        $registryPath = "HKLM:\Software\Microsoft\Windows NT\CurrentVersion\Fonts"
        
        # Get all fonts from registry
        $fonts = Get-ItemProperty -Path $registryPath
        
        # Check if any font name contains our search term
        $installed = $fonts.PSObject.Properties | 
            Where-Object { $_.Name -notmatch '^\$|^PS' } |
            Where-Object { $_.Name -like "*$Name*" }
        
        return [bool]$installed
    }
    catch {
        Write-Log "Error checking if font is installed: $_" -Level ERROR
        throw $_
    }
} 