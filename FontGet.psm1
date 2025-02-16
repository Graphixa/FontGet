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

    # Handle empty command
    if (-not $Arguments -or $Arguments.Count -eq 0) {
        Write-Host "FontGet Google Fonts Manager v$script:fontGetVersion"
        Write-Host "By @graphixa | MIT License: https://github.com/Graphixa/FontGet/License`n"
        Write-Host "Usage: fontget <command> [options]`n"
        Write-Host "Commands:"
        Write-Host "  install    Install Google fonts"
        Write-Host "  uninstall  Remove installed fonts"
        Write-Host "  list       List installed fonts"
        Write-Host "  search     Search available Google fonts"
        Write-Host "  help       Show help information`n"
        Write-Host "For command help, use: fontget help <command>"
        return
    }

    $Command = $Arguments[0].ToLower()
    $CommandArgs = if ($Arguments.Count -gt 1) { $Arguments[1..($Arguments.Count-1)] } else { @() }

    # Handle help flags for any command
    $helpFlags = @('-?', '--help', '--?', '/?', '-h', '--h')
    if ($Command -eq 'help' -or ($CommandArgs | Where-Object { $helpFlags -contains $_ })) {
        $helpTopic = if ($Command -eq 'help') {
            if ($CommandArgs.Count -gt 0) { $CommandArgs[0] } else { 'general' }
        } else {
            $Command
        }

        switch ($helpTopic) {
            'install' {
                Write-Host "Install Google Fonts`n"
                Write-Host "Usage: fontget install <font-name> [--force]`n"
                Write-Host "Examples:"
                Write-Host "  fontget install 'roboto'             # Installs roboto font family"
                Write-Host "  fontget install 'opensans'           # Installs opensans font family"
                Write-Host "  fontget install 'roboto' --force     # Forces reinstallation of roboto`n"
                Write-Host "Options:"
                Write-Host "  --force    Force installation even if font exists"
                return
            }
            'uninstall' {
                Write-Host "Uninstall Fonts`n"
                Write-Host "Usage: fontget uninstall <font-name>`n"
                Write-Host "Examples:"
                Write-Host "  fontget uninstall 'roboto'       # Removes roboto font"
                Write-Host "  fontget uninstall 'opensans'     # Removes opensans font"
                return
            }
            'list' {
                Write-Host "List Installed Fonts`n"
                Write-Host "Usage: fontget list [--google] [--other]`n"
                Write-Host "Examples:"
                Write-Host "  fontget list              # Lists all installed fonts"
                Write-Host "  fontget list --google     # Lists installed Google fonts only"
                Write-Host "  fontget list --other      # Lists installed non-Google fonts only`n"
                Write-Host "Options:"
                Write-Host "  --google   Show only Google fonts"
                Write-Host "  --other    Show only non-Google fonts"
                return
            }
            'search' {
                Write-Host "Search Google Fonts`n"
                Write-Host "Usage: fontget search <keyword>`n"
                Write-Host "Examples:"
                Write-Host "  fontget search 'roboto'     # Searches for fonts that contain 'roboto'"
                Write-Host "  fontget search 'sans'       # Searches for fonts that contain 'sans'"
                return
            }
            default {
                Write-Host "FontGet Google Fonts Manager v$script:fontGetVersion"
                Write-Host "By @graphixa | MIT License: https://github.com/Graphixa/FontGet/License`n"
                Write-Host "Usage: fontget <command> [options]`n"
                Write-Host "Commands:"
                Write-Host "  install       Install Google fonts"
                Write-Host "  uninstall     Remove installed fonts"
                Write-Host "  list          List installed fonts"
                Write-Host "  search        Search available Google fonts"
                Write-Host "  help          Show help information`n"
                Write-Host
                Write-Host "For command help, use: fontget help <command>"
                Write-Host "                  or: fontget <command> --help"
                return
            }
        }
    }

    # Handle command with no args
    if ($CommandArgs.Count -eq 0) {
        switch ($Command) {
            'install' {
                Write-Host "Install Google Fonts`n"
                Write-Host "Usage: fontget install <font-name> [--force]`n"
                Write-Host "Examples:"
                Write-Host "  fontget install 'roboto'             # Installs roboto font family on the system"
                Write-Host "  fontget install 'opensans'           # Installs opensans font family on the system"
                Write-Host "  fontget install 'roboto' --force     # Force reinstall roboto font on the system`n"
                Write-Host "Options:"
                Write-Host "  --force    Force installation even if font exists"
                return
            }
            'uninstall' {
                Write-Host "Uninstall Fonts`n"
                Write-Host "Usage: fontget uninstall <font-name>`n"
                Write-Host "Examples:"
                Write-Host "  fontget uninstall 'roboto'       # Removes roboto font family from the system"
                Write-Host "  fontget uninstall 'opensans'     # Removes opensans font family from the system"
                return
            }
            'list' {
                Show-Fonts
                return
            }
            'search' {
                Write-Host "Search Google Fonts`n"
                Write-Host "Usage: fontget search <keyword>`n"
                Write-Host "Examples:"
                Write-Host "  fontget search 'roboto'     # Searches for fonts that contain 'roboto'"
                Write-Host "  fontget search 'sans'       # Searches for fonts that contain 'sans'"
                return
            }
        }
    }

    # Process commands
    switch ($Command) {
        'install' {
            $fontName = ($CommandArgs | Where-Object { -not $_.StartsWith('-') } | Select-Object -First 1)
            if ($fontName) {
                $params = @{
                    Name = $fontName.Trim('"''')
                }
                if ($CommandArgs -contains '--force') {
                    $params['Force'] = $true
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
            $fontName = ($CommandArgs | Where-Object { -not $_.StartsWith('-') } | Select-Object -First 1)
            if ($fontName) {
                try {
                    Uninstall-GoogleFont -Name $fontName.Trim('"''')
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
            Write-Host "  help       Show help information"
            Write-Host
            Write-Host "Use 'fontget help' for usage information" -ForegroundColor DarkGray
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