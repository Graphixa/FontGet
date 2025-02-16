function Write-Log {
    param (
        [string]$msg,
        [string]$level
    )
    $logPath = "$env:HOMEPATH\AppData\Local\FontGet"

    If (-not (Test-Path -Path $logPath)) {
        New-Item -ItemType Directory -Path $logPath | Out-Null
    }

    $logFile = "GetFont.log"

    #Check Log file size and delete if it is greater than 10MB
    If ((Get-Item -Path "$logPath\$logFile").Length -gt 10485760) {
        Remove-Item -Path "$logPath\$logFile" -Force
        Out-File -FilePath "$logPath\$logFile" -InputObject "$msg" -ForegroundColor $level | Out-Null
    }

    Out-File -FilePath "$logPath\$logFile" -Append -InputObject "$msg" -ForegroundColor $level
}


function FontGet {
    
    Write-Host "Installing Google Fonts"

    $tempDownloadFolder = "$env:TEMP\google_fonts"

    # Function to check if a font is installed
    function Test-FontInstalled {
        param (
            [string]$FontName
        )

        $fontRegistryPath = "HKLM:\Software\Microsoft\Windows NT\CurrentVersion\Fonts"
        $installedFonts = Get-ItemProperty -Path $fontRegistryPath

        # Normalize the font name to lowercase for case-insensitive partial match
        $normalizedFontName = $FontName.ToLower()

        # Loop through the installed fonts and check if any contains the font name
        foreach ($installedFont in $installedFonts.PSObject.Properties.Name) {
            if ($installedFont.ToLower() -like "*$normalizedFontName*") {
                return $true
            }
        }

        return $false
    }

    # Function to download fonts from GitHub
    function Get-Fonts {
        param (
            [string]$fontName,
            [string]$outputPath
        )

        $githubUrl = "https://github.com/google/fonts"
        $fontRepoUrl = "$githubUrl/tree/main/ofl/$fontName"

        # Create output directory if it doesn't exist
        if (-not (Test-Path -Path $outputPath)) {
            New-Item -ItemType Directory -Path $outputPath | Out-Null
        }

        # Fetch font file URLs from GitHub
        $fontFilesPage = Invoke-WebRequest -Uri $fontRepoUrl -UseBasicParsing
        $fontFileLinks = $fontFilesPage.Links | Where-Object { $_.href -match "\.ttf$" -or $_.href -match "\.otf$" }

        foreach ($link in $fontFileLinks) {
            $fileUrl = "https://github.com" + $link.href.Replace("/blob/", "/raw/")
            $fileName = [System.IO.Path]::GetFileName($link.href)

            # Download font file
            Invoke-WebRequest -Uri $fileUrl -OutFile (Join-Path -Path $outputPath -ChildPath $fileName)
        }

        Write-Log "Download complete. Fonts saved to $outputPath"
    }

    # Split fonts list into an array
    $fontsList = $FontsToInstall -split ',' | ForEach-Object { $_.Trim().ToLower() }

    try {
        Write-Log "Installing Google Fonts..." -Level "INFO"

        foreach ($fontName in $fontsList) {
            # Correct the font names for the GitHub repository
            $correctFontName = $fontName -replace "\+", ""

            # Check if the font is already installed
            $isFontInstalled = Test-FontInstalled -FontName $correctFontName

            if ($isFontInstalled) {
                Write-Log "Font $correctFontName is already installed. Skipping Download." -Level "INFO"
                continue
            }

            Write-Log "Downloading & Installing $correctFontName from Google Fonts GitHub repository." -Level "INFO"

            # Download the font files
            Get-Fonts -fontName $correctFontName -outputPath $tempDownloadFolder

            # Install the font files
            $allFonts = Get-ChildItem -Path $tempDownloadFolder -Include *.ttf, *.otf -Recurse
            foreach ($font in $allFonts) {
                $fontDestination = Join-Path -Path $env:windir\Fonts -ChildPath $font.Name
                Copy-Item -Path $font.FullName -Destination $fontDestination -Force

                # Register the font in the windows registry
                New-ItemProperty -Path "HKLM:\Software\Microsoft\Windows NT\CurrentVersion\Fonts" `
                    -Name $font.BaseName `
                    -Value $font.Name `
                    -PropertyType String `
                    -Force
            }

            Write-Log "Font installed: $correctFontName" -Level "INFO"

            # Clean up the downloaded font files
            Remove-Item -Path $tempDownloadFolder -Recurse -Force | Out-Null

        }

        Write-Log "All fonts installed successfully." -Level "INFO"
    }
    catch {
        Write-Log "Error installing fonts: $($_.Exception.Message)" -Level "ERROR"
        Write-ErrorMessage -msg "Error installing fonts"
    }
    
    Write-Host "Success" -ForegroundColor Cyan

}
