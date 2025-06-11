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

    try {
        Write-Log "Invoking FontGet with arguments: $($Arguments -join ' ')" -Level INFO
        Write-Verbose "Processing command: $($Arguments[0])"

        # Extract command and args
        $Command = $Arguments[0].ToString()
        $CommandArgs = if ($Arguments.Count -gt 1) { $Arguments[1..($Arguments.Count-1)] } else { @() }

        Write-Verbose "Processing command: $Command"
        Write-Verbose "With arguments: $($CommandArgs -join ', ')"

        # Define valid commands
        $validCommands = @(
            'install', 'add',
            'uninstall', 'remove',
            'search', 'find',
            'list'
        )

        # Handle invalid commands
        if ($Command -notin $validCommands) {
            Write-Host "Unrecognized command: '$Command'" -ForegroundColor Red
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
                    if ($CommandArgs -contains '--accept-licenses') {
                        $params['AcceptLicenses'] = $true
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
        }
    }
    catch {
        Write-Log "Error in Invoke-FontGet: $_" -Level ERROR
        throw $_
    }
} 