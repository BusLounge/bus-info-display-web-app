# New Database Tables README (Non-ETA Only)

This document lists the newly introduced tables for the advertisement cost flow and TV display flow.

Excluded intentionally:
- ETA engine tables (as requested)

## 1) Prerequisites

Run these once to ensure UUID/default generators are available:

```sql
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
```

## 2) Advertisement Cost Calculation Tables

These tables support:
- traffic-level pricing
- playback log capture
- aggregate cost reporting

### 2.1 Table: advertisement_calculation

Purpose:
- Stores price per second for each traffic level.
- Used as the pricing source when calculating ad cost.

SQL:

```sql
CREATE TABLE IF NOT EXISTS advertisement_calculation (
  traffic_level TEXT PRIMARY KEY,
  cost_per_second NUMERIC(12,4) NOT NULL CHECK (cost_per_second >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO advertisement_calculation (traffic_level, cost_per_second)
VALUES
  ('Peak', 2.00),
  ('Moderate', 1.25),
  ('Off-Peak', 0.75)
ON CONFLICT (traffic_level) DO NOTHING;
```

### 2.2 Table: advertisement_playback_logs

Purpose:
- Stores one record per advertisement playback event.
- Inputs for both detailed auditing and cost aggregation.

SQL:

```sql
CREATE TABLE IF NOT EXISTS advertisement_playback_logs (
  id BIGSERIAL PRIMARY KEY,
  advertisement_id TEXT NOT NULL,
  advertisement_name TEXT NOT NULL,
  traffic_level TEXT NOT NULL,
  duration_seconds INTEGER NOT NULL CHECK (duration_seconds > 0),
  played_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT fk_ad_calc_traffic_level
    FOREIGN KEY (traffic_level) REFERENCES advertisement_calculation(traffic_level)
);

CREATE INDEX IF NOT EXISTS idx_ad_playback_logs_played_at
  ON advertisement_playback_logs (played_at DESC);

CREATE INDEX IF NOT EXISTS idx_ad_playback_logs_ad_id
  ON advertisement_playback_logs (advertisement_id);
```

### 2.3 Table: advertisement_cost_aggregates

Purpose:
- Stores daily aggregated cost totals by advertisement and traffic level.
- Makes report queries fast and stable.

SQL:

```sql
CREATE TABLE IF NOT EXISTS advertisement_cost_aggregates (
  advertisement_id TEXT NOT NULL,
  advertisement_name TEXT NOT NULL,
  traffic_level TEXT NOT NULL,
  cost_date DATE NOT NULL,
  play_count INTEGER NOT NULL DEFAULT 0,
  total_seconds INTEGER NOT NULL DEFAULT 0,
  total_cost NUMERIC(14,2) NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (advertisement_id, traffic_level, cost_date),
  CONSTRAINT fk_ad_cost_agg_traffic_level
    FOREIGN KEY (traffic_level) REFERENCES advertisement_calculation(traffic_level)
);

CREATE INDEX IF NOT EXISTS idx_ad_cost_aggregates_date
  ON advertisement_cost_aggregates (cost_date DESC);

CREATE INDEX IF NOT EXISTS idx_ad_cost_aggregates_ad_id
  ON advertisement_cost_aggregates (advertisement_id);
```

## 3) TV Display Messaging / Ads Tables

These tables support:
- lounge TV announcements
- lounge-specific/default ads

### 3.1 Table: broadcast_messages

Purpose:
- Stores broadcast messages to display on lounge TV screens.
- Supports active windows, priority, and repeat frequency.

SQL:

```sql
CREATE TABLE IF NOT EXISTS broadcast_messages (
  id UUID PRIMARY KEY,
  message TEXT NOT NULL,
  priority TEXT NOT NULL DEFAULT 'normal',
  display_duration_seconds INTEGER NOT NULL CHECK (display_duration_seconds > 0),
  frequency_seconds INTEGER NOT NULL CHECK (frequency_seconds > 0),
  start_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  end_at TIMESTAMPTZ NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  show_on_lounge_tv BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_broadcast_messages_active_window
  ON broadcast_messages (is_active, start_at, end_at);
```

### 3.2 Table: lounge_ads

Purpose:
- Stores ads shown in lounge TV clients.
- Can be lounge-specific or global default ads.

SQL:

```sql
CREATE TABLE IF NOT EXISTS lounge_ads (
  id UUID PRIMARY KEY,
  lounge_id UUID NULL REFERENCES lounges(id) ON DELETE CASCADE,
  advertisement_name TEXT NOT NULL,
  media_url TEXT NOT NULL,
  media_type TEXT NOT NULL,
  duration_seconds INTEGER NOT NULL CHECK (duration_seconds > 0),
  priority TEXT NOT NULL DEFAULT 'normal',
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  is_default_for_all BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_lounge_ads_lounge_active
  ON lounge_ads (lounge_id, is_active);

CREATE INDEX IF NOT EXISTS idx_lounge_ads_default_active
  ON lounge_ads (is_default_for_all, is_active);
```

## 4) Execution Order (Recommended)

Use this order to avoid FK issues:

1. `CREATE EXTENSION` commands
2. `advertisement_calculation`
3. `advertisement_playback_logs`
4. `advertisement_cost_aggregates`
5. `broadcast_messages`
6. `lounge_ads`
7. all index commands
8. seed insert into `advertisement_calculation`

## 5) Verification Queries

Check that all required non-ETA new tables exist:

```sql
SELECT table_name
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_name IN (
    'advertisement_calculation',
    'advertisement_playback_logs',
    'advertisement_cost_aggregates',
    'broadcast_messages',
    'lounge_ads'
  )
ORDER BY table_name;
```

Check pricing seed rows:

```sql
SELECT traffic_level, cost_per_second
FROM advertisement_calculation
ORDER BY CASE traffic_level
  WHEN 'Peak' THEN 1
  WHEN 'Moderate' THEN 2
  WHEN 'Off-Peak' THEN 3
  ELSE 4
END;
```

## 6) Notes

- These SQL statements are compatible with PostgreSQL.
- This README intentionally excludes ETA tables.
- Your existing core table `advertisements` is already in the current schema and is referenced by the application flow.
