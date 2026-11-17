# ETA implementation

How the system works out when each bus reaches a lounge (arrival) and leaves it (departure), as shown on the kiosk boards.

This replaces the earlier designs described in [backend/api/ETA_FIX_SUMMARY.md](backend/api/ETA_FIX_SUMMARY.md) and [backend/api/DEPARTURE_ETA_IMPLEMENTATION.md](backend/api/DEPARTURE_ETA_IMPLEMENTATION.md).

## Summary

- Arrivals and departures come from **one calculation**, so a bus can never be shown leaving a lounge before it arrives.
- **Arrival:**
  - **Live GPS** is used when the conductor app has reported a recent position.
  - Otherwise a **schedule estimate** is used, checked for a believable speed.
- **Departure** = arrival + a short stopover (3 minutes by default).
- **Each board** shows buses due between **15 minutes ago and 6 hours ahead**, earliest first, at most **10**.
- **Buses drop off the board** 15 minutes after their estimated time. They drop off immediately if GPS shows they've passed the lounge, or if the conductor marks the trip completed or cancelled.
- The system has **no "arrived" or "departed" signal**. The conductor app doesn't send one, so everything shown is an estimate.

## Data flow

```
Conductor app ──(trip status, bus GPS)──► active_trips ─┐
Trip generator ──(daily trips)──────────► scheduled_trips ┤
Route editor ──(segments, stops)────────► route_segments, master_route_stops ┤
                                                          ▼
                              API  GET /api/arrivals/lounge/{loungeId}
                                   GET /api/departures/lounge/{loungeId}
                                                          │
                     Sync agent (kiosk, every ~minute) ───┤──► local-store/schedule.json
                                                          ▼
                                               Kiosk display (bids-display)
```

The kiosk shows the `time` and `remarks` the API sends; it doesn't calculate times itself.

## 1. Which trips are considered

Query: `LoungeETARepository.GetLoungeTrips` in [lounge_eta_repository.go](../backend/api/internal/database/lounge_eta_repository.go).

A trip is considered for a lounge when all of these hold:

| Rule | Why |
|---|---|
| The lounge is linked to the trip's route (`lounge_routes`) and is operational | The bus passes it |
| The operator's route direction is `UP` (or not set) | Lounges are defined in the route's stop order. `DOWN` data is inconsistent: half of it lists stops in reverse. |
| The operator stops at both stops either side of the lounge (`bus_owner_routes.selected_stop_ids`), or has no stop list | Express services that skip that part of the route are left out |
| `scheduled_trips.ever_published = true` and status isn't `cancelled` | Only trips the operator has published |
| No active trip yet, or the conductor's status is `not_started`, `in_transit` or `at_stop` | `completed` and `cancelled` trips drop off |
| The trip left the start of its route between 12 hours ago and 6 hours from now | Wide enough to catch every bus that could reach the lounge within the board window |

If a trip has several `active_trips` rows, the most recently updated one is used.

## 2. Where the lounge is on the route

The route is modelled from its `route_segments`, in segment order. Segments with no distance or invalid coordinates are skipped.

- **Lounge position:** the lounge's coordinates are projected onto the route, giving km from the start.
- **Kept between its stops:** the position is limited to lie between the stops before and after the lounge. This matters because several lounges are stored with a default Colombo location instead of their real one.
- **Routes with no segments:** 10 of the 36 routes that pass lounges have none. The calculation then uses the lounge's place in the stop list instead (see 3b).

## 3. Arrival estimate

The function `computeLoungeETAs` in [lounge_eta.go](../backend/api/internal/services/lounge_eta.go) does this for each trip.

### 3a. Live GPS (preferred)

Live GPS is used when **all** of these hold:
- The conductor's trip status is `in_transit` or `at_stop`.
- `active_trips.current_latitude/longitude` are set.
- `last_location_update` is no more than **5 minutes** old (and not more than 2 minutes in the future).
- The bus is within **3 km** of the route.

When it applies:
- **ETA** = time of the GPS position + travel time over the road segments still to go, at road-type speeds (see 3c).
- **Already past the lounge** (more than **0.3 km** beyond it): the trip is left off this lounge's board.
- **Stale or off-route position:** it's ignored, and the schedule estimate is used instead.

### 3b. Schedule estimate (minutes from the start of the route)

These sources are tried in order. The first one whose result implies an average speed of **15–60 km/h** over the distance to the lounge is used:

| Order | Source (`etaSource`) | Calculation |
|---|---|---|
| 1 | `stop-offsets` | `master_route_stops.arrival_time_offset_minutes` of the stops either side, interpolated by where the lounge sits between them. Falls back to their average, or the single offset if only one is set. |
| 2 | `trip-duration` | `scheduled_trips.estimated_duration_minutes` × share of the route covered at the lounge |
| 3 | `route-duration` | `master_routes.estimated_duration_minutes` × share of the route covered at the lounge |
| 4 | `road-type` | Travel time from the start of the route to the lounge at road-type speeds |

- **Share of the route covered:** the lounge's km ÷ the route's total km. Without segments, it's (rank of stop before + rank of stop after) ÷ 2 ÷ (number of stops − 1).
- **When the speed check is skipped:** if the lounge's distance is unknown or under 2 km.
- **If no source passes the check:** the road-type estimate is used if it exists, otherwise the first available source.
- **If nothing can be estimated at all,** the trip isn't shown.

**Arrival** = the trip's `departure_datetime` + the estimate.

### 3c. Road-type speeds

The speed of each segment is **40 km/h ÷ the multiplier** for its `road_type`, from `eta_profiles` in [config.json](../backend/api/internal/config/config.json):

| Road type | Multiplier | Speed |
|---|---|---|
| HIGHWAY | 0.8 | 50 km/h |
| EXPRESSWAY | 0.85 | 47 km/h |
| URBAN, UNKNOWN | 1.0 | 40 km/h |
| ARTERIAL | 1.1 | 36 km/h |
| COLLECTOR | 1.3 | 31 km/h |
| LOCAL | 1.5 | 27 km/h |
| SERVICE | 1.7 | 24 km/h |
| RURAL | 2.0 | 20 km/h |

A segment whose road type isn't in the table uses its own `baseline_speed_kmh`, or 40 km/h if that isn't set.

### 3d. ML engine (optional, off by default)

The Python engine in `backend/eta_engine` can't predict anything until its trained model files exist, so it's disabled.

To turn it on:
1. Set `ETA_ENGINE_ENABLED=true`.
2. Point `ETA_ENGINE_URL` at the engine (default `http://localhost:8000`).

When it's on, trips without a live GPS estimate are sent to `POST /api/v1/predict_batch` with a 3-second timeout. A prediction replaces the schedule estimate (as `eta-engine`) only if it passes the same 15–60 km/h check. Any failure is ignored.

## 4. Departure

**Departure = arrival + `lounge_stopover_minutes`**, set in [config.json](../backend/api/internal/config/config.json). The default is 3.

Because both times come from the same arrival estimate, the departure is always exactly the stopover after the arrival shown on the arrivals board.

## 5. The boards

`LoungeBoard.Arrivals()` and `LoungeBoard.Departures()` in [lounge_eta.go](../backend/api/internal/services/lounge_eta.go) select what each board shows:

- **Window:** buses whose arrival (arrivals board) or departure (departures board) is between **15 minutes ago and 6 hours from now**.
- **Order and limit:** sorted by that time, cut to **10**.

The times and remarks are formatted in the service timezone: `TRIP_GENERATOR_TIMEZONE`, default `Asia/Colombo`.

### Remarks

| Arrivals board: minutes until arrival | Remark |
|---|---|
| more than 5 in the past | `Delayed N min` |
| 0 to 5 in the past | `Arriving Now` |
| 1–5 | `Arriving in N min` |
| 6–15 | `Expected in N min` |
| more than 15 | `Expected at HH:MM` |

| Departures board: minutes until departure | Remark |
|---|---|
| more than 10 in the past | `Delayed N min` |
| 0 to 10 in the past | `Departing Now` |
| 1–5 | `Boarding Now` |
| 6–15 | `Final Call - N min` |
| 16–30 | `Boarding Soon - N min` |
| more than 30 | `Departs at HH:MM` |

The kiosk display colours rows by these remarks.

## 6. API response fields

Each entry in `arrivals[]` and `departures[]` keeps its earlier fields, plus:

| Field | Meaning |
|---|---|
| `time` | Arrival or departure time shown, `HH:MM` |
| `remarks` | See above |
| `eta` (arrivals), `arrivalEta` and `departureEta` (departures) | Full timestamps |
| `stopoverMinutes` (departures) | The stopover used |
| `offsetMinutes`, `calculatedETAMinutes` | Schedule estimate: minutes from the start of the route to the lounge |
| `distanceKm` | Live: km left to the lounge. Schedule: km from the start of the route to the lounge (0 if unknown). |
| `etaSource` | `live-gps`, `eta-engine`, `stop-offsets`, `trip-duration`, `route-duration` or `road-type` |
| `status` | The conductor's trip status, or `not_started` |

## 7. Settings

| Setting | Where | Default |
|---|---|---|
| `lounge_stopover_minutes` | `backend/api/internal/config/config.json` | 3 |
| `eta_profiles` (road-type multipliers) | `backend/api/internal/config/config.json` | see 3c |
| `TRIP_GENERATOR_TIMEZONE` | environment / `.env` | `Asia/Colombo` |
| `ETA_ENGINE_ENABLED`, `ETA_ENGINE_URL` | environment / `.env` | off, `http://localhost:8000` |

Fixed values, all constants at the top of [lounge_eta.go](../backend/api/internal/services/lounge_eta.go):

| What | Value |
|---|---|
| Board window | 15 min ago to 6 h ahead |
| Rows per board | 10 |
| Trip look-back | 12 h |
| Maximum GPS age | 5 min |
| Maximum distance off the route | 3 km |
| "Passed" margin | 0.3 km |
| Plausible average speed | 15–60 km/h |
| Board cache | 10 s |
| Route geometry cache | 10 min |

The API handlers also cache each response for 5 seconds.

## 8. Code map

| File | Contents |
|---|---|
| [internal/services/lounge_eta.go](../backend/api/internal/services/lounge_eta.go) | The whole calculation: route model, lounge position, schedule and live estimates, speed check, board selection, remarks, `LoungeBoardService` with caches |
| [internal/database/lounge_eta_repository.go](../backend/api/internal/database/lounge_eta_repository.go) | Trip query (section 1) and route geometry query |
| [internal/services/eta_engine.go](../backend/api/internal/services/eta_engine.go) | Client for the optional Python ETA engine |
| [internal/services/arrival_service.go](../backend/api/internal/services/arrival_service.go) | `GetArrivalsByLoungeID` turns the board into the API response. Also holds the older all-lounges path (section 9). |
| [internal/services/departure_service.go](../backend/api/internal/services/departure_service.go) | `GetDeparturesByLoungeID` does the same for departures |
| [internal/services/lounge_eta_test.go](../backend/api/internal/services/lounge_eta_test.go) | Unit tests |
| [cmd/server/main.go](../backend/api/cmd/server/main.go) | Wires `LoungeBoardService` into both services |

## 9. Admin "all lounges" pages

The admin Arrival Management and Departure Management pages call `GET /api/arrivals` and `GET /api/departures`. These still use the older per-lounge queries and calculation:
- They only show trips that have an active trip.
- They work through: ML engine (if enabled), then GPS, then road-type speeds, then stop offsets.
- Departures use the SQL estimate + 20 minutes.

The live-GPS "smoothing" step, which added the whole schedule offset to live ETAs, has been removed from this path too. Moving these pages onto `LoungeBoardService` is a follow-up.

## 10. Data the estimates depend on

As of November 2026, against the shared database:

| Data | State | Effect |
|---|---|---|
| Conductor GPS (`active_trips` position) | Newest position is from December 2025 | Live GPS never applies yet. It's covered by unit tests only. |
| Stop offsets | Missing on some routes, for example route 38 | Falls through to durations or road-type |
| Trip durations | Often unrealistic: 120 min for a 215 km route | Rejected by the speed check |
| Road types | About 95% unknown on route 38 | Road-type estimate runs at 40 km/h; run `backend/eta_engine/update_route_segment_road_types.py` to fill them in |
| Route segments | Missing on 10 of 36 lounge routes | Stop-list fallback, no speed check |
| Lounge coordinates | Some are a default Colombo location | Kept between the lounge's stops |
| Published trips | Very few (`ever_published`) | Most boards are empty |

Better data makes the schedule estimate more accurate without code changes. In order of impact: stop offsets on every route, then road types, then realistic durations.

## 11. Testing

Unit tests run without a database:

```
cd backend/api
go test ./internal/services/ -run 'Lounge|Schedule|GPS|Departure|SelectForBoard|Remarks|NoEstimate|Stale'
```

They cover:
- every estimate source and the speed check
- live GPS: normal, passed, stale, off-route, and trip not started
- departure = arrival + stopover
- the board window, sort and limit
- remarks and timezone formatting

To check the boards against real data **without starting the API**, call `LoungeBoardService.Board` (or the arrival and departure services) from a small Go program. That only reads from the database. Starting the API also starts the daily trip generator, which writes to `scheduled_trips`.

## 12. Known limitations and next steps

- **No arrival or departure signal.** Buses only leave the board by time, by GPS position, or when the conductor ends the trip.
- **Only `UP` trips are shown.** Showing `DOWN` trips would need direction-aware positions and consistent `DOWN` stop data.
- **Admin all-lounges pages** still use the older path (section 9).
- **Data improvements** listed in section 10.
- **The ML engine** needs trained model files before it can be enabled.
