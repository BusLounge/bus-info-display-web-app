# ETA Engine - Schema Reference Guide

## 📋 Complete Table Attributes & Examples

---

## 1️⃣ `route_segments` - Route Segmentation

### Purpose
Breaks each route into measurable segments between consecutive points (stops/lounges).

### Attributes

| Column | Type | Required | Description |
|--------|------|----------|-------------|
| `id` | UUID | ✅ | Primary key |
| `master_route_id` | UUID | ✅ | FK to master_routes |
| `start_point_type` | VARCHAR(20) | ✅ | 'stop', 'lounge', 'origin' |
| `start_point_id` | UUID | ✅ | ID of start point |
| `end_point_type` | VARCHAR(20) | ✅ | 'stop', 'lounge', 'destination' |
| `end_point_id` | UUID | ✅ | ID of end point |
| `segment_order` | INTEGER | ✅ | Sequence number (1, 2, 3...) |
| `start_latitude` | NUMERIC(10,7) | ✅ | Starting GPS coordinate |
| `start_longitude` | NUMERIC(10,7) | ✅ | Starting GPS coordinate |
| `end_latitude` | NUMERIC(10,7) | ✅ | Ending GPS coordinate |
| `end_longitude` | NUMERIC(10,7) | ✅ | Ending GPS coordinate |
| `distance_km` | NUMERIC(8,3) | ✅ | Segment distance in kilometers |
| `baseline_duration_minutes` | INTEGER | ✅ | Ideal time (distance/speed) |
| `baseline_speed_kmh` | NUMERIC(5,2) | ❌ | Expected speed (default: 40) |
| `road_type` | VARCHAR(20) | ❌ | 'highway', 'urban', 'rural', 'mixed' |
| `traffic_sensitivity_factor` | NUMERIC(3,2) | ❌ | How affected by traffic (default: 1.0) |
| `elevation_change_meters` | INTEGER | ❌ | Elevation difference |
| `encoded_polyline_segment` | TEXT | ❌ | Google polyline for route visualization |
| `created_at` | TIMESTAMPTZ | ✅ | Record creation time |
| `updated_at` | TIMESTAMPTZ | ✅ | Last update time |

### Example Data

```sql
INSERT INTO route_segments VALUES
-- Segment 1: Colombo Fort → Maradana Stop
(
  'a1b2c3d4-0001-0000-0000-000000000001'::UUID,  -- id
  'route-colombo-kandy'::UUID,                    -- master_route_id
  'origin',                                        -- start_point_type
  'colombo-fort-terminal'::UUID,                  -- start_point_id
  'stop',                                          -- end_point_type
  'maradana-stop'::UUID,                          -- end_point_id
  1,                                               -- segment_order
  6.9271,                                          -- start_latitude
  79.8612,                                         -- start_longitude
  6.9497,                                          -- end_latitude
  79.8656,                                         -- end_longitude
  8.5,                                             -- distance_km
  13,                                              -- baseline_duration_minutes (8.5km / 40km/h ≈ 13min)
  40.0,                                            -- baseline_speed_kmh
  'urban',                                         -- road_type
  1.2,                                             -- traffic_sensitivity_factor
  0,                                               -- elevation_change_meters
  'encoded_polyline_here',                         -- encoded_polyline_segment
  NOW(),                                           -- created_at
  NOW()                                            -- updated_at
),

-- Segment 2: Maradana Stop → City Plaza Lounge
(
  'a1b2c3d4-0002-0000-0000-000000000002'::UUID,
  'route-colombo-kandy'::UUID,
  'stop',
  'maradana-stop'::UUID,
  'lounge',
  'city-plaza-lounge'::UUID,
  2,                                               -- segment_order
  6.9497,
  79.8656,
  7.0121,
  79.9156,
  12.0,                                            -- distance_km
  18,                                              -- baseline_duration_minutes
  40.0,
  'urban',
  1.3,
  15,                                              -- slight uphill
  'encoded_polyline_here',
  NOW(),
  NOW()
),

-- Segment 3: City Plaza Lounge → Peradeniya
(
  'a1b2c3d4-0003-0000-0000-000000000003'::UUID,
  'route-colombo-kandy'::UUID,
  'lounge',
  'city-plaza-lounge'::UUID,
  'destination',
  'peradeniya-terminal'::UUID,
  3,
  7.0121,
  79.9156,
  7.2654,
  80.5981,
  65.0,                                            -- distance_km
  98,                                              -- baseline_duration_minutes
  40.0,
  'highway',
  0.8,                                             -- less affected by traffic
  120,                                             -- more elevation
  'encoded_polyline_here',
  NOW(),
  NOW()
);
```

---

## 2️⃣ `trip_contexts` - Trip Context (One Per Trip)

### Purpose
Stores contextual information that applies to the entire trip (driver, bus, weather, time).

### Attributes

| Column | Type | Required | Description |
|--------|------|----------|-------------|
| `id` | UUID | ✅ | Primary key |
| `active_trip_id` | UUID | ✅ | FK to active_trips (unique) |
| `scheduled_trip_id` | UUID | ❌ | FK to scheduled_trips |
| `trip_date` | DATE | ✅ | Date of trip |
| `driver_id` | UUID | ✅ | FK to bus_staff |
| `driver_experience_years` | INTEGER | ❌ | Years of driving experience |
| `driver_rating` | NUMERIC(3,2) | ❌ | Average driver rating (0-5) |
| `bus_id` | UUID | ✅ | FK to buses |
| `bus_type` | VARCHAR(50) | ❌ | 'AC Luxury', 'Semi-Luxury', 'Normal' |
| `has_ac` | BOOLEAN | ✅ | Air conditioning available |
| `bus_age_years` | NUMERIC(4,2) | ❌ | Bus age in years |
| `departure_time` | TIME | ✅ | Departure time |
| `time_of_day_category` | VARCHAR(20) | ✅ | 'morning_peak', 'midday', 'evening_peak', 'night', 'early_morning' |
| `day_of_week` | VARCHAR(10) | ✅ | 'monday', 'tuesday', etc. |
| `is_weekend` | BOOLEAN | ✅ | Saturday or Sunday |
| `is_holiday` | BOOLEAN | ✅ | Public holiday |
| `weather_condition` | VARCHAR(20) | ✅ | 'clear', 'rain', 'heavy_rain', 'fog' |
| `temperature_celsius` | INTEGER | ❌ | Temperature in Celsius |
| `total_passengers` | INTEGER | ✅ | Number of passengers |
| `route_id` | UUID | ✅ | FK to master_routes |
| `created_at` | TIMESTAMPTZ | ✅ | Record creation time |

### Example Data

```sql
INSERT INTO trip_contexts VALUES
-- Trip 1: Monday morning peak, experienced driver, rainy
(
  'ctx-00001'::UUID,                              -- id
  'trip-active-001'::UUID,                        -- active_trip_id
  'trip-scheduled-001'::UUID,                     -- scheduled_trip_id
  '2026-03-10',                                   -- trip_date
  'driver-john-doe'::UUID,                        -- driver_id
  8,                                              -- driver_experience_years
  4.5,                                            -- driver_rating
  'bus-ac-001'::UUID,                             -- bus_id
  'AC Luxury',                                    -- bus_type
  true,                                           -- has_ac
  2.5,                                            -- bus_age_years
  '08:00:00',                                     -- departure_time
  'morning_peak',                                 -- time_of_day_category
  'monday',                                       -- day_of_week
  false,                                          -- is_weekend
  false,                                          -- is_holiday
  'rain',                                         -- weather_condition
  28,                                             -- temperature_celsius
  45,                                             -- total_passengers
  'route-colombo-kandy'::UUID,                   -- route_id
  '2026-03-10 08:00:00'::TIMESTAMPTZ             -- created_at
),

-- Trip 2: Sunday afternoon, new driver, clear weather
(
  'ctx-00002'::UUID,
  'trip-active-002'::UUID,
  'trip-scheduled-002'::UUID,
  '2026-03-10',
  'driver-jane-smith'::UUID,
  2,                                              -- less experienced
  4.2,
  'bus-normal-005'::UUID,
  'Normal',
  false,                                          -- no AC
  5.0,
  '14:00:00',
  'midday',
  'sunday',
  true,                                           -- is_weekend
  false,
  'clear',
  32,
  28,
  'route-colombo-kandy'::UUID,
  '2026-03-10 14:00:00'::TIMESTAMPTZ
);
```

---

## 3️⃣ `segment_performance_facts` - Performance Metrics

### Purpose
Stores only the performance metrics for each segment (normalized to reduce redundancy).

### Attributes

| Column | Type | Required | Description |
|--------|------|----------|-------------|
| `id` | UUID | ✅ | Primary key |
| `route_segment_id` | UUID | ✅ | FK to route_segments |
| `trip_context_id` | UUID | ✅ | FK to trip_contexts |
| `segment_start_time` | TIMESTAMPTZ | ✅ | When segment started |
| `segment_end_time` | TIMESTAMPTZ | ✅ | When segment ended |
| `actual_duration_minutes` | NUMERIC(8,2) | ✅ | How long it actually took |
| `average_speed_kmh` | NUMERIC(6,2) | ❌ | Average speed during segment |
| `duration_variance_minutes` | NUMERIC(8,2) | ❌ | Difference from baseline |
| `traffic_level` | VARCHAR(20) | ❌ | 'light', 'moderate', 'heavy', 'congested' |
| `actual_distance_km` | NUMERIC(8,3) | ❌ | GPS-measured distance |
| `gps_accuracy_meters` | INTEGER | ❌ | GPS accuracy (lower = better) |
| `data_quality_score` | NUMERIC(3,2) | ✅ | 0.0 to 1.0 (quality indicator) |
| `recorded_at` | TIMESTAMPTZ | ✅ | When data was recorded |

### Example Data

```sql
INSERT INTO segment_performance_facts VALUES
-- Segment 1 performance from Trip 1 (rainy morning, took longer)
(
  'perf-00001'::UUID,                             -- id
  'a1b2c3d4-0001-0000-0000-000000000001'::UUID,  -- route_segment_id (Colombo → Maradana)
  'ctx-00001'::UUID,                              -- trip_context_id
  '2026-03-10 08:00:00'::TIMESTAMPTZ,            -- segment_start_time
  '2026-03-10 08:24:00'::TIMESTAMPTZ,            -- segment_end_time
  24.0,                                           -- actual_duration_minutes (vs 13 baseline)
  21.25,                                          -- average_speed_kmh (8.5km in 24min)
  11.0,                                           -- duration_variance_minutes (24-13)
  'heavy',                                        -- traffic_level
  8.5,                                            -- actual_distance_km
  12,                                             -- gps_accuracy_meters
  0.95,                                           -- data_quality_score
  '2026-03-10 08:24:05'::TIMESTAMPTZ             -- recorded_at
),

-- Segment 1 performance from Trip 2 (clear Sunday, much faster)
(
  'perf-00002'::UUID,
  'a1b2c3d4-0001-0000-0000-000000000001'::UUID,  -- same segment
  'ctx-00002'::UUID,                              -- different trip context
  '2026-03-10 14:00:00'::TIMESTAMPTZ,
  '2026-03-10 14:15:00'::TIMESTAMPTZ,
  15.0,                                           -- actual_duration_minutes (faster!)
  34.0,                                           -- average_speed_kmh
  2.0,                                            -- duration_variance_minutes
  'light',                                        -- traffic_level
  8.5,
  8,
  0.98,
  '2026-03-10 14:15:03'::TIMESTAMPTZ
),

-- Segment 2 performance from Trip 1
(
  'perf-00003'::UUID,
  'a1b2c3d4-0002-0000-0000-000000000002'::UUID,  -- Maradana → City Plaza
  'ctx-00001'::UUID,
  '2026-03-10 08:24:00'::TIMESTAMPTZ,
  '2026-03-10 08:50:00'::TIMESTAMPTZ,
  26.0,                                           -- actual_duration_minutes (vs 18 baseline)
  27.7,                                           -- average_speed_kmh
  8.0,
  'heavy',
  12.0,
  10,
  0.93,
  '2026-03-10 08:50:02'::TIMESTAMPTZ
);
```

---

## 4️⃣ `segment_aggregate_stats` - Pre-Computed Averages

### Purpose
Stores pre-aggregated statistics for fast ETA lookups (refreshed daily).

### Attributes

| Column | Type | Required | Description |
|--------|------|----------|-------------|
| `id` | UUID | ✅ | Primary key |
| `route_segment_id` | UUID | ✅ | FK to route_segments |
| `time_of_day_category` | VARCHAR(20) | ✅ | Peak, midday, etc. |
| `day_of_week` | VARCHAR(10) | ✅ | Monday, Tuesday, etc. |
| `weather_condition` | VARCHAR(20) | ✅ | Clear, rain, etc. |
| `avg_duration_minutes` | NUMERIC(8,2) | ✅ | Average duration |
| `median_duration_minutes` | NUMERIC(8,2) | ❌ | Median duration |
| `stddev_duration_minutes` | NUMERIC(8,2) | ❌ | Standard deviation |
| `min_duration_minutes` | NUMERIC(8,2) | ❌ | Fastest time |
| `max_duration_minutes` | NUMERIC(8,2) | ❌ | Slowest time |
| `p95_duration_minutes` | NUMERIC(8,2) | ❌ | 95th percentile |
| `avg_speed_kmh` | NUMERIC(6,2) | ❌ | Average speed |
| `sample_count` | INTEGER | ✅ | Number of trips used |
| `last_trip_date` | DATE | ❌ | Most recent trip |
| `avg_data_quality` | NUMERIC(3,2) | ❌ | Average quality score |
| `last_calculated_at` | TIMESTAMPTZ | ✅ | Last refresh time |
| `calculation_window_days` | INTEGER | ✅ | Days of data used (default: 90) |

### Example Data

```sql
INSERT INTO segment_aggregate_stats VALUES
-- Segment 1: Monday morning peak + rain = slow
(
  'agg-00001'::UUID,
  'a1b2c3d4-0001-0000-0000-000000000001'::UUID,  -- Colombo → Maradana
  'morning_peak',
  'monday',
  'rain',
  24.5,                                           -- avg_duration_minutes
  24.0,                                           -- median_duration_minutes
  3.2,                                            -- stddev_duration_minutes
  18.0,                                           -- min_duration_minutes
  32.0,                                           -- max_duration_minutes
  28.0,                                           -- p95_duration_minutes
  21.0,                                           -- avg_speed_kmh
  45,                                             -- sample_count (45 trips analyzed)
  '2026-03-10',                                   -- last_trip_date
  0.92,                                           -- avg_data_quality
  '2026-03-11 02:00:00'::TIMESTAMPTZ,            -- last_calculated_at
  90                                              -- calculation_window_days
),

-- Segment 1: Sunday midday + clear = fast
(
  'agg-00002'::UUID,
  'a1b2c3d4-0001-0000-0000-000000000001'::UUID,  -- same segment
  'midday',
  'sunday',
  'clear',
  14.5,                                           -- avg_duration_minutes (much faster!)
  15.0,
  2.1,
  12.0,
  18.0,
  17.0,
  35.2,                                           -- avg_speed_kmh
  28,                                             -- sample_count
  '2026-03-10',
  0.95,
  '2026-03-11 02:00:00'::TIMESTAMPTZ,
  90
),

-- Segment 2: Monday morning peak + rain
(
  'agg-00003'::UUID,
  'a1b2c3d4-0002-0000-0000-000000000002'::UUID,  -- Maradana → City Plaza
  'morning_peak',
  'monday',
  'rain',
  26.8,
  26.0,
  4.5,
  20.0,
  38.0,
  32.0,
  26.9,
  52,
  '2026-03-10',
  0.91,
  '2026-03-11 02:00:00'::TIMESTAMPTZ,
  90
);
```

---

## 5️⃣ `driver_performance_profiles` - Driver Efficiency

### Purpose
Pre-computed driver performance metrics (refreshed daily).

### Attributes

| Column | Type | Required | Description |
|--------|------|----------|-------------|
| `id` | UUID | ✅ | Primary key |
| `driver_id` | UUID | ✅ | FK to bus_staff (unique) |
| `avg_speed_factor` | NUMERIC(4,3) | ✅ | Speed relative to average (0.95 = 5% faster) |
| `punctuality_score` | NUMERIC(3,2) | ✅ | 0.0-1.0 (how often beats baseline) |
| `consistency_score` | NUMERIC(3,2) | ✅ | 0.0-1.0 (low variance = high score) |
| `total_trips_analyzed` | INTEGER | ✅ | Number of trips |
| `total_segments_analyzed` | INTEGER | ✅ | Number of segments |
| `primary_routes` | UUID[] | ❌ | Array of route IDs |
| `last_calculated_at` | TIMESTAMPTZ | ✅ | Last refresh time |
| `calculation_window_days` | INTEGER | ✅ | Days analyzed (default: 90) |

### Example Data

```sql
INSERT INTO driver_performance_profiles VALUES
-- John Doe: Experienced, fast, consistent
(
  'driver-prof-001'::UUID,
  'driver-john-doe'::UUID,
  0.95,                                           -- avg_speed_factor (5% faster than average)
  0.85,                                           -- punctuality_score (beats baseline 85% of time)
  0.92,                                           -- consistency_score (very consistent)
  156,                                            -- total_trips_analyzed
  1248,                                           -- total_segments_analyzed
  ARRAY['route-colombo-kandy'::UUID, 'route-kandy-colombo'::UUID],  -- primary_routes
  '2026-03-11 02:00:00'::TIMESTAMPTZ,
  90
),

-- Jane Smith: Less experienced, slower, less consistent
(
  'driver-prof-002'::UUID,
  'driver-jane-smith'::UUID,
  1.08,                                           -- avg_speed_factor (8% slower)
  0.62,                                           -- punctuality_score
  0.75,                                           -- consistency_score
  45,                                             -- fewer trips
  360,
  ARRAY['route-colombo-kandy'::UUID],
  '2026-03-11 02:00:00'::TIMESTAMPTZ,
  90
),

-- Ahmed Perera: Average performance
(
  'driver-prof-003'::UUID,
  'driver-ahmed-perera'::UUID,
  1.00,                                           -- exactly average
  0.72,
  0.84,
  98,
  784,
  ARRAY['route-colombo-kandy'::UUID],
  '2026-03-11 02:00:00'::TIMESTAMPTZ,
  90
);
```

---

## 6️⃣ `bus_performance_profiles` - Bus Performance

### Purpose
Pre-computed bus performance characteristics (refreshed daily).

### Attributes

| Column | Type | Required | Description |
|--------|------|----------|-------------|
| `id` | UUID | ✅ | Primary key |
| `bus_id` | UUID | ✅ | FK to buses (unique) |
| `avg_speed_factor` | NUMERIC(4,3) | ✅ | Speed relative to average |
| `reliability_score` | NUMERIC(3,2) | ✅ | Data quality/GPS reliability |
| `fuel_efficiency_kmpl` | NUMERIC(6,2) | ❌ | Kilometers per liter |
| `days_since_last_service` | INTEGER | ❌ | Maintenance tracking |
| `breakdown_count_90d` | INTEGER | ✅ | Breakdowns in last 90 days |
| `total_trips_analyzed` | INTEGER | ✅ | Number of trips |
| `total_km_traveled` | NUMERIC(10,2) | ✅ | Total distance |
| `last_calculated_at` | TIMESTAMPTZ | ✅ | Last refresh |
| `calculation_window_days` | INTEGER | ✅ | Days analyzed |

### Example Data

```sql
INSERT INTO bus_performance_profiles VALUES
-- AC-001: New luxury bus, excellent performance
(
  'bus-prof-001'::UUID,
  'bus-ac-001'::UUID,
  0.98,                                           -- avg_speed_factor (2% faster)
  0.96,                                           -- reliability_score
  8.5,                                            -- fuel_efficiency_kmpl
  15,                                             -- days_since_last_service
  0,                                              -- breakdown_count_90d
  245,                                            -- total_trips_analyzed
  32450.75,                                       -- total_km_traveled
  '2026-03-11 02:00:00'::TIMESTAMPTZ,
  90
),

-- Normal-005: Older bus, slower
(
  'bus-prof-002'::UUID,
  'bus-normal-005'::UUID,
  1.05,                                           -- avg_speed_factor (5% slower)
  0.88,
  6.2,
  45,
  2,                                              -- 2 breakdowns
  189,
  25120.30,
  '2026-03-11 02:00:00'::TIMESTAMPTZ,
  90
),

-- Semi-003: Average performance
(
  'bus-prof-003'::UUID,
  'bus-semi-003'::UUID,
  1.00,                                           -- exactly average
  0.92,
  7.5,
  22,
  0,
  198,
  28650.50,
  '2026-03-11 02:00:00'::TIMESTAMPTZ,
  90
);
```

---

## 7️⃣ `lounge_stop_metrics` - Lounge Dwell Times

### Purpose
Aggregated metrics for how long buses stop at each lounge.

### Attributes

| Column | Type | Required | Description |
|--------|------|----------|-------------|
| `id` | UUID | ✅ | Primary key |
| `lounge_id` | UUID | ✅ | FK to lounges |
| `master_route_id` | UUID | ✅ | FK to master_routes |
| `average_dwell_time_minutes` | NUMERIC(6,2) | ✅ | Average stop duration |
| `min_dwell_time_minutes` | NUMERIC(6,2) | ✅ | Minimum stop time |
| `max_dwell_time_minutes` | NUMERIC(6,2) | ✅ | Maximum stop time |
| `peak_hour_dwell_time_minutes` | NUMERIC(6,2) | ✅ | Dwell during peak hours |
| `off_peak_dwell_time_minutes` | NUMERIC(6,2) | ✅ | Dwell during off-peak |
| `average_bookings_per_trip` | INTEGER | ✅ | Avg passengers picked up |
| `total_stops_recorded` | INTEGER | ✅ | Number of stops analyzed |
| `last_calculated_at` | TIMESTAMPTZ | ❌ | Last refresh time |
| `data_points_count` | INTEGER | ✅ | Data points used |
| `created_at` | TIMESTAMPTZ | ✅ | Record creation |
| `updated_at` | TIMESTAMPTZ | ✅ | Last update |

### Example Data

```sql
INSERT INTO lounge_stop_metrics VALUES
-- City Plaza Lounge: Busy urban lounge
(
  'lounge-metric-001'::UUID,
  'city-plaza-lounge'::UUID,
  'route-colombo-kandy'::UUID,
  15.5,                                           -- average_dwell_time_minutes
  8.0,                                            -- min_dwell_time_minutes
  35.0,                                           -- max_dwell_time_minutes
  22.0,                                           -- peak_hour_dwell_time_minutes
  12.0,                                           -- off_peak_dwell_time_minutes
  8,                                              -- average_bookings_per_trip
  342,                                            -- total_stops_recorded
  '2026-03-11 02:00:00'::TIMESTAMPTZ,
  342,
  '2026-01-01 00:00:00'::TIMESTAMPTZ,
  '2026-03-11 02:00:00'::TIMESTAMPTZ
),

-- Hilltop Lounge: Highway rest stop, longer dwell
(
  'lounge-metric-002'::UUID,
  'hilltop-lounge'::UUID,
  'route-colombo-kandy'::UUID,
  18.0,
  10.0,
  40.0,
  25.0,
  15.0,
  12,
  298,
  '2026-03-11 02:00:00'::TIMESTAMPTZ,
  298,
  '2026-01-01 00:00:00'::TIMESTAMPTZ,
  '2026-03-11 02:00:00'::TIMESTAMPTZ
),

-- Peradeniya Lounge: Quick stop near destination
(
  'lounge-metric-003'::UUID,
  'peradeniya-lounge'::UUID,
  'route-colombo-kandy'::UUID,
  10.0,
  5.0,
  20.0,
  12.0,
  8.0,
  5,
  256,
  '2026-03-11 02:00:00'::TIMESTAMPTZ,
  256,
  '2026-01-01 00:00:00'::TIMESTAMPTZ,
  '2026-03-11 02:00:00'::TIMESTAMPTZ
);
```

---

## 8️⃣ `eta_predictions` - Prediction Tracking

### Purpose
Stores ETA predictions for auditing and accuracy improvement.

### Attributes

| Column | Type | Required | Description |
|--------|------|----------|-------------|
| `id` | UUID | ✅ | Primary key |
| `active_trip_id` | UUID | ✅ | FK to active_trips |
| `lounge_id` | UUID | ❌ | FK to lounges (NULL if stop prediction) |
| `stop_id` | UUID | ❌ | FK to master_route_stops (NULL if lounge) |
| `prediction_type` | VARCHAR(20) | ✅ | 'lounge', 'stop', 'destination' |
| `predicted_at` | TIMESTAMPTZ | ✅ | When prediction was made |
| `predicted_arrival_time` | TIMESTAMPTZ | ✅ | Predicted arrival time |
| `current_location_lat` | NUMERIC(10,7) | ❌ | GPS at prediction time |
| `current_location_lng` | NUMERIC(10,7) | ❌ | GPS at prediction time |
| `distance_remaining_km` | NUMERIC(8,3) | ❌ | Distance to target |
| `actual_arrival_time` | TIMESTAMPTZ | ❌ | Actual arrival (filled later) |
| `prediction_error_minutes` | NUMERIC(8,2) | ❌ | Calculated error (generated column) |
| `calculation_method` | VARCHAR(20) | ✅ | 'baseline', 'historical', 'ml', 'hybrid', 'realtime' |
| `confidence_score` | NUMERIC(5,2) | ✅ | 0-100 confidence percentage |
| `context_data` | JSONB | ❌ | Additional metadata |
| `created_at` | TIMESTAMPTZ | ✅ | Record creation |

### Example Data

```sql
INSERT INTO eta_predictions VALUES
-- Prediction 1: Historical method, high confidence, accurate
(
  'pred-00001'::UUID,
  'trip-active-001'::UUID,
  'city-plaza-lounge'::UUID,
  NULL,                                           -- stop_id
  'lounge',
  '2026-03-10 08:00:00'::TIMESTAMPTZ,            -- predicted_at
  '2026-03-10 08:50:00'::TIMESTAMPTZ,            -- predicted_arrival_time
  6.9271,                                         -- current_location_lat
  79.8612,                                        -- current_location_lng
  20.5,                                           -- distance_remaining_km
  '2026-03-10 08:53:00'::TIMESTAMPTZ,            -- actual_arrival_time (came 3 min late)
  3.0,                                            -- prediction_error_minutes (calculated)
  'historical',
  87.5,                                           -- confidence_score
  '{"driver_factor": 0.95, "weather": "rain", "traffic": "heavy"}'::JSONB,
  '2026-03-10 08:00:05'::TIMESTAMPTZ
),

-- Prediction 2: Baseline method (no historical data), low confidence
(
  'pred-00002'::UUID,
  'trip-active-003'::UUID,
  'new-lounge'::UUID,
  NULL,
  'lounge',
  '2026-03-10 10:00:00'::TIMESTAMPTZ,
  '2026-03-10 10:45:00'::TIMESTAMPTZ,
  7.0121,
  79.9156,
  35.0,
  '2026-03-10 10:52:00'::TIMESTAMPTZ,            -- came 7 min late
  7.0,
  'baseline',
  45.0,                                           -- low confidence (no historical data)
  '{"reason": "new_route", "sample_count": 0}'::JSONB,
  '2026-03-10 10:00:02'::TIMESTAMPTZ
),

-- Prediction 3: Real-time method (GPS + speed), very high confidence, very accurate
(
  'pred-00003'::UUID,
  'trip-active-002'::UUID,
  'city-plaza-lounge'::UUID,
  NULL,
  'lounge',
  '2026-03-10 14:30:00'::TIMESTAMPTZ,
  '2026-03-10 14:52:00'::TIMESTAMPTZ,
  6.9500,
  79.8700,
  8.5,
  '2026-03-10 14:51:00'::TIMESTAMPTZ,            -- came 1 min early!
  -1.0,                                           -- negative = arrived early
  'realtime',
  95.0,                                           -- very high confidence
  '{"current_speed": 42.5, "traffic": "light", "gps_accuracy": 8}'::JSONB,
  '2026-03-10 14:30:01'::TIMESTAMPTZ
);
```

---

## 9️⃣ `segment_historical_performance` - Legacy Historical Data

### Purpose
Original historical performance table (can be deprecated after migration to optimized schema).

### Attributes

| Column | Type | Required | Description |
|--------|------|----------|-------------|
| `id` | UUID | ✅ | Primary key |
| `route_segment_id` | UUID | ✅ | FK to route_segments |
| `active_trip_id` | UUID | ❌ | FK to active_trips |
| `scheduled_trip_id` | UUID | ❌ | FK to scheduled_trips |
| `bus_id` | UUID | ❌ | FK to buses |
| `driver_id` | UUID | ❌ | FK to bus_staff |
| `trip_date` | DATE | ✅ | Date of trip |
| `segment_start_time` | TIMESTAMPTZ | ✅ | Segment start |
| `segment_end_time` | TIMESTAMPTZ | ✅ | Segment end |
| `actual_duration_minutes` | NUMERIC(8,2) | ✅ | Actual duration |
| `estimated_duration_minutes` | NUMERIC(8,2) | ❌ | Estimated duration |
| `duration_variance_minutes` | NUMERIC(8,2) | ❌ | Generated: actual - estimated |
| `average_speed_kmh` | NUMERIC(5,2) | ❌ | Average speed |
| `max_speed_kmh` | NUMERIC(5,2) | ❌ | Max speed |
| `min_speed_kmh` | NUMERIC(5,2) | ❌ | Min speed |
| `stop_count` | INTEGER | ❌ | Number of stops |
| `dwell_time_seconds` | INTEGER | ❌ | Total stop time |
| `time_of_day_category` | VARCHAR(20) | ❌ | Peak, midday, etc. |
| `day_of_week` | VARCHAR(10) | ❌ | Monday, etc. |
| `is_holiday` | BOOLEAN | ❌ | Public holiday |
| `weather_condition` | VARCHAR(20) | ❌ | Clear, rain, etc. |
| `bus_type` | VARCHAR(50) | ❌ | Bus type snapshot |
| `bus_occupancy_percentage` | INTEGER | ❌ | Passenger load |
| `driver_experience_years` | INTEGER | ❌ | Driver experience |
| `created_at` | TIMESTAMPTZ | ✅ | Record creation |

### Example Data

```sql
-- Note: After migration, this data should be split into trip_contexts + segment_performance_facts
INSERT INTO segment_historical_performance VALUES
(
  'hist-00001'::UUID,
  'a1b2c3d4-0001-0000-0000-000000000001'::UUID,
  'trip-active-001'::UUID,
  'trip-scheduled-001'::UUID,
  'bus-ac-001'::UUID,
  'driver-john-doe'::UUID,
  '2026-03-10',
  '2026-03-10 08:00:00'::TIMESTAMPTZ,
  '2026-03-10 08:24:00'::TIMESTAMPTZ,
  24.0,
  13.0,                                           -- estimated
  11.0,                                           -- variance
  21.25,
  45.0,
  5.0,
  3,
  120,
  'morning_peak',
  'monday',
  false,
  'rain',
  'AC Luxury',
  75,                                             -- 75% full
  8,
  '2026-03-10 08:24:05'::TIMESTAMPTZ
);
```

---

## 📊 Summary Table

| Table | Records Per Trip | Update Frequency | Primary Use |
|-------|------------------|------------------|-------------|
| `route_segments` | 0 (pre-configured) | Rarely | Route definition |
| `trip_contexts` | 1 | Per trip start | Trip metadata |
| `segment_performance_facts` | 5-15 | Per segment completion | Raw performance data |
| `segment_aggregate_stats` | 0 (computed) | Daily | Fast ETA lookup |
| `driver_performance_profiles` | 0 (computed) | Daily | Driver factors |
| `bus_performance_profiles` | 0 (computed) | Daily | Bus factors |
| `lounge_stop_metrics` | 0 (computed) | Daily | Dwell time lookup |
| `eta_predictions` | 3-8 | Per ETA calculation | Accuracy tracking |
| `segment_historical_performance` | 5-15 | Per segment (legacy) | Migration source |

---

## 🔄 Data Flow

```
Trip Starts
    ├─> Create trip_contexts (1 record)
    │
Bus Completes Segment
    ├─> Create segment_performance_facts (1 record per segment)
    │
Bus Arrives at Lounge
    ├─> Create eta_predictions with actual_arrival_time
    │
Daily (2 AM)
    ├─> Refresh segment_aggregate_stats (from facts)
    ├─> Refresh driver_performance_profiles
    ├─> Refresh bus_performance_profiles
    └─> Refresh lounge_stop_metrics

ETA Calculation Request
    ├─> Read segment_aggregate_stats (fast lookup)
    ├─> Read driver_performance_profiles
    ├─> Read bus_performance_profiles
    ├─> Read lounge_stop_metrics
    ├─> Calculate ETA
    └─> Save eta_predictions (async)
```

---

**Last Updated:** March 12, 2026  
**Schema Version:** 3.0 (Optimized)
