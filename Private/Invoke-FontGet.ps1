<#
.SYNOPSIS
    Processes command-line interface commands for the FontGet module.

.DESCRIPTION
    Internal function that handles the CLI interface for the FontGet module.
    Parses and routes commands to the appropriate functions.
    Supports install, uninstall, list, and search operations.

.PARAMETER Arguments
    Array of command-line arguments passed to the function.

.EXAMPLE
    Invoke-FontGet install "Roboto"
    Routes to Install-GoogleFont with the specified parameters.

.EXAMPLE
    Invoke-FontGet list --google
    Routes to Show-Fonts with the Google fonts filter.

.NOTES
    Internal function, not exported.
    Author: Graphixa
    Module: FontGet
    This is the backend for the 'gfont' and 'fontget' aliases.

.LINK
    https://github.com/Graphixa/FontGet
#>

function Invoke-FontGet {
    [CmdletBinding()]
    param(
        [Parameter(Position = 0, ValueFromRemainingArguments)]
        [string[]]$Arguments
    )

    # Handle empty command or help request
    if (-not $Arguments -or $Arguments.Count -eq 0) {
        Write-Debug "No arguments provided, showing general help"
        Write-Host (Get-CommandHelp -Command 'general')
        return
    }

    # Extract command and args
    $Command = $Arguments[0].ToString()
    $CommandArgs = if ($Arguments.Count -gt 1) { $Arguments[1..($Arguments.Count-1)] } else { @() }

    Write-Debug "Command: $Command"
    Write-Debug "CommandArgs: $($CommandArgs -join ', ')"

    # Define valid commands
    $validCommands = @(
        'install', 'add',
        'uninstall', 'remove',
        'search', 'find',
        'list',
        'help'
    )

    # Handle help requests
    $helpFlags = @('-?', '--help', '--?', '/?', '-h', '--h')
    $isHelpRequest = $Command -eq 'help' -or ($CommandArgs | Where-Object { $helpFlags -contains $_ })

    if ($isHelpRequest) {
        if ($Command -eq 'help') {
            # Handle: gfont help <command>
            $helpTopic = if ($CommandArgs.Count -gt 0) { $CommandArgs[0] } else { 'general' }
        } else {
            # Handle: gfont <command> --help
            $helpTopic = $Command
        }
        
        Write-Debug "Showing help for: $helpTopic"
        Write-Host (Get-CommandHelp -Command $helpTopic)
        return
    }

    # Handle invalid commands
    if ($Command -notin $validCommands) {
        Write-Host "FontGet Google Fonts Manager v$fontGetVersion"
        Write-Host "By @graphixa | MIT License: https://github.com/Graphixa/FontGet/License`n"
        Write-Host "Unrecognized command: '$Command'" -ForegroundColor Red
        Write-Host (Get-CommandHelp -Command 'general')
        return
    }

    # Process commands
    switch ($Command.ToLower()) {
        { $_ -in 'install', 'add' } {
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
        { $_ -in 'uninstall', 'remove' } {
            $fontName = ($CommandArgs | Where-Object { -not $_.StartsWith('-') } | Select-Object -First 1)
            if ($fontName) {
                $params = @{
                    Name = $fontName.Trim('"''')
                }
                Uninstall-GoogleFont @params
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
            $searchTerm = $CommandArgs -join " "
            Search-GoogleFont -Keyword $searchTerm
        }
        'help' {
            if ($CommandArgs.Count -gt 0) {
                Write-Host (Get-CommandHelp -Command $CommandArgs[0])
            } else {
                Write-Host (Get-CommandHelp -Command 'general')
            }
        }
        default {
            Write-Host (Get-CommandHelp -Command 'general')
        }
    }
} 