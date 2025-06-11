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
    Returns all fonts containing "Open" in their name (e.g., "Open Sans", "roboto").

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
        Write-Progress -Activity "Searching Fonts" -Status "Initializing..." -PercentComplete 0
        
        # Normalize the search terms
        $searchTerm = $Keyword.ToLower().Trim()
        $searchTermNoSpace = $searchTerm.Replace(" ", "")
        Write-Log "Searching for fonts matching: $Keyword" -Level INFO
        Write-Verbose "Using normalized search terms: '$searchTerm', '$searchTermNoSpace'"
        Write-Host "Searching Google Fonts for '$searchTerm'..." -ForegroundColor Cyan
        
        Write-Progress -Activity "Searching Fonts" -Status "Fetching font list from GitHub..." -PercentComplete 20
        
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
            Write-Verbose "Found OFL directory, retrieving contents"
            # Get the contents of the 'ofl' directory
            $oflContents = Invoke-RestMethod -Uri $oflTree.url -Headers $headers -Method Get
            $allFonts = $oflContents.tree
            Write-Verbose "Retrieved $($allFonts.Count) fonts from repository"
            
            Write-Progress -Activity "Searching Fonts" -Status "Scoring and filtering matches..." -PercentComplete 40
            
            # Score and sort all matches first
            Write-Verbose "Scoring matches based on search criteria"
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
                    
                    # Add length penalty
                    if ($score -gt 0) {
                        # Penalize longer names (1 point per character after the search term)
                        $lengthPenalty = $fontName.Length - $searchTerm.Length
                        $score -= $lengthPenalty
                    }
                    
                    if ($score -gt 0) {
                        [PSCustomObject]@{
                            Name = $_.path
                            Score = $score
                        }
                    }
                } |
                Where-Object { $_ -ne $null } |
                Sort-Object -Property Score -Descending |
                Select-Object -First 14

            Write-Verbose "Found $($scoredMatches.Count) matching fonts"

            Write-Progress -Activity "Searching Fonts" -Status "Found $($scoredMatches.Count) matches" -PercentComplete 60

            if ($scoredMatches) {
                Write-Progress -Activity "Searching Fonts" -Status "Finding font information..." -PercentComplete 80
                
                # Now enrich only the top matches with metadata and license URLs
                $enrichedMatches = $scoredMatches | ForEach-Object {
                    $fontId = $_.Name
                    $metadataUrl = "https://raw.githubusercontent.com/google/fonts/main/ofl/$fontId/METADATA.pb"
                    $licenseUrl = "https://github.com/google/fonts/blob/main/ofl/$fontId/OFL.txt"
                    
                    try {
                        $metadata = Invoke-RestMethod -Uri $metadataUrl -Headers $headers -Method Get
                        $friendlyName = if ($metadata -match 'name: "([^"]+)"') {
                            $matches[1]
                        } else {
                            $fontId  # Fallback to font ID if no name found
                        }
                    }
                    catch {
                        Write-Verbose "Could not fetch metadata for $fontId : $_"
                        $friendlyName = $fontId  # Fallback to font ID if metadata fetch fails
                    }

                    [PSCustomObject]@{
                        FriendlyName = $friendlyName
                        FontId = $fontId
                        License = $licenseUrl
                    }
                }

                Write-Progress -Activity "Searching Fonts" -Status "Complete" -Completed

                Write-Host "`nFound matching fonts:`n" -ForegroundColor Green
                
                $enrichedMatches | Format-Table -AutoSize @(
                    @{
                        Label = "Font Name"
                        Expression = { $_.FriendlyName }
                        Width = 40
                    },
                    @{
                        Label = "Font ID"
                        Expression = { $_.FontId }
                        Width = 30
                    },
                    @{
                        Label = "License"
                        Expression = { $_.License }
                        Width = 70
                    }
                )

                Write-Host "`nTo install a font, use: fontget install <font-name>" -ForegroundColor Yellow
            } else {
                Write-Host "`nNo fonts found matching '$searchTerm'" -ForegroundColor Yellow
                Write-Host "Try a different search term or check the spelling" -ForegroundColor DarkGray  # Match other help text color
            }
        } else {
            Write-Log "Could not find OFL directory in repository" -Level ERROR
            throw "Could not find OFL directory in repository"
        }
    }
    catch {
        Write-Progress -Activity "Searching Fonts" -Status "Error" -Completed
        Write-Log "Error searching for fonts: $_" -Level ERROR
        throw $_
    }
} 