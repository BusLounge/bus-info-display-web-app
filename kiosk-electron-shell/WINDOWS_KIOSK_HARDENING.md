# Windows Kiosk Hardening

This guide is a deployment example for the Windows account that runs the kiosk. It does not change the Windows shell, kill `explorer.exe`, or modify this computer when the file is added to the project.

## What the script changes

Run [windows-kiosk-hardening.ps1](windows-kiosk-hardening.ps1) while signed in as the dedicated kiosk user. Its default scope is that user's `HKCU` registry hive:

- `Policies\System\DisableTaskMgr = 1` disables Task Manager for that user.
- `Policies\Explorer\NoWinKeys = 1` disables Windows-key shortcut combinations, including normal use of `Win+E`.
- `Policies\Explorer\NoDrives` hides drive letters in Explorer.
- `Policies\Explorer\NoViewOnDrive` blocks normal Explorer navigation to drive letters. The example mask `67108863` covers A: through Z:.

These are user-interface restrictions, not a security boundary. They do not prevent access through every alternate program, command line, recovery environment, or another account. `NoDrives` alone only hides drive icons; `NoViewOnDrive` is included as well. The script intentionally does not terminate Explorer because Explorer is also the Windows desktop shell and killing it can leave a user without a desktop or recovery UI.

## Apply

1. Create a dedicated **standard** Windows account for the kiosk. Keep a separate administrator account and recovery credentials.
2. Sign in as the kiosk account. Run the script in that account's PowerShell session; do not use an elevated prompt belonging to a different administrator account, because `HKCU` would then refer to that administrator's profile.
3. From the script's directory, run:

   ```powershell
   .\windows-kiosk-hardening.ps1 -Mode Apply
   ```

   The script saves the original values to `%LOCALAPPDATA%\BusInfoDisplay\kiosk-policy-backup.json` before changing them. Store a protected copy of that backup outside the kiosk account as part of deployment recovery.
4. Sign out and sign back in, or restart, to apply the Explorer restrictions.
5. Verify the kiosk user cannot launch Task Manager, open Explorer with `Win+E`, or browse local drive letters in Explorer. Separately test your authorized maintenance and recovery procedure.

If script execution is blocked, follow your organization's script-signing and execution-policy requirements; do not change the machine-wide execution policy just for this example. A process-scoped policy can be used only where permitted by local policy.

## Revert

Sign in as the same kiosk account and run:

```powershell
.\windows-kiosk-hardening.ps1 -Mode Revert
```

Revert restores the backed-up DWORD values, or removes values that did not exist before Apply. The backup is removed after a successful restore. Sign out and back in or restart. If the account cannot start PowerShell, use a separate administrator/recovery account to restore the kiosk user's registry hive or use your organization's endpoint-management recovery process; do not delete arbitrary Winlogon values.

The script refuses to overwrite an existing backup so repeated runs cannot silently replace the original state. Revert the first run before applying again.

## Group Policy option

For managed PCs, prefer policy deployment to hand-editing registry values. In Group Policy, configure the kiosk user's policy:

- **User Configuration > Administrative Templates > System > Ctrl+Alt+Del Options > Remove Task Manager**: Enabled.
- **User Configuration > Administrative Templates > Windows Components > File Explorer > Turn off Windows+X hotkeys**: Enabled.
- **User Configuration > Administrative Templates > Windows Components > File Explorer > Hide these specified drives in My Computer**: configure the drives to hide.
- **User Configuration > Administrative Templates > Windows Components > File Explorer > Prevent access to drives from My Computer**: configure the drives to block in Explorer.

Use the matching policy names available in the target Windows edition and ADMX templates. Apply policy to the kiosk account or its dedicated OU, then sign out/in and verify with `gpresult /r`.

## Stronger kiosk isolation

For a production kiosk, use Windows **Assigned Access** for supported single-app or restricted-app experiences, or **Shell Launcher** where a desktop application must replace the shell. These are more appropriate than trying to disable Explorer with a registry edit. Validate Windows edition support, app recovery, updates, accessibility, maintenance access, and physical recovery before deployment. Keep the kiosk account non-administrative and use AppLocker or Windows Defender Application Control where appropriate to restrict alternate executables.

Do not replace `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon\Shell` or `Userinit` as a shortcut. Incorrect values can prevent normal logon; shell replacement should be managed through Shell Launcher or an equivalent supported deployment mechanism.