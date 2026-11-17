# Documentation

All project documentation lives here. The folders mirror the code: `backend/...` covers the server side and `kiosk/...` covers what runs on the kiosk PC or TV box.

## Start here

| Doc | What it covers |
|---|---|
| [QUICKSTART.md](QUICKSTART.md) | Test-run the whole system on macOS or Windows, including the one-command kiosk runner |
| [ETA_IMPLEMENTATION.md](ETA_IMPLEMENTATION.md) | How bus arrival and departure times at a lounge are calculated (current design) |
| [WEBSOCKET_SETUP.md](WEBSOCKET_SETUP.md) | Optional WebSocket update notifications between the API and the sync agent |

## Backend: API (`backend/api`)

| Doc | What it covers |
|---|---|
| [README.md](backend/api/README.md) | API overview |
| [SETUP_GUIDE.md](backend/api/SETUP_GUIDE.md) | Backend setup and installation |
| [DAILY_TRIP_GENERATOR.md](backend/api/DAILY_TRIP_GENERATOR.md) | How daily trips are generated from schedules |
| [MIGRATION_GUIDE.md](backend/api/MIGRATION_GUIDE.md) | Advertisement groups migration |
| [NEW_TABLES_README.md](backend/api/NEW_TABLES_README.md) | Advertisement-cost and TV display tables |
| [ETA_README.md](backend/api/ETA_README.md) | ETA engine v3 tables and `/api/v1/eta` endpoints (older design, see the note in the doc) |
| [migrations/ETA_SCHEMA_REFERENCE.md](backend/api/migrations/ETA_SCHEMA_REFERENCE.md) | ETA engine table reference |
| [migrations/OPTIMIZED_SCHEMA_COMPARISON.md](backend/api/migrations/OPTIMIZED_SCHEMA_COMPARISON.md) | ETA engine schema optimisation notes |
| [ETA_FIX_SUMMARY.md](backend/api/ETA_FIX_SUMMARY.md) | *Superseded:* earlier lounge-specific arrival ETA |
| [DEPARTURE_ETA_IMPLEMENTATION.md](backend/api/DEPARTURE_ETA_IMPLEMENTATION.md) | *Superseded:* earlier departure ETA (arrival + 20 min) |

The training-data pack keeps its own README next to its CSV files: [backend/api/sample_training_data/README.md](../backend/api/sample_training_data/README.md). Its generator script rewrites that file.

## Backend: admin web app (`backend/admin-frontend`)

| Doc | What it covers |
|---|---|
| [README.md](backend/admin-frontend/README.md) | Angular project basics |
| [DEVELOPMENT.md](backend/admin-frontend/DEVELOPMENT.md) | Angular development quick reference |
| [ROUTE_MANAGEMENT_GUIDE.md](backend/admin-frontend/ROUTE_MANAGEMENT_GUIDE.md) | Route management feature |
| [TRIP_INTEGRATION_SETUP.md](backend/admin-frontend/TRIP_INTEGRATION_SETUP.md) | Trip data integration (Supabase) |
| [TV_SYNC_AGENT_PAGE.md](backend/admin-frontend/TV_SYNC_AGENT_PAGE.md) | The TV Sync Agent page |

## Backend: TV sync agent dashboard (`backend/tv-sync-agent-frontend`)

| Doc | What it covers |
|---|---|
| [README.md](backend/tv-sync-agent-frontend/README.md) | Project overview |
| [QUICK_START.md](backend/tv-sync-agent-frontend/QUICK_START.md) | Running it |
| [OVERVIEW.md](backend/tv-sync-agent-frontend/OVERVIEW.md) | What was built |
| [FEATURES.md](backend/tv-sync-agent-frontend/FEATURES.md) | Features |
| [API_INTEGRATION.md](backend/tv-sync-agent-frontend/API_INTEGRATION.md) | API integration |
| [PROJECT_SUMMARY.md](backend/tv-sync-agent-frontend/PROJECT_SUMMARY.md) | Project summary |
| [ENHANCED_VERSION.md](backend/tv-sync-agent-frontend/ENHANCED_VERSION.md) | Enhanced version notes |
| [PROFESSIONAL_UPDATE.md](backend/tv-sync-agent-frontend/PROFESSIONAL_UPDATE.md) | Design update notes |

## Kiosk

| Doc | What it covers |
|---|---|
| [kiosk-electron-shell/WINDOWS_KIOSK_HARDENING.md](kiosk/kiosk-electron-shell/WINDOWS_KIOSK_HARDENING.md) | Locking down a Windows kiosk PC, plus Electron and agent startup |
| [tv-sync-agent-go/README_TV_BOX_SETUP.md](kiosk/tv-sync-agent-go/README_TV_BOX_SETUP.md) | Setting up a Raspberry Pi TV box |
