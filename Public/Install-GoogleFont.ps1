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

.EXAMPLE
    Install-GoogleFont "Roboto"
    Installs the Roboto font family for all users.

.EXAMPLE
    Install-GoogleFont "Roboto" -Force
    Forces reinstallation of the Roboto font family even if it's already installed.

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
        [switch]$Force
    )

    begin {
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

                                    # Copy font file to Windows Fonts directory
                                    Copy-Item -Path $fontFile -Destination $fontDestination -Force

                                    # Register the font in the Windows registry
                                    New-ItemProperty -Path $registryPath `
                                        -Name $([System.IO.Path]::GetFileNameWithoutExtension($fileName)) `
                                        -Value $fileName `
                                        -PropertyType String `
                                        -Force | Out-Null

                                    Write-Log "Added registry entry for: $fileName" -Level INFO
                                    Write-Verbose "Installed: $fileName"
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