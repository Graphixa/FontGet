@{
    # Script module or binary module file associated with this manifest.
    RootModule = 'FontGet.psm1'

    # Version number of this module.
    ModuleVersion = '0.1.0'

    # Author of this module
    Author = 'Your Name'

    # Description of the functionality provided by this module
    Description = 'A PowerShell module for installing Google Fonts'

    # Minimum version of the PowerShell engine required by this module
    PowerShellVersion = '5.1'

    # Functions to export from this module, for best performance, do not use wildcards and do not delete the entry, use an empty array if there are no functions to export.
    FunctionsToExport = @(
        'Install-GoogleFont',
        'Uninstall-GoogleFont',
        'Show-Fonts',
        'Search-GoogleFont',
        'Invoke-FontGet',
        'Test-FontInstalled',
        'Write-Log'
    )

    # Cmdlets to export from this module, for best performance, do not use wildcards and do not delete the entry, use an empty array if there are no cmdlets to export.
    CmdletsToExport = @()

    # Variables to export from this module
    VariablesToExport = '*'

    # Aliases to export from this module, for best performance, do not use wildcards and do not delete the entry, use an empty array if there are no aliases to export.
    AliasesToExport = @('gfont', 'fontget')

    # Private data to pass to the module specified in RootModule/ModuleToProcess. This may also contain a PSData hashtable with additional module metadata used by PowerShell.
    PrivateData = @{
        PSData = @{
            # Tags applied to this module. These help with module discovery in online galleries.
            Tags = @('Font', 'Google-Fonts', 'Installation')

            # A URL to the license for this module.
            LicenseUri = ''

            # A URL to the main website for this project.
            ProjectUri = ''

            # ReleaseNotes of this module
            ReleaseNotes = 'Initial release'
        }
    }

    # Add a new GUID
    GUID = '00000000-0000-0000-0000-000000000000'
} 