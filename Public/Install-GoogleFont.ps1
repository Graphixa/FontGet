<#
.SYNOPSIS
    Installs Google Fonts on Windows.

.DESCRIPTION
    The Install-GoogleFont cmdlet downloads and installs fonts from the Google Fonts repository.
    It supports installing fonts for the current user or all users (requires admin rights).
    Fonts are downloaded from Google's GitHub repository and installed in the appropriate Windows font directory.

.PARAMETER Name
    The name of the font(s) to install. Multiple fonts can be specified using a comma-separated list.

.PARAMETER Force
    Forces font installation even if the font is already installed.

.PARAMETER AcceptLicenses
    Accepts all licenses for the fonts being installed.

.EXAMPLE
    Install-GoogleFont "Roboto"
    Installs the Roboto font family for all users.

.EXAMPLE
    Install-GoogleFont "Roboto" -Force
    Forces reinstallation of the Roboto font family even if it's already installed.

.EXAMPLE
    Install-GoogleFont "Roboto" -AcceptLicenses
    Installs the Roboto font family, automatically accepting all licenses.

.NOTES
    Author: Graphixa
    Module: FontGet
    Requires: Windows PowerShell 5.1 or PowerShell Core 7.0+
    Requires: Administrator rights for AllUsers scope

.LINK
    https://github.com/Graphixa/FontGet

.LINK
    https://fonts.google.com/
#>
function Install-GoogleFont {
    [CmdletBinding(SupportsShouldProcess)]
    param (
        [Parameter(Mandatory=$true, Position=0)]
        [string]$Name,

        [Parameter()]
        [switch]$Force,

        [Parameter()]
        [switch]$AcceptLicenses
    )

    begin {
        # Track if user has chosen "yes to all" for licenses
        $script:acceptAllLicenses = $AcceptLicenses.IsPresent
        
        # Setup temp directory for downloads
        $tempDownloadFolder = Join-Path $env:TEMP 'GoogleFonts'
        
        # Create the main temp directory if it doesn't exist
        if (-not (Test-Path $tempDownloadFolder)) {
            New-Item -ItemType Directory -Path $tempDownloadFolder -Force | Out-Null
        }

        # Windows Fonts directory
        $fontFolder = "$env:windir\Fonts"
        $registryPath = "HKLM:\Software\Microsoft\Windows NT\CurrentVersion\Fonts"

        # Check for admin rights
        if (-not ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
            throw "Administrator rights required to install fonts. Please run as administrator."
        }
    }

    process {
        try {
            Write-Log "Starting font installation process" -Level INFO
            # Split font names and process each
            $fontsList = $Name -split ',' | ForEach-Object { $_.Trim() }
            
            foreach ($fontName in $fontsList) {
                Write-Verbose "Processing font: $fontName"

                # Check if font is already installed
                if ((Test-FontInstalled -Name $fontName) -and -not $Force) {
                    Write-Log "Font '$fontName' is already installed" -Level WARNING
                    Write-Warning "Font '$fontName' is already installed. Use --force to reinstall."
                    continue
                }

                if ($PSCmdlet.ShouldProcess($fontName, "Install font")) {
                    Write-Log -Message "Starting installation of $fontName" -Level 'INFO'

                    # Fetch font metadata and handle license acceptance
                    Write-Progress -Activity "Installing Font" -Status "Fetching font information" -PercentComplete 10
                    
                    # Get the metadata URL for this font
                    $metadataUrl = "https://raw.githubusercontent.com/google/fonts/main/ofl/$fontName/METADATA.pb"
                    Write-Verbose "Fetching metadata from: $metadataUrl"
                    
                    try {
                        $metadata = Invoke-RestMethod -Uri $metadataUrl -Headers $headers -Method Get
                        
                        # Extract the official font name from metadata
                        if ($metadata -match 'name: "([^"]+)"') {
                            $officialFontName = $matches[1]
                            $licenseUrl = "https://github.com/google/fonts/blob/main/ofl/$fontName/OFL.txt"
                            
                            Write-Host "`nInstalling font: $officialFontName" -ForegroundColor Cyan
                            Write-Host "License: $licenseUrl" -ForegroundColor DarkGray

                            # Check if we need to prompt for license acceptance
                            if (-not $script:acceptAllLicenses) {
                                $caption = "License Agreement"
                                $message = "Do you accept the font creator's license? View the license at: $licenseUrl"
                                $choices = [System.Management.Automation.Host.ChoiceDescription[]] @(
                                    New-Object System.Management.Automation.Host.ChoiceDescription "&Yes", "Accept license for this font"
                                    New-Object System.Management.Automation.Host.ChoiceDescription "&No", "Skip this font"
                                    New-Object System.Management.Automation.Host.ChoiceDescription "&All", "Accept all licenses"
                                    New-Object System.Management.Automation.Host.ChoiceDescription "&Cancel", "Stop installation"
                                )
                                
                                $choice = $host.UI.PromptForChoice($caption, $message, $choices, 0)
                                
                                switch ($choice) {
                                    0 { 
                                        Write-Verbose "User accepted license for $officialFontName"
                                    }
                                    1 { 
                                        Write-Host "Skipping $officialFontName" -ForegroundColor Yellow
                                        continue
                                    }
                                    2 { 
                                        Write-Verbose "User accepted all licenses"
                                        $script:acceptAllLicenses = $true
                                    }
                                    3 { 
                                        Write-Host "Installation cancelled by user" -ForegroundColor Yellow
                                        return
                                    }
                                }
                            }
                        }
                    }
                    catch {
                        Write-Verbose "Could not fetch metadata for $fontName : $_"
                    }

                    Write-Progress -Activity "Installing Font" -Status "Downloading font files" -PercentComplete 20

                    # Create temp directory for this font
                    $fontTempPath = Join-Path $tempDownloadFolder $fontName
                    if (-not (Test-Path $fontTempPath)) {
                        Write-Verbose "Creating font directory: $fontTempPath"
                        New-Item -ItemType Directory -Path $fontTempPath -Force | Out-Null
                    }

                    # Download font files
                    Write-Progress -Activity "Installing Font" -Status "Downloading $fontName" -PercentComplete 25
                    Write-Verbose "Downloading fonts to: $fontTempPath"
                    
                    try {
                        $downloadedFiles = Get-FontFiles -FontName $fontName -OutputPath $fontTempPath
                        
                        if ($downloadedFiles -and $downloadedFiles.Count -gt 0) {
                            Write-Log "Retrieved $($downloadedFiles.Count) files to install" -Level INFO
                            Write-Progress -Activity "Installing Font" -Status "Installing $fontName" -PercentComplete 75

                            # Install each font file
                            foreach ($fontFile in $downloadedFiles) {
                                if (Test-Path $fontFile) {
                                    $fileName = Split-Path $fontFile -Leaf
                                    $fontDestination = Join-Path -Path $env:windir\Fonts -ChildPath $fileName

                                    Write-Log "Installing font file: $fileName to $fontDestination" -Level INFO

                                    try {
                                        # Try to copy the font file
                                        Copy-Item -Path $fontFile -Destination $fontDestination -Force -ErrorAction Stop

                                        # Register in Windows registry
                                        $registryValue = $fileName
                                        $registryName = [System.IO.Path]::GetFileNameWithoutExtension($fileName)
                                        
                                        # Use .NET method to add font resource
                                        [System.Runtime.InteropServices.Marshal]::GetLastWin32Error()
                                        $null = [System.Runtime.InteropServices.Marshal]::FreeHGlobal((
                                            [System.Runtime.InteropServices.Marshal]::StringToHGlobalUni($fontDestination)
                                        ))

                                        # Add registry entry
                                        New-ItemProperty -Path $registryPath `
                                            -Name $registryName `
                                            -Value $registryValue `
                                            -PropertyType String `
                                            -Force | Out-Null

                                        Write-Log "Added registry entry for: $fileName" -Level INFO
                                        Write-Verbose "Installed: $fileName"
                                    }
                                    catch {
                                        Write-Log "Failed to install font file: $fileName - $_" -Level ERROR
                                        Write-Warning "Failed to install font file: $fileName"
                                        Write-Warning $_.Exception.Message
                                        continue
                                    }
                                } else {
                                    Write-Log "Font file not found: $fontFile" -Level WARNING
                                }
                            }
                        } else {
                            Write-Log "No font files were downloaded for $fontName" -Level WARNING
                            Write-Host "No font found matching input criteria." -ForegroundColor Red
                            Write-Host "Try 'fontget search to find available fonts." -ForegroundColor DarkGray
                            return
                        }
                    }
                    catch {
                        Write-Log "Failed to download font files for $fontName`: $_" -Level ERROR
                        Write-Host "No font found matching input criteria." -ForegroundColor Red
                        Write-Host "Try 'fontget search to find available fonts." -ForegroundColor DarkGray
                        return
                    }

                    Write-Progress -Activity "Installing Font" -Status "Completed" -PercentComplete 100
                    Write-Log -Message "Successfully installed $fontName" -Level 'INFO'

                    # Clean up this font's temp directory
                    if (Test-Path $fontTempPath) {
                        Remove-Item $fontTempPath -Recurse -Force
                    }
                }
            }
        }
        catch {
            Write-Log "Error installing font: $_" -Level ERROR
            throw $_
        }
    }

    end {
        # Cleanup main temp directory
        if (Test-Path $tempDownloadFolder) {
            Remove-Item $tempDownloadFolder -Recurse -Force
        }
    }
} 