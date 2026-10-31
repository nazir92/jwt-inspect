$ErrorActionPreference = 'Stop'
if ($env:GITHUB_ACTIONS -ne 'true' -or $env:CI -ne 'true' -or
    $env:RUNNER_ENVIRONMENT -ne 'github-hosted' -or $env:RUNNER_OS -ne 'Windows' -or
    -not $env:GITHUB_WORKSPACE) {
    throw 'This integration test may run only on a GitHub-hosted Windows runner.'
}

$local = [Environment]::GetFolderPath('LocalApplicationData')
$programs = Join-Path $local 'Programs'
$targetDir = Join-Path $programs 'jwt-inspect'
$target = Join-Path $targetDir 'jwt-inspect.exe'
$existingTarget = Get-Item -LiteralPath $targetDir -Force -ErrorAction SilentlyContinue
if ($null -ne $existingTarget) {
    throw "Refusing to run: the installer target already exists: $targetDir"
}
$programsExisted = Test-Path -LiteralPath $programs
$fixture = Join-Path $env:GITHUB_WORKSPACE ('.install-test-' + [guid]::NewGuid().ToString('N'))
if (Test-Path -LiteralPath $fixture) { throw "Fixture already exists: $fixture" }
$fixtureInstaller = Join-Path $fixture 'install.ps1'
$source = Join-Path $fixture 'main.go'
$binary = Join-Path $fixture 'jwt-inspect.exe'

$key = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey('Environment', $false)
$originalPath = $null
$originalKind = $null
if ($key) {
    try {
        $originalPath = $key.GetValue('Path', $null, [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
        if ($null -ne $originalPath) { $originalKind = $key.GetValueKind('Path') }
    } finally { $key.Dispose() }
}
foreach ($entry in @($originalPath -split ';' | Where-Object { $_ -ne '' })) {
    if ($originalKind -eq [Microsoft.Win32.RegistryValueKind]::ExpandString) {
        $entry = [Environment]::ExpandEnvironmentVariables($entry)
    }
    if ([string]::Equals($entry.TrimEnd('\'), $targetDir.TrimEnd('\'), [StringComparison]::OrdinalIgnoreCase)) {
        throw 'Refusing to run: the installer target is already in the user PATH.'
    }
}

function Read-UserPath {
    $key = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey('Environment', $false)
    if (-not $key) { throw 'User Environment registry key is missing.' }
    $attemptedInstall = $false
    try {
        $value = $key.GetValue('Path', $null, [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
        if ($null -eq $value) { throw 'User PATH is missing.' }
        return @{ Value = $value; Kind = $key.GetValueKind('Path') }
    } finally { $key.Dispose() }
}

function Build-Fixture([string]$version) {
    $code = 'package main
import "fmt"
func main() { fmt.Print("installer-fixture-' + $version + '") }
'
    [IO.File]::WriteAllText($source, $code)
    & go build -o $binary $source
    if ($LASTEXITCODE -ne 0) { throw 'Building the Go fixture failed.' }
}

function Assert-NewProcess([string]$version, $path) {
    $userPath = $path.Value
    if ($path.Kind -eq [Microsoft.Win32.RegistryValueKind]::ExpandString) {
        $userPath = [Environment]::ExpandEnvironmentVariables($userPath)
    }
    $machinePath = [Environment]::GetEnvironmentVariable('Path', 'Machine')
    $command = @'
$ErrorActionPreference = 'Stop'
$resolved = Get-Command jwt-inspect -CommandType Application
if ($resolved.Source -ine $env:INSTALL_TEST_EXPECTED) { throw "Unexpected command: $($resolved.Source)" }
& jwt-inspect
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
'@
    $encoded = [Convert]::ToBase64String([Text.Encoding]::Unicode.GetBytes($command))
    $start = [Diagnostics.ProcessStartInfo]::new()
    $start.FileName = Join-Path $PSHOME 'pwsh.exe'
    $start.Arguments = "-NoProfile -NonInteractive -EncodedCommand $encoded"
    $start.UseShellExecute = $false
    $start.RedirectStandardOutput = $true
    $start.RedirectStandardError = $true
    $start.EnvironmentVariables['Path'] = "$machinePath;$userPath"
    $start.EnvironmentVariables['INSTALL_TEST_EXPECTED'] = $target
    $process = [Diagnostics.Process]::Start($start)
    try {
        $output = $process.StandardOutput.ReadToEnd()
        $errors = $process.StandardError.ReadToEnd()
        $process.WaitForExit()
        if ($process.ExitCode -ne 0 -or $output.Trim() -ne "installer-fixture-$version") {
            throw "New process did not run the expected binary: $output $errors"
        }
    } finally { $process.Dispose() }
}

try {
    [IO.Directory]::CreateDirectory($fixture) | Out-Null
    [IO.File]::Copy((Join-Path $env:GITHUB_WORKSPACE 'scripts/install.ps1'), $fixtureInstaller, $false)
    $env:GOPROXY = 'off'
    $env:GOWORK = 'off'
    Build-Fixture 'v1'
    if (Get-Item -LiteralPath $targetDir -Force -ErrorAction SilentlyContinue) {
        throw 'Installer target appeared after the initial safety check.'
    }
    $attemptedInstall = $true
    & $fixtureInstaller

    $installedPath = Read-UserPath
    $expectedKind = if ($null -eq $originalKind) {
        [Microsoft.Win32.RegistryValueKind]::ExpandString
    } else { $originalKind }
    if ($installedPath.Kind -ne $expectedKind) { throw 'User PATH registry type changed.' }
    $expectedPath = if ($originalPath) { "$targetDir;$originalPath" } else { $targetDir }
    if ($installedPath.Value -cne $expectedPath) {
        throw 'User PATH was not preserved with the installation directory prepended.'
    }
    Assert-NewProcess 'v1' $installedPath
    $hash = (Get-FileHash -LiteralPath $target).Hash
    & $fixtureInstaller
    $again = Read-UserPath
    if ($again.Value -cne $installedPath.Value -or $again.Kind -ne $installedPath.Kind -or
        (Get-FileHash -LiteralPath $target).Hash -ne $hash) {
        throw 'Reinstallation was not idempotent.'
    }

    Build-Fixture 'v2'
    $rejected = $false
    try {
        & $fixtureInstaller
    } catch {
        if ($_.Exception.Message -notlike '*Use -Force to replace it*') { throw }
        $rejected = $true
    }
    if (-not $rejected -or (Get-FileHash -LiteralPath $target).Hash -ne $hash) {
        throw 'A conflicting binary was not safely rejected.'
    }
    & $fixtureInstaller -Force
    if ((Get-FileHash -LiteralPath $target).Hash -ne (Get-FileHash -LiteralPath $binary).Hash) {
        throw 'Forced upgrade did not install the new binary.'
    }
    $afterUpgrade = Read-UserPath
    if ($afterUpgrade.Value -cne $installedPath.Value -or $afterUpgrade.Kind -ne $installedPath.Kind) {
        throw 'Upgrade changed the user PATH.'
    }
    Assert-NewProcess 'v2' $afterUpgrade
    Write-Output 'Windows installer integration test passed.'
} finally {
    try {
        $key = [Microsoft.Win32.Registry]::CurrentUser.CreateSubKey('Environment')
        try {
            if ($null -eq $originalPath) {
                $key.DeleteValue('Path', $false)
            } else {
                $key.SetValue('Path', $originalPath, $originalKind)
            }
            $restored = $key.GetValue('Path', $null, [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
            if ($restored -cne $originalPath -or
                ($null -ne $originalPath -and $key.GetValueKind('Path') -ne $originalKind)) {
                throw 'Could not verify restoration of the original user PATH.'
            }
        } finally { $key.Dispose() }
        if ('JwtInspectInstaller.NativeMethods' -as [type]) {
            $result = [UIntPtr]::Zero
            [JwtInspectInstaller.NativeMethods]::SendMessageTimeout(
                [IntPtr]0xffff, 0x001a, [UIntPtr]::Zero, 'Environment', 2, 5000, [ref]$result) | Out-Null
        }
    } finally {
        try {
            if ($attemptedInstall) {
                if (Test-Path -LiteralPath $target) { [IO.File]::Delete($target) }
                if (Test-Path -LiteralPath $targetDir) { [IO.Directory]::Delete($targetDir) }
                if (-not $programsExisted -and (Test-Path -LiteralPath $programs)) {
                    [IO.Directory]::Delete($programs)
                }
            }
        } finally {
            foreach ($file in @($binary, $source, $fixtureInstaller)) {
                if (Test-Path -LiteralPath $file) { [IO.File]::Delete($file) }
            }
            if (Test-Path -LiteralPath $fixture) { [IO.Directory]::Delete($fixture) }
        }
    }
}
