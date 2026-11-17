[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('Apply', 'Revert')]
    [string]$Mode,

    [Parameter(Mandatory = $true)]
    [string]$KioskAccount,

    [string]$BackupPath
)

$ErrorActionPreference = 'Stop'
$backupPathWasSpecified = -not [string]::IsNullOrWhiteSpace($BackupPath)

$identity = [System.Security.Principal.WindowsIdentity]::GetCurrent()
$principal = [System.Security.Principal.WindowsPrincipal]::new($identity)
if (-not $principal.IsInRole([System.Security.Principal.WindowsBuiltInRole]::Administrator)) {
    throw 'Run this script from an elevated PowerShell session. -KioskAccount identifies the account to change; the current Administrator profile is not modified.'
}

try {
    $accountSid = ([System.Security.Principal.NTAccount]::new($KioskAccount)).Translate(
        [System.Security.Principal.SecurityIdentifier]
    ).Value
} catch {
    throw "Could not resolve kiosk account '$KioskAccount' to a Windows SID. Use DOMAIN\User or COMPUTER\User."
}

function Test-LocalAdministratorsMembership {
    param([Parameter(Mandatory = $true)][string]$Sid)

    $context = $null
    $adminGroup = $null
    try {
        Add-Type -AssemblyName System.DirectoryServices.AccountManagement -ErrorAction Stop
        $context = [System.DirectoryServices.AccountManagement.PrincipalContext]::new(
            [System.DirectoryServices.AccountManagement.ContextType]::Machine
        )
        $adminGroup = [System.DirectoryServices.AccountManagement.GroupPrincipal]::FindByIdentity(
            $context,
            [System.DirectoryServices.AccountManagement.IdentityType]::Sid,
            'S-1-5-32-544'
        )
        if (-not $adminGroup) {
            throw 'The local Administrators group could not be resolved.'
        }

        foreach ($member in $adminGroup.GetMembers($true)) {
            if ($member.Sid -and $member.Sid.Value -eq $Sid) {
                return $true
            }
        }
        return $false
    } catch {
        throw "Unable to verify whether '$KioskAccount' is a local Administrator: $($_.Exception.Message)"
    } finally {
        if ($adminGroup) { $adminGroup.Dispose() }
        if ($context) { $context.Dispose() }
    }
}

if (Test-LocalAdministratorsMembership -Sid $accountSid) {
    throw 'Error: Kiosk hardening cannot be applied to an Administrator account.'
}

$targetProfile = Get-CimInstance -ClassName Win32_UserProfile |
    Where-Object { $_.SID -eq $accountSid } |
    Select-Object -First 1

if (-not $targetProfile -or [string]::IsNullOrWhiteSpace($targetProfile.LocalPath)) {
    throw "No existing Windows profile was found for '$KioskAccount' ($accountSid). Sign in to that account once, then rerun."
}

$userHiveFile = Join-Path $targetProfile.LocalPath 'NTUSER.DAT'
if (-not (Test-Path -LiteralPath $userHiveFile -PathType Leaf)) {
    throw "The kiosk account profile hive was not found: $userHiveFile"
}

if ([string]::IsNullOrWhiteSpace($BackupPath)) {
    $BackupPath = Join-Path $env:ProgramData "BusInfoDisplay\KioskPolicy\$accountSid.json"
}

$policyValues = @(
    [pscustomobject]@{
        Path = 'Software\Microsoft\Windows\CurrentVersion\Policies\System'
        Name = 'DisableTaskMgr'
        Value = 1
    },
    [pscustomobject]@{
        Path = 'Software\Microsoft\Windows\CurrentVersion\Policies\System'
        Name = 'DisableLockWorkstation'
        Value = 1
    },
    [pscustomobject]@{
        Path = 'Software\Microsoft\Windows\CurrentVersion\Policies\System'
        Name = 'DisableRegistryTools'
        Value = 1
    },
    [pscustomobject]@{
        Path = 'Software\Microsoft\Windows\CurrentVersion\Policies\Explorer'
        Name = 'NoWinKeys'
        Value = 1
    },
    [pscustomobject]@{
        Path = 'Software\Microsoft\Windows\CurrentVersion\Policies\Explorer'
        Name = 'NoDrives'
        Value = 67108863
    },
    [pscustomobject]@{
        Path = 'Software\Microsoft\Windows\CurrentVersion\Policies\Explorer'
        Name = 'NoViewOnDrive'
        Value = 67108863
    },
    [pscustomobject]@{
        Path = 'Software\Microsoft\Windows\CurrentVersion\Policies\Explorer'
        Name = 'NoRun'
        Value = 1
    }
)

function Get-RegistrySnapshot {
    param(
        [Parameter(Mandatory = $true)][string]$HiveRoot,
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][string]$Name
    )

    $keyPath = Join-Path $HiveRoot $Path
    $keyExisted = Test-Path -LiteralPath $keyPath
    $valueExisted = $false
    $value = $null

    if ($keyExisted) {
        $key = Get-Item -LiteralPath $keyPath
        $valueExisted = $key.GetValueNames() -contains $Name
        if ($valueExisted) {
            $kind = $key.GetValueKind($Name)
            if ($kind -ne [Microsoft.Win32.RegistryValueKind]::DWord) {
                throw "Cannot safely replace $keyPath\$Name because its existing registry type is $kind, not DWORD."
            }
            $value = [int]$key.GetValue($Name)
        }
    }

    [pscustomobject]@{
        Path = $Path
        Name = $Name
        KeyExisted = $keyExisted
        ValueExisted = $valueExisted
        Value = $value
    }
}

function Restore-RegistryValues {
    param(
        [Parameter(Mandatory = $true)][string]$HiveRoot,
        [Parameter(Mandatory = $true)][object[]]$Values
    )

    foreach ($entry in $Values) {
        $relativePath = [string]$entry.Path
        if ($relativePath -match '^HKCU:\\') {
            $relativePath = $relativePath -replace '^HKCU:\\', ''
        }
        if ($relativePath -notmatch '^Software\\Microsoft\\Windows\\CurrentVersion\\Policies\\(System|Explorer)$') {
            throw "Refusing to restore a value outside the expected kiosk policy paths: $relativePath"
        }

        $keyPath = Join-Path $HiveRoot $relativePath
        if ($entry.ValueExisted) {
            New-Item -Path $keyPath -Force | Out-Null
            New-ItemProperty -LiteralPath $keyPath -Name $entry.Name -Value ([int]$entry.Value) -PropertyType DWord -Force | Out-Null
        } elseif (Test-Path -LiteralPath $keyPath) {
            Remove-ItemProperty -LiteralPath $keyPath -Name $entry.Name -ErrorAction SilentlyContinue
        }
    }
}

$loadedHiveName = $null
$loadedByThisScript = $false

try {
    if (Test-Path -LiteralPath "Registry::HKEY_USERS\$accountSid") {
        $loadedHiveName = $accountSid
    } else {
        $loadedHiveName = "BusInfoDisplayKioskPolicy_$PID"
        & reg.exe load "HKU\$loadedHiveName" $userHiveFile | Out-Null
        if ($LASTEXITCODE -ne 0) {
            throw "Failed to load the kiosk user's registry hive from '$userHiveFile' (reg.exe exit code $LASTEXITCODE). Ensure the user is signed out and the profile is not in use."
        }
        $loadedByThisScript = $true
    }

    $hiveRoot = "Registry::HKEY_USERS\$loadedHiveName"

    if ($Mode -eq 'Apply') {
        if (Test-Path -LiteralPath $BackupPath) {
            throw "A backup already exists at '$BackupPath'. Revert the previous change first, or specify a different -BackupPath."
        }

        $backupDirectory = Split-Path -Parent $BackupPath
        if (-not (Test-Path -LiteralPath $backupDirectory)) {
            New-Item -ItemType Directory -Path $backupDirectory -Force | Out-Null
        }

        $snapshots = foreach ($entry in $policyValues) {
            Get-RegistrySnapshot -HiveRoot $hiveRoot -Path $entry.Path -Name $entry.Name
        }

        $backup = [pscustomobject]@{
            Version = 2
            TargetSID = $accountSid
            TargetAccount = $KioskAccount
            CreatedAtUtc = [DateTime]::UtcNow.ToString('o')
            Values = @($snapshots)
        }
        $backup | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $BackupPath -Encoding UTF8

        foreach ($entry in $policyValues) {
            $keyPath = Join-Path $hiveRoot $entry.Path
            New-Item -Path $keyPath -Force | Out-Null
            New-ItemProperty -LiteralPath $keyPath -Name $entry.Name -Value $entry.Value -PropertyType DWord -Force | Out-Null
        }

        Write-Output "Per-user kiosk restrictions applied to $KioskAccount ($accountSid)."
        Write-Output "Backup: $BackupPath"
        Write-Warning 'Fast User Switching is not changed: HideFastUserSwitching is a computer-wide policy and would also affect the maintenance Administrator.'
        Write-Output 'Sign out and back in (or restart) for policy changes to take effect.'
        return
    }

    if (-not (Test-Path -LiteralPath $BackupPath -PathType Leaf)) {
        throw "No backup found at '$BackupPath'; nothing was changed."
    }

    $backup = Get-Content -LiteralPath $BackupPath -Raw | ConvertFrom-Json
    if ($backup.Version -eq 2) {
        if ($backup.TargetSID -ne $accountSid -or $null -eq $backup.Values) {
            throw "The backup is invalid or belongs to a different account; expected SID $accountSid."
        }
    } elseif ($backup.Version -eq 1 -and $null -ne $backup.Values) {
        if (-not $backupPathWasSpecified) {
            throw 'This legacy backup has no target SID. Pass its exact path with -BackupPath and verify it belongs to the requested kiosk account.'
        }
        Write-Warning 'Restoring a legacy backup without a recorded target SID. Verify the explicit -BackupPath belongs to the requested kiosk account.'
    } else {
        throw "The backup file '$BackupPath' is not a supported backup."
    }

    Restore-RegistryValues -HiveRoot $hiveRoot -Values @($backup.Values)
    Remove-Item -LiteralPath $BackupPath -Force
    Write-Output "The original policy values were restored for $KioskAccount ($accountSid)."
    Write-Output 'Sign out and back in (or restart) for policy changes to take effect.'
} finally {
    if ($loadedByThisScript) {
        & reg.exe unload "HKU\$loadedHiveName" | Out-Null
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Failed to unload the temporary kiosk registry hive HKU\$loadedHiveName."
        }
    }
}
