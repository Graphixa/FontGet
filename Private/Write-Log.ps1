<#
.SYNOPSIS
    Writes log messages for the FontGet module.

.DESCRIPTION
    Internal function that handles logging for the FontGet module.
    Supports different log levels and formats messages consistently.
    Logs include timestamps and can be written to both console and log file.

.PARAMETER Message
    The message to log.

.PARAMETER Level
    The severity level of the log message.
    Valid values: 'INFO', 'WARNING', 'ERROR'. Default is 'INFO'.

.PARAMETER LogPath
    Optional. The path to the log file. If not specified, only writes to console.

.EXAMPLE
    Write-Log -Message "Installing font" -Level INFO
    Logs an informational message.

.EXAMPLE
    Write-Log "Font not found" -Level ERROR
    Logs an error message.

.NOTES
    Internal function, not exported.
    Author: Graphixa
    Module: FontGet

.LINK
    https://github.com/Graphixa/FontGet
#>

function Write-Log {
    [CmdletBinding()]
    param (
        [Parameter(Mandatory=$true)]
        [string]$Message,

        [Parameter()]
        [ValidateSet('INFO', 'WARNING', 'ERROR')]
        [string]$Level = 'INFO'
    )

    $logPath = "$env:HOMEPATH\AppData\Local\FontGet"
    $logFile = "FontGet.log"
    $logFilePath = Join-Path -Path $logPath -ChildPath $logFile

    # Create log directory if it doesn't exist
    if (-not (Test-Path -Path $logPath)) {
        New-Item -ItemType Directory -Path $logPath | Out-Null
        Write-Verbose "Created log directory: $logPath"
    }

    # Check if log file exists for size check
    if (Test-Path -Path $logFilePath) {
        # Check Log file size and rotate if greater than 10MB
        if ((Get-Item -Path $logFilePath).Length -gt 10485760) {
            Move-Item -Path $logFilePath -Destination "$logFilePath.old" -Force
            Write-Verbose "Rotated log file due to size"
        }
    }

    # Format the log message
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $logMessage = "[$timestamp] [$Level] $Message"

    # Write to log file
    Add-Content -Path $logFilePath -Value $logMessage
    
    # Output to verbose stream if verbose is enabled
    Write-Verbose $logMessage
} 