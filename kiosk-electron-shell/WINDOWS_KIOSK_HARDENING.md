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

### Shell Launcher provisioning template

The project includes [provision-shell-launcher.ps1](provision-shell-launcher.ps1) as a per-account Shell Launcher template for the packaged desktop Electron app. It requires an elevated Windows PowerShell session and a Windows edition with the `Client-EmbeddedShellLauncher` optional feature. Review the script and test it on a non-production device before using it on a fleet.

1. Install the signed/approved Electron application at its final stable path. Create a dedicated standard kiosk account and verify that a separate administrator account can sign in normally.
2. If Shell Launcher is not enabled, run Apply once to enable the Windows optional feature, restart, then rerun Apply. Example:

    ```powershell
    .\provision-shell-launcher.ps1 -Mode Apply `
       -KioskAccount "$env:COMPUTERNAME\KioskUser" `
       -AppPath 'C:\Program Files\Bus Info Display\Bus Info Display.exe'
    ```

    The template records the prior default and per-user shell settings in `%ProgramData%\BusInfoDisplay\ShellLauncher\<SID>.json`, sets Explorer as the default for unassigned users, assigns the Electron executable to the kiosk SID, and enables Shell Launcher. The default exit action is `0` (restart the kiosk shell); values `1`, `2`, and `3` mean restart device, shut down, and do nothing. `-WhatIf` previews the shell assignment operation.
3. Sign out/in or restart. Verify the kiosk account launches the app, the administrator still receives Explorer, and the documented recovery login works. Shell Launcher setup does not install the app or configure Assigned Access.
4. To restore the saved shell configuration for that account, run:

    ```powershell
    .\provision-shell-launcher.ps1 -Mode Revert `
       -KioskAccount "$env:COMPUTERNAME\KioskUser"
    ```

    Revert restores the saved account assignment, default shell, and enabled state. It deliberately leaves the optional Windows feature installed; remove that feature separately only after confirming no other account or management policy uses it.

The template configures shell selection and exit behavior only. It does not block `Alt+Tab`, `Alt+F4`, the Windows key, or `Ctrl+Alt+Del`. Use edition-supported Keyboard Filter/MDM or Group Policy for approved shortcut restrictions, scope them to the kiosk account, and test each chord on the target Windows image. Do not claim or configure an application-level replacement for Windows secure attention; preserve an administrator recovery path.

### Disable only unnecessary functions

Do not apply a generic “debloat” or service-disabling script to an industrial kiosk. Inventory actual device needs and disable features using supported policy/MDM, one change at a time. Candidates may include AutoPlay/AutoRun, consumer experiences, unused user accounts, removable-storage access, and remote access services **only when not used for administration**. Retain Windows Defender/endpoint protection, Firewall, Windows Update, event logging, accessibility required by operators, and the management/recovery channel. Validate application dependencies and rollback after every policy change.

### Firmware and physical checklist

- Set a unique BIOS/UEFI setup password and store it in the approved credential vault.
- Restrict boot order to the internal system drive; disable USB, optical-media, PXE/network, and one-time external boot where supported and not required for service. Enable Secure Boot and TPM where supported; escrow and test BitLocker recovery before enabling it.
- Lock the enclosure and control unused physical ports where practical. Firmware controls are vendor-specific and do not prevent every physical attack; document vendor recovery procedures.
- Before deployment, test power-on, app crash/restart, update/reboot, keyboard escape chords, admin sign-in, firmware boot restrictions, and full rollback on the actual hardware. `Ctrl+Alt+Del` is a Windows secure-attention path: verify the intended supported policy behavior, do not promise that Shell Launcher or Electron blocks it.

## Electron and Go startup

The Windows kiosk runtime is started by [main.js](main.js), separately from Windows shell provisioning:

- `npm run dist` runs [build.js](build.js). It builds the Go agent as `bin\agent.exe`, builds Angular into `www`, then packages the Electron app with electron-builder.
- The electron-builder configuration in [package.json](package.json) copies `bin` and `www` into the packaged application's resources directory. In packaged mode, `main.js` resolves these from `process.resourcesPath`; during development, it resolves them relative to the Electron project directory (`__dirname`). Keep these source and destination folder names aligned. Missing or misplaced assets can prevent the Go process or display page from starting after installation.
- During Electron's `ready` lifecycle, the app starts its local Express server when `www` exists, starts the Go agent, and then creates the kiosk window. Packaged mode starts the bundled `process.resourcesPath\bin\agent.exe`. Development mode starts a local `bin\agent.exe` if present, otherwise it runs the Go source with `go run`.
- Packaged configuration is stored under Electron's per-user `app.getPath('userData')`, not beside the installed executable. Ensure the kiosk account can write to its user-data and local-store directories.
- On Electron `before-quit`, the app waits for the Go child process/tree and the Express server to stop. Windows uses `taskkill` with a termination fallback; the app logs backend startup errors and exits.

The current startup launches the Go process before creating the kiosk window, but does not wait for an explicit Go health check before rendering the page. Include cold-boot timing and backend readiness in target-device acceptance tests. `npm run dist` packages the application; it does not install the app, configure Windows sign-in, or enable Shell Launcher.

### Schedule bridge and offline behavior

The kiosk schedule flow prefers the Go agent's local snapshot so it can continue displaying the last synchronized schedule during a network outage:

- The Go agent fetches lounge departures and arrivals from the backend on `scheduleIntervalCron`, writes the response atomically to `storeDir/schedule.json`, and serves that snapshot at `http://127.0.0.1:<localBridgePort>/local/schedule`. The default local bridge port is `3000`; deployments may configure another port.
- Angular's `LocalBridgeService` requests the `schedule` endpoint through Electron IPC. The `bridge:get` handler in [main.js](main.js) first requests `/local/schedule` with a three-second timeout. If the agent endpoint is unavailable or returns an error, Electron reads `schedule.json` relative to the agent config directory and its configured `storeDir`. If neither source is available, IPC rejects the request so Angular can continue to its remote API fallback.
- The kiosk uses a local snapshot when it contains rows for the requested lounge. Otherwise it requests the backend arrivals and departures APIs. [schedule.service.ts](../frontend/src/app/core/services/schedule.service.ts) propagates request errors instead of converting them to successful empty schedules, and it does not emit temporary empty rows while requests are pending. This allows [bids-display.component.ts](../frontend/src/app/features/bids-display/bids-display.component.ts) to retry from local storage without clearing rows already on screen.
- Local snapshots include `updatedAt`. The display warns when a snapshot is more than ten minutes old, or when the timestamp is missing or invalid. The ten-minute threshold assumes the default five-minute agent schedule sync interval; adjust the threshold if a deployment configures a longer interval. A genuinely successful API response with no rows still renders the normal no-schedule state.

### Route road-type normalization

Route segments store road classification in `route_segments.road_type`; this is not a separate lookup table. The frontend and Go route service use uppercase canonical values: `HIGHWAY`, `EXPRESSWAY`, `ARTERIAL`, `COLLECTOR`, `URBAN`, `RURAL`, `LOCAL`, `SERVICE`, and `UNKNOWN`. The Go service trims and uppercases input; legacy ambiguous values such as `mixed` and `suburban`, blank values, and omitted values fall back to `UNKNOWN` rather than being guessed.

Apply [migration 008](../backend/migrations/008_normalize_route_segment_road_types.sql) to backfill existing NULL, blank, lowercase, and unsupported values; set the database default to `UNKNOWN`; enforce `NOT NULL`; and constrain stored values to the canonical set. The Go repository also maps legacy SQL NULLs to `UNKNOWN` while deployments roll out the migration. `UNKNOWN` prevents invalid storage but does not guarantee ETA accuracy; review or refine unclassified segments with the road-type updater and validate its results against actual roads.

### Local admin access

While the Electron kiosk app is running, press **Ctrl+Shift+A** to open the local admin login window. This is wired for both development and the packaged Angular display: the Angular root captures the shortcut and calls the limited `openAdmin()` API exposed by [preload.js](preload.js), which sends an IPC message to [main.js](main.js). Electron main then creates the local login window; the fallback page [index.html](index.html) also handles the chord when the Angular assets are unavailable.

After successful login, the local admin form in [admin.html](admin.html) can update the lounge identity, display mode/layout, language, sync frequency, resolution, and orientation for this kiosk. Maintenance personnel can use it directly on the thin client without opening the centralized Angular management pages.

The preload bridge keeps renderer code from receiving Electron's unrestricted `ipcRenderer`, but that only describes the IPC boundary; it does not make the current login secure. The login check in `main.js` currently uses hard-coded `admin` / `password` credentials. Replace this with production-grade authentication and authorization before deployment, and do not treat the shortcut or login window as a substitute for Windows kiosk account restrictions.

## Target-device boot and provisioning flow

Use this sequence on each industrial thin client. Prefer centrally managed provisioning (such as your organization's MDM/GPO process) for repeatable fleet deployment. The project includes [provision-shell-launcher.ps1](provision-shell-launcher.ps1); the separate registry hardening script above applies per-user Explorer/Task Manager policies.

1. Confirm the target Windows edition supports Shell Launcher and determine the device CPU architecture. Build and test the app for that OS/architecture, then install the packaged application to a stable path. The current NSIS target does not configure a Shell Launcher assignment; point Windows to the installed product executable, not `node_modules\electron\electron.exe` or a development folder.
2. Create a dedicated standard kiosk account and a separate administrator account. Confirm the administrator account can sign in and receives the normal Explorer shell before changing kiosk provisioning.
3. Using Shell Launcher or a supported device-management workflow, assign the installed Electron executable to the kiosk account's SID. Keep `explorer.exe` as the default shell for unassigned accounts, including the maintenance administrator. Do not directly overwrite Winlogon `Shell` or `Userinit` values.
4. Configure the kiosk account's restrictions and application control according to the earlier sections. Apply per-user settings in that account's context, and retain a tested administrator/recovery route.
5. Reboot for an end-to-end test. Expected sequence: Windows signs in to the kiosk account; Shell Launcher starts the installed Electron app; Electron resolves packaged `bin`/`www` from `process.resourcesPath`; the Go agent and local Express server start; then the kiosk display window opens.
6. Test normal exit, crash, power loss, network loss, and update/reboot. Configure Shell Launcher’s exit action for the desired behavior. A restart-shell action relaunches the app after it exits; Electron first performs its process/server cleanup, but that exit action does not open Explorer for the kiosk user. For maintenance, sign out and use the separate administrator account, or use the documented management/recovery procedure.
7. Document rollback before rollout: use the administrator or management plane to remove the kiosk user's custom shell assignment or disable Shell Launcher, then reboot and verify the default Explorer shell. Keep installation media and recovery credentials controlled by the device administrator.

Before production deployment, test the packaged installer at its final install path and verify that `resources\bin\agent.exe`, `resources\www`, the writable user-data directory, and the local-store directory all exist and work under the standard kiosk account. Also replace the hard-coded admin credentials in the application before treating the kiosk as secured.

