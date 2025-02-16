<#
.SYNOPSIS
    Searches for fonts in the Google Fonts repository.

.DESCRIPTION
    The Search-GoogleFont cmdlet queries the Google Fonts repository to find available fonts.
    It searches through font names and returns matching fonts that can be installed.
    The search is case-insensitive and supports partial matches.

.PARAMETER Keyword
    The search term to find fonts. Can be a partial font name.
    Special characters are automatically handled.

.EXAMPLE
    Search-GoogleFont "Roboto"
    Searches for fonts containing "Roboto" in their name.

.EXAMPLE
    Search-GoogleFont "Open"
    Returns all fonts containing "Open" in their name (e.g., "Open Sans", "OpenDyslexic").

.NOTES
    Author: Graphixa
    Module: FontGet
    Requires: Windows PowerShell 5.1 or PowerShell Core 7.0+
    Requires: Internet connection to access Google Fonts repository

.OUTPUTS
    Displays a formatted table of matching fonts with their names and Google Fonts URLs.
    Each result includes:
    - Name: The font family name
    - URL: Direct link to the font on Google Fonts

.LINK
    https://github.com/Graphixa/FontGet

.LINK
    https://fonts.google.com/
#>

function Search-GoogleFont {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory=$true)]
        [string]$Keyword
    )

    try {
        # Normalize the search terms
        $searchTerm = $Keyword.ToLower().Trim()
        $searchTermNoSpace = $searchTerm.Replace(" ", "")
        Write-Debug "Original keyword: $Keyword"
        Write-Debug "Normalized search term: $searchTerm"
        Write-Debug "Search term without spaces: $searchTermNoSpace"
        Write-Host "Searching Google Fonts for '$searchTerm'..." -ForegroundColor Cyan
        
        # Call GitHub API to get list of fonts
        $apiUrl = "https://api.github.com/repos/google/fonts/git/trees/main"
        $headers = @{
            "Accept" = "application/vnd.github.v3+json"
        }
        
        # Get the main tree
        $mainTree = Invoke-RestMethod -Uri $apiUrl -Headers $headers -Method Get
        
        # Find the 'ofl' directory
        $oflTree = $mainTree.tree | Where-Object { $_.path -eq 'ofl' }
        if ($oflTree) {
            # Get the contents of the 'ofl' directory
            $oflContents = Invoke-RestMethod -Uri $oflTree.url -Headers $headers -Method Get
            $allFonts = $oflContents.tree
            Write-Debug "Total fonts found: $($allFonts.Count)"
            
            # Score and sort all matches
            $scoredMatches = $allFonts | 
                ForEach-Object {
                    $fontName = $_.path.ToLower()
                    $fontNameNoSpace = $fontName.Replace("-", "")
                    
                    # Calculate score based on match type
                    $score = 0
                    if ($fontName -eq $searchTerm -or $fontName -eq $searchTermNoSpace) { 
                        $score = 100  # Exact match
                    }
                    elseif ($fontNameNoSpace -eq $searchTermNoSpace) {
                        $score = 90   # Match when removing spaces/hyphens
                    }
                    elseif ($fontName.StartsWith($searchTerm) -or $fontName.StartsWith($searchTermNoSpace)) { 
                        $score = 80   # Starts with search term
                    }
                    elseif ($fontNameNoSpace.StartsWith($searchTermNoSpace)) {
                        $score = 70   # Starts with (normalized)
                    }
                    elseif ($fontName.Contains($searchTerm) -or $fontName.Contains($searchTermNoSpace)) { 
                        $score = 60   # Contains search term
                    }
                    elseif ($fontNameNoSpace.Contains($searchTermNoSpace)) {
                        $score = 50   # Contains (normalized)
                    }
                    
                    if ($score -gt 0) {
                        [PSCustomObject]@{
                            Name = $_.path
                            URL = "https://fonts.google.com/specimen/$($_.path)"
                            Score = $score
                        }
                    }
                } |
                Where-Object { $_ -ne $null } |
                Sort-Object -Property Score -Descending |
                Select-Object -First 14

            Write-Debug "Matches found: $($scoredMatches.Count)"

            if ($scoredMatches) {
                Write-Host "`nFound matching fonts:`n" -ForegroundColor Green
                
                $scoredMatches | 
                    Select-Object Name, URL | 
                    Format-Table -AutoSize @(
                        @{
                            Label = "Font Name"
                            Expression = { $_.Name }
                            Width = 30
                        },
                        @{
                            Label = "Google Fonts URL"
                            Expression = { $_.URL }
                        }
                    )

                Write-Host "`nTo install a font, use: gfont install <font-name>" -ForegroundColor Yellow
            } else {
                Write-Host "`nNo fonts found matching '$searchTerm'" -ForegroundColor Yellow
                Write-Host "Try a different search term or check the spelling" -ForegroundColor Yellow
            }
        } else {
            Write-Error "Could not find the 'ofl' directory in the repository"
        }
    }
    catch {
        Write-Error "Error searching fonts: $_"
    }
} 