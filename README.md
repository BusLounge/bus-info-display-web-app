# Bus Info Display

The code is split by where it runs.

To test-run everything on macOS or Windows, see [docs/QUICKSTART.md](docs/QUICKSTART.md). All documentation is in [docs/](docs/README.md); the ETA logic is described in [docs/ETA_IMPLEMENTATION.md](docs/ETA_IMPLEMENTATION.md).

## `backend/` — runs on the server

| Folder | What it is | Run it |
|---|---|---|
| `api/` | Go API (port 8083), database schema and migrations | `cd backend/api && go run ./cmd/server` |
| `eta_engine/` | Python ETA prediction service the API calls on port 8000 | `cd backend/eta_engine && uvicorn main:app --port 8000` |
| `admin-frontend/` | Angular admin web app (dashboard, ads, routes, lounges) | `cd backend/admin-frontend && npm start` → http://localhost:4300 |
| `tv-sync-agent-frontend/` | Separate Angular device-management dashboard | `cd backend/tv-sync-agent-frontend && npm start` |

### Running the backend with Docker

```
cd backend
docker compose up --build
```

- API: http://localhost:8083 (reads secrets and the database URL from `api/.env`; create it from `api/.env.example`, see docs/QUICKSTART.md)
- Admin app: http://localhost:4300 (nginx; proxies `/api` and `/media` to the API)
- Uploaded media is kept in the `media` Docker volume.
- The ETA engine is optional: `docker compose --profile eta up --build`. It needs the trained
  model files in `eta_engine/`; without it the API uses its built-in ETA fallback.

The kiosk is not containerised; it is installed on the kiosk PC as a desktop app.

## `kiosk/` — runs on the kiosk PC / TV box

| Folder | What it is | Run it |
|---|---|---|
| `kiosk-electron-shell/` | Electron shell: locked-down window, starts the sync agent, Windows hardening scripts | `cd kiosk/kiosk-electron-shell && npm start` |
| `kiosk-frontend/` | Angular app with only the display page (`/bids-display`) | `cd kiosk/kiosk-frontend && npm start` → http://localhost:4200 |
| `tv-sync-agent-go/` | Go agent that syncs schedules, ads and media from the server to local files (port 3001) | `cd kiosk/tv-sync-agent-go && go run ./cmd/agent` |
| `local-store/` | Local data written by the sync agent in development (not committed) | — |

## Building the kiosk installer

```
cd kiosk/kiosk-electron-shell
npm run dist
```

This builds `tv-sync-agent-go` into `bin/agent.exe`, builds `kiosk-frontend` into `www/`, and packages both with Electron.

## Code shared by the two Angular apps

`kiosk-frontend` has its own copies of these files from `admin-frontend/src/app/core/`:
`services/api.service.ts`, `arrival.service.ts`, `departure.service.ts`, `advertisement.service.ts`,
`local-bridge.service.ts`, `loading.service.ts` and `interceptors/loading.interceptor.ts`.
A change to one copy is not picked up by the other.

The admin app's "View schedules" button opens the kiosk display at `kioskDisplayUrl`
(set in `backend/admin-frontend/src/environments/`).
