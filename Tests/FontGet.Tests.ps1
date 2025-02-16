BeforeAll {
    # Get module path and ensure it exists
    $modulePath = Join-Path (Split-Path -Parent $PSScriptRoot) "FontGet"
    
    # Remove module if loaded
    Get-Module FontGet | Remove-Module -Force -ErrorAction SilentlyContinue
    
    # Import module
    Import-Module $modulePath -Force -ErrorAction Stop

    # Mock filesystem operations
    Mock New-Item { $true } -ModuleName FontGet
    Mock Move-Item { $true } -ModuleName FontGet
    Mock Test-Path { $true } -ModuleName FontGet
}

Describe "FontGet Module" {
    BeforeAll {
        # Setup global mocks that all tests might need
        $mockIdentity = [System.Security.Principal.WindowsIdentity]::GetCurrent()
        $mockPrincipal = [PSCustomObject]@{
            IsInRole = { param($role) $true }
        }
        Mock New-Object -ParameterFilter { 
            $TypeName -eq 'Security.Principal.WindowsPrincipal' -and
            $ArgumentList -and 
            $ArgumentList[0] -eq $mockIdentity
        } -MockWith { $mockPrincipal } -ModuleName FontGet
    }

    Context "Basic Module Functions" {
        It "Should load the module" {
            Get-Module FontGet | Should -Not -BeNull
        }

        It "Should have the gfont alias" {
            Get-Alias gfont | Should -Not -BeNull
        }
    }

    Context "Show-Fonts" {
        BeforeAll {
            # Mock Get-Fonts to return the expected format
            Mock Get-Fonts {
                @(
                    "Arial",
                    "Roboto"
                )
            } -ModuleName FontGet

            # Mock Write-Host to capture output
            Mock Write-Host { } -ModuleName FontGet
        }

        It "Should list unique font families" {
            $result = Show-Fonts -SystemFonts
            $result | Should -Not -BeNullOrEmpty
            $result | Should -Contain "Arial"
            $result | Should -Not -Contain "Arial-Bold"
            
            # Verify Get-Fonts was called with correct parameters
            Should -Invoke Get-Fonts -Times 1 -Exactly -ModuleName FontGet -ParameterFilter {
                $SystemFonts -eq $true
            }
        }
    }

    Context "Search-GoogleFont" {
        BeforeAll {
            Mock Invoke-RestMethod {
                @(
                    @{
                        type = "dir"
                        name = "roboto"
                        html_url = "https://github.com/google/fonts/tree/main/ofl/roboto"
                    }
                )
            } -ModuleName FontGet
            Mock Write-Host { } -ModuleName FontGet
            Mock Write-Error { } -ModuleName FontGet
        }

        It "Should handle special characters in search" {
            { Search-GoogleFont -Keyword "OpenSans" } | Should -Not -Throw
        }

        It "Should handle API errors gracefully" {
            Mock Invoke-RestMethod { throw "API Error" } -ModuleName FontGet
            { Search-GoogleFont -Keyword "Roboto" } | Should -Not -Throw
            Should -Invoke Write-Error -Times 1 -Exactly -ModuleName FontGet
        }
    }

    Context "Install-GoogleFont" {
        BeforeAll {
            # Mock all filesystem operations
            Mock Test-FontInstalled { 
                Write-Verbose "Test-FontInstalled called"
                return $false 
            } -ModuleName FontGet -Verbose

            Mock Copy-Item { 
                Write-Verbose "Copy-Item called with Path: $Path, Destination: $Destination"
                return $true 
            } -ModuleName FontGet -Verbose

            Mock Set-ItemProperty {
                Write-Verbose "Set-ItemProperty called with:"
                Write-Verbose "Path: $Path"
                Write-Verbose "Name: $Name"
                Write-Verbose "Value: $Value"
                return $true
            } -ModuleName FontGet -Verbose

            Mock Test-Path { $true } -ModuleName FontGet
            Mock Write-Log { $true } -ModuleName FontGet
            Mock Move-Item { $true } -ModuleName FontGet
            Mock New-Item { $true } -ModuleName FontGet
            Mock Remove-Item { $true } -ModuleName FontGet
            
            # Mock Get-FontFiles to return expected paths
            Mock Get-FontFiles {
                Write-Verbose "Get-FontFiles called with FontName: $FontName"
                return @(
                    "C:\Temp\GoogleFonts\Roboto\Roboto-Regular.ttf",
                    "C:\Temp\GoogleFonts\Roboto\Roboto-Bold.ttf"
                )
            } -ModuleName FontGet -Verbose
        }

        It "Should install new fonts" {
            $VerbosePreference = 'Continue'
            Write-Verbose "Starting font installation test"
            { Install-GoogleFont -Name "Roboto" -Scope CurrentUser } | Should -Not -Throw
            Write-Verbose "Font installation test completed"
            
            # Verify expected mock calls
            Should -Invoke Copy-Item -Times 2 -Exactly -ModuleName FontGet
            Should -Invoke Set-ItemProperty -Times 2 -Exactly -ModuleName FontGet
            Should -Invoke Test-FontInstalled -Times 1 -Exactly -ModuleName FontGet
        }

        It "Should require admin rights for AllUsers scope" {
            # Override the mock for this test
            $mockPrincipal = [PSCustomObject]@{
                IsInRole = { param($role) $false }
            }
            Mock New-Object -ParameterFilter { 
                $TypeName -eq 'Security.Principal.WindowsPrincipal' 
            } -MockWith { $mockPrincipal } -ModuleName FontGet
            
            { Install-GoogleFont -Name "Roboto" -Scope AllUsers } | Should -Throw
        }

        It "Should handle multiple font names" {
            $VerbosePreference = 'Continue'
            { Install-GoogleFont -Name "Roboto,OpenSans" -Scope CurrentUser } | Should -Not -Throw
            
            # Each font has 2 files, so 4 total copies and registry entries
            Should -Invoke Copy-Item -Times 4 -Exactly -ModuleName FontGet
            Should -Invoke Set-ItemProperty -Times 4 -Exactly -ModuleName FontGet
        }
    }

    Context "Install-GoogleFont Error Handling" {
        BeforeAll {
            Mock Test-FontInstalled { $false } -ModuleName FontGet
            Mock Write-Log { } -ModuleName FontGet
            Mock New-Object -ParameterFilter { 
                $TypeName -eq 'Security.Principal.WindowsPrincipal' 
            } -MockWith { 
                [PSCustomObject]@{
                    IsInRole = { param($role) $true }
                }
            } -ModuleName FontGet

            # Mock Get-Fonts function to return one file per font
            Mock Get-Fonts {
                @(
                    "$env:TEMP\GoogleFonts\$FontName\$FontName-Regular.ttf"
                )
            } -ModuleName FontGet

            # Mock file operations
            Mock Copy-Item { $true } -ModuleName FontGet
            Mock New-ItemProperty { $true } -ModuleName FontGet
            Mock Test-Path { $true } -ModuleName FontGet
            Mock Remove-Item { $true } -ModuleName FontGet -ParameterFilter {
                $Path -like "*.ttf" -or $Path -like "*\GoogleFonts\*"
            }
            Mock New-Item { $true } -ModuleName FontGet
            Mock Write-Host { } -ModuleName FontGet
        }

        It "Should handle invalid font names" {
            { Install-GoogleFont -Name "NonExistentFont123" } | Should -Throw
        }

        It "Should handle network errors" {
            Mock Invoke-WebRequest { throw "Network Error" } -ModuleName FontGet
            { Install-GoogleFont -Name "Roboto" } | Should -Throw
        }
    }

    Context "CLI Interface" {
        BeforeAll {
            # Mock all required functions
            Mock Install-GoogleFont { $true } -ModuleName FontGet
            Mock Show-Fonts { $true } -ModuleName FontGet
            Mock Search-GoogleFont { $true } -ModuleName FontGet
            Mock Uninstall-GoogleFont { $true } -ModuleName FontGet
            Mock Write-Host { } -ModuleName FontGet
            
            $DebugPreference = 'Continue'
        }

        It "Should handle install command" {
            Write-Debug "Starting install command test"
            
            # Call the function directly
            Invoke-GoogleFont install Roboto --scope CurrentUser
            
            # Verify the mock was called with correct parameters
            Should -Invoke Install-GoogleFont -Times 1 -Exactly -ModuleName FontGet -ParameterFilter {
                $Name -eq 'Roboto' -and $Scope -eq 'CurrentUser'
            }
        }

        It "Should handle list command" {
            Write-Debug "Starting list command test"
            
            # Call the function directly
            Invoke-GoogleFont list --system
            
            # Verify the mock was called with correct parameters
            Should -Invoke Show-Fonts -Times 1 -Exactly -ModuleName FontGet -ParameterFilter {
                $SystemFonts -eq $true
            }
        }

        It "Should show help" {
            $output = [System.Collections.ArrayList]::new()
            Mock Write-Host { [void]$output.Add($Object) } -ModuleName FontGet
            
            Invoke-GoogleFont help
            
            $output | Should -Not -BeNullOrEmpty
            $output[0] | Should -Match "FontGet \(gfont\)"
        }
    }

    Context "CLI Interface Edge Cases" {
        BeforeAll {
            Mock Install-GoogleFont { $true } -ModuleName FontGet
            Mock Write-Host { } -ModuleName FontGet
        }

        It "Should handle empty arguments" {
            { Invoke-GoogleFont @() } | Should -Not -Throw
        }

        It "Should handle unknown commands" {
            { Invoke-GoogleFont "unknown-command" } | Should -Not -Throw
        }

        It "Should handle missing required arguments" {
            { Invoke-GoogleFont "install" } | Should -Not -Throw
            { Invoke-GoogleFont "search" } | Should -Not -Throw
        }

        It "Should handle invalid scope values" {
            { Invoke-GoogleFont install "Roboto" --scope "InvalidScope" } | Should -Throw
        }
    }
}

AfterAll {
    # Clean up test artifacts
    $testDirs = @(
        "$env:TEMP\GoogleFonts",
        "$env:LOCALAPPDATA\FontGet"
    )

    foreach ($dir in $testDirs) {
        if (Test-Path $dir) {
          Remove-Item -Path $dir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
} 