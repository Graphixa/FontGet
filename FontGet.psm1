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

function Invoke-GoogleFont {
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
                Write-Host "  fontget install 'Roboto'          # Install Roboto font"
                Write-Host "  fontget install 'Open Sans'       # Install Open Sans font"
                Write-Host "  fontget install 'Roboto' --force  # Force reinstall Roboto font`n"
                Write-Host "Options:"
                Write-Host "  --force    Force installation even if font exists"
                return
            }
            'uninstall' {
                Write-Host "Uninstall Fonts`n"
                Write-Host "Usage: fontget uninstall <font-name>`n"
                Write-Host "Examples:"
                Write-Host "  fontget uninstall 'Roboto'        # Remove Roboto font"
                Write-Host "  fontget uninstall 'Open Sans'     # Remove Open Sans font"
                return
            }
            'list' {
                Write-Host "List Installed Fonts`n"
                Write-Host "Usage: fontget list [--google] [--other]`n"
                Write-Host "Examples:"
                Write-Host "  fontget list                # List all fonts"
                Write-Host "  fontget list --google       # List only Google fonts"
                Write-Host "  fontget list --other        # List only non-Google fonts`n"
                Write-Host "Options:"
                Write-Host "  --google   Show only Google fonts"
                Write-Host "  --other    Show only non-Google fonts"
                return
            }
            'search' {
                Write-Host "Search Google Fonts`n"
                Write-Host "Usage: fontget search <keyword>`n"
                Write-Host "Examples:"
                Write-Host "  fontget search 'Roboto'     # Search for Roboto font"
                Write-Host "  fontget search 'sans'       # Search for fonts containing 'sans'"
                return
            }
            default {
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
                Write-Host "  fontget install 'Roboto'          # Install Roboto font"
                Write-Host "  fontget install 'Open Sans'       # Install Open Sans font"
                Write-Host "  fontget install 'Roboto' --force  # Force reinstall Roboto font`n"
                Write-Host "Options:"
                Write-Host "  --force    Force installation even if font exists"
                return
            }
            'uninstall' {
                Write-Host "Uninstall Fonts`n"
                Write-Host "Usage: fontget uninstall <font-name>`n"
                Write-Host "Examples:"
                Write-Host "  fontget uninstall 'Roboto'        # Remove Roboto font"
                Write-Host "  fontget uninstall 'Open Sans'     # Remove Open Sans font"
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
                Write-Host "  fontget search 'Roboto'     # Search for Roboto font"
                Write-Host "  fontget search 'sans'       # Search for fonts containing 'sans'"
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
                Install-GoogleFont @params
            }
        }
        'uninstall' {
            $fontName = ($CommandArgs | Where-Object { -not $_.StartsWith('-') } | Select-Object -First 1)
            if ($fontName) {
                Uninstall-GoogleFont -Name $fontName.Trim('"''')
            }
        }
        'list' {
            $params = @{}
            if ($CommandArgs -contains '--google') { 
                $params['GoogleOnly'] = $true 
            }
            elseif ($CommandArgs -contains '--other') {
                $params['OtherOnly'] = $true
            }
            Show-Fonts @params
        }
        'search' {
            Search-GoogleFont -Keyword ($CommandArgs -join " ")
        }
        default {
            Write-Host "Unknown command: $Command"
            Write-Host "Use 'fontget help' for usage information"
        }
    }
}

# Create and export the aliases
Set-Alias -Name 'gfont' -Value 'Invoke-GoogleFont'
Set-Alias -Name 'fontget' -Value 'Invoke-GoogleFont'

# Export module members
Export-ModuleMember -Function @(
    'Install-GoogleFont',
    'Uninstall-GoogleFont',
    'Show-Fonts',
    'Search-GoogleFont',
    'Invoke-GoogleFont',
    'Test-FontInstalled',
    'Write-Log'
) -Alias @('gfont', 'fontget') 