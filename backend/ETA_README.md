# ETA Engine - Complete Implementation

## 📚 Table of Contents
- [Overview](#overview)
- [Architecture](#architecture)
- [Database Schema](#database-schema)
- [API Endpoints](#api-endpoints)
- [Quick Start](#quick-start)
- [Usage Examples](#usage-examples)
- [Data Flow](#data-flow)
- [Performance](#performance)

---

## Overview

The ETA (Estimated Time of Arrival) Engine predicts when buses will arrive at lounges using a sophisticated multi-factor calculation system.

### Key Features
✅ **Historical Learning** - Learns from past trips  
✅ **Context-Aware** - Considers time, driver, bus, weather  
✅ **Real-Time Adjustments** - Uses current GPS and speed  
✅ **Confidence Scoring** - Indicates prediction reliability  
✅ **Accuracy Tracking** - Measures and improves over time  

### Calculation Method
```
ETA = Historical Average × Context Factors × Real-Time Adjustment + Dwell Time
```

**Context Factors:**
- **Driver Performance** (0.95 = 5% faster than average)
- **Bus Performance** (1.05 = 5% slower than average)
- **Weather Impact** (1.15 = 15% slower in rain)
- **Time of Day** (1.3 = 30% slower during peak hours)

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                  FRONTEND APPLICATION                   │
│  (Angular - displays ETAs to users)                     │
└──────────────────────┬──────────────────────────────────┘
                       │ HTTP REST API
┌──────────────────────▼──────────────────────────────────┐
│                   GO BACKEND SERVER                      │
│  ┌────────────────────────────────────────────────┐    │
│  │  ETA Handler (eta_handler.go)                  │    │
│  │  - Receives API requests                       │    │
│  │  - Validates input                             │    │
│  │  - Returns JSON responses                      │    │
│  └────────────┬───────────────────────────────────┘    │
│               │                                          │
│  ┌────────────▼───────────────────────────────────┐    │
│  │  ETA Service (eta_service.go)                  │    │
│  │  - Business logic                              │    │
│  │  - Calls database functions                    │    │
│  │  - Data collection                             │    │
│  └────────────┬───────────────────────────────────┘    │
└───────────────┼──────────────────────────────────────────┘
                │ PostgreSQL Query
┌───────────────▼──────────────────────────────────────────┐
│              POSTGRESQL DATABASE                         │
│  ┌─────────────────────────────────────────────────┐   │
│  │ ETA Calculation Functions (003_*.sql)           │   │
│  │ - calculate_trip_lounge_etas()                  │   │
│  │ - get_historical_segment_duration()             │   │
│  │ - get_driver_speed_factor()                     │   │
│  │ - get_bus_speed_factor()                        │   │
│  └─────────────────────────────────────────────────┘   │
│                                                          │
│  ┌─────────────────────────────────────────────────┐   │
│  │ Data Tables                                      │   │
│  │ • trip_contexts (context per trip)              │   │
│  │ • segment_performance_facts (actual performance) │   │
│  │ • segment_aggregate_stats (pre-computed avg)    │   │
│  │ • driver_performance_profiles (driver factors)  │   │
│  │ • bus_performance_profiles (bus factors)        │   │
│  │ • route_segments (route breakdown)              │   │
│  │ • lounge_stop_metrics (dwell times)             │   │
│  │ • eta_predictions (accuracy tracking)           │   │
│  └─────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────┘
```

---

## Database Schema

### Normalized Star Schema

```
trip_contexts (Dimension Table)
├── One record per trip
└── Contains: driver, bus, weather, time context

segment_performance_facts (Fact Table)
├── Multiple records per trip (one per segment)
├── Links to trip_context
└── Contains: actual duration, speed, traffic

segment_aggregate_stats (Aggregation Table)
├── Pre-computed averages
└── Fast ETA lookup table

driver_performance_profiles
├── Driver efficiency metrics
└── Updated daily

bus_performance_profiles
├── Bus performance metrics
└── Updated daily
```

### Migration Files

| File | Purpose | Tables Created |
|------|---------|----------------|
| `001_create_eta_engine_tables.sql` | Base ETA tables | route_segments, segment_historical_performance, lounge_stop_metrics, eta_predictions |
| `002_optimized_eta_schema.sql` | Normalized schema | trip_contexts, segment_performance_facts, segment_aggregate_stats, profiles |
| `003_eta_calculation_functions.sql` | **Calculation functions** | Functions for ETA calculation, accuracy tracking |

---

## API Endpoints

### 1. Calculate Trip ETAs

**GET** `/api/v1/eta/trip/{tripId}?lat={lat}&lng={lng}&speed={speed}`

**Description:** Returns predicted arrival times for all remaining lounges on a trip.

**Query Parameters:**
- `lat` (optional) - Current latitude
- `lng` (optional) - Current longitude  
- `speed` (optional) - Current speed in km/h

**Response:**
```json
{
  "trip_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "calculated_at": "2026-03-12T14:30:00Z",
  "total_lounges": 2,
  "lounges": [
    {
      "lounge_id": "lounge-uuid-1",
      "lounge_name": "City Plaza Lounge",
      "segment_order": 3,
      "distance_remaining_km": 12.5,
      "estimated_arrival_time": "2026-03-12T15:15:00Z",
      "estimated_departure_time": "2026-03-12T15:30:00Z",
      "confidence_score": 87.5,
      "calculation_method": "historical",
      "component_details": {
        "baseline_minutes": 18,
        "historical_minutes": 22,
        "final_minutes": 24.5,
        "driver_factor": 0.95,
        "bus_factor": 1.0,
        "weather_factor": 1.15,
        "time_category": "morning_peak",
        "cumulative_minutes": 45.2,
        "dwell_minutes": 15
      }
    }
  ]
}
```

**Example:**
```bash
curl http://localhost:8080/api/v1/eta/trip/a1b2c3d4-e5f6-7890-abcd-ef1234567890?lat=6.9271&lng=79.8612&speed=45.5
```

---

### 2. Get Accuracy Report

**GET** `/api/v1/eta/accuracy?days={days}`

**Description:** Returns ETA prediction accuracy statistics.

**Query Parameters:**
- `days` (optional, default: 30) - Number of days to analyze

**Response:**
```json
{
  "period_days": 30,
  "generated_at": "2026-03-12T14:30:00Z",
  "reports": [
    {
      "total_predictions": 1250,
      "predictions_with_actuals": 1150,
      "avg_error_minutes": 3.2,
      "median_error_minutes": 2.5,
      "within_3_minutes_pct": 68.5,
      "within_5_minutes_pct": 85.2,
      "calculation_method": "historical",
      "confidence_range": "high"
    }
  ]
}
```

---

### 3. Record Actual Arrival

**POST** `/api/v1/eta/arrival`

**Description:** Records actual arrival time for accuracy tracking.

**Request Body:**
```json
{
  "trip_id": "trip-uuid",
  "lounge_id": "lounge-uuid",
  "actual_arrival_time": "2026-03-12T15:34:00Z"
}
```

---

### 4. Record Segment Performance

**POST** `/api/v1/eta/segment-performance`

**Description:** Records performance data for a completed segment.

**Request Body:**
```json
{
  "route_segment_id": "segment-uuid",
  "trip_context_id": "context-uuid",
  "segment_start_time": "2026-03-12T14:00:00Z",
  "segment_end_time": "2026-03-12T14:24:00Z",
  "actual_duration_minutes": 24.0,
  "average_speed_kmh": 21.25,
  "traffic_level": "heavy",
  "data_quality_score": 0.95
}
```

---

### 5. Create Trip Context

**POST** `/api/v1/eta/trip-context`

**Description:** Creates a trip context for data collection.

**Request Body:**
```json
{
  "active_trip_id": "trip-uuid",
  "trip_date": "2026-03-12",
  "driver_id": "driver-uuid",
  "bus_id": "bus-uuid",
  "departure_time": "08:00:00",
  "time_of_day_category": "morning_peak",
  "day_of_week": "monday",
  "weather_condition": "clear",
  "total_passengers": 25,
  "route_id": "route-uuid"
}
```

---

### 6. Refresh Statistics

**POST** `/api/v1/eta/refresh-stats`

**Description:** Manually triggers refresh of aggregate statistics.

**Response:**
```json
{
  "message": "Statistics refreshed successfully",
  "refreshed_at": "2026-03-12T14:30:00Z"
}
```

---

## Quick Start

### 1. Deploy Database

```bash
cd backend/migrations

# Step 1: Base tables
psql -h localhost -U postgres -d businfo -f 001_create_eta_engine_tables.sql

# Step 2: Optimized schema
psql -h localhost -U postgres -d businfo -f 002_optimized_eta_schema.sql

# Step 3: Calculation functions
psql -h localhost -U postgres -d businfo -f 003_eta_calculation_functions.sql
```

### 2. Update Go Server

```go
// In cmd/server/main.go
import (
    "backend/internal/handlers"
)

func main() {
    db := setupDatabase()
    router := mux.NewRouter()
    
    // Register ETA routes
    etaHandler := handlers.NewETAHandler(db)
    etaHandler.RegisterRoutes(router)
    
    // ... rest of your server setup
}
```

### 3. Test the API

```bash
# Calculate ETAs for a trip
curl http://localhost:8080/api/v1/eta/trip/YOUR-TRIP-UUID

# Check accuracy
curl http://localhost:8080/api/v1/eta/accuracy?days=7
```

---

## Usage Examples

### Frontend Integration (Angular)

```typescript
import { HttpClient } from '@angular/common/http';

export class ETAService {
  constructor(private http: HttpClient) {}
  
  getTripETAs(tripId: string, location?: {lat: number, lng: number, speed: number}) {
    let url = `/api/v1/eta/trip/${tripId}`;
    
    if (location) {
      url += `?lat=${location.lat}&lng=${location.lng}&speed=${location.speed}`;
    }
    
    return this.http.get<ETAResponse>(url);
  }
}
```

### Display ETAs in Component

```typescript
this.etaService.getTripETAs(tripId, currentLocation).subscribe(response => {
  response.lounges.forEach(lounge => {
    console.log(`${lounge.lounge_name}: Arriving at ${lounge.estimated_arrival_time}`);
    console.log(`Confidence: ${lounge.confidence_score}%`);
  });
});
```

### Real-Time Updates

```typescript
// Update ETAs every 30 seconds
setInterval(() => {
  const currentPosition = this.gpsService.getCurrentPosition();
  const currentSpeed = this.gpsService.getCurrentSpeed();
  
  this.etaService.getTripETAs(tripId, {
    lat: currentPosition.latitude,
    lng: currentPosition.longitude,
    speed: currentSpeed
  }).subscribe(response => {
    this.loungeETAs = response.lounges;
  });
}, 30000); // 30 seconds
```

---

## Data Flow

### ETA Calculation Flow

```
1. User Request
   └─> GET /api/v1/eta/trip/{tripId}?lat=6.9271&lng=79.8612&speed=45

2. Handler Layer (eta_handler.go)
   └─> Parse request
   └─> Call etaService.CalculateTripLoungeETAs()

3. Service Layer (eta_service.go)
   └─> Query database: calculate_trip_lounge_etas()

4. Database Function (003_eta_calculation_functions.sql)
   Step 1: Identify current position
   Step 2: Get remaining route segments
   Step 3: For each segment:
           a. Get historical average from segment_aggregate_stats
           b. Get driver factor from driver_performance_profiles
           c. Get bus factor from bus_performance_profiles
           d. Get weather multiplier
           e. Apply real-time speed adjustment
           f. Calculate final segment duration
   Step 4: Sum all segment durations
   Step 5: Add lounge dwell times
   Step 6: Return ETAs with confidence scores

5. Service Layer
   └─> Save predictions (async)
   └─> Return formatted response

6. Handler Layer
   └─> Return JSON response

7. Frontend
   └─> Display ETAs to user
```

### Data Collection Flow

```
1. Trip Starts
   └─> POST /api/v1/eta/trip-context
   └─> Creates trip_contexts record

2. Bus Moves Along Route
   └─> GPS tracking detects segment completion
   └─> POST /api/v1/eta/segment-performance
   └─> Creates segment_performance_facts record

3. Bus Arrives at Lounge
   └─> POST /api/v1/eta/arrival
   └─> Updates eta_predictions with actual_arrival_time
   └─> Calculates prediction_error_minutes

4. Daily Aggregation (2 AM)
   └─> refresh_segment_aggregate_stats()
   └─> refresh_driver_performance_profiles()
   └─> refresh_bus_performance_profiles()
   └─> REFRESH MATERIALIZED VIEW

5. Improved Accuracy
   └─> ETAs get more accurate over time
```

---

## Performance

### Query Performance Targets

| Operation | Target | Notes |
|-----------|--------|-------|
| ETA Calculation | < 500ms | Main calculation query |
| Save Prediction | < 100ms | Single insert |
| Accuracy Report | < 200ms | Pre-aggregated data |

### Optimization Strategies

1. **Pre-computed Aggregates**
   - `segment_aggregate_stats` table holds pre-calculated averages
   - Refreshed daily instead of calculating on every request

2. **Materialized View**
   - `segment_performance_with_context` joins data once
   - Faster than joining on every query

3. **Strategic Indexes**
   - All foreign keys indexed
   - Composite indexes on common query patterns

4. **Asynchronous Prediction Saving**
   - Predictions saved in background
   - Doesn't block ETA response

---

## Monitoring

### Health Checks

```sql
-- Check data collection
SELECT 
  COUNT(*) as trips_today,
  AVG(data_quality_score) as avg_quality
FROM segment_performance_facts spf
JOIN trip_contexts tc ON spf.trip_context_id = tc.id
WHERE tc.trip_date = CURRENT_DATE;

-- Check aggregation freshness
SELECT 
  MAX(last_calculated_at) as last_refresh,
  COUNT(*) as total_stats
FROM segment_aggregate_stats;

-- Check prediction accuracy
SELECT * FROM get_eta_accuracy_report(7);
```

### Accuracy Tracking

**Target Metrics:**
- ✅ 85% predictions within ±5 minutes
- ✅ 65% predictions within ±3 minutes
- ✅ Average error < 4 minutes

---

## Troubleshooting

### Problem: No historical data

**Symptom:** All ETAs use `calculation_method: "baseline"`

**Solution:**
1. Ensure trips are being recorded
2. Run: `SELECT COUNT(*) FROM segment_performance_facts`
3. Run: `SELECT refresh_segment_aggregate_stats()`

### Problem: Low confidence scores

**Symptom:** `confidence_score < 60`

**Solution:**
- Collect more trip data (need 30+ trips per segment)
- Verify GPS accuracy is good (`data_quality_score > 0.7`)

### Problem: Slow queries

**Symptom:** ETA calculation > 1 second

**Solution:**
1. Run: `ANALYZE segment_aggregate_stats`
2. Check index usage: `SELECT * FROM pg_stat_user_indexes`
3. Refresh materialized view: `REFRESH MATERIALIZED VIEW segment_performance_with_context`

---

## Documentation

- 📖 [ETA_ENGINE_IMPLEMENTATION_PLAN.md](../ETA_ENGINE_IMPLEMENTATION_PLAN.md) - Full technical spec
- 📖 [ETA_ENGINE_QUICK_START.md](../ETA_ENGINE_QUICK_START.md) - Conceptual overview
- 📖 [ETA_DEPLOYMENT_GUIDE.md](./ETA_DEPLOYMENT_GUIDE.md) - Deployment instructions

---

**Version:** 1.0  
**Last Updated:** March 12, 2026  
**Status:** ✅ Production Ready
