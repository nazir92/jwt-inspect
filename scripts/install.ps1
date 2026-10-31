param([switch]$Force, [switch]$Help)

$ErrorActionPreference = 'Stop'
if ($Help) {
    Write-Output 'Usage: .\install.ps1 [-Force]'
    Write-Output 'Use -Force to replace an existing, different user installation.'
    return
}

if ([Environment]::OSVersion.Platform -ne [PlatformID]::Win32NT) {
    throw 'This installer requires Windows PowerShell on Windows.'
}

$source = Join-Path $PSScriptRoot 'jwt-inspect.exe'
if (-not [IO.File]::Exists($source) -and (Split-Path $PSScriptRoot -Leaf) -eq 'scripts') {
    $source = Join-Path (Split-Path $PSScriptRoot -Parent) 'jwt-inspect.exe'
}
if (-not [IO.File]::Exists($source)) {
    throw 'jwt-inspect.exe was not found next to the installer or in its archive root.'
}
$sourceItem = Get-Item -LiteralPath $source -Force
if ($sourceItem.PSIsContainer -or ($sourceItem.Attributes -band [IO.FileAttributes]::ReparsePoint)) {
    throw "Source is not a regular file: $source"
}

$localAppData = [Environment]::GetFolderPath('LocalApplicationData')
if (-not $localAppData -or -not [IO.Path]::IsPathRooted($localAppData)) {
    throw 'Cannot determine the current user LocalAppData directory.'
}
$dir = Join-Path $localAppData 'Programs\jwt-inspect'
if ($dir.IndexOfAny([char[]]";%`r`n") -ge 0) {
    throw 'The installation path contains characters that cannot safely be added to PATH.'
}

$environmentKey = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey('Environment', $false)
$userPath = $null
$pathKind = [Microsoft.Win32.RegistryValueKind]::ExpandString
if ($environmentKey) {
    try {
        $userPath = $environmentKey.GetValue('Path', $null, [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
        if ($null -ne $userPath) {
            $pathKind = $environmentKey.GetValueKind('Path')
            if ($pathKind -ne [Microsoft.Win32.RegistryValueKind]::String -and
                $pathKind -ne [Microsoft.Win32.RegistryValueKind]::ExpandString) {
                throw 'The user PATH registry value is not a string.'
            }
        }
    } finally {
        $environmentKey.Dispose()
    }
}
$entries = @($userPath -split ';' | Where-Object { $_ -ne '' })
$present = $false
foreach ($entry in $entries) {
    $expanded = if ($pathKind -eq [Microsoft.Win32.RegistryValueKind]::ExpandString) {
        [Environment]::ExpandEnvironmentVariables($entry).TrimEnd('\')
    } else {
        $entry.TrimEnd('\')
    }
    if ([string]::Equals($expanded, $dir.TrimEnd('\'), [StringComparison]::OrdinalIgnoreCase)) {
        $present = $true
    }
}
$newPath = if ($userPath) { "$dir;$userPath" } else { $dir }
if (-not $present -and $newPath.Length -ge 32767) {
    throw 'The user PATH is too long to add the installation directory.'
}

foreach ($part in @($localAppData, (Join-Path $localAppData 'Programs'), $dir)) {
    if (-not (Test-Path -LiteralPath $part)) {
        New-Item -ItemType Directory -Path $part | Out-Null
    }
    $item = Get-Item -LiteralPath $part -Force
    if (-not $item.PSIsContainer -or ($item.Attributes -band [IO.FileAttributes]::ReparsePoint)) {
        throw "Refusing non-directory or reparse point: $part"
    }
}

$dest = Join-Path $dir 'jwt-inspect.exe'
if (Test-Path -LiteralPath $dest) {
    $item = Get-Item -LiteralPath $dest -Force
    if ($item.PSIsContainer -or ($item.Attributes -band [IO.FileAttributes]::ReparsePoint)) {
        throw "Destination is not a regular file: $dest"
    }
    $oldHash = (Get-FileHash -LiteralPath $dest -Algorithm SHA256).Hash
    if ((Get-FileHash -LiteralPath $source -Algorithm SHA256).Hash -ne $oldHash) {
        if (-not $Force) {
            throw "Destination already exists and differs: $dest. Use -Force to replace it."
        }
        $stage = Join-Path $dir ('.jwt-inspect-' + [guid]::NewGuid().ToString('N') + '.tmp')
        $backup = Join-Path $dir ('.jwt-inspect-' + [guid]::NewGuid().ToString('N') + '.bak')
        try {
            [IO.File]::Copy($source, $stage, $false)
            # Move the old directory entry, rather than opening it through a possible link.
            [IO.File]::Move($dest, $backup)
            $oldItem = Get-Item -LiteralPath $backup -Force
            if (($oldItem.Attributes -band [IO.FileAttributes]::ReparsePoint) -or
                (Get-FileHash -LiteralPath $backup -Algorithm SHA256).Hash -ne $oldHash) {
                throw 'The destination changed during installation; the upgrade was not applied.'
            }
            [IO.File]::Move($stage, $dest)
            [IO.File]::Delete($backup)
        } finally {
            if ([IO.File]::Exists($stage)) { [IO.File]::Delete($stage) }
            if (Test-Path -LiteralPath $backup) {
                if (-not (Test-Path -LiteralPath $dest)) {
                    [IO.File]::Move($backup, $dest)
                } else {
                    Write-Warning "The previous installation was preserved at $backup."
                }
            }
        }
    }
} else {
    [IO.File]::Copy($source, $dest, $false)
}

if (-not $present) {
    $environmentKey = [Microsoft.Win32.Registry]::CurrentUser.CreateSubKey('Environment')
    try {
        $currentPath = $environmentKey.GetValue('Path', $null, [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
        if ($currentPath -cne $userPath -or
            ($null -ne $currentPath -and $environmentKey.GetValueKind('Path') -ne $pathKind)) {
            throw 'The user PATH changed during installation; rerun the installer.'
        }
        $environmentKey.SetValue('Path', $newPath, $pathKind)
        $checkPath = $environmentKey.GetValue('Path', $null, [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
        if ($checkPath -cne $newPath -or $environmentKey.GetValueKind('Path') -ne $pathKind) {
            throw 'Could not verify the persistent user PATH. Check it manually and rerun.'
        }
    } finally {
        $environmentKey.Dispose()
    }

    if (-not ('JwtInspectInstaller.NativeMethods' -as [type])) {
        Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
namespace JwtInspectInstaller {
    public static class NativeMethods {
        [DllImport("user32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
        public static extern IntPtr SendMessageTimeout(IntPtr window, uint message, UIntPtr wParam,
            string lParam, uint flags, uint timeout, out UIntPtr result);
    }
}
'@
    }
    $result = [UIntPtr]::Zero
    $sent = [JwtInspectInstaller.NativeMethods]::SendMessageTimeout(
        [IntPtr]0xffff, 0x001a, [UIntPtr]::Zero, 'Environment', 2, 5000, [ref]$result)
    if ($sent -eq [IntPtr]::Zero) {
        throw 'The user PATH was saved, but Windows could not be notified. Sign out and back in to apply it.'
    }
    Write-Output "Updated the user PATH with $dir"
}
Write-Output "Installed $dest"
Write-Output 'Open a new terminal for the PATH change to take effect. Existing system PATH entries take precedence.'
