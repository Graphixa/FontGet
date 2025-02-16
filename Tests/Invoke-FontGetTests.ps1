# Import Pester
Import-Module Pester -Force

try {
    # Setup test environment
    . "$PSScriptRoot\Setup-TestEnvironment.ps1"

    # Run tests
    $config = New-PesterConfiguration
    $config.Run.Path = "$PSScriptRoot\FontGet.Tests.ps1"
    $config.Output.Verbosity = 'Detailed'
    $config.Run.Exit = $true  # Exit with error code on test failure

    $result = Invoke-Pester -Configuration $config
}
finally {
    # Always cleanup, even if tests fail
    . "$PSScriptRoot\Cleanup-TestEnvironment.ps1"
} 