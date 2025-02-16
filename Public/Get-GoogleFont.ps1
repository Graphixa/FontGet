function Get-GoogleFont {
    [CmdletBinding()]
    param(
        [Parameter(ParameterSetName='System')]
        [switch]$SystemFonts,

        [Parameter(ParameterSetName='User')]
        [switch]$UserFonts,

        [Parameter(ParameterSetName='All')]
        [switch]$AllFonts
    )

    # Helper function to format font list output
    function Format-FontList {
        param($fonts, $source)
        if ($fonts.Count -eq 0) {
            Write-Host "No $source fonts found."
            return
        }
        
        Write-Host "`n$source Fonts:`n" -ForegroundColor Cyan
        $fonts | Format-Table -AutoSize
    }

    switch ($PSCmdlet.ParameterSetName) {
        'System' {
            $systemFonts = Get-ChildItem "$env:windir\Fonts" |
                Where-Object { $_.CreationTime -lt (Get-Item "$env:windir\Fonts").CreationTime } |
                Select-Object @{N='Name';E={$_.BaseName}}, 
                            @{N='Type';E={'System'}},
                            @{N='Format';E={$_.Extension}}
            Format-FontList $systemFonts "System"
        }
        'User' {
            $userFonts = Get-ChildItem "$env:LOCALAPPDATA\Microsoft\Windows\Fonts" -ErrorAction SilentlyContinue |
                Select-Object @{N='Name';E={$_.BaseName}}, 
                            @{N='Type';E={'User'}},
                            @{N='Format';E={$_.Extension}},
                            @{N='Installed';E={$_.CreationTime}}
            Format-FontList $userFonts "User-Installed"
        }
        'All' {
            $allFonts = @()
            $allFonts += (Get-ChildItem "$env:windir\Fonts" |
                Select-Object @{N='Name';E={$_.BaseName}}, 
                            @{N='Type';E={'System'}},
                            @{N='Format';E={$_.Extension}})
            $allFonts += (Get-ChildItem "$env:LOCALAPPDATA\Microsoft\Windows\Fonts" -ErrorAction SilentlyContinue |
                Select-Object @{N='Name';E={$_.BaseName}}, 
                            @{N='Type';E={'User'}},
                            @{N='Format';E={$_.Extension}})
            Format-FontList $allFonts "Installed"
        }
    }
} 