function Get-CommandHelp {
    [CmdletBinding()]
    param(
        [Parameter()]
        [string]$Command
    )

    # Get module path from module info
    $modulePath = (Get-Module FontGet).ModuleBase
    $helpPath = Join-Path $modulePath "Help"

    Write-Debug "Module Path: $modulePath"
    Write-Debug "Help Path: $helpPath"

    if (-not (Test-Path $helpPath)) {
        Write-Error "Help directory not found at: $helpPath"
        return "Error: Help files not found. Please ensure the Help directory exists in the module folder."
    }

    $helpFile = Join-Path $helpPath "$($Command.ToLower()).txt"
    if (-not (Test-Path $helpFile)) {
        Write-Debug "Help file not found: $helpFile, falling back to general.txt"
        $helpFile = Join-Path $helpPath "general.txt"
    }

    try {
        Write-Debug "Loading help file: $helpFile"
        $content = Get-Content $helpFile -Raw -ErrorAction Stop
        return $content.Replace('{{version}}', $script:fontGetVersion)
    }
    catch {
        Write-Error "Failed to load help file: $_"
        return "Error: Could not load help content. Please ensure all help files are present."
    }
} 