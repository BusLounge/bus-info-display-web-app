[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('Apply', 'Revert')]
    [string]$Mode,

    [string]$BackupPath = (Join-Path $env:LOCALAPPDATA 'BusInfoDisplay\kiosk-policy-backup.json')
)

$ErrorActionPreference = 'Stop'

$policyValues = @(
    [pscustomobject]@{
        Path = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Policies\System'
        Name = 'DisableTaskMgr'
        Value = 1
    },
    [pscustomobject]@{
        Path = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Policies\Explorer'
        Name = 'NoWinKeys'
        Value = 1
    },
    [pscustomobject]@{
        Path = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Policies\Explorer'
        Name = 'NoDrives'
        Value = 67108863
    },
    [pscustomobject]@{
        Path = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Policies\Explorer'
        Name = 'NoViewOnDrive'
        Value = 67108863
    }
)

function Get-RegistrySnapshot {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][string]$Name
    )

    $keyExisted = Test-Path -LiteralPath $Path
    $valueExisted = $false
    $value = $null

    if ($keyExisted) {
        $key = Get-Item -LiteralPath $Path
        $valueExisted = $key.GetValueNames() -contains $Name
        if ($valueExisted) {
            $kind = $key.GetValueKind($Name)
            if ($kind -ne [Microsoft.Win32.RegistryValueKind]::DWord) {
                throw "Cannot safely replace $Path\$Name because its existing registry type is $kind, not DWORD."
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

if ($Mode -eq 'Apply') {
    if (Test-Path -LiteralPath $BackupPath) {
        throw "A backup already exists at '$BackupPath'. Revert the previous change first, or specify a different -BackupPath."
    }

    $backupDirectory = Split-Path -Parent $BackupPath
    if (-not (Test-Path -LiteralPath $backupDirectory)) {
        New-Item -ItemType Directory -Path $backupDirectory -Force | Out-Null
    }

    $snapshots = foreach ($entry in $policyValues) {
        Get-RegistrySnapshot -Path $entry.Path -Name $entry.Name
    }

    $backup = [pscustomobject]@{
        Version = 1
        CreatedAtUtc = [DateTime]::UtcNow.ToString('o')
        Values = @($snapshots)
    }
    $backup | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $BackupPath -Encoding UTF8

    foreach ($entry in $policyValues) {
        New-Item -Path $entry.Path -Force | Out-Null
        New-ItemProperty -LiteralPath $entry.Path -Name $entry.Name -Value $entry.Value -PropertyType DWord -Force | Out-Null
    }

    Write-Output "Kiosk restrictions applied for the current user. Backup: $BackupPath"
    Write-Output 'Sign out and back in (or restart) for all Explorer policy changes to take effect.'
    return
}

if (-not (Test-Path -LiteralPath $BackupPath)) {
    throw "No backup found at '$BackupPath'; nothing was changed."
}

$backup = Get-Content -LiteralPath $BackupPath -Raw | ConvertFrom-Json
if ($backup.Version -ne 1 -or $null -eq $backup.Values) {
    throw "The backup file '$BackupPath' is not a supported backup."
}

foreach ($entry in $backup.Values) {
    if ($entry.ValueExisted) {
        New-Item -Path $entry.Path -Force | Out-Null
        New-ItemProperty -LiteralPath $entry.Path -Name $entry.Name -Value ([int]$entry.Value) -PropertyType DWord -Force | Out-Null
    }
    elseif (Test-Path -LiteralPath $entry.Path) {
        Remove-ItemProperty -LiteralPath $entry.Path -Name $entry.Name -ErrorAction SilentlyContinue
    }
}

Remove-Item -LiteralPath $BackupPath -Force
Write-Output 'The original registry values were restored for the current user.'
Write-Output 'Sign out and back in (or restart) for all Explorer policy changes to take effect.'