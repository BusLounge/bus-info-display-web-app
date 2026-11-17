# Quickstart: test-run the system on macOS and Windows

This guide runs the whole system on one computer for testing:

- **Backend** (`backend/`): the Go API and the admin web app.
- **Kiosk** (`kiosk/`): the sync agent and the display, either in a browser or in the Electron kiosk app.

The backend connects to the shared Supabase database configured in `backend/api/.env`. You don't need a local database. Anything you create or edit in the admin app changes that shared database.

| Part | Address |
|---|---|
| API | http://localhost:8083 |
| Admin app | http://localhost:4300 |
| Sync agent (local bridge) | http://localhost:3001 |
| Kiosk display | http://localhost:4200/bids-display |

All commands below are run from the repository root unless a step says otherwise. On Windows, use PowerShell.

---

## 1. Install the prerequisites

| Tool | Version | Needed for |
|---|---|---|
| [Git](https://git-scm.com/downloads) | any | cloning the repo |
| [Go](https://go.dev/dl/) | 1.25 or newer | API (without Docker) and sync agent |
| [Node.js](https://nodejs.org/) | 20.19+, 22.12+ or 24+ | admin app, kiosk display, Electron |
| [Docker Desktop](https://www.docker.com/products/docker-desktop/) | any recent | optional, easiest way to run the backend |

On Windows, Docker Desktop needs WSL 2. The installer offers to set it up.

Check what's installed:

```
go version
node -v
docker version
```

---

## 2. Create the backend settings file

The API reads its settings from `backend/api/.env`. This file holds passwords and secrets, so it isn't in the repo. Create it from the template:

macOS:

```
cp backend/api/.env.example backend/api/.env
```

Windows:

```
Copy-Item backend\api\.env.example backend\api\.env
```

Then fill in the empty values. Ask the team for them:

```
DATABASE_URL=...
JWT_SECRET=...
JWT_REFRESH_SECRET=...
SUPERADMIN_USERNAME=...
SUPERADMIN_PASSWORD=...
```

`SUPERADMIN_USERNAME` and `SUPERADMIN_PASSWORD` are what you log in to the admin app with. Never commit `.env`; it's listed in `.gitignore`.

---

## 3. Start the backend

Use **either** Docker (3A) **or** run it directly (3B), not both: they use the same ports.

### 3A. With Docker (recommended)

Start Docker Desktop and wait until it says it's running. Then, on macOS or Windows:

```
cd backend
docker compose up --build
```

The first build takes a few minutes. When the log shows `Server starting on port 8083`, the API and the admin app are both running.

To stop: press `Ctrl+C`, then run `docker compose down`.

### 3B. Without Docker

Use two terminals.

**Terminal 1: API**

```
cd backend/api
go run ./cmd/server
```

Run it from `backend/api`: the server looks for `.env` and `internal/config/config.json` in the folder it's started from.

**Terminal 2: admin app**

```
cd backend/admin-frontend
npm install
npm start
```

### Check the backend

1. Open http://localhost:8083/health. It should show `OK`.
2. Open http://localhost:4300 and log in with the `SUPERADMIN_*` values from step 2.
3. The dashboard should show trip counts and broadcast messages.

---

## 4. Pick a lounge to display

A kiosk shows one lounge. It only shows rows if that lounge has trips today.

The sync agent's settings are in `kiosk/tv-sync-agent-go/config.json`. Its default lounge (`alohaRavePluse`) has no trips, so its display will be empty. For testing, change these two lines to a lounge that has trips:

```json
"loungeId": "dd971205-6d75-468b-98a9-f540466d3186",
"loungeGroup": "Ceynaturia Longe",
```

`loungeGroup` must be the lounge's exact name; it's used to fetch the lounge's ads.

To check whether a lounge has trips today, open:
`http://localhost:8083/api/arrivals/lounge/<loungeId>`
and look for entries in `"arrivals"`.

---

## Shortcut: run the kiosk with one command

`kiosk/run-kiosk.js` does steps 4 to 7 for you. Set it up in `kiosk/run-kiosk.conf`:

```
os: mac            # mac or windows
mode: electron     # electron = full kiosk app, browser = display in your browser
loungeId: dd971205-6d75-468b-98a9-f540466d3186
loungeName: Ceynaturia Longe
serverUrl: http://localhost:8083
rebuildDisplay: false
openBrowser: true
```

Start the backend (step 3), then on macOS or Windows:

```
cd kiosk
node run-kiosk.js
```

The script:

1. Checks that `os` matches the computer, that Go and npm are installed, and that ports 3001 and 4200 are free.
2. Backs up the files the run will change: the agent's `config.json`, and the agent's downloaded data in both `local-store` folders.
3. Writes the lounge and server from `run-kiosk.conf` into the agent's config. If `bin/agent.exe` exists (it's created by `npm run dist`), it moves it aside for the run.
4. Installs packages if `node_modules` is missing, and builds the kiosk display into `www/` if it hasn't been built yet. Then it starts the kiosk.

Press `Ctrl+C` in the terminal to stop. The script stops every process it started and puts all those files back exactly as they were before the run (not as they are in git, so your own edits are kept).

If the terminal is closed or the script is force-killed, it can't clean up. Stop any leftover processes on ports 3001 and 4200 (see Troubleshooting), then run:

```
node run-kiosk.js --cleanup
```

The next normal run also cleans up leftovers automatically before starting.

The manual steps below do the same thing by hand.

---

## 5. Test the kiosk in a browser (quickest)

This runs the kiosk display as a normal web page. It's the fastest way to check the display and the sync agent. Use two more terminals.

**Terminal 3: sync agent**

```
cd kiosk/tv-sync-agent-go
go run ./cmd/agent
```

You should see `local bridge listening on :3001`. Within about 15 seconds it downloads the schedule, ads and broadcasts into `kiosk/tv-sync-agent-go/local-store/`.

Check http://localhost:3001/local/schedule: it should list the lounge's arrivals and departures.

**Terminal 4: kiosk display**

```
cd kiosk/kiosk-frontend
npm install
npm start
```

Then open (with your lounge's ID and name):

http://localhost:4200/bids-display?loungeId=dd971205-6d75-468b-98a9-f540466d3186&loungeName=Ceynaturia%20Longe&kiosk=true

You should see the lounge name, a live clock, and Departures and Arrivals tables showing five rows per page.

Stop the kiosk display (`Ctrl+C` in terminal 4) before going on to step 6: both use port 4200.

---

## 6. Test the full kiosk app (Electron)

This is what runs on a real kiosk: a full-screen window that starts the sync agent itself.

Stop the sync agent from step 5 first (`Ctrl+C` in terminal 3); Electron starts its own copy.

### 6.1 Build the kiosk display

The Electron app shows a built copy of the display from `kiosk/kiosk-electron-shell/www/`. It's build output, so it isn't in the repo. Build it the first time, and again whenever you change `kiosk/kiosk-frontend`.

macOS:

```
cd kiosk/kiosk-frontend
npm install
npm run build
rm -rf ../kiosk-electron-shell/www
cp -R dist/kiosk-display/browser ../kiosk-electron-shell/www
```

Windows:

```
cd kiosk\kiosk-frontend
npm install
npm run build
if (Test-Path ..\kiosk-electron-shell\www) { Remove-Item -Recurse -Force ..\kiosk-electron-shell\www }
Copy-Item -Recurse dist\kiosk-display\browser ..\kiosk-electron-shell\www
```

If `kiosk/kiosk-electron-shell/bin/agent.exe` exists (it's created by `npm run dist`, see 6.3), delete it before testing. While it's there, the app runs that binary and ignores `kiosk/tv-sync-agent-go/config.json`, using a config that points at a remote server instead.

### 6.2 Start the app

```
cd kiosk/kiosk-electron-shell
npm install
npm start
```

The app takes over the whole screen and hides the mouse cursor. It reads the lounge from `kiosk/tv-sync-agent-go/config.json` and opens the display for it.

- **Admin panel:** press `Ctrl+Shift+A` (or `Ctrl+Shift+Q`) and log in with `admin` / `password`, which are set in `kiosk-electron-shell/main.js`. On a Mac, use the Control key, not Command. The login and settings screens replace the timetable inside the kiosk window; **Back to display** (or the Escape key) returns to it. The panel sets the lounge, display mode, language and resolution, then restarts the agent.
- **Quit:** the window blocks the usual shortcuts (`Cmd+Q`, `Alt+F4`), and on a Mac kiosk mode can also block `Cmd+Tab`. Open the admin panel and click **Exit Kiosk** at the bottom. This closes the app and the sync agent. If you started it with `run-kiosk.js`, the script then cleans up as usual. Pressing `Ctrl+C` in the terminal also works if you can reach it.

### 6.3 Building the Windows installer (Windows only)

```
cd kiosk\kiosk-electron-shell
npm run dist
```

This rebuilds `bin\agent.exe` and `www\` and produces an installer in `dist\`. Run it on Windows: on a Mac, the agent would be built for macOS but named `agent.exe`.

The Windows lockdown scripts are described in `kiosk/kiosk-electron-shell/WINDOWS_KIOSK_HARDENING.md`. Only apply them to a dedicated kiosk machine, never your development computer.

---

## 7. Clean up

Stop every terminal with `Ctrl+C`. If you used Docker, also run `docker compose down` in `backend/`. A running API keeps generating trips in the shared database every midnight.

Files the agent downloads into `local-store/`, and the built `www/` and `bin/` folders, are listed in `.gitignore`, so they never show up as changes.

---

## Troubleshooting

**"Address already in use" or a port is busy.** Something is already running on that port, often an earlier run.

- macOS: `lsof -i :8083` shows the process. Stop it with `kill <PID>`.
- Windows: `netstat -ano | findstr :8083` shows the PID in the last column. Stop it with `taskkill /PID <PID> /F`.

**The kiosk shows "No departures available".** The lounge has no trips today. Check it as described in step 4, or pick another lounge.

**Every row says "Bus TBA" and a large "Delayed … min".** This is expected with the current data: trips have no bus assigned, and trips earlier today are counted as late.

**The API exits with "Failed to read JSON config".** Start it from `backend/api` (step 3B).

**`npm install` fails.** Check `node -v` against the versions in step 1. Then delete `node_modules` and try again.

## Known problems

These are problems in the system, not in your setup:

- **Route Management page** shows "Failed to load routes". The hosted database is missing the `route_segments.traffic_sensitivity_factor` column.
- **Arrival Management and Departure Management pages** are empty. They only show trips that have an active trip record, and current trips don't.
- **ETA engine** (`backend/eta_engine`) can't make predictions: its trained model files aren't in the repo. The API falls back to its own ETA calculation, so ETAs still appear.
- **Ad playback logging** from the kiosk is rejected by the API (HTTP 401), so plays don't reach the ad cost report.
