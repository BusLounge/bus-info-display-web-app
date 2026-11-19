# Clutter to remove

A list of UI, code and files in this repository that are unused, broken, or left over from earlier work. Nothing listed here has been removed yet.

Each item says how it was checked and what to do with it. Items marked **Decide first** may be planned features rather than abandoned ones, so confirm with the team before deleting them.

Surveyed: 2026-11-18, on branch `Working-Branch_Bimsara`.

## Summary

| # | Item | Where | Status | Action |
|---|------|-------|--------|--------|
| 1 | TV Sync Agent admin page | `backend/admin-frontend` | Mostly broken, not in the menu | Decide first |
| 2 | `tv-sync-agent-frontend` app | `backend/tv-sync-agent-frontend` | Not deployed, calls missing API routes | Decide first |
| 3 | Kiosk loading interceptor and service | `kiosk/kiosk-frontend` | Never runs | Safe to remove |
| 4 | Unused admin service, component and barrel file | `backend/admin-frontend` | Nothing imports them | Safe to remove |
| 5 | Admin calls to the kiosk's local bridge | `backend/admin-frontend` | Points at `localhost:3001` | Investigate |
| 6 | Unused `status` / `indicator` row fields | `kiosk/kiosk-frontend` | Calculated, never displayed | Safe to remove |
| 7 | One-off scripts and ad-hoc SQL | `backend/api` | Not part of the build or migrations | Check with the team |

---

## 1. TV Sync Agent page in the admin website: Decide first

**Files**

- [backend/admin-frontend/src/app/features/tv-sync-agent/](../backend/admin-frontend/src/app/features/tv-sync-agent/) (component, template, styles)
- [backend/admin-frontend/src/app/core/services/tv-sync-agent.service.ts](../backend/admin-frontend/src/app/core/services/tv-sync-agent.service.ts)
- [backend/admin-frontend/src/app/core/models/tv-sync-agent.model.ts](../backend/admin-frontend/src/app/core/models/tv-sync-agent.model.ts)
- The `tv-sync-agent` route in [app.routes.ts](../backend/admin-frontend/src/app/app.routes.ts)

**Problem**

- It isn't in the header menu ([header.component.html](../backend/admin-frontend/src/app/shared/components/header/header.component.html)). You can only reach it by typing `/tv-sync-agent` in the address bar.
- Its service calls these endpoints, and **none of them exist in the Go API** ([main.go](../backend/api/cmd/server/main.go)), so every call fails:
  - `/tv-sync/config`, `/tv-sync/config/{loungeId}`
  - `/tv-sync/device/{deviceId}/status`, `/command`, `/screenshot`
  - `/tv-sync/emergency/{loungeId}`
  - `/tv-sync/schedules/{loungeId}`, `/tv-sync/ads/{loungeId}`
  - `/tv-sync/sync/{loungeId}`, `/tv-sync/validate/{loungeId}`
  - `/tv-sync/tracking/playback`, `/tv-sync/billing/report`
- The only parts that work are the lounge-ad calls (`/lounge-ads`, `/lounge-ads/{id}`, `/lounge-ads/lounge/{loungeId}`, `/lounge-ads/slots/{loungeId}`), which are real API routes.

**Options**

- **Abandoned:** delete the folder, service, model and route. If lounge ads still need managing from the admin website, move that part into the Lounge page first.
- **Planned:** the API needs the `/tv-sync/...` routes before this page can work. Until then, leave it out of the menu.

## 2. `backend/tv-sync-agent-frontend`: Decide first

**Files:** the whole [backend/tv-sync-agent-frontend/](../backend/tv-sync-agent-frontend/) folder (35 tracked files). It's a separate Angular app with one dashboard page.

**Problem**

- It isn't in [docker-compose.yml](../backend/docker-compose.yml) or the Railway deployment ([DEPLOY_RAILWAY.md](DEPLOY_RAILWAY.md)). The only references to it are in the READMEs.
- Its device features call `/devices`, `/devices/{id}/refresh`, `/devices/{id}/emergency-message` and `/devices/{id}/billing-report`. **These routes don't exist in the Go API.**
- Everything else it does (language, display mode, layout, lounge ads) talks to the sync agent on `localhost:3001`. The kiosk's built-in admin panel (Ctrl+Shift+A, in `kiosk-electron-shell`) already covers those settings.
- Its production config still has a placeholder address (`wss://your-domain.com/ws`), so it has never been deployed.

**Options**

- **Abandoned:** delete the folder and remove it from [README.md](../README.md) and [docs/README.md](README.md).
- **Planned:** it needs the `/devices` API routes and a real production config.

## 3. Kiosk loading interceptor and service: Safe to remove

**Files**

- [kiosk/kiosk-frontend/src/app/core/interceptors/loading.interceptor.ts](../kiosk/kiosk-frontend/src/app/core/interceptors/loading.interceptor.ts)
- [kiosk/kiosk-frontend/src/app/core/services/loading.service.ts](../kiosk/kiosk-frontend/src/app/core/services/loading.service.ts)

**Problem:** the interceptor is never registered. [app.config.ts](../kiosk/kiosk-frontend/src/app/app.config.ts) calls `provideHttpClient(withFetch())` with no interceptors. The loading service is only used by that interceptor, so neither one ever runs.

**Action:** delete both files.

## 4. Unused admin service, component and barrel file: Safe to remove

| File | Problem |
|------|---------|
| [core/services/theme.service.ts](../backend/admin-frontend/src/app/core/services/theme.service.ts) | Only exported from `core/services/index.ts`; nothing uses it |
| [shared/components/loading-spinner/](../backend/admin-frontend/src/app/shared/components/loading-spinner/) | Only exported from `shared/components/index.ts`; no page uses `<app-loading-spinner>` |
| [features/index.ts](../backend/admin-frontend/src/app/features/index.ts) | Barrel file that nothing imports; the routes import each component directly |

**Action:** delete the files and remove their lines from the matching `index.ts` files.

## 5. Admin website calling the kiosk's local bridge: Investigate

**File:** [backend/admin-frontend/src/app/core/services/local-bridge.service.ts](../backend/admin-frontend/src/app/core/services/local-bridge.service.ts), used by `advertisement.service.ts` and `dashboard.service.ts`.

**Problem:** it calls `http://localhost:3001/local`, which is the sync agent running **on a kiosk machine**. A browser using the admin website (for example the Railway deployment) won't have anything on that port, so those calls fail. This service looks like it was copied from the kiosk app.

**Action:** check what the ads and dashboard pages use it for. If it's a fallback for when the API is offline, it can probably be removed from the admin website.

## 6. Unused `status` / `indicator` fields on kiosk schedule rows: Safe to remove

**Files**

- [kiosk/kiosk-frontend/src/app/core/services/schedule.service.ts](../kiosk/kiosk-frontend/src/app/core/services/schedule.service.ts): `getStatusClass()`, `hasIndicator()`, and the `status` / `indicator` fields on `DisplayDepartureRow` and `DisplayArrivalRow`
- [kiosk/kiosk-frontend/src/app/features/bids-display/bids-display.component.ts](../kiosk/kiosk-frontend/src/app/features/bids-display/bids-display.component.ts): the same two methods and fields in `applyLocalScheduleSnapshot()`

**Problem:** these drove the old kiosk UI's remark colours and dot indicator. The redesigned UI colours remarks with `remarkTone()` and no longer reads `status` or `indicator`.

**Action:** remove the methods and fields.

## 7. One-off scripts and ad-hoc SQL in `backend/api`: Check with the team

**Files**

- `Tools/db_fix/db_fix.go`, `Tools/db_fix2/db_fix2.go`: one-off database fix scripts
- `Tools/eta_sample/SAMPLE_ETA_SERVICE.go`: sample code
- `database_s.sql`: a second schema file next to `database_schema.sql`
- `check_offset_data.sql`, `diagnostic_queries.sql`, `run_migration.sql`, `update_gps_test_data.sql`: ad-hoc SQL

**Problem:** none of these are part of the API build or the numbered `migrations/` folder. They make it unclear which schema file is the source of truth.

**Action:** ask whether anyone still runs them by hand. If not, delete them, or move any that are worth keeping into a `backend/api/scripts/` folder with a short note on what each one does.

---

## How this was checked

- **Routes and menu:** compared the admin [app.routes.ts](../backend/admin-frontend/src/app/app.routes.ts) against the links in the header component.
- **API calls:** listed every endpoint each frontend service calls and compared them with the routes registered in [backend/api/cmd/server/main.go](../backend/api/cmd/server/main.go).
- **Unused files:** searched each Angular app for imports of every service, interceptor and shared component, ignoring `index.ts` barrel re-exports.
- **Deployment:** checked [backend/docker-compose.yml](../backend/docker-compose.yml) and [DEPLOY_RAILWAY.md](DEPLOY_RAILWAY.md) for which apps are actually deployed.

After removing anything, build each affected app to confirm nothing still depended on it:

```bash
cd kiosk/kiosk-frontend && npx ng build
cd backend/admin-frontend && npx ng build
cd backend/api && go build ./...
```
