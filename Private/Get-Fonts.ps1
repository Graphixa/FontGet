<#
.SYNOPSIS
    Retrieves installed fonts from the system.

.DESCRIPTION
    Internal function that gets a list of installed fonts from Windows.
    Can retrieve fonts from both system-wide and user-specific locations.
    Returns font information including name, type, and installation location.

.PARAMETER SystemFonts
    When specified, retrieves only system-wide fonts from Windows Fonts directory.
    By default, retrieves both system and user fonts.

.PARAMETER UserFonts
    When specified, retrieves only user fonts from Windows Fonts directory.
    By default, retrieves both system and user fonts.

.EXAMPLE
    Get-Fonts
    Returns all installed fonts from both system and user locations.

.EXAMPLE
    Get-Fonts -SystemFonts
    Returns only system-wide installed fonts.

.EXAMPLE
    Get-Fonts -UserFonts
    Returns only user-specific installed fonts.

.NOTES
    Internal function, not exported.
    Author: Graphixa
    Module: FontGet

.OUTPUTS
    System.Object[]
    Returns an array of font objects with Name, Type, and Format properties.

.LINK
    https://github.com/Graphixa/FontGet
#>
function Get-Fonts {
    [CmdletBinding()]
    param (
        [Parameter()]
        [switch]$SystemFonts,

        [Parameter()]
        [switch]$UserFonts
    )

    $fonts = @()

    # Get system fonts if requested or if no switches specified
    if ($SystemFonts -or (-not ($PSBoundParameters.ContainsKey('SystemFonts') -or $PSBoundParameters.ContainsKey('UserFonts')))) {
        Write-Verbose "Getting system fonts from $env:windir\Fonts"
        $systemFonts = Get-ChildItem -Path "$env:windir\Fonts" -Include "*.ttf","*.otf" | 
            Select-Object @{N='Name';E={$_.BaseName}},
                        @{N='Type';E={'System'}},
                        @{N='Format';E={$_.Extension}},
                        @{N='Path';E={$_.FullName}},
                        @{N='InstallDate';E={$_.CreationTime}}
        $fonts += $systemFonts
    }

    # Get user fonts if requested or if no switches specified
    if ($UserFonts -or (-not ($PSBoundParameters.ContainsKey('SystemFonts') -or $PSBoundParameters.ContainsKey('UserFonts')))) {
        $userFontPath = "$env:LOCALAPPDATA\Microsoft\Windows\Fonts"
        Write-Verbose "Getting user fonts from $userFontPath"
        
        if (Test-Path $userFontPath) {
            $userFonts = Get-ChildItem -Path $userFontPath -Include "*.ttf","*.otf" -ErrorAction SilentlyContinue |
                Select-Object @{N='Name';E={$_.BaseName}},
                            @{N='Type';E={'User'}},
                            @{N='Format';E={$_.Extension}},
                            @{N='Path';E={$_.FullName}},
                            @{N='InstallDate';E={$_.CreationTime}}
            $fonts += $userFonts
        }
    }

    # Extract unique font family names with additional information
    $fontFamilies = $fonts | ForEach-Object {
        # Remove weight/style variations (e.g., -Bold, -Italic, -Regular)
        if ($_.Name -match '^(.*?)(?:-\w+)?$') {
            [PSCustomObject]@{
                Family = $matches[1]
                Name = $_.Name
                Type = $_.Type
                Format = $_.Format
                Path = $_.Path
                InstallDate = $_.InstallDate
            }
        }
    } | Sort-Object Family | Group-Object Family | ForEach-Object {
        # Create a family object with all variations
        [PSCustomObject]@{
            Family = $_.Name
            Variations = $_.Group.Name
            Type = $_.Group[0].Type
            Format = $_.Group[0].Format
            Path = $_.Group[0].Path
            InstallDate = $_.Group[0].InstallDate
        }
    }

    Write-Verbose "Found $($fontFamilies.Count) font families"
    return $fontFamilies
}

Export-ModuleMember -Function Get-Fonts 