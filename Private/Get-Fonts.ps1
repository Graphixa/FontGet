<#
.SYNOPSIS
    Retrieves installed fonts from the system.

.DESCRIPTION
    Internal function that gets a list of installed fonts from Windows.
    Returns font information including name, type, and installation location.

.EXAMPLE
    Get-Fonts
    Returns all installed fonts.

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
    param()

    $fonts = @()

    # Get fonts from Windows Fonts directory
    Write-Verbose "Getting fonts from $env:windir\Fonts"
    $systemFonts = Get-ChildItem -Path "$env:windir\Fonts" -Include "*.ttf","*.otf" | 
        Select-Object @{N='Name';E={$_.BaseName}},
                    @{N='Format';E={$_.Extension}},
                    @{N='Path';E={$_.FullName}},
                    @{N='InstallDate';E={$_.CreationTime}}
    $fonts += $systemFonts

    # Extract unique font family names with additional information
    $fontFamilies = $fonts | ForEach-Object {
        # Remove weight/style variations (e.g., -Bold, -Italic, -Regular)
        if ($_.Name -match '^(.*?)(?:-\w+)?$') {
            [PSCustomObject]@{
                Family = $matches[1]
                Name = $_.Name
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
            Format = $_.Group[0].Format
            Path = $_.Group[0].Path
            InstallDate = $_.Group[0].InstallDate
        }
    }

    Write-Verbose "Found $($fontFamilies.Count) font families"
    return $fontFamilies
}

Export-ModuleMember -Function Get-Fonts 