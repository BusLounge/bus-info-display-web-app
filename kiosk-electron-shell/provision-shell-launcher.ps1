[CmdletBinding(SupportsShouldProcess = $true, ConfirmImpact = 'High')]
param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('Apply', 'Revert')]
    [string]$Mode,

    [Parameter(Mandatory = $true)]
    [string]$KioskAccount,

    [string]$AppPath,

    [ValidateSet(0, 1, 2, 3)]
    [uint32]$ExitAction = 0
)

$ErrorActionPreference = 'Stop'
$featureName = 'Client-EmbeddedShellLauncher'
$backupRoot = Join-Path $env:ProgramData 'BusInfoDisplay\ShellLauncher'

$identity = [System.Security.Principal.WindowsIdentity]::GetCurrent()
$principal = [System.Security.Principal.WindowsPrincipal]::new($identity)
if (-not $principal.IsInRole([System.Security.Principal.WindowsBuiltInRole]::Administrator)) {
    throw 'Run this script from an elevated Windows PowerShell session.'
}

$kioskSid = ([System.Security.Principal.NTAccount]$KioskAccount).Translate(
    [System.Security.Principal.SecurityIdentifier]
).Value
$backupPath = Join-Path $backupRoot "$kioskSid.json"

try {
    $feature = Get-WindowsOptionalFeature -Online -FeatureName $featureName -ErrorAction Stop
} catch {
    throw "Shell Launcher feature '$featureName' is unavailable. Verify the target Windows edition supports Shell Launcher."
}

if ($feature.State -ne 'Enabled') {
    if ($Mode -eq 'Apply' -and $feature.State -eq 'Disabled') {
        if ($PSCmdlet.ShouldProcess($featureName, 'Enable Windows optional feature')) {
            Enable-WindowsOptionalFeature -Online -FeatureName $featureName -All -NoRestart | Out-Null
            Write-Warning 'Restart Windows, then rerun this script in Apply mode. Shell Launcher was not assigned yet.'
        }
        return
    }

    throw "Shell Launcher is not enabled (current state: $($feature.State)). Enable it and restart Windows before running $Mode."
}

$shellLauncher = [wmiclass]'\\localhost\root\standardcimv2\embedded:WESL_UserSetting'

function Assert-WmiSuccess {
    param(
        [Parameter(Mandatory = $true)]$Result,
        [Parameter(Mandatory = $true)][string]$Operation
    )

    if ($null -eq $Result -or $Result.ReturnValue -ne 0) {
        $returnValue = if ($null -eq $Result) { 'no result' } else { $Result.ReturnValue }
        throw "$Operation failed (return value: $returnValue)."
    }
}

if ($Mode -eq 'Apply') {
    if ([string]::IsNullOrWhiteSpace($AppPath) -or -not (Test-Path -LiteralPath $AppPath -PathType Leaf)) {
        throw 'Provide -AppPath pointing to the installed Electron application executable.'
    }
    if (Test-Path -LiteralPath $backupPath) {
        throw "A backup already exists at '$backupPath'. Revert that assignment before applying again."
    }

    $appItem = Get-Item -LiteralPath $AppPath
    $resolvedAppPath = $appItem.FullName
    $enabledState = $shellLauncher.IsEnabled()
    $defaultShell = $shellLauncher.GetDefaultShell()
    $customShell = $shellLauncher.GetCustomShell($kioskSid)

    Assert-WmiSuccess -Result $enabledState -Operation 'Read Shell Launcher enabled state'
    Assert-WmiSuccess -Result $defaultShell -Operation 'Read default shell'
    if ($customShell.ReturnValue -ne 0 -and $customShell.ReturnValue -ne 1) {
        Assert-WmiSuccess -Result $customShell -Operation 'Read kiosk account shell assignment'
    }

    New-Item -ItemType Directory -Path $backupRoot -Force | Out-Null
    $backup = [pscustomobject]@{
        Version = 1
        Account = $KioskAccount
        Sid = $kioskSid
        ShellLauncherEnabled = [bool]$enabledState.IsEnabled
        DefaultShell = [string]$defaultShell.Shell
        DefaultAction = [uint32]$defaultShell.DefaultAction
        CustomShellExists = -not [string]::IsNullOrWhiteSpace([string]$customShell.Shell)
        CustomShell = [string]$customShell.Shell
        ShellArgs = [string]$customShell.ShellArgs
        StartDirectory = [string]$customShell.StartDirectory
        CustomDefaultAction = [uint32]$customShell.DefaultAction
    }

    if (-not $PSCmdlet.ShouldProcess($KioskAccount, "Assign $resolvedAppPath as Shell Launcher shell")) {
        return
    }

    $backup | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $backupPath -Encoding UTF8

    try {
        Assert-WmiSuccess -Result ($shellLauncher.SetDefaultShell('explorer.exe', [uint32]0)) -Operation 'Set Explorer as default shell'
        Assert-WmiSuccess -Result ($shellLauncher.SetCustomShell(
            $kioskSid,
            $resolvedAppPath,
            '',
            (Split-Path -Parent $resolvedAppPath),
            $ExitAction
        )) -Operation 'Assign Electron shell to kiosk account'
        Assert-WmiSuccess -Result ($shellLauncher.SetEnabled($true)) -Operation 'Enable Shell Launcher'
    } catch {
        Write-Error "Provisioning failed. Backup retained at '$backupPath'. Use Revert mode after correcting the issue. $_"
        throw
    }

    Write-Output "Shell Launcher configured for $KioskAccount ($kioskSid)."
    Write-Output "Application: $resolvedAppPath"
    Write-Output "Backup: $backupPath"
    Write-Output 'Explorer remains the default shell for accounts without a custom assignment.'
    Write-Output 'Sign out and back in or restart to test the kiosk account shell.'
    return
}

if (-not (Test-Path -LiteralPath $backupPath -PathType Leaf)) {
    throw "No Shell Launcher backup found at '$backupPath'; nothing was changed."
}

$backup = Get-Content -LiteralPath $backupPath -Raw | ConvertFrom-Json
if ($backup.Version -ne 1 -or $backup.Sid -ne $kioskSid) {
    throw 'The backup is invalid or belongs to a different account.'
}

if (-not $PSCmdlet.ShouldProcess($KioskAccount, 'Restore prior Shell Launcher assignment')) {
    return
}

Assert-WmiSuccess -Result ($shellLauncher.RemoveCustomShell($kioskSid)) -Operation 'Remove kiosk shell assignment'
if ($backup.CustomShellExists) {
    Assert-WmiSuccess -Result ($shellLauncher.SetCustomShell(
        $kioskSid,
        $backup.CustomShell,
        $backup.ShellArgs,
        $backup.StartDirectory,
        [uint32]$backup.CustomDefaultAction
    )) -Operation 'Restore previous kiosk shell assignment'
}
Assert-WmiSuccess -Result ($shellLauncher.SetDefaultShell(
    $backup.DefaultShell,
    [uint32]$backup.DefaultAction
)) -Operation 'Restore previous default shell'
Assert-WmiSuccess -Result ($shellLauncher.SetEnabled([bool]$backup.ShellLauncherEnabled)) -Operation 'Restore Shell Launcher enabled state'

Remove-Item -LiteralPath $backupPath -Force
Write-Output "Previous Shell Launcher configuration restored for $KioskAccount."
Write-Output 'The Shell Launcher optional Windows feature remains installed; remove it separately only after verifying no other account or policy uses it.'
Write-Output 'Sign out and back in or restart to verify recovery.'