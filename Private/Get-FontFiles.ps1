<#
.SYNOPSIS
    Downloads font files from Google Fonts repository.

.DESCRIPTION
    Internal function that handles downloading font files from Google's GitHub repository.
    Retrieves font metadata and downloads the actual font files.

.PARAMETER FontName
    The name of the font to download.

.PARAMETER OutputPath
    The directory where the font files should be saved.

.EXAMPLE
    Get-FontFiles -FontName "Roboto" -OutputPath "C:\Temp\Fonts"
    Downloads all Roboto font files to the specified directory.

.NOTES
    Internal function, not exported.
    Requires internet connection to access GitHub API and download files.
    Author: Graphixa
    Module: FontGet

.OUTPUTS
    System.String[]
    Returns an array of paths to the downloaded font files.

.LINK
    https://github.com/Graphixa/FontGet
#>

function Normalize-FontName {
    param([string]$Name)
    
    Write-Verbose "Original font name: $Name"
    
    # Clean and normalize the name
    $normalizedName = $Name.ToLower().Trim()
    Write-Verbose "After lowercase and trim: $normalizedName"
    
    # Handle multiple words by adding hyphens
    # This will convert "open sans" or "opensans" to "open-sans"
    if ($normalizedName -match "[a-z]+[A-Z][a-z]" -or $normalizedName.Contains(" ")) {  # Fixed contains check
        Write-Verbose "Contains spaces or camelCase"
        
        # First split by spaces if any
        $words = $normalizedName -split " "
        
        if ($words.Count -eq 1) {
            Write-Verbose "No spaces found, trying to split camelCase"
            # If no spaces, try to split the word into parts
            $parts = @()
            $currentWord = ""
            
            for ($i = 0; $i -lt $normalizedName.Length; $i++) {
                $char = $normalizedName[$i]
                if ($i -gt 0 -and $char -match '[a-z]' -and $normalizedName[$i-1] -match '[a-z]') {
                    $currentWord += $char
                } else {
                    if ($currentWord) {
                        $parts += $currentWord
                        $currentWord = ""
                    }
                    $currentWord += $char
                }
            }
            if ($currentWord) {
                $parts += $currentWord
            }
            $words = $parts
        }
        
        Write-Verbose "Split words: $($words -join ', ')"
        
        # Clean each word and join with hyphens
        $normalizedName = ($words | Where-Object { $_ } | ForEach-Object { $_.ToLower().Trim() }) -join "-"
    }
    
    Write-Verbose "Final normalized name: $normalizedName"
    return $normalizedName
}

function Get-FontFiles {
    [CmdletBinding()]
    param (
        [Parameter(Mandatory=$true)]
        [string]$FontName,

        [Parameter(Mandatory=$true)]
        [string]$OutputPath
    )

    try {
        Write-Log "Starting font download process for: $FontName" -Level INFO
        Write-Log "Output path: $OutputPath" -Level INFO
        
        # Create the base output directory first
        if (-not (Test-Path -Path $OutputPath)) {
            Write-Log "Creating base directory: $OutputPath" -Level INFO
            New-Item -ItemType Directory -Path $OutputPath -Force -ErrorAction Stop | Out-Null
        }

        # Split and process each font name if multiple are provided
        $fontNames = $FontName -split ',' | ForEach-Object { 
            $normalized = Normalize-FontName $_
            Write-Log "Normalized font name from '$_' to '$normalized'" -Level INFO
            $normalized
        }
        
        $downloadedFiles = @()
        
        foreach ($font in $fontNames) {
            Write-Log "Processing font: $font" -Level INFO
            
            # Try both with and without hyphen
            $urlVariants = @(
                "https://api.github.com/repos/google/fonts/contents/ofl/$font",
                "https://api.github.com/repos/google/fonts/contents/ofl/$($font.Replace('-', ''))"
            )

            $success = $false
            foreach ($apiUrl in $urlVariants) {
                Write-Log "Attempting to access: $apiUrl" -Level INFO
                
                try {
                    $response = Invoke-RestMethod -Uri $apiUrl -Method Get -ErrorAction Stop
                    Write-Log "Successfully accessed API URL" -Level INFO
                    
                    if ($response) {
                        $fontFiles = $response | Where-Object { $_.type -eq "file" -and ($_.name -like "*.ttf" -or $_.name -like "*.otf") }
                        Write-Log "Found $($fontFiles.Count) font files" -Level INFO
                        
                        if ($fontFiles) {
                            foreach ($file in $fontFiles) {
                                # Create a safe filename by replacing problematic characters
                                $safeFileName = $file.name.Replace('[', '_').Replace(']', '_')
                                $outputFile = Join-Path -Path $OutputPath -ChildPath $safeFileName
                                Write-Log "Downloading: $($file.name) to $outputFile" -Level INFO
                                
                                try {
                                    $webClient = New-Object System.Net.WebClient
                                    $webClient.DownloadFile($file.download_url, $outputFile)
                                    Write-Log "Successfully downloaded: $($file.name)" -Level INFO
                                    $downloadedFiles += $outputFile
                                }
                                catch {
                                    Write-Log "Failed to download file: $($file.name) - Error: $_" -Level ERROR
                                }
                            }
                            $success = $true
                            break
                        } else {
                            Write-Log "No font files found in response" -Level WARNING
                        }
                    }
                }
                catch {
                    Write-Log "Failed to access: $apiUrl - Error: $_" -Level WARNING
                    continue
                }
            }

            if (-not $success) {
                Write-Log "Font '$font' not found in Google Fonts repository after trying all variants" -Level WARNING
            }
        }

        if ($downloadedFiles.Count -eq 0) {
            Write-Log "No font files were downloaded for any variant" -Level ERROR
            throw "No font files were downloaded"
        }
        
        Write-Log "Successfully downloaded $($downloadedFiles.Count) font files" -Level INFO
        return $downloadedFiles
    }
    catch {
        Write-Log "Error in Get-FontFiles: $_" -Level ERROR
        throw $_
    }
}

Export-ModuleMember -Function Get-FontFiles 