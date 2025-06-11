<#
.SYNOPSIS
    Lists installed fonts on the system.

.DESCRIPTION
    The Show-Fonts cmdlet displays a list of fonts installed on the system.
    Can filter to show only Google fonts or non-Google fonts.

.PARAMETER GoogleOnly
    Shows only fonts that are available on Google Fonts.

.PARAMETER OtherOnly
    Shows only fonts that are not available on Google Fonts.

.EXAMPLE
    Show-Fonts
    Lists all installed fonts.

.EXAMPLE
    Show-Fonts -GoogleOnly
    Lists only fonts that are available on Google Fonts.

.EXAMPLE
    Show-Fonts -OtherOnly
    Lists only fonts that are not available on Google Fonts.

.NOTES
    Author: Graphixa
    Module: FontGet
    Requires: Windows PowerShell 5.1 or PowerShell Core 7.0+

.LINK
    https://github.com/Graphixa/FontGet
#>

function Show-Fonts {
    [CmdletBinding()]
    param(
        [Parameter()]
        [switch]$GoogleOnly,

        [Parameter()]
        [switch]$OtherOnly
    )

    try {
        Write-Log "Getting installed fonts" -Level INFO

        # Get the font registry path
        $registryPath = "HKLM:\Software\Microsoft\Windows NT\CurrentVersion\Fonts"
        
        # Get all fonts from registry
        $fonts = Get-ItemProperty -Path $registryPath
        
        # Show loading message for API
        Write-Host "`nRetrieving Google Fonts data..." -NoNewline -ForegroundColor Cyan
        
        # Get list of Google Fonts for matching
        $apiUrl = "https://api.github.com/repos/google/fonts/git/trees/main"
        $headers = @{
            "Accept" = "application/vnd.github.v3+json"
        }
        
        try {
            # Get the main tree
            $mainTree = Invoke-RestMethod -Uri $apiUrl -Headers $headers -Method Get
            
            # Find the 'ofl' directory
            $oflTree = $mainTree.tree | Where-Object { $_.path -eq 'ofl' }
            if ($oflTree) {
                # Get the contents of the 'ofl' directory
                $oflContents = Invoke-RestMethod -Uri $oflTree.url -Headers $headers -Method Get
                $googleFonts = $oflContents.tree.path
            }
            
            # Clear the loading message
            Write-Host "`r$(' ' * 50)`r" -NoNewline
        }
        catch {
            # Clear the loading message and show error
            Write-Host "`r$(' ' * 50)`r" -NoNewline
            Write-Warning "Unable to retrieve Google Fonts data. Continuing with local fonts only."
            $googleFonts = @()
        }

        # Show processing message
        Write-Host "Processing installed fonts..." -NoNewline -ForegroundColor Cyan
        
        # Create a hashtable to store font families
        $fontFamilies = @{}
        
        # Pre-compile the regex pattern for better performance
        $fontPattern = [regex]"^(.*?)(?:\s*[-_]?\s*(\bBold\b|\bItalic\b|\bLight\b|\bThin\b|\bMedium\b|\bBlack\b|\bSemiBold\b|\bRegular\b|\bExtraBold\b|\bExtraLight\b)?)\s*(?:\(.*\))?$"
        
        # Create a lookup hashtable for Google Fonts
        $googleFontsLookup = @{}
        foreach ($font in $googleFonts) {
            $cleanName = ($font -replace '[^a-zA-Z0-9]', '').ToLower()
            $googleFontsLookup[$cleanName] = $font
        }
        
        # Process fonts and group them by family
        $fonts.PSObject.Properties | 
            Where-Object { $_.Name -notmatch '^\$|^PS' } |
            ForEach-Object {
                $fontName = $_.Name
                $fileName = $_.Value
                
                # Extract font family and weight/style
                $match = $fontPattern.Match($fontName)
                if ($match.Success) {
                    $family = $match.Groups[1].Value.Trim()
                    $weight = if ($match.Groups[2].Value) { $match.Groups[2].Value } else { "Regular" }
                    
                    # Clean family name for Google Fonts matching
                    $cleanFamily = ($family -replace '[^a-zA-Z0-9]', '').ToLower()

                    # Lookup Google Font (much faster than Where-Object)
                    $googleFont = $googleFontsLookup[$cleanFamily]

                    # Create or update family entry
                    if (-not $fontFamilies.ContainsKey($family)) {
                        $fontFamilies[$family] = @{
                            Weights = [System.Collections.Generic.HashSet[string]]::new()
                            GoogleFont = if ($googleFont) { $googleFont } else { "N/A" }
                            IsSystem = $fileName -like "*.fon" -or $fileName -like "*SystemFont*"
                        }
                    }
                    $fontFamilies[$family].Weights.Add($weight) | Out-Null
                }
            }

        # Clear processing message
        Write-Host "`r$(' ' * 50)`r" -NoNewline

        # Display the fonts in a nested format with columns
        $title = if ($GoogleOnly) { "Installed Google Fonts" } elseif ($OtherOnly) { "Installed Other Fonts" } else { "Installed Fonts" }
        Write-Host ("`n{0}:`n" -f $title) -ForegroundColor Green
        
        # Write header
        $familyHeader = "Font Family"
        $googleHeader = "Google Font ID"
        $columnSpacing = 45  # Adjust this value to control space between columns
        
        Write-Host $familyHeader.PadRight($columnSpacing) -NoNewline
        Write-Host $googleHeader -ForegroundColor Yellow
        Write-Host "".PadRight($familyHeader.Length, "-") -NoNewline
        Write-Host "".PadRight($columnSpacing - $familyHeader.Length) -NoNewline  # Add spacing
        Write-Host "".PadRight($googleHeader.Length, "-")

        # Get sorted list of fonts, filtered if GoogleOnly is specified
        $sortedFonts = $fontFamilies.GetEnumerator()
        if ($GoogleOnly) {
            $sortedFonts = $sortedFonts | Where-Object { $_.Value.GoogleFont -ne 'N/A' }
        } elseif ($OtherOnly) {
            $sortedFonts = $sortedFonts | Where-Object { $_.Value.GoogleFont -eq 'N/A' }
        }
        $sortedFonts = $sortedFonts | Sort-Object Key

        if (-not $sortedFonts) {
            Write-Host "No fonts found."
            return
        }

        foreach ($font in $sortedFonts) {
            $family = $font.Key
            $details = $font.Value
            
            # Write family name and Google font name in columns
            Write-Host $family.PadRight($columnSpacing) -NoNewline
            Write-Host $details.GoogleFont -ForegroundColor $(if ($details.GoogleFont -eq "N/A") { "Gray" } else { "Yellow" })
            
            # Write weights indented under the family name
            foreach ($weight in ($details.Weights | Sort-Object)) {
                Write-Host "   - $weight".PadRight($columnSpacing) -ForegroundColor DarkGray
            }
            
            # Add a small space between font families
            Write-Host ""
        }

        Write-Host "`nTotal Font Families: $($sortedFonts.Count)" -ForegroundColor Yellow
        if (-not $GoogleOnly -and -not $OtherOnly) {
            Write-Host "Note: 'N/A' means the font is not available on Google Fonts" -ForegroundColor Gray
        }
    }
    catch {
        Write-Log "Error listing fonts: $_" -Level ERROR
        throw $_
    }
}