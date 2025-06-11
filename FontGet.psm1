# Get all script files
$Public = @(Get-ChildItem -Path $PSScriptRoot\Public\*.ps1 -ErrorAction SilentlyContinue)
$Private = @(Get-ChildItem -Path $PSScriptRoot\Private\*.ps1 -ErrorAction SilentlyContinue)

$script:fontGetVersion = "0.1.0"

# Dot source all functions
$AllFunctions = $Public + $Private
foreach ($import in $AllFunctions) {
    try {
        Write-Verbose "Importing $($import.FullName)"
        . $import.FullName
    }
    catch {
        Write-Error "Failed to import function $($import.FullName): $_"
    }
}

function Invoke-FontGet {
    [CmdletBinding()]
    param(
        [Parameter(Position = 0, ValueFromRemainingArguments)]
        [string[]]$Arguments
    )

    # Handle empty command or special flags
    if (-not $Arguments -or $Arguments.Count -eq 0 -or $Arguments[0].StartsWith('--')) {
        $flag = if ($Arguments.Count -gt 0) { $Arguments[0].ToLower() } else { '' }
        
        switch ($flag) {
            '--logs' {
                $logPath = Join-Path $env:LOCALAPPDATA 'FontGet'
                if (Test-Path $logPath) {
                    Start-Process "explorer.exe" -ArgumentList $logPath
                } else {
                    Write-Host "Log directory not found: $logPath" -ForegroundColor Yellow
                }
                return
            }
            '--info' {
                Write-Host "FontGet Package Manager v$script:fontGetVersion"
                Write-Host "Copyright (c) Graphixa. All rights reserved.`n"
                
                Write-Host "FontGet Directories"
                Write-Host "".PadRight(100, '-')
                Write-Host "Log File Location                 $env:LOCALAPPDATA\FontGet"
                Write-Host "Temp Font Download Directory      $env:TEMP\GoogleFonts"
                Write-Host "Windows Fonts Directory           $env:windir\Fonts`n"
                
                Write-Host "Links"
                Write-Host "".PadRight(100, '-')
                Write-Host "Homepage                          https://github.com/Graphixa/FontGet"
                Write-Host "PowerShell Gallery                [Coming Soon]"
                Write-Host "License                           https://github.com/Graphixa/FontGet/LICENSE"
                Write-Host "Documentation                     https://github.com/Graphixa/FontGet/README.md"
                return
            }
            default {
                # Show default help
                Write-Host "FontGet Google Fonts Manager v$script:fontGetVersion"
                Write-Host "By @graphixa | MIT License: https://github.com/Graphixa/FontGet/License`n"
                Write-Host "Usage: fontget <command> [options]`n"
                Write-Host "Aliases: gfont, fontget`n"
                Write-Host "Commands:"
                Write-Host "  install        Install Google fonts"
                Write-Host "  uninstall      Remove installed fonts"
                Write-Host "  list           List installed fonts"
                Write-Host "  search         Search available Google fonts`n"
                Write-Host "Options:"
                Write-Host "  --info         Display info about the fontget tool"
                Write-Host "  --logs         Open the logs directory`n"
                Write-Host "For command help, use: fontget <command> --? or fontget <command> --help"
                return
            }
        }
    }


    $Command = $Arguments[0].ToLower()
    # Add command alias mapping
    $Command = switch ($Command) {
        'add'    { 'install' }
        'remove' { 'uninstall' }
        'find'   { 'search' }
        default  { $Command }
    }
    $CommandArgs = if ($Arguments.Count -gt 1) { $Arguments[1..($Arguments.Count-1)] } else { @() }

    # Handle help flags for any command
    $helpFlags = @('-?', '--help', '--?', '/?', '-h', '--h')
    if ($CommandArgs | Where-Object { $helpFlags -contains $_ }) {
        switch ($Command) {
            'install' {
                Write-Host "Install Google Fonts`n"
                Write-Host "Usage: fontget install <font-name> [--options]`n"
                Write-Host "The following aliases are available:"
                Write-Host "  add`n"
                Write-Host "Options:"
                Write-Host "  --force              Force installation even if font exists"
                Write-Host "  --accept-licenses    Accept all licenses for the fonts being installed"
                Write-Host "  --verbose            Show detailed progress output"
                Write-Host "  --help, --?          Show help information`n"
                Write-Host "Examples:"
                Write-Host "  fontget install 'roboto'               # Installs roboto font family"
                Write-Host "  fontget install 'opensans'             # Installs opensans font family"
                Write-Host "  fontget install 'roboto' --force       # Forces reinstallation of roboto"
                Write-Host "  fontget install 'roboto' --verbose     # Shows detailed progress`n"
                return
            }
            'uninstall' {
                Write-Host "Uninstall Fonts`n"
                Write-Host "Usage: fontget uninstall <font-name> [--options]`n"
                Write-Host "The following aliases are available:"
                Write-Host "  remove`n"
                Write-Host "Options:"
                Write-Host "  --verbose     Show detailed progress output"
                Write-Host "  --help, --?   Show help information`n"
                Write-Host "Examples:"
                Write-Host "  fontget uninstall 'roboto'       # Removes roboto font family"
                Write-Host "  fontget uninstall 'opensans'     # Removes opensans font family`n"
                return
            }
            'list' {
                Write-Host "List Installed Fonts`n"
                Write-Host "Usage: fontget list [--options]`n"
                Write-Host "Options:"
                Write-Host "  --google      Show only Google fonts"
                Write-Host "  --other       Show only non-Google fonts"
                Write-Host "  --help, --?   Show help information`n"
                Write-Host "Examples:"
                Write-Host "  fontget list              # Lists all installed fonts"
                Write-Host "  fontget list --google     # Lists installed Google fonts only"
                Write-Host "  fontget list --other      # Lists installed non-Google fonts only`n"
                return
            }
            'search' {
                Write-Host "Search Google Fonts`n"
                Write-Host "Usage: fontget search <keyword> [--options]`n"
                Write-Host "The following aliases are available:"
                Write-Host "  find`n"
                Write-Host "Options:"
                Write-Host "  --help, --?   Show help information`n"
                Write-Host "Examples:"
                Write-Host "  fontget search 'roboto'        # Searches for fonts containing 'roboto'"
                Write-Host "  fontget search 'noto sans'     # Searches for fonts containing 'sans'`n"
                return
            }
            default {
                # Show general help for unknown commands
                Write-Host "FontGet Google Fonts Manager v$script:fontGetVersion"
                Write-Host "By @graphixa | MIT License: https://github.com/Graphixa/FontGet/License`n"
                Write-Host "Usage: fontget <command> [options]`n"
                Write-Host "Commands:"
                Write-Host "  install       Install Google fonts"
                Write-Host "  uninstall     Remove installed fonts"
                Write-Host "  list          List installed fonts"
                Write-Host "  search        Search available Google fonts`n"
                Write-Host
                Write-Host "For command help, use: fontget <command> --? or fontget <command> --help"
                return
            }
        }
    }

    # Handle command with no args
    if ($CommandArgs.Count -eq 0) {
        switch ($Command) {
            'install' {
                Write-Host "Install Google Fonts`n"
                Write-Host "Usage: fontget install <font-name> [--options]`n"
                Write-Host "The following aliases are available:"
                Write-Host "  add`n"
                Write-Host "Options:"
                Write-Host "  --force                 Force installation even if font exists"
                Write-Host "  --accept-licenses       Accept all licenses for the fonts being installed"
                Write-Host "  --verbose               Show detailed progress output"
                Write-Host "  --help, --?             Show help information`n"
                return
            }
            'uninstall' {
                Write-Host "Uninstall Fonts`n"
                Write-Host "Usage: fontget uninstall <font-name> [--options]`n"
                Write-Host "The following aliases are available:"
                Write-Host "  remove`n"
                Write-Host "Options:"
                Write-Host "  --verbose     Show detailed progress output"
                Write-Host "  --help, --?   Show help information`n"
                return
            }
            'list' {
                Show-Fonts
                return
            }
            'search' {
                Write-Host "Search Google Fonts`n"
                Write-Host "Usage: fontget search <keyword> [--options]`n"
                Write-Host "The following aliases are available:"
                Write-Host "  find`n"
                Write-Host "Options:"
                Write-Host "  --help, --?   Show help information`n"
                return
            }
        }
    }

    # Process commands
    switch ($Command) {
        'install' {
            # Check for invalid flags first
            $invalidFlags = $CommandArgs | Where-Object { 
                $_ -like '--*' -and $_ -notin @('--force', '--verbose', '--accept-licenses') 
            }
            if ($invalidFlags) {
                Write-Host "Unrecognized option: $($invalidFlags[0])" -ForegroundColor Yellow
                Write-Host "Valid options: --force, --verbose, --accept-licenses" -ForegroundColor DarkGray
                return
            }

            $fontName = ($CommandArgs | Where-Object { -not $_.StartsWith('-') } | Select-Object -First 1)
            if ($fontName) {
                $params = @{
                    Name = $fontName.Trim('"''')
                    Verbose = ($CommandArgs -contains '--verbose')
                }
                if ($CommandArgs -contains '--force') {
                    $params['Force'] = $true
                }
                if ($CommandArgs -contains '--accept-licenses') {
                    $params['AcceptLicenses'] = $true
                }
                try {
                    Install-GoogleFont @params
                } catch {
                    # Error is already handled by Install-GoogleFont
                    return
                }
            } else {
                Write-Host "Font name is required." -ForegroundColor Yellow
                Write-Host "Usage: fontget install <font-name>" -ForegroundColor DarkGray
            }
        }
        'uninstall' {
            # Check for invalid flags
            $invalidFlags = $CommandArgs | Where-Object { $_ -like '--*' }
            if ($invalidFlags) {
                Write-Host "Unrecognized option: $($invalidFlags[0])" -ForegroundColor Yellow
                Write-Host "This command has no options" -ForegroundColor DarkGray
                return
            }

            $fontName = ($CommandArgs | Where-Object { -not $_.StartsWith('-') } | Select-Object -First 1)
            if ($fontName) {
                $params = @{
                    Name = $fontName.Trim('"''')
                    Verbose = ($CommandArgs -contains '--verbose')
                }
                try {
                    Uninstall-GoogleFont @params
                } catch {
                    Write-Host "No installed font found matching input criteria." -ForegroundColor Yellow
                    Write-Host "Try 'fontget list' to see installed fonts." -ForegroundColor DarkGray
                }
            } else {
                Write-Host "Font name is required." -ForegroundColor Yellow
                Write-Host "Usage: fontget uninstall <font-name>" -ForegroundColor DarkGray
            }
        }
        'list' {
            $params = @{}
            $invalidFlags = $CommandArgs | Where-Object { $_ -like '--*' -and $_ -notin @('--google', '--other') }
            if ($invalidFlags) {
                Write-Host "Unrecognized option: $($invalidFlags[0])" -ForegroundColor Yellow
                Write-Host "Valid options: --google, --other" -ForegroundColor DarkGray
                return
            }
            if ($CommandArgs -contains '--google') { 
                $params['GoogleOnly'] = $true 
            }
            elseif ($CommandArgs -contains '--other') {
                $params['OtherOnly'] = $true
            }
            Show-Fonts @params
        }
        'search' {
            # Check for invalid flags
            $invalidFlags = $CommandArgs | Where-Object { $_ -like '--*' }
            if ($invalidFlags) {
                Write-Host "Unrecognized option: $($invalidFlags[0])" -ForegroundColor Yellow
                Write-Host "This command has no options" -ForegroundColor DarkGray
                return
            }

            $searchTerm = ($CommandArgs | Where-Object { -not $_.StartsWith('-') } | Select-Object -First 1)
            if ($searchTerm) {
                try {
                    Search-GoogleFont -Keyword $searchTerm
                } catch {
                    Write-Host "No fonts found matching search criteria." -ForegroundColor Yellow
                    Write-Host "Try a different search term." -ForegroundColor DarkGray
                }
            } else {
                Write-Host "Search term is required." -ForegroundColor Yellow
                Write-Host "Usage: fontget search <keyword>" -ForegroundColor DarkGray
            }
        }
        default {
            Write-Host "FontGet Google Fonts Manager v$script:fontGetVersion"
            Write-Host "By @graphixa | MIT License: https://github.com/Graphixa/FontGet/License`n"
            Write-Host "Unknown command: $Command" -ForegroundColor Red
            Write-Host
            Write-Host "Available commands:"
            Write-Host "  install    Install Google fonts"
            Write-Host "  uninstall  Remove installed fonts"
            Write-Host "  list       List installed fonts"
            Write-Host "  search     Search available Google fonts"
            Write-Host
            Write-Host "Use fontget <command> --? or fontget <command> --help for more information" -ForegroundColor DarkGray
        }
    }
}

# Create and export the aliases
Set-Alias -Name 'gfont' -Value 'Invoke-FontGet'
Set-Alias -Name 'fontget' -Value 'Invoke-FontGet'

# Export module members
Export-ModuleMember -Function @(
    'Install-GoogleFont',
    'Uninstall-GoogleFont',
    'Show-Fonts',
    'Search-GoogleFont',
    'Invoke-FontGet',
    'Test-FontInstalled',
    'Write-Log'
) -Alias @('gfont', 'fontget') 