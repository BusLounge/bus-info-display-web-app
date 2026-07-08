# ETA Engine Schema Optimization - Comparison & Benefits

## Problem with Original Schema

The original `segment_historical_performance` table stored **redundant data**:

```sql
-- ORIGINAL: Everything in one table
CREATE TABLE segment_historical_performance (
  id UUID,
  route_segment_id UUID,
  trip_id UUID,
  active_trip_id UUID,
  
  -- These repeat for EVERY segment in a trip ❌
  driver_id UUID,
  driver_experience_years INTEGER,
  bus_id UUID,
  time_of_day_category VARCHAR(20),
  day_of_week VARCHAR(10),
  weather_condition VARCHAR(20),
  
  -- Actual performance data
  actual_duration_minutes NUMERIC,
  average_speed_kmh NUMERIC,
  ...
);
```

**Example: One trip with 10 segments**
- Driver info stored 10 times ❌
- Bus info stored 10 times ❌
- Weather stored 10 times ❌
- Time category stored 10 times ❌

**Result:**
- High storage overhead
- Data inconsistency risk (what if driver_id differs across segments of same trip?)
- Slower aggregation queries
- Wasted space

---

## Optimized Schema Solution

### 🎯 Core Principle: **Separate Facts from Dimensions**

```
┌─────────────────────────────────────────────────────┐
│                 TRIP CONTEXTS                       │
│  (One record per trip - stores context)             │
│  - driver_id, bus_id, weather, time, date           │
└────────────────┬────────────────────────────────────┘
                 │
                 │ 1:N relationship
                 │
┌────────────────▼────────────────────────────────────┐
│           SEGMENT PERFORMANCE FACTS                 │
│  (Multiple records per trip - stores metrics)       │
│  - segment_id, trip_context_id                      │
│  - actual_duration, speed, variance                 │
└─────────────────────────────────────────────────────┘
```

---

## Schema Breakdown

### 1️⃣ **trip_contexts** (Dimension Table)
**Purpose:** Store contextual info once per trip

| Field | Description |
|-------|-------------|
| `active_trip_id` | Foreign key to active_trips |
| `driver_id`, `bus_id` | Who & what |
| `time_of_day_category` | When (peak, off-peak, etc.) |
| `day_of_week` | Monday, Tuesday, etc. |
| `weather_condition` | Clear, rain, etc. |
| `total_passengers` | Trip load |

**Storage impact:** 1 row per trip (not per segment)

---

### 2️⃣ **segment_performance_facts** (Fact Table)
**Purpose:** Store only the timing/speed metrics

| Field | Description |
|-------|-------------|
| `route_segment_id` | Which segment |
| `trip_context_id` | Link to context |
| `actual_duration_minutes` | How long it took |
| `average_speed_kmh` | Average speed |
| `traffic_level` | Segment-specific traffic |
| `data_quality_score` | GPS accuracy |

**Storage impact:** Multiple rows per trip (one per segment)

---

### 3️⃣ **segment_aggregate_stats** (Pre-computed Table)
**Purpose:** Speed up ETA calculations

| Field | Description |
|-------|-------------|
| `route_segment_id` | Segment identifier |
| `time_of_day_category` | Peak, midday, etc. |
| `day_of_week` | Monday, etc. |
| `avg_duration_minutes` | **Pre-calculated average** |
| `median_duration_minutes` | Median value |
| `stddev_duration_minutes` | Variability |
| `p95_duration_minutes` | 95th percentile |
| `sample_count` | How many trips used |

**Key benefit:** No need to scan millions of rows - just lookup!

---

### 4️⃣ **driver_performance_profiles** (Optional)
**Purpose:** Pre-computed driver efficiency metrics

| Field | Description |
|-------|-------------|
| `driver_id` | Driver identifier |
| `avg_speed_factor` | 0.95 = 5% faster than baseline |
| `punctuality_score` | 0-1, how often on time |
| `consistency_score` | 0-1, low variance = high score |

---

### 5️⃣ **bus_performance_profiles** (Optional)
**Purpose:** Pre-computed bus characteristics

| Field | Description |
|-------|-------------|
| `bus_id` | Bus identifier |
| `avg_speed_factor` | Relative performance |
| `reliability_score` | Data quality indicator |
| `days_since_last_service` | Maintenance tracking |

---

## Storage Savings Example

**Scenario:** 1000 trips, 10 segments per trip, 90 days of data

### Original Schema
```
Records: 1000 trips × 10 segments × 90 days = 900,000 rows

Size per row (estimated):
  - 16 bytes (UUID id)
  - 32 bytes (route_segment_id)
  - 32 bytes (active_trip_id)
  - 32 bytes (driver_id)
  - 32 bytes (bus_id)
  - 50 bytes (time/date columns)
  - 20 bytes (string fields: weather, day, time category)
  - 40 bytes (numeric fields: duration, speed, variance)
  ≈ 254 bytes per row

Total: 900,000 × 254 bytes ≈ 229 MB
```

### Optimized Schema
```
trip_contexts:
  1000 trips × 90 days = 90,000 rows
  150 bytes per row = 13.5 MB

segment_performance_facts:
  900,000 rows (same number)
  80 bytes per row (much smaller - only metrics)
  = 72 MB

segment_aggregate_stats:
  100 segments × 5 time categories × 7 days × 3 weather = 10,500 rows
  100 bytes per row = 1 MB

Total: 13.5 + 72 + 1 = 86.5 MB
```

**Savings: 229 MB → 86.5 MB = 62% reduction!** 🎉

---

## Query Performance Comparison

### Query: Get average duration for morning peak on Mondays

#### ❌ **OLD WAY** (Slow)
```sql
SELECT 
  route_segment_id,
  AVG(actual_duration_minutes) as avg_duration
FROM segment_historical_performance
WHERE time_of_day_category = 'morning_peak'
  AND day_of_week = 'monday'
  AND trip_date >= CURRENT_DATE - INTERVAL '90 days'
GROUP BY route_segment_id;
```
- **Scans:** 900,000 rows
- **Filters:** Checks every row
- **Aggregates:** Groups and calculates AVG
- **Estimated time:** 500ms - 2 seconds

#### ✅ **NEW WAY** (Fast)
```sql
SELECT 
  route_segment_id,
  avg_duration_minutes
FROM segment_aggregate_stats
WHERE time_of_day_category = 'morning_peak'
  AND day_of_week = 'monday';
```
- **Scans:** ~100 rows (pre-aggregated)
- **Filters:** Index lookup
- **Aggregates:** None (already calculated)
- **Estimated time:** 5ms - 20ms

**Speed improvement: 100x faster!** 🚀

---

## ETA Calculation Query Example

### Get ETA for a segment during current context

```sql
-- Step 1: Get current trip context
WITH current_context AS (
  SELECT 
    time_of_day_category,
    day_of_week,
    weather_condition,
    driver_id,
    bus_id
  FROM trip_contexts
  WHERE active_trip_id = $1
)

-- Step 2: Get segment baseline from aggregated stats
SELECT 
  sas.avg_duration_minutes as historical_duration,
  sas.stddev_duration_minutes as variability,
  sas.sample_count as confidence,
  
  -- Get driver factor
  COALESCE(dpp.avg_speed_factor, 1.0) as driver_factor,
  
  -- Get bus factor
  COALESCE(bpp.avg_speed_factor, 1.0) as bus_factor,
  
  -- Calculate final ETA
  sas.avg_duration_minutes * 
    COALESCE(dpp.avg_speed_factor, 1.0) * 
    COALESCE(bpp.avg_speed_factor, 1.0) as estimated_duration

FROM segment_aggregate_stats sas
CROSS JOIN current_context cc

-- Join driver profile
LEFT JOIN driver_performance_profiles dpp 
  ON dpp.driver_id = cc.driver_id
  
-- Join bus profile
LEFT JOIN bus_performance_profiles bpp 
  ON bpp.bus_id = cc.bus_id

WHERE sas.route_segment_id = $2
  AND sas.time_of_day_category = cc.time_of_day_category
  AND sas.day_of_week = cc.day_of_week
  AND sas.weather_condition = cc.weather_condition;
```

**Benefits:**
- ✅ Single fast query
- ✅ Uses pre-computed stats
- ✅ Applies driver/bus factors automatically
- ✅ Sub-100ms response time

---

## Data Quality & Maintenance

### Refresh Strategy

```sql
-- Hourly: Refresh materialized view (contains last 90 days)
REFRESH MATERIALIZED VIEW segment_performance_with_context;

-- Every 6 hours: Recalculate aggregate stats
SELECT refresh_segment_aggregate_stats();

-- Daily: Update driver and bus profiles
SELECT refresh_driver_performance_profiles();
SELECT refresh_bus_performance_profiles();
```

### Data Integrity

The normalized design prevents:
- ❌ Driver ID mismatch across segments of same trip
- ❌ Weather changing mid-trip in database
- ❌ Time category inconsistencies

---

## Migration Path

### If you already have data in `segment_historical_performance`:

```sql
-- Run the migration function
SELECT migrate_to_optimized_schema();
```

This will:
1. Extract unique trips → `trip_contexts`
2. Link segment data → `segment_performance_facts`
3. Preserve all historical data

### Then verify:
```sql
-- Check trip contexts created
SELECT COUNT(*) FROM trip_contexts;

-- Check facts migrated
SELECT COUNT(*) FROM segment_performance_facts;

-- Compare counts
SELECT 
  (SELECT COUNT(*) FROM segment_historical_performance) as old_count,
  (SELECT COUNT(*) FROM segment_performance_facts) as new_count;
```

---

## Recommended Indexes

All critical indexes are automatically created:

```sql
-- Fast trip lookup
CREATE INDEX idx_trip_contexts_active_trip ON trip_contexts(active_trip_id);

-- Fast aggregation lookup (most important!)
CREATE INDEX idx_segment_agg_lookup ON segment_aggregate_stats(
  route_segment_id, 
  time_of_day_category, 
  day_of_week
);

-- Fast driver/bus profile lookup
CREATE INDEX idx_driver_profiles_driver ON driver_performance_profiles(driver_id);
CREATE INDEX idx_bus_profiles_bus ON bus_performance_profiles(bus_id);
```

---

## Summary: Why This is Better

| Aspect | Old Schema | New Schema | Improvement |
|--------|------------|------------|-------------|
| **Storage** | 229 MB | 86 MB | **62% reduction** |
| **Query Speed** | 500ms - 2s | 5ms - 20ms | **100x faster** |
| **Data Integrity** | Risk of inconsistency | Guaranteed consistency | **Better** |
| **Maintainability** | Update N rows per trip | Update 1 row per trip | **N times easier** |
| **Scalability** | Linear growth | Sub-linear growth | **Much better** |
| **Real-time ETA** | Slow aggregation | Pre-computed | **Instant** |

---

## When to Use Which Table

| Use Case | Table to Query |
|----------|----------------|
| **Real-time ETA calculation** | `segment_aggregate_stats` + profiles |
| **Historical analysis** | `segment_performance_with_context` (materialized view) |
| **Trip post-mortem** | Join `trip_contexts` + `segment_performance_facts` |
| **Driver performance review** | `driver_performance_profiles` |
| **Bus maintenance scheduling** | `bus_performance_profiles` |
| **ML training** | `segment_performance_with_context` or raw facts |

---

## Next Steps

1. ✅ **Review this schema** - Make sure it fits your needs
2. ✅ **Run migration** - `002_optimized_eta_schema.sql`
3. ✅ **Migrate existing data** - If you have data in old schema
4. ✅ **Update Go services** - Modify data insertion logic
5. ✅ **Set up refresh jobs** - Schedule aggregation updates
6. ✅ **Monitor performance** - Compare query times

---

## Questions?

- **Q: Can I keep both schemas?**  
  A: Yes, for transition period. Delete old schema once verified.

- **Q: How often should I refresh aggregates?**  
  A: Every 6 hours is good. Hourly if you need fresher data.

- **Q: What if I don't have enough historical data yet?**  
  A: System gracefully falls back to baseline calculations when `sample_count < 3`

- **Q: Does this work with ML models?**  
  A: Yes! Use `segment_performance_with_context` view for training data.
